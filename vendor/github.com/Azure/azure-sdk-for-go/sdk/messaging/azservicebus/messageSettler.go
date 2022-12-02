// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azservicebus

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/utils"
)

type settler interface {
	CompleteMessage(ctx context.Context, message *ReceivedMessage, options *CompleteMessageOptions) error
	AbandonMessage(ctx context.Context, message *ReceivedMessage, options *AbandonMessageOptions) error
	DeferMessage(ctx context.Context, message *ReceivedMessage, options *DeferMessageOptions) error
	DeadLetterMessage(ctx context.Context, message *ReceivedMessage, options *DeadLetterOptions) error
}

type messageSettler struct {
	links        internal.AMQPLinks
	retryOptions RetryOptions

	// used only for tests
	onlyDoBackupSettlement bool
}

func newMessageSettler(links internal.AMQPLinks, retryOptions RetryOptions) settler {
	return &messageSettler{
		links:        links,
		retryOptions: retryOptions,
	}
}

func (s *messageSettler) useManagementLink(m *ReceivedMessage, receiver internal.AMQPReceiver) bool {
	return s.onlyDoBackupSettlement ||
		m.deferred ||
		m.RawAMQPMessage.linkName != receiver.LinkName()
}

func (s *messageSettler) settleWithRetries(ctx context.Context, message *ReceivedMessage, settleFn func(receiver internal.AMQPReceiver, rpcLink internal.RPCLink) error) error {
	if s == nil {
		return internal.NewErrNonRetriable("messages that are received in `ReceiveModeReceiveAndDelete` mode are not settleable")
	}

	err := s.links.Retry(ctx, EventReceiver, "settle", func(ctx context.Context, lwid *internal.LinksWithID, args *utils.RetryFnArgs) error {
		if err := settleFn(lwid.Receiver, lwid.RPC); err != nil {
			return err
		}

		return nil
	}, RetryOptions{})

	return internal.TransformError(err)
}

// CompleteMessageOptions contains optional parameters for the CompleteMessage function.
type CompleteMessageOptions struct {
	// For future expansion
}

// CompleteMessage completes a message, deleting it from the queue or subscription.
func (s *messageSettler) CompleteMessage(ctx context.Context, message *ReceivedMessage, options *CompleteMessageOptions) error {
	return s.settleWithRetries(ctx, message, func(receiver internal.AMQPReceiver, rpcLink internal.RPCLink) error {
		if s.useManagementLink(message, receiver) {
			return internal.SendDisposition(ctx, rpcLink, receiver.LinkName(), bytesToAMQPUUID(message.LockToken), internal.Disposition{Status: internal.CompletedDisposition}, nil)
		} else {
			return receiver.AcceptMessage(ctx, message.RawAMQPMessage.inner)
		}
	})
}

// AbandonMessageOptions contains optional parameters for Client.AbandonMessage
type AbandonMessageOptions struct {
	// PropertiesToModify specifies properties to modify in the message when it is abandoned.
	PropertiesToModify map[string]interface{}
}

// AbandonMessage will cause a message to be returned to the queue or subscription.
// This will increment its delivery count, and potentially cause it to be dead lettered
// depending on your queue or subscription's configuration.
func (s *messageSettler) AbandonMessage(ctx context.Context, message *ReceivedMessage, options *AbandonMessageOptions) error {
	return s.settleWithRetries(ctx, message, func(receiver internal.AMQPReceiver, rpcLink internal.RPCLink) error {
		if s.useManagementLink(message, receiver) {
			d := internal.Disposition{
				Status: internal.AbandonedDisposition,
			}

			var propertiesToModify map[string]interface{}

			if options != nil && options.PropertiesToModify != nil {
				propertiesToModify = options.PropertiesToModify
			}

			return internal.SendDisposition(ctx, rpcLink, receiver.LinkName(), bytesToAMQPUUID(message.LockToken), d, propertiesToModify)
		}

		var annotations amqp.Annotations

		if options != nil {
			annotations = newAnnotations(options.PropertiesToModify)
		}

		return receiver.ModifyMessage(ctx, message.RawAMQPMessage.inner, &amqp.ModifyMessageOptions{
			DeliveryFailed:    false,
			UndeliverableHere: false,
			Annotations:       annotations,
		})
	})
}

// DeferMessageOptions contains optional parameters for Client.DeferMessage
type DeferMessageOptions struct {
	// PropertiesToModify specifies properties to modify in the message when it is deferred
	PropertiesToModify map[string]interface{}
}

// DeferMessage will cause a message to be deferred. Deferred messages
// can be received using `Receiver.ReceiveDeferredMessages`.
func (s *messageSettler) DeferMessage(ctx context.Context, message *ReceivedMessage, options *DeferMessageOptions) error {
	return s.settleWithRetries(ctx, message, func(receiver internal.AMQPReceiver, rpcLink internal.RPCLink) error {
		if s.useManagementLink(message, receiver) {
			d := internal.Disposition{
				Status: internal.DeferredDisposition,
			}

			var propertiesToModify map[string]interface{}

			if options != nil && options.PropertiesToModify != nil {
				propertiesToModify = options.PropertiesToModify
			}

			return internal.SendDisposition(ctx, rpcLink, receiver.LinkName(), bytesToAMQPUUID(message.LockToken), d, propertiesToModify)
		}

		var annotations amqp.Annotations

		if options != nil {
			annotations = newAnnotations(options.PropertiesToModify)
		}

		return receiver.ModifyMessage(ctx, message.RawAMQPMessage.inner,
			&amqp.ModifyMessageOptions{
				DeliveryFailed:    false,
				UndeliverableHere: true,
				Annotations:       annotations,
			})
	})
}

// DeadLetterOptions describe the reason and error description for dead lettering
// a message using the `Receiver.DeadLetterMessage()`
type DeadLetterOptions struct {
	// ErrorDescription that caused the dead lettering of the message.
	ErrorDescription *string

	// Reason for dead lettering the message.
	Reason *string

	// PropertiesToModify specifies properties to modify in the message when it is dead lettered.
	PropertiesToModify map[string]interface{}
}

// DeadLetterMessage settles a message by moving it to the dead letter queue for a
// queue or subscription. To receive these messages create a receiver with `Client.NewReceiver()`
// using the `SubQueue` option.
func (s *messageSettler) DeadLetterMessage(ctx context.Context, message *ReceivedMessage, options *DeadLetterOptions) error {
	return s.settleWithRetries(ctx, message, func(receiver internal.AMQPReceiver, rpcLink internal.RPCLink) error {
		reason := ""
		description := ""

		if options != nil {
			if options.Reason != nil {
				reason = *options.Reason
			}

			if options.ErrorDescription != nil {
				description = *options.ErrorDescription
			}
		}

		if s.useManagementLink(message, receiver) {
			d := internal.Disposition{
				Status:                internal.SuspendedDisposition,
				DeadLetterDescription: &description,
				DeadLetterReason:      &reason,
			}

			var propertiesToModify map[string]interface{}

			if options != nil && options.PropertiesToModify != nil {
				propertiesToModify = options.PropertiesToModify
			}

			return internal.SendDisposition(ctx, rpcLink, receiver.LinkName(), bytesToAMQPUUID(message.LockToken), d, propertiesToModify)
		}

		info := map[string]interface{}{
			"DeadLetterReason":           reason,
			"DeadLetterErrorDescription": description,
		}

		if options != nil && options.PropertiesToModify != nil {
			for key, val := range options.PropertiesToModify {
				info[key] = val
			}
		}

		amqpErr := amqp.Error{
			Condition: "com.microsoft:dead-letter",
			Info:      info,
		}

		return receiver.RejectMessage(ctx, message.RawAMQPMessage.inner, &amqpErr)
	})
}

func bytesToAMQPUUID(bytes [16]byte) *amqp.UUID {
	uuid := amqp.UUID(bytes)
	return &uuid
}

func newAnnotations(propertiesToModify map[string]interface{}) amqp.Annotations {
	var annotations amqp.Annotations

	for k, v := range propertiesToModify {
		if annotations == nil {
			annotations = amqp.Annotations{}
		}

		annotations[k] = v
	}

	return annotations
}
