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

package internal

// All code in this file is private to the package.

import (
	"context"
	"time"

	"go.temporal.io/sdk/internal/common/backoff"
)

const (
	retryServiceOperationInitialInterval    = 200 * time.Millisecond
	retryServiceOperationExpirationInterval = 60 * time.Second
	retryServiceOperationBackoff            = 2
)

// Creates a retry policy which allows appropriate retries for the deadline passed in as context.
// It uses the context deadline to set MaxInterval as 1/10th of context timeout
// MaxInterval = Max(context_timeout/10, 20ms)
// defaults to ExpirationInterval of 60 seconds, or uses context deadline as expiration interval
func createDynamicServiceRetryPolicy(ctx context.Context) backoff.RetryPolicy {
	timeout := retryServiceOperationExpirationInterval
	if ctx != nil {
		now := time.Now()
		if expiration, ok := ctx.Deadline(); ok && expiration.After(now) {
			timeout = expiration.Sub(now)
		}
	}
	initialInterval := retryServiceOperationInitialInterval
	maximumInterval := timeout / 10
	if maximumInterval < retryServiceOperationInitialInterval {
		maximumInterval = retryServiceOperationInitialInterval
	}

	policy := backoff.NewExponentialRetryPolicy(initialInterval)
	policy.SetBackoffCoefficient(retryServiceOperationBackoff)
	policy.SetMaximumInterval(maximumInterval)
	policy.SetExpirationInterval(timeout)
	return policy
}
