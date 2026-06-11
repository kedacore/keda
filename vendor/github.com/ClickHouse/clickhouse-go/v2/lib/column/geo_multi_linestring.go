package column

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/paulmach/orb"
)

type MultiLineString struct {
	set  *Array
	name string
}

func (col *MultiLineString) Reset() {
	col.set.Reset()
}

func (col *MultiLineString) Name() string {
	return col.name
}

func (col *MultiLineString) Type() Type {
	return "MultiLineString"
}

func (col *MultiLineString) ScanType() reflect.Type {
	return scanTypeMultiLineString
}

func (col *MultiLineString) Rows() int {
	return col.set.Rows()
}

func (col *MultiLineString) Row(i int, ptr bool) any {
	value := col.row(i)
	if ptr {
		return &value
	}
	return value
}

func (col *MultiLineString) ScanRow(dest any, row int) error {
	switch d := dest.(type) {
	case *orb.MultiLineString:
		*d = col.row(row)
	case **orb.MultiLineString:
		*d = new(orb.MultiLineString)
		**d = col.row(row)
	default:
		if scan, ok := dest.(sql.Scanner); ok {
			return scan.Scan(col.row(row))
		}
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "MultiLineString",
			Hint: fmt.Sprintf("try using *%s", col.ScanType()),
		}
	}
	return nil
}

func (col *MultiLineString) Append(v any) (nulls []uint8, err error) {
	switch v := v.(type) {
	case []orb.MultiLineString:
		values := make([][]orb.LineString, 0, len(v))
		for _, v := range v {
			values = append(values, v)
		}
		return col.set.Append(values)
	case []*orb.MultiLineString:
		nulls = make([]uint8, len(v))
		values := make([][]orb.LineString, 0, len(v))
		for i, v := range v {
			if v == nil {
				nulls[i] = 1
				values = append(values, orb.MultiLineString{})
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
					To:   "MultiLineString",
					From: fmt.Sprintf("%T", v),
					Hint: fmt.Sprintf("could not get driver.Valuer value, try using %s", col.Type()),
				}
			}
			return col.Append(val)
		}
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   "MultiLineString",
			From: fmt.Sprintf("%T", v),
		}
	}
}

func (col *MultiLineString) AppendRow(v any) error {
	switch v := v.(type) {
	case orb.MultiLineString:
		return col.set.AppendRow([]orb.LineString(v))
	case *orb.MultiLineString:
		return col.set.AppendRow([]orb.LineString(*v))
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   "MultiLineString",
					From: fmt.Sprintf("%T", v),
					Hint: fmt.Sprintf("could not get driver.Valuer value, try using %s", col.Type()),
				}
			}
			return col.AppendRow(val)
		}
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   "MultiLineString",
			From: fmt.Sprintf("%T", v),
		}
	}
}

func (col *MultiLineString) Decode(reader *proto.Reader, rows int) error {
	return col.set.Decode(reader, rows)
}

func (col *MultiLineString) Encode(buffer *proto.Buffer) {
	col.set.Encode(buffer)
}

func (col *MultiLineString) row(i int) orb.MultiLineString {
	var value []orb.LineString
	{
		col.set.ScanRow(&value, i)
	}
	return value
}

var _ Interface = (*MultiLineString)(nil)
