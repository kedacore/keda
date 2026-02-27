//go:build (amd64 || arm64 || riscv64) && !purego

package proto

import (
	"unsafe"

	"github.com/go-faster/errors"
	"github.com/google/uuid"
	"github.com/segmentio/asm/bswap"
)

func (c *ColUUID) DecodeColumn(r *Reader, rows int) error {
	if rows == 0 {
		return nil
	}
	*c = append(*c, make([]uuid.UUID, rows)...)

	// Memory layout of [N]UUID is same as [N*sizeof(UUID)]byte.
	// So just interpret c as byte slice and read data into it.
	s := *(*slice)(unsafe.Pointer(c)) // #nosec: G103 // memory layout matches
	const size = 16
	s.Len *= size
	s.Cap *= size
	dst := *(*[]byte)(unsafe.Pointer(&s)) // #nosec: G103 // memory layout matches
	if err := r.ReadFull(dst); err != nil {
		return errors.Wrap(err, "read full")
	}
	bswap.Swap64(dst) // BE <-> LE

	return nil
}

// EncodeColumn encodes ColUUID rows to *Buffer.
func (c ColUUID) EncodeColumn(b *Buffer) {
	if len(c) == 0 {
		return
	}
	offset := len(b.Buf)
	const size = 16
	b.Buf = append(b.Buf, make([]byte, size*len(c))...)
	s := *(*slice)(unsafe.Pointer(&c)) // #nosec: G103 // memory layout matches
	s.Len *= size
	s.Cap *= size
	src := *(*[]byte)(unsafe.Pointer(&s)) // #nosec: G103 // memory layout matches
	dst := b.Buf[offset:]
	copy(dst, src)
	bswap.Swap64(dst) // BE <-> LE
}
