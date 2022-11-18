// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package exported

import "time"

// NOTE: this is exposed via type-aliasing in azservicebus/client.go

// RetryOptions represent the options for retries.
type RetryOptions struct {
	// MaxRetries specifies the maximum number of attempts a failed operation will be retried
	// before producing an error.
	// The default value is three.  A value less than zero means one try and no retries.
	MaxRetries int32

	// RetryDelay specifies the initial amount of delay to use before retrying an operation.
	// The delay increases exponentially with each retry up to the maximum specified by MaxRetryDelay.
	// The default value is four seconds.  A value less than zero means no delay between retries.
	RetryDelay time.Duration

	// MaxRetryDelay specifies the maximum delay allowed before retrying an operation.
	// Typically the value is greater than or equal to the value specified in RetryDelay.
	// The default Value is 120 seconds.  A value less than zero means there is no cap.
	MaxRetryDelay time.Duration
}
