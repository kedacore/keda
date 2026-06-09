package proto

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-faster/errors"
)

// ColInput column.
type ColInput interface {
	Type() ColumnType
	Rows() int
	EncodeColumn(b *Buffer)
	WriteColumn(w *Writer)
}

// ColResult column.
type ColResult interface {
	Type() ColumnType
	Rows() int
	DecodeColumn(r *Reader, rows int) error
	Resettable
}

type Column interface {
	ColResult
	ColInput
}

// ColumnOf is generic Column(T) constraint.
type ColumnOf[T any] interface {
	Column
	Append(v T)
	AppendArr(v []T)
	Row(i int) T
}

type StateEncoder interface {
	EncodeState(b *Buffer)
}

type StateDecoder interface {
	DecodeState(r *Reader) error
}

type Stateful interface {
	StateEncoder
	StateDecoder
}

// Inferable can be inferenced from type.
type Inferable interface {
	Infer(t ColumnType) error
}

// Preparable should be prepared before encoding or decoding.
type Preparable interface {
	Prepare() error
}

// TODO: merge preparable with inferable?

// ColumnType is type of column element.
type ColumnType string

func (c ColumnType) String() string {
	return string(c)
}

func (c ColumnType) Base() ColumnType {
	if c == "" {
		return ""
	}
	var (
		v     = string(c)
		start = strings.IndexByte(v, '(')
		end   = strings.LastIndexByte(v, ')')
	)
	if start <= 0 || end <= 0 || end < start {
		return c
	}
	return c[:start]
}

// reduces Decimal(P, ...) to Decimal32/Decimal64/Decimal128/Decimal256
// returns c if any errors occur during conversion
func (c ColumnType) decimalDowncast() ColumnType {
	if c.Base() != ColumnTypeDecimal {
		return c
	}
	elem := c.Elem()
	precStr, _, _ := strings.Cut(string(elem), ",")
	precStr = strings.TrimSpace(precStr)
	prec, err := strconv.Atoi(precStr)
	if err != nil {
		return c
	}
	switch {
	case prec < 10:
		return ColumnTypeDecimal32
	case prec < 19:
		return ColumnTypeDecimal64
	case prec < 39:
		return ColumnTypeDecimal128
	case prec < 77:
		return ColumnTypeDecimal256
	default:
		return c
	}
}

// Conflicts reports whether two types conflict.
func (c ColumnType) Conflicts(b ColumnType) bool {
	if c == b {
		return false
	}
	cBase := c.Base()
	bBase := b.Base()
	if (cBase == ColumnTypeEnum8 && b == ColumnTypeInt8) ||
		(cBase == ColumnTypeEnum16 && b == ColumnTypeInt16) ||
		(bBase == ColumnTypeEnum8 && c == ColumnTypeInt8) ||
		(bBase == ColumnTypeEnum16 && c == ColumnTypeInt16) {
		return false
	}
	if cBase == ColumnTypeDecimal || bBase == ColumnTypeDecimal {
		return c.decimalDowncast() != b.decimalDowncast()
	}

	if cBase != bBase {
		return true
	}
	switch cBase {
	case ColumnTypeEnum8, ColumnTypeEnum16:
		return false
	}

	if c.normalizeCommas() == b.normalizeCommas() {
		return false
	}
	switch cBase {
	case ColumnTypeArray, ColumnTypeNullable, ColumnTypeLowCardinality:
		return c.Elem().Conflicts(b.Elem())
	case ColumnTypeDateTime, ColumnTypeDateTime64:
		// TODO(ernado): improve check
		return false
	}
	return true
}

func (c ColumnType) normalizeCommas() ColumnType {
	// Should we check for escaped commas in enums here?
	const sep = ","
	var elems []string
	for _, e := range strings.Split(string(c), sep) {
		elems = append(elems, strings.TrimSpace(e))
	}
	return ColumnType(strings.Join(elems, sep))
}

// With returns ColumnType(p1, p2, ...) from ColumnType.
func (c ColumnType) With(params ...string) ColumnType {
	if len(params) == 0 {
		return c
	}
	s := fmt.Sprintf("%s(%s)",
		c, strings.Join(params, ", "),
	)
	return ColumnType(s)
}

// Sub of T returns T(A, B, ...).
func (c ColumnType) Sub(subtypes ...ColumnType) ColumnType {
	var params []string
	for _, t := range subtypes {
		params = append(params, t.String())
	}
	return c.With(params...)
}

func (c ColumnType) Elem() ColumnType {
	if c == "" {
		return ""
	}
	var (
		v     = string(c)
		start = strings.IndexByte(v, '(')
		end   = strings.LastIndexByte(v, ')')
	)
	if start <= 0 || end <= 0 || end < start {
		// No element.
		return ""
	}
	return c[start+1 : end]
}

// IsArray reports whether ColumnType is composite.
func (c ColumnType) IsArray() bool {
	return strings.HasPrefix(string(c), string(ColumnTypeArray))
}

// Array returns Array(ColumnType).
func (c ColumnType) Array() ColumnType {
	return ColumnTypeArray.Sub(c)
}

// Common colum type names. Does not represent full set of supported types,
// because ColumnTypeArray is composable; actual type is composite.
//
// For example: Array(Int8) or even Array(Array(String)).
const (
	ColumnTypeNone           ColumnType = ""
	ColumnTypeInt8           ColumnType = "Int8"
	ColumnTypeInt16          ColumnType = "Int16"
	ColumnTypeInt32          ColumnType = "Int32"
	ColumnTypeInt64          ColumnType = "Int64"
	ColumnTypeInt128         ColumnType = "Int128"
	ColumnTypeInt256         ColumnType = "Int256"
	ColumnTypeUInt8          ColumnType = "UInt8"
	ColumnTypeUInt16         ColumnType = "UInt16"
	ColumnTypeUInt32         ColumnType = "UInt32"
	ColumnTypeUInt64         ColumnType = "UInt64"
	ColumnTypeUInt128        ColumnType = "UInt128"
	ColumnTypeUInt256        ColumnType = "UInt256"
	ColumnTypeFloat32        ColumnType = "Float32"
	ColumnTypeFloat64        ColumnType = "Float64"
	ColumnTypeBFloat16       ColumnType = "BFloat16"
	ColumnTypeString         ColumnType = "String"
	ColumnTypeFixedString    ColumnType = "FixedString"
	ColumnTypeArray          ColumnType = "Array"
	ColumnTypeIPv4           ColumnType = "IPv4"
	ColumnTypeIPv6           ColumnType = "IPv6"
	ColumnTypeDateTime       ColumnType = "DateTime"
	ColumnTypeDateTime64     ColumnType = "DateTime64"
	ColumnTypeTime32         ColumnType = "Time32"
	ColumnTypeTime64         ColumnType = "Time64"
	ColumnTypeDate           ColumnType = "Date"
	ColumnTypeDate32         ColumnType = "Date32"
	ColumnTypeUUID           ColumnType = "UUID"
	ColumnTypeEnum8          ColumnType = "Enum8"
	ColumnTypeEnum16         ColumnType = "Enum16"
	ColumnTypeLowCardinality ColumnType = "LowCardinality"
	ColumnTypeMap            ColumnType = "Map"
	ColumnTypeBool           ColumnType = "Bool"
	ColumnTypeTuple          ColumnType = "Tuple"
	ColumnTypeNullable       ColumnType = "Nullable"
	ColumnTypeDecimal        ColumnType = "Decimal"
	ColumnTypeDecimal32      ColumnType = "Decimal32"
	ColumnTypeDecimal64      ColumnType = "Decimal64"
	ColumnTypeDecimal128     ColumnType = "Decimal128"
	ColumnTypeDecimal256     ColumnType = "Decimal256"
	ColumnTypePoint          ColumnType = "Point"
	ColumnTypeInterval       ColumnType = "Interval"
	ColumnTypeNothing        ColumnType = "Nothing"
	ColumnTypeJSON           ColumnType = "JSON"
	ColumnTypeQBit           ColumnType = "QBit"
)

// colWrap wraps Column with type t.
type colWrap struct {
	Column
	t ColumnType
}

func (c colWrap) Type() ColumnType { return c.t }

// Wrap Column with type parameters.
//
// So if c type is T, result type will be T(arg0, arg1, ...).
func Wrap(c Column, args ...interface{}) Column {
	var params []string
	for _, a := range args {
		params = append(params, fmt.Sprint(a))
	}
	t := c.Type().With(params...)
	return Alias(c, t)
}

// Alias column as other type.
//
// E.g. Bool is domain of UInt8, so can be aliased from UInt8.
func Alias(c Column, t ColumnType) Column {
	return colWrap{
		Column: c,
		t:      t,
	}
}

// ColInfo wraps Name and Type of column.
type ColInfo struct {
	Name string
	Type ColumnType
}

// ColInfoInput saves column info on decoding.
type ColInfoInput []ColInfo

func (s *ColInfoInput) Reset() {
	*s = (*s)[:0]
}

func (s *ColInfoInput) DecodeResult(r *Reader, version int, b Block) error {
	s.Reset()
	if b.Rows > 0 {
		return errors.New("got unexpected rows")
	}
	for i := 0; i < b.Columns; i++ {
		columnName, err := r.Str()
		if err != nil {
			return errors.Wrapf(err, "column [%d] name", i)
		}
		columnTypeRaw, err := r.Str()
		if err != nil {
			return errors.Wrapf(err, "column [%d] type", i)
		}
		if FeatureCustomSerialization.In(version) {
			customSerialization, err := r.Bool()
			if err != nil {
				return errors.Wrapf(err, "column [%d] custom serialization", i)
			}
			if customSerialization {
				return errors.Wrapf(err, "column [%d] has custom serialization (not supported)", i)
			}
		}
		*s = append(*s, ColInfo{
			Name: columnName,
			Type: ColumnType(columnTypeRaw),
		})
	}
	return nil
}
