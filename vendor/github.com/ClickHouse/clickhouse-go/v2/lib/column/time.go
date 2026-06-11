package column

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/ClickHouse/ch-go/proto"
)

type Time struct {
	chType Type
	name   string
	col    proto.ColTime
}

func (col *Time) Reset() {
	col.col.Reset()
}

func (col *Time) Name() string {
	return col.name
}

func (col *Time) Type() Type {
	return col.chType
}

func (col *Time) ScanType() reflect.Type {
	return scanTypeDuration
}

func (col *Time) Rows() int {
	return col.col.Rows()
}

func (col *Time) Row(i int, ptr bool) any {
	value := col.row(i)
	if ptr {
		return &value
	}
	return value
}

// ScanRow implements column.Interface.
// It is used to read a single column value of the row and store in
// `dest` Go variable.
func (col *Time) ScanRow(dest any, row int) error {
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
			From: "Time",
		}
	}
	return nil
}

// Append implements column.Interface.
// It is used for columnar inserts. Insert multiple Go value for
// single ClickHouse Time type.
func (col *Time) Append(v any) (nulls []uint8, err error) {
	switch v := v.(type) {
	case []time.Duration:
		nulls = make([]uint8, len(v)) // default all zeros, meaning no null values
		for i := range v {
			col.col.Append(proto.IntoTime32(v[i]))
		}
	case []*time.Duration:
		nulls = make([]uint8, len(v))
		for i := range v {
			switch {
			case v[i] != nil:
				col.col.Append(proto.IntoTime32(*v[i]))
			default:
				col.col.Append(proto.IntoTime32(time.Duration(0)))
				nulls[i] = 1
			}
		}
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   "Time",
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.Append(val)
		}
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   "Time",
			From: fmt.Sprintf("%T", v),
		}
	}
	return
}

// AppendRow implements column.Interface.
// It is used to insert column value in a row.
// Converts Go type into ClickHouse type to be inserted.
func (col *Time) AppendRow(v any) error {
	switch v := v.(type) {
	case time.Duration:
		col.col.Append(proto.IntoTime32(v))
	case *time.Duration:
		switch {
		case v != nil:
			col.col.Append(proto.IntoTime32(*v))
		default:
			col.col.Append(proto.IntoTime32(time.Duration(0)))
		}
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   "Time",
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.AppendRow(val)
		}
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   "Time",
			From: fmt.Sprintf("%T", v),
		}
	}
	return nil
}

func (col *Time) Decode(reader *proto.Reader, rows int) error {
	return col.col.DecodeColumn(reader, rows)
}

func (col *Time) Encode(buffer *proto.Buffer) {
	col.col.EncodeColumn(buffer)
}

func (col *Time) row(i int) time.Duration {
	return col.col.Row(i).Duration()
}

func (col *Time) parseTime(value string) (time.Duration, error) {
	return parseDuration(value)
}

// helpers

func parseDuration(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Duration(0), nil
	}

	return time.ParseDuration(value)
}
