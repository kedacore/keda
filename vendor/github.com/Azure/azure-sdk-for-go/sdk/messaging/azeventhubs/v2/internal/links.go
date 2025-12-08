// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package internal

import (
	"context"
	"fmt"
	"sync"

	azlog "github.com/Azure/azure-sdk-for-go/sdk/internal/log"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/amqpwrap"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/exported"
)

type AMQPLink interface {
	Close(ctx context.Context) error
	LinkName() string
}

// LinksForPartitionClient are the functions that the PartitionClient uses within Links[T]
// (for unit testing only)
type LinksForPartitionClient[LinkT AMQPLink] interface {
	// Retry is [Links.Retry]
	Retry(ctx context.Context, eventName azlog.Event, operation string, partitionID string, retryOptions exported.RetryOptions, fn func(ctx context.Context, lwid LinkWithID[LinkT]) error) error

	// Close is [Links.Close]
	Close(ctx context.Context) error
}

type Links[LinkT AMQPLink] struct {
	ns NamespaceForAMQPLinks

	linksMu *sync.RWMutex
	links   map[string]*linkState[LinkT]

	managementLinkMu *sync.RWMutex
	managementLink   *linkState[amqpwrap.RPCLink]

	managementPath string
	newLinkFn      NewLinksFn[LinkT]
	entityPathFn   func(partitionID string) string

	lr LinkRetrier[LinkT]
	mr LinkRetrier[amqpwrap.RPCLink]
}

type NewLinksFn[LinkT AMQPLink] func(ctx context.Context, session amqpwrap.AMQPSession, entityPath string, partitionID string) (LinkT, error)

func NewLinks[LinkT AMQPLink](ns NamespaceForAMQPLinks, managementPath string, entityPathFn func(partitionID string) string, newLinkFn NewLinksFn[LinkT]) *Links[LinkT] {
	l := &Links[LinkT]{
		ns:               ns,
		linksMu:          &sync.RWMutex{},
		links:            map[string]*linkState[LinkT]{},
		managementLinkMu: &sync.RWMutex{},
		managementPath:   managementPath,

		newLinkFn:    newLinkFn,
		entityPathFn: entityPathFn,
	}

	l.lr = LinkRetrier[LinkT]{
		GetLink:   l.GetLink,
		CloseLink: l.closePartitionLinkIfMatch,
		NSRecover: l.ns.Recover,
	}

	l.mr = LinkRetrier[amqpwrap.RPCLink]{
		GetLink: func(ctx context.Context, partitionID string) (LinkWithID[amqpwrap.RPCLink], error) {
			return l.GetManagementLink(ctx)
		},
		CloseLink: func(ctx context.Context, _, linkName string) error {
			return l.closeManagementLinkIfMatch(ctx, linkName)
		},
		NSRecover: l.ns.Recover,
	}

	return l
}

func (l *Links[LinkT]) RetryManagement(ctx context.Context, eventName azlog.Event, operation string, retryOptions exported.RetryOptions, fn func(ctx context.Context, lwid LinkWithID[amqpwrap.RPCLink]) error) error {
	return l.mr.Retry(ctx, eventName, operation, "", retryOptions, fn)
}

func (l *Links[LinkT]) Retry(ctx context.Context, eventName azlog.Event, operation string, partitionID string, retryOptions exported.RetryOptions, fn func(ctx context.Context, lwid LinkWithID[LinkT]) error) error {
	return l.lr.Retry(ctx, eventName, operation, partitionID, retryOptions, fn)
}

func (l *Links[LinkT]) GetLink(ctx context.Context, partitionID string) (LinkWithID[LinkT], error) {
	if err := l.checkOpen(); err != nil {
		return nil, err
	}

	l.linksMu.RLock()
	current := l.links[partitionID]
	l.linksMu.RUnlock()

	if current != nil {
		return current, nil
	}

	// no existing link, let's create a new one within the write lock.
	l.linksMu.Lock()
	defer l.linksMu.Unlock()

	// check again now that we have the write lock
	current = l.links[partitionID]

	if current == nil {
		ls, err := l.newLinkState(ctx, partitionID)

		if err != nil {
			return nil, err
		}

		l.links[partitionID] = ls
		current = ls
	}

	return current, nil
}

func (l *Links[LinkT]) GetManagementLink(ctx context.Context) (LinkWithID[amqpwrap.RPCLink], error) {
	if err := l.checkOpen(); err != nil {
		return nil, err
	}

	l.managementLinkMu.Lock()
	defer l.managementLinkMu.Unlock()

	if l.managementLink == nil {
		ls, err := l.newManagementLinkState(ctx)

		if err != nil {
			return nil, err
		}

		l.managementLink = ls
	}

	return l.managementLink, nil
}

func (l *Links[LinkT]) newLinkState(ctx context.Context, partitionID string) (*linkState[LinkT], error) {
	azlog.Writef(exported.EventConn, "Creating link for partition ID '%s'", partitionID)

	// check again now that we have the write lock
	ls := &linkState[LinkT]{
		partitionID: partitionID,
	}

	cancelAuth, _, err := l.ns.NegotiateClaim(ctx, l.entityPathFn(partitionID))

	if err != nil {
		azlog.Writef(exported.EventConn, "(%s): Failed to negotiate claim for partition ID '%s': %s", ls.String(), partitionID, err)
		return nil, err
	}

	ls.cancelAuth = cancelAuth

	session, connID, err := l.ns.NewAMQPSession(ctx)

	if err != nil {
		azlog.Writef(exported.EventConn, "(%s): Failed to create AMQP session for partition ID '%s': %s", ls.String(), partitionID, err)
		_ = ls.Close(ctx)
		return nil, err
	}

	ls.session = session
	ls.connID = connID

	tmpLink, err := l.newLinkFn(ctx, session, l.entityPathFn(partitionID), partitionID)

	if err != nil {
		azlog.Writef(exported.EventConn, "(%s): Failed to create link for partition ID '%s': %s", ls.String(), partitionID, err)
		_ = ls.Close(ctx)
		return nil, err
	}

	ls.link = &tmpLink

	azlog.Writef(exported.EventConn, "(%s): Succesfully created link for partition ID '%s'", ls.String(), partitionID)
	return ls, nil
}

func (l *Links[LinkT]) newManagementLinkState(ctx context.Context) (*linkState[amqpwrap.RPCLink], error) {
	ls := &linkState[amqpwrap.RPCLink]{}

	cancelAuth, _, err := l.ns.NegotiateClaim(ctx, l.managementPath)

	if err != nil {
		return nil, err
	}

	ls.cancelAuth = cancelAuth

	tmpRPCLink, connID, err := l.ns.NewRPCLink(ctx, "$management")

	if err != nil {
		_ = ls.Close(ctx)
		return nil, err
	}

	ls.connID = connID
	ls.link = &tmpRPCLink

	return ls, nil
}

func (l *Links[LinkT]) Close(ctx context.Context) error {
	return l.closeLinks(ctx, true)
}

func (l *Links[LinkT]) closeLinks(ctx context.Context, permanent bool) error {
	cancelled := false

	// clear out the management link
	func() {
		l.managementLinkMu.Lock()
		defer l.managementLinkMu.Unlock()

		if l.managementLink == nil {
			return
		}

		mgmtLink := l.managementLink
		l.managementLink = nil

		if err := mgmtLink.Close(ctx); err != nil {
			azlog.Writef(exported.EventConn, "Error while cleaning up management link while doing connection recovery: %s", err.Error())

			if IsCancelError(err) {
				cancelled = true
			}
		}
	}()

	l.linksMu.Lock()
	defer l.linksMu.Unlock()

	tmpLinks := l.links
	l.links = nil

	for partitionID, link := range tmpLinks {
		if err := link.Close(ctx); err != nil {
			azlog.Writef(exported.EventConn, "Error while cleaning up link for partition ID '%s' while doing connection recovery: %s", partitionID, err.Error())

			if IsCancelError(err) {
				cancelled = true
			}
		}
	}

	if !permanent {
		l.links = map[string]*linkState[LinkT]{}
	}

	if cancelled {
		// this is the only kind of error I'd consider usable from Close() - it'll indicate
		// that some of the links haven't been cleanly closed.
		return ctx.Err()
	}

	return nil
}

func (l *Links[LinkT]) checkOpen() error {
	l.linksMu.RLock()
	defer l.linksMu.RUnlock()

	if l.links == nil {
		return NewErrNonRetriable("client has been closed by user")
	}

	return nil
}

// closePartitionLinkIfMatch will close the link in the cache if it matches the passed in linkName.
// This is similar to how an etag works - we'll only close it if you are working with the latest link -
// if not, it's a no-op since somebody else has already 'saved' (recovered) before you.
//
// Note that the only error that can be returned here will come from go-amqp. Cleanup of _our_ internal state
// will always happen, if needed.
func (l *Links[LinkT]) closePartitionLinkIfMatch(ctx context.Context, partitionID string, linkName string) error {
	l.linksMu.RLock()
	current, exists := l.links[partitionID]
	l.linksMu.RUnlock()

	if !exists ||
		current.Link().LinkName() != linkName { // we've already created a new link, their link was stale.
		return nil
	}

	l.linksMu.Lock()
	defer l.linksMu.Unlock()

	current, exists = l.links[partitionID]

	if !exists ||
		current.Link().LinkName() != linkName { // we've already created a new link, their link was stale.
		return nil
	}

	delete(l.links, partitionID)
	return current.Close(ctx)
}

func (l *Links[LinkT]) closeManagementLinkIfMatch(ctx context.Context, linkName string) error {
	l.managementLinkMu.Lock()
	defer l.managementLinkMu.Unlock()

	if l.managementLink != nil && l.managementLink.Link().LinkName() == linkName {
		err := l.managementLink.Close(ctx)
		l.managementLink = nil
		return err
	}

	return nil
}

type linkState[LinkT AMQPLink] struct {
	// connID is an arbitrary (but unique) integer that represents the
	// current connection. This comes back from the Namespace, anytime
	// it hands back a connection.
	connID uint64

	// link will be either an [amqpwrap.AMQPSenderCloser], [amqpwrap.AMQPReceiverCloser] or [amqpwrap.RPCLink]
	link *LinkT

	// partitionID, if available.
	partitionID string

	// cancelAuth cancels the backround claim negotation for this link.
	cancelAuth func()

	// optional session, if we created one for this
	// link.
	session amqpwrap.AMQPSession
}

// String returns a string that can be used for logging, of the format:
// (c:<connid>,l:<5 characters of link id>)
//
// It can also handle nil and partial initialization.
func (ls *linkState[LinkT]) String() string {
	if ls == nil {
		return "none"
	}

	linkName := ""

	if ls.link != nil {
		linkName = ls.Link().LinkName()
	}

	return formatLogPrefix(ls.connID, linkName, ls.partitionID)
}

// Close cancels the background authentication loop for this link and
// then closes the AMQP links.
// NOTE: this avoids any issues where closing fails on the broker-side or
// locally and we leak a goroutine.
func (ls *linkState[LinkT]) Close(ctx context.Context) error {
	if ls.cancelAuth != nil {
		ls.cancelAuth()
	}

	var linkCloseErr error

	if ls.link != nil {
		// we're more interested in a link failing to close than we are in
		// the session.
		linkCloseErr = ls.Link().Close(ctx)
	}

	if ls.session != nil {
		_ = ls.session.Close(ctx)
	}

	return linkCloseErr
}

func (ls *linkState[LinkT]) PartitionID() string {
	return ls.partitionID
}

func (ls *linkState[LinkT]) ConnID() uint64 {
	return ls.connID
}

func (ls *linkState[LinkT]) Link() LinkT {
	return *ls.link
}

// LinkWithID is a readonly interface over the top of a linkState.
type LinkWithID[LinkT AMQPLink] interface {
	ConnID() uint64
	Link() LinkT
	PartitionID() string
	Close(ctx context.Context) error
	String() string
}

func formatLogPrefix(connID uint64, linkName, partitionID string) string {
	return fmt.Sprintf("c:%d,l:%.5s,p:%s", connID, linkName, partitionID)
}
