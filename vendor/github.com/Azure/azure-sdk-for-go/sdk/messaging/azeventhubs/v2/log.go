// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azeventhubs

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/log"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/exported"
)

const (
	// EventConn is used whenever we create a connection or any links (ie: producers, consumers).
	EventConn log.Event = exported.EventConn

	// EventAuth is used when we're doing authentication/claims negotiation.
	EventAuth log.Event = exported.EventAuth

	// EventProducer represents operations that happen on Producers.
	EventProducer log.Event = exported.EventProducer

	// EventConsumer represents operations that happen on Consumers.
	EventConsumer log.Event = exported.EventConsumer
)
