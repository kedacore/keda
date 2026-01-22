package proto

import "github.com/go-faster/errors"

type Point struct {
	X, Y float64
}

// Compile-time assertions for ColPoint.
var (
	_ ColInput        = ColPoint{}
	_ ColResult       = (*ColPoint)(nil)
	_ Column          = (*ColPoint)(nil)
	_ ColumnOf[Point] = (*ColPoint)(nil)
)

type ColPoint struct {
	X, Y ColFloat64
}

func (c *ColPoint) Append(v Point) {
	c.X.Append(v.X)
	c.Y.Append(v.Y)
}

func (c *ColPoint) AppendArr(v []Point) {
	for _, vv := range v {
		c.Append(vv)
	}
}

func (c ColPoint) Row(i int) Point {
	return Point{
		X: c.X.Row(i),
		Y: c.Y.Row(i),
	}
}

func (c ColPoint) Type() ColumnType { return ColumnTypePoint }
func (c ColPoint) Rows() int        { return c.X.Rows() }

func (c *ColPoint) DecodeColumn(r *Reader, rows int) error {
	if err := c.X.DecodeColumn(r, rows); err != nil {
		return errors.Wrap(err, "x")
	}
	if err := c.Y.DecodeColumn(r, rows); err != nil {
		return errors.Wrap(err, "y")
	}
	return nil
}

func (c *ColPoint) Reset() {
	c.X.Reset()
	c.Y.Reset()
}

func (c ColPoint) EncodeColumn(b *Buffer) {
	if b == nil {
		return
	}
	c.X.EncodeColumn(b)
	c.Y.EncodeColumn(b)
}
