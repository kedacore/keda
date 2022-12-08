// Copyright (C) 2017 Kale Blankenship
// Portions Copyright (c) Microsoft Corporation
package buffer

import (
	"encoding/binary"
	"io"
)

// buffer is similar to bytes.Buffer but specialized for this package
type Buffer struct {
	b []byte
	i int
}

func New(b []byte) *Buffer {
	return &Buffer{b: b}
}

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

func (b *Buffer) Skip(n int) {
	b.i += n
}

func (b *Buffer) Reset() {
	b.b = b.b[:0]
	b.i = 0
}

// reclaim shifts used buffer space to the beginning of the
// underlying slice.
func (b *Buffer) Reclaim() {
	l := b.Len()
	copy(b.b[:l], b.b[b.i:])
	b.b = b.b[:l]
	b.i = 0
}

func (b *Buffer) readCheck(n int64) bool {
	return int64(b.i)+n > int64(len(b.b))
}

func (b *Buffer) ReadByte() (byte, error) {
	if b.readCheck(1) {
		return 0, io.EOF
	}

	byte_ := b.b[b.i]
	b.i++
	return byte_, nil
}

func (b *Buffer) PeekByte() (byte, error) {
	if b.readCheck(1) {
		return 0, io.EOF
	}

	return b.b[b.i], nil
}

func (b *Buffer) ReadUint16() (uint16, error) {
	if b.readCheck(2) {
		return 0, io.EOF
	}

	n := binary.BigEndian.Uint16(b.b[b.i:])
	b.i += 2
	return n, nil
}

func (b *Buffer) ReadUint32() (uint32, error) {
	if b.readCheck(4) {
		return 0, io.EOF
	}

	n := binary.BigEndian.Uint32(b.b[b.i:])
	b.i += 4
	return n, nil
}

func (b *Buffer) ReadUint64() (uint64, error) {
	if b.readCheck(8) {
		return 0, io.EOF
	}

	n := binary.BigEndian.Uint64(b.b[b.i : b.i+8])
	b.i += 8
	return n, nil
}

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

func (b *Buffer) Append(p []byte) {
	b.b = append(b.b, p...)
}

func (b *Buffer) AppendByte(bb byte) {
	b.b = append(b.b, bb)
}

func (b *Buffer) AppendString(s string) {
	b.b = append(b.b, s...)
}

func (b *Buffer) Len() int {
	return len(b.b) - b.i
}

func (b *Buffer) Size() int {
	return b.i
}

func (b *Buffer) Bytes() []byte {
	return b.b[b.i:]
}

func (b *Buffer) Detach() []byte {
	temp := b.b
	b.b = nil
	b.i = 0
	return temp
}

func (b *Buffer) AppendUint16(n uint16) {
	b.b = append(b.b,
		byte(n>>8),
		byte(n),
	)
}

func (b *Buffer) AppendUint32(n uint32) {
	b.b = append(b.b,
		byte(n>>24),
		byte(n>>16),
		byte(n>>8),
		byte(n),
	)
}

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
