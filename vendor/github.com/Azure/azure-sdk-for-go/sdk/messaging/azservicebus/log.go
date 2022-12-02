// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azservicebus

import "github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/exported"

const (
	// EventConn is used whenever we create a connection or any links (ie: receivers, senders).
	EventConn = exported.EventConn

	// EventAuth is used when we're doing authentication/claims negotiation.
	EventAuth = exported.EventAuth

	// EventReceiver represents operations that happen on Receivers.
	EventReceiver = exported.EventReceiver

	// EventSender represents operations that happen on Senders.
	EventSender = exported.EventSender

	// EventAdmin is used for operations in the azservicebus/admin.Client
	EventAdmin = exported.EventAdmin
)
