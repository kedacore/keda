// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azeventhubs

import "github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/exported"

// Error represents an Event Hub specific error.
// NOTE: the Code is considered part of the published API but the message that
// comes back from Error(), as well as the underlying wrapped error, are NOT and
// are subject to change.
type Error = exported.Error

// ErrorCode is an error code, usable by consuming code to work with
// programatically.
type ErrorCode = exported.ErrorCode

const (
	// ErrorCodeUnauthorizedAccess means the credentials provided are not valid for use with
	// a particular entity, or have expired.
	ErrorCodeUnauthorizedAccess ErrorCode = exported.ErrorCodeUnauthorizedAccess

	// ErrorCodeConnectionLost means our connection was lost and all retry attempts failed.
	// This typically reflects an extended outage or connection disruption and may
	// require manual intervention.
	ErrorCodeConnectionLost ErrorCode = exported.ErrorCodeConnectionLost

	// ErrorCodeOwnershipLost means that a partition that you were reading from was opened
	// by another link with a higher epoch/owner level.
	ErrorCodeOwnershipLost ErrorCode = exported.ErrorCodeOwnershipLost
)
