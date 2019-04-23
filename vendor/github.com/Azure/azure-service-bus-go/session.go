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
	"sync"
	"sync/atomic"

	"github.com/Azure/azure-amqp-common-go/log"

	"github.com/Azure/azure-amqp-common-go/uuid"
	"pack.ag/amqp"
)

type (
	// session is a wrapper for the AMQP session with some added information to help with Service Bus messaging
	session struct {
		*amqp.Session
		SessionID string
		counter   uint32
	}

	// QueueSession wraps Service Bus session functionality over a Queue
	QueueSession struct {
		builder   SendAndReceiveBuilder
		builderMu sync.Mutex
		sessionID *string
		receiver  *Receiver
		sender    *Sender
	}

	// SubscriptionSession wraps Service Bus session functionality over a Subscription
	SubscriptionSession struct {
		builder   ReceiveBuilder
		builderMu sync.Mutex
		sessionID *string
		receiver  *Receiver
	}

	// TopicSession wraps Service Bus session functionality over a Topic
	TopicSession struct {
		builder   SenderBuilder
		builderMu sync.Mutex
		sessionID *string
		sender    *Sender
	}

	// ReceiverBuilder describes the ability of an entity to build receiver links
	ReceiverBuilder interface {
		NewReceiver(ctx context.Context, opts ...ReceiverOption) (*Receiver, error)
	}

	// SenderBuilder describes the ability of an entity to build sender links
	SenderBuilder interface {
		NewSender(ctx context.Context, opts ...SenderOption) (*Sender, error)
	}

	// EntityManagementAddresser describes the ability of an entity to provide an addressable path to it's management
	// endpoint
	EntityManagementAddresser interface {
		ManagementPath() string
	}

	// SendAndReceiveBuilder is a ReceiverBuilder, SenderBuilder and EntityManagementAddresser
	SendAndReceiveBuilder interface {
		ReceiveBuilder
		SenderBuilder
	}

	// ReceiveBuilder is a ReceiverBuilder and EntityManagementAddresser
	ReceiveBuilder interface {
		ReceiverBuilder
		EntityManagementAddresser
	}
)

// newSession is a constructor for a Service Bus session which will pre-populate the SessionID with a new UUID
func newSession(amqpSession *amqp.Session) (*session, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	return &session{
		Session:   amqpSession,
		SessionID: id.String(),
		counter:   0,
	}, nil
}

// getNext gets and increments the next group sequence number for the session
func (s *session) getNext() uint32 {
	return atomic.AddUint32(&s.counter, 1)
}

func (s *session) String() string {
	return s.SessionID
}

// NewQueueSession creates a new session sender and receiver to communicate with a Service Bus queue.
//
// Microsoft Azure Service Bus sessions enable joint and ordered handling of unbounded sequences of related messages.
// To realize a FIFO guarantee in Service Bus, use Sessions. Service Bus is not prescriptive about the nature of the
// relationship between the messages, and also does not define a particular model for determining where a message
// sequence starts or ends.
func NewQueueSession(builder SendAndReceiveBuilder, sessionID *string) *QueueSession {
	return &QueueSession{
		sessionID: sessionID,
		builder:   builder,
	}
}

// ReceiveOne waits for the lock on a particular session to become available, takes it, then process the session.
// The session can contain multiple messages. ReceiveOneSession will receive all messages within that session.
//
// Handler must call a disposition action such as Complete, Abandon, Deadletter on the message. If the messages does not
// have a disposition set, the Queue's DefaultDisposition will be used.
//
// If the handler returns an error, the receive loop will be terminated.
func (qs *QueueSession) ReceiveOne(ctx context.Context, handler SessionHandler) error {
	span, ctx := startConsumerSpanFromContext(ctx, "sb.QueueSession.ReceiveOneSession")
	defer span.Finish()

	if err := qs.ensureReceiver(ctx); err != nil {
		return err
	}

	ms, err := newMessageSession(qs.receiver, qs.builder, qs.sessionID)
	if err != nil {
		return err
	}

	err = handler.Start(ms)
	if err != nil {
		return err
	}

	defer handler.End()
	handle := qs.receiver.Listen(ctx, handler)

	select {
	case <-handle.Done():
		return handle.Err()
	case <-ms.done:
		return nil
	}
}

// Send the message to the queue within a session
func (qs *QueueSession) Send(ctx context.Context, msg *Message) error {
	if err := qs.ensureSender(ctx); err != nil {
		return err
	}

	if msg.SessionID == nil {
		msg.SessionID = qs.sessionID
	}
	return qs.sender.Send(ctx, msg)
}

// Close the underlying connection to Service Bus
func (qs *QueueSession) Close(ctx context.Context) error {
	if qs.receiver != nil {
		if err := qs.receiver.Close(ctx); err != nil {
			log.For(ctx).Error(err)
			if qs.sender != nil {
				if senderErr := qs.sender.Close(ctx); err != nil {
					log.For(ctx).Error(senderErr)
				}
			}
			return err
		}
	}

	if qs.sender != nil {
		if err := qs.sender.Close(ctx); err != nil {
			log.For(ctx).Error(err)
			return err
		}
	}
	return nil
}

// SessionID is the identifier for the Service Bus session
func (qs *QueueSession) SessionID() *string {
	return qs.sessionID
}

func (qs *QueueSession) ensureSender(ctx context.Context) error {
	qs.builderMu.Lock()
	defer qs.builderMu.Unlock()

	s, err := qs.builder.NewSender(ctx, SenderWithSession(qs.sessionID))
	if err != nil {
		return err
	}

	qs.sender = s
	return nil
}

func (qs *QueueSession) ensureReceiver(ctx context.Context) error {
	qs.builderMu.Lock()
	defer qs.builderMu.Unlock()

	r, err := qs.builder.NewReceiver(ctx, ReceiverWithSession(qs.sessionID))
	if err != nil {
		return err
	}

	qs.receiver = r
	return nil
}

// NewSubscriptionSession creates a new session receiver to receive from a Service Bus subscription.
//
// Microsoft Azure Service Bus sessions enable joint and ordered handling of unbounded sequences of related messages.
// To realize a FIFO guarantee in Service Bus, use Sessions. Service Bus is not prescriptive about the nature of the
// relationship between the messages, and also does not define a particular model for determining where a message
// sequence starts or ends.
func NewSubscriptionSession(builder ReceiveBuilder, sessionID *string) *SubscriptionSession {
	return &SubscriptionSession{
		sessionID: sessionID,
		builder:   builder,
	}
}

// ReceiveOne waits for the lock on a particular session to become available, takes it, then process the session.
// The session can contain multiple messages. ReceiveOneSession will receive all messages within that session.
//
// Handler must call a disposition action such as Complete, Abandon, Deadletter on the message. If the messages does not
// have a disposition set, the Queue's DefaultDisposition will be used.
//
// If the handler returns an error, the receive loop will be terminated.
func (ss *SubscriptionSession) ReceiveOne(ctx context.Context, handler SessionHandler) error {
	span, ctx := startConsumerSpanFromContext(ctx, "sb.SubscriptionSession.ReceiveOneSession")
	defer span.Finish()

	if err := ss.ensureReceiver(ctx); err != nil {
		return err
	}

	ms, err := newMessageSession(ss.receiver, ss.builder, ss.sessionID)
	if err != nil {
		return err
	}

	err = handler.Start(ms)
	if err != nil {
		return err
	}

	defer handler.End()
	handle := ss.receiver.Listen(ctx, handler)

	select {
	case <-handle.Done():
		return handle.Err()
	case <-ms.done:
		return nil
	}
}

// Close the underlying connection to Service Bus
func (ss *SubscriptionSession) Close(ctx context.Context) error {
	if ss.receiver != nil {
		return ss.receiver.Close(ctx)
	}
	return nil
}

func (ss *SubscriptionSession) ensureReceiver(ctx context.Context) error {
	ss.builderMu.Lock()
	defer ss.builderMu.Unlock()

	r, err := ss.builder.NewReceiver(ctx, ReceiverWithSession(ss.sessionID))
	if err != nil {
		return err
	}

	ss.receiver = r
	return nil
}

// SessionID is the identifier for the Service Bus session
func (ss *SubscriptionSession) SessionID() *string {
	return ss.sessionID
}

// NewTopicSession creates a new session receiver to receive from a Service Bus topic.
//
// Microsoft Azure Service Bus sessions enable joint and ordered handling of unbounded sequences of related messages.
// To realize a FIFO guarantee in Service Bus, use Sessions. Service Bus is not prescriptive about the nature of the
// relationship between the messages, and also does not define a particular model for determining where a message
// sequence starts or ends.
func NewTopicSession(builder SenderBuilder, sessionID *string) *TopicSession {
	return &TopicSession{
		sessionID: sessionID,
		builder:   builder,
	}
}

// Send the message to the queue within a session
func (ts *TopicSession) Send(ctx context.Context, msg *Message) error {
	if err := ts.ensureSender(ctx); err != nil {
		return err
	}

	if msg.SessionID == nil {
		msg.SessionID = ts.sessionID
	}
	return ts.sender.Send(ctx, msg)
}

// Close the underlying connection to Service Bus
func (ts *TopicSession) Close(ctx context.Context) error {
	if ts.sender != nil {
		if err := ts.sender.Close(ctx); err != nil {
			log.For(ctx).Error(err)
			return err
		}
	}
	return nil
}

// SessionID is the identifier for the Service Bus session
func (ts *TopicSession) SessionID() *string {
	return ts.sessionID
}

func (ts *TopicSession) ensureSender(ctx context.Context) error {
	ts.builderMu.Lock()
	defer ts.builderMu.Unlock()

	s, err := ts.builder.NewSender(ctx, SenderWithSession(ts.sessionID))
	if err != nil {
		return err
	}

	ts.sender = s
	return nil
}
