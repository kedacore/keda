package column

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ClickHouse/ch-go/proto"
)

type SimpleAggregateFunction struct {
	base   Interface
	chType Type
	name   string
}

func (col *SimpleAggregateFunction) Reset() {
	col.base.Reset()
}

func (col *SimpleAggregateFunction) Name() string {
	return col.name
}

func (col *SimpleAggregateFunction) parse(t Type, sc *ServerContext) (_ Interface, err error) {
	col.chType = t
	base := strings.TrimSpace(strings.SplitN(t.params(), ",", 2)[1])
	if col.base, err = Type(base).Column(col.name, sc); err == nil {
		return col, nil
	}
	return nil, &UnsupportedColumnTypeError{
		t: t,
	}
}

func (col *SimpleAggregateFunction) Type() Type {
	return col.chType
}
func (col *SimpleAggregateFunction) ScanType() reflect.Type {
	return col.base.ScanType()
}
func (col *SimpleAggregateFunction) Rows() int {
	return col.base.Rows()
}
func (col *SimpleAggregateFunction) Row(i int, ptr bool) any {
	return col.base.Row(i, ptr)
}
func (col *SimpleAggregateFunction) ScanRow(dest any, rows int) error {
	return col.base.ScanRow(dest, rows)
}
func (col *SimpleAggregateFunction) Append(v any) ([]uint8, error) {
	return col.base.Append(v)
}
func (col *SimpleAggregateFunction) AppendRow(v any) error {
	return col.base.AppendRow(v)
}
func (col *SimpleAggregateFunction) Decode(reader *proto.Reader, rows int) error {
	return col.base.Decode(reader, rows)
}
func (col *SimpleAggregateFunction) Encode(buffer *proto.Buffer) {
	col.base.Encode(buffer)
}

func (col *SimpleAggregateFunction) ReadStatePrefix(reader *proto.Reader) error {
	if serialize, ok := col.base.(CustomSerialization); ok {
		if err := serialize.ReadStatePrefix(reader); err != nil {
			return fmt.Errorf("failed to read prefix for SimpleAggregateFunction base type %s: %w", col.base.Type(), err)
		}
	}

	return nil
}

func (col *SimpleAggregateFunction) WriteStatePrefix(buffer *proto.Buffer) error {
	if serialize, ok := col.base.(CustomSerialization); ok {
		if err := serialize.WriteStatePrefix(buffer); err != nil {
			return fmt.Errorf("failed to write prefix for SimpleAggregateFunction base type %s: %w", col.base.Type(), err)
		}
	}

	return nil
}

var _ Interface = (*SimpleAggregateFunction)(nil)
var _ CustomSerialization = (*SimpleAggregateFunction)(nil)
