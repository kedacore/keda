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
	"math"
	"math/rand"
	"time"

	"go.temporal.io/sdk/internal/common/retry"
)

const (
	done time.Duration = -1
)

type (
	// RetryPolicy is the API which needs to be implemented by various retry policy implementations
	RetryPolicy interface {
		ComputeNextDelay(elapsedTime time.Duration, attempt int) time.Duration
		GrpcRetryConfig() *retry.GrpcRetryConfig
	}

	// Retrier manages the state of retry operation
	Retrier interface {
		GetElapsedTime() time.Duration
		NextBackOff() time.Duration
		Reset()
	}

	// Clock used by ExponentialRetryPolicy implementation to get the current time.  Mainly used for unit testing
	Clock interface {
		Now() time.Time
	}

	// ExponentialRetryPolicy provides the implementation for retry policy using a coefficient to compute the next delay.
	// Formula used to compute the next delay is: initialInterval * math.Pow(backoffCoefficient, currentAttempt)
	ExponentialRetryPolicy struct {
		initialInterval    time.Duration
		backoffCoefficient float64
		maximumInterval    time.Duration
		expirationInterval time.Duration
		maximumAttempts    int
	}

	systemClock struct{}

	retrierImpl struct {
		policy         RetryPolicy
		clock          Clock
		currentAttempt int
		startTime      time.Time
	}
)

// SystemClock implements Clock interface that uses time.Now().
var SystemClock = systemClock{}

// NewExponentialRetryPolicy returns an instance of ExponentialRetryPolicy using the provided initialInterval
func NewExponentialRetryPolicy(initialInterval time.Duration) *ExponentialRetryPolicy {
	p := &ExponentialRetryPolicy{
		initialInterval:    initialInterval,
		backoffCoefficient: retry.DefaultBackoffCoefficient,
		maximumInterval:    retry.DefaultMaximumInterval,
		expirationInterval: retry.DefaultExpirationInterval,
		maximumAttempts:    retry.DefaultMaximumAttempts,
	}

	return p
}

// NewRetrier is used for creating a new instance of Retrier
func NewRetrier(policy RetryPolicy, clock Clock) Retrier {
	return &retrierImpl{
		policy:         policy,
		clock:          clock,
		startTime:      clock.Now(),
		currentAttempt: 1,
	}
}

// SetInitialInterval sets the initial interval used by ExponentialRetryPolicy for the very first retry
// All later retries are computed using the following formula:
// initialInterval * math.Pow(backoffCoefficient, currentAttempt)
func (p *ExponentialRetryPolicy) SetInitialInterval(initialInterval time.Duration) {
	p.initialInterval = initialInterval
}

// SetBackoffCoefficient sets the coefficient used by ExponentialRetryPolicy to compute next delay for each retry
// All retries are computed using the following formula:
// initialInterval * math.Pow(backoffCoefficient, currentAttempt)
func (p *ExponentialRetryPolicy) SetBackoffCoefficient(backoffCoefficient float64) {
	p.backoffCoefficient = backoffCoefficient
}

// SetMaximumInterval sets the maximum interval for each retry
func (p *ExponentialRetryPolicy) SetMaximumInterval(maximumInterval time.Duration) {
	p.maximumInterval = maximumInterval
}

// SetExpirationInterval sets the absolute expiration interval for all retries
func (p *ExponentialRetryPolicy) SetExpirationInterval(expirationInterval time.Duration) {
	p.expirationInterval = expirationInterval
}

// SetMaximumAttempts sets the maximum number of retry attempts
func (p *ExponentialRetryPolicy) SetMaximumAttempts(maximumAttempts int) {
	p.maximumAttempts = maximumAttempts
}

// ComputeNextDelay returns the next delay interval.  This is used by Retrier to delay calling the operation again
func (p *ExponentialRetryPolicy) ComputeNextDelay(elapsedTime time.Duration, attempt int) time.Duration {
	// Check to see if we ran out of maximum number of attempts
	if p.maximumAttempts != retry.UnlimitedMaximumAttempts && attempt >= p.maximumAttempts {
		return done
	}

	// Stop retrying after expiration interval is elapsed
	if p.expirationInterval != retry.UnlimitedInterval && elapsedTime > p.expirationInterval {
		return done
	}

	nextInterval := float64(p.initialInterval) * math.Pow(p.backoffCoefficient, float64(attempt-1))
	// Disallow retries if initialInterval is negative or nextInterval overflows
	if nextInterval <= 0 {
		return done
	}
	if p.maximumInterval != retry.UnlimitedInterval {
		nextInterval = math.Min(nextInterval, float64(p.maximumInterval))
	}

	if p.expirationInterval != retry.UnlimitedInterval {
		remainingTime := math.Max(0, float64(p.expirationInterval-elapsedTime))
		nextInterval = math.Min(remainingTime, nextInterval)
	}

	// Bail out if the next interval is smaller than initial retry interval
	nextDuration := time.Duration(nextInterval)
	if nextDuration < p.initialInterval {
		return done
	}

	// add jitter to avoid global synchronization
	jitterPortion := int(retry.DefaultJitter * nextInterval)
	// Prevent overflow
	if jitterPortion < 1 {
		jitterPortion = 1
	}
	nextInterval = nextInterval*(1-retry.DefaultJitter) + float64(rand.Intn(jitterPortion))

	return time.Duration(nextInterval)
}

// GrpcRetryConfig converts retry policy into retry config.
func (p *ExponentialRetryPolicy) GrpcRetryConfig() *retry.GrpcRetryConfig {
	retryConfig := retry.NewGrpcRetryConfig(p.initialInterval)
	retryConfig.SetBackoffCoefficient(p.backoffCoefficient)
	retryConfig.SetExpirationInterval(p.expirationInterval)
	retryConfig.SetMaximumAttempts(p.maximumAttempts)
	retryConfig.SetMaximumInterval(p.maximumInterval)
	return retryConfig
}

// Now returns the current time using the system clock
func (t systemClock) Now() time.Time {
	return time.Now()
}

// Reset will set the Retrier into initial state
func (r *retrierImpl) Reset() {
	r.startTime = r.clock.Now()
	r.currentAttempt = 1
}

// NextBackOff returns the next delay interval.  This is used by Retry to delay calling the operation again
func (r *retrierImpl) NextBackOff() time.Duration {
	nextInterval := r.policy.ComputeNextDelay(r.GetElapsedTime(), r.currentAttempt)

	// Now increment the current attempt
	r.currentAttempt++
	return nextInterval
}

// GetElapsedTime returns the amount of time since the retrier was created or the last reset,
// whatever was sooner.
func (r *retrierImpl) GetElapsedTime() time.Duration {
	return r.clock.Now().Sub(r.startTime)
}
