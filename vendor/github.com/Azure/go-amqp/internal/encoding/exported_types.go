package encoding

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"

	"github.com/Azure/go-amqp/internal/buffer"
)

// Durability Policies
const (
	// No terminus state is retained durably.
	DurabilityNone Durability = 0

	// Only the existence and configuration of the terminus is
	// retained durably.
	DurabilityConfiguration Durability = 1

	// In addition to the existence and configuration of the
	// terminus, the unsettled state for durable messages is
	// retained durably.
	DurabilityUnsettledState Durability = 2
)

// Durability specifies the durability of a link.
type Durability uint32

// String implements the [fmt.Stringer] interface.
// Note that the values are for diagnostic purposes and may change over time.
func (d *Durability) String() string {
	if d == nil {
		return "<nil>"
	}

	switch *d {
	case DurabilityNone:
		return "none"
	case DurabilityConfiguration:
		return "configuration"
	case DurabilityUnsettledState:
		return "unsettled-state"
	default:
		return fmt.Sprintf("unknown durability %d", *d)
	}
}

// Marshal encodes this type into a buffer. It is not intended for public use.
func (d Durability) Marshal(wr *buffer.Buffer) error {
	return Marshal(wr, uint32(d))
}

// Unmarshal decodes a buffer into this type. It is not intended for public use.
func (d *Durability) Unmarshal(r *buffer.Buffer) error {
	return Unmarshal(r, (*uint32)(d))
}

// Expiry Policies
const (
	// The expiry timer starts when terminus is detached.
	ExpiryLinkDetach ExpiryPolicy = "link-detach"

	// The expiry timer starts when the most recently
	// associated session is ended.
	ExpirySessionEnd ExpiryPolicy = "session-end"

	// The expiry timer starts when most recently associated
	// connection is closed.
	ExpiryConnectionClose ExpiryPolicy = "connection-close"

	// The terminus never expires.
	ExpiryNever ExpiryPolicy = "never"
)

// ExpiryPolicy specifies when the expiry timer of a terminus
// starts counting down from the timeout value.
//
// If the link is subsequently re-attached before the terminus is expired,
// then the count down is aborted. If the conditions for the
// terminus-expiry-policy are subsequently re-met, the expiry timer restarts
// from its originally configured timeout value.
type ExpiryPolicy Symbol

// Marshal encodes this type into a buffer. It is not intended for public use.
func (e ExpiryPolicy) Marshal(wr *buffer.Buffer) error {
	return Symbol(e).Marshal(wr)
}

// Unmarshal decodes a buffer into this type. It is not intended for public use.
func (e *ExpiryPolicy) Unmarshal(r *buffer.Buffer) error {
	err := Unmarshal(r, (*Symbol)(e))
	if err != nil {
		return err
	}
	return ValidateExpiryPolicy(*e)
}

// String implements the [fmt.Stringer] interface.
// Note that the values are for diagnostic purposes and may change over time.
func (e *ExpiryPolicy) String() string {
	if e == nil {
		return "<nil>"
	}
	return string(*e)
}

// Sender Settlement Modes
const (
	// Sender will send all deliveries initially unsettled to the receiver.
	SenderSettleModeUnsettled SenderSettleMode = 0

	// Sender will send all deliveries settled to the receiver.
	SenderSettleModeSettled SenderSettleMode = 1

	// Sender MAY send a mixture of settled and unsettled deliveries to the receiver.
	SenderSettleModeMixed SenderSettleMode = 2
)

// SenderSettleMode specifies how the sender will settle messages.
type SenderSettleMode uint8

// Ptr returns a pointer to the value of m.
func (m SenderSettleMode) Ptr() *SenderSettleMode {
	return &m
}

// String implements the [fmt.Stringer] interface.
// Note that the values are for diagnostic purposes and may change over time.
func (m *SenderSettleMode) String() string {
	if m == nil {
		return "<nil>"
	}

	switch *m {
	case SenderSettleModeUnsettled:
		return "unsettled"

	case SenderSettleModeSettled:
		return "settled"

	case SenderSettleModeMixed:
		return "mixed"

	default:
		return fmt.Sprintf("unknown sender mode %d", uint8(*m))
	}
}

// Marshal encodes this type into a buffer. It is not intended for public use.
func (m SenderSettleMode) Marshal(wr *buffer.Buffer) error {
	return Marshal(wr, uint8(m))
}

// Unmarshal decodes a buffer into this type. It is not intended for public use.
func (m *SenderSettleMode) Unmarshal(r *buffer.Buffer) error {
	n, err := ReadUbyte(r)
	*m = SenderSettleMode(n)
	return err
}

// Receiver Settlement Modes
const (
	// Receiver will spontaneously settle all incoming transfers.
	ReceiverSettleModeFirst ReceiverSettleMode = 0

	// Receiver will only settle after sending the disposition to the
	// sender and receiving a disposition indicating settlement of
	// the delivery from the sender.
	ReceiverSettleModeSecond ReceiverSettleMode = 1
)

// ReceiverSettleMode specifies how the receiver will settle messages.
type ReceiverSettleMode uint8

// Ptr returns a pointer to the value of m.
func (m ReceiverSettleMode) Ptr() *ReceiverSettleMode {
	return &m
}

// String implements the [fmt.Stringer] interface.
// Note that the values are for diagnostic purposes and may change over time.
func (m *ReceiverSettleMode) String() string {
	if m == nil {
		return "<nil>"
	}

	switch *m {
	case ReceiverSettleModeFirst:
		return "first"

	case ReceiverSettleModeSecond:
		return "second"

	default:
		return fmt.Sprintf("unknown receiver mode %d", uint8(*m))
	}
}

// Marshal encodes this type into a buffer. It is not intended for public use.
func (m ReceiverSettleMode) Marshal(wr *buffer.Buffer) error {
	return Marshal(wr, uint8(m))
}

// Unmarshal decodes a buffer into this type. It is not intended for public use.
func (m *ReceiverSettleMode) Unmarshal(r *buffer.Buffer) error {
	n, err := ReadUbyte(r)
	*m = ReceiverSettleMode(n)
	return err
}

// Filter is a set of named filters.
// http://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-messaging-v1.0-os.html#type-filter-set
type Filter map[Symbol]*DescribedType

// Marshal encodes this type into a buffer. It is not intended for public use.
func (f Filter) Marshal(wr *buffer.Buffer) error {
	return writeMap(wr, f)
}

// Unmarshal decodes a buffer into this type. It is not intended for public use.
func (f *Filter) Unmarshal(r *buffer.Buffer) error {
	count, err := readMapHeader(r)
	if err != nil {
		return err
	}

	m := make(Filter, count/2)
	for i := uint32(0); i < count; i += 2 {
		key, err := ReadString(r)
		if err != nil {
			return err
		}
		var value DescribedType
		err = Unmarshal(r, &value)
		if err != nil {
			return err
		}
		m[Symbol(key)] = &value
	}
	*f = m
	return nil
}

// Annotations keys must be of type string, int, or int64.
//
// String keys are encoded as AMQP Symbols.
type Annotations map[any]any

// Marshal encodes this type into a buffer. It is not intended for public use.
func (a Annotations) Marshal(wr *buffer.Buffer) error {
	return writeMap(wr, a)
}

// Unmarshal decodes a buffer into this type. It is not intended for public use.
func (a *Annotations) Unmarshal(r *buffer.Buffer) error {
	count, err := readMapHeader(r)
	if err != nil {
		return err
	}

	m := make(Annotations, count/2)
	for i := uint32(0); i < count; i += 2 {
		key, err := ReadAny(r)
		if err != nil {
			return err
		}
		value, err := ReadAny(r)
		if err != nil {
			return err
		}
		m[key] = value
	}
	*a = m
	return nil
}

// ErrCond is one of the error conditions defined in the AMQP spec.
type ErrCond string

// Marshal encodes this type into a buffer. It is not intended for public use.
func (ec ErrCond) Marshal(wr *buffer.Buffer) error {
	return (Symbol)(ec).Marshal(wr)
}

// Unmarshal decodes a buffer into this type. It is not intended for public use.
func (ec *ErrCond) Unmarshal(r *buffer.Buffer) error {
	s, err := ReadString(r)
	*ec = ErrCond(s)
	return err
}

/*
<type name="error" class="composite" source="list">
    <descriptor name="amqp:error:list" code="0x00000000:0x0000001d"/>
    <field name="condition" type="symbol" requires="error-condition" mandatory="true"/>
    <field name="description" type="string"/>
    <field name="info" type="fields"/>
</type>
*/

// Error is an AMQP error.
type Error struct {
	// A symbolic value indicating the error condition.
	Condition ErrCond

	// descriptive text about the error condition
	//
	// This text supplies any supplementary details not indicated by the condition field.
	// This text can be logged as an aid to resolving issues.
	Description string

	// map carrying information about the error condition
	Info map[string]any
}

// Marshal encodes this type into a buffer. It is not intended for public use.
func (e *Error) Marshal(wr *buffer.Buffer) error {
	return MarshalComposite(wr, TypeCodeError, []MarshalField{
		{Value: &e.Condition, Omit: false},
		{Value: &e.Description, Omit: e.Description == ""},
		{Value: e.Info, Omit: len(e.Info) == 0},
	})
}

// Unmarshal decodes a buffer into this type. It is not intended for public use.
func (e *Error) Unmarshal(r *buffer.Buffer) error {
	return UnmarshalComposite(r, TypeCodeError, []UnmarshalField{
		{Field: &e.Condition, HandleNull: func() error { return errors.New("Error.Condition is required") }},
		{Field: &e.Description},
		{Field: &e.Info},
	}...)
}

// String implements the [fmt.Stringer] interface.
// Note that the values are for diagnostic purposes and may change over time.
func (e *Error) String() string {
	if e == nil {
		return "*Error(nil)"
	}
	return fmt.Sprintf("*Error{Condition: %s, Description: %s, Info: %v}",
		e.Condition,
		e.Description,
		e.Info,
	)
}

// Error implements the error interface.
func (e *Error) Error() string {
	return e.String()
}

// Symbol is an AMQP symbolic string.
type Symbol string

// Marshal encodes this type into a buffer. It is not intended for public use.
func (s Symbol) Marshal(wr *buffer.Buffer) error {
	l := len(s)
	switch {
	// Sym8
	case l < 256:
		wr.Append([]byte{
			byte(TypeCodeSym8),
			byte(l),
		})
		wr.AppendString(string(s))

	// Sym32
	case uint(l) < math.MaxUint32:
		wr.AppendByte(uint8(TypeCodeSym32))
		wr.AppendUint32(uint32(l))
		wr.AppendString(string(s))
	default:
		return errors.New("too long")
	}
	return nil
}

// UUID is a 128 bit identifier as defined in RFC 4122.
type UUID [16]byte

// String returns the hex encoded representation described in RFC 4122, Section 3.
func (u UUID) String() string {
	var buf [36]byte
	hex.Encode(buf[:8], u[:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], u[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], u[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], u[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:], u[10:])
	return string(buf[:])
}

// Marshal encodes this type into a buffer. It is not intended for public use.
func (u UUID) Marshal(wr *buffer.Buffer) error {
	wr.AppendByte(byte(TypeCodeUUID))
	wr.Append(u[:])
	return nil
}

// Unmarshal decodes a buffer into this type. It is not intended for public use.
func (u *UUID) Unmarshal(r *buffer.Buffer) error {
	un, err := readUUID(r)
	*u = un
	return err
}

// DescribedType is used for describing a filter.
// http://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-messaging-v1.0-os.html#type-filter-set
type DescribedType struct {
	Descriptor any
	Value      any
}

// Marshal encodes this type into a buffer. It is not intended for public use.
func (t DescribedType) Marshal(wr *buffer.Buffer) error {
	wr.AppendByte(0x0) // descriptor constructor
	err := Marshal(wr, t.Descriptor)
	if err != nil {
		return err
	}
	return Marshal(wr, t.Value)
}

// Unmarshal decodes a buffer into this type. It is not intended for public use.
func (t *DescribedType) Unmarshal(r *buffer.Buffer) error {
	b, err := r.ReadByte()
	if err != nil {
		return err
	}

	if b != 0x0 {
		return fmt.Errorf("invalid described type header %02x", b)
	}

	err = Unmarshal(r, &t.Descriptor)
	if err != nil {
		return err
	}
	return Unmarshal(r, &t.Value)
}

// String implements the [fmt.Stringer] interface.
// Note that the values are for diagnostic purposes and may change over time.
func (t DescribedType) String() string {
	return fmt.Sprintf("DescribedType{descriptor: %v, value: %v}",
		t.Descriptor,
		t.Value,
	)
}

// DeliveryState encapsulates the various concrete delivery states.
// http://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-messaging-v1.0-os.html#section-delivery-state
// TODO: http://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-transactions-v1.0-os.html#type-declared
type DeliveryState interface {
	deliveryState() // marker method
}

/*
<type name="received" class="composite" source="list" provides="delivery-state">
    <descriptor name="amqp:received:list" code="0x00000000:0x00000023"/>
    <field name="section-number" type="uint" mandatory="true"/>
    <field name="section-offset" type="ulong" mandatory="true"/>
</type>
*/

// StateReceived indicates the furthest point in the payload of the message which the
// target will not need to have resent if the link is resumed.
type StateReceived struct {
	// When sent by the sender this indicates the first section of the message
	// (with section-number 0 being the first section) for which data can be resent.
	// Data from sections prior to the given section cannot be retransmitted for
	// this delivery.
	//
	// When sent by the receiver this indicates the first section of the message
	// for which all data might not yet have been received.
	SectionNumber uint32

	// When sent by the sender this indicates the first byte of the encoded section
	// data of the section given by section-number for which data can be resent
	// (with section-offset 0 being the first byte). Bytes from the same section
	// prior to the given offset section cannot be retransmitted for this delivery.
	//
	// When sent by the receiver this indicates the first byte of the given section
	// which has not yet been received. Note that if a receiver has received all of
	// section number X (which contains N bytes of data), but none of section number
	// X + 1, then it can indicate this by sending either Received(section-number=X,
	// section-offset=N) or Received(section-number=X+1, section-offset=0). The state
	// Received(section-number=0, section-offset=0) indicates that no message data
	// at all has been transferred.
	SectionOffset uint64
}

func (sr *StateReceived) deliveryState() {}

// Marshal encodes this type into a buffer. It is not intended for public use.
func (sr *StateReceived) Marshal(wr *buffer.Buffer) error {
	return MarshalComposite(wr, TypeCodeStateReceived, []MarshalField{
		{Value: &sr.SectionNumber, Omit: false},
		{Value: &sr.SectionOffset, Omit: false},
	})
}

// Unmarshal decodes a buffer into this type. It is not intended for public use.
func (sr *StateReceived) Unmarshal(r *buffer.Buffer) error {
	return UnmarshalComposite(r, TypeCodeStateReceived, []UnmarshalField{
		{Field: &sr.SectionNumber, HandleNull: func() error { return errors.New("StateReceiver.SectionNumber is required") }},
		{Field: &sr.SectionOffset, HandleNull: func() error { return errors.New("StateReceiver.SectionOffset is required") }},
	}...)
}

// String implements the [fmt.Stringer] interface.
// Note that the values are for diagnostic purposes and may change over time.
func (sr *StateReceived) String() string {
	return fmt.Sprintf("StateReceived{SectionNumber : %d, SectionOffset: %d}", sr.SectionNumber, sr.SectionOffset)
}

/*
<type name="accepted" class="composite" source="list" provides="delivery-state, outcome">
    <descriptor name="amqp:accepted:list" code="0x00000000:0x00000024"/>
</type>
*/

// StateAccepted indicates that an incoming message has been successfully processed,
// and that the receiver of the message is expecting the sender to transition the
// delivery to the accepted state at the source.
type StateAccepted struct{}

func (sr *StateAccepted) deliveryState() {}

// Marshal encodes this type into a buffer. It is not intended for public use.
func (sa *StateAccepted) Marshal(wr *buffer.Buffer) error {
	return MarshalComposite(wr, TypeCodeStateAccepted, nil)
}

// Unmarshal decodes a buffer into this type. It is not intended for public use.
func (sa *StateAccepted) Unmarshal(r *buffer.Buffer) error {
	return UnmarshalComposite(r, TypeCodeStateAccepted)
}

// String implements the [fmt.Stringer] interface.
// Note that the values are for diagnostic purposes and may change over time.
func (sa *StateAccepted) String() string {
	return "StateAccepted{}"
}

/*
<type name="rejected" class="composite" source="list" provides="delivery-state, outcome">
    <descriptor name="amqp:rejected:list" code="0x00000000:0x00000025"/>
    <field name="error" type="error"/>
</type>
*/

// StateRejected indicates that an incoming message is invalid and therefore unprocessable.
// The rejected outcome when applied to a message will cause the delivery-count to be
// incremented in the header of the rejected message.
type StateRejected struct {
	Error *Error
}

func (sr *StateRejected) deliveryState() {}

// Marshal encodes this type into a buffer. It is not intended for public use.
func (sr *StateRejected) Marshal(wr *buffer.Buffer) error {
	return MarshalComposite(wr, TypeCodeStateRejected, []MarshalField{
		{Value: sr.Error, Omit: sr.Error == nil},
	})
}

// Unmarshal decodes a buffer into this type. It is not intended for public use.
func (sr *StateRejected) Unmarshal(r *buffer.Buffer) error {
	return UnmarshalComposite(r, TypeCodeStateRejected,
		UnmarshalField{Field: &sr.Error},
	)
}

// String implements the [fmt.Stringer] interface.
// Note that the values are for diagnostic purposes and may change over time.
func (sr *StateRejected) String() string {
	return fmt.Sprintf("StateRejected{Error: %v}", sr.Error)
}

/*
<type name="released" class="composite" source="list" provides="delivery-state, outcome">
    <descriptor name="amqp:released:list" code="0x00000000:0x00000026"/>
</type>
*/

// StateReleased indicates that a given transfer was not and will not be acted upon.
type StateReleased struct{}

func (sr *StateReleased) deliveryState() {}

// Marshal encodes this type into a buffer. It is not intended for public use.
func (sr *StateReleased) Marshal(wr *buffer.Buffer) error {
	return MarshalComposite(wr, TypeCodeStateReleased, nil)
}

// Unmarshal decodes a buffer into this type. It is not intended for public use.
func (sr *StateReleased) Unmarshal(r *buffer.Buffer) error {
	return UnmarshalComposite(r, TypeCodeStateReleased)
}

// String implements the [fmt.Stringer] interface.
// Note that the values are for diagnostic purposes and may change over time.
func (sr *StateReleased) String() string {
	return "StateReleased{}"
}

/*
<type name="modified" class="composite" source="list" provides="delivery-state, outcome">
    <descriptor name="amqp:modified:list" code="0x00000000:0x00000027"/>
    <field name="delivery-failed" type="boolean"/>
    <field name="undeliverable-here" type="boolean"/>
    <field name="message-annotations" type="fields"/>
</type>
*/

// StateModifies indicates that a given transfer was not and will not be acted upon,
// and that the message SHOULD be modified in the specified ways at the node.
type StateModified struct {
	// count the transfer as an unsuccessful delivery attempt
	//
	// If the delivery-failed flag is set, any messages modified
	// MUST have their delivery-count incremented.
	DeliveryFailed bool

	// prevent redelivery
	//
	// If the undeliverable-here is set, then any messages released MUST NOT
	// be redelivered to the modifying link endpoint.
	UndeliverableHere bool

	// message attributes
	// Map containing attributes to combine with the existing message-annotations
	// held in the message's header section. Where the existing message-annotations
	// of the message contain an entry with the same key as an entry in this field,
	// the value in this field associated with that key replaces the one in the
	// existing headers; where the existing message-annotations has no such value,
	// the value in this map is added.
	MessageAnnotations Annotations
}

func (sr *StateModified) deliveryState() {}

// Marshal encodes this type into a buffer. It is not intended for public use.
func (sm *StateModified) Marshal(wr *buffer.Buffer) error {
	return MarshalComposite(wr, TypeCodeStateModified, []MarshalField{
		{Value: &sm.DeliveryFailed, Omit: !sm.DeliveryFailed},
		{Value: &sm.UndeliverableHere, Omit: !sm.UndeliverableHere},
		{Value: sm.MessageAnnotations, Omit: sm.MessageAnnotations == nil},
	})
}

// Unmarshal decodes a buffer into this type. It is not intended for public use.
func (sm *StateModified) Unmarshal(r *buffer.Buffer) error {
	return UnmarshalComposite(r, TypeCodeStateModified, []UnmarshalField{
		{Field: &sm.DeliveryFailed},
		{Field: &sm.UndeliverableHere},
		{Field: &sm.MessageAnnotations},
	}...)
}

// String implements the [fmt.Stringer] interface.
// Note that the values are for diagnostic purposes and may change over time.
func (sm *StateModified) String() string {
	return fmt.Sprintf("StateModified{DeliveryFailed: %t, UndeliverableHere: %t, MessageAnnotations: %v}", sm.DeliveryFailed, sm.UndeliverableHere, sm.MessageAnnotations)
}
