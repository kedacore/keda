// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package exported

// NOTE: this struct is exported via client.go:NewWebSocketConnArgs

// NewWebSocketConnArgs are the arguments to the NewWebSocketConn function you pass if you want
// to enable websockets.
type NewWebSocketConnArgs struct {
	// Host is the the `wss://<host>` to connect to
	Host string
}
