package column

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/ch-go/proto"
)

type Time64 struct {
	chType Type
	name   string
	col    proto.ColTime64
}

func (col *Time64) Reset() {
	col.col.Reset()
}

func (col *Time64) Name() string {
	return col.name
}

func (col *Time64) parse(t Type) (_ Interface, err error) {
	col.chType = t
	// if no precision is given say just Time64 (instead of Time64(3|6|9))
	// it is treated as 3 (milliseconds)
	precision := int64(3)

	if strings.HasPrefix(string(t), "Time64(") {
		params := strings.TrimSuffix(strings.TrimPrefix(string(t), "Time64("), ")")
		precision, err = strconv.ParseInt(params, 10, 8)
		if err != nil {
			return nil, err
		}
	}
	p := byte(precision)
	col.col.WithPrecision(proto.Precision(p))
	return col, nil

}

func (col *Time64) Type() Type {
	return col.chType
}

func (col *Time64) ScanType() reflect.Type {
	return scanTypeDuration
}

func (col *Time64) Precision() (int64, bool) {
	return int64(col.col.Precision), col.col.PrecisionSet
}

func (col *Time64) Rows() int {
	return col.col.Rows()
}

func (col *Time64) Row(i int, ptr bool) any {
	value := col.row(i)
	if ptr {
		return &value
	}
	return value
}

// ScanRow implements column.Interface.
// It is used to read a single column value of the row and store in
// `dest` Go variable.
func (col *Time64) ScanRow(dest any, row int) error {
	switch d := dest.(type) {
	case *time.Duration:
		*d = col.row(row)
	case **time.Duration:
		*d = new(time.Duration)
		**d = col.row(row)
	default:
		if scan, ok := dest.(sql.Scanner); ok {
			return scan.Scan(col.row(row))
		}
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "Time64",
		}
	}
	return nil
}

// Append implements column.Interface.
// It is used for columnar inserts. Insert multiple Go value for
// single ClickHouse Time64 type.
func (col *Time64) Append(v any) (nulls []uint8, err error) {
	switch v := v.(type) {
	case []time.Duration:
		nulls = make([]uint8, len(v)) // default all zeros, meaning no null values
		for i := range v {
			col.col.Append(proto.IntoTime64WithPrecision(v[i], col.col.Precision))
		}
	case []*time.Duration:
		nulls = make([]uint8, len(v))
		for i := range v {
			switch {
			case v[i] != nil:
				col.col.Append(proto.IntoTime64WithPrecision(*v[i], col.col.Precision))
			default:
				col.col.Append(proto.IntoTime64WithPrecision(time.Duration(0), col.col.Precision))
				nulls[i] = 1
			}
		}
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   "Time64",
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.Append(val)
		}
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   "Time64",
			From: fmt.Sprintf("%T", v),
		}
	}
	return
}

// AppendRow implements column.Interface.
// It is used to insert column value in a row.
// Converts Go type into ClickHouse type to be inserted.
func (col *Time64) AppendRow(v any) error {
	switch v := v.(type) {
	case time.Duration:
		col.col.Append(proto.IntoTime64WithPrecision(v, col.col.Precision))
	case *time.Duration:
		switch {
		case v != nil:
			col.col.Append(proto.IntoTime64WithPrecision(*v, col.col.Precision))
		default:
			col.col.Append(proto.IntoTime64WithPrecision(time.Duration(0), col.col.Precision))
		}
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   "Time64",
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.AppendRow(val)
		}
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   "Time64",
			From: fmt.Sprintf("%T", v),
		}
	}
	return nil
}

func (col *Time64) Decode(reader *proto.Reader, rows int) error {
	return col.col.DecodeColumn(reader, rows)
}

func (col *Time64) Encode(buffer *proto.Buffer) {
	col.col.EncodeColumn(buffer)
}

func (col *Time64) row(i int) time.Duration {
	return col.col.Row(i).ToDurationWithPrecision(col.col.Precision)
}

func (col *Time64) parseTime(value string) (time.Duration, error) {
	return parseDuration(value)
}
