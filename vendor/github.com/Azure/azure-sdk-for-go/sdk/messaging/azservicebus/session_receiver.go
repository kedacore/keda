// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azservicebus

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/amqpwrap"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/utils"
	"github.com/Azure/go-amqp"
)

// SessionReceiver is a Receiver that handles sessions.
type SessionReceiver struct {
	inner             *Receiver
	sessionID         *string
	acceptNextTimeout time.Duration
	lockedUntil       time.Time
}

// SessionReceiverOptions contains options for the `Client.AcceptSessionForQueue/Subscription` or `Client.AcceptNextSessionForQueue/Subscription`
// functions.
type SessionReceiverOptions struct {
	// ReceiveMode controls when a message is deleted from Service Bus.
	//
	// [ReceiveModePeekLock] is the default. The message is locked, preventing multiple
	// receivers from processing the message at once. You control the lock state of the message
	// using one of the message settlement functions like [SessionReceiver.CompleteMessage], which removes
	// it from Service Bus, or [SessionReceiver.AbandonMessage], which makes it available again.
	//
	// [ReceiveModeReceiveAndDelete] causes Service Bus to remove the message as soon
	// as it's received.
	//
	// More information about receive modes:
	// https://docs.microsoft.com/azure/service-bus-messaging/message-transfers-locks-settlement#settling-receive-operations
	ReceiveMode ReceiveMode
}

func toReceiverOptions(sropts *SessionReceiverOptions) *ReceiverOptions {
	if sropts == nil {
		return nil
	}

	return &ReceiverOptions{
		ReceiveMode: sropts.ReceiveMode,
	}
}

type newSessionReceiverArgs struct {
	sessionID         *string
	ns                internal.NamespaceForAMQPLinks
	entity            entity
	cleanupOnClose    func()
	retryOptions      RetryOptions
	acceptNextTimeout time.Duration
}

func newSessionReceiver(ctx context.Context, args newSessionReceiverArgs, options *ReceiverOptions) (*SessionReceiver, error) {
	sessionReceiver := &SessionReceiver{
		sessionID:   args.sessionID,
		lockedUntil: time.Time{},
	}

	r, err := newReceiver(newReceiverArgs{
		ns:                  args.ns,
		entity:              args.entity,
		cleanupOnClose:      args.cleanupOnClose,
		newLinkFn:           sessionReceiver.newLink,
		getRecoveryKindFunc: internal.GetRecoveryKindForSession,
		retryOptions:        args.retryOptions,
	}, options)

	if err != nil {
		return nil, err
	}

	sessionReceiver.acceptNextTimeout = args.acceptNextTimeout
	sessionReceiver.inner = r

	return sessionReceiver, nil
}

func (r *SessionReceiver) newLink(ctx context.Context, session amqpwrap.AMQPSession) (amqpwrap.AMQPSenderCloser, amqpwrap.AMQPReceiverCloser, error) {
	const sessionFilterName = "com.microsoft:session-filter"
	const code = uint64(0x00000137000000C)

	linkOptions := createLinkOptions(r.inner.receiveMode)

	if r.sessionID == nil {
		linkOptions.Filters = append(linkOptions.Filters, amqp.NewLinkFilter(sessionFilterName, code, nil))
	} else {
		linkOptions.Filters = append(linkOptions.Filters, amqp.NewLinkFilter(sessionFilterName, code, r.sessionID))
	}

	if r.acceptNextTimeout > 0 {
		if linkOptions.Properties == nil {
			linkOptions.Properties = map[string]any{}
		}

		// the remote side of this seems _very_ picky that the type not be larger than 32-bits.
		timeoutInMS := uint32(r.acceptNextTimeout / time.Millisecond)
		linkOptions.Properties["com.microsoft:timeout"] = timeoutInMS
	}

	link, err := session.NewReceiver(ctx, r.inner.amqpLinks.EntityPath(), linkOptions)

	if err != nil {
		return nil, nil, err
	}

	// check the session ID that came back - if we asked for a named session ID and didn't get it then
	// we failed to get the lock.
	// if we specified nil then we can _set_ our internally held session ID now that we know the value.
	receivedSessionID := link.LinkSourceFilterValue(sessionFilterName)
	receivedSessionIDStr, ok := receivedSessionID.(string)

	if !ok || (r.sessionID != nil && receivedSessionIDStr != *r.sessionID) {
		return nil, nil, fmt.Errorf("invalid type/value for returned sessionID(type:%T, value:%v)", receivedSessionID, receivedSessionID)
	}

	r.sessionID = &receivedSessionIDStr

	if props := link.Properties(); props != nil {
		if lockTimeTicks, hasLockTime := props["com.microsoft:locked-until-utc"].(int64); hasLockTime {
			r.lockedUntil = ticksToUnixTime(lockTimeTicks)
		}
	}

	return nil, link, nil
}

// ReceiveMessages receives a fixed number of messages, up to maxMessages.
// This function will block until at least one message is received or until the ctx is cancelled.
//
// NOTE: when the Receiver is in [ReceiveModeReceiveAndDelete] mode, you can call [Receiver.ReceiveMessages] even
// after the Receiver has been closed. This allows you to continue reading from the Receiver's internal cache, until
// it is empty. When you've completely read all cached messages, ReceiveMessages returns an [*Error]
// with `.Code` == azservicebus.CodeClosed.
//
// This is NOT necessary when using [ReceiveModePeekLock] (the default).
//
// If the operation fails it can return an [*Error] type if the failure is actionable.
func (r *SessionReceiver) ReceiveMessages(ctx context.Context, maxMessages int, options *ReceiveMessagesOptions) ([]*ReceivedMessage, error) {
	return r.inner.ReceiveMessages(ctx, maxMessages, options)
}

// ReceiveDeferredMessages receives messages that were deferred using `Receiver.DeferMessage`.
// If the operation fails it can return an [*Error] type if the failure is actionable.
func (r *SessionReceiver) ReceiveDeferredMessages(ctx context.Context, sequenceNumbers []int64, options *ReceiveDeferredMessagesOptions) ([]*ReceivedMessage, error) {
	return r.inner.ReceiveDeferredMessages(ctx, sequenceNumbers, options)
}

// PeekMessages will peek messages without locking or deleting messages.
//
// The SessionReceiver stores the last peeked sequence number internally, and will use it as the
// start location for the next PeekMessages() call. You can override this behavior by passing an
// explicit sequence number in [azservicebus.PeekMessagesOptions.FromSequenceNumber].
//
// Messages that are peeked are not locked, so settlement methods like [SessionReceiver.CompleteMessage],
// [SessionReceiver.AbandonMessage], [SessionReceiver.DeferMessage] or [SessionReceiver.DeadLetterMessage] will not work with them.
//
// If the operation fails it can return an [*Error] type if the failure is actionable.
//
// For more information about peeking/message-browsing see https://aka.ms/azsdk/servicebus/message-browsing
func (r *SessionReceiver) PeekMessages(ctx context.Context, maxMessageCount int, options *PeekMessagesOptions) ([]*ReceivedMessage, error) {
	return r.inner.PeekMessages(ctx, maxMessageCount, options)
}

// Close permanently closes the receiver.
func (r *SessionReceiver) Close(ctx context.Context) error {
	return r.inner.Close(ctx)
}

// CompleteMessage completes a message, deleting it from the queue or subscription.
// If the operation fails it can return an [*Error] type if the failure is actionable.
func (r *SessionReceiver) CompleteMessage(ctx context.Context, message *ReceivedMessage, options *CompleteMessageOptions) error {
	return r.inner.CompleteMessage(ctx, message, options)
}

// AbandonMessage will cause a message to be returned to the queue or subscription.
// This will increment its delivery count, and potentially cause it to be dead lettered
// depending on your queue or subscription's configuration.
// If the operation fails it can return an [*Error] type if the failure is actionable.
func (r *SessionReceiver) AbandonMessage(ctx context.Context, message *ReceivedMessage, options *AbandonMessageOptions) error {
	return r.inner.AbandonMessage(ctx, message, options)
}

// DeferMessage will cause a message to be deferred. Deferred messages
// can be received using `Receiver.ReceiveDeferredMessages`.
// If the operation fails it can return an [*Error] type if the failure is actionable.
func (r *SessionReceiver) DeferMessage(ctx context.Context, message *ReceivedMessage, options *DeferMessageOptions) error {
	return r.inner.DeferMessage(ctx, message, options)
}

// DeadLetterMessage settles a message by moving it to the dead letter queue for a
// queue or subscription. To receive these messages create a receiver with `Client.NewReceiverForQueue()`
// or `Client.NewReceiverForSubscription()` using the `ReceiverOptions.SubQueue` option.
// If the operation fails it can return an [*Error] type if the failure is actionable.
func (r *SessionReceiver) DeadLetterMessage(ctx context.Context, message *ReceivedMessage, options *DeadLetterOptions) error {
	return r.inner.DeadLetterMessage(ctx, message, options)
}

// SessionID is the session ID for this SessionReceiver.
func (sr *SessionReceiver) SessionID() string {
	// return the ultimately assigned session ID for this link (anonymous will get it from the
	// link filter options, non-anonymous is set in newSessionReceiver)
	return *sr.sessionID
}

// LockedUntil is the time the lock on this session expires.
// The lock can be renewed using `SessionReceiver.RenewSessionLock`.
func (sr *SessionReceiver) LockedUntil() time.Time {
	return sr.lockedUntil
}

// GetSessionStateOptions contains optional parameters for the GetSessionState function.
type GetSessionStateOptions struct {
	// For future expansion
}

// GetSessionState retrieves state associated with the session.
// If the operation fails it can return an [*Error] type if the failure is actionable.
func (sr *SessionReceiver) GetSessionState(ctx context.Context, options *GetSessionStateOptions) ([]byte, error) {
	var sessionState []byte

	err := sr.inner.amqpLinks.Retry(ctx, EventReceiver, "GetSessionState", func(ctx context.Context, lwv *internal.LinksWithID, args *utils.RetryFnArgs) error {
		s, err := internal.GetSessionState(ctx, lwv.RPC, lwv.Receiver.LinkName(), sr.SessionID())

		if err != nil {
			return err
		}

		sessionState = s
		return nil
	}, sr.inner.retryOptions)

	return sessionState, internal.TransformError(err)
}

// SetSessionStateOptions contains optional parameters for the SetSessionState function.
type SetSessionStateOptions struct {
	// For future expansion
}

// SetSessionState sets the state associated with the session.
// Pass nil for the state parameter to clear the stored session state.
// If the operation fails it can return an [*Error] type if the failure is actionable.
func (sr *SessionReceiver) SetSessionState(ctx context.Context, state []byte, options *SetSessionStateOptions) error {
	err := sr.inner.amqpLinks.Retry(ctx, EventReceiver, "SetSessionState", func(ctx context.Context, lwv *internal.LinksWithID, args *utils.RetryFnArgs) error {
		return internal.SetSessionState(ctx, lwv.RPC, lwv.Receiver.LinkName(), sr.SessionID(), state)
	}, sr.inner.retryOptions)

	return internal.TransformError(err)
}

// RenewSessionLockOptions contains optional parameters for the RenewSessionLock function.
type RenewSessionLockOptions struct {
	// For future expansion
}

// RenewSessionLock renews this session's lock. The new expiration time is available
// using `LockedUntil`.
// If the operation fails it can return an [*Error] type if the failure is actionable.
func (sr *SessionReceiver) RenewSessionLock(ctx context.Context, options *RenewSessionLockOptions) error {
	err := sr.inner.amqpLinks.Retry(ctx, EventReceiver, "RenewSessionLock", func(ctx context.Context, lwv *internal.LinksWithID, args *utils.RetryFnArgs) error {
		newLockedUntil, err := internal.RenewSessionLock(ctx, lwv.RPC, lwv.Receiver.LinkName(), *sr.sessionID)

		if err != nil {
			return err
		}

		sr.lockedUntil = newLockedUntil
		return nil
	}, sr.inner.retryOptions)

	return internal.TransformError(err)
}

// init ensures the link was created, guaranteeing that we get our expected session lock.
func (sr *SessionReceiver) init(ctx context.Context) error {
	// initialize the links
	_, err := sr.inner.amqpLinks.Get(ctx)
	return internal.TransformError(err)
}

// 1970-01-01, represented in "ticks" (100ns per millisecond) (ie: .NET's time unit for DateTimeOffset)
const epochTicks = int64(621355968000000000)

func ticksToUnixTime(ticks int64) time.Time {
	// normalize our time so it starts from the Unix epoch, then convert from ticks
	// to milliseconds.
	millisFromTicks := (ticks - epochTicks) / 10000

	return time.UnixMilli(millisFromTicks).UTC()
}
