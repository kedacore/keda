// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package eh

import (
	"errors"

	"github.com/Azure/go-amqp"
)

// ErrCondGeoReplicationOffset occurs when using an old integer offset against a hub that has
// geo-replication enabled, which requires the new stroffset format.
const ErrCondGeoReplicationOffset = amqp.ErrCond("com.microsoft:georeplication:invalid-offset")

// IsGeoReplicationOffsetError checks if we've received a "bad offset" error from Event Hubs.
// This should only happpen if:
//
//	a. You're working with an Event Hub namespace that has geo-replication enabled...
//	b. ...and pass in an older style offset rather than the newer "stroffset" style (contains broker and partition information).
func IsGeoReplicationOffsetError(err error) bool {
	if amqpErr := (*amqp.Error)(nil); errors.As(err, &amqpErr) {
		if amqpErr.Condition == ErrCondGeoReplicationOffset {
			return true
		}
	}
	return false
}
