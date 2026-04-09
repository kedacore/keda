// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package utils

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/internal/log"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/exported"
)

type RetryFnArgs struct {
	// I is the iteration of the retry "loop" and starts at 0.
	// The 0th iteration is the first call, and doesn't count as a retry.
	// The last try will equal RetryOptions.MaxRetries
	I int32
	// LastErr is the returned error from the previous loop.
	// If you have potentially expensive
	LastErr error

	resetAttempts bool
}

// ResetAttempts resets all Retry() attempts, starting back
// at iteration 0.
func (rf *RetryFnArgs) ResetAttempts() {
	rf.resetAttempts = true
}

// Retry runs a standard retry loop. It executes your passed in fn as the body of the loop.
// It returns if it exceeds the number of configured retry options or if 'isFatal' returns true.
func Retry(ctx context.Context, eventName log.Event, prefix func() string, o exported.RetryOptions, fn func(ctx context.Context, callbackArgs *RetryFnArgs) error, isFatalFn func(err error) bool) error {
	if isFatalFn == nil {
		panic("isFatalFn is nil, errors would panic")
	}

	ro := o
	setDefaults(&ro)

	var err error

	for i := int32(0); i <= ro.MaxRetries; i++ {
		if i > 0 {
			sleep := calcDelay(ro, i)
			log.Writef(eventName, "(%s) Retry attempt %d sleeping for %s", prefix(), i, sleep)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(sleep):
			}
		}

		args := RetryFnArgs{
			I:       i,
			LastErr: err,
		}
		err = fn(ctx, &args)

		if args.resetAttempts {
			log.Writef(eventName, "(%s) Resetting retry attempts", prefix())

			// it looks weird, but we're doing -1 here because the post-increment
			// will set it back to 0, which is what we want - go back to the 0th
			// iteration so we don't sleep before the attempt.
			//
			// You'll use this when you want to get another "fast" retry attempt.
			i = int32(-1)
		}

		if err != nil {
			if isFatalFn(err) {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					log.Writef(eventName, "(%s) Retry attempt %d was cancelled, stopping: %s", prefix(), i, err.Error())
				} else {
					log.Writef(eventName, "(%s) Retry attempt %d returned non-retryable error: %s", prefix(), i, err.Error())
				}
				return err
			} else {
				log.Writef(eventName, "(%s) Retry attempt %d returned retryable error: %s", prefix(), i, err.Error())
			}

			continue
		}

		return nil
	}

	return err
}

func setDefaults(o *exported.RetryOptions) {
	if o.MaxRetries == 0 {
		o.MaxRetries = 3
	} else if o.MaxRetries < 0 {
		o.MaxRetries = 0
	}
	if o.MaxRetryDelay == 0 {
		o.MaxRetryDelay = 120 * time.Second
	} else if o.MaxRetryDelay < 0 {
		// not really an unlimited cap, but sufficiently large enough to be considered as such
		o.MaxRetryDelay = math.MaxInt64
	}
	if o.RetryDelay == 0 {
		o.RetryDelay = 4 * time.Second
	} else if o.RetryDelay < 0 {
		o.RetryDelay = 0
	}
}

// (adapted from from azcore/policy_retry)
func calcDelay(o exported.RetryOptions, try int32) time.Duration { // try is >=1; never 0
	// avoid overflow when shifting left
	factor := time.Duration(math.MaxInt64)
	if try < 63 {
		factor = time.Duration(int64(1<<try) - 1)
	}

	delay := factor * o.RetryDelay
	if delay < factor {
		// overflow has happened so set to max value
		delay = time.Duration(math.MaxInt64)
	}

	// Introduce jitter:  [0.0, 1.0) / 2 = [0.0, 0.5) + 0.8 = [0.8, 1.3)
	jitterMultiplier := rand.Float64()/2 + 0.8 // NOTE: We want math/rand; not crypto/rand

	delayFloat := float64(delay) * jitterMultiplier
	if delayFloat > float64(math.MaxInt64) {
		// the jitter pushed us over MaxInt64, so just use MaxInt64
		delay = time.Duration(math.MaxInt64)
	} else {
		delay = time.Duration(delayFloat)
	}

	if delay > o.MaxRetryDelay { // MaxRetryDelay is backfilled with non-negative value
		delay = o.MaxRetryDelay
	}

	return delay
}
