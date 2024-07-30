// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package exported

import "fmt"

// ErrorCode is an error code, usable by consuming code to work with
// programatically.
type ErrorCode string

const (
	// ErrorCodeUnauthorizedAccess means the credentials provided are not valid for use with
	// a particular entity, or have expired.
	ErrorCodeUnauthorizedAccess ErrorCode = "unauthorized"

	// ErrorCodeConnectionLost means our connection was lost and all retry attempts failed.
	// This typically reflects an extended outage or connection disruption and may
	// require manual intervention.
	ErrorCodeConnectionLost ErrorCode = "connlost"

	// ErrorCodeOwnershipLost means that a partition that you were reading from was opened
	// by another link with an epoch/owner level greater or equal to your [PartitionClient].
	//
	// When using types like the [Processor], partition ownership will change as instances
	// rebalance.
	ErrorCodeOwnershipLost ErrorCode = "ownershiplost"
)

// Error represents an Event Hub specific error.
// NOTE: the Code is considered part of the published API but the message that
// comes back from Error(), as well as the underlying wrapped error, are NOT and
// are subject to change.
type Error struct {
	// Code is a stable error code which can be used as part of programatic error handling.
	// The codes can expand in the future, but the values (and their meaning) will remain the same.
	Code     ErrorCode
	innerErr error
}

// Error is an error message containing the code and a user friendly message, if any.
func (e *Error) Error() string {
	msg := "unknown error"
	if e.innerErr != nil {
		msg = e.innerErr.Error()
	}
	return fmt.Sprintf("(%s): %s", e.Code, msg)
}

// NewError creates a new `Error` instance.
// NOTE: this function is only exported so it can be used by the `internal`
// package. It is not available for customers.
func NewError(code ErrorCode, innerErr error) error {
	return &Error{
		Code:     code,
		innerErr: innerErr,
	}
}
