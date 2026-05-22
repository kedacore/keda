package query

import (
	"encoding/csv"
	"github.com/Azure/azure-kusto-go/azkustodata/errors"
	"github.com/Azure/azure-kusto-go/azkustodata/types"
	"github.com/Azure/azure-kusto-go/azkustodata/value"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"reflect"
	"strings"
	"time"
)

type row struct {
	columns      Columns
	columnByName func(string) Column
	values       value.Values
	ordinal      int
}

func NewRow(t BaseTable, ordinal int, values value.Values) Row {
	return NewRowFromParts(t.Columns(), t.ColumnByName, ordinal, values)
}

func NewRowFromParts(c Columns, columnByName func(string) Column, ordinal int, values value.Values) Row {
	return &row{
		columns:      c,
		columnByName: columnByName,
		ordinal:      ordinal,
		values:       values,
	}
}

func (r *row) Columns() Columns {
	return r.columns
}

func (r *row) Index() int {
	return r.ordinal
}

func (r *row) Values() value.Values {
	return r.values
}

func (r *row) Value(i int) (value.Kusto, error) {
	if i < 0 || i >= len(r.values) {
		return nil, errors.ES(errors.OpTableAccess, errors.KClientArgs, "index %d out of range", i)
	}

	return r.values[i], nil
}

func (r *row) ValueByColumn(c Column) (value.Kusto, error) {
	return r.Value(c.Index())
}

func (r *row) ValueByName(name string) (value.Kusto, error) {
	col := r.columnByName(name)
	if col == nil {
		return nil, columnNotFoundError(name)
	}
	return r.Value(col.Index())
}

// ToStruct fetches the columns in a row into the fields of a struct. p must be a pointer to struct.
// The rules for mapping a row's columns into a struct's exported fields are:
//
//  1. If a field has a `kusto: "column_name"` tag, then decode column
//     'column_name' into the field. A special case is the `column_name: "-"`
//     tag, which instructs ToStruct to ignore the field during decoding.
//
//  2. Otherwise, if the name of a field matches the name of a column (ignoring case),
//     decode the column into the field.
//
// Slice and pointer fields will be set to nil if the source column is a null value, and a
// non-nil value if the column is not NULL. To decode NULL values of other types, use
// one of the kusto types (Int, Long, Dynamic, ...) as the type of the destination field.
// You can check the .Valid field of those types to see if the value was set.
func (r *row) ToStruct(p interface{}) error {
	// Check if p is a pointer to a struct
	if t := reflect.TypeOf(p); t == nil || t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return errors.ES(errors.OpTableAccess, errors.KClientArgs, "type %T is not a pointer to a struct", p)
	}
	if len(r.Columns()) != len(r.Values()) {
		return errors.ES(errors.OpTableAccess, errors.KClientArgs, "row does not have the correct number of values(%d) for the number of columns(%d)", len(r.Values()), len(r.Columns()))
	}

	return decodeToStruct(r.Columns(), r.Values(), p)
}

// String implements fmt.Stringer for a Row. This simply outputs a CSV version of the row.
func (r *row) String() string {
	var line []string
	for _, v := range r.Values() {
		line = append(line, v.String())
	}
	b := &strings.Builder{}
	w := csv.NewWriter(b)
	err := w.Write(line)
	if err != nil {
		return ""
	}
	w.Flush()
	return b.String()
}

func conversionError(from string, to string) error {
	return errors.ES(errors.OpTableAccess, errors.KOther, "cannot convert %s to %s", from, to)
}

func columnNotFoundError(name string) error {
	return errors.ES(errors.OpTableAccess, errors.KOther, "column %s not found", name)
}

// contains all types *bool, etc
type kustoTypeGeneric interface {
	*bool | *int32 | *int64 | *float64 | *decimal.Decimal | string | interface{} | *time.Time | *time.Duration
}

func byIndex[T kustoTypeGeneric](r *row, colType types.Column, i int, defaultValue T) (T, error) {
	val, err := r.Value(i)
	if err != nil {
		return defaultValue, err
	}
	if val.GetType() != colType {
		return defaultValue, conversionError(string(val.GetType()), string(colType))
	}

	return val.GetValue().(T), nil
}

func byName[T kustoTypeGeneric](r *row, colType types.Column, name string, defaultValue T) (T, error) {
	col := r.columnByName(name)
	if col == nil {
		return defaultValue, columnNotFoundError(name)
	}
	return byIndex(r, colType, col.Index(), defaultValue)
}

func (r *row) BoolByIndex(i int) (*bool, error) {
	return byIndex(r, types.Bool, i, (*bool)(nil))
}

func (r *row) IntByIndex(i int) (*int32, error) {
	return byIndex(r, types.Int, i, (*int32)(nil))
}

func (r *row) LongByIndex(i int) (*int64, error) {
	return byIndex(r, types.Long, i, (*int64)(nil))
}

func (r *row) RealByIndex(i int) (*float64, error) {
	return byIndex(r, types.Real, i, (*float64)(nil))
}

func (r *row) DecimalByIndex(i int) (*decimal.Decimal, error) {
	return byIndex(r, types.Decimal, i, (*decimal.Decimal)(nil))
}

func (r *row) StringByIndex(i int) (string, error) {
	return byIndex(r, types.String, i, "")
}

func (r *row) DynamicByIndex(i int) ([]byte, error) {
	return byIndex[[]byte](r, types.Dynamic, i, nil)
}

func (r *row) DateTimeByIndex(i int) (*time.Time, error) {
	return byIndex(r, types.DateTime, i, (*time.Time)(nil))
}

func (r *row) TimespanByIndex(i int) (*time.Duration, error) {
	return byIndex(r, types.Timespan, i, (*time.Duration)(nil))
}

func (r *row) GuidByIndex(i int) (*uuid.UUID, error) {
	return byIndex(r, types.GUID, i, (*uuid.UUID)(nil))
}

func (r *row) BoolByName(name string) (*bool, error) {
	return byName(r, types.Bool, name, (*bool)(nil))
}

func (r *row) IntByName(name string) (*int32, error) {
	return byName(r, types.Int, name, (*int32)(nil))
}

func (r *row) LongByName(name string) (*int64, error) {
	return byName(r, types.Long, name, (*int64)(nil))
}

func (r *row) RealByName(name string) (*float64, error) {
	return byName(r, types.Real, name, (*float64)(nil))
}

func (r *row) DecimalByName(name string) (*decimal.Decimal, error) {
	return byName(r, types.Decimal, name, (*decimal.Decimal)(nil))
}

func (r *row) StringByName(name string) (string, error) {
	return byName(r, types.String, name, "")
}

func (r *row) DynamicByName(name string) ([]byte, error) {
	return byName[[]byte](r, types.Dynamic, name, nil)
}

func (r *row) DateTimeByName(name string) (*time.Time, error) {
	return byName(r, types.DateTime, name, (*time.Time)(nil))
}

func (r *row) TimespanByName(name string) (*time.Duration, error) {
	return byName(r, types.Timespan, name, (*time.Duration)(nil))
}

func (r *row) GuidByName(name string) (*uuid.UUID, error) {
	return byName(r, types.GUID, name, (*uuid.UUID)(nil))
}

// ToStructs converts a table, a non-iterative dataset or a slice of rows into a slice of structs.
// If a dataset is provided, it should contain exactly one table.
func ToStructs[T any](data interface{}) ([]T, error) {
	var rows []Row
	var errs error

	switch v := data.(type) {
	case Table:
		rows = v.Rows()
	case IterativeTable:
		full, err := v.ToTable()
		if err != nil {
			return nil, err
		}
		rows = full.Rows()
	case []Row:
		rows = v
	case Row:
		rows = []Row{v}
	case Dataset:
		tables := v.Tables()
		if len(tables) == 0 {
			return nil, errors.ES(errors.OpUnknown, errors.KInternal, "dataset does not contain any tables")
		}
		if !tables[0].IsPrimaryResult() {
			return nil, errors.ES(errors.OpUnknown, errors.KInternal, "dataset contains no primary results")
		}
		rows = tables[0].Rows()
	default:
		return nil, errors.ES(errors.OpUnknown, errors.KInternal, "invalid data type - expected Dataset, Table, BaseTable or []Row")
	}

	if rows == nil || len(rows) == 0 {
		return nil, errs
	}

	out := make([]T, len(rows))
	for i, r := range rows {
		if err := r.ToStruct(&out[i]); err != nil {
			out = out[:i]
			if len(out) == 0 {
				out = nil
			}
			return out, err
		}
	}

	return out, errs
}

type StructResult[T any] struct {
	Out T
	Err error
}

func ToStructsIterative[T any](tb IterativeTable) chan StructResult[T] {
	out := make(chan StructResult[T])

	go func() {
		defer close(out)
		for rowResult := range tb.Rows() {
			if rowResult.Err() != nil {
				out <- StructResult[T]{Err: rowResult.Err()}
			} else {
				var s T
				if err := rowResult.Row().ToStruct(&s); err != nil {
					out <- StructResult[T]{Err: err}
				} else {
					out <- StructResult[T]{Out: s}
				}
			}
		}
	}()

	return out
}
