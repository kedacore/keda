// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package internal

import (
	"context"
	"errors"

	azlog "github.com/Azure/azure-sdk-for-go/sdk/internal/log"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/internal/amqpwrap"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/internal/utils"
)

type LinkRetrier[LinkT AMQPLink] struct {
	GetLink   func(ctx context.Context, partitionID string) (LinkWithID[LinkT], error)
	CloseLink func(ctx context.Context, partitionID string, linkName string) error
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
	didQuickRetry := false

	isFatalErrorFunc := func(err error) bool {
		return GetRecoveryKind(err) == RecoveryKindFatal
	}

	currentPrefix := ""

	prefix := func() string {
		return currentPrefix
	}

	return utils.Retry(ctx, eventName, prefix, retryOptions, func(ctx context.Context, args *utils.RetryFnArgs) error {
		if err := l.RecoverIfNeeded(ctx, args.LastErr); err != nil {
			return err
		}

		linkWithID, err := l.GetLink(ctx, partitionID)

		if err != nil {
			return err
		}

		currentPrefix = linkWithID.String()

		if err := fn(ctx, linkWithID); err != nil {
			if args.I == 0 && !didQuickRetry && IsQuickRecoveryError(err) {
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
				azlog.Writef(exported.EventConn, "(%s, %s) Link was previously detached. Attempting quick reconnect to recover from error: %s", linkWithID.String(), operation, err.Error())
				didQuickRetry = true
				args.ResetAttempts()
			}

			return err
		}

		return nil
	}, isFatalErrorFunc)
}

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
