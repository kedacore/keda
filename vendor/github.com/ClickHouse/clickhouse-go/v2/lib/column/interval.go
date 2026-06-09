package column

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/ClickHouse/ch-go/proto"
)

type Interval struct {
	chType Type
	name   string
	col    proto.ColInt64
}

func (col *Interval) Reset() {
	col.col.Reset()
}

func (col *Interval) Name() string {
	return col.name
}

func (col *Interval) parse(t Type) (Interface, error) {
	switch col.chType = t; col.chType {
	case "IntervalNanosecond", "IntervalMicrosecond", "IntervalMillisecond", "IntervalSecond", "IntervalMinute", "IntervalHour", "IntervalDay", "IntervalWeek", "IntervalMonth", "IntervalQuarter", "IntervalYear":
		return col, nil
	}
	return nil, &UnsupportedColumnTypeError{
		t: t,
	}
}

func (col *Interval) Type() Type             { return col.chType }
func (col *Interval) ScanType() reflect.Type { return scanTypeString }
func (col *Interval) Rows() int              { return col.col.Rows() }
func (col *Interval) Row(i int, ptr bool) any {
	val := col.row(i)
	if ptr {
		return &val
	}
	return val
}
func (col *Interval) ScanRow(dest any, row int) error {
	switch d := dest.(type) {
	case *string:
		*d = col.row(row)
	case **string:
		*d = new(string)
		**d = col.row(row)
	default:
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "Interval",
		}
	}
	return nil
}

func (Interval) Append(any) ([]uint8, error) {
	return nil, &Error{
		ColumnType: "Interval",
		Err:        errors.New("data type values can't be stored in tables"),
	}
}

func (Interval) AppendRow(any) error {
	return &Error{
		ColumnType: "Interval",
		Err:        errors.New("data type values can't be stored in tables"),
	}
}

func (col *Interval) Decode(reader *proto.Reader, rows int) error {
	return col.col.DecodeColumn(reader, rows)
}

func (Interval) Encode(buffer *proto.Buffer) {
}

func (col *Interval) row(i int) string {
	val := col.col.Row(i)
	v := fmt.Sprintf("%d %s", val, strings.TrimPrefix(string(col.chType), "Interval"))
	if val > 1 {
		v += "s"
	}
	return v
}

var _ Interface = (*Interval)(nil)
