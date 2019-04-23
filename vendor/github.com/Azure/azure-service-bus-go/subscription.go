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
	"strings"
	"sync"

	"github.com/Azure/azure-amqp-common-go/log"
	"pack.ag/amqp"
)

type (
	// Subscription represents a Service Bus Subscription entity which are used to receive topic messages. A topic
	// subscription resembles a virtual queue that receives copies of the messages that are sent to the topic.
	//Messages are received from a subscription identically to the way they are received from a queue.
	Subscription struct {
		*entity
		Topic             *Topic
		receiver          *Receiver
		receiverMu        sync.Mutex
		receiveMode       ReceiveMode
		requiredSessionID *string
		prefetchCount     *uint32
	}

	// SubscriptionOption configures the Subscription Azure Service Bus client
	SubscriptionOption func(*Subscription) error
)

// SubscriptionWithReceiveAndDelete configures a subscription to pop and delete messages off of the queue upon receiving the message.
// This differs from the default, PeekLock, where PeekLock receives a message, locks it for a period of time, then sends
// a disposition to the broker when the message has been processed.
func SubscriptionWithReceiveAndDelete() SubscriptionOption {
	return func(s *Subscription) error {
		s.receiveMode = ReceiveAndDeleteMode
		return nil
	}
}

// SubscriptionWithPrefetchCount configures the subscription to attempt to fetch the number of messages specified by the
// prefetch count at one time.
//
// The default is 1 message at a time.
//
// Caution: Using PeekLock, messages have a set lock timeout, which can be renewed. By setting a high prefetch count, a
// local queue of messages could build up and cause message locks to expire before the message lands in the handler. If
// this happens, the message disposition will fail and will be re-queued and processed again.
func SubscriptionWithPrefetchCount(prefetch uint32) SubscriptionOption {
	return func(q *Subscription) error {
		q.prefetchCount = &prefetch
		return nil
	}
}

// NewSubscription creates a new Topic Subscription client
func (t *Topic) NewSubscription(name string, opts ...SubscriptionOption) (*Subscription, error) {
	sub := &Subscription{
		entity: &entity{
			namespace: t.namespace,
			Name:      name,
		},
		Topic: t,
	}

	for i := range opts {
		if err := opts[i](sub); err != nil {
			return nil, err
		}
	}
	return sub, nil
}

// Peek fetches a list of Messages from the Service Bus broker, with-out acquiring a lock or committing to a
// disposition. The messages are delivered as close to sequence order as possible.
//
// The MessageIterator that is returned has the following properties:
// - Messages are fetches from the server in pages. Page size is configurable with PeekOptions.
// - The MessageIterator will always return "false" for Done().
// - When Next() is called, it will return either: a slice of messages and no error, nil with an error related to being
// unable to complete the operation, or an empty slice of messages and an instance of "ErrNoMessages" signifying that
// there are currently no messages in the subscription with a sequence ID larger than previously viewed ones.
func (s *Subscription) Peek(ctx context.Context, options ...PeekOption) (MessageIterator, error) {
	err := s.ensureReceiver(ctx)
	if err != nil {
		return nil, err
	}

	return newPeekIterator(s, options...)
}

// PeekOne fetches a single Message from the Service Bus broker without acquiring a lock or committing to a disposition.
func (s *Subscription) PeekOne(ctx context.Context, options ...PeekOption) (*Message, error) {
	err := s.ensureReceiver(ctx)
	if err != nil {
		return nil, err
	}

	// Adding PeekWithPageSize(1) as the last option assures that either:
	// - creating the iterator will fail because two of the same option will be applied.
	// - PeekWithPageSize(1) will be applied after all others, so we will not wastefully pull down messages destined to
	//   be unread.
	options = append(options, PeekWithPageSize(1))

	it, err := newPeekIterator(s, options...)
	if err != nil {
		return nil, err
	}
	return it.Next(ctx)
}

// ReceiveOne will listen to receive a single message. ReceiveOne will only wait as long as the context allows.
//
// Handler must call a disposition action such as Complete, Abandon, Deadletter on the message. If the messages does not
// have a disposition set, the Queue's DefaultDisposition will be used.
func (s *Subscription) ReceiveOne(ctx context.Context, handler Handler) error {
	span, ctx := s.startSpanFromContext(ctx, "sb.Subscription.ReceiveOne")
	defer span.Finish()

	if err := s.ensureReceiver(ctx); err != nil {
		return err
	}

	return s.receiver.ReceiveOne(ctx, handler)
}

// Receive subscribes for messages sent to the Subscription
//
// Handler must call a disposition action such as Complete, Abandon, Deadletter on the message. If the messages does not
// have a disposition set, the Queue's DefaultDisposition will be used.
//
// If the handler returns an error, the receive loop will be terminated.
func (s *Subscription) Receive(ctx context.Context, handler Handler) error {
	span, ctx := s.startSpanFromContext(ctx, "sb.Subscription.Receive")
	defer span.Finish()

	if err := s.ensureReceiver(ctx); err != nil {
		return err
	}
	handle := s.receiver.Listen(ctx, handler)
	<-handle.Done()
	return handle.Err()
}

// ReceiveDeferred will receive and handle a set of deferred messages
//
// When a queue or subscription client receives a message that it is willing to process, but for which processing is
// not currently possible due to special circumstances inside of the application, it has the option of "deferring"
// retrieval of the message to a later point. The message remains in the queue or subscription, but it is set aside.
//
// Deferral is a feature specifically created for workflow processing scenarios. Workflow frameworks may require certain
// operations to be processed in a particular order, and may have to postpone processing of some received messages
// until prescribed prior work that is informed by other messages has been completed.
//
// A simple illustrative example is an order processing sequence in which a payment notification from an external
// payment provider appears in a system before the matching purchase order has been propagated from the store front
// to the fulfillment system. In that case, the fulfillment system might defer processing the payment notification
// until there is an order with which to associate it. In rendezvous scenarios, where messages from different sources
// drive a workflow forward, the real-time execution order may indeed be correct, but the messages reflecting the
// outcomes may arrive out of order.
//
// Ultimately, deferral aids in reordering messages from the arrival order into an order in which they can be
// processed, while leaving those messages safely in the message store for which processing needs to be postponed.
func (s *Subscription) ReceiveDeferred(ctx context.Context, handler Handler, sequenceNumbers ...int64) error {
	return receiveDeferred(ctx, s, handler, sequenceNumbers...)
}

// NewSession will create a new session based receiver for the subscription
//
// Microsoft Azure Service Bus sessions enable joint and ordered handling of unbounded sequences of related messages.
// To realize a FIFO guarantee in Service Bus, use Sessions. Service Bus is not prescriptive about the nature of the
// relationship between the messages, and also does not define a particular model for determining where a message
// sequence starts or ends.
func (s *Subscription) NewSession(sessionID *string) *SubscriptionSession {
	return NewSubscriptionSession(s, sessionID)
}

// NewReceiver will create a new Receiver for receiving messages off of the queue
func (s *Subscription) NewReceiver(ctx context.Context, opts ...ReceiverOption) (*Receiver, error) {
	span, ctx := s.startSpanFromContext(ctx, "sb.Subscription.NewReceiver")
	defer span.Finish()

	opts = append(opts, ReceiverWithReceiveMode(s.receiveMode))

	if s.prefetchCount != nil {
		opts = append(opts, ReceiverWithPrefetchCount(*s.prefetchCount))
	}

	return s.namespace.NewReceiver(ctx, s.Topic.Name+"/Subscriptions/"+s.Name, opts...)
}

// NewDeadLetter creates an entity that represents the dead letter sub queue of the queue
//
// Azure Service Bus queues and topic subscriptions provide a secondary sub-queue, called a dead-letter queue
// (DLQ). The dead-letter queue does not need to be explicitly created and cannot be deleted or otherwise managed
// independent of the main entity.
//
// The purpose of the dead-letter queue is to hold messages that cannot be delivered to any receiver, or messages
// that could not be processed. Messages can then be removed from the DLQ and inspected. An application might, with
// help of an operator, correct issues and resubmit the message, log the fact that there was an error, and take
// corrective action.
//
// From an API and protocol perspective, the DLQ is mostly similar to any other queue, except that messages can only
// be submitted via the dead-letter operation of the parent entity. In addition, time-to-live is not observed, and
// you can't dead-letter a message from a DLQ. The dead-letter queue fully supports peek-lock delivery and
// transactional operations.
//
// Note that there is no automatic cleanup of the DLQ. Messages remain in the DLQ until you explicitly retrieve
// them from the DLQ and call Complete() on the dead-letter message.
func (s *Subscription) NewDeadLetter() *DeadLetter {
	return NewDeadLetter(s)
}

// NewDeadLetterReceiver builds a receiver for the Subscriptions's dead letter queue
func (s *Subscription) NewDeadLetterReceiver(ctx context.Context, opts ...ReceiverOption) (ReceiveOner, error) {
	span, ctx := s.startSpanFromContext(ctx, "sb.Subscription.NewDeadLetterReceiver")
	defer span.Finish()

	deadLetterEntityPath := strings.Join([]string{s.Topic.Name, "Subscriptions", s.Name, DeadLetterQueueName}, "/")
	return s.namespace.NewReceiver(ctx, deadLetterEntityPath, opts...)
}

// NewTransferDeadLetter creates an entity that represents the transfer dead letter sub queue of the subscription
//
// Messages will be sent to the transfer dead-letter queue under the following conditions:
//   - A message passes through more than 3 queues or topics that are chained together.
//   - The destination queue or topic is disabled or deleted.
//   - The destination queue or topic exceeds the maximum entity size.
func (s *Subscription) NewTransferDeadLetter() *TransferDeadLetter {
	return NewTransferDeadLetter(s)
}

// NewTransferDeadLetterReceiver builds a receiver for the Queue's transfer dead letter queue
//
// Messages will be sent to the transfer dead-letter queue under the following conditions:
//   - A message passes through more than 3 queues or topics that are chained together.
//   - The destination queue or topic is disabled or deleted.
//   - The destination queue or topic exceeds the maximum entity size.
func (s *Subscription) NewTransferDeadLetterReceiver(ctx context.Context, opts ...ReceiverOption) (ReceiveOner, error) {
	span, ctx := s.startSpanFromContext(ctx, "sb.Subscription.NewTransferDeadLetterReceiver")
	defer span.Finish()

	transferDeadLetterEntityPath := strings.Join([]string{s.Topic.Name, "subscriptions", s.Name, TransferDeadLetterQueueName}, "/")
	return s.namespace.NewReceiver(ctx, transferDeadLetterEntityPath, opts...)
}

// RenewLocks renews the locks on messages provided
func (s *Subscription) RenewLocks(ctx context.Context, messages ...*Message) error {
	return renewLocks(ctx, s, messages...)
}

// Close the underlying connection to Service Bus
func (s *Subscription) Close(ctx context.Context) error {
	if s.receiver != nil {
		err := s.receiver.Close(ctx)
		if err != nil && !isConnectionClosed(err) {
			log.For(ctx).Error(err)
			return err
		}
	}
	return nil
}

// ManagementPath is the relative uri to address the entity's management functionality
func (s *Subscription) ManagementPath() string {
	return strings.Join([]string{s.Topic.Name, "subscriptions", s.Name, "$management"}, "/")
}

func (s *Subscription) ensureReceiver(ctx context.Context, opts ...ReceiverOption) error {
	span, ctx := s.startSpanFromContext(ctx, "sb.Subscription.ensureReceiver")
	defer span.Finish()

	s.receiverMu.Lock()
	defer s.receiverMu.Unlock()

	// if a receiver is already in established, just return
	if s.receiver != nil {
		return nil
	}

	receiver, err := s.NewReceiver(ctx, opts...)
	if err != nil {
		log.For(ctx).Error(err)
		return err
	}

	s.receiver = receiver
	return nil
}

func (s *Subscription) connection(ctx context.Context) (*amqp.Client, error) {
	span, ctx := s.startSpanFromContext(ctx, "sb.Subscription.connection")
	defer span.Finish()

	if err := s.ensureReceiver(ctx); err != nil {
		return nil, err
	}
	return s.receiver.connection, nil
}
