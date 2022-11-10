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

	"github.com/Azure/azure-sdk-for-go/sdk/internal/log"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/amqpwrap"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/utils"
)

// ReceiveMode represents the lock style to use for a receiver - either
// `PeekLock` or `ReceiveAndDelete`
type ReceiveMode = exported.ReceiveMode

const (
	// ReceiveModePeekLock will lock messages as they are received and can be settled
	// using the Receiver's (Complete|Abandon|DeadLetter|Defer)Message
	// functions.
	ReceiveModePeekLock ReceiveMode = exported.PeekLock
	// ReceiveModeReceiveAndDelete will delete messages as they are received.
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
	receiveMode ReceiveMode
	entityPath  string

	settler        settler
	retryOptions   RetryOptions
	cleanupOnClose func()

	lastPeekedSequenceNumber int64
	amqpLinks                internal.AMQPLinks

	mu        sync.Mutex
	receiving bool

	defaultDrainTimeout      time.Duration
	defaultTimeAfterFirstMsg time.Duration

	cancelReleaser *atomic.Value
}

// ReceiverOptions contains options for the `Client.NewReceiverForQueue` or `Client.NewReceiverForSubscription`
// functions.
type ReceiverOptions struct {
	// ReceiveMode controls when a message is deleted from Service Bus.
	//
	// ReceiveModePeekLock is the default. The message is locked, preventing multiple
	// receivers from processing the message at once. You control the lock state of the message
	// using one of the message settlement functions like Receiver.CompleteMessage(), which removes
	// it from Service Bus, or Receiver.AbandonMessage(), which makes it available again.
	//
	// ReceiveModeReceiveAndDelete causes Service Bus to remove the message as soon
	// as it's received.
	//
	// More information about receive modes:
	// https://docs.microsoft.com/azure/service-bus-messaging/message-transfers-locks-settlement#settling-receive-operations
	ReceiveMode ReceiveMode

	// SubQueue should be set to connect to the sub queue (ex: dead letter queue)
	// of the queue or subscription.
	SubQueue SubQueue
}

const defaultLinkRxBuffer = 2048

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
	ns                  internal.NamespaceWithNewAMQPLinks
	entity              entity
	cleanupOnClose      func()
	getRecoveryKindFunc func(err error) internal.RecoveryKind
	newLinkFn           func(ctx context.Context, session amqpwrap.AMQPSession) (internal.AMQPSenderCloser, internal.AMQPReceiverCloser, error)
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
		lastPeekedSequenceNumber: 0,
		cleanupOnClose:           args.cleanupOnClose,
		defaultDrainTimeout:      time.Second,
		defaultTimeAfterFirstMsg: 20 * time.Millisecond,
		retryOptions:             args.retryOptions,
		cancelReleaser:           &atomic.Value{},
	}

	receiver.cancelReleaser.Store(emptyCancelFn)

	if err := applyReceiverOptions(receiver, &args.entity, options); err != nil {
		return nil, err
	}

	if receiver.receiveMode == ReceiveModeReceiveAndDelete {
		// TODO: there appears to be a bit more overhead when receiving messages
		// in ReceiveAndDelete. Need to investigate if this is related to our
		// auto-accepting logic in go-amqp.
		receiver.defaultTimeAfterFirstMsg = time.Second
	}

	newLinkFn := receiver.newReceiverLink

	if args.newLinkFn != nil {
		newLinkFn = args.newLinkFn
	}

	receiver.amqpLinks = args.ns.NewAMQPLinks(receiver.entityPath, newLinkFn, args.getRecoveryKindFunc)

	// 'nil' settler handles returning an error message for receiveAndDelete links.
	if receiver.receiveMode == ReceiveModePeekLock {
		receiver.settler = newMessageSettler(receiver.amqpLinks, receiver.retryOptions)
	} else {
		receiver.settler = (*messageSettler)(nil)
	}

	return receiver, nil
}

func (r *Receiver) newReceiverLink(ctx context.Context, session amqpwrap.AMQPSession) (internal.AMQPSenderCloser, internal.AMQPReceiverCloser, error) {
	linkOptions := createLinkOptions(r.receiveMode)
	link, err := session.NewReceiver(ctx, r.entityPath, linkOptions)
	return nil, link, err
}

// ReceiveMessagesOptions are options for the ReceiveMessages function.
type ReceiveMessagesOptions struct {
	// For future expansion
}

// ReceiveMessages receives a fixed number of messages, up to numMessages.
// This function will block until at least one message is received or until the ctx is cancelled.
// If the operation fails it can return an *azservicebus.Error type if the failure is actionable.
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
// If the operation fails it can return an *azservicebus.Error type if the failure is actionable.
func (r *Receiver) ReceiveDeferredMessages(ctx context.Context, sequenceNumbers []int64, options *ReceiveDeferredMessagesOptions) ([]*ReceivedMessage, error) {
	var receivedMessages []*ReceivedMessage

	err := r.amqpLinks.Retry(ctx, EventReceiver, "receiveDeferredMessages", func(ctx context.Context, lwid *internal.LinksWithID, args *utils.RetryFnArgs) error {
		amqpMessages, err := internal.ReceiveDeferred(ctx, lwid.RPC, lwid.Receiver.LinkName(), r.receiveMode, sequenceNumbers)

		if err != nil {
			return err
		}

		for _, amqpMsg := range amqpMessages {
			receivedMsg := newReceivedMessage(amqpMsg)
			receivedMsg.deferred = true

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
// explicit sequence number in PeekMessagesOptions.FromSequenceNumber.
//
// Messages that are peeked do not have lock tokens, so settlement methods
// like CompleteMessage, AbandonMessage, DeferMessage or DeadLetterMessage
// will not work with them.
// If the operation fails it can return an *azservicebus.Error type if the failure is actionable.
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
			receivedMessages[i] = newReceivedMessage(messages[i])
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
// If the operation fails it can return an *azservicebus.Error type if the failure is actionable.
func (r *Receiver) RenewMessageLock(ctx context.Context, msg *ReceivedMessage, options *RenewMessageLockOptions) error {
	err := r.amqpLinks.Retry(ctx, EventReceiver, "renewMessageLock", func(ctx context.Context, linksWithVersion *internal.LinksWithID, args *utils.RetryFnArgs) error {
		newExpirationTime, err := internal.RenewLocks(ctx, linksWithVersion.RPC, msg.RawAMQPMessage.linkName, []amqp.UUID{
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
	log.Writef(EventReceiver, "Stopped message releaser with ID '%s'", releaserID)

	r.cleanupOnClose()
	return r.amqpLinks.Close(ctx, true)
}

// CompleteMessage completes a message, deleting it from the queue or subscription.
// This function can only be used when the Receiver has been opened with ReceiveModePeekLock.
// If the operation fails it can return an *azservicebus.Error type if the failure is actionable.
func (r *Receiver) CompleteMessage(ctx context.Context, message *ReceivedMessage, options *CompleteMessageOptions) error {
	return r.settler.CompleteMessage(ctx, message, options)
}

// AbandonMessage will cause a message to be  available again from the queue or subscription.
// This will increment its delivery count, and potentially cause it to be dead-lettered
// depending on your queue or subscription's configuration.
// This function can only be used when the Receiver has been opened with `ReceiveModePeekLock`.
// If the operation fails it can return an *azservicebus.Error type if the failure is actionable.
func (r *Receiver) AbandonMessage(ctx context.Context, message *ReceivedMessage, options *AbandonMessageOptions) error {
	return r.settler.AbandonMessage(ctx, message, options)
}

// DeferMessage will cause a message to be deferred. Deferred messages can be received using
// `Receiver.ReceiveDeferredMessages`.
// This function can only be used when the Receiver has been opened with `ReceiveModePeekLock`.
// If the operation fails it can return an *azservicebus.Error type if the failure is actionable.
func (r *Receiver) DeferMessage(ctx context.Context, message *ReceivedMessage, options *DeferMessageOptions) error {
	return r.settler.DeferMessage(ctx, message, options)
}

// DeadLetterMessage settles a message by moving it to the dead letter queue for a
// queue or subscription. To receive these messages create a receiver with `Client.NewReceiverForQueue()`
// or `Client.NewReceiverForSubscription()` using the `ReceiverOptions.SubQueue` option.
// This function can only be used when the Receiver has been opened with `ReceiveModePeekLock`.
// If the operation fails it can return an *azservicebus.Error type if the failure is actionable.
func (r *Receiver) DeadLetterMessage(ctx context.Context, message *ReceivedMessage, options *DeadLetterOptions) error {
	return r.settler.DeadLetterMessage(ctx, message, options)
}

func (r *Receiver) receiveMessagesImpl(ctx context.Context, maxMessages int, options *ReceiveMessagesOptions) ([]*ReceivedMessage, error) {
	cancelReleaser := r.cancelReleaser.Swap(emptyCancelFn).(func() string)
	_ = cancelReleaser()

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
	log.Writef(EventReceiver, "Asking for %d credits", maxMessages)

	if creditsToIssue > 0 {
		log.Writef(EventReceiver, "Only need to issue %d additional credits", creditsToIssue)

		if err := linksWithID.Receiver.IssueCredit(uint32(creditsToIssue)); err != nil {
			return nil, err
		}
	} else {
		log.Writef(EventReceiver, "No additional credits needed, still have %d credits active", currentReceiverCredits)
	}

	result := r.fetchMessages(ctx, linksWithID.Receiver, maxMessages, r.defaultTimeAfterFirstMsg)

	log.Writef(EventReceiver, "Received %d/%d messages", len(result.Messages), maxMessages)

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
		log.Writef(EventReceiver, "Failure when receiving messages: %s", result.Error)
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
		receivedMessages = append(receivedMessages, newReceivedMessage(msg))
	}

	return receivedMessages, nil
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
	receiveMode := amqp.ModeSecond

	if mode == ReceiveModeReceiveAndDelete {
		receiveMode = amqp.ModeFirst
	}

	receiverOpts := &amqp.ReceiverOptions{
		SettlementMode: receiveMode.Ptr(),
		ManualCredits:  true,
		Credit:         defaultLinkRxBuffer,
	}

	if mode == ReceiveModeReceiveAndDelete {
		receiverOpts.RequestedSenderSettleMode = amqp.ModeSettled.Ptr()
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
	firstMsg, err := receiver.Receive(parentCtx)

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
		msg, err := receiver.Receive(ctx)

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
	done := make(chan struct{})
	released := 0

	// this func gets called when a new ReceiveMessages() starts
	r.cancelReleaser.Store(func() string {
		cancel()
		<-done
		return receiver.LinkName()
	})

	return func() {
		defer close(done)

		log.Writef(EventReceiver, "[%s] Message releaser starting...", receiver.LinkName())

		for {
			// we might not have all the messages we need here.
			msg, err := receiver.Receive(ctx)

			if err == nil {
				err = receiver.ReleaseMessage(ctx, msg)
			}

			if err == nil {
				released++
			}

			if internal.IsCancelError(err) {
				log.Writef(exported.EventReceiver, "[%s] Message releaser pausing. Released %d messages", receiver.LinkName(), released)
				break
			} else if internal.GetRecoveryKind(err) != internal.RecoveryKindNone {
				log.Writef(exported.EventReceiver, "[%s] Message releaser stopping because of link failure. Released %d messages. Will start again after next receive: %s", receiver.LinkName(), released, err)
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
