package proto

import (
	"strings"
	"time"

	"github.com/go-faster/errors"
)

var (
	_ ColumnOf[time.Time] = (*ColDateTime)(nil)
	_ Inferable           = (*ColDateTime)(nil)
)

// ColDateTime implements ColumnOf[time.Time].
type ColDateTime struct {
	Data     []DateTime
	Location *time.Location
}

func (c *ColDateTime) Reset() {
	c.Data = c.Data[:0]
}

func (c ColDateTime) Rows() int {
	return len(c.Data)
}

func (c ColDateTime) Type() ColumnType {
	if c.Location == nil {
		return ColumnTypeDateTime
	}
	return ColumnTypeDateTime.With(`'` + c.Location.String() + `'`)
}

func (c *ColDateTime) Infer(t ColumnType) error {
	sub := t.Elem()
	if sub == "" {
		c.Location = nil
		return nil
	}
	rawLoc := string(sub)
	rawLoc = strings.Trim(rawLoc, `'`)
	loc, err := time.LoadLocation(rawLoc)
	if err != nil {
		return errors.Wrap(err, "load location")
	}
	c.Location = loc
	return nil
}

func (c ColDateTime) loc() *time.Location {
	if c.Location == nil {
		// Defaulting to local timezone (not UTC).
		return time.Local
	}
	return c.Location
}

func (c ColDateTime) Row(i int) time.Time {
	return c.Data[i].Time().In(c.loc())
}

func (c *ColDateTime) AppendRaw(v DateTime) {
	c.Data = append(c.Data, v)
}

func (c *ColDateTime) Append(v time.Time) {
	c.Data = append(c.Data, ToDateTime(v))
}

func (c *ColDateTime) AppendArr(vs []time.Time) {
	var dates = make([]DateTime, len(vs))

	for i, v := range vs {
		dates[i] = ToDateTime(v)
	}

	c.Data = append(c.Data, dates...)
}

// LowCardinality returns LowCardinality for Enum8.
func (c *ColDateTime) LowCardinality() *ColLowCardinality[time.Time] {
	return &ColLowCardinality[time.Time]{
		index: c,
	}
}

// Array is helper that creates Array of Enum8.
func (c *ColDateTime) Array() *ColArr[time.Time] {
	return &ColArr[time.Time]{
		Data: c,
	}
}

// Nullable is helper that creates Nullable(Enum8).
func (c *ColDateTime) Nullable() *ColNullable[time.Time] {
	return &ColNullable[time.Time]{
		Values: c,
	}
}

// NewArrDateTime returns new Array(DateTime).
func NewArrDateTime() *ColArr[time.Time] {
	return &ColArr[time.Time]{
		Data: &ColDateTime{},
	}
}
