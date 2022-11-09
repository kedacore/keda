package eventhub

//	MIT License
//
//	Copyright (c) Microsoft Corporation. All rights reserved.
//
//	Permission is hereby granted, free of charge, to any person obtaining a copy
//	of this software and associated documentation files (the "Software"), to deal
//	in the Software without restriction, including without limitation the rights
//	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
//	copies of the Software, and to permit persons to whom the Software is
//	furnished to do so, subject to the following conditions:
//
//	The above copyright notice and this permission notice shall be included in all
//	copies or substantial portions of the Software.
//
//	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
//	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
//	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
//	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
//	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//	SOFTWARE

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Azure/azure-amqp-common-go/v3/uuid"
	"github.com/Azure/go-amqp"
	"github.com/devigned/tab"
	"github.com/jpillora/backoff"
)

const (
	errorServerBusy amqp.ErrorCondition = "com.microsoft:server-busy"
	errorTimeout    amqp.ErrorCondition = "com.microsoft:timeout"
)

// sender provides session and link handling for an sending entity path
type (
	sender struct {
		hub          *Hub
		connection   *amqp.Client
		session      *session
		sender       atomic.Value // holds a *amqp.Sender
		partitionID  *string
		Name         string
		retryOptions *senderRetryOptions
		// cond and recovering are used to atomically implement Recover()
		cond       *sync.Cond
		recovering bool
	}

	// SendOption provides a way to customize a message on sending
	SendOption func(event *Event) error

	eventer interface {
		tab.Carrier
		toMsg() (*amqp.Message, error)
	}

	// amqpSender is the bare minimum we need from an AMQP based sender.
	// (used for testing)
	// Implemented by *amqp.Sender
	amqpSender interface {
		LinkName() string
		Send(ctx context.Context, msg *amqp.Message) error
		Close(ctx context.Context) error
	}

	// getAmqpSender should return a live sender (exactly mimics the `amqpSender()` function below)
	// (used for testing)
	getAmqpSender func() amqpSender

	senderRetryOptions struct {
		recoveryBackoff *backoff.Backoff

		// maxRetries controls how many times we try (in addition to the first attempt)
		// 0 indicates no retries, and < 0 will cause infinite retries.
		// Defaults to -1.
		maxRetries int
	}
)

func newSenderRetryOptions() *senderRetryOptions {
	return &senderRetryOptions{
		recoveryBackoff: &backoff.Backoff{
			Min:    10 * time.Millisecond,
			Max:    4 * time.Second,
			Jitter: true,
		},
		maxRetries: -1, // default to infinite retries
	}
}

// newSender creates a new Service Bus message sender given an AMQP client and entity path
func (h *Hub) newSender(ctx context.Context, retryOptions *senderRetryOptions) (*sender, error) {
	span, ctx := h.startSpanFromContext(ctx, "eh.sender.newSender")
	defer span.End()

	s := &sender{
		hub:          h,
		partitionID:  h.senderPartitionID,
		retryOptions: retryOptions,
		cond:         sync.NewCond(&sync.Mutex{}),
	}
	tab.For(ctx).Debug(fmt.Sprintf("creating a new sender for entity path %s", s.getAddress()))
	err := s.newSessionAndLink(ctx)
	return s, err
}

func (s *sender) amqpSender() amqpSender {
	// in reality, an *amqp.Sender
	return s.sender.Load().(amqpSender)
}

// Recover will attempt to close the current connectino, session and link, then rebuild them.
func (s *sender) Recover(ctx context.Context) error {
	return s.recoverWithExpectedLinkID(ctx, "")
}

// recoverWithExpectedLinkID attemps to recover the link as cheaply as possible.
// - It does not recover the link if expectedLinkID is not "" and does NOT match
//   the current link ID, as this would indicate that the previous bad link has
//   already been closed and removed.
func (s *sender) recoverWithExpectedLinkID(ctx context.Context, expectedLinkID string) error {
	span, ctx := s.startProducerSpanFromContext(ctx, "eh.sender.Recover")
	defer span.End()

	recover := false

	// acquire exclusive lock to see if this goroutine should recover
	s.cond.L.Lock() // block 1

	// if the link they started with has already been closed and removed we don't
	// need to trigger an additional recovery.
	if expectedLinkID != "" && s.amqpSender().LinkName() != expectedLinkID {
		tab.For(ctx).Debug("original linkID does not match, no recovery necessary")
	} else if !s.recovering {
		// another goroutine isn't recovering, so this one will
		tab.For(ctx).Debug("will recover connection")
		s.recovering = true
		recover = true
	} else {
		// wait for the recovery to finish
		tab.For(ctx).Debug("waiting for connection to recover")
		s.cond.Wait()
	}

	s.cond.L.Unlock()

	var err error
	if recover {
		tab.For(ctx).Debug("recovering connection")
		// we expect the sender, session or client is in an error state, ignore errors
		closeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		// update shared state
		s.cond.L.Lock() // block 2

		// TODO: we should be able to recover more quickly if we don't close the connection
		// to recover (and just attempt to recreate the link). newSessionAndLink, currently,
		// creates a new connection so we'd need to change that.
		_ = s.amqpSender().Close(closeCtx)
		_ = s.session.Close(closeCtx)
		_ = s.connection.Close()
		err = s.newSessionAndLink(ctx)

		s.recovering = false
		s.cond.L.Unlock()
		// signal to waiters that recovery is complete
		s.cond.Broadcast()
	}
	return err
}

// Close will close the AMQP connection, session and link of the sender
func (s *sender) Close(ctx context.Context) error {
	span, _ := s.startProducerSpanFromContext(ctx, "eh.sender.Close")
	defer span.End()

	err := s.amqpSender().Close(ctx)
	if err != nil {
		tab.For(ctx).Error(err)
		if sessionErr := s.session.Close(ctx); sessionErr != nil {
			tab.For(ctx).Error(sessionErr)
		}

		if connErr := s.connection.Close(); connErr != nil {
			tab.For(ctx).Error(connErr)
		}

		return err
	}

	if sessionErr := s.session.Close(ctx); sessionErr != nil {
		tab.For(ctx).Error(sessionErr)

		if connErr := s.connection.Close(); connErr != nil {
			tab.For(ctx).Error(connErr)
		}

		return sessionErr
	}

	return s.connection.Close()
}

// Send will send a message to the entity path with options
//
// This will retry sending the message if the server responds with a busy error.
func (s *sender) Send(ctx context.Context, event *Event, opts ...SendOption) error {
	span, ctx := s.startProducerSpanFromContext(ctx, "eh.sender.Send")
	defer span.End()

	for _, opt := range opts {
		err := opt(event)
		if err != nil {
			return err
		}
	}

	if event.ID == "" {
		id, err := uuid.NewV4()
		if err != nil {
			return err
		}
		event.ID = id.String()
	}

	return s.trySend(ctx, event)
}

func (s *sender) trySend(ctx context.Context, evt eventer) error {
	sp, ctx := s.startProducerSpanFromContext(ctx, "eh.sender.trySend")
	defer sp.End()

	if err := sp.Inject(evt); err != nil {
		tab.For(ctx).Error(err)
		return err
	}

	msg, err := evt.toMsg()
	if err != nil {
		tab.For(ctx).Error(err)
		return err
	}

	if str, ok := msg.Properties.MessageID.(string); ok {
		sp.AddAttributes(tab.StringAttribute("he.message_id", str))
	}

	// create a per goroutine copy as Duration() and Reset() modify its state
	backoff := s.retryOptions.recoveryBackoff.Copy()

	recvr := func(linkID string, err error, recover bool) {
		duration := backoff.Duration()
		tab.For(ctx).Debug("amqp error, delaying " + strconv.FormatInt(int64(duration/time.Millisecond), 10) + " millis: " + err.Error())
		select {
		case <-time.After(duration):
			// ok, continue to recover
		case <-ctx.Done():
			// context expired, exit
			return
		}
		if recover {
			err = s.recoverWithExpectedLinkID(ctx, linkID)
			if err != nil {
				tab.For(ctx).Debug("failed to recover connection")
			} else {
				tab.For(ctx).Debug("recovered connection")
				backoff.Reset()
			}
		}
	}

	// try as long as the context is not dead
	// successful send
	// don't rebuild the connection in this case, just delay and try again
	return sendMessage(ctx, s.amqpSender, s.retryOptions.maxRetries, msg, recvr)
}

func sendMessage(ctx context.Context, getAmqpSender getAmqpSender, maxRetries int, msg *amqp.Message, recoverLink func(linkID string, err error, recover bool)) error {
	var lastError error

	// maxRetries >= 0 == finite retries
	// maxRetries < 0 == infinite retries
	for i := 0; i < maxRetries+1 || maxRetries < 0; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sender := getAmqpSender()
			err := sender.Send(ctx, msg)
			if err == nil {
				return err
			}

			lastError = err

			switch e := err.(type) {
			case *amqp.Error:
				if e.Condition == errorServerBusy || e.Condition == errorTimeout {
					recoverLink(sender.LinkName(), err, false)
					break
				}
				recoverLink(sender.LinkName(), err, true)
			case *amqp.DetachError, net.Error:
				recoverLink(sender.LinkName(), err, true)
			default:
				if !isRecoverableCloseError(err) {
					return err
				}

				recoverLink(sender.LinkName(), err, true)
			}
		}
	}

	return lastError
}

func (s *sender) String() string {
	return s.Name
}

func (s *sender) getAddress() string {
	if s.partitionID != nil {
		return fmt.Sprintf("%s/Partitions/%s", s.hub.name, *s.partitionID)
	}
	return s.hub.name
}

func (s *sender) getFullIdentifier() string {
	return s.hub.namespace.getEntityAudience(s.getAddress())
}

// newSessionAndLink will replace the existing connection, session and link
func (s *sender) newSessionAndLink(ctx context.Context) error {
	span, ctx := s.startProducerSpanFromContext(ctx, "eh.sender.newSessionAndLink")
	defer span.End()

	connection, err := s.hub.namespace.newConnection()
	if err != nil {
		tab.For(ctx).Error(err)
		return err
	}
	s.connection = connection

	err = s.hub.namespace.negotiateClaim(ctx, connection, s.getAddress())
	if err != nil {
		tab.For(ctx).Error(err)
		return err
	}

	amqpSession, err := connection.NewSession()
	if err != nil {
		tab.For(ctx).Error(err)
		return err
	}

	amqpSender, err := amqpSession.NewSender(
		amqp.LinkSenderSettle(amqp.ModeMixed),
		amqp.LinkReceiverSettle(amqp.ModeFirst),
		amqp.LinkTargetAddress(s.getAddress()),
		amqp.LinkDetachOnDispositionError(false),
	)
	if err != nil {
		tab.For(ctx).Error(err)
		return err
	}

	s.session, err = newSession(amqpSession)
	if err != nil {
		tab.For(ctx).Error(err)
		return err
	}

	s.sender.Store(amqpSender)
	return nil
}

// SendWithMessageID configures the message with a message ID
func SendWithMessageID(messageID string) SendOption {
	return func(event *Event) error {
		event.ID = messageID
		return nil
	}
}
