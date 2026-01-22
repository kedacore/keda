//go:build (amd64 || arm64 || riscv64) && !purego

package proto

import (
	"strconv"
	"unsafe"

	"github.com/go-faster/errors"
)

// ColRawOf is generic raw column.
type ColRawOf[X comparable] []X

func (c *ColRawOf[X]) AppendArr(v []X) {
	for _, x := range v {
		c.Append(x)
	}
}

func (c ColRawOf[X]) Size() int {
	var x X
	return int(unsafe.Sizeof(x)) // #nosec G103
}

// Type returns ColumnType of ColRawOf.
func (c ColRawOf[X]) Type() ColumnType {
	return ColumnTypeFixedString.With(strconv.Itoa(c.Size()))
}

// Rows returns count of rows in column.
func (c ColRawOf[X]) Rows() int {
	return len(c)
}

// Row returns value of "i" row.
func (c ColRawOf[X]) Row(i int) X {
	return c[i]
}

// Reset resets data in row, preserving capacity for efficiency.
func (c *ColRawOf[X]) Reset() {
	*c = (*c)[:0]
}

// Append value to column.
func (c *ColRawOf[X]) Append(v X) {
	*c = append(*c, v)
}

// EncodeColumn encodes ColRawOf rows to *Buffer.
func (c ColRawOf[X]) EncodeColumn(b *Buffer) {
	if len(c) == 0 {
		return
	}
	offset := len(b.Buf)
	var x X
	size := unsafe.Sizeof(x) // #nosec G103
	b.Buf = append(b.Buf, make([]byte, int(size)*len(c))...)
	s := *(*slice)(unsafe.Pointer(&c)) // #nosec G103
	s.Len *= size
	s.Cap *= size
	src := *(*[]byte)(unsafe.Pointer(&s)) // #nosec G103
	dst := b.Buf[offset:]
	copy(dst, src)
}

// DecodeColumn decodes ColRawOf rows from *Reader.
func (c *ColRawOf[X]) DecodeColumn(r *Reader, rows int) error {
	if rows == 0 {
		return nil
	}
	*c = append(*c, make([]X, rows)...)
	s := *(*slice)(unsafe.Pointer(c)) // #nosec G103
	var x X
	size := unsafe.Sizeof(x) // #nosec G103
	s.Len *= size
	s.Cap *= size
	dst := *(*[]byte)(unsafe.Pointer(&s)) // #nosec G103
	if err := r.ReadFull(dst); err != nil {
		return errors.Wrap(err, "read full")
	}
	return nil
}
