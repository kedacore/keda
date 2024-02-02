// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package exported

import (
	"github.com/Azure/azure-sdk-for-go/sdk/internal/log"
)

// NOTE: these are publicly exported via type-aliasing in azeventhubs/log.go
const (
	// EventConn is used whenever we create a connection or any links (ie: receivers, senders).
	EventConn log.Event = "azeh.Conn"

	// EventAuth is used when we're doing authentication/claims negotiation.
	EventAuth log.Event = "azeh.Auth"

	// EventProducer represents operations that happen on Producers.
	EventProducer log.Event = "azeh.Producer"

	// EventConsumer represents operations that happen on Consumers.
	EventConsumer log.Event = "azeh.Consumer"
)
