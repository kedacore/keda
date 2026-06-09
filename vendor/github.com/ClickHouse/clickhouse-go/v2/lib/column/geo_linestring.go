package column

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/paulmach/orb"
)

type LineString struct {
	set  *Array
	name string
}

func (col *LineString) Reset() {
	col.set.Reset()
}

func (col *LineString) Name() string {
	return col.name
}

func (col *LineString) Type() Type {
	return "LineString"
}

func (col *LineString) ScanType() reflect.Type {
	return scanTypeLineString
}

func (col *LineString) Rows() int {
	return col.set.Rows()
}

func (col *LineString) Row(i int, ptr bool) any {
	value := col.row(i)
	if ptr {
		return &value
	}
	return value
}

func (col *LineString) ScanRow(dest any, row int) error {
	switch d := dest.(type) {
	case *orb.LineString:
		*d = col.row(row)
	case **orb.LineString:
		*d = new(orb.LineString)
		**d = col.row(row)
	default:
		if scan, ok := dest.(sql.Scanner); ok {
			return scan.Scan(col.row(row))
		}
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "LineString",
			Hint: fmt.Sprintf("try using *%s", col.ScanType()),
		}
	}
	return nil
}

func (col *LineString) Append(v any) (nulls []uint8, err error) {
	switch v := v.(type) {
	case []orb.LineString:
		values := make([][]orb.Point, 0, len(v))
		for _, v := range v {
			values = append(values, v)
		}
		return col.set.Append(values)
	case []*orb.LineString:
		nulls = make([]uint8, len(v))
		values := make([][]orb.Point, 0, len(v))
		for i, v := range v {
			if v == nil {
				nulls[i] = 1
				values = append(values, orb.LineString{})
			} else {
				values = append(values, *v)
			}
		}
		return col.set.Append(values)
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   "LineString",
					From: fmt.Sprintf("%T", v),
					Hint: fmt.Sprintf("could not get driver.Valuer value, try using %s", col.Type()),
				}
			}
			return col.Append(val)
		}
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   "LineString",
			From: fmt.Sprintf("%T", v),
		}
	}
}

func (col *LineString) AppendRow(v any) error {
	switch v := v.(type) {
	case orb.LineString:
		return col.set.AppendRow([]orb.Point(v))
	case *orb.LineString:
		return col.set.AppendRow([]orb.Point(*v))
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   "LineString",
					From: fmt.Sprintf("%T", v),
					Hint: fmt.Sprintf("could not get driver.Valuer value, try using %s", col.Type()),
				}
			}
			return col.AppendRow(val)
		}
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   "LineString",
			From: fmt.Sprintf("%T", v),
		}
	}
}

func (col *LineString) Decode(reader *proto.Reader, rows int) error {
	return col.set.Decode(reader, rows)
}

func (col *LineString) Encode(buffer *proto.Buffer) {
	col.set.Encode(buffer)
}

func (col *LineString) row(i int) orb.LineString {
	var value []orb.Point
	{
		col.set.ScanRow(&value, i)
	}
	return value
}

var _ Interface = (*LineString)(nil)
