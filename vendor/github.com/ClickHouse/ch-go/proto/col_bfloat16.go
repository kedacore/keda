package proto

// ColBFloat16 is ClickHouse's BFloat16 column type.
// BFloat16 (Brain Floating Point) is a 16-bit floating point format
// with 1 sign bit, 8 exponent bits and 7 mantissa bits.
// It is represented as []uint16 internally and exposed as float32 in
// the APIs.
type ColBFloat16 []uint16

// Make ColBFloat16 always satisfies required Column related
// interfaces.
var (
	_ ColInput  = ColBFloat16{}
	_ ColResult = (*ColBFloat16)(nil)
	_ Column    = (*ColBFloat16)(nil)
)

func (c ColBFloat16) Rows() int {
	return len(c)
}

func (c *ColBFloat16) Reset() {
	*c = (*c)[:0]
}

func (c ColBFloat16) Type() ColumnType {
	return ColumnTypeBFloat16
}

func (c ColBFloat16) Row(i int) float32 {
	return BFloat16ToFloat32(c[i])
}

func (c *ColBFloat16) Append(v float32) {
	*c = append(*c, Float32ToBFloat16(v))
}

func (c *ColBFloat16) AppendArr(vs []float32) {
	for _, v := range vs {
		c.Append(v)
	}
}

// Array is a helper that creates Array of BFloat16.
func (c *ColBFloat16) Array() *ColArr[float32] {
	return &ColArr[float32]{
		Data: &colBFloat16Adapter{col: c},
	}
}

// Nullable is a helper that creates Nullable(BFloat16).
func (c *ColBFloat16) Nullable() *ColNullable[float32] {
	return &ColNullable[float32]{
		Values: &colBFloat16Adapter{col: c},
	}
}

// NewArrBFloat16 returns new Array(BFloat16).
func NewArrBFloat16() *ColArr[float32] {
	return &ColArr[float32]{
		Data: new(ColBFloat16),
	}
}

// LowCardinality is a helper that creates LowCardinality(BFloat16)
func (c *ColBFloat16) LowCardinality() *ColLowCardinality[float32] {
	return &ColLowCardinality[float32]{
		index: &colBFloat16Adapter{col: c},
	}
}

// colBFloat16Adapter is a wrapper on top of ColBFloat16 to
// make it work with generic column types (like `LowCardinality()`, `Array()`, `Nullable()`)
type colBFloat16Adapter struct {
	col *ColBFloat16
}

func (a *colBFloat16Adapter) Rows() int {
	return a.col.Rows()
}

func (a *colBFloat16Adapter) Reset() {
	a.col.Reset()
}

func (a *colBFloat16Adapter) Type() ColumnType {
	return ColumnTypeBFloat16
}

func (a *colBFloat16Adapter) Row(i int) float32 {
	return a.col.Row(i)
}

func (a *colBFloat16Adapter) Append(v float32) {
	a.col.Append(v)
}

func (a *colBFloat16Adapter) AppendArr(vs []float32) {
	a.col.AppendArr(vs)
}

func (a *colBFloat16Adapter) EncodeColumn(b *Buffer) {
	a.col.EncodeColumn(b)
}

func (a *colBFloat16Adapter) DecodeColumn(r *Reader, rows int) error {
	return a.col.DecodeColumn(r, rows)
}

func (a *colBFloat16Adapter) WriteColumn(w *Writer) {
	a.col.WriteColumn(w)
}
