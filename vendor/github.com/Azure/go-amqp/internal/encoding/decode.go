package encoding

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/Azure/go-amqp/internal/buffer"
)

// unmarshaler is fulfilled by types that can unmarshal
// themselves from AMQP data.
type unmarshaler interface {
	Unmarshal(r *buffer.Buffer) error
}

// unmarshal decodes AMQP encoded data into i.
//
// The decoding method is based on the type of i.
//
// If i implements unmarshaler, i.Unmarshal() will be called.
//
// Pointers to primitive types will be decoded via the appropriate read[Type] function.
//
// If i is a pointer to a pointer (**Type), it will be dereferenced and a new instance
// of (*Type) is allocated via reflection.
//
// Common map types (map[string]string, map[Symbol]interface{}, and
// map[interface{}]interface{}), will be decoded via conversion to the mapStringAny,
// mapSymbolAny, and mapAnyAny types.
func Unmarshal(r *buffer.Buffer, i interface{}) error {
	if tryReadNull(r) {
		return nil
	}

	switch t := i.(type) {
	case *int:
		val, err := readInt(r)
		if err != nil {
			return err
		}
		*t = val
	case *int8:
		val, err := readSbyte(r)
		if err != nil {
			return err
		}
		*t = val
	case *int16:
		val, err := readShort(r)
		if err != nil {
			return err
		}
		*t = val
	case *int32:
		val, err := readInt32(r)
		if err != nil {
			return err
		}
		*t = val
	case *int64:
		val, err := readLong(r)
		if err != nil {
			return err
		}
		*t = val
	case *uint64:
		val, err := readUlong(r)
		if err != nil {
			return err
		}
		*t = val
	case *uint32:
		val, err := readUint32(r)
		if err != nil {
			return err
		}
		*t = val
	case **uint32: // fastpath for uint32 pointer fields
		val, err := readUint32(r)
		if err != nil {
			return err
		}
		*t = &val
	case *uint16:
		val, err := readUshort(r)
		if err != nil {
			return err
		}
		*t = val
	case *uint8:
		val, err := ReadUbyte(r)
		if err != nil {
			return err
		}
		*t = val
	case *float32:
		val, err := readFloat(r)
		if err != nil {
			return err
		}
		*t = val
	case *float64:
		val, err := readDouble(r)
		if err != nil {
			return err
		}
		*t = val
	case *string:
		val, err := ReadString(r)
		if err != nil {
			return err
		}
		*t = val
	case *Symbol:
		s, err := ReadString(r)
		if err != nil {
			return err
		}
		*t = Symbol(s)
	case *[]byte:
		val, err := readBinary(r)
		if err != nil {
			return err
		}
		*t = val
	case *bool:
		b, err := readBool(r)
		if err != nil {
			return err
		}
		*t = b
	case *time.Time:
		ts, err := readTimestamp(r)
		if err != nil {
			return err
		}
		*t = ts
	case *[]int8:
		return (*arrayInt8)(t).Unmarshal(r)
	case *[]uint16:
		return (*arrayUint16)(t).Unmarshal(r)
	case *[]int16:
		return (*arrayInt16)(t).Unmarshal(r)
	case *[]uint32:
		return (*arrayUint32)(t).Unmarshal(r)
	case *[]int32:
		return (*arrayInt32)(t).Unmarshal(r)
	case *[]uint64:
		return (*arrayUint64)(t).Unmarshal(r)
	case *[]int64:
		return (*arrayInt64)(t).Unmarshal(r)
	case *[]float32:
		return (*arrayFloat)(t).Unmarshal(r)
	case *[]float64:
		return (*arrayDouble)(t).Unmarshal(r)
	case *[]bool:
		return (*arrayBool)(t).Unmarshal(r)
	case *[]string:
		return (*arrayString)(t).Unmarshal(r)
	case *[]Symbol:
		return (*arraySymbol)(t).Unmarshal(r)
	case *[][]byte:
		return (*arrayBinary)(t).Unmarshal(r)
	case *[]time.Time:
		return (*arrayTimestamp)(t).Unmarshal(r)
	case *[]UUID:
		return (*arrayUUID)(t).Unmarshal(r)
	case *[]interface{}:
		return (*list)(t).Unmarshal(r)
	case *map[interface{}]interface{}:
		return (*mapAnyAny)(t).Unmarshal(r)
	case *map[string]interface{}:
		return (*mapStringAny)(t).Unmarshal(r)
	case *map[Symbol]interface{}:
		return (*mapSymbolAny)(t).Unmarshal(r)
	case *DeliveryState:
		type_, _, err := PeekMessageType(r.Bytes())
		if err != nil {
			return err
		}

		switch AMQPType(type_) {
		case TypeCodeStateAccepted:
			*t = new(StateAccepted)
		case TypeCodeStateModified:
			*t = new(StateModified)
		case TypeCodeStateReceived:
			*t = new(StateReceived)
		case TypeCodeStateRejected:
			*t = new(StateRejected)
		case TypeCodeStateReleased:
			*t = new(StateReleased)
		default:
			return fmt.Errorf("unexpected type %d for deliveryState", type_)
		}
		return Unmarshal(r, *t)

	case *interface{}:
		v, err := ReadAny(r)
		if err != nil {
			return err
		}
		*t = v

	case unmarshaler:
		return t.Unmarshal(r)
	default:
		// handle **T
		v := reflect.Indirect(reflect.ValueOf(i))

		// can't unmarshal into a non-pointer
		if v.Kind() != reflect.Ptr {
			return fmt.Errorf("unable to unmarshal %T", i)
		}

		// if nil pointer, allocate a new value to
		// unmarshal into
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}

		return Unmarshal(r, v.Interface())
	}
	return nil
}

// unmarshalComposite is a helper for use in a composite's unmarshal() function.
//
// The composite from r will be unmarshaled into zero or more fields. An error
// will be returned if typ does not match the decoded type.
func UnmarshalComposite(r *buffer.Buffer, type_ AMQPType, fields ...UnmarshalField) error {
	cType, numFields, err := readCompositeHeader(r)
	if err != nil {
		return err
	}

	// check type matches expectation
	if cType != type_ {
		return fmt.Errorf("invalid header %#0x for %#0x", cType, type_)
	}

	// Validate the field count is less than or equal to the number of fields
	// provided. Fields may be omitted by the sender if they are not set.
	if numFields > int64(len(fields)) {
		return fmt.Errorf("invalid field count %d for %#0x", numFields, type_)
	}

	for i, field := range fields[:numFields] {
		// If the field is null and handleNull is set, call it.
		if tryReadNull(r) {
			if field.HandleNull != nil {
				err = field.HandleNull()
				if err != nil {
					return err
				}
			}
			continue
		}

		// Unmarshal each of the received fields.
		err = Unmarshal(r, field.Field)
		if err != nil {
			return fmt.Errorf("unmarshaling field %d: %v", i, err)
		}
	}

	// check and call handleNull for the remaining fields
	for _, field := range fields[numFields:] {
		if field.HandleNull != nil {
			err = field.HandleNull()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// unmarshalField is a struct that contains a field to be unmarshaled into.
//
// An optional nullHandler can be set. If the composite field being unmarshaled
// is null and handleNull is not nil, nullHandler will be called.
type UnmarshalField struct {
	Field      interface{}
	HandleNull NullHandler
}

// nullHandler is a function to be called when a composite's field
// is null.
type NullHandler func() error

func readType(r *buffer.Buffer) (AMQPType, error) {
	n, err := r.ReadByte()
	return AMQPType(n), err
}

func peekType(r *buffer.Buffer) (AMQPType, error) {
	n, err := r.PeekByte()
	return AMQPType(n), err
}

// readCompositeHeader reads and consumes the composite header from r.
func readCompositeHeader(r *buffer.Buffer) (_ AMQPType, fields int64, _ error) {
	type_, err := readType(r)
	if err != nil {
		return 0, 0, err
	}

	// compsites always start with 0x0
	if type_ != 0 {
		return 0, 0, fmt.Errorf("invalid composite header %#02x", type_)
	}

	// next, the composite type is encoded as an AMQP uint8
	v, err := readUlong(r)
	if err != nil {
		return 0, 0, err
	}

	// fields are represented as a list
	fields, err = readListHeader(r)

	return AMQPType(v), fields, err
}

func readListHeader(r *buffer.Buffer) (length int64, _ error) {
	type_, err := readType(r)
	if err != nil {
		return 0, err
	}

	listLength := r.Len()

	switch type_ {
	case TypeCodeList0:
		return 0, nil
	case TypeCodeList8:
		buf, ok := r.Next(2)
		if !ok {
			return 0, errors.New("invalid length")
		}
		_ = buf[1]

		size := int(buf[0])
		if size > listLength-1 {
			return 0, errors.New("invalid length")
		}
		length = int64(buf[1])
	case TypeCodeList32:
		buf, ok := r.Next(8)
		if !ok {
			return 0, errors.New("invalid length")
		}
		_ = buf[7]

		size := int(binary.BigEndian.Uint32(buf[:4]))
		if size > listLength-4 {
			return 0, errors.New("invalid length")
		}
		length = int64(binary.BigEndian.Uint32(buf[4:8]))
	default:
		return 0, fmt.Errorf("type code %#02x is not a recognized list type", type_)
	}

	return length, nil
}

func readArrayHeader(r *buffer.Buffer) (length int64, _ error) {
	type_, err := readType(r)
	if err != nil {
		return 0, err
	}

	arrayLength := r.Len()

	switch type_ {
	case TypeCodeArray8:
		buf, ok := r.Next(2)
		if !ok {
			return 0, errors.New("invalid length")
		}
		_ = buf[1]

		size := int(buf[0])
		if size > arrayLength-1 {
			return 0, errors.New("invalid length")
		}
		length = int64(buf[1])
	case TypeCodeArray32:
		buf, ok := r.Next(8)
		if !ok {
			return 0, errors.New("invalid length")
		}
		_ = buf[7]

		size := binary.BigEndian.Uint32(buf[:4])
		if int(size) > arrayLength-4 {
			return 0, fmt.Errorf("invalid length for type %02x", type_)
		}
		length = int64(binary.BigEndian.Uint32(buf[4:8]))
	default:
		return 0, fmt.Errorf("type code %#02x is not a recognized array type", type_)
	}
	return length, nil
}

func ReadString(r *buffer.Buffer) (string, error) {
	type_, err := readType(r)
	if err != nil {
		return "", err
	}

	var length int64
	switch type_ {
	case TypeCodeStr8, TypeCodeSym8:
		n, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		length = int64(n)
	case TypeCodeStr32, TypeCodeSym32:
		buf, ok := r.Next(4)
		if !ok {
			return "", fmt.Errorf("invalid length for type %#02x", type_)
		}
		length = int64(binary.BigEndian.Uint32(buf))
	default:
		return "", fmt.Errorf("type code %#02x is not a recognized string type", type_)
	}

	buf, ok := r.Next(length)
	if !ok {
		return "", errors.New("invalid length")
	}
	return string(buf), nil
}

func readBinary(r *buffer.Buffer) ([]byte, error) {
	type_, err := readType(r)
	if err != nil {
		return nil, err
	}

	var length int64
	switch type_ {
	case TypeCodeVbin8:
		n, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		length = int64(n)
	case TypeCodeVbin32:
		buf, ok := r.Next(4)
		if !ok {
			return nil, fmt.Errorf("invalid length for type %#02x", type_)
		}
		length = int64(binary.BigEndian.Uint32(buf))
	default:
		return nil, fmt.Errorf("type code %#02x is not a recognized binary type", type_)
	}

	if length == 0 {
		// An empty value and a nil value are distinct,
		// ensure that the returned value is not nil in this case.
		return make([]byte, 0), nil
	}

	buf, ok := r.Next(length)
	if !ok {
		return nil, errors.New("invalid length")
	}
	return append([]byte(nil), buf...), nil
}

func ReadAny(r *buffer.Buffer) (interface{}, error) {
	if tryReadNull(r) {
		return nil, nil
	}

	type_, err := peekType(r)
	if err != nil {
		return nil, errors.New("invalid length")
	}

	switch type_ {
	// composite
	case 0x0:
		return readComposite(r)

	// bool
	case TypeCodeBool, TypeCodeBoolTrue, TypeCodeBoolFalse:
		return readBool(r)

	// uint
	case TypeCodeUbyte:
		return ReadUbyte(r)
	case TypeCodeUshort:
		return readUshort(r)
	case TypeCodeUint,
		TypeCodeSmallUint,
		TypeCodeUint0:
		return readUint32(r)
	case TypeCodeUlong,
		TypeCodeSmallUlong,
		TypeCodeUlong0:
		return readUlong(r)

	// int
	case TypeCodeByte:
		return readSbyte(r)
	case TypeCodeShort:
		return readShort(r)
	case TypeCodeInt,
		TypeCodeSmallint:
		return readInt32(r)
	case TypeCodeLong,
		TypeCodeSmalllong:
		return readLong(r)

	// floating point
	case TypeCodeFloat:
		return readFloat(r)
	case TypeCodeDouble:
		return readDouble(r)

	// binary
	case TypeCodeVbin8, TypeCodeVbin32:
		return readBinary(r)

	// strings
	case TypeCodeStr8, TypeCodeStr32:
		return ReadString(r)
	case TypeCodeSym8, TypeCodeSym32:
		// symbols currently decoded as string to avoid
		// exposing symbol type in message, this may need
		// to change if users need to distinguish strings
		// from symbols
		return ReadString(r)

	// timestamp
	case TypeCodeTimestamp:
		return readTimestamp(r)

	// UUID
	case TypeCodeUUID:
		return readUUID(r)

	// arrays
	case TypeCodeArray8, TypeCodeArray32:
		return readAnyArray(r)

	// lists
	case TypeCodeList0, TypeCodeList8, TypeCodeList32:
		return readAnyList(r)

	// maps
	case TypeCodeMap8:
		return readAnyMap(r)
	case TypeCodeMap32:
		return readAnyMap(r)

	// TODO: implement
	case TypeCodeDecimal32:
		return nil, errors.New("decimal32 not implemented")
	case TypeCodeDecimal64:
		return nil, errors.New("decimal64 not implemented")
	case TypeCodeDecimal128:
		return nil, errors.New("decimal128 not implemented")
	case TypeCodeChar:
		return nil, errors.New("char not implemented")
	default:
		return nil, fmt.Errorf("unknown type %#02x", type_)
	}
}

func readAnyMap(r *buffer.Buffer) (interface{}, error) {
	var m map[interface{}]interface{}
	err := (*mapAnyAny)(&m).Unmarshal(r)
	if err != nil {
		return nil, err
	}

	if len(m) == 0 {
		return m, nil
	}

	stringKeys := true
Loop:
	for key := range m {
		switch key.(type) {
		case string:
		case Symbol:
		default:
			stringKeys = false
			break Loop
		}
	}

	if stringKeys {
		mm := make(map[string]interface{}, len(m))
		for key, value := range m {
			switch key := key.(type) {
			case string:
				mm[key] = value
			case Symbol:
				mm[string(key)] = value
			}
		}
		return mm, nil
	}

	return m, nil
}

func readAnyList(r *buffer.Buffer) (interface{}, error) {
	var a []interface{}
	err := (*list)(&a).Unmarshal(r)
	return a, err
}

func readAnyArray(r *buffer.Buffer) (interface{}, error) {
	// get the array type
	buf := r.Bytes()
	if len(buf) < 1 {
		return nil, errors.New("invalid length")
	}

	var typeIdx int
	switch AMQPType(buf[0]) {
	case TypeCodeArray8:
		typeIdx = 3
	case TypeCodeArray32:
		typeIdx = 9
	default:
		return nil, fmt.Errorf("invalid array type %02x", buf[0])
	}
	if len(buf) < typeIdx+1 {
		return nil, errors.New("invalid length")
	}

	switch AMQPType(buf[typeIdx]) {
	case TypeCodeByte:
		var a []int8
		err := (*arrayInt8)(&a).Unmarshal(r)
		return a, err
	case TypeCodeUbyte:
		var a ArrayUByte
		err := a.Unmarshal(r)
		return a, err
	case TypeCodeUshort:
		var a []uint16
		err := (*arrayUint16)(&a).Unmarshal(r)
		return a, err
	case TypeCodeShort:
		var a []int16
		err := (*arrayInt16)(&a).Unmarshal(r)
		return a, err
	case TypeCodeUint0, TypeCodeSmallUint, TypeCodeUint:
		var a []uint32
		err := (*arrayUint32)(&a).Unmarshal(r)
		return a, err
	case TypeCodeSmallint, TypeCodeInt:
		var a []int32
		err := (*arrayInt32)(&a).Unmarshal(r)
		return a, err
	case TypeCodeUlong0, TypeCodeSmallUlong, TypeCodeUlong:
		var a []uint64
		err := (*arrayUint64)(&a).Unmarshal(r)
		return a, err
	case TypeCodeSmalllong, TypeCodeLong:
		var a []int64
		err := (*arrayInt64)(&a).Unmarshal(r)
		return a, err
	case TypeCodeFloat:
		var a []float32
		err := (*arrayFloat)(&a).Unmarshal(r)
		return a, err
	case TypeCodeDouble:
		var a []float64
		err := (*arrayDouble)(&a).Unmarshal(r)
		return a, err
	case TypeCodeBool, TypeCodeBoolTrue, TypeCodeBoolFalse:
		var a []bool
		err := (*arrayBool)(&a).Unmarshal(r)
		return a, err
	case TypeCodeStr8, TypeCodeStr32:
		var a []string
		err := (*arrayString)(&a).Unmarshal(r)
		return a, err
	case TypeCodeSym8, TypeCodeSym32:
		var a []Symbol
		err := (*arraySymbol)(&a).Unmarshal(r)
		return a, err
	case TypeCodeVbin8, TypeCodeVbin32:
		var a [][]byte
		err := (*arrayBinary)(&a).Unmarshal(r)
		return a, err
	case TypeCodeTimestamp:
		var a []time.Time
		err := (*arrayTimestamp)(&a).Unmarshal(r)
		return a, err
	case TypeCodeUUID:
		var a []UUID
		err := (*arrayUUID)(&a).Unmarshal(r)
		return a, err
	default:
		return nil, fmt.Errorf("array decoding not implemented for %#02x", buf[typeIdx])
	}
}

func readComposite(r *buffer.Buffer) (interface{}, error) {
	buf := r.Bytes()

	if len(buf) < 2 {
		return nil, errors.New("invalid length for composite")
	}

	// compsites start with 0x0
	if AMQPType(buf[0]) != 0x0 {
		return nil, fmt.Errorf("invalid composite header %#02x", buf[0])
	}

	var compositeType uint64
	switch AMQPType(buf[1]) {
	case TypeCodeSmallUlong:
		if len(buf) < 3 {
			return nil, errors.New("invalid length for smallulong")
		}
		compositeType = uint64(buf[2])
	case TypeCodeUlong:
		if len(buf) < 10 {
			return nil, errors.New("invalid length for ulong")
		}
		compositeType = binary.BigEndian.Uint64(buf[2:])
	}

	if compositeType > math.MaxUint8 {
		// try as described type
		var dt DescribedType
		err := dt.Unmarshal(r)
		return dt, err
	}

	switch AMQPType(compositeType) {
	// Error
	case TypeCodeError:
		t := new(Error)
		err := t.Unmarshal(r)
		return t, err

	// Lifetime Policies
	case TypeCodeDeleteOnClose:
		t := DeleteOnClose
		err := t.Unmarshal(r)
		return t, err
	case TypeCodeDeleteOnNoMessages:
		t := DeleteOnNoMessages
		err := t.Unmarshal(r)
		return t, err
	case TypeCodeDeleteOnNoLinks:
		t := DeleteOnNoLinks
		err := t.Unmarshal(r)
		return t, err
	case TypeCodeDeleteOnNoLinksOrMessages:
		t := DeleteOnNoLinksOrMessages
		err := t.Unmarshal(r)
		return t, err

	// Delivery States
	case TypeCodeStateAccepted:
		t := new(StateAccepted)
		err := t.Unmarshal(r)
		return t, err
	case TypeCodeStateModified:
		t := new(StateModified)
		err := t.Unmarshal(r)
		return t, err
	case TypeCodeStateReceived:
		t := new(StateReceived)
		err := t.Unmarshal(r)
		return t, err
	case TypeCodeStateRejected:
		t := new(StateRejected)
		err := t.Unmarshal(r)
		return t, err
	case TypeCodeStateReleased:
		t := new(StateReleased)
		err := t.Unmarshal(r)
		return t, err

	case TypeCodeOpen,
		TypeCodeBegin,
		TypeCodeAttach,
		TypeCodeFlow,
		TypeCodeTransfer,
		TypeCodeDisposition,
		TypeCodeDetach,
		TypeCodeEnd,
		TypeCodeClose,
		TypeCodeSource,
		TypeCodeTarget,
		TypeCodeMessageHeader,
		TypeCodeDeliveryAnnotations,
		TypeCodeMessageAnnotations,
		TypeCodeMessageProperties,
		TypeCodeApplicationProperties,
		TypeCodeApplicationData,
		TypeCodeAMQPSequence,
		TypeCodeAMQPValue,
		TypeCodeFooter,
		TypeCodeSASLMechanism,
		TypeCodeSASLInit,
		TypeCodeSASLChallenge,
		TypeCodeSASLResponse,
		TypeCodeSASLOutcome:
		return nil, fmt.Errorf("readComposite unmarshal not implemented for %#02x", compositeType)

	default:
		// try as described type
		var dt DescribedType
		err := dt.Unmarshal(r)
		return dt, err
	}
}

func readTimestamp(r *buffer.Buffer) (time.Time, error) {
	type_, err := readType(r)
	if err != nil {
		return time.Time{}, err
	}

	if type_ != TypeCodeTimestamp {
		return time.Time{}, fmt.Errorf("invalid type for timestamp %02x", type_)
	}

	n, err := r.ReadUint64()
	ms := int64(n)
	return time.Unix(ms/1000, (ms%1000)*1000000).UTC(), err
}

func readInt(r *buffer.Buffer) (int, error) {
	type_, err := peekType(r)
	if err != nil {
		return 0, err
	}

	switch type_ {
	// Unsigned
	case TypeCodeUbyte:
		n, err := ReadUbyte(r)
		return int(n), err
	case TypeCodeUshort:
		n, err := readUshort(r)
		return int(n), err
	case TypeCodeUint0, TypeCodeSmallUint, TypeCodeUint:
		n, err := readUint32(r)
		return int(n), err
	case TypeCodeUlong0, TypeCodeSmallUlong, TypeCodeUlong:
		n, err := readUlong(r)
		return int(n), err

	// Signed
	case TypeCodeByte:
		n, err := readSbyte(r)
		return int(n), err
	case TypeCodeShort:
		n, err := readShort(r)
		return int(n), err
	case TypeCodeSmallint, TypeCodeInt:
		n, err := readInt32(r)
		return int(n), err
	case TypeCodeSmalllong, TypeCodeLong:
		n, err := readLong(r)
		return int(n), err
	default:
		return 0, fmt.Errorf("type code %#02x is not a recognized number type", type_)
	}
}

func readLong(r *buffer.Buffer) (int64, error) {
	type_, err := readType(r)
	if err != nil {
		return 0, err
	}

	switch type_ {
	case TypeCodeSmalllong:
		n, err := r.ReadByte()
		return int64(n), err
	case TypeCodeLong:
		n, err := r.ReadUint64()
		return int64(n), err
	default:
		return 0, fmt.Errorf("invalid type for uint32 %02x", type_)
	}
}

func readInt32(r *buffer.Buffer) (int32, error) {
	type_, err := readType(r)
	if err != nil {
		return 0, err
	}

	switch type_ {
	case TypeCodeSmallint:
		n, err := r.ReadByte()
		return int32(n), err
	case TypeCodeInt:
		n, err := r.ReadUint32()
		return int32(n), err
	default:
		return 0, fmt.Errorf("invalid type for int32 %02x", type_)
	}
}

func readShort(r *buffer.Buffer) (int16, error) {
	type_, err := readType(r)
	if err != nil {
		return 0, err
	}

	if type_ != TypeCodeShort {
		return 0, fmt.Errorf("invalid type for short %02x", type_)
	}

	n, err := r.ReadUint16()
	return int16(n), err
}

func readSbyte(r *buffer.Buffer) (int8, error) {
	type_, err := readType(r)
	if err != nil {
		return 0, err
	}

	if type_ != TypeCodeByte {
		return 0, fmt.Errorf("invalid type for int8 %02x", type_)
	}

	n, err := r.ReadByte()
	return int8(n), err
}

func ReadUbyte(r *buffer.Buffer) (uint8, error) {
	type_, err := readType(r)
	if err != nil {
		return 0, err
	}

	if type_ != TypeCodeUbyte {
		return 0, fmt.Errorf("invalid type for ubyte %02x", type_)
	}

	return r.ReadByte()
}

func readUshort(r *buffer.Buffer) (uint16, error) {
	type_, err := readType(r)
	if err != nil {
		return 0, err
	}

	if type_ != TypeCodeUshort {
		return 0, fmt.Errorf("invalid type for ushort %02x", type_)
	}

	return r.ReadUint16()
}

func readUint32(r *buffer.Buffer) (uint32, error) {
	type_, err := readType(r)
	if err != nil {
		return 0, err
	}

	switch type_ {
	case TypeCodeUint0:
		return 0, nil
	case TypeCodeSmallUint:
		n, err := r.ReadByte()
		return uint32(n), err
	case TypeCodeUint:
		return r.ReadUint32()
	default:
		return 0, fmt.Errorf("invalid type for uint32 %02x", type_)
	}
}

func readUlong(r *buffer.Buffer) (uint64, error) {
	type_, err := readType(r)
	if err != nil {
		return 0, err
	}

	switch type_ {
	case TypeCodeUlong0:
		return 0, nil
	case TypeCodeSmallUlong:
		n, err := r.ReadByte()
		return uint64(n), err
	case TypeCodeUlong:
		return r.ReadUint64()
	default:
		return 0, fmt.Errorf("invalid type for uint32 %02x", type_)
	}
}

func readFloat(r *buffer.Buffer) (float32, error) {
	type_, err := readType(r)
	if err != nil {
		return 0, err
	}

	if type_ != TypeCodeFloat {
		return 0, fmt.Errorf("invalid type for float32 %02x", type_)
	}

	bits, err := r.ReadUint32()
	return math.Float32frombits(bits), err
}

func readDouble(r *buffer.Buffer) (float64, error) {
	type_, err := readType(r)
	if err != nil {
		return 0, err
	}

	if type_ != TypeCodeDouble {
		return 0, fmt.Errorf("invalid type for float64 %02x", type_)
	}

	bits, err := r.ReadUint64()
	return math.Float64frombits(bits), err
}

func readBool(r *buffer.Buffer) (bool, error) {
	type_, err := readType(r)
	if err != nil {
		return false, err
	}

	switch type_ {
	case TypeCodeBool:
		b, err := r.ReadByte()
		return b != 0, err
	case TypeCodeBoolTrue:
		return true, nil
	case TypeCodeBoolFalse:
		return false, nil
	default:
		return false, fmt.Errorf("type code %#02x is not a recognized bool type", type_)
	}
}

func readUint(r *buffer.Buffer) (value uint64, _ error) {
	type_, err := readType(r)
	if err != nil {
		return 0, err
	}

	switch type_ {
	case TypeCodeUint0, TypeCodeUlong0:
		return 0, nil
	case TypeCodeUbyte, TypeCodeSmallUint, TypeCodeSmallUlong:
		n, err := r.ReadByte()
		return uint64(n), err
	case TypeCodeUshort:
		n, err := r.ReadUint16()
		return uint64(n), err
	case TypeCodeUint:
		n, err := r.ReadUint32()
		return uint64(n), err
	case TypeCodeUlong:
		return r.ReadUint64()
	default:
		return 0, fmt.Errorf("type code %#02x is not a recognized number type", type_)
	}
}

func readUUID(r *buffer.Buffer) (UUID, error) {
	var uuid UUID

	type_, err := readType(r)
	if err != nil {
		return uuid, err
	}

	if type_ != TypeCodeUUID {
		return uuid, fmt.Errorf("type code %#00x is not a UUID", type_)
	}

	buf, ok := r.Next(16)
	if !ok {
		return uuid, errors.New("invalid length")
	}
	copy(uuid[:], buf)

	return uuid, nil
}

func readMapHeader(r *buffer.Buffer) (count uint32, _ error) {
	type_, err := readType(r)
	if err != nil {
		return 0, err
	}

	length := r.Len()

	switch type_ {
	case TypeCodeMap8:
		buf, ok := r.Next(2)
		if !ok {
			return 0, errors.New("invalid length")
		}
		_ = buf[1]

		size := int(buf[0])
		if size > length-1 {
			return 0, errors.New("invalid length")
		}
		count = uint32(buf[1])
	case TypeCodeMap32:
		buf, ok := r.Next(8)
		if !ok {
			return 0, errors.New("invalid length")
		}
		_ = buf[7]

		size := int(binary.BigEndian.Uint32(buf[:4]))
		if size > length-4 {
			return 0, errors.New("invalid length")
		}
		count = binary.BigEndian.Uint32(buf[4:8])
	default:
		return 0, fmt.Errorf("invalid map type %#02x", type_)
	}

	if int(count) > r.Len() {
		return 0, errors.New("invalid length")
	}
	return count, nil
}
