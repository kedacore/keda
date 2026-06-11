package proto

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
)

// Buffer implements ClickHouse binary protocol encoding.
type Buffer struct {
	Buf []byte
}

// Reader returns new *Reader from *Buffer.
func (b *Buffer) Reader() *Reader {
	return NewReader(bytes.NewReader(b.Buf))
}

// Ensure Buf length.
func (b *Buffer) Ensure(n int) {
	if cap(b.Buf) < n {
		b.Buf = make([]byte, n)
	} else {
		b.Buf = b.Buf[:n] // Set length to n (zeros not guaranteed)
	}
}

// Encoder implements encoding to Buffer.
type Encoder interface {
	Encode(b *Buffer)
}

// AwareEncoder implements encoding to Buffer that depends on version.
type AwareEncoder interface {
	EncodeAware(b *Buffer, version int)
}

// EncodeAware value that implements AwareEncoder.
func (b *Buffer) EncodeAware(e AwareEncoder, version int) {
	e.EncodeAware(b, version)
}

// Encode value that implements Encoder.
func (b *Buffer) Encode(e Encoder) {
	e.Encode(b)
}

// Reset buffer to zero length.
func (b *Buffer) Reset() {
	b.Buf = b.Buf[:0]
}

// Read implements io.Reader.
func (b *Buffer) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(b.Buf) == 0 {
		return 0, io.EOF
	}
	n = copy(p, b.Buf)
	b.Buf = b.Buf[n:]
	return n, nil
}

// PutRaw writes v as raw bytes to buffer.
func (b *Buffer) PutRaw(v []byte) {
	b.Buf = append(b.Buf, v...)
}

// PutUVarInt encodes x as uvarint.
func (b *Buffer) PutUVarInt(x uint64) {
	buf := [binary.MaxVarintLen64]byte{}
	n := binary.PutUvarint(buf[:], x)
	b.Buf = append(b.Buf, buf[:n]...)
}

// PutInt encodes integer as uvarint.
func (b *Buffer) PutInt(x int) {
	b.PutUVarInt(uint64(x))
}

// PutByte encodes byte as uint8.
func (b *Buffer) PutByte(x byte) {
	b.PutUInt8(x)
}

// PutLen encodes length to buffer as uvarint.
func (b *Buffer) PutLen(x int) {
	b.PutUVarInt(uint64(x))
}

// PutString encodes sting value to buffer.
func (b *Buffer) PutString(s string) {
	b.PutLen(len(s))
	b.Buf = append(b.Buf, s...)
}

func (b *Buffer) PutUInt8(x uint8) {
	b.Buf = append(b.Buf, x)
}

func (b *Buffer) PutUInt16(x uint16) {
	buf := [16 / 8]byte{}
	binary.LittleEndian.PutUint16(buf[:], x)
	b.Buf = append(b.Buf, buf[:]...)
}

func (b *Buffer) PutUInt32(x uint32) {
	buf := [32 / 8]byte{}
	binary.LittleEndian.PutUint32(buf[:], x)
	b.Buf = append(b.Buf, buf[:]...)
}

func (b *Buffer) PutUInt64(x uint64) {
	buf := [64 / 8]byte{}
	binary.LittleEndian.PutUint64(buf[:], x)
	b.Buf = append(b.Buf, buf[:]...)
}

func (b *Buffer) PutUInt128(x UInt128) {
	buf := [128 / 8]byte{}
	binPutUInt128(buf[:], x)
	b.Buf = append(b.Buf, buf[:]...)
}

func (b *Buffer) PutInt8(v int8) {
	b.PutUInt8(uint8(v))
}

func (b *Buffer) PutInt16(v int16) {
	b.PutUInt16(uint16(v))
}

func (b *Buffer) PutInt32(x int32) {
	b.PutUInt32(uint32(x))
}

func (b *Buffer) PutInt64(x int64) {
	b.PutUInt64(uint64(x))
}

func (b *Buffer) PutInt128(x Int128) {
	b.PutUInt128(UInt128(x))
}

func (b *Buffer) PutBool(v bool) {
	if v {
		b.PutUInt8(boolTrue)
	} else {
		b.PutUInt8(boolFalse)
	}
}

func (b *Buffer) PutFloat64(v float64) {
	b.PutUInt64(math.Float64bits(v))
}

func (b *Buffer) PutFloat32(v float32) {
	b.PutUInt32(math.Float32bits(v))
}
