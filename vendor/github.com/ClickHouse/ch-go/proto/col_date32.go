package proto

import "time"

func (c *ColDate32) Append(v time.Time) {
	*c = append(*c, ToDate32(v))
}

func (c *ColDate32) AppendArr(vs []time.Time) {
	var dates = make([]Date32, len(vs))

	for i, v := range vs {
		dates[i] = ToDate32(v)
	}

	*c = append(*c, dates...)
}

func (c ColDate32) Row(i int) time.Time {
	return c[i].Time()
}

// LowCardinality returns LowCardinality for Enum8 .
func (c *ColDate32) LowCardinality() *ColLowCardinality[time.Time] {
	return &ColLowCardinality[time.Time]{
		index: c,
	}
}

// Array is helper that creates Array of Enum8.
func (c *ColDate32) Array() *ColArr[time.Time] {
	return &ColArr[time.Time]{
		Data: c,
	}
}

// Nullable is helper that creates Nullable(Enum8).
func (c *ColDate32) Nullable() *ColNullable[time.Time] {
	return &ColNullable[time.Time]{
		Values: c,
	}
}

// NewArrDate32 returns new Array(Date32).
func NewArrDate32() *ColArr[time.Time] {
	return &ColArr[time.Time]{
		Data: new(ColDate32),
	}
}
