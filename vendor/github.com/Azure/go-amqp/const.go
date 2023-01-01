package amqp

import "github.com/Azure/go-amqp/internal/encoding"

// Sender Settlement Modes
const (
	// Sender will send all deliveries initially unsettled to the receiver.
	SenderSettleModeUnsettled SenderSettleMode = encoding.SenderSettleModeUnsettled

	// Sender will send all deliveries settled to the receiver.
	SenderSettleModeSettled SenderSettleMode = encoding.SenderSettleModeSettled

	// Sender MAY send a mixture of settled and unsettled deliveries to the receiver.
	SenderSettleModeMixed SenderSettleMode = encoding.SenderSettleModeMixed
)

// SenderSettleMode specifies how the sender will settle messages.
type SenderSettleMode = encoding.SenderSettleMode

func senderSettleModeValue(m *SenderSettleMode) SenderSettleMode {
	if m == nil {
		return SenderSettleModeMixed
	}
	return *m
}

// Receiver Settlement Modes
const (
	// Receiver will spontaneously settle all incoming transfers.
	ReceiverSettleModeFirst ReceiverSettleMode = encoding.ReceiverSettleModeFirst

	// Receiver will only settle after sending the disposition to the
	// sender and receiving a disposition indicating settlement of
	// the delivery from the sender.
	ReceiverSettleModeSecond ReceiverSettleMode = encoding.ReceiverSettleModeSecond
)

// ReceiverSettleMode specifies how the receiver will settle messages.
type ReceiverSettleMode = encoding.ReceiverSettleMode

func receiverSettleModeValue(m *ReceiverSettleMode) ReceiverSettleMode {
	if m == nil {
		return ReceiverSettleModeFirst
	}
	return *m
}

// Durability Policies
const (
	// No terminus state is retained durably.
	DurabilityNone Durability = encoding.DurabilityNone

	// Only the existence and configuration of the terminus is
	// retained durably.
	DurabilityConfiguration Durability = encoding.DurabilityConfiguration

	// In addition to the existence and configuration of the
	// terminus, the unsettled state for durable messages is
	// retained durably.
	DurabilityUnsettledState Durability = encoding.DurabilityUnsettledState
)

// Durability specifies the durability of a link.
type Durability = encoding.Durability

// Expiry Policies
const (
	// The expiry timer starts when terminus is detached.
	ExpiryPolicyLinkDetach ExpiryPolicy = encoding.ExpiryLinkDetach

	// The expiry timer starts when the most recently
	// associated session is ended.
	ExpiryPolicySessionEnd ExpiryPolicy = encoding.ExpirySessionEnd

	// The expiry timer starts when most recently associated
	// connection is closed.
	ExpiryPolicyConnectionClose ExpiryPolicy = encoding.ExpiryConnectionClose

	// The terminus never expires.
	ExpiryPolicyNever ExpiryPolicy = encoding.ExpiryNever
)

// ExpiryPolicy specifies when the expiry timer of a terminus
// starts counting down from the timeout value.
//
// If the link is subsequently re-attached before the terminus is expired,
// then the count down is aborted. If the conditions for the
// terminus-expiry-policy are subsequently re-met, the expiry timer restarts
// from its originally configured timeout value.
type ExpiryPolicy = encoding.ExpiryPolicy
