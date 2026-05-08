//go:build go1.16

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

// Package azeventhubs provides clients for sending events and consuming events.
//
// For sending events, use the [ProducerClient].
//
// There are two clients for consuming events:
//   - [Processor], which handles checkpointing and load balancing using durable storage.
//   - [ConsumerClient], which is fully manual, but provides full control.

package azeventhubs
