// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package internal

import (
	"context"
	"errors"
	"time"
)

type LocalIdleTracker struct {
	// MaxDuration controls how long we'll wait, on the client side, for the first message
	MaxDuration time.Duration

	// IdleStart marks the first time the user cancelled in a string of cancellations.
	// It gets reset any time there is a success or a non-cancel related failure.
	// NOTE: this is public for some unit tests but isn't intended to be set by external code.
	IdleStart time.Time
}

var localIdleError = errors.New("link was idle, detaching (will be reattached).")

func IsLocalIdleError(err error) bool {
	return errors.Is(err, localIdleError)
}

// NewContextWithDeadline creates a context that has an appropriate deadline that will expire
// when the idle period has completed.
func (idle *LocalIdleTracker) NewContextWithDeadline(ctx context.Context) (context.Context, context.CancelFunc) {
	if idle.IdleStart.IsZero() {
		// we're not in the middle of an idle period, so we'll start from now.
		return context.WithTimeout(ctx, idle.MaxDuration)
	}

	// we've already idled before.
	return context.WithDeadline(ctx, idle.IdleStart.Add(idle.MaxDuration))
}

// Check checks if we are actually idle, taking into account when we initially
// started being idle ([idle.IdleStart]) vs the current time.
//
// If it turns out the link should be considered idle it'll return idleError.
// Else, it'll return the err parameter.
func (idle *LocalIdleTracker) Check(parentCtx context.Context, operationStart time.Time, err error) error {
	if err == nil || !IsCancelError(err) {
		// either no error occurred (in which case the link is working)
		// or a non-cancel error happened. The non-cancel error will just
		// be handled by the normal recovery path.
		idle.IdleStart = time.Time{}
		return err
	}

	// okay, we're dealing with a cancellation error. Was it the user cancelling (ie, parentCtx) or
	// was it our idle timer firing?
	if parentCtx.Err() != nil {
		// The user cancelled. These cancels come from a single Receive() call on the link, which means we
		// didn't get a message back.
		if idle.IdleStart.IsZero() {
			idle.IdleStart = operationStart
		}

		return err
	}

	// It's our idle timeout that caused us to cancel, which means the idle interval has expired.
	// We'll clear our internally stored time and indicate we're idle with the sentinel 'idleError'
	idle.IdleStart = time.Time{}

	return localIdleError
}
