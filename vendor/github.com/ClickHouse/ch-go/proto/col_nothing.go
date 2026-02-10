package proto

import (
	"fmt"

	"github.com/go-faster/errors"
)

// Nothing represents NULL value.
type Nothing struct{}

// ColNothing represents column of null values.
// Value is row count.
//
// https://clickhouse.com/docs/ru/sql-reference/data-types/special-data-types/nothing
type ColNothing int

func (c *ColNothing) Append(_ Nothing) {
	*c++
}

func (c *ColNothing) AppendArr(vs []Nothing) {
	*c = ColNothing(int(*c) + len(vs))
}

func (c ColNothing) Row(i int) Nothing {
	if i >= int(c) {
		panic(fmt.Sprintf("[%d] of [%d]Nothing", i, c))
	}
	return Nothing{}
}

func (c ColNothing) Type() ColumnType {
	return ColumnTypeNothing
}

func (c ColNothing) Rows() int {
	return int(c)
}

func (c *ColNothing) DecodeColumn(r *Reader, rows int) error {
	*c = ColNothing(rows)
	if rows == 0 {
		return nil
	}
	if _, err := r.ReadRaw(rows); err != nil {
		return errors.Wrap(err, "read")
	}
	return nil
}

func (c *ColNothing) Reset() {
	*c = 0
}

func (c *ColNothing) Nullable() *ColNullable[Nothing] {
	return &ColNullable[Nothing]{
		Values: c,
	}
}

func (c *ColNothing) Array() *ColArr[Nothing] {
	return &ColArr[Nothing]{
		Data: c,
	}
}

func (c ColNothing) EncodeColumn(b *Buffer) {
	if c == 0 {
		return
	}
	b.PutRaw(make([]byte, c))
}
