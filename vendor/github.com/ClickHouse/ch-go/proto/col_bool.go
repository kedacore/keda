package proto

// ColBool is Bool column.
type ColBool []bool

// Compile-time assertions for ColBool.
var (
	_ ColInput       = ColBool{}
	_ ColResult      = (*ColBool)(nil)
	_ Column         = (*ColBool)(nil)
	_ ColumnOf[bool] = (*ColBool)(nil)
)

func (c ColBool) Row(i int) bool {
	return c[i]
}

func (c *ColBool) Append(v bool) {
	*c = append(*c, v)
}

func (c *ColBool) AppendArr(vs []bool) {
	*c = append(*c, vs...)
}

// Type returns ColumnType of Bool.
func (ColBool) Type() ColumnType {
	return ColumnTypeBool
}

// Rows returns count of rows in column.
func (c ColBool) Rows() int {
	return len(c)
}

// Reset resets data in row, preserving capacity for efficiency.
func (c *ColBool) Reset() {
	*c = (*c)[:0]
}

// Array is helper that creates Array(Bool).
func (c *ColBool) Array() *ColArr[bool] {
	return &ColArr[bool]{
		Data: c,
	}
}

// Nullable is helper that creates Nullable(Bool).
func (c *ColBool) Nullable() *ColNullable[bool] {
	return &ColNullable[bool]{
		Values: c,
	}
}
