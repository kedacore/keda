// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package amqpwrap

import "context"

// Closeable is implemented by pretty much any AMQP link/client
// including our own higher level Receiver/Sender.
type Closeable interface {
	Close(ctx context.Context) error
}
