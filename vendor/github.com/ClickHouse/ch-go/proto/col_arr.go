package proto

import (
	"github.com/go-faster/errors"
)

// Compile-time assertions for Array.
var (
	_ ColInput     = NewArray[string]((*ColStr)(nil))
	_ ColResult    = NewArray[string]((*ColStr)(nil))
	_ Column       = NewArray[string]((*ColStr)(nil))
	_ StateEncoder = NewArray[string]((*ColStr)(nil))
	_ StateDecoder = NewArray[string]((*ColStr)(nil))
	_ Inferable    = NewArray[string]((*ColStr)(nil))
	_ Preparable   = NewArray[string]((*ColStr)(nil))
)

// Arrayable constraint specifies ability of column T to be Array(T).
type Arrayable[T any] interface {
	Array() *ColArr[T]
}

// ColArr is Array(T).
type ColArr[T any] struct {
	Offsets ColUInt64
	Data    ColumnOf[T]
}

// NewArray returns ColArr of c.
//
// Example: NewArray[string](new(ColStr))
func NewArray[T any](c ColumnOf[T]) *ColArr[T] {
	return &ColArr[T]{
		Data: c,
	}
}

// Type returns type of array, i.e. Array(T).
func (c ColArr[T]) Type() ColumnType {
	return ColumnTypeArray.Sub(c.Data.Type())
}

// Rows returns rows count.
func (c ColArr[T]) Rows() int {
	return c.Offsets.Rows()
}

func (c *ColArr[T]) DecodeState(r *Reader) error {
	if s, ok := c.Data.(StateDecoder); ok {
		if err := s.DecodeState(r); err != nil {
			return errors.Wrap(err, "data state")
		}
	}
	return nil
}

func (c *ColArr[T]) EncodeState(b *Buffer) {
	if s, ok := c.Data.(StateEncoder); ok {
		s.EncodeState(b)
	}
}

// Prepare ensures Preparable column propagation.
func (c *ColArr[T]) Prepare() error {
	if v, ok := c.Data.(Preparable); ok {
		if err := v.Prepare(); err != nil {
			return errors.Wrap(err, "prepare data")
		}
	}
	return nil
}

// Infer ensures Inferable column propagation.
func (c *ColArr[T]) Infer(t ColumnType) error {
	if v, ok := c.Data.(Inferable); ok {
		if err := v.Infer(t.Elem()); err != nil {
			return errors.Wrap(err, "infer data")
		}
	}
	return nil
}

// RowLen returns i-th row array length.
func (c ColArr[T]) RowLen(i int) int {
	var start int
	if i > 0 {
		start = int(c.Offsets[i-1])
	}

	end := int(c.Offsets[i])

	return end - start
}

// RowAppend appends i-th row to target and returns it.
func (c ColArr[T]) RowAppend(i int, target []T) []T {
	var start int
	end := int(c.Offsets[i])
	if i > 0 {
		start = int(c.Offsets[i-1])
	}
	for idx := start; idx < end; idx++ {
		target = append(target, c.Data.Row(idx))
	}

	return target
}

// Row returns i-th row.
func (c ColArr[T]) Row(i int) []T {
	return c.RowAppend(i, nil)
}

// DecodeColumn implements ColResult.
func (c *ColArr[T]) DecodeColumn(r *Reader, rows int) error {
	if err := c.Offsets.DecodeColumn(r, rows); err != nil {
		return errors.Wrap(err, "read offsets")
	}
	var size int
	if l := len(c.Offsets); l > 0 {
		// Pick last offset as total size of "elements" column.
		size = int(c.Offsets[l-1])
	}
	if err := checkRows(size); err != nil {
		return errors.Wrap(err, "array size")
	}
	if err := c.Data.DecodeColumn(r, size); err != nil {
		return errors.Wrap(err, "decode data")
	}
	return nil
}

// Reset implements ColResult.
func (c *ColArr[T]) Reset() {
	c.Data.Reset()
	c.Offsets.Reset()
}

// EncodeColumn implements ColInput.
func (c ColArr[T]) EncodeColumn(b *Buffer) {
	c.Offsets.EncodeColumn(b)
	c.Data.EncodeColumn(b)
}

// WriteColumn implements ColInput.
func (c ColArr[T]) WriteColumn(w *Writer) {
	c.Offsets.WriteColumn(w)
	c.Data.WriteColumn(w)
}

// Append appends new row to column.
func (c *ColArr[T]) Append(v []T) {
	c.Data.AppendArr(v)
	c.Offsets = append(c.Offsets, uint64(c.Data.Rows()))
}

// AppendArr appends new slice of rows to column.
func (c *ColArr[T]) AppendArr(vs [][]T) {
	for _, v := range vs {
		c.Data.AppendArr(v)
		c.Offsets = append(c.Offsets, uint64(c.Data.Rows()))
	}
}

// Result for current column.
func (c *ColArr[T]) Result(column string) ResultColumn {
	return ResultColumn{Name: column, Data: c}
}

// Results return Results containing single column.
func (c *ColArr[T]) Results(column string) Results {
	return Results{c.Result(column)}
}
