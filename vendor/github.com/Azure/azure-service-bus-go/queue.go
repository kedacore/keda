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
	"encoding/xml"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-amqp-common-go/log"
	"github.com/Azure/azure-amqp-common-go/rpc"
	"github.com/Azure/azure-amqp-common-go/uuid"
	"github.com/Azure/go-autorest/autorest/date"
	"pack.ag/amqp"
)

type (
	entity struct {
		Name                  string
		namespace             *Namespace
		renewMessageLockMutex sync.Mutex
	}

	// Queue represents a Service Bus Queue entity, which offers First In, First Out (FIFO) message delivery to one or
	// more competing consumers. That is, messages are typically expected to be received and processed by the receivers
	// in the order in which they were added to the queue, and each message is received and processed by only one
	// message consumer.
	Queue struct {
		*entity
		sender            *Sender
		receiver          *Receiver
		receiverMu        sync.Mutex
		senderMu          sync.Mutex
		receiveMode       ReceiveMode
		requiredSessionID *string
		prefetchCount     *uint32
	}

	// queueContent is a specialized Queue body for an Atom entry
	queueContent struct {
		XMLName          xml.Name         `xml:"content"`
		Type             string           `xml:"type,attr"`
		QueueDescription QueueDescription `xml:"QueueDescription"`
	}

	// QueueDescription is the content type for Queue management requests
	QueueDescription struct {
		XMLName xml.Name `xml:"QueueDescription"`
		BaseEntityDescription
		LockDuration                        *string       `xml:"LockDuration,omitempty"`               // LockDuration - ISO 8601 timespan duration of a peek-lock; that is, the amount of time that the message is locked for other receivers. The maximum value for LockDuration is 5 minutes; the default value is 1 minute.
		MaxSizeInMegabytes                  *int32        `xml:"MaxSizeInMegabytes,omitempty"`         // MaxSizeInMegabytes - The maximum size of the queue in megabytes, which is the size of memory allocated for the queue. Default is 1024.
		RequiresDuplicateDetection          *bool         `xml:"RequiresDuplicateDetection,omitempty"` // RequiresDuplicateDetection - A value indicating if this queue requires duplicate detection.
		RequiresSession                     *bool         `xml:"RequiresSession,omitempty"`
		DefaultMessageTimeToLive            *string       `xml:"DefaultMessageTimeToLive,omitempty"`            // DefaultMessageTimeToLive - ISO 8601 default message timespan to live value. This is the duration after which the message expires, starting from when the message is sent to Service Bus. This is the default value used when TimeToLive is not set on a message itself.
		DeadLetteringOnMessageExpiration    *bool         `xml:"DeadLetteringOnMessageExpiration,omitempty"`    // DeadLetteringOnMessageExpiration - A value that indicates whether this queue has dead letter support when a message expires.
		DuplicateDetectionHistoryTimeWindow *string       `xml:"DuplicateDetectionHistoryTimeWindow,omitempty"` // DuplicateDetectionHistoryTimeWindow - ISO 8601 timeSpan structure that defines the duration of the duplicate detection history. The default value is 10 minutes.
		MaxDeliveryCount                    *int32        `xml:"MaxDeliveryCount,omitempty"`                    // MaxDeliveryCount - The maximum delivery count. A message is automatically deadlettered after this number of deliveries. default value is 10.
		EnableBatchedOperations             *bool         `xml:"EnableBatchedOperations,omitempty"`             // EnableBatchedOperations - Value that indicates whether server-side batched operations are enabled.
		SizeInBytes                         *int64        `xml:"SizeInBytes,omitempty"`                         // SizeInBytes - The size of the queue, in bytes.
		MessageCount                        *int64        `xml:"MessageCount,omitempty"`                        // MessageCount - The number of messages in the queue.
		IsAnonymousAccessible               *bool         `xml:"IsAnonymousAccessible,omitempty"`
		Status                              *EntityStatus `xml:"Status,omitempty"`
		CreatedAt                           *date.Time    `xml:"CreatedAt,omitempty"`
		UpdatedAt                           *date.Time    `xml:"UpdatedAt,omitempty"`
		SupportOrdering                     *bool         `xml:"SupportOrdering,omitempty"`
		AutoDeleteOnIdle                    *string       `xml:"AutoDeleteOnIdle,omitempty"`
		EnablePartitioning                  *bool         `xml:"EnablePartitioning,omitempty"`
		EnableExpress                       *bool         `xml:"EnableExpress,omitempty"`
		CountDetails                        *CountDetails `xml:"CountDetails,omitempty"`
		ForwardTo                           *string       `xml:"ForwardTo,omitempty"`
		ForwardDeadLetteredMessagesTo       *string       `xml:"ForwardDeadLetteredMessagesTo,omitempty"` // ForwardDeadLetteredMessagesTo - absolute URI of the entity to forward dead letter messages
	}

	// QueueOption represents named options for assisting Queue message handling
	QueueOption func(*Queue) error

	// ReceiveMode represents the behavior when consuming a message from a queue
	ReceiveMode int

	entityConnector interface {
		EntityManagementAddresser
		connection(ctx context.Context) (*amqp.Client, error)
	}
)

const (
	// PeekLockMode causes a Receiver to peek at a message, lock it so no others can consume and have the queue wait for
	// the DispositionAction
	PeekLockMode ReceiveMode = 0
	// ReceiveAndDeleteMode causes a Receiver to pop messages off of the queue without waiting for DispositionAction
	ReceiveAndDeleteMode ReceiveMode = 1

	// DeadLetterQueueName is the name of the dead letter queue to be appended to the entity path
	DeadLetterQueueName = "$DeadLetterQueue"

	// TransferDeadLetterQueueName is the name of the transfer dead letter queue which is appended to the entity name to
	// build the full address of the transfer dead letter queue.
	TransferDeadLetterQueueName = "$Transfer/" + DeadLetterQueueName
)

// QueueWithReceiveAndDelete configures a queue to pop and delete messages off of the queue upon receiving the message.
// This differs from the default, PeekLock, where PeekLock receives a message, locks it for a period of time, then sends
// a disposition to the broker when the message has been processed.
func QueueWithReceiveAndDelete() QueueOption {
	return func(q *Queue) error {
		q.receiveMode = ReceiveAndDeleteMode
		return nil
	}
}

// QueueWithPrefetchCount configures the queue to attempt to fetch the number of messages specified by the
// prefetch count at one time.
//
// The default is 1 message at a time.
//
// Caution: Using PeekLock, messages have a set lock timeout, which can be renewed. By setting a high prefetch count, a
// local queue of messages could build up and cause message locks to expire before the message lands in the handler. If
// this happens, the message disposition will fail and will be re-queued and processed again.
func QueueWithPrefetchCount(prefetch uint32) QueueOption {
	return func(q *Queue) error {
		q.prefetchCount = &prefetch
		return nil
	}
}

// NewQueue creates a new Queue Sender / Receiver
func (ns *Namespace) NewQueue(name string, opts ...QueueOption) (*Queue, error) {
	queue := &Queue{
		entity: &entity{
			namespace: ns,
			Name:      name,
		},
		receiveMode: PeekLockMode,
	}

	for _, opt := range opts {
		if err := opt(queue); err != nil {
			return nil, err
		}
	}
	return queue, nil
}

// Send sends messages to the Queue
func (q *Queue) Send(ctx context.Context, msg *Message) error {
	span, ctx := q.startSpanFromContext(ctx, "sb.Queue.Send")
	defer span.Finish()

	err := q.ensureSender(ctx)
	if err != nil {
		log.For(ctx).Error(err)
		return err
	}
	return q.sender.Send(ctx, msg)
}

// SendBatch sends a batch of messages to the Queue
func (q *Queue) SendBatch(ctx context.Context, iterator BatchIterator) error {
	span, ctx := q.startSpanFromContext(ctx, "sb.Queue.SendBatch")
	defer span.Finish()

	err := q.ensureSender(ctx)
	if err != nil {
		log.For(ctx).Error(err)
		return err
	}

	for !iterator.Done() {
		id, err := uuid.NewV4()
		if err != nil {
			log.For(ctx).Error(err)
			return err
		}

		batch, err := iterator.Next(id.String(), &BatchOptions{
			SessionID: q.sender.sessionID,
		})
		if err != nil {
			log.For(ctx).Error(err)
			return err
		}

		if err := q.sender.trySend(ctx, batch); err != nil {
			log.For(ctx).Error(err)
			return err
		}
	}

	return nil
}

// ScheduleAt will send a batch of messages to a Queue, schedule them to be enqueued, and return the sequence numbers
// that can be used to cancel each message.
func (q *Queue) ScheduleAt(ctx context.Context, enqueueTime time.Time, messages ...*Message) ([]int64, error) {
	if len(messages) <= 0 {
		return nil, errors.New("expected one or more messages")
	}

	transformed := make([]interface{}, 0, len(messages))
	for i := range messages {
		messages[i].ScheduleAt(enqueueTime)

		if messages[i].ID == "" {
			id, err := uuid.NewV4()
			if err != nil {
				return nil, err
			}
			messages[i].ID = id.String()
		}

		rawAmqp, err := messages[i].toMsg()
		if err != nil {
			return nil, err
		}
		encoded, err := rawAmqp.MarshalBinary()
		if err != nil {
			return nil, err
		}

		individualMessage := map[string]interface{}{
			"message-id": messages[i].ID,
			"message":    encoded,
		}
		if messages[i].SessionID != nil {
			individualMessage["session-id"] = *messages[i].SessionID
		}
		if partitionKey := messages[i].SystemProperties.PartitionKey; partitionKey != nil {
			individualMessage["partition-key"] = *partitionKey
		}
		if viaPartitionKey := messages[i].SystemProperties.ViaPartitionKey; viaPartitionKey != nil {
			individualMessage["via-partition-key"] = *viaPartitionKey
		}

		transformed = append(transformed, individualMessage)
	}

	msg := &amqp.Message{
		ApplicationProperties: map[string]interface{}{
			operationFieldName: scheduleMessageOperationID,
		},
		Value: map[string]interface{}{
			"messages": transformed,
		},
	}

	if deadline, ok := ctx.Deadline(); ok {
		msg.ApplicationProperties[serverTimeoutFieldName] = uint(time.Until(deadline) / time.Millisecond)
	}

	err := q.ensureSender(ctx)
	if err != nil {
		return nil, err
	}

	link, err := rpc.NewLink(q.sender.connection, q.ManagementPath())
	if err != nil {
		return nil, err
	}

	resp, err := link.RetryableRPC(ctx, 5, 5*time.Second, msg)
	if err != nil {
		return nil, err
	}

	if resp.Code != 200 {
		return nil, ErrAMQP(*resp)
	}

	retval := make([]int64, 0, len(messages))

	if rawVal, ok := resp.Message.Value.(map[string]interface{}); ok {
		const sequenceFieldName = "sequence-numbers"
		if rawArr, ok := rawVal[sequenceFieldName]; ok {
			if arr, ok := rawArr.([]int64); ok {
				for i := range arr {
					retval = append(retval, arr[i])
				}
				return retval, nil
			}
			return nil, newErrIncorrectType(sequenceFieldName, []int64{}, rawArr)
		}
		return nil, ErrMissingField(sequenceFieldName)
	}
	return nil, newErrIncorrectType("value", map[string]interface{}{}, resp.Message.Value)
}

// CancelScheduled allows for removal of messages that have been handed to the Service Bus broker for later delivery,
// but have not yet ben enqueued.
func (q *Queue) CancelScheduled(ctx context.Context, seq ...int64) error {
	msg := &amqp.Message{
		ApplicationProperties: map[string]interface{}{
			operationFieldName: cancelScheduledOperationID,
		},
		Value: map[string]interface{}{
			"sequence-numbers": seq,
		},
	}

	if deadline, ok := ctx.Deadline(); ok {
		msg.ApplicationProperties[serverTimeoutFieldName] = uint(time.Until(deadline) / time.Millisecond)
	}

	err := q.ensureSender(ctx)
	if err != nil {
		return err
	}

	link, err := rpc.NewLink(q.sender.connection, q.ManagementPath())
	if err != nil {
		return err
	}

	resp, err := link.RetryableRPC(ctx, 5, 5*time.Second, msg)
	if err != nil {
		return err
	}

	if resp.Code != 200 {
		return ErrAMQP(*resp)
	}

	return nil
}

// Peek fetches a list of Messages from the Service Bus broker without acquiring a lock or committing to a disposition.
// The messages are delivered as close to sequence order as possible.
//
// The MessageIterator that is returned has the following properties:
// - Messages are fetches from the server in pages. Page size is configurable with PeekOptions.
// - The MessageIterator will always return "false" for Done().
// - When Next() is called, it will return either: a slice of messages and no error, nil with an error related to being
// unable to complete the operation, or an empty slice of messages and an instance of "ErrNoMessages" signifying that
// there are currently no messages in the queue with a sequence ID larger than previously viewed ones.
func (q *Queue) Peek(ctx context.Context, options ...PeekOption) (MessageIterator, error) {
	err := q.ensureReceiver(ctx)
	if err != nil {
		return nil, err
	}

	return newPeekIterator(q, options...)
}

// PeekOne fetches a single Message from the Service Bus broker without acquiring a lock or committing to a disposition.
func (q *Queue) PeekOne(ctx context.Context, options ...PeekOption) (*Message, error) {
	err := q.ensureReceiver(ctx)
	if err != nil {
		return nil, err
	}

	// Adding PeekWithPageSize(1) as the last option assures that either:
	// - creating the iterator will fail because two of the same option will be applied.
	// - PeekWithPageSize(1) will be applied after all others, so we will not wastefully pull down messages destined to
	//   be unread.
	options = append(options, PeekWithPageSize(1))

	it, err := newPeekIterator(q, options...)
	if err != nil {
		return nil, err
	}
	return it.Next(ctx)
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
func (q *Queue) ReceiveDeferred(ctx context.Context, handler Handler, sequenceNumbers ...int64) error {
	return receiveDeferred(ctx, q, handler, sequenceNumbers...)
}

// ReceiveOne will listen to receive a single message. ReceiveOne will only wait as long as the context allows.
//
// Handler must call a disposition action such as Complete, Abandon, Deadletter on the message. If the messages does not
// have a disposition set, the Queue's DefaultDisposition will be used.
func (q *Queue) ReceiveOne(ctx context.Context, handler Handler) error {
	span, ctx := q.startSpanFromContext(ctx, "sb.Queue.ReceiveOne")
	defer span.Finish()

	if err := q.ensureReceiver(ctx); err != nil {
		return err
	}

	return q.receiver.ReceiveOne(ctx, handler)
}

// Receive subscribes for messages sent to the Queue. If the messages not within a session, messages will arrive
// unordered.
//
// Handler must call a disposition action such as Complete, Abandon, Deadletter on the message. If the messages does not
// have a disposition set, the Queue's DefaultDisposition will be used.
//
// If the handler returns an error, the receive loop will be terminated.
func (q *Queue) Receive(ctx context.Context, handler Handler) error {
	span, ctx := q.startSpanFromContext(ctx, "sb.Queue.Receive")
	defer span.Finish()

	err := q.ensureReceiver(ctx)
	if err != nil {
		return err
	}

	handle := q.receiver.Listen(ctx, handler)
	<-handle.Done()
	return handle.Err()
}

// NewSession will create a new session based receiver and sender for the queue
//
// Microsoft Azure Service Bus sessions enable joint and ordered handling of unbounded sequences of related messages.
// To realize a FIFO guarantee in Service Bus, use Sessions. Service Bus is not prescriptive about the nature of the
// relationship between the messages, and also does not define a particular model for determining where a message
// sequence starts or ends.
func (q *Queue) NewSession(sessionID *string) *QueueSession {
	return NewQueueSession(q, sessionID)
}

// NewReceiver will create a new Receiver for receiving messages off of a queue
func (q *Queue) NewReceiver(ctx context.Context, opts ...ReceiverOption) (*Receiver, error) {
	span, ctx := q.startSpanFromContext(ctx, "sb.Queue.NewReceiver")
	defer span.Finish()

	opts = append(opts, ReceiverWithReceiveMode(q.receiveMode))
	return q.namespace.NewReceiver(ctx, q.Name, opts...)
}

// NewSender will create a new Sender for sending messages to the queue
func (q *Queue) NewSender(ctx context.Context, opts ...SenderOption) (*Sender, error) {
	span, ctx := q.startSpanFromContext(ctx, "sb.Queue.NewSender")
	defer span.Finish()

	return q.namespace.NewSender(ctx, q.Name)
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
func (q *Queue) NewDeadLetter() *DeadLetter {
	return NewDeadLetter(q)
}

// NewDeadLetterReceiver builds a receiver for the Queue's dead letter queue
func (q *Queue) NewDeadLetterReceiver(ctx context.Context, opts ...ReceiverOption) (ReceiveOner, error) {
	span, ctx := q.startSpanFromContext(ctx, "sb.Queue.NewDeadLetterReceiver")
	defer span.Finish()

	deadLetterEntityPath := strings.Join([]string{q.Name, DeadLetterQueueName}, "/")
	return q.namespace.NewReceiver(ctx, deadLetterEntityPath, opts...)
}

// NewTransferDeadLetter creates an entity that represents the transfer dead letter sub queue of the queue
//
// Messages will be sent to the transfer dead-letter queue under the following conditions:
//   - A message passes through more than 3 queues or topics that are chained together.
//   - The destination queue or topic is disabled or deleted.
//   - The destination queue or topic exceeds the maximum entity size.
func (q *Queue) NewTransferDeadLetter() *TransferDeadLetter {
	return NewTransferDeadLetter(q)
}

// NewTransferDeadLetterReceiver builds a receiver for the Queue's transfer dead letter queue
//
// Messages will be sent to the transfer dead-letter queue under the following conditions:
//   - A message passes through more than 3 queues or topics that are chained together.
//   - The destination queue or topic is disabled or deleted.
//   - The destination queue or topic exceeds the maximum entity size.
func (q *Queue) NewTransferDeadLetterReceiver(ctx context.Context, opts ...ReceiverOption) (ReceiveOner, error) {
	span, ctx := q.startSpanFromContext(ctx, "sb.Queue.NewTransferDeadLetterReceiver")
	defer span.Finish()

	transferDeadLetterEntityPath := strings.Join([]string{q.Name, TransferDeadLetterQueueName}, "/")
	return q.namespace.NewReceiver(ctx, transferDeadLetterEntityPath, opts...)
}

// RenewLocks renews the locks on messages provided
func (q *Queue) RenewLocks(ctx context.Context, messages ...*Message) error {
	return renewLocks(ctx, q, messages...)
}

func receiveDeferred(ctx context.Context, ec entityConnector, handler Handler, sequenceNumbers ...int64) error {
	span, ctx := startConsumerSpanFromContext(ctx, "sb.receiveDeferred")
	defer span.Finish()

	const messagesField, messageField = "messages", "message"
	msg := &amqp.Message{
		ApplicationProperties: map[string]interface{}{
			operationFieldName: "com.microsoft:receive-by-sequence-number",
		},
		Value: map[string]interface{}{
			"sequence-numbers":     sequenceNumbers,
			"receiver-settle-mode": uint32(1), // pick up messages with peek lock
		},
	}

	conn, err := ec.connection(ctx)
	if err != nil {
		return err
	}

	link, err := rpc.NewLink(conn, ec.ManagementPath())
	if err != nil {
		return err
	}

	rsp, err := link.RetryableRPC(ctx, 5, 5*time.Second, msg)
	if err != nil {
		return err
	}

	if rsp.Code == 204 {
		return ErrNoMessages{}
	}

	// Deferred messages come back in a relatively convoluted manner:
	// a map (always with one key: "messages")
	// 	of arrays
	// 		of maps (always with one key: "message")
	// 			of an array with raw encoded Service Bus messages
	val, ok := rsp.Message.Value.(map[string]interface{})
	if !ok {
		return newErrIncorrectType(messageField, map[string]interface{}{}, rsp.Message.Value)
	}

	rawMessages, ok := val[messagesField]
	if !ok {
		return ErrMissingField(messagesField)
	}

	messages, ok := rawMessages.([]interface{})
	if !ok {
		return newErrIncorrectType(messagesField, []interface{}{}, rawMessages)
	}

	transformedMessages := make([]*Message, len(messages))
	for i := range messages {
		rawEntry, ok := messages[i].(map[string]interface{})
		if !ok {
			return newErrIncorrectType(messageField, map[string]interface{}{}, messages[i])
		}

		rawMessage, ok := rawEntry[messageField]
		if !ok {
			return ErrMissingField(messageField)
		}

		marshaled, ok := rawMessage.([]byte)
		if !ok {
			return new(ErrMalformedMessage)
		}

		var rehydrated amqp.Message
		err = rehydrated.UnmarshalBinary(marshaled)
		if err != nil {
			return err
		}

		transformedMessages[i], err = messageFromAMQPMessage(&rehydrated)
		if err != nil {
			return err
		}
	}

	// This sort is done to ensure that folks wanting to peek messages in sequence order may do so.
	sort.Slice(transformedMessages, func(i, j int) bool {
		iSeq := *transformedMessages[i].SystemProperties.SequenceNumber
		jSeq := *transformedMessages[j].SystemProperties.SequenceNumber
		return iSeq < jSeq
	})

	for _, msg := range transformedMessages {
		msg.ec = ec
		err := handler.Handle(ctx, msg)
		if err != nil {
			return err
		}
	}

	return nil
}

// Close the underlying connection to Service Bus
func (q *Queue) Close(ctx context.Context) error {
	span, ctx := q.startSpanFromContext(ctx, "sb.Queue.Close")
	defer span.Finish()

	if q.receiver != nil {
		if err := q.receiver.Close(ctx); err != nil {
			if q.sender != nil {
				if err := q.sender.Close(ctx); err != nil && !isConnectionClosed(err) {
					log.For(ctx).Error(err)
				}
			}

			if !isConnectionClosed(err) {
				log.For(ctx).Error(err)
				return err
			}

			return nil
		}
	}

	if q.sender != nil {
		err := q.sender.Close(ctx)
		if err != nil && !isConnectionClosed(err) {
			return err
		}
	}

	return nil
}

func isConnectionClosed(err error) bool {
	return err.Error() == "amqp: connection closed"
}

func (q *Queue) newReceiver(ctx context.Context, opts ...ReceiverOption) (*Receiver, error) {
	span, ctx := q.startSpanFromContext(ctx, "sb.Queue.NewReceiver")
	defer span.Finish()

	opts = append(opts, ReceiverWithReceiveMode(q.receiveMode))

	if q.prefetchCount != nil {
		opts = append(opts, ReceiverWithPrefetchCount(*q.prefetchCount))
	}

	return q.namespace.NewReceiver(ctx, q.Name, opts...)
}

func (q *Queue) ensureReceiver(ctx context.Context, opts ...ReceiverOption) error {
	span, ctx := q.startSpanFromContext(ctx, "sb.Queue.ensureReceiver")
	defer span.Finish()

	q.receiverMu.Lock()
	defer q.receiverMu.Unlock()

	if q.receiver != nil {
		return nil
	}

	receiver, err := q.newReceiver(ctx, opts...)
	if err != nil {
		log.For(ctx).Error(err)
		return err
	}

	q.receiver = receiver
	return nil
}

func (q *Queue) ensureSender(ctx context.Context) error {
	span, ctx := q.startSpanFromContext(ctx, "sb.Queue.ensureSender")
	defer span.Finish()

	q.senderMu.Lock()
	defer q.senderMu.Unlock()

	if q.sender != nil {
		return nil
	}

	s, err := q.NewSender(ctx)
	if err != nil {
		log.For(ctx).Error(err)
		return err
	}
	q.sender = s
	return nil
}

// ManagementPath is the relative uri to address the entity's management functionality
func (q *Queue) ManagementPath() string {
	return fmt.Sprintf("%s/$management", q.Name)
}

func (q *Queue) connection(ctx context.Context) (*amqp.Client, error) {
	span, ctx := q.startSpanFromContext(ctx, "sb.Queue.connection")
	defer span.Finish()

	if err := q.ensureReceiver(ctx); err != nil {
		return nil, err
	}
	return q.receiver.connection, nil
}

func (e *entity) lockMutex() *sync.Mutex {
	return &e.renewMessageLockMutex
}
