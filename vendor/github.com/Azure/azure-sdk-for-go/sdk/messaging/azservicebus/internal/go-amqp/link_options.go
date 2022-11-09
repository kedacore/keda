// Copyright (C) 2017 Kale Blankenship
// Portions Copyright (c) Microsoft Corporation
package amqp

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp/internal/encoding"
)

type SenderOptions struct {
	// Capabilities is the list of extension capabilities the sender supports/desires.
	Capabilities []string

	// Durability indicates what state of the sender will be retained durably.
	//
	// Default: DurabilityNone.
	Durability Durability

	// DynamicAddress indicates a dynamic address is to be used.
	// Any specified address will be ignored.
	//
	// Default: false.
	DynamicAddress bool

	// ExpiryPolicy determines when the expiry timer of the sender starts counting
	// down from the timeout value.  If the link is subsequently re-attached before
	// the timeout is reached, the count down is aborted.
	//
	// Default: ExpirySessionEnd.
	ExpiryPolicy ExpiryPolicy

	// ExpiryTimeout is the duration in seconds that the sender will be retained.
	//
	// Default: 0.
	ExpiryTimeout uint32

	// IgnoreDispositionErrors controls automatic detach on disposition errors.
	//
	// Default: false.
	IgnoreDispositionErrors bool

	// Name sets the name of the link.
	//
	// Link names must be unique per-connection and direction.
	//
	// Default: randomly generated.
	Name string

	// Properties sets an entry in the link properties map sent to the server.
	Properties map[string]interface{}

	// RequestedReceiverSettleMode sets the requested receiver settlement mode.
	//
	// If a settlement mode is explicitly set and the server does not
	// honor it an error will be returned during link attachment.
	//
	// Default: Accept the settlement mode set by the server, commonly ModeFirst.
	RequestedReceiverSettleMode *ReceiverSettleMode

	// SettlementMode sets the settlement mode in use by this sender.
	//
	// Default: ModeMixed.
	SettlementMode *SenderSettleMode

	// SourceAddress specifies the source address for this sender.
	SourceAddress string
}

type ReceiverOptions struct {
	// LinkBatching toggles batching of message disposition.
	//
	// When enabled, accepting a message does not send the disposition
	// to the server until the batch is equal to link credit or the
	// batch max age expires.
	//
	// Default: false.
	Batching bool

	// BatchMaxAge sets the maximum time between the start
	// of a disposition batch and sending the batch to the server.
	//
	// Has no effect when Batching is false.
	//
	// Default: 5 seconds.
	BatchMaxAge time.Duration

	// Capabilities is the list of extension capabilities the receiver supports/desires.
	Capabilities []string

	// Credit specifies the maximum number of unacknowledged messages
	// the sender can transmit.
	//
	// Default: 1.
	Credit uint32

	// Durability indicates what state of the receiver will be retained durably.
	//
	// Default: DurabilityNone.
	Durability Durability

	// DynamicAddress indicates a dynamic address is to be used.
	// Any specified address will be ignored.
	//
	// Default: false.
	DynamicAddress bool

	// ExpiryPolicy determines when the expiry timer of the sender starts counting
	// down from the timeout value.  If the link is subsequently re-attached before
	// the timeout is reached, the count down is aborted.
	//
	// Default: ExpirySessionEnd.
	ExpiryPolicy ExpiryPolicy

	// ExpiryTimeout is the duration in seconds that the sender will be retained.
	//
	// Default: 0.
	ExpiryTimeout uint32

	// Filters contains the desired filters for this receiver.
	// If the peer cannot fulfill the filters the link will be detached.
	Filters []LinkFilter

	// ManualCredits enables manual credit management for this link.
	// Credits can be added with IssueCredit(), and links can also be
	// drained with DrainCredit().
	ManualCredits bool

	// MaxMessageSize sets the maximum message size that can
	// be received on the link.
	//
	// A size of zero indicates no limit.
	//
	// Default: 0.
	MaxMessageSize uint64

	// Name sets the name of the link.
	//
	// Link names must be unique per-connection and direction.
	//
	// Default: randomly generated.
	Name string

	// Properties sets an entry in the link properties map sent to the server.
	Properties map[string]interface{}

	// RequestedSenderSettleMode sets the requested sender settlement mode.
	//
	// If a settlement mode is explicitly set and the server does not
	// honor it an error will be returned during link attachment.
	//
	// Default: Accept the settlement mode set by the server, commonly ModeMixed.
	RequestedSenderSettleMode *SenderSettleMode

	// SettlementMode sets the settlement mode in use by this receiver.
	//
	// Default: ModeFirst.
	SettlementMode *ReceiverSettleMode

	// TargetAddress specifies the target address for this receiver.
	TargetAddress string
}

// LinkFilter is an advanced API for setting non-standard source filters.
// Please file an issue or open a PR if a standard filter is missing from this
// library.
//
// The name is the key for the filter map. It will be encoded as an AMQP symbol type.
//
// The code is the descriptor of the described type value. The domain-id and descriptor-id
// should be concatenated together. If 0 is passed as the code, the name will be used as
// the descriptor.
//
// The value is the value of the descriped types. Acceptable types for value are specific
// to the filter.
//
// Example:
//
// The standard selector-filter is defined as:
//
//	<descriptor name="apache.org:selector-filter:string" code="0x0000468C:0x00000004"/>
//
// In this case the name is "apache.org:selector-filter:string" and the code is
// 0x0000468C00000004.
//
//	LinkSourceFilter("apache.org:selector-filter:string", 0x0000468C00000004, exampleValue)
//
// References:
//
//	http://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-messaging-v1.0-os.html#type-filter-set
//	http://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-types-v1.0-os.html#section-descriptor-values
type LinkFilter func(encoding.Filter)

// LinkFilterSource creates or updates the named filter for this LinkFilter.
func LinkFilterSource(name string, code uint64, value interface{}) LinkFilter {
	return func(f encoding.Filter) {
		var descriptor interface{}
		if code != 0 {
			descriptor = code
		} else {
			descriptor = encoding.Symbol(name)
		}
		f[encoding.Symbol(name)] = &encoding.DescribedType{
			Descriptor: descriptor,
			Value:      value,
		}
	}
}

// LinkFilterSelector creates or updates the selector filter (apache.org:selector-filter:string) for this LinkFilter.
func LinkFilterSelector(filter string) LinkFilter {
	return LinkFilterSource(selectorFilter, selectorFilterCode, filter)
}

const (
	selectorFilter     = "apache.org:selector-filter:string"
	selectorFilterCode = uint64(0x0000468C00000004)
)
