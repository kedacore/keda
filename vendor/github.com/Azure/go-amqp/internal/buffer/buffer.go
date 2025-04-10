package buffer

import (
	"encoding/binary"
	"io"
)

// Buffer is similar to bytes.Buffer but specialized for this module.
// The zero-value is an empty buffer ready for use.
type Buffer struct {
	b []byte
	i int
}

// New creates a new Buffer with b as its initial contents.
// Use this to start reading from b.
func New(b []byte) *Buffer {
	return &Buffer{b: b}
}

// Next returns a slice containing the next n bytes from the buffer and advances the buffer.
// If there are fewer than n bytes in the buffer, Next returns the remaining contents, false.
// The slice is only valid until the next call to a read or write method.
func (b *Buffer) Next(n int64) ([]byte, bool) {
	if b.readCheck(n) {
		buf := b.b[b.i:len(b.b)]
		b.i = len(b.b)
		return buf, false
	}

	buf := b.b[b.i : b.i+int(n)]
	b.i += int(n)
	return buf, true
}

// Skip advances the buffer by n bytes.
func (b *Buffer) Skip(n int) {
	b.i += n
}

// Reset resets the buffer to be empty but retains
// the underlying storage for use by future writes.
func (b *Buffer) Reset() {
	b.b = b.b[:0]
	b.i = 0
}

// Reclaim moves the unread portion of the buffer to the
// beginning of the underlying slice and resets the index.
func (b *Buffer) Reclaim() {
	l := b.Len()
	copy(b.b[:l], b.b[b.i:])
	b.b = b.b[:l]
	b.i = 0
}

// returns true if n is larger than the unread portion of the buffer
func (b *Buffer) readCheck(n int64) bool {
	return int64(b.i)+n > int64(len(b.b))
}

// ReadByte reads one byte from the buffer and advances the buffer.
// If there are insufficient bytes, an error is returned.
func (b *Buffer) ReadByte() (byte, error) {
	if b.readCheck(1) {
		return 0, io.EOF
	}

	byte_ := b.b[b.i]
	b.i++
	return byte_, nil
}

// PeekByte returns the next byte in the buffer without advancing the buffer.
// If there are insufficient bytes, an error is returned.
func (b *Buffer) PeekByte() (byte, error) {
	if b.readCheck(1) {
		return 0, io.EOF
	}

	return b.b[b.i], nil
}

// ReadUint16 reads two bytes from the buffer and decodes them
// as big-endian into a uint16. Advances the buffer by two.
// If there are insufficient bytes, an error is returned.
func (b *Buffer) ReadUint16() (uint16, error) {
	if b.readCheck(2) {
		return 0, io.EOF
	}

	n := binary.BigEndian.Uint16(b.b[b.i:])
	b.i += 2
	return n, nil
}

// ReadUint32 reads four bytes from the buffer and decodes them
// as big-endian into a uint32. Advances the buffer by four.
// If there are insufficient bytes, an error is returned.
func (b *Buffer) ReadUint32() (uint32, error) {
	if b.readCheck(4) {
		return 0, io.EOF
	}

	n := binary.BigEndian.Uint32(b.b[b.i:])
	b.i += 4
	return n, nil
}

// ReadUint64 reads eight bytes from the buffer and decodes them
// as big-endian into a uint64. Advances the buffer by eight.
// If there are insufficient bytes, an error is returned.
func (b *Buffer) ReadUint64() (uint64, error) {
	if b.readCheck(8) {
		return 0, io.EOF
	}

	n := binary.BigEndian.Uint64(b.b[b.i : b.i+8])
	b.i += 8
	return n, nil
}

// ReadFromOnce reads from r to populate the buffer.
// Reads up to cap - len of the underlying slice.
func (b *Buffer) ReadFromOnce(r io.Reader) error {
	const minRead = 512

	l := len(b.b)
	if cap(b.b)-l < minRead {
		total := l * 2
		if total == 0 {
			total = minRead
		}
		new := make([]byte, l, total)
		copy(new, b.b)
		b.b = new
	}

	n, err := r.Read(b.b[l:cap(b.b)])
	b.b = b.b[:l+n]
	return err
}

// Append appends p to the existing buffer.
func (b *Buffer) Append(p []byte) {
	b.b = append(b.b, p...)
}

// AppendByte appends bb to the existing buffer.
func (b *Buffer) AppendByte(bb byte) {
	b.b = append(b.b, bb)
}

// AppendString appends s to the existing buffer.
func (b *Buffer) AppendString(s string) {
	b.b = append(b.b, s...)
}

// Len returns the number of bytes of the unread portion of the buffer.
func (b *Buffer) Len() int {
	return len(b.b) - b.i
}

// Size returns the number of bytes that have been read from this buffer.
// This implies a minimum size of the underlying buffer.
func (b *Buffer) Size() int {
	return b.i
}

// Bytes returns a slice containing the unread portion of the buffer.
func (b *Buffer) Bytes() []byte {
	return b.b[b.i:]
}

// Detach returns the underlying byte slice, disassociating it from the buffer.
func (b *Buffer) Detach() []byte {
	temp := b.b
	b.b = nil
	b.i = 0
	return temp
}

// AppendUint16 appends n as two bytes in big-endian encoding.
func (b *Buffer) AppendUint16(n uint16) {
	b.b = append(b.b,
		byte(n>>8),
		byte(n),
	)
}

// AppendUint32 appends n as four bytes in big-endian encoding.
func (b *Buffer) AppendUint32(n uint32) {
	b.b = append(b.b,
		byte(n>>24),
		byte(n>>16),
		byte(n>>8),
		byte(n),
	)
}

// AppendUint64 appends n as eight bytes in big-endian encoding.
func (b *Buffer) AppendUint64(n uint64) {
	b.b = append(b.b,
		byte(n>>56),
		byte(n>>48),
		byte(n>>40),
		byte(n>>32),
		byte(n>>24),
		byte(n>>16),
		byte(n>>8),
		byte(n),
	)
}
