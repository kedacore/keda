package proto

import "time"

func (c *ColDate) Append(v time.Time) {
	*c = append(*c, ToDate(v))
}

func (c *ColDate) AppendArr(vs []time.Time) {
	var dates = make([]Date, len(vs))

	for i, v := range vs {
		dates[i] = ToDate(v)
	}

	*c = append(*c, dates...)
}

func (c ColDate) Row(i int) time.Time {
	return c[i].Time()
}

// LowCardinality returns LowCardinality for Enum8.
func (c *ColDate) LowCardinality() *ColLowCardinality[time.Time] {
	return &ColLowCardinality[time.Time]{
		index: c,
	}
}

// Array is helper that creates Array of Enum8.
func (c *ColDate) Array() *ColArr[time.Time] {
	return &ColArr[time.Time]{
		Data: c,
	}
}

// Nullable is helper that creates Nullable(Enum8).
func (c *ColDate) Nullable() *ColNullable[time.Time] {
	return &ColNullable[time.Time]{
		Values: c,
	}
}

// NewArrDate returns new Array(Date).
func NewArrDate() *ColArr[time.Time] {
	return &ColArr[time.Time]{
		Data: new(ColDate),
	}
}
