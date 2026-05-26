/*
Package types holds Kusto type information that is used to describe what type would be held in a cell based
on the column's type setting.

The type information here is used to define what types of paramaters will substituted for in Stmt objects and to
do discovery on table columns, if trying to do some type of dynamic discovery.

# Column

Column represents a Column type. A user should never try to implement these. Instead they should use
the constants defined in this package, such as types.Bool, types.DateTime.
*/
package types

// Column represents a type of column defined for Kusto.
// For more information, please see: https://docs.microsoft.com/en-us/azure/kusto/query/scalar-data-types/
type Column string

// NormalizeColumn checks if the column is a valid column type and returns the normalized column type.
// If the column is not valid, it returns an empty string.
func NormalizeColumn(c string) Column {
	if mapped, ok := mappedNames[c]; ok {
		return mapped
	}

	return ""
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

var mappedNames = map[string]Column{
	string(Bool): Bool,
	"boolean":    Bool,

	string(DateTime): DateTime,
	"date":           DateTime,

	string(Dynamic): Dynamic,

	string(GUID): GUID,
	"uuid":       GUID,
	"uniqueid":   GUID,

	string(Int): Int,
	"int32":     Int,

	string(Long): Long,
	"int64":      Long,

	string(Real): Real,
	"double":     Real,

	string(String):   String,
	string(Timespan): Timespan,
	"time":           Timespan,

	string(Decimal): Decimal,
}
