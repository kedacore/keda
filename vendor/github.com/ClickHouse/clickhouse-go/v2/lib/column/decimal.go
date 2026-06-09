package column

import (
	"database/sql"
	"database/sql/driver"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"github.com/ClickHouse/ch-go/proto"

	"github.com/shopspring/decimal"
)

type Decimal struct {
	chType    Type
	scale     int
	precision int
	name      string
	col       proto.Column
}

func (col *Decimal) Name() string {
	return col.name
}

func (col *Decimal) Reset() {
	col.col.Reset()
}

func (col *Decimal) parse(t Type) (_ *Decimal, err error) {
	col.chType = t
	params := strings.Split(t.params(), ",")
	if len(params) != 2 {
		return nil, fmt.Errorf("invalid Decimal format: '%s'", t)
	}
	params[0] = strings.TrimSpace(params[0])
	params[1] = strings.TrimSpace(params[1])

	if col.precision, err = strconv.Atoi(params[0]); err != nil {
		return nil, fmt.Errorf("'%s' is not Decimal type: %s", t, err)
	} else if col.precision < 1 {
		return nil, errors.New("wrong precision of Decimal type")
	}

	if col.scale, err = strconv.Atoi(params[1]); err != nil {
		return nil, fmt.Errorf("'%s' is not Decimal type: %s", t, err)
	} else if col.scale < 0 || col.scale > col.precision {
		return nil, errors.New("wrong scale of Decimal type")
	}
	switch {
	case col.precision <= 9:
		col.col = &proto.ColDecimal32{}
	case col.precision <= 18:
		col.col = &proto.ColDecimal64{}
	case col.precision <= 38:
		col.col = &proto.ColDecimal128{}
	default:
		col.col = &proto.ColDecimal256{}
	}
	return col, nil
}

func (col *Decimal) Type() Type {
	return col.chType
}

func (col *Decimal) ScanType() reflect.Type {
	return scanTypeDecimal
}

func (col *Decimal) Rows() int {
	return col.col.Rows()
}

func (col *Decimal) Row(i int, ptr bool) any {
	value := col.row(i)
	if ptr {
		return value
	}
	return *value
}

func (col *Decimal) row(i int) *decimal.Decimal {
	var value decimal.Decimal
	switch vCol := col.col.(type) {
	case *proto.ColDecimal32:
		v := vCol.Row(i)
		value = decimal.New(int64(v), int32(-col.scale))
	case *proto.ColDecimal64:
		v := vCol.Row(i)
		value = decimal.New(int64(v), int32(-col.scale))
	case *proto.ColDecimal128:
		v := vCol.Row(i)
		b := make([]byte, 16)
		binary.LittleEndian.PutUint64(b[0:64/8], v.Low)
		binary.LittleEndian.PutUint64(b[64/8:128/8], v.High)
		bv := rawToBigInt(b, true)
		value = decimal.NewFromBigInt(bv, int32(-col.scale))
	case *proto.ColDecimal256:
		v := vCol.Row(i)
		b := make([]byte, 32)
		binary.LittleEndian.PutUint64(b[0:64/8], v.Low.Low)
		binary.LittleEndian.PutUint64(b[64/8:128/8], v.Low.High)
		binary.LittleEndian.PutUint64(b[128/8:192/8], v.High.Low)
		binary.LittleEndian.PutUint64(b[192/8:256/8], v.High.High)
		bv := rawToBigInt(b, true)
		value = decimal.NewFromBigInt(bv, int32(-col.scale))
	}
	return &value
}

func (col *Decimal) ScanRow(dest any, row int) error {
	switch d := dest.(type) {
	case *decimal.Decimal:
		*d = *col.row(row)
	case **decimal.Decimal:
		*d = new(decimal.Decimal)
		**d = *col.row(row)
	default:
		if scan, ok := dest.(sql.Scanner); ok {
			return scan.Scan(*col.row(row))
		}
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "Decimal",
		}
	}
	return nil
}

func (col *Decimal) Append(v any) (nulls []uint8, err error) {
	switch v := v.(type) {
	case []decimal.Decimal:
		nulls = make([]uint8, len(v))
		for i := range v {
			col.append(&v[i])
		}
	case []*decimal.Decimal:
		nulls = make([]uint8, len(v))
		for i := range v {
			switch {
			case v[i] != nil:
				col.append(v[i])
			default:
				nulls[i] = 1
				value := decimal.New(0, 0)
				col.append(&value)
			}
		}
	case []string:
		nulls = make([]uint8, len(v))
		for i := range v {
			d, err := decimal.NewFromString(v[i])
			if err != nil {
				return nil, fmt.Errorf("could not convert \"%v\" to decimal: %w", v[i], err)
			}
			col.append(&d)
		}
	case []*string:
		nulls = make([]uint8, len(v))
		for i := range v {
			if v[i] == nil {
				nulls[i] = 1
				value := decimal.New(0, 0)
				col.append(&value)

				continue
			}

			d, err := decimal.NewFromString(*v[i])
			if err != nil {
				return nil, fmt.Errorf("could not convert \"%v\" to decimal: %w", *v[i], err)
			}
			col.append(&d)
		}
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   string(col.chType),
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.Append(val)
		}
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   string(col.chType),
			From: fmt.Sprintf("%T", v),
		}
	}
	return
}

func (col *Decimal) AppendRow(v any) error {
	value := decimal.New(0, 0)
	switch v := v.(type) {
	case decimal.Decimal:
		value = v
	case *decimal.Decimal:
		if v != nil {
			value = *v
		}
	case string:
		d, err := decimal.NewFromString(v)
		if err != nil {
			return fmt.Errorf("could not convert \"%v\" to decimal: %w", v, err)
		}
		value = d
	case *string:
		if v != nil {
			d, err := decimal.NewFromString(*v)
			if err != nil {
				return fmt.Errorf("could not convert \"%v\" to decimal: %w", *v, err)
			}
			value = d
		}
	case nil:
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   string(col.chType),
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.AppendRow(val)
		}
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   string(col.chType),
			From: fmt.Sprintf("%T", v),
		}
	}
	col.append(&value)
	return nil
}

func (col *Decimal) append(v *decimal.Decimal) {
	switch vCol := col.col.(type) {
	case *proto.ColDecimal32:
		var part uint32
		part = uint32(decimal.NewFromBigInt(v.Coefficient(), v.Exponent()+int32(col.scale)).IntPart())
		vCol.Append(proto.Decimal32(part))
	case *proto.ColDecimal64:
		var part uint64
		part = uint64(decimal.NewFromBigInt(v.Coefficient(), v.Exponent()+int32(col.scale)).IntPart())
		vCol.Append(proto.Decimal64(part))
	case *proto.ColDecimal128:
		var bi *big.Int
		bi = decimal.NewFromBigInt(v.Coefficient(), v.Exponent()+int32(col.scale)).BigInt()
		dest := make([]byte, 16)
		bigIntToRaw(dest, bi)
		vCol.Append(proto.Decimal128{
			Low:  binary.LittleEndian.Uint64(dest[0 : 64/8]),
			High: binary.LittleEndian.Uint64(dest[64/8 : 128/8]),
		})
	case *proto.ColDecimal256:
		var bi *big.Int
		bi = decimal.NewFromBigInt(v.Coefficient(), v.Exponent()+int32(col.scale)).BigInt()
		dest := make([]byte, 32)
		bigIntToRaw(dest, bi)
		vCol.Append(proto.Decimal256{
			Low: proto.UInt128{
				Low:  binary.LittleEndian.Uint64(dest[0 : 64/8]),
				High: binary.LittleEndian.Uint64(dest[64/8 : 128/8]),
			},
			High: proto.UInt128{
				Low:  binary.LittleEndian.Uint64(dest[128/8 : 192/8]),
				High: binary.LittleEndian.Uint64(dest[192/8 : 256/8]),
			},
		})
	}
}

func (col *Decimal) Decode(reader *proto.Reader, rows int) error {
	return col.col.DecodeColumn(reader, rows)
}

func (col *Decimal) Encode(buffer *proto.Buffer) {
	col.col.EncodeColumn(buffer)
}

func (col *Decimal) Scale() int64 {
	return int64(col.scale)
}

func (col *Decimal) Precision() int64 {
	return int64(col.precision)
}

var _ Interface = (*Decimal)(nil)
