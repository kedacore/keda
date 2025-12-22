// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azeventhubs

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/internal/uuid"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/amqpwrap"
	"github.com/Azure/go-amqp"
)

// ErrEventDataTooLarge is returned when a message cannot fit into a batch when using the [azeventhubs.EventDataBatch.AddEventData] function.
var ErrEventDataTooLarge = errors.New("the EventData could not be added because it is too large for the batch")

type (
	// EventDataBatch is used to efficiently pack up EventData before sending it to Event Hubs.
	//
	// EventDataBatch's are not meant to be created directly. Use [ProducerClient.NewEventDataBatch],
	// which will create them with the proper size limit for your Event Hub.
	EventDataBatch struct {
		mu sync.RWMutex

		marshaledMessages [][]byte
		batchEnvelope     *amqp.Message

		maxBytes    uint64
		currentSize uint64

		partitionID  *string
		partitionKey *string
	}
)

const (
	batchMessageFormat uint32 = 0x80013700
)

// AddEventDataOptions contains optional parameters for the AddEventData function.
type AddEventDataOptions struct {
	// For future expansion
}

// AddEventData adds an EventData to the batch, failing if the EventData would
// cause the EventDataBatch to be too large to send.
//
// This size limit was set when the EventDataBatch was created, in options to
// [ProducerClient.NewEventDataBatch], or (by default) from Event
// Hubs itself.
//
// Returns ErrMessageTooLarge if the event cannot fit, or a non-nil error for
// other failures.
func (b *EventDataBatch) AddEventData(ed *EventData, options *AddEventDataOptions) error {
	return b.addAMQPMessage(ed.toAMQPMessage())
}

// AddAMQPAnnotatedMessage adds an AMQPAnnotatedMessage to the batch, failing
// if the AMQPAnnotatedMessage would cause the EventDataBatch to be too large to send.
//
// This size limit was set when the EventDataBatch was created, in options to
// [ProducerClient.NewEventDataBatch], or (by default) from Event
// Hubs itself.
//
// Returns ErrMessageTooLarge if the message cannot fit, or a non-nil error for
// other failures.
func (b *EventDataBatch) AddAMQPAnnotatedMessage(annotatedMessage *AMQPAnnotatedMessage, options *AddEventDataOptions) error {
	return b.addAMQPMessage(annotatedMessage.toAMQPMessage())
}

// NumBytes is the number of bytes in the batch.
func (b *EventDataBatch) NumBytes() uint64 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.currentSize
}

// NumEvents returns the number of events in the batch.
func (b *EventDataBatch) NumEvents() int32 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return int32(len(b.marshaledMessages))
}

// toAMQPMessage converts this batch into a sendable *amqp.Message
// NOTE: not idempotent!
func (b *EventDataBatch) toAMQPMessage() (*amqp.Message, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.marshaledMessages) == 0 {
		return nil, internal.NewErrNonRetriable("batch is nil or empty")
	}

	b.batchEnvelope.Data = make([][]byte, len(b.marshaledMessages))
	b.batchEnvelope.Format = batchMessageFormat

	if b.partitionKey != nil {
		if b.batchEnvelope.Annotations == nil {
			b.batchEnvelope.Annotations = make(amqp.Annotations)
		}

		b.batchEnvelope.Annotations[partitionKeyAnnotation] = *b.partitionKey
	}

	copy(b.batchEnvelope.Data, b.marshaledMessages)
	return b.batchEnvelope, nil
}

func (b *EventDataBatch) addAMQPMessage(msg *amqp.Message) error {
	if msg.Properties.MessageID == nil || msg.Properties.MessageID == "" {
		uid, err := uuid.New()
		if err != nil {
			return err
		}
		msg.Properties.MessageID = uid.String()
	}

	if b.partitionKey != nil {
		if msg.Annotations == nil {
			msg.Annotations = make(amqp.Annotations)
		}

		msg.Annotations[partitionKeyAnnotation] = *b.partitionKey
	}

	bin, err := msg.MarshalBinary()
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.marshaledMessages) == 0 {
		// the first message is special - we use its properties and annotations as the
		// actual envelope for the batch message.
		batchEnv, batchEnvLen, err := createBatchEnvelope(msg)

		if err != nil {
			return err
		}

		// (we'll undo this if it turns out the message was too big)
		b.currentSize = uint64(batchEnvLen)
		b.batchEnvelope = batchEnv
	}

	actualPayloadSize := calcActualSizeForPayload(bin)

	if b.currentSize+actualPayloadSize > b.maxBytes {
		if len(b.marshaledMessages) == 0 {
			// reset our our properties, this didn't end up being our first message.
			b.currentSize = 0
			b.batchEnvelope = nil
		}

		return ErrEventDataTooLarge
	}

	b.currentSize += actualPayloadSize
	b.marshaledMessages = append(b.marshaledMessages, bin)

	return nil
}

// createBatchEnvelope makes a copy of the properties of the message, minus any
// payload fields (like Data, Value or Sequence). The data field will be
// filled in with all the messages when the batch is completed.
func createBatchEnvelope(am *amqp.Message) (*amqp.Message, int, error) {
	batchEnvelope := *am

	batchEnvelope.Data = nil
	batchEnvelope.Value = nil
	batchEnvelope.Sequence = nil

	bytes, err := batchEnvelope.MarshalBinary()

	if err != nil {
		return nil, 0, err
	}

	return &batchEnvelope, len(bytes), nil
}

// calcActualSizeForPayload calculates the payload size based
// on overhead from AMQP encoding.
func calcActualSizeForPayload(payload []byte) uint64 {
	const vbin8Overhead = 5
	const vbin32Overhead = 8

	if len(payload) < 256 {
		return uint64(vbin8Overhead + len(payload))
	}

	return uint64(vbin32Overhead + len(payload))
}

func newEventDataBatch(sender amqpwrap.AMQPSenderCloser, options *EventDataBatchOptions) (*EventDataBatch, error) {
	if options == nil {
		options = &EventDataBatchOptions{}
	}

	if options.PartitionID != nil && options.PartitionKey != nil {
		return nil, errors.New("either PartitionID or PartitionKey can be set, but not both")
	}

	var batch EventDataBatch

	if options.PartitionID != nil {
		// they want to send to a particular partition. The batch size should be the same for any
		// link but we might as well use the one they're going to send to.
		pid := *options.PartitionID
		batch.partitionID = &pid
	} else if options.PartitionKey != nil {
		partKey := *options.PartitionKey
		batch.partitionKey = &partKey
	}

	if options.MaxBytes == 0 {
		batch.maxBytes = sender.MaxMessageSize()
		return &batch, nil
	}

	if options.MaxBytes > sender.MaxMessageSize() {
		return nil, internal.NewErrNonRetriable(fmt.Sprintf("maximum message size for batch was set to %d bytes, which is larger than the maximum size allowed by link (%d)", options.MaxBytes, sender.MaxMessageSize()))
	}

	batch.maxBytes = options.MaxBytes
	return &batch, nil
}
