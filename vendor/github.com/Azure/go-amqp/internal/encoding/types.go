package encoding

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"reflect"
	"time"
	"unicode/utf8"

	"github.com/Azure/go-amqp/internal/buffer"
)

type AMQPType uint8

// Type codes
const (
	TypeCodeNull AMQPType = 0x40

	// Bool
	TypeCodeBool      AMQPType = 0x56 // boolean with the octet 0x00 being false and octet 0x01 being true
	TypeCodeBoolTrue  AMQPType = 0x41
	TypeCodeBoolFalse AMQPType = 0x42

	// Unsigned
	TypeCodeUbyte      AMQPType = 0x50 // 8-bit unsigned integer (1)
	TypeCodeUshort     AMQPType = 0x60 // 16-bit unsigned integer in network byte order (2)
	TypeCodeUint       AMQPType = 0x70 // 32-bit unsigned integer in network byte order (4)
	TypeCodeSmallUint  AMQPType = 0x52 // unsigned integer value in the range 0 to 255 inclusive (1)
	TypeCodeUint0      AMQPType = 0x43 // the uint value 0 (0)
	TypeCodeUlong      AMQPType = 0x80 // 64-bit unsigned integer in network byte order (8)
	TypeCodeSmallUlong AMQPType = 0x53 // unsigned long value in the range 0 to 255 inclusive (1)
	TypeCodeUlong0     AMQPType = 0x44 // the ulong value 0 (0)

	// Signed
	TypeCodeByte      AMQPType = 0x51 // 8-bit two's-complement integer (1)
	TypeCodeShort     AMQPType = 0x61 // 16-bit two's-complement integer in network byte order (2)
	TypeCodeInt       AMQPType = 0x71 // 32-bit two's-complement integer in network byte order (4)
	TypeCodeSmallint  AMQPType = 0x54 // 8-bit two's-complement integer (1)
	TypeCodeLong      AMQPType = 0x81 // 64-bit two's-complement integer in network byte order (8)
	TypeCodeSmalllong AMQPType = 0x55 // 8-bit two's-complement integer

	// Decimal
	TypeCodeFloat      AMQPType = 0x72 // IEEE 754-2008 binary32 (4)
	TypeCodeDouble     AMQPType = 0x82 // IEEE 754-2008 binary64 (8)
	TypeCodeDecimal32  AMQPType = 0x74 // IEEE 754-2008 decimal32 using the Binary Integer Decimal encoding (4)
	TypeCodeDecimal64  AMQPType = 0x84 // IEEE 754-2008 decimal64 using the Binary Integer Decimal encoding (8)
	TypeCodeDecimal128 AMQPType = 0x94 // IEEE 754-2008 decimal128 using the Binary Integer Decimal encoding (16)

	// Other
	TypeCodeChar      AMQPType = 0x73 // a UTF-32BE encoded Unicode character (4)
	TypeCodeTimestamp AMQPType = 0x83 // 64-bit two's-complement integer representing milliseconds since the unix epoch
	TypeCodeUUID      AMQPType = 0x98 // UUID as defined in section 4.1.2 of RFC-4122

	// Variable Length
	TypeCodeVbin8  AMQPType = 0xa0 // up to 2^8 - 1 octets of binary data (1 + variable)
	TypeCodeVbin32 AMQPType = 0xb0 // up to 2^32 - 1 octets of binary data (4 + variable)
	TypeCodeStr8   AMQPType = 0xa1 // up to 2^8 - 1 octets worth of UTF-8 Unicode (with no byte order mark) (1 + variable)
	TypeCodeStr32  AMQPType = 0xb1 // up to 2^32 - 1 octets worth of UTF-8 Unicode (with no byte order mark) (4 +variable)
	TypeCodeSym8   AMQPType = 0xa3 // up to 2^8 - 1 seven bit ASCII characters representing a symbolic value (1 + variable)
	TypeCodeSym32  AMQPType = 0xb3 // up to 2^32 - 1 seven bit ASCII characters representing a symbolic value (4 + variable)

	// Compound
	TypeCodeList0   AMQPType = 0x45 // the empty list (i.e. the list with no elements) (0)
	TypeCodeList8   AMQPType = 0xc0 // up to 2^8 - 1 list elements with total size less than 2^8 octets (1 + compound)
	TypeCodeList32  AMQPType = 0xd0 // up to 2^32 - 1 list elements with total size less than 2^32 octets (4 + compound)
	TypeCodeMap8    AMQPType = 0xc1 // up to 2^8 - 1 octets of encoded map data (1 + compound)
	TypeCodeMap32   AMQPType = 0xd1 // up to 2^32 - 1 octets of encoded map data (4 + compound)
	TypeCodeArray8  AMQPType = 0xe0 // up to 2^8 - 1 array elements with total size less than 2^8 octets (1 + array)
	TypeCodeArray32 AMQPType = 0xf0 // up to 2^32 - 1 array elements with total size less than 2^32 octets (4 + array)

	// Composites
	TypeCodeOpen        AMQPType = 0x10
	TypeCodeBegin       AMQPType = 0x11
	TypeCodeAttach      AMQPType = 0x12
	TypeCodeFlow        AMQPType = 0x13
	TypeCodeTransfer    AMQPType = 0x14
	TypeCodeDisposition AMQPType = 0x15
	TypeCodeDetach      AMQPType = 0x16
	TypeCodeEnd         AMQPType = 0x17
	TypeCodeClose       AMQPType = 0x18

	TypeCodeSource AMQPType = 0x28
	TypeCodeTarget AMQPType = 0x29
	TypeCodeError  AMQPType = 0x1d

	TypeCodeMessageHeader         AMQPType = 0x70
	TypeCodeDeliveryAnnotations   AMQPType = 0x71
	TypeCodeMessageAnnotations    AMQPType = 0x72
	TypeCodeMessageProperties     AMQPType = 0x73
	TypeCodeApplicationProperties AMQPType = 0x74
	TypeCodeApplicationData       AMQPType = 0x75
	TypeCodeAMQPSequence          AMQPType = 0x76
	TypeCodeAMQPValue             AMQPType = 0x77
	TypeCodeFooter                AMQPType = 0x78

	TypeCodeStateReceived AMQPType = 0x23
	TypeCodeStateAccepted AMQPType = 0x24
	TypeCodeStateRejected AMQPType = 0x25
	TypeCodeStateReleased AMQPType = 0x26
	TypeCodeStateModified AMQPType = 0x27

	TypeCodeSASLMechanism AMQPType = 0x40
	TypeCodeSASLInit      AMQPType = 0x41
	TypeCodeSASLChallenge AMQPType = 0x42
	TypeCodeSASLResponse  AMQPType = 0x43
	TypeCodeSASLOutcome   AMQPType = 0x44

	TypeCodeDeleteOnClose             AMQPType = 0x2b
	TypeCodeDeleteOnNoLinks           AMQPType = 0x2c
	TypeCodeDeleteOnNoMessages        AMQPType = 0x2d
	TypeCodeDeleteOnNoLinksOrMessages AMQPType = 0x2e
)

func ValidateExpiryPolicy(e ExpiryPolicy) error {
	switch e {
	case ExpiryLinkDetach,
		ExpirySessionEnd,
		ExpiryConnectionClose,
		ExpiryNever:
		return nil
	default:
		return fmt.Errorf("unknown expiry-policy %q", e)
	}
}

type Role bool

const (
	RoleSender   Role = false
	RoleReceiver Role = true
)

func (rl Role) String() string {
	if rl {
		return "Receiver"
	}
	return "Sender"
}

func (rl *Role) Unmarshal(r *buffer.Buffer) error {
	b, err := readBool(r)
	*rl = Role(b)
	return err
}

func (rl Role) Marshal(wr *buffer.Buffer) error {
	return Marshal(wr, (bool)(rl))
}

type SASLCode uint8

// SASL Codes
const (
	CodeSASLOK      SASLCode = iota // Connection authentication succeeded.
	CodeSASLAuth                    // Connection authentication failed due to an unspecified problem with the supplied credentials.
	CodeSASLSysPerm                 // Connection authentication failed due to a system error that is unlikely to be corrected without intervention.
)

func (s SASLCode) Marshal(wr *buffer.Buffer) error {
	return Marshal(wr, uint8(s))
}

func (s *SASLCode) Unmarshal(r *buffer.Buffer) error {
	n, err := ReadUbyte(r)
	*s = SASLCode(n)
	return err
}

type Unsettled map[string]DeliveryState

func (u Unsettled) Marshal(wr *buffer.Buffer) error {
	return writeMap(wr, u)
}

func (u *Unsettled) Unmarshal(r *buffer.Buffer) error {
	count, err := readMapHeader(r)
	if err != nil {
		return err
	}

	m := make(Unsettled, count/2)
	for i := uint32(0); i < count; i += 2 {
		key, err := ReadString(r)
		if err != nil {
			return err
		}
		var value DeliveryState
		err = Unmarshal(r, &value)
		if err != nil {
			return err
		}
		m[key] = value
	}
	*u = m
	return nil
}

// peekMessageType reads the message type without
// modifying any data.
func PeekMessageType(buf []byte) (uint8, uint8, error) {
	if len(buf) < 3 {
		return 0, 0, errors.New("invalid message")
	}

	if buf[0] != 0 {
		return 0, 0, fmt.Errorf("invalid composite header %02x", buf[0])
	}

	// copied from readUlong to avoid allocations
	t := AMQPType(buf[1])
	if t == TypeCodeUlong0 {
		return 0, 2, nil
	}

	if t == TypeCodeSmallUlong {
		if len(buf[2:]) == 0 {
			return 0, 0, errors.New("invalid ulong")
		}
		return buf[2], 3, nil
	}

	if t != TypeCodeUlong {
		return 0, 0, fmt.Errorf("invalid type for uint32 %02x", t)
	}

	if len(buf[2:]) < 8 {
		return 0, 0, errors.New("invalid ulong")
	}
	v := binary.BigEndian.Uint64(buf[2:10])

	return uint8(v), 10, nil
}

func tryReadNull(r *buffer.Buffer) bool {
	if r.Len() > 0 && AMQPType(r.Bytes()[0]) == TypeCodeNull {
		r.Skip(1)
		return true
	}
	return false
}

type Milliseconds time.Duration

func (m Milliseconds) Marshal(wr *buffer.Buffer) error {
	writeUint32(wr, uint32(m/Milliseconds(time.Millisecond)))
	return nil
}

func (m *Milliseconds) Unmarshal(r *buffer.Buffer) error {
	n, err := readUint(r)
	*m = Milliseconds(time.Duration(n) * time.Millisecond)
	return err
}

// mapAnyAny is used to decode AMQP maps who's keys are undefined or
// inconsistently typed.
type mapAnyAny map[any]any

func (m mapAnyAny) Marshal(wr *buffer.Buffer) error {
	return writeMap(wr, map[any]any(m))
}

func (m *mapAnyAny) unmarshalMap8(r *buffer.Buffer) error {
	count, err := readMap8Header(r)
	if err != nil {
		return err
	}
	return m.unmarshalMapItems(r, count)
}

func (m *mapAnyAny) unmarshalMap32(r *buffer.Buffer) error {
	count, err := readMap32Header(r)
	if err != nil {
		return err
	}
	return m.unmarshalMapItems(r, count)
}

func (m *mapAnyAny) Unmarshal(r *buffer.Buffer) error {
	count, err := readMapHeader(r)
	if err != nil {
		return err
	}
	return m.unmarshalMapItems(r, count)
}

func (m *mapAnyAny) unmarshalMapItems(r *buffer.Buffer, count uint32) error {
	mm := make(mapAnyAny, count/2)
	for i := uint32(0); i < count; i += 2 {
		key, err := ReadAny(r)
		if err != nil {
			return err
		}
		value, err := ReadAny(r)
		if err != nil {
			return err
		}

		// https://golang.org/ref/spec#Map_types:
		// The comparison operators == and != must be fully defined
		// for operands of the key type; thus the key type must not
		// be a function, map, or slice.
		switch reflect.ValueOf(key).Kind() {
		case reflect.Slice, reflect.Func, reflect.Map:
			return errors.New("invalid map key")
		}

		mm[key] = value
	}
	*m = mm
	return nil
}

// mapStringAny is used to decode AMQP maps that have string keys
type mapStringAny map[string]any

func (m mapStringAny) Marshal(wr *buffer.Buffer) error {
	return writeMap(wr, map[string]any(m))
}

func (m *mapStringAny) Unmarshal(r *buffer.Buffer) error {
	count, err := readMapHeader(r)
	if err != nil {
		return err
	}

	mm := make(mapStringAny, count/2)
	for i := uint32(0); i < count; i += 2 {
		key, err := ReadString(r)
		if err != nil {
			return err
		}
		value, err := ReadAny(r)
		if err != nil {
			return err
		}
		mm[key] = value
	}
	*m = mm

	return nil
}

// mapStringAny is used to decode AMQP maps that have Symbol keys
type mapSymbolAny map[Symbol]any

func (m mapSymbolAny) Marshal(wr *buffer.Buffer) error {
	return writeMap(wr, map[Symbol]any(m))
}

func (m *mapSymbolAny) Unmarshal(r *buffer.Buffer) error {
	count, err := readMapHeader(r)
	if err != nil {
		return err
	}

	mm := make(mapSymbolAny, count/2)
	for i := uint32(0); i < count; i += 2 {
		key, err := ReadString(r)
		if err != nil {
			return err
		}
		value, err := ReadAny(r)
		if err != nil {
			return err
		}
		mm[Symbol(key)] = value
	}
	*m = mm
	return nil
}

type LifetimePolicy uint8

const (
	DeleteOnClose             = LifetimePolicy(TypeCodeDeleteOnClose)
	DeleteOnNoLinks           = LifetimePolicy(TypeCodeDeleteOnNoLinks)
	DeleteOnNoMessages        = LifetimePolicy(TypeCodeDeleteOnNoMessages)
	DeleteOnNoLinksOrMessages = LifetimePolicy(TypeCodeDeleteOnNoLinksOrMessages)
)

func (p LifetimePolicy) Marshal(wr *buffer.Buffer) error {
	wr.Append([]byte{
		0x0,
		byte(TypeCodeSmallUlong),
		byte(p),
		byte(TypeCodeList0),
	})
	return nil
}

func (p *LifetimePolicy) Unmarshal(r *buffer.Buffer) error {
	typ, fields, err := readCompositeHeader(r)
	if err != nil {
		return err
	}
	if fields != 0 {
		return fmt.Errorf("invalid size %d for lifetime-policy", fields)
	}
	*p = LifetimePolicy(typ)
	return nil
}

// SLICES

// ArrayUByte allows encoding []uint8/[]byte as an array
// rather than binary data.
type ArrayUByte []uint8

func (a ArrayUByte) Marshal(wr *buffer.Buffer) error {
	const typeSize = 1

	writeArrayHeader(wr, len(a), typeSize, TypeCodeUbyte)
	wr.Append(a)

	return nil
}

func (a *ArrayUByte) Unmarshal(r *buffer.Buffer) error {
	length, err := readArrayHeader(r)
	if err != nil {
		return err
	}

	type_, err := readType(r)
	if err != nil {
		return err
	}
	if type_ != TypeCodeUbyte {
		return fmt.Errorf("invalid type for []uint16 %02x", type_)
	}

	buf, ok := r.Next(length)
	if !ok {
		return fmt.Errorf("invalid length %d", length)
	}
	*a = append([]byte(nil), buf...)

	return nil
}

type arrayInt8 []int8

func (a arrayInt8) Marshal(wr *buffer.Buffer) error {
	const typeSize = 1

	writeArrayHeader(wr, len(a), typeSize, TypeCodeByte)

	for _, value := range a {
		wr.AppendByte(uint8(value))
	}

	return nil
}

func (a *arrayInt8) Unmarshal(r *buffer.Buffer) error {
	length, err := readArrayHeader(r)
	if err != nil {
		return err
	}

	type_, err := readType(r)
	if err != nil {
		return err
	}
	if type_ != TypeCodeByte {
		return fmt.Errorf("invalid type for []uint16 %02x", type_)
	}

	buf, ok := r.Next(length)
	if !ok {
		return fmt.Errorf("invalid length %d", length)
	}

	aa := (*a)[:0]
	if int64(cap(aa)) < length {
		aa = make([]int8, length)
	} else {
		aa = aa[:length]
	}

	for i, value := range buf {
		aa[i] = int8(value)
	}

	*a = aa
	return nil
}

type arrayUint16 []uint16

func (a arrayUint16) Marshal(wr *buffer.Buffer) error {
	const typeSize = 2

	writeArrayHeader(wr, len(a), typeSize, TypeCodeUshort)

	for _, element := range a {
		wr.AppendUint16(element)
	}

	return nil
}

func (a *arrayUint16) Unmarshal(r *buffer.Buffer) error {
	length, err := readArrayHeader(r)
	if err != nil {
		return err
	}

	type_, err := readType(r)
	if err != nil {
		return err
	}
	if type_ != TypeCodeUshort {
		return fmt.Errorf("invalid type for []uint16 %02x", type_)
	}

	const typeSize = 2
	buf, ok := r.Next(length * typeSize)
	if !ok {
		return fmt.Errorf("invalid length %d", length)
	}

	aa := (*a)[:0]
	if int64(cap(aa)) < length {
		aa = make([]uint16, length)
	} else {
		aa = aa[:length]
	}

	var bufIdx int
	for i := range aa {
		aa[i] = binary.BigEndian.Uint16(buf[bufIdx:])
		bufIdx += 2
	}

	*a = aa
	return nil
}

type arrayInt16 []int16

func (a arrayInt16) Marshal(wr *buffer.Buffer) error {
	const typeSize = 2

	writeArrayHeader(wr, len(a), typeSize, TypeCodeShort)

	for _, element := range a {
		wr.AppendUint16(uint16(element))
	}

	return nil
}

func (a *arrayInt16) Unmarshal(r *buffer.Buffer) error {
	length, err := readArrayHeader(r)
	if err != nil {
		return err
	}

	type_, err := readType(r)
	if err != nil {
		return err
	}
	if type_ != TypeCodeShort {
		return fmt.Errorf("invalid type for []uint16 %02x", type_)
	}

	const typeSize = 2
	buf, ok := r.Next(length * typeSize)
	if !ok {
		return fmt.Errorf("invalid length %d", length)
	}

	aa := (*a)[:0]
	if int64(cap(aa)) < length {
		aa = make([]int16, length)
	} else {
		aa = aa[:length]
	}

	var bufIdx int
	for i := range aa {
		aa[i] = int16(binary.BigEndian.Uint16(buf[bufIdx : bufIdx+2]))
		bufIdx += 2
	}

	*a = aa
	return nil
}

type arrayUint32 []uint32

func (a arrayUint32) Marshal(wr *buffer.Buffer) error {
	var (
		typeSize = 1
		TypeCode = TypeCodeSmallUint
	)
	for _, n := range a {
		if n > math.MaxUint8 {
			typeSize = 4
			TypeCode = TypeCodeUint
			break
		}
	}

	writeArrayHeader(wr, len(a), typeSize, TypeCode)

	if TypeCode == TypeCodeUint {
		for _, element := range a {
			wr.AppendUint32(element)
		}
	} else {
		for _, element := range a {
			wr.AppendByte(byte(element))
		}
	}

	return nil
}

func (a *arrayUint32) Unmarshal(r *buffer.Buffer) error {
	length, err := readArrayHeader(r)
	if err != nil {
		return err
	}

	aa := (*a)[:0]

	type_, err := readType(r)
	if err != nil {
		return err
	}
	switch type_ {
	case TypeCodeUint0:
		if int64(cap(aa)) < length {
			aa = make([]uint32, length)
		} else {
			aa = aa[:length]
			for i := range aa {
				aa[i] = 0
			}
		}
	case TypeCodeSmallUint:
		buf, ok := r.Next(length)
		if !ok {
			return errors.New("invalid length")
		}

		if int64(cap(aa)) < length {
			aa = make([]uint32, length)
		} else {
			aa = aa[:length]
		}

		for i, n := range buf {
			aa[i] = uint32(n)
		}
	case TypeCodeUint:
		const typeSize = 4
		buf, ok := r.Next(length * typeSize)
		if !ok {
			return fmt.Errorf("invalid length %d", length)
		}

		if int64(cap(aa)) < length {
			aa = make([]uint32, length)
		} else {
			aa = aa[:length]
		}

		var bufIdx int
		for i := range aa {
			aa[i] = binary.BigEndian.Uint32(buf[bufIdx : bufIdx+4])
			bufIdx += 4
		}
	default:
		return fmt.Errorf("invalid type for []uint32 %02x", type_)
	}

	*a = aa
	return nil
}

type arrayInt32 []int32

func (a arrayInt32) Marshal(wr *buffer.Buffer) error {
	var (
		typeSize = 1
		TypeCode = TypeCodeSmallint
	)
	for _, n := range a {
		if n > math.MaxInt8 {
			typeSize = 4
			TypeCode = TypeCodeInt
			break
		}
	}

	writeArrayHeader(wr, len(a), typeSize, TypeCode)

	if TypeCode == TypeCodeInt {
		for _, element := range a {
			wr.AppendUint32(uint32(element))
		}
	} else {
		for _, element := range a {
			wr.AppendByte(byte(element))
		}
	}

	return nil
}

func (a *arrayInt32) Unmarshal(r *buffer.Buffer) error {
	length, err := readArrayHeader(r)
	if err != nil {
		return err
	}

	aa := (*a)[:0]

	type_, err := readType(r)
	if err != nil {
		return err
	}
	switch type_ {
	case TypeCodeSmallint:
		buf, ok := r.Next(length)
		if !ok {
			return errors.New("invalid length")
		}

		if int64(cap(aa)) < length {
			aa = make([]int32, length)
		} else {
			aa = aa[:length]
		}

		for i, n := range buf {
			aa[i] = int32(int8(n))
		}
	case TypeCodeInt:
		const typeSize = 4
		buf, ok := r.Next(length * typeSize)
		if !ok {
			return fmt.Errorf("invalid length %d", length)
		}

		if int64(cap(aa)) < length {
			aa = make([]int32, length)
		} else {
			aa = aa[:length]
		}

		var bufIdx int
		for i := range aa {
			aa[i] = int32(binary.BigEndian.Uint32(buf[bufIdx:]))
			bufIdx += 4
		}
	default:
		return fmt.Errorf("invalid type for []int32 %02x", type_)
	}

	*a = aa
	return nil
}

type arrayUint64 []uint64

func (a arrayUint64) Marshal(wr *buffer.Buffer) error {
	var (
		typeSize = 1
		TypeCode = TypeCodeSmallUlong
	)
	for _, n := range a {
		if n > math.MaxUint8 {
			typeSize = 8
			TypeCode = TypeCodeUlong
			break
		}
	}

	writeArrayHeader(wr, len(a), typeSize, TypeCode)

	if TypeCode == TypeCodeUlong {
		for _, element := range a {
			wr.AppendUint64(element)
		}
	} else {
		for _, element := range a {
			wr.AppendByte(byte(element))
		}
	}

	return nil
}

func (a *arrayUint64) Unmarshal(r *buffer.Buffer) error {
	length, err := readArrayHeader(r)
	if err != nil {
		return err
	}

	aa := (*a)[:0]

	type_, err := readType(r)
	if err != nil {
		return err
	}
	switch type_ {
	case TypeCodeUlong0:
		if int64(cap(aa)) < length {
			aa = make([]uint64, length)
		} else {
			aa = aa[:length]
			for i := range aa {
				aa[i] = 0
			}
		}
	case TypeCodeSmallUlong:
		buf, ok := r.Next(length)
		if !ok {
			return errors.New("invalid length")
		}

		if int64(cap(aa)) < length {
			aa = make([]uint64, length)
		} else {
			aa = aa[:length]
		}

		for i, n := range buf {
			aa[i] = uint64(n)
		}
	case TypeCodeUlong:
		const typeSize = 8
		buf, ok := r.Next(length * typeSize)
		if !ok {
			return errors.New("invalid length")
		}

		if int64(cap(aa)) < length {
			aa = make([]uint64, length)
		} else {
			aa = aa[:length]
		}

		var bufIdx int
		for i := range aa {
			aa[i] = binary.BigEndian.Uint64(buf[bufIdx : bufIdx+8])
			bufIdx += 8
		}
	default:
		return fmt.Errorf("invalid type for []uint64 %02x", type_)
	}

	*a = aa
	return nil
}

type arrayInt64 []int64

func (a arrayInt64) Marshal(wr *buffer.Buffer) error {
	var (
		typeSize = 1
		TypeCode = TypeCodeSmalllong
	)
	for _, n := range a {
		if n > math.MaxInt8 {
			typeSize = 8
			TypeCode = TypeCodeLong
			break
		}
	}

	writeArrayHeader(wr, len(a), typeSize, TypeCode)

	if TypeCode == TypeCodeLong {
		for _, element := range a {
			wr.AppendUint64(uint64(element))
		}
	} else {
		for _, element := range a {
			wr.AppendByte(byte(element))
		}
	}

	return nil
}

func (a *arrayInt64) Unmarshal(r *buffer.Buffer) error {
	length, err := readArrayHeader(r)
	if err != nil {
		return err
	}

	aa := (*a)[:0]

	type_, err := readType(r)
	if err != nil {
		return err
	}
	switch type_ {
	case TypeCodeSmalllong:
		buf, ok := r.Next(length)
		if !ok {
			return errors.New("invalid length")
		}

		if int64(cap(aa)) < length {
			aa = make([]int64, length)
		} else {
			aa = aa[:length]
		}

		for i, n := range buf {
			aa[i] = int64(int8(n))
		}
	case TypeCodeLong:
		const typeSize = 8
		buf, ok := r.Next(length * typeSize)
		if !ok {
			return errors.New("invalid length")
		}

		if int64(cap(aa)) < length {
			aa = make([]int64, length)
		} else {
			aa = aa[:length]
		}

		var bufIdx int
		for i := range aa {
			aa[i] = int64(binary.BigEndian.Uint64(buf[bufIdx:]))
			bufIdx += 8
		}
	default:
		return fmt.Errorf("invalid type for []uint64 %02x", type_)
	}

	*a = aa
	return nil
}

type arrayFloat []float32

func (a arrayFloat) Marshal(wr *buffer.Buffer) error {
	const typeSize = 4

	writeArrayHeader(wr, len(a), typeSize, TypeCodeFloat)

	for _, element := range a {
		wr.AppendUint32(math.Float32bits(element))
	}

	return nil
}

func (a *arrayFloat) Unmarshal(r *buffer.Buffer) error {
	length, err := readArrayHeader(r)
	if err != nil {
		return err
	}

	type_, err := readType(r)
	if err != nil {
		return err
	}
	if type_ != TypeCodeFloat {
		return fmt.Errorf("invalid type for []float32 %02x", type_)
	}

	const typeSize = 4
	buf, ok := r.Next(length * typeSize)
	if !ok {
		return fmt.Errorf("invalid length %d", length)
	}

	aa := (*a)[:0]
	if int64(cap(aa)) < length {
		aa = make([]float32, length)
	} else {
		aa = aa[:length]
	}

	var bufIdx int
	for i := range aa {
		bits := binary.BigEndian.Uint32(buf[bufIdx:])
		aa[i] = math.Float32frombits(bits)
		bufIdx += typeSize
	}

	*a = aa
	return nil
}

type arrayDouble []float64

func (a arrayDouble) Marshal(wr *buffer.Buffer) error {
	const typeSize = 8

	writeArrayHeader(wr, len(a), typeSize, TypeCodeDouble)

	for _, element := range a {
		wr.AppendUint64(math.Float64bits(element))
	}

	return nil
}

func (a *arrayDouble) Unmarshal(r *buffer.Buffer) error {
	length, err := readArrayHeader(r)
	if err != nil {
		return err
	}

	type_, err := readType(r)
	if err != nil {
		return err
	}
	if type_ != TypeCodeDouble {
		return fmt.Errorf("invalid type for []float64 %02x", type_)
	}

	const typeSize = 8
	buf, ok := r.Next(length * typeSize)
	if !ok {
		return fmt.Errorf("invalid length %d", length)
	}

	aa := (*a)[:0]
	if int64(cap(aa)) < length {
		aa = make([]float64, length)
	} else {
		aa = aa[:length]
	}

	var bufIdx int
	for i := range aa {
		bits := binary.BigEndian.Uint64(buf[bufIdx:])
		aa[i] = math.Float64frombits(bits)
		bufIdx += typeSize
	}

	*a = aa
	return nil
}

type arrayBool []bool

func (a arrayBool) Marshal(wr *buffer.Buffer) error {
	const typeSize = 1

	writeArrayHeader(wr, len(a), typeSize, TypeCodeBool)

	for _, element := range a {
		value := byte(0)
		if element {
			value = 1
		}
		wr.AppendByte(value)
	}

	return nil
}

func (a *arrayBool) Unmarshal(r *buffer.Buffer) error {
	length, err := readArrayHeader(r)
	if err != nil {
		return err
	}

	aa := (*a)[:0]
	if int64(cap(aa)) < length {
		aa = make([]bool, length)
	} else {
		aa = aa[:length]
	}

	type_, err := readType(r)
	if err != nil {
		return err
	}
	switch type_ {
	case TypeCodeBool:
		buf, ok := r.Next(length)
		if !ok {
			return errors.New("invalid length")
		}

		for i, value := range buf {
			if value == 0 {
				aa[i] = false
			} else {
				aa[i] = true
			}
		}

	case TypeCodeBoolTrue:
		for i := range aa {
			aa[i] = true
		}
	case TypeCodeBoolFalse:
		for i := range aa {
			aa[i] = false
		}
	default:
		return fmt.Errorf("invalid type for []bool %02x", type_)
	}

	*a = aa
	return nil
}

type arrayString []string

func (a arrayString) Marshal(wr *buffer.Buffer) error {
	var (
		elementType       = TypeCodeStr8
		elementsSizeTotal int
	)
	for _, element := range a {
		if !utf8.ValidString(element) {
			return errors.New("not a valid UTF-8 string")
		}

		elementsSizeTotal += len(element)

		if len(element) > math.MaxUint8 {
			elementType = TypeCodeStr32
		}
	}

	writeVariableArrayHeader(wr, len(a), elementsSizeTotal, elementType)

	if elementType == TypeCodeStr32 {
		for _, element := range a {
			wr.AppendUint32(uint32(len(element)))
			wr.AppendString(element)
		}
	} else {
		for _, element := range a {
			wr.AppendByte(byte(len(element)))
			wr.AppendString(element)
		}
	}

	return nil
}

func (a *arrayString) Unmarshal(r *buffer.Buffer) error {
	length, err := readArrayHeader(r)
	if err != nil {
		return err
	}

	const typeSize = 2 // assume all strings are at least 2 bytes
	if length*typeSize > int64(r.Len()) {
		return fmt.Errorf("invalid length %d", length)
	}

	aa := (*a)[:0]
	if int64(cap(aa)) < length {
		aa = make([]string, length)
	} else {
		aa = aa[:length]
	}

	type_, err := readType(r)
	if err != nil {
		return err
	}
	switch type_ {
	case TypeCodeStr8:
		for i := range aa {
			size, err := r.ReadByte()
			if err != nil {
				return err
			}

			buf, ok := r.Next(int64(size))
			if !ok {
				return errors.New("invalid length")
			}

			aa[i] = string(buf)
		}
	case TypeCodeStr32:
		for i := range aa {
			buf, ok := r.Next(4)
			if !ok {
				return errors.New("invalid length")
			}
			size := int64(binary.BigEndian.Uint32(buf))

			buf, ok = r.Next(size)
			if !ok {
				return errors.New("invalid length")
			}
			aa[i] = string(buf)
		}
	default:
		return fmt.Errorf("invalid type for []string %02x", type_)
	}

	*a = aa
	return nil
}

type arraySymbol []Symbol

func (a arraySymbol) Marshal(wr *buffer.Buffer) error {
	var (
		elementType       = TypeCodeSym8
		elementsSizeTotal int
	)
	for _, element := range a {
		elementsSizeTotal += len(element)

		if len(element) > math.MaxUint8 {
			elementType = TypeCodeSym32
		}
	}

	writeVariableArrayHeader(wr, len(a), elementsSizeTotal, elementType)

	if elementType == TypeCodeSym32 {
		for _, element := range a {
			wr.AppendUint32(uint32(len(element)))
			wr.AppendString(string(element))
		}
	} else {
		for _, element := range a {
			wr.AppendByte(byte(len(element)))
			wr.AppendString(string(element))
		}
	}

	return nil
}

func (a *arraySymbol) Unmarshal(r *buffer.Buffer) error {
	length, err := readArrayHeader(r)
	if err != nil {
		return err
	}

	const typeSize = 2 // assume all symbols are at least 2 bytes
	if length*typeSize > int64(r.Len()) {
		return fmt.Errorf("invalid length %d", length)
	}

	aa := (*a)[:0]
	if int64(cap(aa)) < length {
		aa = make([]Symbol, length)
	} else {
		aa = aa[:length]
	}

	type_, err := readType(r)
	if err != nil {
		return err
	}
	switch type_ {
	case TypeCodeSym8:
		for i := range aa {
			size, err := r.ReadByte()
			if err != nil {
				return err
			}

			buf, ok := r.Next(int64(size))
			if !ok {
				return errors.New("invalid length")
			}
			aa[i] = Symbol(buf)
		}
	case TypeCodeSym32:
		for i := range aa {
			buf, ok := r.Next(4)
			if !ok {
				return errors.New("invalid length")
			}
			size := int64(binary.BigEndian.Uint32(buf))

			buf, ok = r.Next(size)
			if !ok {
				return errors.New("invalid length")
			}
			aa[i] = Symbol(buf)
		}
	default:
		return fmt.Errorf("invalid type for []Symbol %02x", type_)
	}

	*a = aa
	return nil
}

type arrayBinary [][]byte

func (a arrayBinary) Marshal(wr *buffer.Buffer) error {
	var (
		elementType       = TypeCodeVbin8
		elementsSizeTotal int
	)
	for _, element := range a {
		elementsSizeTotal += len(element)

		if len(element) > math.MaxUint8 {
			elementType = TypeCodeVbin32
		}
	}

	writeVariableArrayHeader(wr, len(a), elementsSizeTotal, elementType)

	if elementType == TypeCodeVbin32 {
		for _, element := range a {
			wr.AppendUint32(uint32(len(element)))
			wr.Append(element)
		}
	} else {
		for _, element := range a {
			wr.AppendByte(byte(len(element)))
			wr.Append(element)
		}
	}

	return nil
}

func (a *arrayBinary) Unmarshal(r *buffer.Buffer) error {
	length, err := readArrayHeader(r)
	if err != nil {
		return err
	}

	const typeSize = 2 // assume all binary is at least 2 bytes
	if length*typeSize > int64(r.Len()) {
		return fmt.Errorf("invalid length %d", length)
	}

	aa := (*a)[:0]
	if int64(cap(aa)) < length {
		aa = make([][]byte, length)
	} else {
		aa = aa[:length]
	}

	type_, err := readType(r)
	if err != nil {
		return err
	}
	switch type_ {
	case TypeCodeVbin8:
		for i := range aa {
			size, err := r.ReadByte()
			if err != nil {
				return err
			}

			buf, ok := r.Next(int64(size))
			if !ok {
				return fmt.Errorf("invalid length %d", length)
			}
			aa[i] = append([]byte(nil), buf...)
		}
	case TypeCodeVbin32:
		for i := range aa {
			buf, ok := r.Next(4)
			if !ok {
				return errors.New("invalid length")
			}
			size := binary.BigEndian.Uint32(buf)

			buf, ok = r.Next(int64(size))
			if !ok {
				return errors.New("invalid length")
			}
			aa[i] = append([]byte(nil), buf...)
		}
	default:
		return fmt.Errorf("invalid type for [][]byte %02x", type_)
	}

	*a = aa
	return nil
}

type arrayTimestamp []time.Time

func (a arrayTimestamp) Marshal(wr *buffer.Buffer) error {
	const typeSize = 8

	writeArrayHeader(wr, len(a), typeSize, TypeCodeTimestamp)

	for _, element := range a {
		ms := element.UnixNano() / int64(time.Millisecond)
		wr.AppendUint64(uint64(ms))
	}

	return nil
}

func (a *arrayTimestamp) Unmarshal(r *buffer.Buffer) error {
	length, err := readArrayHeader(r)
	if err != nil {
		return err
	}

	type_, err := readType(r)
	if err != nil {
		return err
	}
	if type_ != TypeCodeTimestamp {
		return fmt.Errorf("invalid type for []time.Time %02x", type_)
	}

	const typeSize = 8
	buf, ok := r.Next(length * typeSize)
	if !ok {
		return fmt.Errorf("invalid length %d", length)
	}

	aa := (*a)[:0]
	if int64(cap(aa)) < length {
		aa = make([]time.Time, length)
	} else {
		aa = aa[:length]
	}

	var bufIdx int
	for i := range aa {
		ms := int64(binary.BigEndian.Uint64(buf[bufIdx:]))
		bufIdx += typeSize
		aa[i] = time.Unix(ms/1000, (ms%1000)*1000000).UTC()
	}

	*a = aa
	return nil
}

type arrayUUID []UUID

func (a arrayUUID) Marshal(wr *buffer.Buffer) error {
	const typeSize = 16

	writeArrayHeader(wr, len(a), typeSize, TypeCodeUUID)

	for _, element := range a {
		wr.Append(element[:])
	}

	return nil
}

func (a *arrayUUID) Unmarshal(r *buffer.Buffer) error {
	length, err := readArrayHeader(r)
	if err != nil {
		return err
	}

	type_, err := readType(r)
	if err != nil {
		return err
	}
	if type_ != TypeCodeUUID {
		return fmt.Errorf("invalid type for []UUID %#02x", type_)
	}

	const typeSize = 16
	buf, ok := r.Next(length * typeSize)
	if !ok {
		return fmt.Errorf("invalid length %d", length)
	}

	aa := (*a)[:0]
	if int64(cap(aa)) < length {
		aa = make([]UUID, length)
	} else {
		aa = aa[:length]
	}

	var bufIdx int
	for i := range aa {
		copy(aa[i][:], buf[bufIdx:bufIdx+16])
		bufIdx += 16
	}

	*a = aa
	return nil
}

// LIST

type list []any

func (l list) Marshal(wr *buffer.Buffer) error {
	length := len(l)

	// type
	if length == 0 {
		wr.AppendByte(byte(TypeCodeList0))
		return nil
	}
	wr.AppendByte(byte(TypeCodeList32))

	// size
	sizeIdx := wr.Len()
	wr.Append([]byte{0, 0, 0, 0})

	// length
	wr.AppendUint32(uint32(length))

	for _, element := range l {
		err := Marshal(wr, element)
		if err != nil {
			return err
		}
	}

	// overwrite size
	binary.BigEndian.PutUint32(wr.Bytes()[sizeIdx:], uint32(wr.Len()-(sizeIdx+4)))

	return nil
}

func (l *list) Unmarshal(r *buffer.Buffer) error {
	length, err := readListHeader(r)
	if err != nil {
		return err
	}

	// assume that all types are at least 1 byte
	if length > int64(r.Len()) {
		return fmt.Errorf("invalid length %d", length)
	}

	ll := *l
	if int64(cap(ll)) < length {
		ll = make([]any, length)
	} else {
		ll = ll[:length]
	}

	for i := range ll {
		ll[i], err = ReadAny(r)
		if err != nil {
			return err
		}
	}

	*l = ll
	return nil
}

// multiSymbol can decode a single symbol or an array.
type MultiSymbol []Symbol

func (ms MultiSymbol) Marshal(wr *buffer.Buffer) error {
	return Marshal(wr, []Symbol(ms))
}

func (ms *MultiSymbol) Unmarshal(r *buffer.Buffer) error {
	type_, err := peekType(r)
	if err != nil {
		return err
	}

	if type_ == TypeCodeSym8 || type_ == TypeCodeSym32 {
		s, err := ReadString(r)
		if err != nil {
			return err
		}

		*ms = []Symbol{Symbol(s)}
		return nil
	}

	return Unmarshal(r, (*[]Symbol)(ms))
}

type arrayMap []map[any]any

func (a arrayMap) Marshal(wr *buffer.Buffer) error {
	// type
	wr.AppendByte(byte(TypeCodeArray32))

	// array size placeholder
	sizeIdx := wr.Len()
	wr.Append([]byte{0, 0, 0, 0})

	// count
	wr.AppendUint32(uint32(len(a)))

	// array element type
	wr.AppendByte(byte(TypeCodeMap32))

	// marshal each map (without the type code)
	for _, element := range a {
		if err := writeMap32(wr, element); err != nil {
			return err
		}
	}

	// overwrite array size
	binary.BigEndian.PutUint32(wr.Bytes()[sizeIdx:], uint32(wr.Len()-(sizeIdx+4)))

	return nil
}

func (a *arrayMap) Unmarshal(r *buffer.Buffer) error {
	length, err := readArrayHeader(r)
	if err != nil {
		return err
	}

	type_, err := readType(r)
	if err != nil {
		return err
	}
	if type_ != TypeCodeMap8 && type_ != TypeCodeMap32 {
		return fmt.Errorf("invalid type for []map[any]any %02x", type_)
	}

	aa := (*a)[:0]
	if int64(cap(aa)) < length {
		aa = make([]map[any]any, length)
	} else {
		aa = aa[:length]
	}

	for i := range aa {
		var value any
		switch type_ {
		case TypeCodeMap8:
			value, err = readMap8(r)
		case TypeCodeMap32:
			value, err = readMap32(r)
		}

		if err != nil {
			return err
		}

		if m, ok := value.(map[any]any); ok {
			aa[i] = m
		} else if m, ok := value.(map[string]any); ok {
			// convert to map[any]any
			anyMap := make(map[any]any, len(m))
			for k, v := range m {
				anyMap[k] = v
			}
			aa[i] = anyMap
		} else {
			return fmt.Errorf("unexpected map type: %T", value)
		}
	}

	*a = aa
	return nil
}
