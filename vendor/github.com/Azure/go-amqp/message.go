package amqp

import (
	"fmt"
	"time"

	"github.com/Azure/go-amqp/internal/buffer"
	"github.com/Azure/go-amqp/internal/encoding"
)

// Message is an AMQP message.
type Message struct {
	// Message format code.
	//
	// The upper three octets of a message format code identify a particular message
	// format. The lowest octet indicates the version of said message format. Any
	// given version of a format is forwards compatible with all higher versions.
	Format uint32

	// The DeliveryTag can be up to 32 octets of binary data.
	// Note that when mode one is enabled there will be no delivery tag.
	DeliveryTag []byte

	// The header section carries standard delivery details about the transfer
	// of a message through the AMQP network.
	Header *MessageHeader
	// If the header section is omitted the receiver MUST assume the appropriate
	// default values (or the meaning implied by no value being set) for the
	// fields within the header unless other target or node specific defaults
	// have otherwise been set.

	// The delivery-annotations section is used for delivery-specific non-standard
	// properties at the head of the message. Delivery annotations convey information
	// from the sending peer to the receiving peer.
	DeliveryAnnotations encoding.Annotations
	// If the recipient does not understand the annotation it cannot be acted upon
	// and its effects (such as any implied propagation) cannot be acted upon.
	// Annotations might be specific to one implementation, or common to multiple
	// implementations. The capabilities negotiated on link attach and on the source
	// and target SHOULD be used to establish which annotations a peer supports. A
	// registry of defined annotations and their meanings is maintained [AMQPDELANN].
	// The symbolic key "rejected" is reserved for the use of communicating error
	// information regarding rejected messages. Any values associated with the
	// "rejected" key MUST be of type error.
	//
	// If the delivery-annotations section is omitted, it is equivalent to a
	// delivery-annotations section containing an empty map of annotations.

	// The message-annotations section is used for properties of the message which
	// are aimed at the infrastructure.
	Annotations encoding.Annotations
	// The message-annotations section is used for properties of the message which
	// are aimed at the infrastructure and SHOULD be propagated across every
	// delivery step. Message annotations convey information about the message.
	// Intermediaries MUST propagate the annotations unless the annotations are
	// explicitly augmented or modified (e.g., by the use of the modified outcome).
	//
	// The capabilities negotiated on link attach and on the source and target can
	// be used to establish which annotations a peer understands; however, in a
	// network of AMQP intermediaries it might not be possible to know if every
	// intermediary will understand the annotation. Note that for some annotations
	// it might not be necessary for the intermediary to understand their purpose,
	// i.e., they could be used purely as an attribute which can be filtered on.
	//
	// A registry of defined annotations and their meanings is maintained [AMQPMESSANN].
	//
	// If the message-annotations section is omitted, it is equivalent to a
	// message-annotations section containing an empty map of annotations.

	// The properties section is used for a defined set of standard properties of
	// the message.
	Properties *MessageProperties
	// The properties section is part of the bare message; therefore,
	// if retransmitted by an intermediary, it MUST remain unaltered.

	// The application-properties section is a part of the bare message used for
	// structured application data. Intermediaries can use the data within this
	// structure for the purposes of filtering or routing.
	ApplicationProperties map[string]interface{}
	// The keys of this map are restricted to be of type string (which excludes
	// the possibility of a null key) and the values are restricted to be of
	// simple types only, that is, excluding map, list, and array types.

	// Data payloads.
	// A data section contains opaque binary data.
	Data [][]byte

	// Value payload.
	// An amqp-value section contains a single AMQP value.
	Value interface{}

	// Sequence will contain AMQP sequence sections from the body of the message.
	// An amqp-sequence section contains an AMQP sequence.
	Sequence [][]interface{}

	// The footer section is used for details about the message or delivery which
	// can only be calculated or evaluated once the whole bare message has been
	// constructed or seen (for example message hashes, HMACs, signatures and
	// encryption details).
	Footer encoding.Annotations

	// Mark the message as settled when LinkSenderSettle is ModeMixed.
	//
	// This field is ignored when LinkSenderSettle is not ModeMixed.
	SendSettled bool

	link       *link  // the receiving link
	deliveryID uint32 // used when sending disposition
	settled    bool   // whether transfer was settled by sender
}

// NewMessage returns a *Message with data as the payload.
//
// This constructor is intended as a helper for basic Messages with a
// single data payload. It is valid to construct a Message directly for
// more complex usages.
func NewMessage(data []byte) *Message {
	return &Message{
		Data: [][]byte{data},
	}
}

// GetData returns the first []byte from the Data field
// or nil if Data is empty.
func (m *Message) GetData() []byte {
	if len(m.Data) < 1 {
		return nil
	}
	return m.Data[0]
}

// LinkName returns the receiving link name or the empty string.
func (m *Message) LinkName() string {
	if m.link != nil {
		return m.link.Key.name
	}
	return ""
}

// MarshalBinary encodes the message into binary form.
func (m *Message) MarshalBinary() ([]byte, error) {
	buf := &buffer.Buffer{}
	err := m.Marshal(buf)
	return buf.Detach(), err
}

func (m *Message) shouldSendDisposition() bool {
	return !m.settled
}

func (m *Message) Marshal(wr *buffer.Buffer) error {
	if m.Header != nil {
		err := m.Header.Marshal(wr)
		if err != nil {
			return err
		}
	}

	if m.DeliveryAnnotations != nil {
		encoding.WriteDescriptor(wr, encoding.TypeCodeDeliveryAnnotations)
		err := encoding.Marshal(wr, m.DeliveryAnnotations)
		if err != nil {
			return err
		}
	}

	if m.Annotations != nil {
		encoding.WriteDescriptor(wr, encoding.TypeCodeMessageAnnotations)
		err := encoding.Marshal(wr, m.Annotations)
		if err != nil {
			return err
		}
	}

	if m.Properties != nil {
		err := encoding.Marshal(wr, m.Properties)
		if err != nil {
			return err
		}
	}

	if m.ApplicationProperties != nil {
		encoding.WriteDescriptor(wr, encoding.TypeCodeApplicationProperties)
		err := encoding.Marshal(wr, m.ApplicationProperties)
		if err != nil {
			return err
		}
	}

	for _, data := range m.Data {
		encoding.WriteDescriptor(wr, encoding.TypeCodeApplicationData)
		err := encoding.WriteBinary(wr, data)
		if err != nil {
			return err
		}
	}

	if m.Value != nil {
		encoding.WriteDescriptor(wr, encoding.TypeCodeAMQPValue)
		err := encoding.Marshal(wr, m.Value)
		if err != nil {
			return err
		}
	}

	if m.Sequence != nil {
		// the body can basically be one of three different types (value, data or sequence).
		// When it's sequence it's actually _several_ sequence sections, one for each sub-array.
		for _, v := range m.Sequence {
			encoding.WriteDescriptor(wr, encoding.TypeCodeAMQPSequence)
			err := encoding.Marshal(wr, v)
			if err != nil {
				return err
			}
		}
	}

	if m.Footer != nil {
		encoding.WriteDescriptor(wr, encoding.TypeCodeFooter)
		err := encoding.Marshal(wr, m.Footer)
		if err != nil {
			return err
		}
	}

	return nil
}

// UnmarshalBinary decodes the message from binary form.
func (m *Message) UnmarshalBinary(data []byte) error {
	buf := buffer.New(data)
	return m.Unmarshal(buf)
}

func (m *Message) Unmarshal(r *buffer.Buffer) error {
	// loop, decoding sections until bytes have been consumed
	for r.Len() > 0 {
		// determine type
		type_, headerLength, err := encoding.PeekMessageType(r.Bytes())
		if err != nil {
			return err
		}

		var (
			section interface{}
			// section header is read from r before
			// unmarshaling section is set to true
			discardHeader = true
		)
		switch encoding.AMQPType(type_) {

		case encoding.TypeCodeMessageHeader:
			discardHeader = false
			section = &m.Header

		case encoding.TypeCodeDeliveryAnnotations:
			section = &m.DeliveryAnnotations

		case encoding.TypeCodeMessageAnnotations:
			section = &m.Annotations

		case encoding.TypeCodeMessageProperties:
			discardHeader = false
			section = &m.Properties

		case encoding.TypeCodeApplicationProperties:
			section = &m.ApplicationProperties

		case encoding.TypeCodeApplicationData:
			r.Skip(int(headerLength))

			var data []byte
			err = encoding.Unmarshal(r, &data)
			if err != nil {
				return err
			}

			m.Data = append(m.Data, data)
			continue

		case encoding.TypeCodeAMQPSequence:
			r.Skip(int(headerLength))

			var data []interface{}
			err = encoding.Unmarshal(r, &data)
			if err != nil {
				return err
			}

			m.Sequence = append(m.Sequence, data)
			continue

		case encoding.TypeCodeFooter:
			section = &m.Footer

		case encoding.TypeCodeAMQPValue:
			section = &m.Value

		default:
			return fmt.Errorf("unknown message section %#02x", type_)
		}

		if discardHeader {
			r.Skip(int(headerLength))
		}

		err = encoding.Unmarshal(r, section)
		if err != nil {
			return err
		}
	}
	return nil
}

/*
<type name="header" class="composite" source="list" provides="section">
    <descriptor name="amqp:header:list" code="0x00000000:0x00000070"/>
    <field name="durable" type="boolean" default="false"/>
    <field name="priority" type="ubyte" default="4"/>
    <field name="ttl" type="milliseconds"/>
    <field name="first-acquirer" type="boolean" default="false"/>
    <field name="delivery-count" type="uint" default="0"/>
</type>
*/

// MessageHeader carries standard delivery details about the transfer
// of a message.
type MessageHeader struct {
	Durable       bool
	Priority      uint8
	TTL           time.Duration // from milliseconds
	FirstAcquirer bool
	DeliveryCount uint32
}

func (h *MessageHeader) Marshal(wr *buffer.Buffer) error {
	return encoding.MarshalComposite(wr, encoding.TypeCodeMessageHeader, []encoding.MarshalField{
		{Value: &h.Durable, Omit: !h.Durable},
		{Value: &h.Priority, Omit: h.Priority == 4},
		{Value: (*encoding.Milliseconds)(&h.TTL), Omit: h.TTL == 0},
		{Value: &h.FirstAcquirer, Omit: !h.FirstAcquirer},
		{Value: &h.DeliveryCount, Omit: h.DeliveryCount == 0},
	})
}

func (h *MessageHeader) Unmarshal(r *buffer.Buffer) error {
	return encoding.UnmarshalComposite(r, encoding.TypeCodeMessageHeader, []encoding.UnmarshalField{
		{Field: &h.Durable},
		{Field: &h.Priority, HandleNull: func() error { h.Priority = 4; return nil }},
		{Field: (*encoding.Milliseconds)(&h.TTL)},
		{Field: &h.FirstAcquirer},
		{Field: &h.DeliveryCount},
	}...)
}

// AMQP types
type (
	// AMQPAddress corresponds to the 'address' type in the AMQP spec.
	// <type name="address-string" class="restricted" source="string" provides="address"/>
	AMQPAddress = string

	// AMQPMessageID corresponds to the 'message-id' type in the AMQP spec. Internally it can
	// be one of the following:
	// - uint64:    <type name="message-id-ulong" class="restricted" source="ulong" provides="message-id"/>
	// - amqp.UUID: <type name="message-id-uuid" class="restricted" source="uuid" provides="message-id"/>
	// - []byte:    <type name="message-id-binary" class="restricted" source="binary" provides="message-id"/>
	// - string:    <type name="message-id-string" class="restricted" source="string" provides="message-id"/>
	AMQPMessageID = interface{}

	// AMQPSymbol corresponds to the 'symbol' type in the AMQP spec.
	// <type name="symbol" class="primitive"/>
	// And either:
	// - variable-width, 1 byte size	up to 2^8 - 1 seven bit ASCII characters representing a symbolic value
	// - variable-width, 4 byte size	up to 2^32 - 1 seven bit ASCII characters representing a symbolic value
	AMQPSymbol = string

	// AMQPSequenceNumber corresponds to the `sequence-no` type in the AMQP spec.
	// <type name="sequence-no" class="restricted" source="uint"/>
	AMQPSequenceNumber = uint32

	// AMQPBinary corresponds to the `binary` type in the AMQP spec.
	AMQPBinary = []byte
)

/*
<type name="properties" class="composite" source="list" provides="section">
    <descriptor name="amqp:properties:list" code="0x00000000:0x00000073"/>
    <field name="message-id" type="*" requires="message-id"/>
    <field name="user-id" type="binary"/>
    <field name="to" type="*" requires="address"/>
    <field name="subject" type="string"/>
    <field name="reply-to" type="*" requires="address"/>
    <field name="correlation-id" type="*" requires="message-id"/>
    <field name="content-type" type="symbol"/>
    <field name="content-encoding" type="symbol"/>
    <field name="absolute-expiry-time" type="timestamp"/>
    <field name="creation-time" type="timestamp"/>
    <field name="group-id" type="string"/>
    <field name="group-sequence" type="sequence-no"/>
    <field name="reply-to-group-id" type="string"/>
</type>
*/

// MessageProperties is the defined set of properties for AMQP messages.
type MessageProperties struct {
	// Message-id, if set, uniquely identifies a message within the message system.
	// The message producer is usually responsible for setting the message-id in
	// such a way that it is assured to be globally unique. A broker MAY discard a
	// message as a duplicate if the value of the message-id matches that of a
	// previously received message sent to the same node.
	MessageID AMQPMessageID // uint64, UUID, []byte, or string

	// The identity of the user responsible for producing the message.
	// The client sets this value, and it MAY be authenticated by intermediaries.
	UserID AMQPBinary

	// The to field identifies the node that is the intended destination of the message.
	// On any given transfer this might not be the node at the receiving end of the link.
	To *AMQPAddress

	// A common field for summary information about the message content and purpose.
	Subject *string

	// The address of the node to send replies to.
	ReplyTo *AMQPAddress

	// This is a client-specific id that can be used to mark or identify messages
	// between clients.
	CorrelationID AMQPMessageID // uint64, UUID, []byte, or string

	// The RFC-2046 [RFC2046] MIME type for the message's application-data section
	// (body). As per RFC-2046 [RFC2046] this can contain a charset parameter defining
	// the character encoding used: e.g., 'text/plain; charset="utf-8"'.
	//
	// For clarity, as per section 7.2.1 of RFC-2616 [RFC2616], where the content type
	// is unknown the content-type SHOULD NOT be set. This allows the recipient the
	// opportunity to determine the actual type. Where the section is known to be truly
	// opaque binary data, the content-type SHOULD be set to application/octet-stream.
	//
	// When using an application-data section with a section code other than data,
	// content-type SHOULD NOT be set.
	ContentType *AMQPSymbol

	// The content-encoding property is used as a modifier to the content-type.
	// When present, its value indicates what additional content encodings have been
	// applied to the application-data, and thus what decoding mechanisms need to be
	// applied in order to obtain the media-type referenced by the content-type header
	// field.
	//
	// Content-encoding is primarily used to allow a document to be compressed without
	// losing the identity of its underlying content type.
	//
	// Content-encodings are to be interpreted as per section 3.5 of RFC 2616 [RFC2616].
	// Valid content-encodings are registered at IANA [IANAHTTPPARAMS].
	//
	// The content-encoding MUST NOT be set when the application-data section is other
	// than data. The binary representation of all other application-data section types
	// is defined completely in terms of the AMQP type system.
	//
	// Implementations MUST NOT use the identity encoding. Instead, implementations
	// SHOULD NOT set this property. Implementations SHOULD NOT use the compress encoding,
	// except as to remain compatible with messages originally sent with other protocols,
	// e.g. HTTP or SMTP.
	//
	// Implementations SHOULD NOT specify multiple content-encoding values except as to
	// be compatible with messages originally sent with other protocols, e.g. HTTP or SMTP.
	ContentEncoding *AMQPSymbol

	// An absolute time when this message is considered to be expired.
	AbsoluteExpiryTime *time.Time

	// An absolute time when this message was created.
	CreationTime *time.Time

	// Identifies the group the message belongs to.
	GroupID *string

	// The relative position of this message within its group.
	GroupSequence *AMQPSequenceNumber // RFC-1982 sequence number

	// This is a client-specific id that is used so that client can send replies to this
	// message to a specific group.
	ReplyToGroupID *string
}

func (p *MessageProperties) Marshal(wr *buffer.Buffer) error {
	return encoding.MarshalComposite(wr, encoding.TypeCodeMessageProperties, []encoding.MarshalField{
		{Value: p.MessageID, Omit: p.MessageID == nil},
		{Value: &p.UserID, Omit: len(p.UserID) == 0},
		{Value: p.To, Omit: p.To == nil},
		{Value: p.Subject, Omit: p.Subject == nil},
		{Value: p.ReplyTo, Omit: p.ReplyTo == nil},
		{Value: p.CorrelationID, Omit: p.CorrelationID == nil},
		{Value: (*encoding.Symbol)(p.ContentType), Omit: p.ContentType == nil},
		{Value: (*encoding.Symbol)(p.ContentEncoding), Omit: p.ContentEncoding == nil},
		{Value: p.AbsoluteExpiryTime, Omit: p.AbsoluteExpiryTime == nil},
		{Value: p.CreationTime, Omit: p.CreationTime == nil},
		{Value: p.GroupID, Omit: p.GroupID == nil},
		{Value: p.GroupSequence, Omit: p.GroupSequence == nil},
		{Value: p.ReplyToGroupID, Omit: p.ReplyToGroupID == nil},
	})
}

func (p *MessageProperties) Unmarshal(r *buffer.Buffer) error {
	return encoding.UnmarshalComposite(r, encoding.TypeCodeMessageProperties, []encoding.UnmarshalField{
		{Field: &p.MessageID},
		{Field: &p.UserID},
		{Field: &p.To},
		{Field: &p.Subject},
		{Field: &p.ReplyTo},
		{Field: &p.CorrelationID},
		{Field: &p.ContentType},
		{Field: &p.ContentEncoding},
		{Field: &p.AbsoluteExpiryTime},
		{Field: &p.CreationTime},
		{Field: &p.GroupID},
		{Field: &p.GroupSequence},
		{Field: &p.ReplyToGroupID},
	}...)
}

// Annotations keys must be of type string, int, or int64.
//
// String keys are encoded as AMQP Symbols.
type Annotations = encoding.Annotations

type UUID = encoding.UUID
