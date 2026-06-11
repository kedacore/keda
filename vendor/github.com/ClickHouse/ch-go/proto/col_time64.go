package proto

import (
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/go-faster/errors"
)

var (
	_ ColumnOf[Time64] = (*ColTime64)(nil)
	_ Inferable        = (*ColTime64)(nil)
)

// ColTime64 implements ColumnOf[Time64].
type ColTime64 struct {
	Data         []Time64
	Precision    Precision
	PrecisionSet bool
}

func (c *ColTime64) Reset() {
	c.Data = c.Data[:0]
}

func (c ColTime64) Rows() int {
	return len(c.Data)
}

func (c ColTime64) Type() ColumnType {
	return ColumnTypeTime64.With(fmt.Sprintf("%d", c.Precision))
}

func (c *ColTime64) Infer(t ColumnType) error {
	elem := t.Elem()
	if elem == "" {
		c.Precision = PrecisionNano
		return nil
	}

	precision, err := strconv.Atoi(string(elem))
	if err != nil {
		return errors.Wrap(err, "parse precision")
	}

	if precision < 0 || precision > 9 {
		return errors.New("precision must be between 0 and 9")
	}

	c.Precision = Precision(precision)
	return nil
}

func (c ColTime64) Row(i int) Time64 {
	return c.Data[i]
}

func (c *ColTime64) Append(v Time64) {
	c.Data = append(c.Data, v)
}

func (c *ColTime64) AppendArr(vs []Time64) {
	c.Data = append(c.Data, vs...)
}

func (c *ColTime64) LowCardinality() *ColLowCardinality[Time64] {
	return &ColLowCardinality[Time64]{
		index: c,
	}
}

func (c *ColTime64) Array() *ColArr[Time64] {
	return &ColArr[Time64]{
		Data: c,
	}
}

func (c *ColTime64) Nullable() *ColNullable[Time64] {
	return &ColNullable[Time64]{
		Values: c,
	}
}

func NewArrTime64() *ColArr[Time64] {
	return &ColArr[Time64]{
		Data: &ColTime64{},
	}
}

func (c *ColTime64) WithPrecision(p Precision) *ColTime64 {
	c.Precision = p
	c.PrecisionSet = true
	return c
}

func (c *ColTime64) DecodeColumn(r *Reader, rows int) error {
	if rows == 0 {
		return nil
	}
	const size = 64 / 8
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
			Time64(binary.LittleEndian.Uint64(data[i:i+size])),
		)
	}
	c.Data = v
	return nil
}

// EncodeColumn encodes Time64 rows to *Buffer.
func (c ColTime64) EncodeColumn(b *Buffer) {
	v := c.Data
	if len(v) == 0 {
		return
	}
	const size = 64 / 8
	offset := len(b.Buf)
	b.Buf = append(b.Buf, make([]byte, size*len(v))...)
	for _, vv := range v {
		binary.LittleEndian.PutUint64(
			b.Buf[offset:offset+size],
			uint64(vv),
		)
		offset += size
	}
}

func (c ColTime64) WriteColumn(w *Writer) {
	w.ChainBuffer(c.EncodeColumn)
}
