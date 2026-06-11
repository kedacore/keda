package column

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"time"

	"github.com/ClickHouse/ch-go/proto"
)

type Nullable struct {
	base     Interface
	nulls    proto.ColUInt8
	enable   bool
	scanType reflect.Type
	name     string
}

func (col *Nullable) Reset() {
	col.base.Reset()
	col.nulls.Reset()
}

func (col *Nullable) Name() string {
	return col.name
}

func (col *Nullable) parse(t Type, sc *ServerContext) (_ *Nullable, err error) {
	col.enable = true
	if col.base, err = Type(t.params()).Column(col.name, sc); err != nil {
		return nil, err
	}
	switch base := col.base.ScanType(); {
	case base == nil:
		col.scanType = reflect.TypeOf(nil)
	case base.Kind() == reflect.Ptr:
		col.scanType = base
	default:
		col.scanType = reflect.New(base).Type()
	}
	return col, nil
}

func (col *Nullable) Base() Interface {
	return col.base
}

func (col *Nullable) Type() Type {
	return "Nullable(" + col.base.Type() + ")"
}

func (col *Nullable) ScanType() reflect.Type {
	return col.scanType
}

func (col *Nullable) Rows() int {
	if !col.enable {
		return col.base.Rows()
	}
	return col.nulls.Rows()
}

func (col *Nullable) Row(i int, ptr bool) any {
	if col.enable {
		if col.nulls.Row(i) == 1 {
			return nil
		}
	}
	return col.base.Row(i, true)
}

func (col *Nullable) ScanRow(dest any, row int) error {
	if col.enable {
		switch col.nulls.Row(row) {
		case 1:
			switch v := dest.(type) {
			case **uint64:
				*v = nil
			case **int64:
				*v = nil
			case **uint32:
				*v = nil
			case **int32:
				*v = nil
			case **uint16:
				*v = nil
			case **int16:
				*v = nil
			case **uint8:
				*v = nil
			case **int8:
				*v = nil
			case **string:
				*v = nil
			case **float32:
				*v = nil
			case **float64:
				*v = nil
			case **time.Time:
				*v = nil
			}
			if scan, ok := dest.(sql.Scanner); ok {
				return scan.Scan(nil)
			}
			return nil
		}
	}
	return col.base.ScanRow(dest, row)
}

func (col *Nullable) Append(v any) ([]uint8, error) {
	nulls, err := col.base.Append(v)
	if err != nil {
		return nil, err
	}
	for i := range nulls {
		col.nulls.Append(nulls[i])
	}
	return nulls, nil
}

func (col *Nullable) AppendRow(v any) error {
	// Might receive double pointers like **String, because of how Nullable columns are read
	// Unpack because we can't write double pointers
	rv := reflect.ValueOf(v)
	if v != nil && rv.Kind() == reflect.Pointer && !rv.IsNil() && rv.Elem().Kind() == reflect.Pointer {
		v = rv.Elem().Interface()
		rv = reflect.ValueOf(v)
	}

	if v == nil || ((rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Map) && rv.IsNil()) {
		col.nulls.Append(1)
		// used to detect sql.Null* types
	} else if val, ok := v.(driver.Valuer); ok {
		val, err := val.Value()
		if err != nil {
			return err
		}
		if val == nil {
			col.nulls.Append(1)
		} else {
			col.nulls.Append(0)
		}
	} else {
		col.nulls.Append(0)
	}
	return col.base.AppendRow(v)
}

func (col *Nullable) Decode(reader *proto.Reader, rows int) error {
	if col.enable {
		if err := col.nulls.DecodeColumn(reader, rows); err != nil {
			return err
		}
	}
	if err := col.base.Decode(reader, rows); err != nil {
		return err
	}
	return nil
}

func (col *Nullable) Encode(buffer *proto.Buffer) {
	if col.enable {
		col.nulls.EncodeColumn(buffer)
	}
	col.base.Encode(buffer)
}

func (col *Nullable) ReadStatePrefix(reader *proto.Reader) error {
	if serialize, ok := col.base.(CustomSerialization); ok {
		if err := serialize.ReadStatePrefix(reader); err != nil {
			return fmt.Errorf("failed to read prefix for Nullable base type %s: %w", col.base.Type(), err)
		}
	}

	return nil
}

func (col *Nullable) WriteStatePrefix(buffer *proto.Buffer) error {
	if serialize, ok := col.base.(CustomSerialization); ok {
		if err := serialize.WriteStatePrefix(buffer); err != nil {
			return fmt.Errorf("failed to write prefix for Nullable base type %s: %w", col.base.Type(), err)
		}
	}

	return nil
}

var _ Interface = (*Nullable)(nil)
