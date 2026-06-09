package proto

import (
	"github.com/go-faster/errors"
)

const JSONStringSerializationVersion uint64 = 1

// ColJSONStr represents String column.
//
// Use ColJSONBytes for []bytes ColumnOf implementation.
type ColJSONStr struct {
	Str ColStr
}

// Append string to column.
func (c *ColJSONStr) Append(v string) {
	c.Str.Append(v)
}

// AppendBytes append byte slice as string to column.
func (c *ColJSONStr) AppendBytes(v []byte) {
	c.Str.AppendBytes(v)
}

func (c *ColJSONStr) AppendArr(v []string) {
	c.Str.AppendArr(v)
}

// Compile-time assertions for ColJSONStr.
var (
	_ ColInput          = ColJSONStr{}
	_ ColResult         = (*ColJSONStr)(nil)
	_ Column            = (*ColJSONStr)(nil)
	_ ColumnOf[string]  = (*ColJSONStr)(nil)
	_ Arrayable[string] = (*ColJSONStr)(nil)
	_ StateEncoder      = (*ColJSONStr)(nil)
	_ StateDecoder      = (*ColJSONStr)(nil)
)

// Type returns ColumnType of JSON.
func (ColJSONStr) Type() ColumnType {
	return ColumnTypeJSON
}

// Rows returns count of rows in column.
func (c ColJSONStr) Rows() int {
	return c.Str.Rows()
}

// Reset resets data in row, preserving capacity for efficiency.
func (c *ColJSONStr) Reset() {
	c.Str.Reset()
}

// EncodeState encodes the JSON serialization version
func (c *ColJSONStr) EncodeState(b *Buffer) {
	b.PutUInt64(JSONStringSerializationVersion)
}

// EncodeColumn encodes String rows to *Buffer.
func (c ColJSONStr) EncodeColumn(b *Buffer) {
	c.Str.EncodeColumn(b)
}

// WriteColumn writes JSON rows to *Writer.
func (c ColJSONStr) WriteColumn(w *Writer) {
	c.Str.WriteColumn(w)
}

// ForEach calls f on each string from column.
func (c ColJSONStr) ForEach(f func(i int, s string) error) error {
	return c.Str.ForEach(f)
}

// First returns the first row of the column.
func (c ColJSONStr) First() string {
	return c.Str.First()
}

// Row returns row with number i.
func (c ColJSONStr) Row(i int) string {
	return c.Str.Row(i)
}

// RowBytes returns row with number i as byte slice.
func (c ColJSONStr) RowBytes(i int) []byte {
	return c.Str.RowBytes(i)
}

// ForEachBytes calls f on each string from column as byte slice.
func (c ColJSONStr) ForEachBytes(f func(i int, b []byte) error) error {
	return c.Str.ForEachBytes(f)
}

// DecodeState decodes the JSON serialization version
func (c *ColJSONStr) DecodeState(r *Reader) error {
	jsonSerializationVersion, err := r.UInt64()
	if err != nil {
		return errors.Wrap(err, "failed to read json serialization version")
	}

	if jsonSerializationVersion != JSONStringSerializationVersion {
		return errors.Errorf("received invalid JSON string serialization version %d. Setting \"output_format_native_write_json_as_string\" must be enabled.", jsonSerializationVersion)
	}

	return nil
}

// DecodeColumn decodes String rows from *Reader.
func (c *ColJSONStr) DecodeColumn(r *Reader, rows int) error {
	return c.Str.DecodeColumn(r, rows)
}

// LowCardinality returns LowCardinality(JSON).
func (c *ColJSONStr) LowCardinality() *ColLowCardinality[string] {
	return c.Str.LowCardinality()
}

// Array is helper that creates Array(JSON).
func (c *ColJSONStr) Array() *ColArr[string] {
	return c.Str.Array()
}

// Nullable is helper that creates Nullable(JSON).
func (c *ColJSONStr) Nullable() *ColNullable[string] {
	return c.Str.Nullable()
}

// ColJSONBytes is ColJSONStr wrapper to be ColumnOf for []byte.
type ColJSONBytes struct {
	ColJSONStr
}

// Row returns row with number i.
func (c ColJSONBytes) Row(i int) []byte {
	return c.RowBytes(i)
}

// Append byte slice to column.
func (c *ColJSONBytes) Append(v []byte) {
	c.AppendBytes(v)
}

// AppendArr append slice of byte slices to column.
func (c *ColJSONBytes) AppendArr(v [][]byte) {
	for _, s := range v {
		c.Append(s)
	}
}

// Array is helper that creates Array(JSON).
func (c *ColJSONBytes) Array() *ColArr[[]byte] {
	return &ColArr[[]byte]{
		Data: c,
	}
}

// Nullable is helper that creates Nullable(JSON).
func (c *ColJSONBytes) Nullable() *ColNullable[[]byte] {
	return &ColNullable[[]byte]{
		Values: c,
	}
}
