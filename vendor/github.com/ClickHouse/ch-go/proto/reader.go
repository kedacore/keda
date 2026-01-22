package proto

import (
	"bufio"
	"encoding/binary"
	"io"
	"math"

	"github.com/go-faster/errors"

	"github.com/ClickHouse/ch-go/compress"
)

// Decoder implements decoding from Reader.
type Decoder interface {
	Decode(r *Reader) error
}

// AwareDecoder implements encoding to Buffer that depends on version.
type AwareDecoder interface {
	DecodeAware(r *Reader, version int) error
}

// Reader implements ClickHouse protocol decoding from buffered reader.
// Not goroutine-safe.
type Reader struct {
	raw  *bufio.Reader // raw bytes, e.g. on the wire
	data io.Reader     // data, decompressed or same as raw
	b    *Buffer       // internal buffer

	decompressed io.Reader // decompressed data stream, from raw
}

func (r *Reader) ReadByte() (byte, error) {
	if err := r.readFull(1); err != nil {
		return 0, err
	}
	return r.b.Buf[0], nil
}

// EnableCompression makes next reads use decompressed source of data.
func (r *Reader) EnableCompression() {
	r.data = r.decompressed
}

// DisableCompression makes next read use raw source of data.
func (r *Reader) DisableCompression() {
	r.data = r.raw
}

func (r *Reader) Read(p []byte) (n int, err error) {
	return r.data.Read(p)
}

// Decode value.
func (r *Reader) Decode(v Decoder) error {
	return v.Decode(r)
}

func (r *Reader) ReadFull(buf []byte) error {
	if _, err := io.ReadFull(r, buf); err != nil {
		return errors.Wrap(err, "read")
	}
	return nil
}

func (r *Reader) readFull(n int) error {
	r.b.Ensure(n)
	return r.ReadFull(r.b.Buf)
}

// ReadRaw reads raw n bytes.
func (r *Reader) ReadRaw(n int) ([]byte, error) {
	if err := r.readFull(n); err != nil {
		return nil, errors.Wrap(err, "read full")
	}

	return r.b.Buf, nil
}

// UVarInt reads uint64 from internal reader.
func (r *Reader) UVarInt() (uint64, error) {
	n, err := binary.ReadUvarint(r)
	if err != nil {
		return 0, errors.Wrap(err, "read")
	}
	return n, nil
}

func (r *Reader) StrLen() (int, error) {
	n, err := r.Int()
	if err != nil {
		return 0, errors.Wrap(err, "read length")
	}

	if n < 0 {
		return 0, errors.Errorf("size %d is invalid", n)
	}

	return n, nil
}

// StrRaw decodes string to internal buffer and returns it directly.
//
// Do not retain returned slice.
func (r *Reader) StrRaw() ([]byte, error) {
	n, err := r.StrLen()
	if err != nil {
		return nil, errors.Wrap(err, "read length")
	}
	r.b.Ensure(n)
	if _, err := io.ReadFull(r.data, r.b.Buf); err != nil {
		return nil, errors.Wrap(err, "read str")
	}

	return r.b.Buf, nil
}

// StrAppend decodes string and appends it to provided buf.
func (r *Reader) StrAppend(buf []byte) ([]byte, error) {
	defer r.b.Reset()

	str, err := r.StrRaw()
	if err != nil {
		return nil, errors.Wrap(err, "raw")
	}

	return append(buf, str...), nil
}

// StrBytes decodes string and allocates new byte slice with result.
func (r *Reader) StrBytes() ([]byte, error) {
	return r.StrAppend(nil)
}

// Str decodes string.
func (r *Reader) Str() (string, error) {
	s, err := r.StrBytes()
	if err != nil {
		return "", errors.Wrap(err, "bytes")
	}

	return string(s), err
}

// Int decodes uvarint as int.
func (r *Reader) Int() (int, error) {
	n, err := r.UVarInt()
	if err != nil {
		return 0, errors.Wrap(err, "uvarint")
	}
	return int(n), nil
}

// Int8 decodes int8 value.
func (r *Reader) Int8() (int8, error) {
	v, err := r.UInt8()
	if err != nil {
		return 0, err
	}
	return int8(v), nil
}

// Int16 decodes int16 value.
func (r *Reader) Int16() (int16, error) {
	v, err := r.UInt16()
	if err != nil {
		return 0, err
	}
	return int16(v), nil
}

// Int32 decodes int32 value.
func (r *Reader) Int32() (int32, error) {
	v, err := r.UInt32()
	if err != nil {
		return 0, err
	}
	return int32(v), nil
}

// Int64 decodes int64 value.
func (r *Reader) Int64() (int64, error) {
	v, err := r.UInt64()
	if err != nil {
		return 0, err
	}
	return int64(v), nil
}

// Int128 decodes Int128 value.
func (r *Reader) Int128() (Int128, error) {
	v, err := r.UInt128()
	if err != nil {
		return Int128{}, err
	}
	return Int128(v), nil
}

// Byte decodes byte value.
func (r *Reader) Byte() (byte, error) {
	return r.UInt8()
}

// UInt8 decodes uint8 value.
func (r *Reader) UInt8() (uint8, error) {
	if err := r.readFull(1); err != nil {
		return 0, errors.Wrap(err, "read")
	}
	return r.b.Buf[0], nil
}

// UInt16 decodes uint16 value.
func (r *Reader) UInt16() (uint16, error) {
	if err := r.readFull(2); err != nil {
		return 0, errors.Wrap(err, "read")
	}
	return binary.LittleEndian.Uint16(r.b.Buf), nil
}

// UInt32 decodes uint32 value.
func (r *Reader) UInt32() (uint32, error) {
	if err := r.readFull(32 / 8); err != nil {
		return 0, errors.Wrap(err, "read")
	}
	return binary.LittleEndian.Uint32(r.b.Buf), nil
}

// UInt64 decodes uint64 value.
func (r *Reader) UInt64() (uint64, error) {
	if err := r.readFull(64 / 8); err != nil {
		return 0, errors.Wrap(err, "read")
	}
	return binary.LittleEndian.Uint64(r.b.Buf), nil
}

// UInt128 decodes UInt128 value.
func (r *Reader) UInt128() (UInt128, error) {
	if err := r.readFull(128 / 8); err != nil {
		return UInt128{}, errors.Wrap(err, "read")
	}
	return binUInt128(r.b.Buf), nil
}

// Float32 decodes float32 value.
func (r *Reader) Float32() (float32, error) {
	v, err := r.UInt32()
	if err != nil {
		return 0, errors.Wrap(err, "bits")
	}
	return math.Float32frombits(v), nil
}

// Float64 decodes float64 value.
func (r *Reader) Float64() (float64, error) {
	v, err := r.UInt64()
	if err != nil {
		return 0, errors.Wrap(err, "bits")
	}
	return math.Float64frombits(v), nil
}

// Bool decodes bool as uint8.
func (r *Reader) Bool() (bool, error) {
	v, err := r.UInt8()
	if err != nil {
		return false, errors.Wrap(err, "uint8")
	}
	switch v {
	case boolTrue:
		return true, nil
	case boolFalse:
		return false, nil
	default:
		return false, errors.Errorf("unexpected value %d for boolean", v)
	}
}

const defaultReaderSize = 1024 * 128 // 128kb

// NewReader initializes new Reader from provided io.Reader.
func NewReader(r io.Reader) *Reader {
	c := bufio.NewReaderSize(r, defaultReaderSize)
	return &Reader{
		raw:          c,
		data:         c,
		b:            &Buffer{},
		decompressed: compress.NewReader(c),
	}
}
