package amqp

import (
	"errors"
	"fmt"
	"time"

	"github.com/Azure/go-amqp/internal/encoding"
	"github.com/Azure/go-amqp/internal/frames"
)

// LinkOption is a function for configuring an AMQP link.
//
// A link may be a Sender or a Receiver.
type LinkOption func(*link) error

// LinkProperty sets an entry in the link properties map sent to the server.
//
// This option can be used multiple times.
func LinkProperty(key, value string) LinkOption {
	return linkProperty(key, value)
}

// LinkPropertyInt64 sets an entry in the link properties map sent to the server.
//
// This option can be used multiple times.
func LinkPropertyInt64(key string, value int64) LinkOption {
	return linkProperty(key, value)
}

// LinkPropertyInt32 sets an entry in the link properties map sent to the server.
//
// This option can be set multiple times.
func LinkPropertyInt32(key string, value int32) LinkOption {
	return linkProperty(key, value)
}

func linkProperty(key string, value interface{}) LinkOption {
	return func(l *link) error {
		if key == "" {
			return errors.New("link property key must not be empty")
		}
		if l.properties == nil {
			l.properties = make(map[encoding.Symbol]interface{})
		}
		l.properties[encoding.Symbol(key)] = value
		return nil
	}
}

// LinkName sets the name of the link.
//
// The link names must be unique per-connection and direction.
//
// Default: randomly generated.
func LinkName(name string) LinkOption {
	return func(l *link) error {
		l.Key.name = name
		return nil
	}
}

// LinkSourceCapabilities sets the source capabilities.
func LinkSourceCapabilities(capabilities ...string) LinkOption {
	return func(l *link) error {
		if l.Source == nil {
			l.Source = new(frames.Source)
		}

		// Convert string to symbol
		symbolCapabilities := make([]encoding.Symbol, len(capabilities))
		for i, v := range capabilities {
			symbolCapabilities[i] = encoding.Symbol(v)
		}

		l.Source.Capabilities = append(l.Source.Capabilities, symbolCapabilities...)
		return nil
	}
}

// LinkSourceAddress sets the source address.
func LinkSourceAddress(addr string) LinkOption {
	return func(l *link) error {
		if l.Source == nil {
			l.Source = new(frames.Source)
		}
		l.Source.Address = addr
		return nil
	}
}

// LinkTargetAddress sets the target address.
func LinkTargetAddress(addr string) LinkOption {
	return func(l *link) error {
		if l.Target == nil {
			l.Target = new(frames.Target)
		}
		l.Target.Address = addr
		return nil
	}
}

// LinkAddressDynamic requests a dynamically created address from the server.
func LinkAddressDynamic() LinkOption {
	return func(l *link) error {
		l.dynamicAddr = true
		return nil
	}
}

// LinkCredit specifies the maximum number of unacknowledged messages
// the sender can transmit.
func LinkCredit(credit uint32) LinkOption {
	return func(l *link) error {
		if l.receiver == nil {
			return errors.New("LinkCredit is not valid for Sender")
		}

		l.receiver.maxCredit = credit
		return nil
	}
}

// LinkWithManualCredits enables manual credit management for this link.
// Credits can be added with IssueCredit(), and links can also be drained
// with DrainCredit().
func LinkWithManualCredits() LinkOption {
	return func(l *link) error {
		if l.receiver == nil {
			return errors.New("LinkWithManualCredits is not valid for Sender")
		}

		l.receiver.manualCreditor = &manualCreditor{}
		return nil
	}
}

// LinkBatching toggles batching of message disposition.
//
// When enabled, accepting a message does not send the disposition
// to the server until the batch is equal to link credit or the
// batch max age expires.
func LinkBatching(enable bool) LinkOption {
	return func(l *link) error {
		l.receiver.batching = enable
		return nil
	}
}

// LinkBatchMaxAge sets the maximum time between the start
// of a disposition batch and sending the batch to the server.
func LinkBatchMaxAge(d time.Duration) LinkOption {
	return func(l *link) error {
		l.receiver.batchMaxAge = d
		return nil
	}
}

// LinkSenderSettle sets the requested sender settlement mode.
//
// If a settlement mode is explicitly set and the server does not
// honor it an error will be returned during link attachment.
//
// Default: Accept the settlement mode set by the server, commonly ModeMixed.
func LinkSenderSettle(mode SenderSettleMode) LinkOption {
	return func(l *link) error {
		if mode > ModeMixed {
			return fmt.Errorf("invalid SenderSettlementMode %d", mode)
		}
		l.SenderSettleMode = &mode
		return nil
	}
}

// LinkReceiverSettle sets the requested receiver settlement mode.
//
// If a settlement mode is explicitly set and the server does not
// honor it an error will be returned during link attachment.
//
// Default: Accept the settlement mode set by the server, commonly ModeFirst.
func LinkReceiverSettle(mode ReceiverSettleMode) LinkOption {
	return func(l *link) error {
		if mode > ModeSecond {
			return fmt.Errorf("invalid ReceiverSettlementMode %d", mode)
		}
		l.ReceiverSettleMode = &mode
		return nil
	}
}

// LinkSelectorFilter sets a selector filter (apache.org:selector-filter:string) on the link source.
func LinkSelectorFilter(filter string) LinkOption {
	// <descriptor name="apache.org:selector-filter:string" code="0x0000468C:0x00000004"/>
	return LinkSourceFilter("apache.org:selector-filter:string", 0x0000468C00000004, filter)
}

// LinkSourceFilter is an advanced API for setting non-standard source filters.
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
//  <descriptor name="apache.org:selector-filter:string" code="0x0000468C:0x00000004"/>
// In this case the name is "apache.org:selector-filter:string" and the code is
// 0x0000468C00000004.
//  LinkSourceFilter("apache.org:selector-filter:string", 0x0000468C00000004, exampleValue)
//
// References:
//  http://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-messaging-v1.0-os.html#type-filter-set
//  http://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-types-v1.0-os.html#section-descriptor-values
func LinkSourceFilter(name string, code uint64, value interface{}) LinkOption {
	return func(l *link) error {
		if l.Source == nil {
			l.Source = new(frames.Source)
		}
		if l.Source.Filter == nil {
			l.Source.Filter = make(map[encoding.Symbol]*encoding.DescribedType)
		}

		var descriptor interface{}
		if code != 0 {
			descriptor = code
		} else {
			descriptor = encoding.Symbol(name)
		}

		l.Source.Filter[encoding.Symbol(name)] = &encoding.DescribedType{
			Descriptor: descriptor,
			Value:      value,
		}
		return nil
	}
}

// LinkMaxMessageSize sets the maximum message size that can
// be sent or received on the link.
//
// A size of zero indicates no limit.
//
// Default: 0.
func LinkMaxMessageSize(size uint64) LinkOption {
	return func(l *link) error {
		l.MaxMessageSize = size
		return nil
	}
}

// LinkTargetDurability sets the target durability policy.
//
// Default: DurabilityNone.
func LinkTargetDurability(d Durability) LinkOption {
	return func(l *link) error {
		if d > DurabilityUnsettledState {
			return fmt.Errorf("invalid Durability %d", d)
		}

		if l.Target == nil {
			l.Target = new(frames.Target)
		}
		l.Target.Durable = d

		return nil
	}
}

// LinkTargetExpiryPolicy sets the link expiration policy.
//
// Default: ExpirySessionEnd.
func LinkTargetExpiryPolicy(p ExpiryPolicy) LinkOption {
	return func(l *link) error {
		err := encoding.ValidateExpiryPolicy(p)
		if err != nil {
			return err
		}

		if l.Target == nil {
			l.Target = new(frames.Target)
		}
		l.Target.ExpiryPolicy = p

		return nil
	}
}

// LinkTargetTimeout sets the duration that an expiring target will be retained.
//
// Default: 0.
func LinkTargetTimeout(timeout uint32) LinkOption {
	return func(l *link) error {
		if l.Target == nil {
			l.Target = new(frames.Target)
		}
		l.Target.Timeout = timeout

		return nil
	}
}

// LinkSourceDurability sets the source durability policy.
//
// Default: DurabilityNone.
func LinkSourceDurability(d Durability) LinkOption {
	return func(l *link) error {
		if d > DurabilityUnsettledState {
			return fmt.Errorf("invalid Durability %d", d)
		}

		if l.Source == nil {
			l.Source = new(frames.Source)
		}
		l.Source.Durable = d

		return nil
	}
}

// LinkSourceExpiryPolicy sets the link expiration policy.
//
// Default: ExpirySessionEnd.
func LinkSourceExpiryPolicy(p ExpiryPolicy) LinkOption {
	return func(l *link) error {
		err := encoding.ValidateExpiryPolicy(p)
		if err != nil {
			return err
		}

		if l.Source == nil {
			l.Source = new(frames.Source)
		}
		l.Source.ExpiryPolicy = p

		return nil
	}
}

// LinkSourceTimeout sets the duration that an expiring source will be retained.
//
// Default: 0.
func LinkSourceTimeout(timeout uint32) LinkOption {
	return func(l *link) error {
		if l.Source == nil {
			l.Source = new(frames.Source)
		}
		l.Source.Timeout = timeout

		return nil
	}
}

// LinkDetachOnDispositionError controls whether you detach on disposition
// errors (subject to some simple logic) or do NOT detach at all on disposition
// errors.
// Defaults to true.
func LinkDetachOnDispositionError(detachOnDispositionError bool) LinkOption {
	return func(l *link) error {
		l.detachOnDispositionError = detachOnDispositionError
		return nil
	}
}
