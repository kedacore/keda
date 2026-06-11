//go:build (amd64 || arm64 || riscv64) && !purego

package proto

import (
	"unsafe"

	"github.com/go-faster/errors"
)

// EncodeColumn encodes Bool rows to *Buffer.
func (c ColBool) EncodeColumn(b *Buffer) {
	if len(c) == 0 {
		return
	}
	offset := len(b.Buf)
	b.Buf = append(b.Buf, make([]byte, len(c))...)
	s := *(*slice)(unsafe.Pointer(&c))    // #nosec G103
	src := *(*[]byte)(unsafe.Pointer(&s)) // #nosec G103
	dst := b.Buf[offset:]
	copy(dst, src)
}

// DecodeColumn decodes Bool rows from *Reader.
func (c *ColBool) DecodeColumn(r *Reader, rows int) error {
	if rows == 0 {
		return nil
	}
	*c = append(*c, make([]bool, rows)...)
	s := *(*slice)(unsafe.Pointer(c))     // #nosec G103
	dst := *(*[]byte)(unsafe.Pointer(&s)) // #nosec G103
	if err := r.ReadFull(dst); err != nil {
		return errors.Wrap(err, "read full")
	}
	return nil
}

// WriteColumn writes Bool rows to *Writer.
func (c ColBool) WriteColumn(w *Writer) {
	if len(c) == 0 {
		return
	}
	s := *(*slice)(unsafe.Pointer(&c))    // #nosec G103
	src := *(*[]byte)(unsafe.Pointer(&s)) // #nosec G103
	w.ChainWrite(src)
}
