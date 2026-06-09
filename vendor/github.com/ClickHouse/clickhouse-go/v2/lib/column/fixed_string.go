package column

import (
	"database/sql"
	"database/sql/driver"
	"encoding"
	"fmt"
	"reflect"

	"github.com/ClickHouse/ch-go/proto"

	"github.com/ClickHouse/clickhouse-go/v2/lib/binary"
)

type FixedString struct {
	name string
	col  proto.ColFixedStr
}

func (col *FixedString) Reset() {
	col.col.Reset()
}

func (col *FixedString) Name() string {
	return col.name
}

func (col *FixedString) parse(t Type) (*FixedString, error) {
	if _, err := fmt.Sscanf(string(t), "FixedString(%d)", &col.col.Size); err != nil {
		return nil, err
	}
	return col, nil
}

func (col *FixedString) Type() Type {
	return Type(fmt.Sprintf("FixedString(%d)", col.col.Size))
}

func (col *FixedString) ScanType() reflect.Type {
	return scanTypeString
}

func (col *FixedString) Rows() int {
	return col.col.Rows()
}

func (col *FixedString) Row(i int, ptr bool) any {
	value := col.row(i)
	if ptr {
		return &value
	}
	return value
}

func (col *FixedString) ScanRow(dest any, row int) error {
	switch d := dest.(type) {
	case *string:
		*d = col.row(row)
	case **string:
		*d = new(string)
		**d = col.row(row)
	case encoding.BinaryUnmarshaler:
		return d.UnmarshalBinary(col.rowBytes(row))
	case *[]byte:
		*d = col.rowBytes(row)
	default:
		// handle for *[n]byte
		if t := reflect.TypeOf(dest); t.Kind() == reflect.Pointer &&
			t.Elem().Kind() == reflect.Array &&
			t.Elem().Elem() == reflect.TypeOf(byte(0)) {
			size := t.Elem().Len()
			if size != col.col.Size {
				return &ColumnConverterError{
					Op:   "ScanRow",
					To:   fmt.Sprintf("%T", dest),
					From: "FixedString",
					Hint: fmt.Sprintf("invalid size %d, expect %d", size, col.col.Size),
				}
			}
			rv := reflect.ValueOf(dest).Elem()
			reflect.Copy(rv, reflect.ValueOf(col.row(row)))
			return nil
		}

		if scan, ok := dest.(sql.Scanner); ok {
			return scan.Scan(col.row(row))
		}
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "FixedString",
		}
	}
	return nil
}

// safeAppendRow appends the value to the underlying column with a length check.
// This re-implements the logic from ch-go but without the panic.
// It also fills unused space with zeros.
func (col *FixedString) safeAppendRow(v []byte) error {
	if col.col.Size == 0 {
		// If unset, use first value's length for the string size
		col.col.Size = len(v)
	}

	if len(v) > col.col.Size {
		return fmt.Errorf("input value with length %d exceeds FixedString(%d) capacity", len(v), col.col.Size)
	}

	col.col.Buf = append(col.col.Buf, v...)

	// Fill the unused space of the fixed string with zeros
	padding := col.col.Size - len(v)
	for i := 0; i < padding; i++ {
		col.col.Buf = append(col.col.Buf, 0)
	}

	return nil
}

func (col *FixedString) Append(v any) (nulls []uint8, err error) {
	switch v := v.(type) {
	case []string:
		nulls = make([]uint8, len(v))
		for _, v := range v {
			var err error
			if v == "" {
				err = col.safeAppendRow(nil)
			} else {
				err = col.safeAppendRow(binary.Str2Bytes(v, col.col.Size))
			}

			if err != nil {
				return nil, err
			}
		}
	case []*string:
		nulls = make([]uint8, len(v))
		for i, v := range v {
			var err error
			switch {
			case v == nil:
				nulls[i] = 1
				err = col.safeAppendRow(nil)
			case *v == "":
				err = col.safeAppendRow(nil)
			default:
				err = col.safeAppendRow(binary.Str2Bytes(*v, col.col.Size))
			}

			if err != nil {
				return nil, err
			}
		}
	case encoding.BinaryMarshaler:
		data, err := v.MarshalBinary()
		if err != nil {
			return nil, err
		}
		err = col.safeAppendRow(data)
		if err != nil {
			return nil, err
		}

		var size = 0
		if col.col.Size != 0 {
			size = len(data) / col.col.Size
		}
		nulls = make([]uint8, size)

	case [][]byte:
		nulls = make([]uint8, len(v))
		for i, v := range v {
			if v == nil {
				nulls[i] = 1
			}
			n := len(v)
			var err error
			switch {
			case n == 0:
				err = col.safeAppendRow(nil)
			case n >= col.col.Size:
				err = col.safeAppendRow(v[0:col.col.Size])
			default:
				err = col.safeAppendRow(v)
			}

			if err != nil {
				return nil, err
			}
		}
	default:
		// handle for [][n]byte
		if t := reflect.TypeOf(v); t.Kind() == reflect.Slice &&
			t.Elem().Kind() == reflect.Array &&
			t.Elem().Elem() == reflect.TypeOf(byte(0)) {
			rv := reflect.ValueOf(v)
			nulls = make([]uint8, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				e := rv.Index(i)
				data := make([]byte, e.Len())
				reflect.Copy(reflect.ValueOf(data), e)
				err := col.safeAppendRow(data)
				if err != nil {
					return nil, err
				}
			}
			return
		}

		if s, ok := v.(driver.Valuer); ok {
			val, err := s.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   "FixedString",
					From: fmt.Sprintf("%T", s),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.Append(val)
		}
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   "FixedString",
			From: fmt.Sprintf("%T", v),
		}
	}
	return
}

func (col *FixedString) AppendRow(v any) error {
	switch v := v.(type) {
	case []byte:
		err := col.safeAppendRow(v)
		if err != nil {
			return err
		}
	case string:
		err := col.safeAppendRow(binary.Str2Bytes(v, col.col.Size))
		if err != nil {
			return err
		}
	case *string:
		var data []byte
		if v != nil {
			data = binary.Str2Bytes(*v, col.col.Size)
		}

		err := col.safeAppendRow(data)
		if err != nil {
			return err
		}
	case nil:
		err := col.safeAppendRow(nil)
		if err != nil {
			return err
		}
	case encoding.BinaryMarshaler:
		data, err := v.MarshalBinary()
		if err != nil {
			return err
		}

		err = col.safeAppendRow(data)
		if err != nil {
			return err
		}
	default:
		if t := reflect.TypeOf(v); t.Kind() == reflect.Array && t.Elem() == reflect.TypeOf(byte(0)) {
			if t.Len() != col.col.Size {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   "FixedString",
					From: fmt.Sprintf("%T", v),
					Hint: fmt.Sprintf("invalid size %d, expect %d", t.Len(), col.col.Size),
				}
			}

			data := make([]byte, col.col.Size)
			reflect.Copy(reflect.ValueOf(data), reflect.ValueOf(v))
			err := col.safeAppendRow(data)
			if err != nil {
				return err
			}

			return nil
		}

		if s, ok := v.(driver.Valuer); ok {
			val, err := s.Value()
			if err != nil {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   "FixedString",
					From: fmt.Sprintf("%T", s),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.AppendRow(val)
		}

		if s, ok := v.(fmt.Stringer); ok {
			return col.AppendRow(s.String())
		}

		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   "FixedString",
			From: fmt.Sprintf("%T", v),
		}
	}

	return nil
}

func (col *FixedString) Decode(reader *proto.Reader, rows int) error {
	return col.col.DecodeColumn(reader, rows)
}

func (col *FixedString) Encode(buffer *proto.Buffer) {
	col.col.EncodeColumn(buffer)
}

func (col *FixedString) row(i int) string {
	v := col.col.Row(i)
	return string(v)
}

func (col *FixedString) rowBytes(i int) []byte {
	return col.col.Row(i)
}

var _ Interface = (*FixedString)(nil)
