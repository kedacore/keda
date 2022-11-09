package encoding

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"
	"unicode/utf8"

	"github.com/Azure/go-amqp/internal/buffer"
)

type marshaler interface {
	Marshal(*buffer.Buffer) error
}

func Marshal(wr *buffer.Buffer, i interface{}) error {
	switch t := i.(type) {
	case nil:
		wr.AppendByte(byte(TypeCodeNull))
	case bool:
		if t {
			wr.AppendByte(byte(TypeCodeBoolTrue))
		} else {
			wr.AppendByte(byte(TypeCodeBoolFalse))
		}
	case *bool:
		if *t {
			wr.AppendByte(byte(TypeCodeBoolTrue))
		} else {
			wr.AppendByte(byte(TypeCodeBoolFalse))
		}
	case uint:
		writeUint64(wr, uint64(t))
	case *uint:
		writeUint64(wr, uint64(*t))
	case uint64:
		writeUint64(wr, t)
	case *uint64:
		writeUint64(wr, *t)
	case uint32:
		writeUint32(wr, t)
	case *uint32:
		writeUint32(wr, *t)
	case uint16:
		wr.AppendByte(byte(TypeCodeUshort))
		wr.AppendUint16(t)
	case *uint16:
		wr.AppendByte(byte(TypeCodeUshort))
		wr.AppendUint16(*t)
	case uint8:
		wr.Append([]byte{
			byte(TypeCodeUbyte),
			t,
		})
	case *uint8:
		wr.Append([]byte{
			byte(TypeCodeUbyte),
			*t,
		})
	case int:
		writeInt64(wr, int64(t))
	case *int:
		writeInt64(wr, int64(*t))
	case int8:
		wr.Append([]byte{
			byte(TypeCodeByte),
			uint8(t),
		})
	case *int8:
		wr.Append([]byte{
			byte(TypeCodeByte),
			uint8(*t),
		})
	case int16:
		wr.AppendByte(byte(TypeCodeShort))
		wr.AppendUint16(uint16(t))
	case *int16:
		wr.AppendByte(byte(TypeCodeShort))
		wr.AppendUint16(uint16(*t))
	case int32:
		writeInt32(wr, t)
	case *int32:
		writeInt32(wr, *t)
	case int64:
		writeInt64(wr, t)
	case *int64:
		writeInt64(wr, *t)
	case float32:
		writeFloat(wr, t)
	case *float32:
		writeFloat(wr, *t)
	case float64:
		writeDouble(wr, t)
	case *float64:
		writeDouble(wr, *t)
	case string:
		return writeString(wr, t)
	case *string:
		return writeString(wr, *t)
	case []byte:
		return WriteBinary(wr, t)
	case *[]byte:
		return WriteBinary(wr, *t)
	case map[interface{}]interface{}:
		return writeMap(wr, t)
	case *map[interface{}]interface{}:
		return writeMap(wr, *t)
	case map[string]interface{}:
		return writeMap(wr, t)
	case *map[string]interface{}:
		return writeMap(wr, *t)
	case map[Symbol]interface{}:
		return writeMap(wr, t)
	case *map[Symbol]interface{}:
		return writeMap(wr, *t)
	case Unsettled:
		return writeMap(wr, t)
	case *Unsettled:
		return writeMap(wr, *t)
	case time.Time:
		writeTimestamp(wr, t)
	case *time.Time:
		writeTimestamp(wr, *t)
	case []int8:
		return arrayInt8(t).Marshal(wr)
	case *[]int8:
		return arrayInt8(*t).Marshal(wr)
	case []uint16:
		return arrayUint16(t).Marshal(wr)
	case *[]uint16:
		return arrayUint16(*t).Marshal(wr)
	case []int16:
		return arrayInt16(t).Marshal(wr)
	case *[]int16:
		return arrayInt16(*t).Marshal(wr)
	case []uint32:
		return arrayUint32(t).Marshal(wr)
	case *[]uint32:
		return arrayUint32(*t).Marshal(wr)
	case []int32:
		return arrayInt32(t).Marshal(wr)
	case *[]int32:
		return arrayInt32(*t).Marshal(wr)
	case []uint64:
		return arrayUint64(t).Marshal(wr)
	case *[]uint64:
		return arrayUint64(*t).Marshal(wr)
	case []int64:
		return arrayInt64(t).Marshal(wr)
	case *[]int64:
		return arrayInt64(*t).Marshal(wr)
	case []float32:
		return arrayFloat(t).Marshal(wr)
	case *[]float32:
		return arrayFloat(*t).Marshal(wr)
	case []float64:
		return arrayDouble(t).Marshal(wr)
	case *[]float64:
		return arrayDouble(*t).Marshal(wr)
	case []bool:
		return arrayBool(t).Marshal(wr)
	case *[]bool:
		return arrayBool(*t).Marshal(wr)
	case []string:
		return arrayString(t).Marshal(wr)
	case *[]string:
		return arrayString(*t).Marshal(wr)
	case []Symbol:
		return arraySymbol(t).Marshal(wr)
	case *[]Symbol:
		return arraySymbol(*t).Marshal(wr)
	case [][]byte:
		return arrayBinary(t).Marshal(wr)
	case *[][]byte:
		return arrayBinary(*t).Marshal(wr)
	case []time.Time:
		return arrayTimestamp(t).Marshal(wr)
	case *[]time.Time:
		return arrayTimestamp(*t).Marshal(wr)
	case []UUID:
		return arrayUUID(t).Marshal(wr)
	case *[]UUID:
		return arrayUUID(*t).Marshal(wr)
	case []interface{}:
		return list(t).Marshal(wr)
	case *[]interface{}:
		return list(*t).Marshal(wr)
	case marshaler:
		return t.Marshal(wr)
	default:
		return fmt.Errorf("marshal not implemented for %T", i)
	}
	return nil
}

func writeInt32(wr *buffer.Buffer, n int32) {
	if n < 128 && n >= -128 {
		wr.Append([]byte{
			byte(TypeCodeSmallint),
			byte(n),
		})
		return
	}

	wr.AppendByte(byte(TypeCodeInt))
	wr.AppendUint32(uint32(n))
}

func writeInt64(wr *buffer.Buffer, n int64) {
	if n < 128 && n >= -128 {
		wr.Append([]byte{
			byte(TypeCodeSmalllong),
			byte(n),
		})
		return
	}

	wr.AppendByte(byte(TypeCodeLong))
	wr.AppendUint64(uint64(n))
}

func writeUint32(wr *buffer.Buffer, n uint32) {
	if n == 0 {
		wr.AppendByte(byte(TypeCodeUint0))
		return
	}

	if n < 256 {
		wr.Append([]byte{
			byte(TypeCodeSmallUint),
			byte(n),
		})
		return
	}

	wr.AppendByte(byte(TypeCodeUint))
	wr.AppendUint32(n)
}

func writeUint64(wr *buffer.Buffer, n uint64) {
	if n == 0 {
		wr.AppendByte(byte(TypeCodeUlong0))
		return
	}

	if n < 256 {
		wr.Append([]byte{
			byte(TypeCodeSmallUlong),
			byte(n),
		})
		return
	}

	wr.AppendByte(byte(TypeCodeUlong))
	wr.AppendUint64(n)
}

func writeFloat(wr *buffer.Buffer, f float32) {
	wr.AppendByte(byte(TypeCodeFloat))
	wr.AppendUint32(math.Float32bits(f))
}

func writeDouble(wr *buffer.Buffer, f float64) {
	wr.AppendByte(byte(TypeCodeDouble))
	wr.AppendUint64(math.Float64bits(f))
}

func writeTimestamp(wr *buffer.Buffer, t time.Time) {
	wr.AppendByte(byte(TypeCodeTimestamp))
	ms := t.UnixNano() / int64(time.Millisecond)
	wr.AppendUint64(uint64(ms))
}

// marshalField is a field to be marshaled
type MarshalField struct {
	Value interface{} // value to be marshaled, use pointers to avoid interface conversion overhead
	Omit  bool        // indicates that this field should be omitted (set to null)
}

// marshalComposite is a helper for us in a composite's marshal() function.
//
// The returned bytes include the composite header and fields. Fields with
// omit set to true will be encoded as null or omitted altogether if there are
// no non-null fields after them.
func MarshalComposite(wr *buffer.Buffer, code AMQPType, fields []MarshalField) error {
	// lastSetIdx is the last index to have a non-omitted field.
	// start at -1 as it's possible to have no fields in a composite
	lastSetIdx := -1

	// marshal each field into it's index in rawFields,
	// null fields are skipped, leaving the index nil.
	for i, f := range fields {
		if f.Omit {
			continue
		}
		lastSetIdx = i
	}

	// write header only
	if lastSetIdx == -1 {
		wr.Append([]byte{
			0x0,
			byte(TypeCodeSmallUlong),
			byte(code),
			byte(TypeCodeList0),
		})
		return nil
	}

	// write header
	WriteDescriptor(wr, code)

	// write fields
	wr.AppendByte(byte(TypeCodeList32))

	// write temp size, replace later
	sizeIdx := wr.Len()
	wr.Append([]byte{0, 0, 0, 0})
	preFieldLen := wr.Len()

	// field count
	wr.AppendUint32(uint32(lastSetIdx + 1))

	// write null to each index up to lastSetIdx
	for _, f := range fields[:lastSetIdx+1] {
		if f.Omit {
			wr.AppendByte(byte(TypeCodeNull))
			continue
		}
		err := Marshal(wr, f.Value)
		if err != nil {
			return err
		}
	}

	// fix size
	size := uint32(wr.Len() - preFieldLen)
	buf := wr.Bytes()
	binary.BigEndian.PutUint32(buf[sizeIdx:], size)

	return nil
}

func WriteDescriptor(wr *buffer.Buffer, code AMQPType) {
	wr.Append([]byte{
		0x0,
		byte(TypeCodeSmallUlong),
		byte(code),
	})
}

func writeString(wr *buffer.Buffer, str string) error {
	if !utf8.ValidString(str) {
		return errors.New("not a valid UTF-8 string")
	}
	l := len(str)

	switch {
	// Str8
	case l < 256:
		wr.Append([]byte{
			byte(TypeCodeStr8),
			byte(l),
		})
		wr.AppendString(str)
		return nil

	// Str32
	case uint(l) < math.MaxUint32:
		wr.AppendByte(byte(TypeCodeStr32))
		wr.AppendUint32(uint32(l))
		wr.AppendString(str)
		return nil

	default:
		return errors.New("too long")
	}
}

func WriteBinary(wr *buffer.Buffer, bin []byte) error {
	l := len(bin)

	switch {
	// List8
	case l < 256:
		wr.Append([]byte{
			byte(TypeCodeVbin8),
			byte(l),
		})
		wr.Append(bin)
		return nil

	// List32
	case uint(l) < math.MaxUint32:
		wr.AppendByte(byte(TypeCodeVbin32))
		wr.AppendUint32(uint32(l))
		wr.Append(bin)
		return nil

	default:
		return errors.New("too long")
	}
}

func writeMap(wr *buffer.Buffer, m interface{}) error {
	startIdx := wr.Len()
	wr.Append([]byte{
		byte(TypeCodeMap32), // type
		0, 0, 0, 0,          // size placeholder
		0, 0, 0, 0, // length placeholder
	})

	var pairs int
	switch m := m.(type) {
	case map[interface{}]interface{}:
		pairs = len(m) * 2
		for key, val := range m {
			err := Marshal(wr, key)
			if err != nil {
				return err
			}
			err = Marshal(wr, val)
			if err != nil {
				return err
			}
		}
	case map[string]interface{}:
		pairs = len(m) * 2
		for key, val := range m {
			err := writeString(wr, key)
			if err != nil {
				return err
			}
			err = Marshal(wr, val)
			if err != nil {
				return err
			}
		}
	case map[Symbol]interface{}:
		pairs = len(m) * 2
		for key, val := range m {
			err := key.Marshal(wr)
			if err != nil {
				return err
			}
			err = Marshal(wr, val)
			if err != nil {
				return err
			}
		}
	case Unsettled:
		pairs = len(m) * 2
		for key, val := range m {
			err := writeString(wr, key)
			if err != nil {
				return err
			}
			err = Marshal(wr, val)
			if err != nil {
				return err
			}
		}
	case Filter:
		pairs = len(m) * 2
		for key, val := range m {
			err := key.Marshal(wr)
			if err != nil {
				return err
			}
			err = val.Marshal(wr)
			if err != nil {
				return err
			}
		}
	case Annotations:
		pairs = len(m) * 2
		for key, val := range m {
			switch key := key.(type) {
			case string:
				err := Symbol(key).Marshal(wr)
				if err != nil {
					return err
				}
			case Symbol:
				err := key.Marshal(wr)
				if err != nil {
					return err
				}
			case int64:
				writeInt64(wr, key)
			case int:
				writeInt64(wr, int64(key))
			default:
				return fmt.Errorf("unsupported Annotations key type %T", key)
			}

			err := Marshal(wr, val)
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported map type %T", m)
	}

	if uint(pairs) > math.MaxUint32-4 {
		return errors.New("map contains too many elements")
	}

	// overwrite placeholder size and length
	bytes := wr.Bytes()[startIdx+1 : startIdx+9]
	_ = bytes[7] // bounds check hint

	length := wr.Len() - startIdx - 1 - 4 // -1 for type, -4 for length
	binary.BigEndian.PutUint32(bytes[:4], uint32(length))
	binary.BigEndian.PutUint32(bytes[4:8], uint32(pairs))

	return nil
}

// type length sizes
const (
	array8TLSize  = 2
	array32TLSize = 5
)

func writeArrayHeader(wr *buffer.Buffer, length, typeSize int, type_ AMQPType) {
	size := length * typeSize

	// array type
	if size+array8TLSize <= math.MaxUint8 {
		wr.Append([]byte{
			byte(TypeCodeArray8),      // type
			byte(size + array8TLSize), // size
			byte(length),              // length
			byte(type_),               // element type
		})
	} else {
		wr.AppendByte(byte(TypeCodeArray32))          //type
		wr.AppendUint32(uint32(size + array32TLSize)) // size
		wr.AppendUint32(uint32(length))               // length
		wr.AppendByte(byte(type_))                    // element type
	}
}

func writeVariableArrayHeader(wr *buffer.Buffer, length, elementsSizeTotal int, type_ AMQPType) {
	// 0xA_ == 1, 0xB_ == 4
	// http://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-types-v1.0-os.html#doc-idp82960
	elementTypeSize := 1
	if type_&0xf0 == 0xb0 {
		elementTypeSize = 4
	}

	size := elementsSizeTotal + (length * elementTypeSize) // size excluding array length
	if size+array8TLSize <= math.MaxUint8 {
		wr.Append([]byte{
			byte(TypeCodeArray8),      // type
			byte(size + array8TLSize), // size
			byte(length),              // length
			byte(type_),               // element type
		})
	} else {
		wr.AppendByte(byte(TypeCodeArray32))          // type
		wr.AppendUint32(uint32(size + array32TLSize)) // size
		wr.AppendUint32(uint32(length))               // length
		wr.AppendByte(byte(type_))                    // element type
	}
}
