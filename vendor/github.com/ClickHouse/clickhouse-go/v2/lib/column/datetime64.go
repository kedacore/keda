package column

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/ch-go/proto"

	"github.com/ClickHouse/clickhouse-go/v2/lib/timezone"
)

var (
	minDateTime64, _ = time.Parse("2006-01-02 15:04:05", "1900-01-01 00:00:00")
	maxDateTime64, _ = time.Parse("2006-01-02 15:04:05", "2262-04-11 23:47:16")
)

const (
	defaultDateTime64FormatNoZone   = "2006-01-02 15:04:05.999999999"
	defaultDateTime64FormatWithZone = "2006-01-02 15:04:05.999999999 -07:00"
)

type DateTime64 struct {
	chType   Type
	timezone *time.Location
	name     string
	col      proto.ColDateTime64
}

func (col *DateTime64) Reset() {
	col.col.Reset()
}

func (col *DateTime64) Name() string {
	return col.name
}

func (col *DateTime64) parse(t Type, tz *time.Location) (_ Interface, err error) {
	col.chType = t
	switch params := strings.Split(t.params(), ","); len(params) {
	case 2:
		precision, err := strconv.ParseInt(params[0], 10, 8)
		if err != nil {
			return nil, err
		}
		p := byte(precision)
		col.col.WithPrecision(proto.Precision(p))
		timezone, err := timezone.Load(params[1][2 : len(params[1])-1])
		if err != nil {
			return nil, err
		}
		col.col.WithLocation(timezone)
	case 1:
		precision, err := strconv.ParseInt(params[0], 10, 8)
		if err != nil {
			return nil, err
		}
		p := byte(precision)
		col.col.WithPrecision(proto.Precision(p))
		col.col.WithLocation(tz)
	default:
		return nil, &UnsupportedColumnTypeError{
			t: t,
		}
	}
	return col, nil
}

func (col *DateTime64) Type() Type {
	return col.chType
}

func (col *DateTime64) ScanType() reflect.Type {
	return scanTypeTime
}

func (col *DateTime64) Precision() (int64, bool) {
	return int64(col.col.Precision), col.col.PrecisionSet
}

func (col *DateTime64) Rows() int {
	return col.col.Rows()
}

func (col *DateTime64) Row(i int, ptr bool) any {
	value := col.row(i)
	if ptr {
		return &value
	}
	return value
}

func (col *DateTime64) ScanRow(dest any, row int) error {
	switch d := dest.(type) {
	case *time.Time:
		*d = col.row(row)
	case **time.Time:
		*d = new(time.Time)
		**d = col.row(row)
	case *int64:
		*d = int64(proto.ToDateTime64(col.row(row), col.col.Precision))
	case **int64:
		*d = new(int64)
		**d = int64(proto.ToDateTime64(col.row(row), col.col.Precision))
	case *sql.NullTime:
		return d.Scan(col.row(row))
	default:
		if scan, ok := dest.(sql.Scanner); ok {
			return scan.Scan(col.row(row))
		}
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "Datetime64",
		}
	}
	return nil
}

func (col *DateTime64) Append(v any) (nulls []uint8, err error) {
	switch v := v.(type) {
	// we assume int64 is in milliseconds and don't currently scale to the precision - no tests to indicate intended
	// historical behaviour
	case []int64:
		nulls = make([]uint8, len(v))
		for i := range v {
			col.col.Append(time.UnixMilli(v[i]))
		}
	case []*int64:
		nulls = make([]uint8, len(v))
		for i := range v {
			switch {
			case v[i] != nil:
				col.col.Append(time.UnixMilli(*v[i]))
			default:
				col.col.Append(time.UnixMilli(0))
				nulls[i] = 1
			}
		}
	case []time.Time:
		nulls = make([]uint8, len(v))
		for i := range v {
			col.col.Append(v[i])
		}
	case []*time.Time:
		nulls = make([]uint8, len(v))
		for i := range v {
			switch {
			case v[i] != nil:
				col.col.Append(*v[i])
			default:
				col.col.Append(time.Time{})
				nulls[i] = 1
			}
		}
	case []string:
		nulls = make([]uint8, len(v))
		for i := range v {
			value, err := col.parseDateTime(v[i])
			if err != nil {
				return nil, err
			}
			col.col.Append(value)
		}
	case []sql.NullTime:
		nulls = make([]uint8, len(v))
		for i := range v {
			col.AppendRow(v[i])
		}
	case []*sql.NullTime:
		nulls = make([]uint8, len(v))
		for i := range v {
			if v[i] == nil {
				nulls[i] = 1
			}
			col.AppendRow(v[i])
		}
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   "Datetime64",
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.Append(val)
		}
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   "Datetime64",
			From: fmt.Sprintf("%T", v),
		}
	}
	return
}

func (col *DateTime64) AppendRow(v any) error {
	switch v := v.(type) {
	case int64:
		col.col.Append(time.UnixMilli(v))
	case *int64:
		switch {
		case v != nil:
			col.col.Append(time.UnixMilli(*v))
		default:
			col.col.Append(time.Time{})
		}
	case time.Time:
		col.col.Append(v)
	case *time.Time:
		switch {
		case v != nil:
			col.col.Append(*v)
		default:
			col.col.Append(time.Time{})
		}
	case sql.NullTime:
		switch v.Valid {
		case true:
			col.col.Append(v.Time)
		default:
			col.col.Append(time.Time{})
		}
	case *sql.NullTime:
		switch v.Valid {
		case true:
			col.col.Append(v.Time)
		default:
			col.col.Append(time.Time{})
		}
	case string:
		datetime, err := col.parseDateTime(v)
		if err != nil {
			return err
		}
		col.col.Append(datetime)
	case nil:
		col.col.Append(time.Time{})
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   "Datetime64",
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.AppendRow(val)
		}
		s, ok := v.(fmt.Stringer)
		if ok {
			return col.AppendRow(s.String())
		}
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   "Datetime64",
			From: fmt.Sprintf("%T", v),
		}
	}
	return nil
}

func (col *DateTime64) Decode(reader *proto.Reader, rows int) error {
	return col.col.DecodeColumn(reader, rows)
}

func (col *DateTime64) Encode(buffer *proto.Buffer) {
	col.col.EncodeColumn(buffer)
}

func (col *DateTime64) row(i int) time.Time {
	time := col.col.Row(i)
	if col.timezone != nil {
		time = time.In(col.timezone)
	}
	return time
}

func (col *DateTime64) timeToInt64(t time.Time) int64 {
	var timestamp int64
	if !t.IsZero() {
		timestamp = t.UnixNano()
	}
	return timestamp / int64(math.Pow10(9-int(col.col.Precision)))
}

func (col *DateTime64) parseDateTime(value string) (tv time.Time, err error) {
	if tv, err = time.Parse(defaultDateTime64FormatWithZone, value); err == nil {
		return tv, nil
	}
	if tv, err = time.Parse(defaultDateTime64FormatNoZone, value); err == nil {
		return getTimeWithDifferentLocation(tv, time.Local), nil
	}
	return time.Time{}, err
}

var _ Interface = (*DateTime64)(nil)
