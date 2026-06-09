package proto

import (
	"strconv"

	"github.com/go-faster/errors"
)

// ColFixedStr represents FixedString(Size) column. Size is required.
//
// Can be used to store SHA256, MD5 or similar fixed size binary values.
// See https://clickhouse.com/docs/en/sql-reference/data-types/fixedstring/.
type ColFixedStr struct {
	Buf  []byte
	Size int // N
}

// Compile-time assertions for ColFixedStr.
var (
	_ ColInput  = ColFixedStr{}
	_ ColResult = (*ColFixedStr)(nil)
	_ Column    = (*ColFixedStr)(nil)
)

// Type returns ColumnType of FixedString.
func (c ColFixedStr) Type() ColumnType {
	return ColumnTypeFixedString.With(strconv.Itoa(c.Size))
}

// SetSize sets Size of FixedString(Size) to n.
//
// Can be called during decode to infer size from result.
func (c *ColFixedStr) SetSize(n int) {
	c.Size = n
}

// Rows returns count of rows in column.
func (c ColFixedStr) Rows() int {
	if c.Size == 0 {
		return 0
	}
	return len(c.Buf) / c.Size
}

// Row returns value of "i" row.
func (c ColFixedStr) Row(i int) []byte {
	return c.Buf[i*c.Size : (i+1)*c.Size]
}

// Reset resets data in row, preserving capacity for efficiency.
func (c *ColFixedStr) Reset() {
	c.Buf = c.Buf[:0]
}

// Append value to column. Panics if len(b) != Size.
//
// If Size is not set, will set to len of first value.
func (c *ColFixedStr) Append(b []byte) {
	if c.Size == 0 {
		// Automatic size set.
		c.Size = len(b)
	}
	if len(b) != c.Size {
		panic("invalid size")
	}
	c.Buf = append(c.Buf, b...)
}

func (c *ColFixedStr) AppendArr(vs [][]byte) {
	for _, v := range vs {
		c.Append(v)
	}
}

// EncodeColumn encodes ColFixedStr rows to *Buffer.
func (c ColFixedStr) EncodeColumn(b *Buffer) {
	b.Buf = append(b.Buf, c.Buf...)
}

// DecodeColumn decodes ColFixedStr rows from *Reader.
func (c *ColFixedStr) DecodeColumn(r *Reader, rows int) error {
	c.Buf = append(c.Buf[:0], make([]byte, rows*c.Size)...)
	if err := r.ReadFull(c.Buf); err != nil {
		return errors.Wrap(err, "read full")
	}
	return nil
}

// WriteColumn writes ColFixedStr rows to *Writer.
func (c ColFixedStr) WriteColumn(w *Writer) {
	w.ChainWrite(c.Buf)
}

// Array returns new Array(FixedString).
func (c *ColFixedStr) Array() *ColArr[[]byte] {
	return &ColArr[[]byte]{
		Data: c,
	}
}
