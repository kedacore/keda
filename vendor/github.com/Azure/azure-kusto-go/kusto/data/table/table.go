// Package table contains types that represent the makeup of a Kusto table.
package table

import (
	"encoding/csv"
	"fmt"
	"reflect"
	"strings"

	"github.com/Azure/azure-kusto-go/kusto/data/errors"
	"github.com/Azure/azure-kusto-go/kusto/data/types"
	"github.com/Azure/azure-kusto-go/kusto/data/value"
)

// Column describes a column descriptor.
type Column struct {
	// Name is the name of the column.
	Name string `json:"ColumnName"`
	// Type is the type of value stored in this column. These are described
	// via constants starting with CT<type>.
	Type types.Column `json:"ColumnType"`
}

// Columns is a set of columns.
type Columns []Column

func (c Columns) Validate() error {
	if len(c) == 0 {
		return fmt.Errorf("Columns is zero length")
	}

	names := make(map[string]bool, len(c))

	for i, col := range c {
		if col.Name == "" {
			return fmt.Errorf("column[%d].Name is empty string", i)
		}
		if names[col.Name] {
			return fmt.Errorf("column[%d].Name(%s) is already defined", i, col.Name)
		}
		names[col.Name] = true

		if !col.Type.Valid() {
			return fmt.Errorf("column[%d] if of type %q, which is not valid", i, col.Type)
		}
	}
	return nil
}

// Row represents a row of Kusto data. Methods are not thread-safe.
type Row struct {
	// ColumnType contains all the column type information for the row.
	ColumnTypes Columns
	// Values is the list of values that make up the row.
	Values value.Values
	// Op is the operation that resulted in the row. This is for internal use.
	Op errors.Op
	// Replace indicates whether the existing result set should be cleared and replaced with this row.
	Replace bool

	columnNames []string
}

// ColumnNames returns a list of all column names.
func (r *Row) ColumnNames() []string {
	if r.columnNames == nil {
		for _, col := range r.ColumnTypes {
			r.columnNames = append(r.columnNames, col.Name)
		}
	}
	return r.columnNames
}

// Size returns the number of columns contained in Row.
func (r *Row) Size() int {
	return len(r.ColumnTypes)
}

// Columns fetches all column names in the row at once.
// The name of the kth column will be decoded into the kth argument to Columns.
// The number of arguments must be equal to the number of columns.
// Pass nil to specify that a column should be ignored.
// ptrs may be either the *string or *types.Column type. An error in decoding may leave
// some ptrs set and others not.
func (r *Row) Columns(ptrs ...interface{}) error {
	if len(ptrs) != len(r.ColumnTypes) {
		return errors.ES(r.Op, errors.KClientArgs, ".Columns() requires %d arguments for this row, had %d", len(r.ColumnTypes), len(ptrs))
	}

	for i, col := range r.ColumnTypes {
		if ptrs[i] == nil {
			continue
		}
		switch v := ptrs[i].(type) {
		case *string:
			*v = col.Name
		case *Column:
			v.Name = col.Name
			v.Type = col.Type
		default:
			return errors.ES(r.Op, errors.KClientArgs, ".Columns() received argument at position %d that was not a *string, *types.Columns: was %T", i, ptrs[i])
		}
	}

	return nil
}

// ExtractValues fetches all values in the row at once.
// The value of the kth column will be decoded into the kth argument to ExtractValues.
// The number of arguments must be equal to the number of columns.
// Pass nil to specify that a column should be ignored.
// ptrs should be compatible with column types. An error in decoding may leave
// some ptrs set and others not.
func (r *Row) ExtractValues(ptrs ...interface{}) error {
	if len(ptrs) != len(r.ColumnTypes) {
		return errors.ES(r.Op, errors.KClientArgs, ".Columns() requires %d arguments for this row, had %d", len(r.ColumnTypes), len(ptrs))
	}

	for i, val := range r.Values {
		if ptrs[i] == nil {
			continue
		}
		if err := val.Convert(reflect.ValueOf(ptrs[i]).Elem()); err != nil {
			return err
		}
	}

	return nil
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
func (r *Row) ToStruct(p interface{}) error {
	// Check if p is a pointer to a struct
	if t := reflect.TypeOf(p); t == nil || t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return errors.ES(r.Op, errors.KClientArgs, "type %T is not a pointer to a struct", p)
	}
	if len(r.ColumnTypes) != len(r.Values) {
		return errors.ES(r.Op, errors.KClientArgs, "row does not have the correct number of values(%d) for the number of columns(%d)", len(r.Values), len(r.ColumnTypes))
	}

	return decodeToStruct(r.ColumnTypes, r.Values, p)
}

// String implements fmt.Stringer for a Row. This simply outputs a CSV version of the row.
func (r *Row) String() string {
	line := []string{}
	for _, v := range r.Values {
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

// Rows is a set of rows.
type Rows []*Row
