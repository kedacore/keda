// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package amqpwrap

import (
	"context"

	"github.com/Azure/go-amqp"
)

// RPCResponse is the simplified response structure from an RPC like call
type RPCResponse struct {
	// Code is the response code - these originate from Service Bus. Some
	// common values are called out below, with the RPCResponseCode* constants.
	Code        int
	Description string
	Message     *amqp.Message
}

// RPCLink is implemented by *rpc.Link
type RPCLink interface {
	Close(ctx context.Context) error
	ConnID() uint64
	RPC(ctx context.Context, msg *amqp.Message) (*RPCResponse, error)
	LinkName() string
}
