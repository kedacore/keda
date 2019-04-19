package servicebus

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
	"math/rand"
	"time"

	"github.com/Azure/azure-amqp-common-go/log"
	"github.com/Azure/azure-amqp-common-go/uuid"
	"github.com/opentracing/opentracing-go"
	"pack.ag/amqp"
)

type (
	// Sender provides connection, session and link handling for an sending to an entity path
	Sender struct {
		namespace  *Namespace
		connection *amqp.Client
		session    *session
		sender     *amqp.Sender
		entityPath string
		Name       string
		sessionID  *string
	}

	// SendOption provides a way to customize a message on sending
	SendOption func(event *Message) error

	eventer interface {
		toMsg() (*amqp.Message, error)
	}

	// SenderOption provides a way to customize a Sender
	SenderOption func(*Sender) error
)

// NewSender creates a new Service Bus message Sender given an AMQP client and entity path
func (ns *Namespace) NewSender(ctx context.Context, entityPath string, opts ...SenderOption) (*Sender, error) {
	span, ctx := ns.startSpanFromContext(ctx, "sb.Sender.NewSender")
	defer span.Finish()

	s := &Sender{
		namespace:  ns,
		entityPath: entityPath,
	}

	for _, opt := range opts {
		if err := opt(s); err != nil {
			log.For(ctx).Error(err)
			return nil, err
		}
	}

	err := s.newSessionAndLink(ctx)
	if err != nil {
		log.For(ctx).Error(err)
	}
	return s, err
}

// Recover will attempt to close the current session and link, then rebuild them
func (s *Sender) Recover(ctx context.Context) error {
	span, ctx := s.startProducerSpanFromContext(ctx, "sb.Sender.Recover")
	defer span.Finish()

	// we expect the Sender, session or client is in an error state, ignore errors
	closeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	closeCtx = opentracing.ContextWithSpan(closeCtx, span)
	defer cancel()
	_ = s.sender.Close(closeCtx)
	_ = s.session.Close(closeCtx)
	_ = s.connection.Close()
	return s.newSessionAndLink(ctx)
}

// Close will close the AMQP connection, session and link of the Sender
func (s *Sender) Close(ctx context.Context) error {
	span, _ := s.startProducerSpanFromContext(ctx, "sb.Sender.Close")
	defer span.Finish()

	err := s.sender.Close(ctx)
	if err != nil {
		_ = s.session.Close(ctx)
		_ = s.connection.Close()
		return err
	}

	err = s.session.Close(ctx)
	if err != nil {
		_ = s.connection.Close()
		return err
	}

	return s.connection.Close()
}

// Send will send a message to the entity path with options
//
// This will retry sending the message if the server responds with a busy error.
func (s *Sender) Send(ctx context.Context, msg *Message, opts ...SendOption) error {
	span, ctx := s.startProducerSpanFromContext(ctx, "sb.Sender.Send")
	defer span.Finish()

	if msg.SessionID == nil {
		msg.SessionID = &s.session.SessionID
		next := s.session.getNext()
		msg.GroupSequence = &next
	}

	if msg.ID == "" {
		id, err := uuid.NewV4()
		if err != nil {
			log.For(ctx).Error(err)
			return err
		}
		msg.ID = id.String()
	}

	for _, opt := range opts {
		err := opt(msg)
		if err != nil {
			log.For(ctx).Error(err)
			return err
		}
	}

	return s.trySend(ctx, msg)
}

func (s *Sender) trySend(ctx context.Context, evt eventer) error {
	sp, ctx := s.startProducerSpanFromContext(ctx, "sb.Sender.trySend")
	defer sp.Finish()

	err := opentracing.GlobalTracer().Inject(sp.Context(), opentracing.TextMap, evt)
	if err != nil {
		log.For(ctx).Error(err)
		return err
	}

	msg, err := evt.toMsg()
	if err != nil {
		log.For(ctx).Error(err)
		return err
	}

	if msg.Properties != nil {
		sp.SetTag("sb.message-id", msg.Properties.MessageID)
	}

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.For(ctx).Error(err)
			}
			return ctx.Err()
		default:
			// try as long as the context is not dead
			err = s.sender.Send(ctx, msg)
			if err == nil {
				// successful send
				return err
			}

			switch err.(type) {
			case *amqp.Error, *amqp.DetachError:
				log.For(ctx).Debug("amqp error, delaying 4 seconds: " + err.Error())
				skew := time.Duration(rand.Intn(1000)-500) * time.Millisecond
				time.Sleep(4*time.Second + skew)
				err := s.Recover(ctx)
				if err != nil {
					log.For(ctx).Debug("failed to recover connection")
				}
				log.For(ctx).Debug("recovered connection")
			default:
				log.For(ctx).Error(err)
				return err
			}
		}
	}
}

func (s *Sender) String() string {
	return s.Name
}

func (s *Sender) getAddress() string {
	return s.entityPath
}

func (s *Sender) getFullIdentifier() string {
	return s.namespace.getEntityAudience(s.getAddress())
}

// newSessionAndLink will replace the existing session and link
func (s *Sender) newSessionAndLink(ctx context.Context) error {
	span, ctx := s.startProducerSpanFromContext(ctx, "sb.Sender.newSessionAndLink")
	defer span.Finish()

	connection, err := s.namespace.newConnection()
	if err != nil {
		log.For(ctx).Error(err)
		return err
	}
	s.connection = connection

	err = s.namespace.negotiateClaim(ctx, connection, s.getAddress())
	if err != nil {
		log.For(ctx).Error(err)
		return err
	}

	amqpSession, err := connection.NewSession()
	if err != nil {
		log.For(ctx).Error(err)
		return err
	}

	amqpSender, err := amqpSession.NewSender(
		amqp.LinkSenderSettle(amqp.ModeUnsettled),
		amqp.LinkTargetAddress(s.getAddress()))
	if err != nil {
		log.For(ctx).Error(err)
		return err
	}

	s.session, err = newSession(amqpSession)
	if err != nil {
		log.For(ctx).Error(err)
		return err
	}
	if s.sessionID != nil {
		s.session.SessionID = *s.sessionID
	}

	s.sender = amqpSender
	return nil
}

// SenderWithSession configures the message to send with a specific session and sequence. By default, a Sender has a
// default session (uuid.NewV4()) and sequence generator.
func SenderWithSession(sessionID *string) SenderOption {
	return func(sender *Sender) error {
		sender.sessionID = sessionID
		return nil
	}
}
