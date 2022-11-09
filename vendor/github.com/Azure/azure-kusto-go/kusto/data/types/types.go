/*
Package types holds Kusto type information that is used to describe what type would be held in a cell based
on the column's type setting.

The type information here is used to define what types of paramaters will substituted for in Stmt objects and to
do discovery on table columns, if trying to do some type of dynamic discovery.


Column

Column represents a Column type. A user should never try to implemenet these. Instead they should use
the constants defined in this package, such as types.Bool, types.DateTime, ...  Any other value that these
will not pass the Column.Valid() method and will be rejected by the API.
*/
package types

// Column represents a type of column defined for Kusto.
// For more information, please see: https://docs.microsoft.com/en-us/azure/kusto/query/scalar-data-types/
type Column string

// Valid returns true if the Column is a valid value.
func (c Column) Valid() bool {
	return valid[c]
}

// These constants represent the value type stored in a Column.
const (
	// Bool indicates that a Column stores a Kusto boolean value.
	Bool Column = "bool"
	// DateTime indicates that a Column stores a Kusto datetime value.
	DateTime Column = "datetime"
	// Dynamic indicates that a Column stores a Kusto dynamic value.
	Dynamic Column = "dynamic"
	// GUID indicates that a Column stores a Kusto guid value.
	GUID Column = "guid"
	// Int indicates that a Column stores a Kusto int value.
	Int Column = "int"
	// Long indicates that a Column stores a Kusto long value.
	Long Column = "long"
	// Real indicates that a Column stores a Kusto real value.
	Real Column = "real"
	// String indicates that a Column stores a Kusto string value.
	String Column = "string"
	// Timespan indicates that a Column stores a Kusto timespan value.
	Timespan Column = "timespan"
	// Decimal indicates that a Column stores a Kusto decimal value.
	Decimal Column = "decimal" // We have NOT written a conversion
)

var valid = map[Column]bool{
	Bool:     true,
	DateTime: true,
	Dynamic:  true,
	GUID:     true,
	Int:      true,
	Long:     true,
	Real:     true,
	String:   true,
	Timespan: true,
	Decimal:  true,
}
