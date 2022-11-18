// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azservicebus

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp"
)

// AMQPAnnotatedMessage represents the AMQP message, as received from Service Bus.
// For details about these properties, refer to the AMQP specification:
//
//	https://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-messaging-v1.0-os.html#section-message-format
//
// Some fields in this struct are typed 'any', which means they will accept AMQP primitives, or in some
// cases slices and maps.
//
// AMQP simple types include:
// - int (any size), uint (any size)
// - float (any size)
// - string
// - bool
// - time.Time
type AMQPAnnotatedMessage struct {
	// ApplicationProperties corresponds to the "application-properties" section of an AMQP message.
	//
	// The values of the map are restricted to AMQP simple types, as listed in the comment for AMQPAnnotatedMessage.
	ApplicationProperties map[string]any

	// Body represents the body of an AMQP message.
	Body AMQPAnnotatedMessageBody

	// DeliveryAnnotations corresponds to the "delivery-annotations" section in an AMQP message.
	//
	// The values of the map are restricted to AMQP simple types, as listed in the comment for AMQPAnnotatedMessage.
	DeliveryAnnotations map[any]any

	// DeliveryTag corresponds to the delivery-tag property of the TRANSFER frame
	// for this message.
	DeliveryTag []byte

	// Footer is the transport footers for this AMQP message.
	//
	// The values of the map are restricted to AMQP simple types, as listed in the comment for AMQPAnnotatedMessage.
	Footer map[any]any

	// Header is the transport headers for this AMQP message.
	Header *AMQPAnnotatedMessageHeader

	// MessageAnnotations corresponds to the message-annotations section of an AMQP message.
	//
	// The values of the map are restricted to AMQP simple types, as listed in the comment for AMQPAnnotatedMessage.
	MessageAnnotations map[any]any

	// Properties corresponds to the properties section of an AMQP message.
	Properties *AMQPAnnotatedMessageProperties

	linkName string

	// inner is the AMQP message we originally received, which contains some hidden
	// data that's needed to settle with go-amqp. We strip out most of the underlying
	// data so it's fairly minimal.
	inner *amqp.Message
}

// AMQPAnnotatedMessageProperties represents the properties of an AMQP message.
// See here for more details:
// http://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-messaging-v1.0-os.html#type-properties
type AMQPAnnotatedMessageProperties struct {
	// AbsoluteExpiryTime corresponds to the 'absolute-expiry-time' property.
	AbsoluteExpiryTime *time.Time

	// ContentEncoding corresponds to the 'content-encoding' property.
	ContentEncoding *string

	// ContentType corresponds to the 'content-type' property
	ContentType *string

	// CorrelationID corresponds to the 'correlation-id' property.
	// The type of CorrelationID can be a uint64, UUID, []byte, or a string
	CorrelationID any

	// CreationTime corresponds to the 'creation-time' property.
	CreationTime *time.Time

	// GroupID corresponds to the 'group-id' property.
	GroupID *string

	// GroupSequence corresponds to the 'group-sequence' property.
	GroupSequence *uint32

	// MessageID corresponds to the 'message-id' property.
	// The type of MessageID can be a uint64, UUID, []byte, or string
	MessageID any

	// ReplyTo corresponds to the 'reply-to' property.
	ReplyTo *string

	// ReplyToGroupID corresponds to the 'reply-to-group-id' property.
	ReplyToGroupID *string

	// Subject corresponds to the 'subject' property.
	Subject *string

	// To corresponds to the 'to' property.
	To *string

	// UserID corresponds to the 'user-id' property.
	UserID []byte
}

// AMQPAnnotatedMessageBody represents the body of an AMQP message.
// Only one of these fields can be used a a time. They are mutually exclusive.
type AMQPAnnotatedMessageBody struct {
	// Data is encoded/decoded as multiple data sections in the body.
	Data [][]byte

	// Sequence is encoded/decoded as one or more amqp-sequence sections in the body.
	//
	// The values of the slices are are restricted to AMQP simple types, as listed in the comment for AMQPAnnotatedMessage.
	Sequence [][]any

	// Value is encoded/decoded as the amqp-value section in the body.
	//
	// The type of Value can be any of the AMQP simple types, as listed in the comment for AMQPAnnotatedMessage,
	// as well as slices or maps of AMQP simple types.
	Value any
}

// AMQPAnnotatedMessageHeader carries standard delivery details about the transfer
// of a message.
// See https://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-messaging-v1.0-os.html#type-header
// for more details.
type AMQPAnnotatedMessageHeader struct {
	// DeliveryCount is the number of unsuccessful previous attempts to deliver this message.
	// It corresponds to the 'delivery-count' property.
	DeliveryCount uint32

	// Durable corresponds to the 'durable' property.
	Durable bool

	// FirstAcquirer corresponds to the 'first-acquirer' property.
	FirstAcquirer bool

	// Priority corresponds to the 'priority' property.
	Priority uint8

	// TTL corresponds to the 'ttl' property.
	TTL time.Duration
}

// toAMQPMessage converts between our (azservicebus) AMQP message
// to the underlying message used by go-amqp.
func (am *AMQPAnnotatedMessage) toAMQPMessage() *amqp.Message {
	var header *amqp.MessageHeader

	if am.Header != nil {
		header = &amqp.MessageHeader{
			DeliveryCount: am.Header.DeliveryCount,
			Durable:       am.Header.Durable,
			FirstAcquirer: am.Header.FirstAcquirer,
			Priority:      am.Header.Priority,
			TTL:           am.Header.TTL,
		}
	}

	var properties *amqp.MessageProperties

	if am.Properties != nil {
		properties = &amqp.MessageProperties{
			AbsoluteExpiryTime: am.Properties.AbsoluteExpiryTime,
			ContentEncoding:    am.Properties.ContentEncoding,
			ContentType:        am.Properties.ContentType,
			CorrelationID:      am.Properties.CorrelationID,
			CreationTime:       am.Properties.CreationTime,
			GroupID:            am.Properties.GroupID,
			GroupSequence:      am.Properties.GroupSequence,
			MessageID:          am.Properties.MessageID,
			ReplyTo:            am.Properties.ReplyTo,
			ReplyToGroupID:     am.Properties.ReplyToGroupID,
			Subject:            am.Properties.Subject,
			To:                 am.Properties.To,
			UserID:             am.Properties.UserID,
		}
	} else {
		properties = &amqp.MessageProperties{}
	}

	var footer amqp.Annotations

	if am.Footer != nil {
		footer = (amqp.Annotations)(am.Footer)
	}

	return &amqp.Message{
		Annotations:           copyAnnotations(am.MessageAnnotations),
		ApplicationProperties: am.ApplicationProperties,
		Data:                  am.Body.Data,
		DeliveryAnnotations:   amqp.Annotations(am.DeliveryAnnotations),
		DeliveryTag:           am.DeliveryTag,
		Footer:                footer,
		Header:                header,
		Properties:            properties,
		Sequence:              am.Body.Sequence,
		Value:                 am.Body.Value,
	}
}

func copyAnnotations(src map[any]any) amqp.Annotations {
	if src == nil {
		return amqp.Annotations{}
	}

	dest := amqp.Annotations{}

	for k, v := range src {
		dest[k] = v
	}

	return dest
}

func newAMQPAnnotatedMessage(goAMQPMessage *amqp.Message) *AMQPAnnotatedMessage {
	var header *AMQPAnnotatedMessageHeader

	if goAMQPMessage.Header != nil {
		header = &AMQPAnnotatedMessageHeader{
			DeliveryCount: goAMQPMessage.Header.DeliveryCount,
			Durable:       goAMQPMessage.Header.Durable,
			FirstAcquirer: goAMQPMessage.Header.FirstAcquirer,
			Priority:      goAMQPMessage.Header.Priority,
			TTL:           goAMQPMessage.Header.TTL,
		}
	}

	var properties *AMQPAnnotatedMessageProperties

	if goAMQPMessage.Properties != nil {
		properties = &AMQPAnnotatedMessageProperties{
			AbsoluteExpiryTime: goAMQPMessage.Properties.AbsoluteExpiryTime,
			ContentEncoding:    goAMQPMessage.Properties.ContentEncoding,
			ContentType:        goAMQPMessage.Properties.ContentType,
			CorrelationID:      goAMQPMessage.Properties.CorrelationID,
			CreationTime:       goAMQPMessage.Properties.CreationTime,
			GroupID:            goAMQPMessage.Properties.GroupID,
			GroupSequence:      goAMQPMessage.Properties.GroupSequence,
			MessageID:          goAMQPMessage.Properties.MessageID,
			ReplyTo:            goAMQPMessage.Properties.ReplyTo,
			ReplyToGroupID:     goAMQPMessage.Properties.ReplyToGroupID,
			Subject:            goAMQPMessage.Properties.Subject,
			To:                 goAMQPMessage.Properties.To,
			UserID:             goAMQPMessage.Properties.UserID,
		}
	}

	var footer map[any]any

	if goAMQPMessage.Footer != nil {
		footer = (map[any]any)(goAMQPMessage.Footer)
	}

	return &AMQPAnnotatedMessage{
		MessageAnnotations:    map[any]any(goAMQPMessage.Annotations),
		ApplicationProperties: goAMQPMessage.ApplicationProperties,
		Body: AMQPAnnotatedMessageBody{
			Data:     goAMQPMessage.Data,
			Sequence: goAMQPMessage.Sequence,
			Value:    goAMQPMessage.Value,
		},
		DeliveryAnnotations: map[any]any(goAMQPMessage.DeliveryAnnotations),
		DeliveryTag:         goAMQPMessage.DeliveryTag,
		Footer:              footer,
		Header:              header,
		linkName:            goAMQPMessage.LinkName(),
		Properties:          properties,
		inner:               goAMQPMessage,
	}
}
