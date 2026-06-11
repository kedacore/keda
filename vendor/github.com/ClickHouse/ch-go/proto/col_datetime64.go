package proto

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-faster/errors"
)

var (
	_ ColumnOf[time.Time] = (*ColDateTime64)(nil)
	_ Inferable           = (*ColDateTime64)(nil)
	_ Column              = (*ColDateTime64)(nil)
)

// ColDateTime64 implements ColumnOf[time.Time].
//
// If Precision is not set, Append and Row() panics.
// Use ColDateTime64Raw to work with raw DateTime64 values.
type ColDateTime64 struct {
	Data         []DateTime64
	Location     *time.Location
	Precision    Precision
	PrecisionSet bool
}

func (c *ColDateTime64) WithPrecision(p Precision) *ColDateTime64 {
	c.Precision = p
	c.PrecisionSet = true
	return c
}

func (c *ColDateTime64) WithLocation(loc *time.Location) *ColDateTime64 {
	c.Location = loc
	return c
}

func (c ColDateTime64) Rows() int {
	return len(c.Data)
}

func (c *ColDateTime64) Reset() {
	c.Data = c.Data[:0]
}

func (c ColDateTime64) Type() ColumnType {
	var elems []string
	if p := c.Precision; c.PrecisionSet {
		elems = append(elems, strconv.Itoa(int(p)))
	}
	if loc := c.Location; loc != nil {
		elems = append(elems, fmt.Sprintf(`'%s'`, loc))
	}
	return ColumnTypeDateTime64.With(elems...)
}

func (c *ColDateTime64) Infer(t ColumnType) error {
	elem := string(t.Elem())
	if elem == "" {
		return errors.Errorf("invalid DateTime64: no elements in %q", t)
	}
	pStr, locStr, hasloc := strings.Cut(elem, ",")
	pStr = strings.Trim(pStr, `' `)
	locStr = strings.Trim(locStr, `' `)
	n, err := strconv.ParseUint(pStr, 10, 8)
	if err != nil {
		return errors.Wrap(err, "parse precision")
	}
	p := Precision(n)
	if !p.Valid() {
		return errors.Errorf("precision %d is invalid", n)
	}
	c.Precision = p
	c.PrecisionSet = true
	if hasloc {
		loc, err := time.LoadLocation(locStr)
		if err != nil {
			return errors.Wrap(err, "invalid location")
		}
		c.Location = loc
	}
	return nil
}

func (c ColDateTime64) Row(i int) time.Time {
	if !c.PrecisionSet {
		panic("DateTime64: no precision set")
	}
	return c.Data[i].Time(c.Precision).In(c.loc())
}

func (c ColDateTime64) loc() *time.Location {
	if c.Location == nil {
		// Defaulting to local timezone (not UTC).
		return time.Local
	}
	return c.Location
}

func (c *ColDateTime64) AppendRaw(v DateTime64) {
	c.Data = append(c.Data, v)
}

func (c *ColDateTime64) Append(v time.Time) {
	if !c.PrecisionSet {
		panic("DateTime64: no precision set")
	}
	c.AppendRaw(ToDateTime64(v, c.Precision))
}

func (c *ColDateTime64) AppendArr(v []time.Time) {
	if !c.PrecisionSet {
		panic("DateTime64: no precision set")
	}

	for _, item := range v {
		c.AppendRaw(ToDateTime64(item, c.Precision))
	}
}

// Raw version of ColDateTime64 for ColumnOf[DateTime64].
func (c ColDateTime64) Raw() *ColDateTime64Raw {
	return &ColDateTime64Raw{ColDateTime64: c}
}

func (c *ColDateTime64) Nullable() *ColNullable[time.Time] {
	return &ColNullable[time.Time]{Values: c}
}

func (c *ColDateTime64) Array() *ColArr[time.Time] {
	return &ColArr[time.Time]{Data: c}
}

var (
	_ ColumnOf[DateTime64] = (*ColDateTime64Raw)(nil)
	_ Inferable            = (*ColDateTime64Raw)(nil)
	_ Column               = (*ColDateTime64Raw)(nil)
)

// ColDateTime64Raw is DateTime64 wrapper to implement ColumnOf[DateTime64].
type ColDateTime64Raw struct {
	ColDateTime64
}

func (c *ColDateTime64Raw) Append(v DateTime64) { c.AppendRaw(v) }
func (c *ColDateTime64Raw) AppendArr(vs []DateTime64) {
	for _, v := range vs {
		c.AppendRaw(v)
	}
}
func (c ColDateTime64Raw) Row(i int) DateTime64 { return c.Data[i] }
