package proto

import "github.com/go-faster/errors"

// Compile-time assertions for ColNullable.
var (
	_ ColInput                   = (*ColNullable[string])(nil)
	_ ColResult                  = (*ColNullable[string])(nil)
	_ Column                     = (*ColNullable[string])(nil)
	_ ColumnOf[Nullable[string]] = (*ColNullable[string])(nil)
	_ StateEncoder               = (*ColNullable[string])(nil)
	_ StateDecoder               = (*ColNullable[string])(nil)

	_ = ColNullable[string]{
		Values: new(ColStr),
	}
)

// Nullable is T value that can be null.
type Nullable[T any] struct {
	Set   bool
	Value T
}

// NewNullable returns set value of Nullable[T] to v.
func NewNullable[T any](v T) Nullable[T] {
	return Nullable[T]{Set: true, Value: v}
}

// Null returns null value for Nullable[T].
func Null[T any]() Nullable[T] {
	return Nullable[T]{}
}

func (n Nullable[T]) IsSet() bool { return n.Set }

func (n Nullable[T]) Or(v T) T {
	if !n.Set {
		return v
	}
	return n.Value
}

// NewColNullable returns new Nullable(T) from v column.
func NewColNullable[T any](v ColumnOf[T]) *ColNullable[T] {
	return &ColNullable[T]{
		Values: v,
	}
}

// ColNullable represents Nullable(T) column.
//
// Nulls is nullable "mask" on Values column.
// For example, to encode [null, "", "hello", null, "world"]
//
//	Values: ["", "", "hello", "", "world"] (len: 5)
//	Nulls:  [ 1,  0,       0,  1,       0] (len: 5)
//
// Values and Nulls row counts are always equal.
type ColNullable[T any] struct {
	Nulls  ColUInt8
	Values ColumnOf[T]
}

func (c *ColNullable[T]) DecodeState(r *Reader) error {
	if s, ok := c.Values.(StateDecoder); ok {
		if err := s.DecodeState(r); err != nil {
			return errors.Wrap(err, "values state")
		}
	}
	return nil
}

func (c ColNullable[T]) EncodeState(b *Buffer) {
	if s, ok := c.Values.(StateEncoder); ok {
		s.EncodeState(b)
	}
}

func (c ColNullable[T]) Type() ColumnType {
	return ColumnTypeNullable.Sub(c.Values.Type())
}

func (c *ColNullable[T]) DecodeColumn(r *Reader, rows int) error {
	if err := c.Nulls.DecodeColumn(r, rows); err != nil {
		return errors.Wrap(err, "nulls")
	}
	if err := c.Values.DecodeColumn(r, rows); err != nil {
		return errors.Wrap(err, "values")
	}
	return nil
}

func (c ColNullable[T]) Rows() int {
	return c.Nulls.Rows()
}

func (c *ColNullable[T]) Append(v Nullable[T]) {
	null := boolTrue
	if v.Set {
		null = boolFalse
	}
	c.Nulls.Append(null)
	c.Values.Append(v.Value)
}

func (c *ColNullable[T]) AppendArr(v []Nullable[T]) {
	for _, vv := range v {
		c.Append(vv)
	}
}

func (c ColNullable[T]) Row(i int) Nullable[T] {
	return Nullable[T]{
		Value: c.Values.Row(i),
		Set:   c.Nulls.Row(i) == boolFalse,
	}
}

func (c *ColNullable[T]) Array() *ColArr[Nullable[T]] {
	return &ColArr[Nullable[T]]{
		Data: c,
	}
}

func (c *ColNullable[T]) Reset() {
	c.Nulls.Reset()
	c.Values.Reset()
}

func (c ColNullable[T]) EncodeColumn(b *Buffer) {
	c.Nulls.EncodeColumn(b)
	c.Values.EncodeColumn(b)
}

func (c ColNullable[T]) WriteColumn(w *Writer) {
	c.Nulls.WriteColumn(w)
	c.Values.WriteColumn(w)
}

func (c ColNullable[T]) IsElemNull(i int) bool {
	if i < c.Rows() {
		return c.Nulls[i] == boolTrue
	}
	return false
}
