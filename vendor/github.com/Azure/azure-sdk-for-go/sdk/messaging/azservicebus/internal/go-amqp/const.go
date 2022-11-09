// Copyright (C) 2017 Kale Blankenship
// Portions Copyright (c) Microsoft Corporation
package amqp

import "github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp/internal/encoding"

// Sender Settlement Modes
const (
	// Sender will send all deliveries initially unsettled to the receiver.
	ModeUnsettled = encoding.ModeUnsettled

	// Sender will send all deliveries settled to the receiver.
	ModeSettled = encoding.ModeSettled

	// Sender MAY send a mixture of settled and unsettled deliveries to the receiver.
	ModeMixed = encoding.ModeMixed
)

// SenderSettleMode specifies how the sender will settle messages.
type SenderSettleMode = encoding.SenderSettleMode

func senderSettleModeValue(m *SenderSettleMode) SenderSettleMode {
	if m == nil {
		return ModeMixed
	}
	return *m
}

// Receiver Settlement Modes
const (
	// Receiver will spontaneously settle all incoming transfers.
	ModeFirst = encoding.ModeFirst

	// Receiver will only settle after sending the disposition to the
	// sender and receiving a disposition indicating settlement of
	// the delivery from the sender.
	ModeSecond = encoding.ModeSecond
)

// ReceiverSettleMode specifies how the receiver will settle messages.
type ReceiverSettleMode = encoding.ReceiverSettleMode

func receiverSettleModeValue(m *ReceiverSettleMode) ReceiverSettleMode {
	if m == nil {
		return ModeFirst
	}
	return *m
}

// Durability Policies
const (
	// No terminus state is retained durably.
	DurabilityNone = encoding.DurabilityNone

	// Only the existence and configuration of the terminus is
	// retained durably.
	DurabilityConfiguration = encoding.DurabilityConfiguration

	// In addition to the existence and configuration of the
	// terminus, the unsettled state for durable messages is
	// retained durably.
	DurabilityUnsettledState = encoding.DurabilityUnsettledState
)

// Durability specifies the durability of a link.
type Durability = encoding.Durability

// Expiry Policies
const (
	// The expiry timer starts when terminus is detached.
	ExpiryLinkDetach = encoding.ExpiryLinkDetach

	// The expiry timer starts when the most recently
	// associated session is ended.
	ExpirySessionEnd = encoding.ExpirySessionEnd

	// The expiry timer starts when most recently associated
	// connection is closed.
	ExpiryConnectionClose = encoding.ExpiryConnectionClose

	// The terminus never expires.
	ExpiryNever = encoding.ExpiryNever
)

// ExpiryPolicy specifies when the expiry timer of a terminus
// starts counting down from the timeout value.
//
// If the link is subsequently re-attached before the terminus is expired,
// then the count down is aborted. If the conditions for the
// terminus-expiry-policy are subsequently re-met, the expiry timer restarts
// from its originally configured timeout value.
type ExpiryPolicy = encoding.ExpiryPolicy
