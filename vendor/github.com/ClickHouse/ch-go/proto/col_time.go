package proto

import (
	"encoding/binary"

	"github.com/go-faster/errors"
)

var (
	_ ColumnOf[Time32] = (*ColTime32)(nil)
	_ Inferable        = (*ColTime32)(nil)
)

// ColTime32 implements ColumnOf[Time32].
type ColTime32 struct {
	Data         []Time32
	Precision    Precision
	PrecisionSet bool
}

func (c *ColTime32) WithPrecision(p Precision) *ColTime32 {
	c.Precision = p
	c.PrecisionSet = true
	return c
}

func (c *ColTime32) Reset() {
	c.Data = c.Data[:0]
}

func (c ColTime32) Rows() int {
	return len(c.Data)
}

func (c ColTime32) Type() ColumnType {
	return ColumnTypeTime32
}

func (c *ColTime32) Infer(t ColumnType) error {
	return nil
}

func (c ColTime32) Row(i int) Time32 {
	return c.Data[i]
}

func (c *ColTime32) Append(v Time32) {
	c.Data = append(c.Data, v)
}

func (c *ColTime32) AppendArr(vs []Time32) {
	c.Data = append(c.Data, vs...)
}

func (c *ColTime32) LowCardinality() *ColLowCardinality[Time32] {
	return &ColLowCardinality[Time32]{
		index: c,
	}
}

func (c *ColTime32) Array() *ColArr[Time32] {
	return &ColArr[Time32]{
		Data: c,
	}
}

func (c *ColTime32) Nullable() *ColNullable[Time32] {
	return &ColNullable[Time32]{
		Values: c,
	}
}

func NewArrTime32() *ColArr[Time32] {
	return &ColArr[Time32]{
		Data: &ColTime32{},
	}
}

// ColTime is an alias for ColTime32
type ColTime = ColTime32

func (c *ColTime32) DecodeColumn(r *Reader, rows int) error {
	if rows == 0 {
		return nil
	}
	const size = 32 / 8
	data, err := r.ReadRaw(rows * size)
	if err != nil {
		return errors.Wrap(err, "read")
	}
	v := c.Data
	// Move bound check out of loop.
	//
	// See https://github.com/golang/go/issues/30945.
	_ = data[len(data)-size]
	for i := 0; i <= len(data)-size; i += size {
		v = append(v,
			Time32(binary.LittleEndian.Uint32(data[i:i+size])),
		)
	}
	c.Data = v
	return nil
}

// EncodeColumn encodes Time32 rows to *Buffer.
func (c ColTime32) EncodeColumn(b *Buffer) {
	v := c.Data
	if len(v) == 0 {
		return
	}
	const size = 32 / 8
	offset := len(b.Buf)
	b.Buf = append(b.Buf, make([]byte, size*len(v))...)
	for _, vv := range v {
		binary.LittleEndian.PutUint32(
			b.Buf[offset:offset+size],
			uint32(vv),
		)
		offset += size
	}
}

func (c ColTime32) WriteColumn(w *Writer) {
	w.ChainBuffer(c.EncodeColumn)
}
