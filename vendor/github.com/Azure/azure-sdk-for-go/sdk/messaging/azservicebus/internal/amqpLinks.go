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
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/amqpwrap"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/utils"
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

const defaultCloseTimeout = time.Minute

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

	contextWithTimeoutFn contextWithTimeoutFn
}

// CreateLinkFunc creates the links, using the given session. Typically you'll only create either an
// *amqp.Sender or a *amqp.Receiver. AMQPLinks handles it either way.
type CreateLinkFunc func(ctx context.Context, session amqpwrap.AMQPSession) (amqpwrap.AMQPSenderCloser, amqpwrap.AMQPReceiverCloser, error)

type NewAMQPLinksArgs struct {
	NS                  NamespaceForAMQPLinks
	EntityPath          string
	CreateLinkFunc      CreateLinkFunc
	GetRecoveryKindFunc func(err error) RecoveryKind
}

// NewAMQPLinks creates a session, starts the claim refresher and creates an associated
// management link for a specific entity path.
func NewAMQPLinks(args NewAMQPLinksArgs) AMQPLinks {
	l := &AMQPLinksImpl{
		entityPath:           args.EntityPath,
		managementPath:       fmt.Sprintf("%s/$management", args.EntityPath),
		audience:             args.NS.GetEntityAudience(args.EntityPath),
		createLink:           args.CreateLinkFunc,
		closedPermanently:    false,
		getRecoveryKindFunc:  args.GetRecoveryKindFunc,
		ns:                   args.NS,
		contextWithTimeoutFn: context.WithTimeout,
	}

	return l
}

// ManagementPath is the management path for the associated entity.
func (links *AMQPLinksImpl) ManagementPath() string {
	return links.managementPath
}

// recoverLink will recycle all associated links (mgmt, receiver, sender and session)
// and recreate them using the link.linkCreator function.
func (links *AMQPLinksImpl) recoverLink(ctx context.Context, theirLinkRevision LinkID) error {
	log.Writef(exported.EventConn, "Recovering link only")

	links.mu.RLock()
	closedPermanently := links.closedPermanently
	ourLinkRevision := links.id
	links.mu.RUnlock()

	if closedPermanently {
		return NewErrNonRetriable("link was closed by user")
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

	log.Writef(exported.EventConn, "Recovering link for error %s", origErr.Error())

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	rk := links.getRecoveryKindFunc(origErr)

	if rk == RecoveryKindLink {
		if err := links.recoverLink(ctx, theirID); err != nil {
			if errors.Is(err, errConnResetNeeded) {
				log.Writef(exported.EventConn, "Connection reset for recovery instead of link. Link closing has timed out.")

				if err := links.recoverConnection(ctx, theirID); err != nil {
					log.Writef(exported.EventConn, "failed to recreate connection: %s", err.Error())
				}
				return err
			} else {
				log.Writef(exported.EventConn, "failed to recreate link: %s", err.Error())
				return err
			}
		}

		log.Writef(exported.EventConn, "Recovered links")
		return nil
	} else if rk == RecoveryKindConn {
		if err := links.recoverConnection(ctx, theirID); err != nil {
			log.Writef(exported.EventConn, "failed to recreate connection: %s", err.Error())
			return err
		}

		log.Writef(exported.EventConn, "Recovered connection and links")
		return nil
	}

	log.Writef(exported.EventConn, "Recovered, no action needed")
	return nil
}

func (links *AMQPLinksImpl) recoverConnection(ctx context.Context, theirID LinkID) error {
	log.Writef(exported.EventConn, "Recovering connection (and links)")

	links.mu.Lock()
	defer links.mu.Unlock()

	created, err := links.ns.Recover(ctx, uint64(theirID.Conn))

	if err != nil {
		log.Writef(exported.EventConn, "Recover connection failure: %s", err)
		return err
	}

	// We'll recreate the link if:
	// - `created` is true, meaning we recreated the AMQP connection (ie, all old links are invalid)
	// - the link they received an error on is our current link, so it needs to be recreated.
	//   (if it wasn't the same then we've already recovered and created a new link,
	//    so no recovery would be needed)
	if created || theirID.Link == links.id.Link {
		log.Writef(exported.EventConn, "recreating link: c: %v, current:%v, old:%v", created, links.id, theirID)
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
		return nil, NewErrNonRetriable("link was closed by user")
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

func (l *AMQPLinksImpl) Retry(ctx context.Context, eventName log.Event, operation string, fn RetryWithLinksFn, o exported.RetryOptions) error {
	var lastID LinkID

	didQuickRetry := false

	isFatalErrorFunc := func(err error) bool {
		return l.getRecoveryKindFunc(err) == RecoveryKindFatal
	}

	return utils.Retry(ctx, eventName, operation, func(ctx context.Context, args *utils.RetryFnArgs) error {
		if err := l.RecoverIfNeeded(ctx, lastID, args.LastErr); err != nil {
			return err
		}

		linksWithVersion, err := l.Get(ctx)

		if err != nil {
			return err
		}

		lastID = linksWithVersion.ID

		if err := fn(ctx, linksWithVersion, args); err != nil {
			if args.I == 0 && !didQuickRetry && IsDetachError(err) {
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
				log.Writef(exported.EventConn, "(%s) Link was previously detached. Attempting quick reconnect to recover from error: %s", operation, err.Error())
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
func (l *AMQPLinksImpl) CloseIfNeeded(ctx context.Context, err error) RecoveryKind {
	l.mu.Lock()
	defer l.mu.Unlock()

	if IsCancelError(err) {
		log.Writef(exported.EventConn, "No close needed for cancellation")
		return RecoveryKindNone
	}

	rk := l.getRecoveryKindFunc(err)

	switch rk {
	case RecoveryKindLink:
		log.Writef(exported.EventConn, "Closing links for error %s", err.Error())

		ctx, cancel := l.contextWithTimeoutFn(ctx, defaultCloseTimeout)
		defer cancel()

		if err := l.closeWithoutLocking(ctx, false); err != nil {
			// Two cases:
			//
			// 1. Context is cancelled by the user. This interrupted some recovery logic so our best bet is just to close
			//    the connection, which will cause it to get recovered on the next call, or the user will need to recover
			//    it. The important bit is that cancelling does the best job it can of leaving things in a clean state.
			//
			// 2. We time out - now that we have an internal timeout here we can ensure that we don't just hang forever
			//    internally and force or notify that a reset is needed. This will get passed into the next recovery call
			//    and we'll do the normal recovery logic from there.

			if errors.Is(err, errConnResetNeeded) {
				log.Writef(exported.EventConn, "Connection closed instead. Link closing has timed out.")

				// if closing leaves us in an indeterminate state we'll go ahead
				// and close the old connection, forcing recovery.
				_ = l.ns.Close(false)

				// upgrade to connection level closing instead.
				rk = RecoveryKindConn
			}
		}

		return rk
	case RecoveryKindFatal:
		log.Writef(exported.EventConn, "Fatal error cleanup")
		fallthrough
	case RecoveryKindConn:
		log.Writef(exported.EventConn, "Closing connection AND links for error %s", err.Error())
		_ = l.closeWithoutLocking(ctx, false)
		_ = l.ns.Close(false)
		return rk
	case RecoveryKindNone:
		return rk
	default:
		panic(fmt.Sprintf("Unhandled recovery kind %s for error %s", rk, err.Error()))
	}
}

// initWithoutLocking will create a new link, unconditionally.
func (l *AMQPLinksImpl) initWithoutLocking(ctx context.Context) error {
	// shut down any links we have
	if err := l.closeWithoutLocking(ctx, false); err != nil {
		// connection is destabilized since we can't close the link.
		if errors.Is(err, errConnResetNeeded) {
			return err
		}
		// any other error won't affect future links so we can ignore it.
	}

	tmpCancelAuthRefreshLink, _, err := l.ns.NegotiateClaim(ctx, l.entityPath)

	if err != nil {
		if err := l.closeWithoutLocking(ctx, false); err != nil {
			log.Writef(exported.EventConn, "Failure during link cleanup after negotiateClaim: %s", err.Error())
		}
		return err
	}

	l.cancelAuthRefreshLink = tmpCancelAuthRefreshLink

	tmpCancelAuthRefreshMgmtLink, _, err := l.ns.NegotiateClaim(ctx, l.managementPath)

	if err != nil {
		if err := l.closeWithoutLocking(ctx, false); err != nil {
			log.Writef(exported.EventConn, "Failure during link cleanup after negotiate claim for mgmt link: %s", err.Error())
		}
		return err
	}

	l.cancelAuthRefreshMgmtLink = tmpCancelAuthRefreshMgmtLink

	tmpSession, cr, err := l.ns.NewAMQPSession(ctx)

	if err != nil {
		if err := l.closeWithoutLocking(ctx, false); err != nil {
			log.Writef(exported.EventConn, "Failure during link cleanup after creating AMQP session: %s", err.Error())
		}
		return err
	}

	l.session = tmpSession
	l.id.Conn = cr

	tmpSender, tmpReceiver, err := l.createLink(ctx, l.session)

	if err != nil {
		if err := l.closeWithoutLocking(ctx, false); err != nil {
			log.Writef(exported.EventConn, "Failure during link cleanup after creating link: %s", err.Error())
		}
		return err
	}

	l.Sender, l.Receiver = tmpSender, tmpReceiver

	tmpRPCLink, err := l.ns.NewRPCLink(ctx, l.ManagementPath())

	if err != nil {
		if err := l.closeWithoutLocking(ctx, false); err != nil {
			log.Writef("Failure during link cleanup after creating mgmt client: %s", err.Error())
		}
		return err
	}

	l.RPCLink = tmpRPCLink
	l.id.Link++
	return nil
}

// closeWithoutLocking closes the links ($management and normal entity links) and cancels the
// background authentication goroutines.
//
// If the context argument is cancelled we return errConnResetNeeded, rather than
// context.Err(), as failing to close can leave our connection in an indeterminate
// state.
//
// Regardless of cancellation or Close() call failures, all local state will be cleaned up.
//
// NOTE: No locking is done in this function, call `Close` if you require locking.
func (l *AMQPLinksImpl) closeWithoutLocking(ctx context.Context, permanent bool) error {
	if l.closedPermanently {
		return nil
	}

	defer func() {
		if permanent {
			l.closedPermanently = true
		}
	}()

	var messages []string

	if l.cancelAuthRefreshLink != nil {
		l.cancelAuthRefreshLink()
		l.cancelAuthRefreshLink = nil
	}

	if l.cancelAuthRefreshMgmtLink != nil {
		l.cancelAuthRefreshMgmtLink()
		l.cancelAuthRefreshMgmtLink = nil
	}

	closeables := []struct {
		name     string
		instance amqpwrap.Closeable
	}{
		{"Sender", l.Sender},
		{"Receiver", l.Receiver},
		{"Session", l.session},
		{"RPC", l.RPCLink},
	}

	wasCancelled := false

	// only allow a max of defaultCloseTimeout - it's possible for Close() to hang
	// indefinitely if there's some sync issue between the service and us.
	ctx, cancel := l.contextWithTimeoutFn(ctx, defaultCloseTimeout)
	defer cancel()

	for _, c := range closeables {
		if c.instance == nil {
			continue
		}

		if err := c.instance.Close(ctx); err != nil {
			if IsCancelError(err) {
				wasCancelled = true
			}

			messages = append(messages, fmt.Sprintf("%s close error: %s", c.name, err.Error()))
		}
	}

	l.Sender, l.Receiver, l.session, l.RPCLink = nil, nil, nil, nil

	if wasCancelled {
		return errConnResetNeeded
	}

	if len(messages) > 0 {
		return errors.New(strings.Join(messages, "\n"))
	}

	return nil
}
