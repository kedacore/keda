// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package backoff

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.temporal.io/api/serviceerror"
)

type (
	// Operation to retry
	Operation func() error

	// IsRetryable handler can be used to exclude certain errors during retry
	IsRetryable func(error) bool

	// ConcurrentRetrier is used for client-side throttling. It determines whether to
	// throttle outgoing traffic in case downstream backend server rejects
	// requests due to out-of-quota or server busy errors.
	ConcurrentRetrier struct {
		sync.Mutex
		retrier                 Retrier // Backoff retrier
		secondaryRetrier        Retrier
		failureCount            int64 // Number of consecutive failures seen
		includeSecondaryRetrier bool
	}
)

// Throttle Sleep if there were failures since the last success call. The
// provided done channel provides a way to exit early.
func (c *ConcurrentRetrier) Throttle(doneCh <-chan struct{}) {
	c.throttleInternal(doneCh)
}

// GetElapsedTime gets the amount of time since that last ConcurrentRetrier.Succeeded call
func (c *ConcurrentRetrier) GetElapsedTime() time.Duration {
	c.Lock()
	defer c.Unlock()
	return c.retrier.GetElapsedTime()
}

func (c *ConcurrentRetrier) throttleInternal(doneCh <-chan struct{}) time.Duration {
	next := done

	// Check if we have failure count.
	c.Lock()
	if c.failureCount > 0 {
		next = c.retrier.NextBackOff()
		// If secondary is included, use the greatest of the two (which also means
		// if one is "done", which is -1, the one that's not done is chosen)
		if c.includeSecondaryRetrier {
			c.includeSecondaryRetrier = false
			if c.secondaryRetrier != nil {
				if secNext := c.secondaryRetrier.NextBackOff(); secNext > next {
					next = secNext
				}
			}
		}
	}
	c.Unlock()

	if next != done {
		select {
		case <-doneCh:
		case <-time.After(next):
		}
	}

	return next
}

// Succeeded marks client request succeeded.
func (c *ConcurrentRetrier) Succeeded() {
	defer c.Unlock()
	c.Lock()
	c.failureCount = 0
	c.includeSecondaryRetrier = false
	c.retrier.Reset()
	if c.secondaryRetrier != nil {
		c.secondaryRetrier.Reset()
	}
}

// Failed marks client request failed because backend is busy. If
// includeSecondaryRetryPolicy is true, see SetSecondaryRetryPolicy for effects.
func (c *ConcurrentRetrier) Failed(includeSecondaryRetryPolicy bool) {
	defer c.Unlock()
	c.Lock()
	c.failureCount++
	c.includeSecondaryRetrier = includeSecondaryRetryPolicy
}

// SetSecondaryRetryPolicy sets a secondary retry policy that, if Failed is
// called with true, will trigger the secondary retry policy in addition to the
// primary and will use the result of the secondary if longer than the primary.
func (c *ConcurrentRetrier) SetSecondaryRetryPolicy(retryPolicy RetryPolicy) {
	c.Lock()
	defer c.Unlock()
	if retryPolicy == nil {
		c.secondaryRetrier = nil
	} else {
		c.secondaryRetrier = NewRetrier(retryPolicy, SystemClock)
	}
}

// NewConcurrentRetrier returns an instance of concurrent backoff retrier.
func NewConcurrentRetrier(retryPolicy RetryPolicy) *ConcurrentRetrier {
	retrier := NewRetrier(retryPolicy, SystemClock)
	return &ConcurrentRetrier{retrier: retrier}
}

// Retry function can be used to wrap any call with retry logic using the passed in policy
func Retry(ctx context.Context, operation Operation, policy RetryPolicy, isRetryable IsRetryable) error {
	var lastErr error
	var next time.Duration

	r := NewRetrier(policy, SystemClock)
	for {
		opErr := operation()
		if opErr == nil {
			// operation completed successfully. No need to retry.
			return nil
		}

		// Usually, after number of retry attempts, last attempt fails with DeadlineExceeded error.
		// It is not informative and actual error reason is in the error occurred on previous attempt.
		// Therefore, update lastErr only if it is not set (first attempt) or opErr is not a DeadlineExceeded error.
		// This lastErr is returned if retry attempts are exhausted.
		var errDeadlineExceeded *serviceerror.DeadlineExceeded
		if lastErr == nil || !(errors.Is(opErr, context.DeadlineExceeded) || errors.As(opErr, &errDeadlineExceeded)) {
			lastErr = opErr
		}

		if next = r.NextBackOff(); next == done {
			return lastErr
		}

		// Check if the error is retryable
		if isRetryable != nil && !isRetryable(opErr) {
			return lastErr
		}

		// check if ctx is done
		if ctxDone := ctx.Done(); ctxDone != nil {
			timer := time.NewTimer(next)
			select {
			case <-ctxDone:
				return lastErr
			case <-timer.C:
				continue
			}
		}

		// ctx is not cancellable
		time.Sleep(next)
	}
}

// IgnoreErrors can be used as IsRetryable handler for Retry function to exclude certain errors from the retry list
func IgnoreErrors(errorsToExclude []error) func(error) bool {
	return func(err error) bool {
		for _, errorToExclude := range errorsToExclude {
			if errors.Is(err, errorToExclude) {
				return false
			}
		}

		return true
	}
}
