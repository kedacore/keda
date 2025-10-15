// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azservicebus

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/amqpwrap"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/utils"
	"github.com/Azure/go-amqp"
)

type messageSettler struct {
	links        internal.AMQPLinks
	retryOptions RetryOptions

	// these are used for testing so we can tell which paths we tried to settle.
	notifySettleOnLink       func(message *ReceivedMessage)
	notifySettleOnManagement func(message *ReceivedMessage)
}

func newMessageSettler(links internal.AMQPLinks, retryOptions RetryOptions) *messageSettler {
	return &messageSettler{
		links:                    links,
		retryOptions:             retryOptions,
		notifySettleOnLink:       func(message *ReceivedMessage) {},
		notifySettleOnManagement: func(message *ReceivedMessage) {},
	}
}

func (s *messageSettler) settleWithRetries(ctx context.Context, settleFn func(receiver amqpwrap.AMQPReceiver, rpcLink amqpwrap.RPCLink) error) error {
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
func (ms *messageSettler) CompleteMessage(ctx context.Context, message *ReceivedMessage, options *CompleteMessageOptions) error {
	return ms.settleWithRetries(ctx, func(receiver amqpwrap.AMQPReceiver, rpcLink amqpwrap.RPCLink) error {
		var err error

		if shouldSettleOnReceiver(message) {
			ms.notifySettleOnLink(message)
			err = receiver.AcceptMessage(ctx, message.RawAMQPMessage.inner)

			// NOTE: we're intentionally falling through. If we failed to settle
			// we might be able to attempt to settle against the management link.
		}

		if shouldSettleOnMgmtLink(err, message) {
			ms.notifySettleOnManagement(message)
			return internal.SettleOnMgmtLink(ctx, rpcLink, receiver.LinkName(),
				bytesToAMQPUUID(message.LockToken), internal.Disposition{Status: internal.CompletedDisposition}, nil)
		}

		return err
	})
}

// AbandonMessageOptions contains optional parameters for Client.AbandonMessage
type AbandonMessageOptions struct {
	// PropertiesToModify specifies properties to modify in the message when it is abandoned.
	PropertiesToModify map[string]any
}

// AbandonMessage will cause a message to be returned to the queue or subscription.
// This will increment its delivery count, and potentially cause it to be dead lettered
// depending on your queue or subscription's configuration.
func (ms *messageSettler) AbandonMessage(ctx context.Context, message *ReceivedMessage, options *AbandonMessageOptions) error {
	return ms.settleWithRetries(ctx, func(receiver amqpwrap.AMQPReceiver, rpcLink amqpwrap.RPCLink) error {
		var err error

		if shouldSettleOnReceiver(message) {
			ms.notifySettleOnLink(message)

			var annotations amqp.Annotations

			if options != nil {
				annotations = newAnnotations(options.PropertiesToModify)
			}

			err = receiver.ModifyMessage(ctx, message.RawAMQPMessage.inner, &amqp.ModifyMessageOptions{
				DeliveryFailed:    false,
				UndeliverableHere: false,
				Annotations:       annotations,
			})

			// NOTE: we're intentionally falling through. If we failed to settle
			// we might be able to attempt to settle against the management link.
		}

		if shouldSettleOnMgmtLink(err, message) {
			ms.notifySettleOnManagement(message)

			d := internal.Disposition{
				Status: internal.AbandonedDisposition,
			}

			var propertiesToModify map[string]any

			if options != nil && options.PropertiesToModify != nil {
				propertiesToModify = options.PropertiesToModify
			}

			return internal.SettleOnMgmtLink(ctx, rpcLink, receiver.LinkName(), bytesToAMQPUUID(message.LockToken), d, propertiesToModify)
		}

		return err
	})
}

// DeferMessageOptions contains optional parameters for Client.DeferMessage
type DeferMessageOptions struct {
	// PropertiesToModify specifies properties to modify in the message when it is deferred
	PropertiesToModify map[string]any
}

// DeferMessage will cause a message to be deferred. Deferred messages
// can be received using `Receiver.ReceiveDeferredMessages`.
func (ms *messageSettler) DeferMessage(ctx context.Context, message *ReceivedMessage, options *DeferMessageOptions) error {
	return ms.settleWithRetries(ctx, func(receiver amqpwrap.AMQPReceiver, rpcLink amqpwrap.RPCLink) error {
		var err error

		if shouldSettleOnReceiver(message) {
			ms.notifySettleOnLink(message)

			var annotations amqp.Annotations

			if options != nil {
				annotations = newAnnotations(options.PropertiesToModify)
			}

			err = receiver.ModifyMessage(ctx, message.RawAMQPMessage.inner,
				&amqp.ModifyMessageOptions{
					DeliveryFailed:    false,
					UndeliverableHere: true,
					Annotations:       annotations,
				})

			// NOTE: we're intentionally falling through. If we failed to settle
			// we might be able to attempt to settle against the management link.
		}

		if shouldSettleOnMgmtLink(err, message) {
			ms.notifySettleOnManagement(message)

			d := internal.Disposition{
				Status: internal.DeferredDisposition,
			}

			var propertiesToModify map[string]any

			if options != nil && options.PropertiesToModify != nil {
				propertiesToModify = options.PropertiesToModify
			}

			return internal.SettleOnMgmtLink(ctx, rpcLink, receiver.LinkName(), bytesToAMQPUUID(message.LockToken), d, propertiesToModify)
		}

		return err
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
	PropertiesToModify map[string]any
}

// DeadLetterMessage settles a message by moving it to the dead letter queue for a
// queue or subscription. To receive these messages create a receiver with `Client.NewReceiver()`
// using the `SubQueue` option.
func (ms *messageSettler) DeadLetterMessage(ctx context.Context, message *ReceivedMessage, options *DeadLetterOptions) error {
	return ms.settleWithRetries(ctx, func(receiver amqpwrap.AMQPReceiver, rpcLink amqpwrap.RPCLink) error {
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

		var err error

		if shouldSettleOnReceiver(message) {
			ms.notifySettleOnLink(message)

			info := map[string]any{
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

			err = receiver.RejectMessage(ctx, message.RawAMQPMessage.inner, &amqpErr)

			// NOTE: we're intentionally falling through. If we failed to settle
			// we might be able to attempt to settle against the management link.
		}

		if shouldSettleOnMgmtLink(err, message) {
			ms.notifySettleOnManagement(message)

			d := internal.Disposition{
				Status:                internal.SuspendedDisposition,
				DeadLetterDescription: &description,
				DeadLetterReason:      &reason,
			}

			var propertiesToModify map[string]any

			if options != nil && options.PropertiesToModify != nil {
				propertiesToModify = options.PropertiesToModify
			}

			return internal.SettleOnMgmtLink(ctx, rpcLink, receiver.LinkName(), bytesToAMQPUUID(message.LockToken), d, propertiesToModify)
		}

		return err
	})
}

func bytesToAMQPUUID(bytes [16]byte) *amqp.UUID {
	uuid := amqp.UUID(bytes)
	return &uuid
}

func newAnnotations(propertiesToModify map[string]any) amqp.Annotations {
	var annotations amqp.Annotations

	for k, v := range propertiesToModify {
		if annotations == nil {
			annotations = amqp.Annotations{}
		}

		annotations[k] = v
	}

	return annotations
}

// shouldSettleOnReceiver determines if a message can be settled on an AMQP
// link or should only be settled on the management link.
func shouldSettleOnReceiver(message *ReceivedMessage) bool {
	if message.RawAMQPMessage == nil || message.RawAMQPMessage.inner == nil {
		// messages that have been deserialized, or partially copied (for instance, for rehydration of links)
		// won't have a raw message with an .inner field. We don't need that for settlement.
		return false
	}

	// deferred messages always go through the management link
	return !message.settleOnMgmtLink
}

// shouldSettleOnMgmtLink checks if we can fallback to settling on the management
// link (if `err` was a connection/link failure) or if the message always needs
// to be settled on the management link, like with deferred messages.
func shouldSettleOnMgmtLink(settlementErr error, message *ReceivedMessage) bool {
	if message.settleOnMgmtLink {
		// deferred messages always go through the management link
		return true
	}

	if message.RawAMQPMessage == nil || message.RawAMQPMessage.inner == nil {
		// this is a message they constructed _just_ to settle with.
		return true
	}

	if settlementErr == nil {
		// we settled on the original receiver
		return false
	}

	// if we got a connection or link error we can try settling against the mgmt link since
	// our original receiver is gone.
	var linkErr *amqp.LinkError
	return errors.As(settlementErr, &linkErr)
}
