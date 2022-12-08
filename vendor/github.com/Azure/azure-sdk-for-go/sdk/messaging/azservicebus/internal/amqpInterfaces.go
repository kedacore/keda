// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package internal

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/amqpwrap"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp"
)

type AMQPReceiver = amqpwrap.AMQPReceiver
type AMQPReceiverCloser = amqpwrap.AMQPReceiverCloser
type AMQPSender = amqpwrap.AMQPSender
type AMQPSenderCloser = amqpwrap.AMQPSenderCloser

// RPCLink is implemented by *rpc.Link
type RPCLink interface {
	Close(ctx context.Context) error
	RPC(ctx context.Context, msg *amqp.Message) (*RPCResponse, error)
}

// Closeable is implemented by pretty much any AMQP link/client
// including our own higher level Receiver/Sender.
type Closeable interface {
	Close(ctx context.Context) error
}
