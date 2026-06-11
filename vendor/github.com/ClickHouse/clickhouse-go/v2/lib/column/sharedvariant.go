package column

import (
	"reflect"

	"github.com/ClickHouse/ch-go/proto"
)

// SharedVariant deprecated. Use Dynamic/JSON serialization version 3.
type SharedVariant struct {
	name       string
	stringData String
}

func (c *SharedVariant) Name() string {
	return c.name
}

func (c *SharedVariant) Type() Type {
	return "SharedVariant"
}

func (c *SharedVariant) Rows() int {
	return c.stringData.Rows()
}

func (c *SharedVariant) Row(i int, ptr bool) any {
	return c.stringData.Row(i, ptr)
}

func (c *SharedVariant) ScanRow(dest any, row int) error {
	return c.stringData.ScanRow(dest, row)
}

func (c *SharedVariant) Append(v any) (nulls []uint8, err error) {
	return c.stringData.Append(v)
}

func (c *SharedVariant) AppendRow(v any) error {
	return c.stringData.AppendRow(v)
}

func (c *SharedVariant) Encode(buffer *proto.Buffer) {
	c.stringData.Encode(buffer)
}

func (c *SharedVariant) Decode(reader *proto.Reader, rows int) error {
	return c.stringData.Decode(reader, rows)
}

func (c *SharedVariant) ScanType() reflect.Type {
	return c.stringData.ScanType()
}

func (c *SharedVariant) Reset() {
	c.stringData.Reset()
}
