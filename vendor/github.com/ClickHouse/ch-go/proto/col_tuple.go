package proto

import "github.com/go-faster/errors"

// ColTuple is Tuple column.
//
// Basically it is just a group of columns.
type ColTuple []Column

// Compile-time assertions for ColTuple.
var (
	_ ColInput     = ColTuple(nil)
	_ ColResult    = ColTuple(nil)
	_ Column       = ColTuple(nil)
	_ StateEncoder = ColTuple(nil)
	_ StateDecoder = ColTuple(nil)
	_ Inferable    = ColTuple(nil)
	_ Preparable   = ColTuple(nil)
)

func (c ColTuple) DecodeState(r *Reader) error {
	for i, v := range c {
		if s, ok := v.(StateDecoder); ok {
			if err := s.DecodeState(r); err != nil {
				return errors.Wrapf(err, "[%d]", i)
			}
		}
	}
	return nil
}

// ColNamed is named column.
// Used in named tuples.
type ColNamed[T any] struct {
	ColumnOf[T]
	Name string
}

func (c *ColNamed[T]) Infer(t ColumnType) error {
	if v, ok := c.ColumnOf.(Inferable); ok {
		if err := v.Infer(t); err != nil {
			return errors.Wrap(err, "named")
		}
	}
	return nil
}

func (c *ColNamed[T]) Prepare() error {
	if v, ok := c.ColumnOf.(Preparable); ok {
		if err := v.Prepare(); err != nil {
			return errors.Wrap(err, "named")
		}
	}
	return nil
}

func (c ColNamed[T]) DecodeState(r *Reader) error {
	if v, ok := c.ColumnOf.(StateDecoder); ok {
		if err := v.DecodeState(r); err != nil {
			return errors.Wrap(err, "named")
		}
	}
	return nil
}

func (c ColNamed[T]) EncodeState(b *Buffer) {
	if v, ok := c.ColumnOf.(StateEncoder); ok {
		v.EncodeState(b)
	}
}

// Compile-time assertions for ColNamed.
var (
	_ ColInput     = Named[string]((*ColStr)(nil), "name")
	_ ColResult    = Named[string]((*ColStr)(nil), "name")
	_ Column       = Named[string]((*ColStr)(nil), "name")
	_ StateEncoder = Named[string]((*ColStr)(nil), "name")
	_ StateDecoder = Named[string]((*ColStr)(nil), "name")
	_ Inferable    = Named[string]((*ColStr)(nil), "name")
	_ Preparable   = Named[string]((*ColStr)(nil), "name")
)

func Named[T any](data ColumnOf[T], name string) *ColNamed[T] {
	return &ColNamed[T]{
		ColumnOf: data,
		Name:     name,
	}
}

func (c ColNamed[T]) ColumnName() string {
	return c.Name
}

func (c ColNamed[T]) Type() ColumnType {
	return ColumnType(c.Name + " " + c.ColumnOf.Type().String())
}

func (c ColTuple) Prepare() error {
	for _, v := range c {
		if s, ok := v.(Preparable); ok {
			if err := s.Prepare(); err != nil {
				return errors.Wrap(err, "prepare")
			}
		}
	}
	return nil
}

func (c ColTuple) Infer(t ColumnType) error {
	for _, v := range c {
		if s, ok := v.(Inferable); ok {
			if err := s.Infer(t); err != nil {
				return errors.Wrap(err, "infer")
			}
		}
	}
	return nil
}

func (c ColTuple) EncodeState(b *Buffer) {
	for _, v := range c {
		if s, ok := v.(StateEncoder); ok {
			s.EncodeState(b)
		}
	}
}

func (c ColTuple) Type() ColumnType {
	var types []ColumnType
	for _, v := range c {
		types = append(types, v.Type())
	}
	return ColumnTypeTuple.Sub(types...)
}

func (c ColTuple) First() Column {
	if len(c) == 0 {
		return nil
	}
	return c[0]
}

func (c ColTuple) Rows() int {
	if f := c.First(); f != nil {
		return f.Rows()
	}
	return 0
}

func (c ColTuple) DecodeColumn(r *Reader, rows int) error {
	for i, v := range c {
		if err := v.DecodeColumn(r, rows); err != nil {
			return errors.Wrapf(err, "[%d]", i)
		}
	}
	return nil
}

func (c ColTuple) Reset() {
	for _, v := range c {
		v.Reset()
	}
}

func (c ColTuple) EncodeColumn(b *Buffer) {
	for _, v := range c {
		v.EncodeColumn(b)
	}
}

func (c ColTuple) WriteColumn(w *Writer) {
	for _, v := range c {
		v.WriteColumn(w)
	}
}
