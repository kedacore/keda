// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package internal

import "net/http"

// Response codes that come back over the "RPC" style links like cbs or management.
const (
	// RPCResponseCodeLockLost comes back if you lose a message lock _or_ a session lock.
	// (NOTE: this is the one HTTP code that doesn't make intuitive sense. For all others I've just
	// used the HTTP status code instead.
	RPCResponseCodeLockLost = http.StatusGone
)
