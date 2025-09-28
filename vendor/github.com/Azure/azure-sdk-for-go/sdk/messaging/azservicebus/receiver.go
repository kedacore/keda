// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azservicebus

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/amqpwrap"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/utils"
	"github.com/Azure/go-amqp"
)

// ReceiveMode represents the lock style to use for a receiver - either
// [ReceiveModePeekLock] or [ReceiveModeReceiveAndDelete]
type ReceiveMode = exported.ReceiveMode

const (
	// ReceiveModePeekLock will lock messages as they are received and can be settled using one of
	// the following methods:
	// - [Receiver.CompleteMessage]
	// - [Receiver.AbandonMessage]
	// - [Receiver.DeadLetterMessage]
	// - [Receiver.DeferMessage]
	ReceiveModePeekLock ReceiveMode = exported.PeekLock

	// ReceiveModeReceiveAndDelete will delete messages as they are received.
	//
	// NOTE: when the Receiver is in [ReceiveModeReceiveAndDelete] mode, you can call [Receiver.ReceiveMessages] even
	// after the Receiver has been closed. This allows you to continue reading from the Receiver's internal cache, until
	// it is empty. When you've completely read all cached messages, ReceiveMessages returns an [*Error]
	// with `.Code` == azservicebus.CodeClosed.
	ReceiveModeReceiveAndDelete ReceiveMode = exported.ReceiveAndDelete
)

// SubQueue allows you to target a subqueue of a queue or subscription.
// Ex: the dead letter queue (SubQueueDeadLetter).
type SubQueue int

const (
	// SubQueueDeadLetter targets the dead letter queue for a queue or subscription.
	SubQueueDeadLetter SubQueue = 1
	// SubQueueTransfer targets the transfer dead letter queue for a queue or subscription.
	SubQueueTransfer SubQueue = 2
)

// Receiver receives messages using pull based functions (ReceiveMessages).
type Receiver struct {
	amqpLinks                internal.AMQPLinks
	cancelReleaser           *atomic.Value
	cleanupOnClose           func()
	entityPath               string
	lastPeekedSequenceNumber int64
	maxAllowedCredits        uint32
	mu                       sync.Mutex
	receiveMode              ReceiveMode
	receiving                bool
	retryOptions             RetryOptions
	settler                  *messageSettler
	defaultReleaserTimeout   time.Duration // defaults to 1min, settable for unit tests.

	cachedMessages atomic.Pointer[[]*amqp.Message] // messages that were extracted from a ReceiveAndDelete receiver, after it was closed.
}

// ReceiverOptions contains options for the `Client.NewReceiverForQueue` or `Client.NewReceiverForSubscription`
// functions.
type ReceiverOptions struct {
	// ReceiveMode controls when a message is deleted from Service Bus.
	//
	// [ReceiveModePeekLock] is the default. The message is locked, preventing multiple
	// receivers from processing the message at once. You control the lock state of the message
	// using one of the message settlement functions like [Receiver.CompleteMessage], which removes
	// it from Service Bus, or [Receiver.AbandonMessage], which makes it available again.
	//
	// [ReceiveModeReceiveAndDelete] causes Service Bus to remove the message as soon
	// as it's received.
	//
	// More information about receive modes:
	// https://docs.microsoft.com/azure/service-bus-messaging/message-transfers-locks-settlement#settling-receive-operations
	ReceiveMode ReceiveMode

	// SubQueue should be set to connect to the sub queue (ex: dead letter queue)
	// of the queue or subscription.
	SubQueue SubQueue
}

// defaultLinkRxBuffer is the maximum number of transfer frames we can handle
// on the Receiver. This matches the current default window size that go-amqp
// uses for sessions.
const defaultLinkRxBuffer uint32 = 5000

func applyReceiverOptions(receiver *Receiver, entity *entity, options *ReceiverOptions) error {
	if options == nil {
		receiver.receiveMode = ReceiveModePeekLock
	} else {
		if err := checkReceiverMode(options.ReceiveMode); err != nil {
			return err
		}

		receiver.receiveMode = options.ReceiveMode

		if err := entity.SetSubQueue(options.SubQueue); err != nil {
			return err
		}
	}

	entityPath, err := entity.String()

	if err != nil {
		return err
	}

	receiver.entityPath = entityPath
	return nil
}

type newReceiverArgs struct {
	ns                  internal.NamespaceForAMQPLinks
	entity              entity
	cleanupOnClose      func()
	getRecoveryKindFunc func(err error) internal.RecoveryKind
	newLinkFn           func(ctx context.Context, session amqpwrap.AMQPSession) (amqpwrap.AMQPSenderCloser, amqpwrap.AMQPReceiverCloser, error)
	retryOptions        RetryOptions
}

var emptyCancelFn = func() string {
	return "empty"
}

func newReceiver(args newReceiverArgs, options *ReceiverOptions) (*Receiver, error) {
	if err := args.ns.Check(); err != nil {
		return nil, err
	}

	receiver := &Receiver{
		cancelReleaser:           &atomic.Value{},
		cleanupOnClose:           args.cleanupOnClose,
		lastPeekedSequenceNumber: 0,
		maxAllowedCredits:        defaultLinkRxBuffer,
		retryOptions:             args.retryOptions,
		defaultReleaserTimeout:   time.Minute,
	}

	receiver.cancelReleaser.Store(emptyCancelFn)

	if err := applyReceiverOptions(receiver, &args.entity, options); err != nil {
		return nil, err
	}

	newLinkFn := receiver.newReceiverLink

	if args.newLinkFn != nil {
		newLinkFn = args.newLinkFn
	}

	amqpLinksArgs := internal.NewAMQPLinksArgs{
		NS:                  args.ns,
		EntityPath:          receiver.entityPath,
		CreateLinkFunc:      newLinkFn,
		GetRecoveryKindFunc: args.getRecoveryKindFunc,
	}

	if receiver.receiveMode == ReceiveModeReceiveAndDelete {
		amqpLinksArgs.PrefetchedMessagesAfterClose = func(messages []*amqp.Message) {
			receiver.cachedMessages.Store(&messages)
		}
	}

	receiver.amqpLinks = internal.NewAMQPLinks(amqpLinksArgs)

	// 'nil' settler handles returning an error message for receiveAndDelete links.
	if receiver.receiveMode == ReceiveModePeekLock {
		receiver.settler = newMessageSettler(receiver.amqpLinks, receiver.retryOptions)
	} else {
		receiver.settler = (*messageSettler)(nil)
	}

	return receiver, nil
}

func (r *Receiver) newReceiverLink(ctx context.Context, session amqpwrap.AMQPSession) (amqpwrap.AMQPSenderCloser, amqpwrap.AMQPReceiverCloser, error) {
	linkOptions := createLinkOptions(r.receiveMode)
	link, err := session.NewReceiver(ctx, r.entityPath, linkOptions)
	return nil, link, err
}

// ReceiveMessagesOptions are options for the ReceiveMessages function.
type ReceiveMessagesOptions struct {
	// TimeAfterFirstMessage controls how long, after a message has been received, before we return the
	// accumulated batch of messages.
	//
	// Default value depends on the receive mode:
	// - 20ms when the receiver is in ReceiveModePeekLock
	// - 1s when the receiver is in ReceiveModeReceiveAndDelete
	TimeAfterFirstMessage time.Duration
}

// ReceiveMessages receives a fixed number of messages, up to numMessages.
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
func (r *Receiver) ReceiveMessages(ctx context.Context, maxMessages int, options *ReceiveMessagesOptions) ([]*ReceivedMessage, error) {
	r.mu.Lock()
	isReceiving := r.receiving

	if !isReceiving {
		r.receiving = true

		defer func() {
			r.mu.Lock()
			r.receiving = false
			r.mu.Unlock()
		}()
	}
	r.mu.Unlock()

	if isReceiving {
		return nil, errors.New("receiver is already receiving messages. ReceiveMessages() cannot be called concurrently")
	}

	messages, err := r.receiveMessagesImpl(ctx, maxMessages, options)
	return messages, internal.TransformError(err)
}

// ReceiveDeferredMessagesOptions contains optional parameters for the ReceiveDeferredMessages function.
type ReceiveDeferredMessagesOptions struct {
	// For future expansion
}

// ReceiveDeferredMessages receives messages that were deferred using `Receiver.DeferMessage`.
// If the operation fails it can return an [*Error] type if the failure is actionable.
func (r *Receiver) ReceiveDeferredMessages(ctx context.Context, sequenceNumbers []int64, options *ReceiveDeferredMessagesOptions) ([]*ReceivedMessage, error) {
	var receivedMessages []*ReceivedMessage

	err := r.amqpLinks.Retry(ctx, EventReceiver, "receiveDeferredMessages", func(ctx context.Context, lwid *internal.LinksWithID, args *utils.RetryFnArgs) error {
		amqpMessages, err := internal.ReceiveDeferred(ctx, lwid.RPC, lwid.Receiver.LinkName(), r.receiveMode, sequenceNumbers)

		if err != nil {
			return err
		}

		for _, amqpMsg := range amqpMessages {
			receivedMsg := newReceivedMessage(amqpMsg, lwid.Receiver)
			receivedMsg.settleOnMgmtLink = true

			receivedMessages = append(receivedMessages, receivedMsg)
		}

		return nil
	}, r.retryOptions)

	return receivedMessages, internal.TransformError(err)
}

// PeekMessagesOptions contains options for the `Receiver.PeekMessages`
// function.
type PeekMessagesOptions struct {
	// FromSequenceNumber is the sequence number to start with when peeking messages.
	FromSequenceNumber *int64
}

// PeekMessages will peek messages without locking or deleting messages.
//
// The Receiver stores the last peeked sequence number internally, and will use it as the
// start location for the next PeekMessages() call. You can override this behavior by passing an
// explicit sequence number in [azservicebus.PeekMessagesOptions.FromSequenceNumber].
//
// Messages that are peeked are not locked, so settlement methods like [Receiver.CompleteMessage],
// [Receiver.AbandonMessage], [Receiver.DeferMessage] or [Receiver.DeadLetterMessage] will not work with them.
//
// If the operation fails it can return an [*Error] type if the failure is actionable.
//
// For more information about peeking/message-browsing see https://aka.ms/azsdk/servicebus/message-browsing
func (r *Receiver) PeekMessages(ctx context.Context, maxMessageCount int, options *PeekMessagesOptions) ([]*ReceivedMessage, error) {
	var receivedMessages []*ReceivedMessage

	err := r.amqpLinks.Retry(ctx, EventReceiver, "peekMessages", func(ctx context.Context, links *internal.LinksWithID, args *utils.RetryFnArgs) error {
		var sequenceNumber = r.lastPeekedSequenceNumber + 1
		updateInternalSequenceNumber := true

		if options != nil && options.FromSequenceNumber != nil {
			sequenceNumber = *options.FromSequenceNumber
			updateInternalSequenceNumber = false
		}

		messages, err := internal.PeekMessages(ctx, links.RPC, links.Receiver.LinkName(), sequenceNumber, int32(maxMessageCount))

		if err != nil {
			return err
		}

		receivedMessages = make([]*ReceivedMessage, len(messages))

		for i := 0; i < len(messages); i++ {
			receivedMessages[i] = newReceivedMessage(messages[i], links.Receiver)
		}

		if len(receivedMessages) > 0 && updateInternalSequenceNumber {
			// only update this if they're doing the implicit iteration as part of the receiver.
			r.lastPeekedSequenceNumber = *receivedMessages[len(receivedMessages)-1].SequenceNumber
		}

		return nil
	}, r.retryOptions)

	return receivedMessages, internal.TransformError(err)
}

// RenewMessageLockOptions contains optional parameters for the RenewMessageLock function.
type RenewMessageLockOptions struct {
	// For future expansion
}

// RenewMessageLock renews the lock on a message, updating the `LockedUntil` field on `msg`.
// If the operation fails it can return an [*Error] type if the failure is actionable.
func (r *Receiver) RenewMessageLock(ctx context.Context, msg *ReceivedMessage, options *RenewMessageLockOptions) error {
	err := r.amqpLinks.Retry(ctx, EventReceiver, "renewMessageLock", func(ctx context.Context, linksWithVersion *internal.LinksWithID, args *utils.RetryFnArgs) error {
		newExpirationTime, err := internal.RenewLocks(ctx, linksWithVersion.RPC, msg.linkName, []amqp.UUID{
			(amqp.UUID)(msg.LockToken),
		})

		if err != nil {
			return err
		}

		msg.LockedUntil = &newExpirationTime[0]
		return nil
	}, r.retryOptions)

	return internal.TransformError(err)
}

// Close permanently closes the receiver.
func (r *Receiver) Close(ctx context.Context) error {
	cancelReleaser := r.cancelReleaser.Swap(emptyCancelFn).(func() string)
	releaserID := cancelReleaser()
	r.amqpLinks.Writef(EventReceiver, "Stopped message releaser with ID '%s'", releaserID)

	r.cleanupOnClose()

	return r.amqpLinks.Close(ctx, true)
}

// CompleteMessage completes a message, deleting it from the queue or subscription.
// This function can only be used when the Receiver has been opened with ReceiveModePeekLock.
// If the operation fails it can return an [*Error] type if the failure is actionable.
func (r *Receiver) CompleteMessage(ctx context.Context, message *ReceivedMessage, options *CompleteMessageOptions) error {
	return r.settler.CompleteMessage(ctx, message, options)
}

// AbandonMessage will cause a message to be available again from the queue or subscription.
// This will increment its delivery count, and potentially cause it to be dead-lettered
// depending on your queue or subscription's configuration.
// This function can only be used when the Receiver has been opened with `ReceiveModePeekLock`.
// If the operation fails it can return an [*Error] type if the failure is actionable.
func (r *Receiver) AbandonMessage(ctx context.Context, message *ReceivedMessage, options *AbandonMessageOptions) error {
	return r.settler.AbandonMessage(ctx, message, options)
}

// DeferMessage will cause a message to be deferred. Deferred messages can be received using
// `Receiver.ReceiveDeferredMessages`.
// This function can only be used when the Receiver has been opened with `ReceiveModePeekLock`.
// If the operation fails it can return an [*Error] type if the failure is actionable.
func (r *Receiver) DeferMessage(ctx context.Context, message *ReceivedMessage, options *DeferMessageOptions) error {
	return r.settler.DeferMessage(ctx, message, options)
}

// DeadLetterMessage settles a message by moving it to the dead letter queue for a
// queue or subscription. To receive these messages create a receiver with `Client.NewReceiverForQueue()`
// or `Client.NewReceiverForSubscription()` using the `ReceiverOptions.SubQueue` option.
// This function can only be used when the Receiver has been opened with `ReceiveModePeekLock`.
// If the operation fails it can return an [*Error] type if the failure is actionable.
func (r *Receiver) DeadLetterMessage(ctx context.Context, message *ReceivedMessage, options *DeadLetterOptions) error {
	return r.settler.DeadLetterMessage(ctx, message, options)
}

func (r *Receiver) receiveMessagesImpl(ctx context.Context, maxMessages int, options *ReceiveMessagesOptions) ([]*ReceivedMessage, error) {
	cancelReleaser := r.cancelReleaser.Swap(emptyCancelFn).(func() string)
	_ = cancelReleaser()

	if maxMessages <= 0 {
		return nil, internal.NewErrNonRetriable("maxMessages should be greater than 0")
	}

	if maxMessages > int(r.maxAllowedCredits) {
		return nil, internal.NewErrNonRetriable(fmt.Sprintf("maxMessages cannot exceed %d", r.maxAllowedCredits))
	}

	if msgs := r.receiveFromCache(maxMessages); len(msgs) > 0 {
		return msgs, nil
	}

	var linksWithID *internal.LinksWithID

	err := r.amqpLinks.Retry(ctx, EventReceiver, "receiveMessages.getlinks", func(ctx context.Context, lwid *internal.LinksWithID, args *utils.RetryFnArgs) error {
		linksWithID = lwid
		return nil
	}, r.retryOptions)

	if err != nil {
		return nil, err
	}

	// request just the right amount of credits, taking into account the credits
	// that are already active on the link from prior ReceiveMessages() calls that
	// might have exited before all credits were used up.
	currentReceiverCredits := int64(linksWithID.Receiver.Credits())
	creditsToIssue := int64(maxMessages) - currentReceiverCredits

	if creditsToIssue > 0 {
		r.amqpLinks.Writef(EventReceiver, "Issuing %d credits, have %d", creditsToIssue, currentReceiverCredits)

		if err := linksWithID.Receiver.IssueCredit(uint32(creditsToIssue)); err != nil {
			return nil, err
		}
	} else {
		r.amqpLinks.Writef(EventReceiver, "Have %d credits, no new credits needed", currentReceiverCredits)
	}

	timeAfterFirstMessage := 20 * time.Millisecond

	if options != nil && options.TimeAfterFirstMessage > 0 {
		timeAfterFirstMessage = options.TimeAfterFirstMessage
	} else if r.receiveMode == ReceiveModeReceiveAndDelete {
		timeAfterFirstMessage = time.Second
	}

	result := r.fetchMessages(ctx, linksWithID.Receiver, maxMessages, timeAfterFirstMessage)

	r.amqpLinks.Writef(EventReceiver, "Received %d/%d messages", len(result.Messages), maxMessages)

	// this'll only close anything if the error indicates that the link/connection is bad.
	// it's safe to call with cancellation errors.
	rk := r.amqpLinks.CloseIfNeeded(context.Background(), result.Error)

	if rk == internal.RecoveryKindNone {
		// The link is still alive - we'll start the releaser which will releasing any messages
		// that arrive between this call and the next call to ReceiveMessages().
		//
		// This prevents us from holding onto messages for a long period of time in our
		// internal cache where they'll just keep expiring.
		releaserFunc := r.newReleaserFunc(linksWithID.Receiver)
		go releaserFunc()
	} else {
		r.amqpLinks.Writef(EventReceiver, "Failure when receiving messages: %s", result.Error)
	}

	// If the user does get some messages we ignore 'error' and return only the messages.
	//
	// Doing otherwise would break the idiom that people are used to where people expected
	// a non-nil error to mean any other values in the return are nil or not useful (ie,
	// partial success is not idiomatic).
	//
	// This is mostly safe because the next call to ReceiveMessages() (or any other
	// function on Receiver).will have the same issue and will return the relevant error
	// at that time
	if len(result.Messages) == 0 {
		if internal.IsCancelError(result.Error) || rk == internal.RecoveryKindFatal {
			return nil, result.Error
		}

		return nil, nil
	}

	var receivedMessages []*ReceivedMessage

	for _, msg := range result.Messages {
		receivedMessages = append(receivedMessages, newReceivedMessage(msg, linksWithID.Receiver))
	}

	return receivedMessages, nil
}

// receiveFromCache gets any messages that were retrieved after the Receiver was closed
// It returns an empty slice when the cache has been exhausted.
func (r *Receiver) receiveFromCache(maxMessages int) []*ReceivedMessage {
	cachedMessages := r.cachedMessages.Load()
	if cachedMessages == nil || len(*cachedMessages) == 0 {
		return nil
	}

	n := min(len(*cachedMessages), maxMessages)

	receivedMessages := make([]*ReceivedMessage, n)

	for i := range n {
		receivedMessages[i] = newReceivedMessage((*cachedMessages)[i], nil)
	}

	(*cachedMessages) = (*cachedMessages)[n:]
	return receivedMessages
}

type entity struct {
	subqueue     SubQueue
	Queue        string
	Topic        string
	Subscription string
}

func (e *entity) String() (string, error) {
	entityPath := ""

	if e.Queue != "" {
		entityPath = e.Queue
	} else if e.Topic != "" && e.Subscription != "" {
		entityPath = fmt.Sprintf("%s/Subscriptions/%s", e.Topic, e.Subscription)
	} else {
		return "", errors.New("a queue or subscription was not specified")
	}

	if e.subqueue == SubQueueDeadLetter {
		entityPath += "/$DeadLetterQueue"
	} else if e.subqueue == SubQueueTransfer {
		entityPath += "/$Transfer/$DeadLetterQueue"
	}

	return entityPath, nil
}

func (e *entity) SetSubQueue(subQueue SubQueue) error {
	if subQueue == 0 {
		return nil
	} else if subQueue == SubQueueDeadLetter || subQueue == SubQueueTransfer {
		e.subqueue = subQueue
		return nil
	}

	return fmt.Errorf("unknown SubQueue %d", subQueue)
}

func createLinkOptions(mode ReceiveMode) *amqp.ReceiverOptions {
	receiveMode := amqp.ReceiverSettleModeSecond

	if mode == ReceiveModeReceiveAndDelete {
		receiveMode = amqp.ReceiverSettleModeFirst
	}

	receiverOpts := &amqp.ReceiverOptions{
		SettlementMode: receiveMode.Ptr(),
		Credit:         -1,
	}

	if mode == ReceiveModeReceiveAndDelete {
		receiverOpts.RequestedSenderSettleMode = amqp.SenderSettleModeSettled.Ptr()
	}

	return receiverOpts
}

func checkReceiverMode(receiveMode ReceiveMode) error {
	if receiveMode == ReceiveModePeekLock || receiveMode == ReceiveModeReceiveAndDelete {
		return nil
	}

	return fmt.Errorf("invalid receive mode %d, must be either azservicebus.PeekLock or azservicebus.ReceiveAndDelete", receiveMode)
}

// fetchMessagesResult is the result from a fetchMessages
// call.
// NOTE: that you can get both an error and messages!
type fetchMessagesResult struct {
	Messages []*amqp.Message
	Error    error
}

// fetchMessages receives messages, blocking indefinitely until at least one
// message arrives, the parentCtx parameter is cancelled, or the receiver itself
// is disconnected from Service Bus.
//
// Note, if you want to only receive prefetched messages send the parentCtx in
// pre-cancelled. This will cause us to only flush the prefetch buffer.
func (r *Receiver) fetchMessages(parentCtx context.Context, receiver amqpwrap.AMQPReceiver, count int, timeAfterFirstMessage time.Duration) fetchMessagesResult {
	// The first receive is a bit special - we activate a short timer after this
	// so the user doesn't end up in a situation where we're holding onto a bunch
	// of messages but never return because they never cancelled and we never
	// received all 'count' number of messages.
	firstMsg, err := receiver.Receive(parentCtx, nil)

	if err != nil {
		// drain the prefetch buffer - we're stopping because of a
		// failure on the link/connection _or_ the user cancelled the
		// operation.
		return fetchMessagesResult{
			Error: err,
			// Since our link is always active it's possible some
			// messages were sitting in the prefetched buffer from before
			//
			// This particularly affects us in ReceiveAndDelete mode since the
			// local copy of the message can never be retrieved from the server
			// again (they're pre-settled).
			Messages: getAllPrefetched(receiver, count),
		}
	}

	messages := []*amqp.Message{firstMsg}

	// after we get one message we will try to receive as much as we can
	// during the `timeAfterFirstMessage` duration.
	ctx, cancel := context.WithTimeout(parentCtx, timeAfterFirstMessage)
	defer cancel()

	var lastErr error

	for i := 0; i < count-1; i++ {
		msg, err := receiver.Receive(ctx, nil)

		if err != nil {
			lastErr = err
			break
		}

		messages = append(messages, msg)
	}

	// drain the prefetch buffer - we're stopping because of a
	// failure on the link/connection _or_ the user cancelled the
	// operation.
	messages = append(messages, getAllPrefetched(receiver, count-len(messages))...)

	if internal.IsCancelError(lastErr) {
		return fetchMessagesResult{
			Messages: messages,
			// we might have cancelled here (because we stop receiving after `timeAfterFirstMessage` expires)
			// or _they_ cancelled the ReceiveMessages() call.
			//
			// If we cancel: we want a nil error since there's no failure. In that case parentCtx.Err() is nil
			// If they cancel: we want to forward on their cancellation error.
			Error: parentCtx.Err(),
		}
	} else {
		return fetchMessagesResult{
			Error:    lastErr,
			Messages: messages,
		}
	}
}

// newReleaserFunc creates a function that continually receives on a
// receiver and amqpReceiver.Release(msg)'s until cancelled. We use this
// lieu of a 'drain' strategy so we don't hold onto messages in our internal
// cache only for them to expire.
func (r *Receiver) newReleaserFunc(receiver amqpwrap.AMQPReceiver) func() {
	if r.receiveMode == ReceiveModeReceiveAndDelete {
		// you can't disposition messages that are received in this mode - these messages
		// are "presettled" so we do NOT want to discard these messages.
		return func() {}
	}

	ctx, cancel := context.WithCancel(context.Background())
	releaseLoopDone := make(chan struct{})
	released := 0

	// this func gets called when a new ReceiveMessages() starts
	r.cancelReleaser.Store(func() string {
		cancel()
		<-releaseLoopDone
		return receiver.LinkName()
	})

	return func() {
		defer close(releaseLoopDone)

		for {
			msg, err := receiver.Receive(ctx, nil)

			if err == nil {
				releaseCtx, cancelRelease := context.WithTimeout(context.Background(), r.defaultReleaserTimeout)

				// We don't use `ctx` here to avoid cancelling Release(), and leaving this message
				// in limbo until it expires.
				err = receiver.ReleaseMessage(releaseCtx, msg)
				cancelRelease()

				if err == nil {
					released++
				}
			}

			// We check `ctx.Err()` here, instead of testing the returned err from .Receive(), because Receive()
			// ignores cancellation if it has any messages in its prefetch queue.
			if ctx.Err() != nil {
				if released > 0 {
					r.amqpLinks.Writef(exported.EventReceiver, "Message releaser pausing. Released %d messages", released)
				}
				break
			} else if internal.GetRecoveryKind(err) != internal.RecoveryKindNone {
				r.amqpLinks.Writef(exported.EventReceiver, "Message releaser stopping because of link failure. Released %d messages. Will start again after next receive: %s", released, err)
				break
			}
		}
	}
}

func getAllPrefetched(receiver amqpwrap.AMQPReceiver, max int) []*amqp.Message {
	var messages []*amqp.Message

	for i := 0; i < max; i++ {
		msg := receiver.Prefetched()

		if msg == nil {
			break
		}

		messages = append(messages, msg)
	}

	return messages
}
