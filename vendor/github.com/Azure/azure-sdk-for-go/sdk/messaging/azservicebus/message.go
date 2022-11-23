// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azservicebus

import (
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/internal/log"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp"
)

// ReceivedMessage is a received message from a Client.NewReceiver().
type ReceivedMessage struct {
	// ApplicationProperties can be used to store custom metadata for a message.
	ApplicationProperties map[string]interface{}

	// Body is the payload for a message.
	Body []byte

	// ContentType describes the payload of the message, with a descriptor following
	// the format of Content-Type, specified by RFC2045 (ex: "application/json").
	ContentType *string

	// CorrelationID allows an application to specify a context for the message for the purposes of
	// correlation, for example reflecting the MessageID of a message that is being
	// replied to.
	CorrelationID *string

	// DeadLetterErrorDescription is the description set when the message was dead-lettered.
	DeadLetterErrorDescription *string

	// DeadLetterReason is the reason set when the message was dead-lettered.
	DeadLetterReason *string

	// DeadLetterSource is the name of the queue or subscription this message was enqueued on
	// before it was dead-lettered.
	DeadLetterSource *string

	// DeliveryCount is number of times this message has been delivered.
	// This number is incremented when a message lock expires or if the message is explicitly abandoned
	// with Receiver.AbandonMessage.
	DeliveryCount uint32

	// EnqueuedSequenceNumber is the original sequence number assigned to a message, before it
	// was auto-forwarded.
	EnqueuedSequenceNumber *int64

	// EnqueuedTime is the UTC time when the message was accepted and stored by Service Bus.
	EnqueuedTime *time.Time

	// ExpiresAt is the time when this message will expire.
	//
	// This time is calculated by adding the TimeToLive property, set in the message that was sent, along  with the
	// EnqueuedTime of the message.
	ExpiresAt *time.Time

	// LockedUntil is the time when the lock expires for this message.
	// This can be extended by using Receiver.RenewMessageLock.
	LockedUntil *time.Time

	// LockToken is the lock token for a message received from a Receiver created with a receive mode of ReceiveModePeekLock.
	LockToken [16]byte

	// MessageID is an application-defined value that uniquely identifies
	// the message and its payload. The identifier is a free-form string.
	//
	// If enabled, the duplicate detection feature identifies and removes further submissions
	// of messages with the same MessageId.
	MessageID string

	// PartitionKey is used with a partitioned entity and enables assigning related messages
	// to the same internal partition. This ensures that the submission sequence order is correctly
	// recorded. The partition is chosen by a hash function in Service Bus and cannot be chosen
	// directly.
	//
	// For session-aware entities, the ReceivedMessage.SessionID overrides this value.
	PartitionKey *string

	// ReplyTo is an application-defined value specify a reply path to the receiver of the message. When
	// a sender expects a reply, it sets the value to the absolute or relative path of the queue or topic
	// it expects the reply to be sent to.
	ReplyTo *string

	// ReplyToSessionID augments the ReplyTo information and specifies which SessionId should
	// be set for the reply when sent to the reply entity.
	ReplyToSessionID *string

	// ScheduledEnqueueTime specifies a time when a message will be enqueued. The message is transferred
	// to the broker but will not available until the scheduled time.
	ScheduledEnqueueTime *time.Time

	// SequenceNumber is a unique number assigned to a message by Service Bus.
	SequenceNumber *int64

	// SessionID is used with session-aware entities and associates a message with an application-defined
	// session ID. Note that an empty string is a valid session identifier.
	// Messages with the same session identifier are subject to summary locking and enable
	// exact in-order processing and demultiplexing. For session-unaware entities, this value is ignored.
	SessionID *string

	// State represents the current state of the message (Active, Scheduled, Deferred).
	State MessageState

	// Subject enables an application to indicate the purpose of the message, similar to an email subject line.
	Subject *string

	// TimeToLive is the duration after which the message expires, starting from the instant the
	// message has been accepted and stored by the broker, found in the ReceivedMessage.EnqueuedTime
	// property.
	//
	// When not set explicitly, the assumed value is the DefaultTimeToLive for the queue or topic.
	// A message's TimeToLive cannot be longer than the entity's DefaultTimeToLive, and is silently
	// adjusted if it is.
	TimeToLive *time.Duration

	// To is reserved for future use in routing scenarios but is not currently used by Service Bus.
	// Applications can use this value to indicate the logical destination of the message.
	To *string

	// RawAMQPMessage is the AMQP message, as received by the client. This can be useful to get access
	// to properties that are not exposed by ReceivedMessage such as payloads encoded into the
	// Value or Sequence section, payloads sent as multiple Data sections, as well as Footer
	// and Header fields.
	RawAMQPMessage *AMQPAnnotatedMessage

	// deferred indicates we received it using ReceiveDeferredMessages. These messages
	// will still go through the normal Receiver.Settle functions but internally will
	// always be settled with the management link.
	deferred bool
}

// MessageState represents the current state of a message (Active, Scheduled, Deferred).
type MessageState int32

const (
	// MessageStateActive indicates the message is active.
	MessageStateActive MessageState = 0
	// MessageStateDeferred indicates the message is deferred.
	MessageStateDeferred MessageState = 1
	// MessageStateScheduled indicates the message is scheduled.
	MessageStateScheduled MessageState = 2
)

// Message is a message with a body and commonly used properties.
// Properties that are pointers are optional.
type Message struct {
	// ApplicationProperties can be used to store custom metadata for a message.
	ApplicationProperties map[string]interface{}

	// Body corresponds to the first []byte array in the Data section of an AMQP message.
	Body []byte

	// ContentType describes the payload of the message, with a descriptor following
	// the format of Content-Type, specified by RFC2045 (ex: "application/json").
	ContentType *string

	// CorrelationID allows an application to specify a context for the message for the purposes of
	// correlation, for example reflecting the MessageID of a message that is being
	// replied to.
	CorrelationID *string

	// MessageID is an application-defined value that uniquely identifies
	// the message and its payload. The identifier is a free-form string.
	//
	// If enabled, the duplicate detection feature identifies and removes further submissions
	// of messages with the same MessageId.
	MessageID *string

	// PartitionKey is used with a partitioned entity and enables assigning related messages
	// to the same internal partition. This ensures that the submission sequence order is correctly
	// recorded. The partition is chosen by a hash function in Service Bus and cannot be chosen
	// directly.
	//
	// For session-aware entities, the ReceivedMessage.SessionID overrides this value.
	PartitionKey *string

	// ReplyTo is an application-defined value specify a reply path to the receiver of the message. When
	// a sender expects a reply, it sets the value to the absolute or relative path of the queue or topic
	// it expects the reply to be sent to.
	ReplyTo *string

	// ReplyToSessionID augments the ReplyTo information and specifies which SessionId should
	// be set for the reply when sent to the reply entity.
	ReplyToSessionID *string

	// ScheduledEnqueueTime specifies a time when a message will be enqueued. The message is transferred
	// to the broker but will not available until the scheduled time.
	ScheduledEnqueueTime *time.Time

	// SessionID is used with session-aware entities and associates a message with an application-defined
	// session ID. Note that an empty string is a valid session identifier.
	// Messages with the same session identifier are subject to summary locking and enable
	// exact in-order processing and demultiplexing. For session-unaware entities, this value is ignored.
	SessionID *string

	// Subject enables an application to indicate the purpose of the message, similar to an email subject line.
	Subject *string

	// TimeToLive is the duration after which the message expires, starting from the instant the
	// message has been accepted and stored by the broker, found in the ReceivedMessage.EnqueuedTime
	// property.
	//
	// When not set explicitly, the assumed value is the DefaultTimeToLive for the queue or topic.
	// A message's TimeToLive cannot be longer than the entity's DefaultTimeToLive is silently
	// adjusted if it does.
	TimeToLive *time.Duration

	// To is reserved for future use in routing scenarios but is not currently used by Service Bus.
	// Applications can use this value to indicate the logical destination of the message.
	To *string
}

// Service Bus custom properties
const (
	// DeliveryAnnotation properties
	lockTokenDeliveryAnnotation = "x-opt-lock-token"

	// Annotation properties
	partitionKeyAnnotation           = "x-opt-partition-key"
	scheduledEnqueuedTimeAnnotation  = "x-opt-scheduled-enqueue-time"
	lockedUntilAnnotation            = "x-opt-locked-until"
	sequenceNumberAnnotation         = "x-opt-sequence-number"
	enqueuedTimeAnnotation           = "x-opt-enqueued-time"
	deadLetterSourceAnnotation       = "x-opt-deadletter-source"
	enqueuedSequenceNumberAnnotation = "x-opt-enqueue-sequence-number"
	messageStateAnnotation           = "x-opt-message-state"
)

func (m *Message) toAMQPMessage() *amqp.Message {
	amqpMsg := amqp.NewMessage(m.Body)

	if m.TimeToLive != nil {
		if amqpMsg.Header == nil {
			amqpMsg.Header = new(amqp.MessageHeader)
		}
		amqpMsg.Header.TTL = *m.TimeToLive
	}

	var messageID interface{}

	if m.MessageID != nil {
		messageID = *m.MessageID
	}

	amqpMsg.Properties = &amqp.MessageProperties{
		MessageID: messageID,
	}

	if m.SessionID != nil {
		amqpMsg.Properties.GroupID = m.SessionID
	}

	// if m.GroupSequence != nil {
	// 	amqpMsg.Properties.GroupSequence = *m.GroupSequence
	// }

	if m.CorrelationID != nil {
		amqpMsg.Properties.CorrelationID = *m.CorrelationID
	}

	amqpMsg.Properties.ContentType = m.ContentType
	amqpMsg.Properties.Subject = m.Subject
	amqpMsg.Properties.To = m.To
	amqpMsg.Properties.ReplyTo = m.ReplyTo
	amqpMsg.Properties.ReplyToGroupID = m.ReplyToSessionID

	if len(m.ApplicationProperties) > 0 {
		amqpMsg.ApplicationProperties = make(map[string]interface{})
		for key, value := range m.ApplicationProperties {
			amqpMsg.ApplicationProperties[key] = value
		}
	}

	amqpMsg.Annotations = map[interface{}]interface{}{}

	if m.PartitionKey != nil {
		amqpMsg.Annotations[partitionKeyAnnotation] = *m.PartitionKey
	}

	if m.ScheduledEnqueueTime != nil {
		amqpMsg.Annotations[scheduledEnqueuedTimeAnnotation] = *m.ScheduledEnqueueTime
	}

	// TODO: These are 'received' message properties so I believe their inclusion here was just an artifact of only
	// having one message type.

	// if m.SystemProperties != nil {
	// 	// Set the raw annotations first (they may be nil) and add the explicit
	// 	// system properties second to ensure they're set properly.
	// 	amqpMsg.Annotations = addMapToAnnotations(amqpMsg.Annotations, m.SystemProperties.Annotations)

	// 	sysPropMap, err := encodeStructureToMap(m.SystemProperties)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	amqpMsg.Annotations = addMapToAnnotations(amqpMsg.Annotations, sysPropMap)
	// }

	// if m.LockToken != nil {
	// 	if amqpMsg.DeliveryAnnotations == nil {
	// 		amqpMsg.DeliveryAnnotations = make(amqp.Annotations)
	// 	}
	// 	amqpMsg.DeliveryAnnotations[lockTokenName] = *m.LockToken
	// }

	return amqpMsg
}

// newReceivedMessage creates a received message from an AMQP message.
// NOTE: this converter assumes that the Body of this message will be the first
// serialized byte array in the Data section of the messsage.
func newReceivedMessage(amqpMsg *amqp.Message) *ReceivedMessage {
	msg := &ReceivedMessage{
		RawAMQPMessage: newAMQPAnnotatedMessage(amqpMsg),
		State:          MessageStateActive,
	}

	if len(msg.RawAMQPMessage.Body.Data) == 1 {
		msg.Body = msg.RawAMQPMessage.Body.Data[0]
	}

	if amqpMsg.Properties != nil {
		if id, ok := amqpMsg.Properties.MessageID.(string); ok {
			msg.MessageID = id
		}
		msg.SessionID = amqpMsg.Properties.GroupID
		//msg.GroupSequence = &amqpMsg.Properties.GroupSequence

		if id, ok := amqpMsg.Properties.CorrelationID.(string); ok {
			msg.CorrelationID = &id
		}
		msg.ContentType = amqpMsg.Properties.ContentType
		msg.Subject = amqpMsg.Properties.Subject
		msg.To = amqpMsg.Properties.To
		msg.ReplyTo = amqpMsg.Properties.ReplyTo
		msg.ReplyToSessionID = amqpMsg.Properties.ReplyToGroupID
		if amqpMsg.Header != nil {
			msg.DeliveryCount = amqpMsg.Header.DeliveryCount + 1
			msg.TimeToLive = &amqpMsg.Header.TTL
		}
	}

	if amqpMsg.ApplicationProperties != nil {
		msg.ApplicationProperties = make(map[string]interface{}, len(amqpMsg.ApplicationProperties))
		for key, value := range amqpMsg.ApplicationProperties {
			msg.ApplicationProperties[key] = value
		}

		if deadLetterErrorDescription, ok := amqpMsg.ApplicationProperties["DeadLetterErrorDescription"]; ok {
			msg.DeadLetterErrorDescription = to.Ptr(deadLetterErrorDescription.(string))
		}

		if deadLetterReason, ok := amqpMsg.ApplicationProperties["DeadLetterReason"]; ok {
			msg.DeadLetterReason = to.Ptr(deadLetterReason.(string))
		}
	}

	if amqpMsg.Annotations != nil {
		if lockedUntil, ok := amqpMsg.Annotations[lockedUntilAnnotation]; ok {
			t := lockedUntil.(time.Time)
			msg.LockedUntil = &t
		}

		if sequenceNumber, ok := amqpMsg.Annotations[sequenceNumberAnnotation]; ok {
			msg.SequenceNumber = to.Ptr(sequenceNumber.(int64))
		}

		if partitionKey, ok := amqpMsg.Annotations[partitionKeyAnnotation]; ok {
			msg.PartitionKey = to.Ptr(partitionKey.(string))
		}

		if enqueuedTime, ok := amqpMsg.Annotations[enqueuedTimeAnnotation]; ok {
			t := enqueuedTime.(time.Time)
			msg.EnqueuedTime = &t
		}

		if deadLetterSource, ok := amqpMsg.Annotations[deadLetterSourceAnnotation]; ok {
			msg.DeadLetterSource = to.Ptr(deadLetterSource.(string))
		}

		if scheduledEnqueueTime, ok := amqpMsg.Annotations[scheduledEnqueuedTimeAnnotation]; ok {
			t := scheduledEnqueueTime.(time.Time)
			msg.ScheduledEnqueueTime = &t
		}

		if enqueuedSequenceNumber, ok := amqpMsg.Annotations[enqueuedSequenceNumberAnnotation]; ok {
			msg.EnqueuedSequenceNumber = to.Ptr(enqueuedSequenceNumber.(int64))
		}

		switch asInt64(amqpMsg.Annotations[messageStateAnnotation], 0) {
		case 1:
			msg.State = MessageStateDeferred
		case 2:
			msg.State = MessageStateScheduled
		default:
			msg.State = MessageStateActive
		}

		// TODO: annotation propagation is a thing. Currently these are only stored inside
		// of the underlying AMQP message, but not inside of the message itself.

		// If we didn't populate any system properties, set up the struct so we
		// can put the annotations in it
		// if msg.SystemProperties == nil {
		// 	msg.SystemProperties = new(SystemProperties)
		// }

		// Take all string-keyed annotations because the protocol reserves all
		// numeric keys for itself and there are no numeric keys defined in the
		// protocol today:
		//
		//	http://www.amqp.org/sites/amqp.org/files/amqp.pdf (section 3.2.10)
		//
		// This approach is also consistent with the behavior of .NET:
		//
		//	https://docs.microsoft.com/en-us/dotnet/api/azure.messaging.eventhubs.eventdata.systemproperties?view=azure-dotnet#Azure_Messaging_EventHubs_EventData_SystemProperties
		// msg.SystemProperties.Annotations = make(map[string]interface{})
		// for key, val := range amqpMsg.Annotations {
		// 	if s, ok := key.(string); ok {
		// 		msg.SystemProperties.Annotations[s] = val
		// 	}
		// }
	}

	if amqpMsg.DeliveryTag != nil && len(amqpMsg.DeliveryTag) > 0 {
		lockToken, err := lockTokenFromMessageTag(amqpMsg)

		if err == nil {
			msg.LockToken = *(*amqp.UUID)(lockToken)
		} else {
			log.Writef(EventReceiver, "msg.DeliveryTag could not be converted into a UUID: %s", err.Error())
		}
	}

	if token, ok := amqpMsg.DeliveryAnnotations[lockTokenDeliveryAnnotation]; ok {
		if id, ok := token.(amqp.UUID); ok {
			msg.LockToken = [16]byte(id)
		}
	}

	if msg.EnqueuedTime != nil && msg.TimeToLive != nil {
		expiresAt := msg.EnqueuedTime.Add(*msg.TimeToLive)
		msg.ExpiresAt = &expiresAt
	}

	return msg
}

func lockTokenFromMessageTag(msg *amqp.Message) (*amqp.UUID, error) {
	return uuidFromLockTokenBytes(msg.DeliveryTag)
}

func uuidFromLockTokenBytes(bytes []byte) (*amqp.UUID, error) {
	if len(bytes) != 16 {
		return nil, fmt.Errorf("invalid lock token, token was not 16 bytes long")
	}

	var swapIndex = func(indexOne, indexTwo int, array *[16]byte) {
		v1 := array[indexOne]
		array[indexOne] = array[indexTwo]
		array[indexTwo] = v1
	}

	// Get lock token from the deliveryTag
	var lockTokenBytes [16]byte
	copy(lockTokenBytes[:], bytes[:16])
	// translate from .net guid byte serialisation format to amqp rfc standard
	swapIndex(0, 3, &lockTokenBytes)
	swapIndex(1, 2, &lockTokenBytes)
	swapIndex(4, 5, &lockTokenBytes)
	swapIndex(6, 7, &lockTokenBytes)
	amqpUUID := amqp.UUID(lockTokenBytes)

	return &amqpUUID, nil
}

func asInt64(v interface{}, defVal int64) int64 {
	switch v2 := v.(type) {
	case int32:
		return int64(v2)
	case int64:
		return int64(v2)
	case int:
		return int64(v2)
	default:
		return defVal
	}
}
