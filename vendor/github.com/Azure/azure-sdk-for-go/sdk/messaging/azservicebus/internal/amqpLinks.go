// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package internal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/internal/log"
	azlog "github.com/Azure/azure-sdk-for-go/sdk/internal/log"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/amqpwrap"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/utils"
	"github.com/Azure/go-amqp"
)

type LinksWithID struct {
	Sender   amqpwrap.AMQPSender
	Receiver amqpwrap.AMQPReceiver
	RPC      amqpwrap.RPCLink
	ID       LinkID
}

type RetryWithLinksFn func(ctx context.Context, lwid *LinksWithID, args *utils.RetryFnArgs) error

// contextWithTimeoutFn matches the signature for `context.WithTimeout` and is used when we want to
// stub things out for tests.
type contextWithTimeoutFn func(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc)

type AMQPLinks interface {
	EntityPath() string
	ManagementPath() string

	Audience() string

	// Get will initialize a session and call its link.linkCreator function.
	// If this link has been closed via Close() it will return an non retriable error.
	Get(ctx context.Context) (*LinksWithID, error)

	// Retry will run your callback, recovering links when necessary.
	Retry(ctx context.Context, name log.Event, operation string, fn RetryWithLinksFn, o exported.RetryOptions) error

	// RecoverIfNeeded will check if an error requires recovery, and will recover
	// the link or, possibly, the connection.
	RecoverIfNeeded(ctx context.Context, linkID LinkID, err error) error

	// Close will close the the link.
	// If permanent is true the link will not be auto-recreated if Get/Recover
	// are called. All functions will return `ErrLinksClosed`
	Close(ctx context.Context, permanent bool) error

	// CloseIfNeeded closes the links or connection if the error is recoverable.
	// Use this if you don't want to recreate the connection/links at this point.
	CloseIfNeeded(ctx context.Context, err error) RecoveryKind

	// ClosedPermanently is true if AMQPLinks.Close(ctx, true) has been called.
	ClosedPermanently() bool

	// Writef logs a message, with a prefix that represents the AMQPLinks instance
	// for better traceability.
	Writef(evt azlog.Event, format string, args ...any)

	// Prefix is the current logging prefix, usable for logging and continuity.
	Prefix() string
}

// AMQPLinksImpl manages the set of AMQP links (and detritus) typically needed to work
// within Service Bus:
//
// - An *goamqp.Sender or *goamqp.Receiver AMQP link (could also be 'both' if needed)
// - A `$management` link
// - an *goamqp.Session
//
// State management can be done through Recover (close and reopen), Close (close permanently, return failures)
// and Get() (retrieve latest version of all AMQPLinksImpl, or create if needed).
type AMQPLinksImpl struct {
	// NOTE: values need to be 64-bit aligned. Simplest way to make sure this happens
	// is just to make it the first value in the struct
	// See:
	//   Godoc: https://pkg.go.dev/sync/atomic#pkg-note-BUG
	//   PR: https://github.com/Azure/azure-sdk-for-go/pull/16847
	id LinkID

	entityPath     string
	managementPath string
	audience       string
	createLink     CreateLinkFunc

	getRecoveryKindFunc func(err error) RecoveryKind

	mu sync.RWMutex

	// RPCLink lets you interact with the $management link for your entity.
	RPCLink amqpwrap.RPCLink

	// the AMQP session for either the 'sender' or 'receiver' link
	session amqpwrap.AMQPSession

	// these are populated by your `createLinkFunc` when you construct
	// the amqpLinks
	Sender   amqpwrap.AMQPSenderCloser
	Receiver amqpwrap.AMQPReceiverCloser

	// whether this links set has been closed permanently (via Close)
	// Recover() does not affect this value.
	closedPermanently bool

	cancelAuthRefreshLink     func()
	cancelAuthRefreshMgmtLink func()

	ns NamespaceForAMQPLinks

	// prefetchedMessagesAfterClose is called after a Receiver is closed. We pass all messages
	// we received using the Receiver.Prefetched() function.
	prefetchedMessagesAfterClose func(messages []*amqp.Message)

	utils.Logger
}

// CreateLinkFunc creates the links, using the given session. Typically you'll only create either an
// *amqp.Sender or a *amqp.Receiver. AMQPLinks handles it either way.
type CreateLinkFunc func(ctx context.Context, session amqpwrap.AMQPSession) (amqpwrap.AMQPSenderCloser, amqpwrap.AMQPReceiverCloser, error)

type NewAMQPLinksArgs struct {
	NS                  NamespaceForAMQPLinks
	EntityPath          string
	CreateLinkFunc      CreateLinkFunc
	GetRecoveryKindFunc func(err error) RecoveryKind

	// PrefetchedMessagesAfterClose is called after a Receiver (in ReceiveAndDelete mode) is closed. It gets passed all
	// the messages we could read from [amqp.Receiver.Prefetched].
	PrefetchedMessagesAfterClose func(messages []*amqp.Message)
}

// NewAMQPLinks creates a session, starts the claim refresher and creates an associated
// management link for a specific entity path.
func NewAMQPLinks(args NewAMQPLinksArgs) AMQPLinks {
	l := &AMQPLinksImpl{
		entityPath:                   args.EntityPath,
		managementPath:               fmt.Sprintf("%s/$management", args.EntityPath),
		audience:                     args.NS.GetEntityAudience(args.EntityPath),
		createLink:                   args.CreateLinkFunc,
		closedPermanently:            false,
		getRecoveryKindFunc:          args.GetRecoveryKindFunc,
		ns:                           args.NS,
		Logger:                       utils.NewLogger(),
		prefetchedMessagesAfterClose: args.PrefetchedMessagesAfterClose,
	}

	return l
}

// ManagementPath is the management path for the associated entity.
func (links *AMQPLinksImpl) ManagementPath() string {
	return links.managementPath
}

// errClosed is used when you try to use a closed link.
var errClosed = NewErrNonRetriable("link was closed by user")

// recoverLink will recycle all associated links (mgmt, receiver, sender and session)
// and recreate them using the link.linkCreator function.
func (links *AMQPLinksImpl) recoverLink(ctx context.Context, theirLinkRevision LinkID) error {
	links.Writef(exported.EventConn, "Recovering link only")

	links.mu.RLock()
	closedPermanently := links.closedPermanently
	ourLinkRevision := links.id
	links.mu.RUnlock()

	if closedPermanently {
		return errClosed
	}

	// cheap check before we do the lock
	if ourLinkRevision.Link != theirLinkRevision.Link {
		// we've already recovered past their failure.
		return nil
	}

	links.mu.Lock()
	defer links.mu.Unlock()

	// check once more, just in case someone else modified it before we took
	// the lock.
	if links.id.Link != theirLinkRevision.Link {
		// we've already recovered past their failure.
		return nil
	}

	// best effort close, the connection these were built on is gone.
	_ = links.closeWithoutLocking(ctx, false)
	err := links.initWithoutLocking(ctx)

	if err != nil {
		return err
	}

	return nil
}

// Recover will recover the links or the connection, depending
// on the severity of the error.
func (links *AMQPLinksImpl) RecoverIfNeeded(ctx context.Context, theirID LinkID, origErr error) error {
	if origErr == nil || IsCancelError(origErr) {
		return nil
	}

	links.Writef(exported.EventConn, "Recovering link for error %s", origErr.Error())

	rk := links.getRecoveryKindFunc(origErr)

	if rk == RecoveryKindLink {
		oldPrefix := links.Prefix()

		if err := links.recoverLink(ctx, theirID); err != nil {
			links.Writef(exported.EventConn, "Error when recovering link for recovery: %s", err)
			return err
		}

		links.Writef(exported.EventConn, "Recovered links (old: %s)", oldPrefix)
		return nil
	} else if rk == RecoveryKindConn {
		oldPrefix := links.Prefix()

		if err := links.recoverConnection(ctx, theirID); err != nil {
			links.Writef(exported.EventConn, "failed to recreate connection: %s", err.Error())
			return err
		}

		links.Writef(exported.EventConn, "Recovered connection and links (old: %s)", oldPrefix)
		return nil
	}

	links.Writef(exported.EventConn, "Recovered, no action needed")
	return nil
}

func (links *AMQPLinksImpl) recoverConnection(ctx context.Context, theirID LinkID) error {
	links.Writef(exported.EventConn, "Recovering connection (and links)")

	links.mu.Lock()
	defer links.mu.Unlock()

	if theirID.Link == links.id.Link {
		links.Writef(exported.EventConn, "closing old link: current:%v, old:%v", links.id, theirID)

		// we're clearing out this link because the connection is about to get recreated. So we can
		// safely ignore any problems here, we're just trying to make sure the state is reset.
		_ = links.closeWithoutLocking(ctx, false)
	}

	created, err := links.ns.Recover(ctx, uint64(theirID.Conn))

	if err != nil {
		links.Writef(exported.EventConn, "Recover connection failure: %s", err)
		return err
	}

	// We'll recreate the link if:
	// - `created` is true, meaning we recreated the AMQP connection (ie, all old links are invalid)
	// - the link they received an error on is our current link, so it needs to be recreated.
	//   (if it wasn't the same then we've already recovered and created a new link,
	//    so no recovery would be needed)
	if created || theirID.Link == links.id.Link {
		links.Writef(exported.EventConn, "recreating link: c: %v, current:%v, old:%v", created, links.id, theirID)

		// best effort close, the connection these were built on is gone.
		_ = links.closeWithoutLocking(ctx, false)

		if err := links.initWithoutLocking(ctx); err != nil {
			return err
		}
	}

	return nil
}

// LinkID is ID that represent our current link and the client used to create it.
// These are used when trying to determine what parts need to be recreated when
// an error occurs, to prevent recovering a connection/link repeatedly.
// See amqpLinks.RecoverIfNeeded() for usage.
type LinkID struct {
	// Conn is the ID of the connection we used to create our links.
	Conn uint64

	// Link is the ID of our current link.
	Link uint64
}

// Get will initialize a session and call its link.linkCreator function.
// If this link has been closed via Close() it will return an non retriable error.
func (l *AMQPLinksImpl) Get(ctx context.Context) (*LinksWithID, error) {
	l.mu.RLock()
	sender, receiver, mgmtLink, revision, closedPermanently := l.Sender, l.Receiver, l.RPCLink, l.id, l.closedPermanently
	l.mu.RUnlock()

	if closedPermanently {
		return nil, errClosed
	}

	if sender != nil || receiver != nil {
		return &LinksWithID{
			Sender:   sender,
			Receiver: receiver,
			RPC:      mgmtLink,
			ID:       revision,
		}, nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.initWithoutLocking(ctx); err != nil {
		return nil, err
	}

	return &LinksWithID{
		Sender:   l.Sender,
		Receiver: l.Receiver,
		RPC:      l.RPCLink,
		ID:       l.id,
	}, nil
}

func (links *AMQPLinksImpl) Retry(ctx context.Context, eventName log.Event, operation string, fn RetryWithLinksFn, o exported.RetryOptions) error {
	var lastID LinkID

	didQuickRetry := false

	isFatalErrorFunc := func(err error) bool {
		return links.getRecoveryKindFunc(err) == RecoveryKindFatal
	}

	return utils.Retry(ctx, eventName, links.Prefix()+"("+operation+")", func(ctx context.Context, args *utils.RetryFnArgs) error {
		if err := links.RecoverIfNeeded(ctx, lastID, args.LastErr); err != nil {
			return err
		}

		linksWithVersion, err := links.Get(ctx)

		if err != nil {
			return err
		}

		lastID = linksWithVersion.ID

		if err := fn(ctx, linksWithVersion, args); err != nil {
			if args.I == 0 && !didQuickRetry && IsLinkError(err) {
				// go-amqp will asynchronously handle detaches. This means errors that you get
				// back from Send(), for instance, can actually be from much earlier in time
				// depending on the last time you called into Send().
				//
				// This means we'll sometimes do an unneeded sleep after a failed retry when
				// it would have just immediately worked. To counteract that we'll do a one-time
				// quick attempt to recreate link immediately if we see a detach error. This might
				// waste a bit of time attempting to do the creation, but since it's just link creation
				// it should be fairly fast.
				//
				// So when we've received a detach is:
				//   0th attempt
				//   extra immediate 0th attempt (if last error was detach)
				//   (actual retries)
				//
				// Whereas normally you'd do (for non-detach errors):
				//   0th attempt
				//   (actual retries)
				links.Writef(exported.EventConn, "(%s) Link was previously detached. Attempting quick reconnect to recover from error: %s", operation, err.Error())
				didQuickRetry = true
				args.ResetAttempts()
			}

			return err
		}

		return nil
	}, isFatalErrorFunc, o)
}

// EntityPath is the full entity path for the queue/topic/subscription.
func (l *AMQPLinksImpl) EntityPath() string {
	return l.entityPath
}

// EntityPath is the audience for the queue/topic/subscription.
func (l *AMQPLinksImpl) Audience() string {
	return l.audience
}

// ClosedPermanently is true if AMQPLinks.Close(ctx, true) has been called.
func (l *AMQPLinksImpl) ClosedPermanently() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.closedPermanently
}

// Close will close the the link permanently.
// Any further calls to Get()/Recover() to return ErrLinksClosed.
func (l *AMQPLinksImpl) Close(ctx context.Context, permanent bool) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.closeWithoutLocking(ctx, permanent)
}

// CloseIfNeeded closes the links or connection if the error is recoverable.
// Use this if you want to make it so the _next_ call on your Sender/Receiver
// eats the cost of recovery, instead of doing it immediately. This is useful
// if you're trying to exit out of a function quickly but still need to react
// to a returned error.
func (links *AMQPLinksImpl) CloseIfNeeded(ctx context.Context, err error) RecoveryKind {
	links.mu.Lock()
	defer links.mu.Unlock()

	if IsCancelError(err) {
		links.Writef(exported.EventConn, "No close needed for cancellation")
		return RecoveryKindNone
	}

	rk := links.getRecoveryKindFunc(err)

	switch rk {
	case RecoveryKindLink:
		links.Writef(exported.EventConn, "Closing links for error %s", err.Error())
		_ = links.closeWithoutLocking(ctx, false)
		return rk
	case RecoveryKindFatal:
		links.Writef(exported.EventConn, "Fatal error cleanup")
		fallthrough
	case RecoveryKindConn:
		links.Writef(exported.EventConn, "Closing connection AND links for error %s", err.Error())
		_ = links.closeWithoutLocking(ctx, false)
		_ = links.ns.Close(false)
		return rk
	case RecoveryKindNone:
		return rk
	default:
		panic(fmt.Sprintf("Unhandled recovery kind %s for error %s", rk, err.Error()))
	}
}

// initWithoutLocking will create a new link, unconditionally.
func (links *AMQPLinksImpl) initWithoutLocking(ctx context.Context) error {
	tmpCancelAuthRefreshLink, _, err := links.ns.NegotiateClaim(ctx, links.entityPath)

	if err != nil {
		if err := links.closeWithoutLocking(ctx, false); err != nil {
			links.Writef(exported.EventConn, "Failure during link cleanup after negotiateClaim: %s", err.Error())
		}
		return err
	}

	links.cancelAuthRefreshLink = tmpCancelAuthRefreshLink

	tmpCancelAuthRefreshMgmtLink, _, err := links.ns.NegotiateClaim(ctx, links.managementPath)

	if err != nil {
		if err := links.closeWithoutLocking(ctx, false); err != nil {
			links.Writef(exported.EventConn, "Failure during link cleanup after negotiate claim for mgmt link: %s", err.Error())
		}
		return err
	}

	links.cancelAuthRefreshMgmtLink = tmpCancelAuthRefreshMgmtLink

	tmpSession, cr, err := links.ns.NewAMQPSession(ctx)

	if err != nil {
		if err := links.closeWithoutLocking(ctx, false); err != nil {
			links.Writef(exported.EventConn, "Failure during link cleanup after creating AMQP session: %s", err.Error())
		}
		return err
	}

	links.session = tmpSession
	links.id.Conn = cr

	tmpSender, tmpReceiver, err := links.createLink(ctx, links.session)

	if err != nil {
		if err := links.closeWithoutLocking(ctx, false); err != nil {
			links.Writef(exported.EventConn, "Failure during link cleanup after creating link: %s", err.Error())
		}
		return err
	}

	if tmpReceiver == nil && tmpSender == nil {
		panic("Both tmpReceiver and tmpSender are nil")
	}

	links.Sender, links.Receiver = tmpSender, tmpReceiver

	tmpRPCLink, err := links.ns.NewRPCLink(ctx, links.ManagementPath())

	if err != nil {
		if err := links.closeWithoutLocking(ctx, false); err != nil {
			links.Writef(exported.EventConn, "Failure during link cleanup after creating mgmt client: %s", err.Error())
		}
		return err
	}

	links.RPCLink = tmpRPCLink
	links.id.Link++

	if links.Sender != nil {
		linkName := links.Sender.LinkName()
		links.SetPrefix("c:%d, l:%d, s:name:%0.6s", links.id.Conn, links.id.Link, linkName)
	} else if links.Receiver != nil {
		linkName := links.Receiver.LinkName()
		links.SetPrefix("c:%d, l:%d, r:name:%0.6s", links.id.Conn, links.id.Link, linkName)
	}

	links.Writef(exported.EventConn, "Links created")
	return nil
}

// closeWithoutLocking closes the links ($management and normal entity links) and cancels the
// background authentication goroutines.
//
// If the context argument is cancelled we return amqpwrap.ErrConnResetNeeded, rather than
// context.Err(), as failing to close can leave our connection in an indeterminate
// state.
//
// Regardless of cancellation or Close() call failures, all local state will be cleaned up.
//
// NOTE: No locking is done in this function, call `Close` if you require locking.
func (links *AMQPLinksImpl) closeWithoutLocking(ctx context.Context, permanent bool) error {
	if links.closedPermanently {
		return nil
	}

	links.Writef(exported.EventConn, "Links closing (permanent: %v)", permanent)

	defer func() {
		if permanent {
			links.closedPermanently = true
		}
	}()

	var messages []string

	if links.cancelAuthRefreshLink != nil {
		links.cancelAuthRefreshLink()
		links.cancelAuthRefreshLink = nil
	}

	if links.cancelAuthRefreshMgmtLink != nil {
		links.cancelAuthRefreshMgmtLink()
		links.cancelAuthRefreshMgmtLink = nil
	}

	closeables := []struct {
		name     string
		instance amqpwrap.Closeable
	}{
		{"Sender", links.Sender},
		{"Receiver", links.Receiver},
		{"Session", links.session},
		{"RPC", links.RPCLink},
	}

	wasCancelled := false

	// only allow a max of defaultCloseTimeout - it's possible for Close() to hang
	// indefinitely if there's some sync issue between the service and us.
	for _, c := range closeables {
		if c.instance == nil {
			continue
		}

		links.Writef(exported.EventConn, "Closing %s", c.name)

		if err := c.instance.Close(ctx); err != nil {
			if IsCancelError(err) {
				wasCancelled = true
			}

			messages = append(messages, fmt.Sprintf("%s close error: %s", c.name, err.Error()))
		}
	}

	pushPrefetchedMessagesWithoutLocking(links, permanent)

	links.Sender, links.Receiver, links.session, links.RPCLink = nil, nil, nil, nil

	if wasCancelled {
		return ctx.Err()
	}

	if len(messages) > 0 {
		return errors.New(strings.Join(messages, "\n"))
	}

	return nil
}

// pushPrefetchedMessagesWithoutLocking clears the prefetched messages from the link. This function assumes
// the caller has taken care of proper locking and has already closed the Receiver link, if it exists.
func pushPrefetchedMessagesWithoutLocking(links *AMQPLinksImpl, permanent bool) {
	if !permanent { // only activate this when the Receiver is shut down. This avoids any possible concurrency issues.
		return
	}

	if links.Receiver == nil || links.prefetchedMessagesAfterClose == nil {
		return
	}

	var prefetched []*amqp.Message

	for {
		m := links.Receiver.Prefetched()

		if m == nil {
			break
		}

		prefetched = append(prefetched, m)
	}

	if len(prefetched) == 0 {
		links.Writef(exported.EventConn, "No messages on receiver after closing.")
		return
	}

	links.Writef(exported.EventConn, "Got %d messages on receiver after closing. These can be received using ReceiveMessages().", len(prefetched))
	links.prefetchedMessagesAfterClose(prefetched)
}
