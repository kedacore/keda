// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package internal

import (
	"context"
	"errors"

	azlog "github.com/Azure/azure-sdk-for-go/sdk/internal/log"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/amqpwrap"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/utils"
)

type LinkRetrier[LinkT AMQPLink] struct {
	// GetLink is set to [Links.GetLink]
	GetLink func(ctx context.Context, partitionID string) (LinkWithID[LinkT], error)

	// CloseLink is set to [Links.closePartitionLinkIfMatch]
	CloseLink func(ctx context.Context, partitionID string, linkName string) error

	// NSRecover is set to [Namespace.Recover]
	NSRecover func(ctx context.Context, connID uint64) error
}

type RetryCallback[LinkT AMQPLink] func(ctx context.Context, lwid LinkWithID[LinkT]) error

// Retry runs the fn argument in a loop, respecting retry counts.
// If connection/link failures occur it also takes care of running recovery logic
// to bring them back, or return an appropriate error if retries are exhausted.
func (l LinkRetrier[LinkT]) Retry(ctx context.Context,
	eventName azlog.Event,
	operation string,
	partitionID string,
	retryOptions exported.RetryOptions,
	fn RetryCallback[LinkT]) error {

	isFatalErrorFunc := func(err error) bool {
		return GetRecoveryKind(err) == RecoveryKindFatal
	}

	currentPrefix := ""

	prefix := func() string {
		return currentPrefix
	}

	// Track if we've done the one-time quick retry for async detach handling.
	// go-amqp handles detaches asynchronously, so errors from Send() etc. can be
	// from earlier detaches. We do one immediate retry to avoid unnecessary backoff
	// when the link may already be recovered.
	didQuickRetry := false

	return utils.Retry(ctx, eventName, prefix, retryOptions, func(ctx context.Context, args *utils.RetryFnArgs) error {
		linkWithID, err := l.GetLink(ctx, partitionID)

		if err != nil {
			return err
		}

		currentPrefix = linkWithID.String()

		if err := fn(ctx, linkWithID); err != nil {
			// Quick retry: on the first attempt, if we get a link error, do one immediate
			// retry without backoff. This handles the case where go-amqp's async detach
			// processing means the error is from an earlier detach and the link is already
			// ready to be recreated.
			if args.I == 0 && !didQuickRetry && IsQuickRecoveryError(err) {
				azlog.Writef(exported.EventConn, "(%s) Link was previously detached. Attempting quick reconnect to recover from error: %s", currentPrefix, err.Error())
				didQuickRetry = true
				args.ResetAttempts()
			}

			if recoveryErr := l.RecoverIfNeeded(ctx, err); recoveryErr != nil {
				// it's okay to return this error, and we're still in an okay state. The next loop through will end
				// up reopening all the closed links and will either get the same error again (ie, network is _still_
				// down) or will work and then things proceed as normal.
				return recoveryErr
			}

			// it's critical that we still return the original error here (that came from fn()) and NOT nil,
			// otherwise we'll end up terminating the retry loop.
			return err
		}

		return nil
	}, isFatalErrorFunc)
}

// RecoverIfNeeded will check the error and pick the correct minimal recovery pattern (none, link only, connection and link, etc..)
// NOTE: if 'ctx' is cancelled this function will still close out all the connections/links involved.
func (l LinkRetrier[LinkT]) RecoverIfNeeded(ctx context.Context, err error) error {
	rk := GetRecoveryKind(err)

	switch rk {
	case RecoveryKindNone:
		return nil
	case RecoveryKindLink:
		var awErr amqpwrap.Error

		if !errors.As(err, &awErr) {
			azlog.Writef(exported.EventConn, "RecoveryKindLink, but not an amqpwrap.Error: %T,%v", err, err)
			return nil
		}

		if err := l.CloseLink(ctx, awErr.PartitionID, awErr.LinkName); err != nil {
			azlog.Writef(exported.EventConn, "(%s) Error when cleaning up old link for link recovery: %s", formatLogPrefix(awErr.ConnID, awErr.LinkName, awErr.PartitionID), err)
			return err
		}

		return nil
	case RecoveryKindConn:
		var awErr amqpwrap.Error

		if !errors.As(err, &awErr) {
			azlog.Writef(exported.EventConn, "RecoveryKindConn, but not an amqpwrap.Error: %T,%v", err, err)
			return nil
		}

		// We only close _this_ partition's link. Other partitions will also get an error, and will recover.
		// We used to close _all_ the links, but no longer do that since it's possible (when we do receiver
		// redirect) to have more than one active connection at a time which means not all links would be
		// affected when a single connection goes down.
		if err := l.CloseLink(ctx, awErr.PartitionID, awErr.LinkName); err != nil {
			azlog.Writef(exported.EventConn, "(%s) Error when cleaning up old link: %s", formatLogPrefix(awErr.ConnID, awErr.LinkName, awErr.PartitionID), err)

			// NOTE: this is best effort - it's probable the connection is dead anyways so we'll log
			// but ignore the error for recovery purposes.
		}

		// There are two possibilities here:
		//
		// 1. (stale) The caller got this error but the `lwid` they're passing us is 'stale' - ie, '
		//    the connection the error happened on doesn't exist anymore (we recovered already) or
		//    the link itself is no longer active in our cache.
		//
		// 2. (current) The caller got this error and is the current link and/or connection, so we're going to
		//    need to recycle the connection (possibly) and links.
		//
		// For #1, we basically don't need to do anything. Recover(old-connection-id) will be a no-op
		// and the closePartitionLinkIfMatch() will no-op as well since the link they passed us will
		// not match the current link.
		//
		// For #2, we may recreate the connection. It's possible we won't if the connection itself
		// has already been recovered by another goroutine.
		err := l.NSRecover(ctx, awErr.ConnID)

		if err != nil {
			azlog.Writef(exported.EventConn, "(%s) Failure recovering connection for link: %s", formatLogPrefix(awErr.ConnID, awErr.LinkName, awErr.PartitionID), err)
			return err
		}

		return nil
	default:
		return err
	}
}
