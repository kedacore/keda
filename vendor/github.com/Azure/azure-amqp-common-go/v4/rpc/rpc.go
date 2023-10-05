// Package rpc provides functionality for request / reply messaging. It is used by package mgmt and cbs.
package rpc

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
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/devigned/tab"

	common "github.com/Azure/azure-amqp-common-go/v4"
	"github.com/Azure/azure-amqp-common-go/v4/internal/tracing"
	"github.com/Azure/azure-amqp-common-go/v4/uuid"
	"github.com/Azure/go-amqp"
)

const (
	replyPostfix           = "-reply-to-"
	statusCodeKey          = "status-code"
	descriptionKey         = "status-description"
	defaultReceiverCredits = 1000
)

type (
	// Link is the bidirectional communication structure used for CBS negotiation
	Link struct {
		session *amqp.Session

		receiver amqpReceiver // *amqp.Receiver
		sender   amqpSender   // *amqp.Sender

		clientAddress string
		sessionID     *string
		useSessionID  bool
		id            string

		responseMu              sync.Mutex
		startResponseRouterOnce *sync.Once
		responseMap             map[string]chan rpcResponse

		// for unit tests
		uuidNewV4     func() (uuid.UUID, error)
		messageAccept func(ctx context.Context, message *amqp.Message) error
	}

	// Response is the simplified response structure from an RPC like call
	Response struct {
		Code        int
		Description string
		Message     *amqp.Message
	}

	// LinkOption provides a way to customize the construction of a Link
	LinkOption func(link *Link) error

	rpcResponse struct {
		message *amqp.Message
		err     error
	}

	// Actually: *amqp.Receiver
	amqpReceiver interface {
		Receive(ctx context.Context, o *amqp.ReceiveOptions) (*amqp.Message, error)
		Close(ctx context.Context) error
	}

	amqpSender interface {
		Send(ctx context.Context, msg *amqp.Message, o *amqp.SendOptions) error
		Close(ctx context.Context) error
	}
)

// LinkWithSessionFilter configures a Link to use a session filter
func LinkWithSessionFilter(sessionID *string) LinkOption {
	return func(l *Link) error {
		l.sessionID = sessionID
		l.useSessionID = true
		return nil
	}
}

// NewLink will build a new request response link
func NewLink(ctx context.Context, conn *amqp.Conn, address string, opts ...LinkOption) (*Link, error) {
	authSession, err := conn.NewSession(ctx, nil)
	if err != nil {
		return nil, err
	}

	return NewLinkWithSession(ctx, authSession, address, opts...)
}

// NewLinkWithSession will build a new request response link, but will reuse an existing AMQP session
func NewLinkWithSession(ctx context.Context, session *amqp.Session, address string, opts ...LinkOption) (*Link, error) {
	linkID, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	id := linkID.String()
	link := &Link{
		session:       session,
		clientAddress: strings.Replace("$", "", address, -1) + replyPostfix + id,
		id:            id,

		uuidNewV4:               uuid.NewV4,
		responseMap:             map[string]chan rpcResponse{},
		startResponseRouterOnce: &sync.Once{},
	}

	for _, opt := range opts {
		if err := opt(link); err != nil {
			return nil, err
		}
	}

	sender, err := session.NewSender(ctx, address, nil)
	if err != nil {
		return nil, err
	}

	receiverOpts := amqp.ReceiverOptions{
		Credit:        defaultReceiverCredits,
		TargetAddress: link.clientAddress,
	}

	if link.sessionID != nil {
		const name = "com.microsoft:session-filter"
		const code = uint64(0x00000137000000C)
		if link.sessionID == nil {
			receiverOpts.Filters = append(receiverOpts.Filters, amqp.NewLinkFilter(name, code, nil))
		} else {
			receiverOpts.Filters = append(receiverOpts.Filters, amqp.NewLinkFilter(name, code, link.sessionID))
		}
	}

	receiver, err := session.NewReceiver(ctx, address, &receiverOpts)
	if err != nil {
		// make sure we close the sender
		clsCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		_ = sender.Close(clsCtx)
		return nil, err
	}

	link.sender = sender
	link.receiver = receiver
	link.messageAccept = receiver.AcceptMessage

	return link, nil
}

// RetryableRPC attempts to retry a request a number of times with delay
func (l *Link) RetryableRPC(ctx context.Context, times int, delay time.Duration, msg *amqp.Message) (*Response, error) {
	ctx, span := tracing.StartSpanFromContext(ctx, "az-amqp-common.rpc.RetryableRPC")
	defer span.End()

	res, err := common.Retry(times, delay, func() (interface{}, error) {
		ctx, span := tracing.StartSpanFromContext(ctx, "az-amqp-common.rpc.RetryableRPC.retry")
		defer span.End()

		res, err := l.RPC(ctx, msg)

		if err != nil {
			tab.For(ctx).Error(fmt.Errorf("error in RPC via link %s: %v", l.id, err))
			return nil, err
		}

		switch {
		case res.Code >= 200 && res.Code < 300:
			tab.For(ctx).Debug(fmt.Sprintf("successful rpc on link %s: status code %d and description: %s", l.id, res.Code, res.Description))
			return res, nil
		case res.Code >= 500:
			errMessage := fmt.Sprintf("server error link %s: status code %d and description: %s", l.id, res.Code, res.Description)
			tab.For(ctx).Error(errors.New(errMessage))
			return nil, common.Retryable(errMessage)
		default:
			errMessage := fmt.Sprintf("unhandled error link %s: status code %d and description: %s", l.id, res.Code, res.Description)
			tab.For(ctx).Error(errors.New(errMessage))
			return nil, common.Retryable(errMessage)
		}
	})
	if err != nil {
		tab.For(ctx).Error(err)
		return nil, err
	}
	return res.(*Response), nil
}

// startResponseRouter is responsible for taking any messages received on the 'response'
// link and forwarding it to the proper channel. The channel is being select'd by the
// original `RPC` call.
func (l *Link) startResponseRouter() {
	for {
		res, err := l.receiver.Receive(context.Background(), nil)

		// You'll see this when the link is shutting down (either
		// service-initiated via 'detach' or a user-initiated shutdown)
		if isClosedError(err) {
			l.broadcastError(err)
			break
		} else if err != nil {
			// this is some transient error, sleep before trying again
			time.Sleep(time.Second)
		}

		// I don't believe this should happen. The JS version of this same code
		// ignores errors as well since responses should always be correlated
		// to actual send requests. So this is just here for completeness.
		if res == nil {
			continue
		}

		autogenMessageId, ok := res.Properties.CorrelationID.(string)

		if !ok {
			// TODO: it'd be good to track these in some way. We don't have a good way to
			// forward this on at this point.
			continue
		}

		ch := l.deleteChannelFromMap(autogenMessageId)

		if ch != nil {
			ch <- rpcResponse{message: res, err: err}
		}
	}
}

// RPC sends a request and waits on a response for that request
func (l *Link) RPC(ctx context.Context, msg *amqp.Message) (*Response, error) {
	l.startResponseRouterOnce.Do(func() {
		go l.startResponseRouter()
	})

	copiedMessage, messageID, err := addMessageID(msg, l.uuidNewV4)

	if err != nil {
		return nil, err
	}

	// use the copiedMessage from this point
	msg = copiedMessage

	const altStatusCodeKey, altDescriptionKey = "statusCode", "statusDescription"

	ctx, span := tracing.StartSpanFromContext(ctx, "az-amqp-common.rpc.RPC")
	defer span.End()

	msg.Properties.ReplyTo = &l.clientAddress

	if msg.ApplicationProperties == nil {
		msg.ApplicationProperties = make(map[string]interface{})
	}

	if _, ok := msg.ApplicationProperties["server-timeout"]; !ok {
		if deadline, ok := ctx.Deadline(); ok {
			msg.ApplicationProperties["server-timeout"] = uint(time.Until(deadline) / time.Millisecond)
		}
	}

	responseCh := l.addChannelToMap(messageID)

	if responseCh == nil {
		return nil, &amqp.LinkError{}
	}

	err = l.sender.Send(ctx, msg, nil)

	if err != nil {
		l.deleteChannelFromMap(messageID)
		tab.For(ctx).Error(err)
		return nil, err
	}

	var res *amqp.Message

	select {
	case <-ctx.Done():
		l.deleteChannelFromMap(messageID)
		res, err = nil, ctx.Err()
	case resp := <-responseCh:
		// this will get triggered by the loop in 'startReceiverRouter' when it receives
		// a message with our autoGenMessageID set in the correlation_id property.
		res, err = resp.message, resp.err
	}

	if err != nil {
		tab.For(ctx).Error(err)
		return nil, err
	}

	var statusCode int
	statusCodeCandidates := []string{statusCodeKey, altStatusCodeKey}
	for i := range statusCodeCandidates {
		if rawStatusCode, ok := res.ApplicationProperties[statusCodeCandidates[i]]; ok {
			if cast, ok := rawStatusCode.(int32); ok {
				statusCode = int(cast)
				break
			} else {
				err := errors.New("status code was not of expected type int32")
				tab.For(ctx).Error(err)
				return nil, err
			}
		}
	}
	if statusCode == 0 {
		err := errors.New("status codes was not found on rpc message")
		tab.For(ctx).Error(err)
		return nil, err
	}

	var description string
	descriptionCandidates := []string{descriptionKey, altDescriptionKey}
	for i := range descriptionCandidates {
		if rawDescription, ok := res.ApplicationProperties[descriptionCandidates[i]]; ok {
			if description, ok = rawDescription.(string); ok || rawDescription == nil {
				break
			} else {
				return nil, errors.New("status description was not of expected type string")
			}
		}
	}

	span.AddAttributes(tab.StringAttribute("http.status_code", fmt.Sprintf("%d", statusCode)))

	response := &Response{
		Code:        int(statusCode),
		Description: description,
		Message:     res,
	}

	if err := l.messageAccept(ctx, res); err != nil {
		tab.For(ctx).Error(err)
		return response, err
	}

	return response, err
}

// Close the link receiver, sender and session
func (l *Link) Close(ctx context.Context) error {
	ctx, span := tracing.StartSpanFromContext(ctx, "az-amqp-common.rpc.Close")
	defer span.End()

	if err := l.closeReceiver(ctx); err != nil {
		_ = l.closeSender(ctx)
		_ = l.closeSession(ctx)
		return err
	}

	if err := l.closeSender(ctx); err != nil {
		_ = l.closeSession(ctx)
		return err
	}

	return l.closeSession(ctx)
}

func (l *Link) closeReceiver(ctx context.Context) error {
	ctx, span := tracing.StartSpanFromContext(ctx, "az-amqp-common.rpc.closeReceiver")
	defer span.End()

	if l.receiver != nil {
		return l.receiver.Close(ctx)
	}
	return nil
}

func (l *Link) closeSender(ctx context.Context) error {
	ctx, span := tracing.StartSpanFromContext(ctx, "az-amqp-common.rpc.closeSender")
	defer span.End()

	if l.sender != nil {
		return l.sender.Close(ctx)
	}
	return nil
}

func (l *Link) closeSession(ctx context.Context) error {
	ctx, span := tracing.StartSpanFromContext(ctx, "az-amqp-common.rpc.closeSession")
	defer span.End()

	if l.session != nil {
		return l.session.Close(ctx)
	}
	return nil
}

// addChannelToMap adds a channel which will be used by the response router to
// notify when there is a response to the request.
// If l.responseMap is nil (for instance, via broadcastError) this function will
// return nil.
func (l *Link) addChannelToMap(messageID string) chan rpcResponse {
	l.responseMu.Lock()
	defer l.responseMu.Unlock()

	if l.responseMap == nil {
		return nil
	}

	responseCh := make(chan rpcResponse, 1)
	l.responseMap[messageID] = responseCh

	return responseCh
}

// deleteChannelFromMap removes the message from our internal map and returns
// a channel that the corresponding RPC() call is waiting on.
// If l.responseMap is nil (for instance, via broadcastError) this function will
// return nil.
func (l *Link) deleteChannelFromMap(messageID string) chan rpcResponse {
	l.responseMu.Lock()
	defer l.responseMu.Unlock()

	if l.responseMap == nil {
		return nil
	}

	ch := l.responseMap[messageID]
	delete(l.responseMap, messageID)

	return ch
}

// broadcastError notifies the anyone waiting for a response that the link/session/connection
// has closed.
func (l *Link) broadcastError(err error) {
	l.responseMu.Lock()
	defer l.responseMu.Unlock()

	for _, ch := range l.responseMap {
		ch <- rpcResponse{err: err}
	}

	l.responseMap = nil
}

// addMessageID generates a unique UUID for the message. When the service
// responds it will fill out the correlation ID property of the response
// with this ID, allowing us to link the request and response together.
//
// NOTE: this function copies 'message', adding in a 'Properties' object
// if it does not already exist.
func addMessageID(message *amqp.Message, uuidNewV4 func() (uuid.UUID, error)) (*amqp.Message, string, error) {
	uuid, err := uuidNewV4()

	if err != nil {
		return nil, "", err
	}

	autoGenMessageID := uuid.String()

	// we need to modify the message so we'll make a copy
	copiedMessage := *message

	if message.Properties == nil {
		copiedMessage.Properties = &amqp.MessageProperties{
			MessageID: autoGenMessageID,
		}
	} else {
		// properties already exist, make a copy and then update
		// the message ID
		copiedProperties := *message.Properties
		copiedProperties.MessageID = autoGenMessageID

		copiedMessage.Properties = &copiedProperties
	}

	return &copiedMessage, autoGenMessageID, nil
}

func isClosedError(err error) bool {
	var connError *amqp.ConnError
	var sessionError *amqp.SessionError
	var linkError *amqp.LinkError

	return (errors.As(err, &linkError) && linkError.RemoteErr == nil) ||
		errors.As(err, &sessionError) ||
		errors.As(err, &connError)
}
