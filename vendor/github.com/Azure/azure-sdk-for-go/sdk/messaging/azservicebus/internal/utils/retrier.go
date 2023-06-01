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
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/exported"
)

// EventRetry is the name for retry events
const EventRetry = "azsb.Retry"

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
func Retry(ctx context.Context, eventName log.Event, operation string, fn func(ctx context.Context, args *RetryFnArgs) error, isFatalFn func(err error) bool, o exported.RetryOptions) error {
	if isFatalFn == nil {
		panic("isFatalFn is nil, errors would panic")
	}

	var ro exported.RetryOptions = o
	setDefaults(&ro)

	var err error

	for i := int32(0); i <= ro.MaxRetries; i++ {
		if i > 0 {
			sleep := calcDelay(ro, i)
			log.Writef(eventName, "%s Retry attempt %d sleeping for %s", operation, i, sleep)

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
			log.Writef(eventName, "%s Resetting retry attempts", operation)

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
					log.Writef(eventName, "%s Retry attempt %d was cancelled, stopping: %s", operation, i, err.Error())
				} else {
					log.Writef(eventName, "%s Retry attempt %d returned non-retryable error: %s", operation, i, err.Error())
				}
				return err
			} else {
				log.Writef(eventName, "%s Retry attempt %d returned retryable error: %s", operation, i, err.Error())
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
func calcDelay(o exported.RetryOptions, try int32) time.Duration {
	if try == 0 {
		return 0
	}

	pow := func(number int64, exponent int32) int64 { // pow is nested helper function
		var result int64 = 1
		for n := int32(0); n < exponent; n++ {
			result *= number
		}
		return result
	}

	delay := time.Duration(pow(2, try)-1) * o.RetryDelay

	// Introduce some jitter:  [0.0, 1.0) / 2 = [0.0, 0.5) + 0.8 = [0.8, 1.3)
	delay = time.Duration(delay.Seconds() * (rand.Float64()/2 + 0.8) * float64(time.Second)) // NOTE: We want math/rand; not crypto/rand
	if delay > o.MaxRetryDelay {
		delay = o.MaxRetryDelay
	}
	return delay
}
