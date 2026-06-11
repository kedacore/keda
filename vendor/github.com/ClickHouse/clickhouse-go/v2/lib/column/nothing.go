package column

import (
	"errors"
	"reflect"

	"github.com/ClickHouse/ch-go/proto"
)

type Nothing struct {
	name string
	col  proto.ColNothing
}

func (col *Nothing) Reset() {
	col.col.Reset()
}

func (col Nothing) Name() string {
	return col.name
}

func (Nothing) Type() Type             { return "Nothing" }
func (Nothing) ScanType() reflect.Type { return reflect.TypeOf((*any)(nil)) }
func (Nothing) Rows() int              { return 0 }
func (Nothing) Row(int, bool) any      { return nil }
func (Nothing) ScanRow(any, int) error {
	return nil
}
func (Nothing) Append(any) ([]uint8, error) {
	return nil, &Error{
		ColumnType: "Nothing",
		Err:        errors.New("data type values can't be stored in tables"),
	}
}
func (col Nothing) AppendRow(any) error {
	return &Error{
		ColumnType: "Nothing",
		Err:        errors.New("data type values can't be stored in tables"),
	}
}

func (col Nothing) Decode(reader *proto.Reader, rows int) error {
	return col.col.DecodeColumn(reader, rows)
}

func (Nothing) Encode(buffer *proto.Buffer) {
}

var _ Interface = (*Nothing)(nil)
