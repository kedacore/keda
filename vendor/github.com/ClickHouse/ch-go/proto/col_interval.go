package proto

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-faster/errors"
)

//go:generate go run github.com/dmarkham/enumer -type IntervalScale -output interval_enum.go

type IntervalScale byte

const (
	IntervalSecond IntervalScale = iota
	IntervalMinute
	IntervalHour
	IntervalDay
	IntervalWeek
	IntervalMonth
	IntervalQuarter
	IntervalYear
)

type Interval struct {
	Scale IntervalScale
	Value int64
}

// Add Interval to time.Time.
func (i Interval) Add(t time.Time) time.Time {
	switch i.Scale {
	case IntervalSecond:
		return t.Add(time.Second * time.Duration(i.Value))
	case IntervalMinute:
		return t.Add(time.Minute * time.Duration(i.Value))
	case IntervalHour:
		return t.Add(time.Hour * time.Duration(i.Value))
	case IntervalDay:
		return t.AddDate(0, 0, int(i.Value))
	case IntervalWeek:
		return t.AddDate(0, 0, int(i.Value)*7)
	case IntervalMonth:
		return t.AddDate(0, int(i.Value), 0)
	case IntervalQuarter:
		return t.AddDate(0, int(i.Value)*4, 0)
	case IntervalYear:
		return t.AddDate(int(i.Value), 0, 0)
	default:
		panic(fmt.Sprintf("unknown interval scale %s", i.Scale))
	}
}

func (i Interval) String() string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("%d", i.Value))
	out.WriteRune(' ')
	out.WriteString(strings.ToLower(strings.TrimPrefix(i.Scale.String(), ColumnTypeInterval.String())))
	if i.Value > 1 || i.Value < 1 {
		out.WriteRune('s')
	}
	return out.String()
}

type ColInterval struct {
	Scale  IntervalScale
	Values ColInt64
}

func (c *ColInterval) Infer(t ColumnType) error {
	scale, err := IntervalScaleString(t.String())
	if err != nil {
		return errors.Wrap(err, "scale")
	}
	c.Scale = scale
	return nil
}

func (c *ColInterval) Append(v Interval) {
	if v.Scale != c.Scale {
		panic(fmt.Sprintf("append: cant append %s to %s", v.Scale, c.Scale))
	}
	c.Values.Append(v.Value)
}

func (c ColInterval) Row(i int) Interval {
	return Interval{
		Scale: c.Scale,
		Value: c.Values.Row(i),
	}
}

func (c ColInterval) Type() ColumnType {
	return ColumnType(c.Scale.String())
}

func (c ColInterval) Rows() int {
	return len(c.Values)
}

func (c *ColInterval) DecodeColumn(r *Reader, rows int) error {
	return c.Values.DecodeColumn(r, rows)
}

func (c *ColInterval) Reset() {
	c.Values.Reset()
}

func (c ColInterval) EncodeColumn(b *Buffer) {
	c.Values.EncodeColumn(b)
}

func (c ColInterval) WriteColumn(w *Writer) {
	c.Values.WriteColumn(w)
}
