// Package v1 holds framing information for the v1 REST API.
package v1

import (
	"strings"

	"github.com/Azure/azure-kusto-go/kusto/data/errors"
	"github.com/Azure/azure-kusto-go/kusto/data/table"
	"github.com/Azure/azure-kusto-go/kusto/data/types"
	"github.com/Azure/azure-kusto-go/kusto/data/value"
	"github.com/Azure/azure-kusto-go/kusto/internal/frames"
)

// DataTypes is a V1 version of table.Columns that have to be dealt with and converted.
type DataTypes []DataType

// ToColumns converts the DataTypes into a table.Columns so our upstream can have one data representation.
func (dt DataTypes) ToColumns() (table.Columns, error) {
	cols := make(table.Columns, 0, len(dt))

	for _, dataType := range dt {
		col, err := dataType.toColumn()
		if err != nil {
			return cols, err
		}
		cols = append(cols, col)
	}
	return cols, nil
}

// DataType is the Column representation in V1 series frames.
type DataType struct {
	ColumnName string
	ColumnType string
	DataType   string
}

func (dt DataType) rawType() string {
	if dt.ColumnType != "" {
		return dt.ColumnType
	}
	return dt.DataType
}

func (dt DataType) toColumn() (table.Column, error) {
	col := table.Column{Name: dt.ColumnName}

	if dt.rawType() == "" {
		return col, errors.ES(errors.OpMgmt, errors.KInternal, "DataTable.Columns(v1) did not have ColumnType or DataType provided")
	}

	var ok bool
	col.Type, ok = translate[strings.ToLower(dt.rawType())]
	if !ok {
		return col, errors.ES(errors.OpMgmt, errors.KInternal, "DataTable.Columns(v1) had string entry with either .ColumnType/.DataType set to %s, which is not supported", dt.rawType())
	}
	return col, nil
}

// DataTable represents a Kusto REST v1 DataTable that is returned in a DataSet.
type DataTable struct {
	TableName frames.TableKind
	DataTypes DataTypes `json:"Columns"`
	Rows      []interface{}
	KustoRows []value.Values
	RowErrors []errors.Error
	Op        errors.Op
}

// IsFrame implements frames.Frame.
func (DataTable) IsFrame() {}

var translate = map[string]types.Column{
	"bool":                            types.Bool,
	"boolean":                         types.Bool,
	"system.boolean":                  types.Bool,
	"datetime":                        types.DateTime,
	"date":                            types.DateTime,
	"system.datetime":                 types.DateTime,
	"dynamic":                         types.Dynamic,
	"object":                          types.Dynamic,
	"system.object":                   types.Dynamic,
	"guid":                            types.GUID,
	"uuid":                            types.GUID,
	"uniqueid":                        types.GUID,
	"system.guid":                     types.GUID,
	"int":                             types.Int,
	"int32":                           types.Int,
	"system.int32":                    types.Int,
	"long":                            types.Long,
	"int64":                           types.Long,
	"system.int64":                    types.Long,
	"real":                            types.Real,
	"double":                          types.Real,
	"system.double":                   types.Real,
	"string":                          types.String,
	"system.string":                   types.String,
	"timespan":                        types.Timespan,
	"time":                            types.Timespan,
	"system.timeSpan":                 types.Timespan,
	"decimal":                         types.Decimal,
	"system.data.sqltypes.sqldecimal": types.Decimal,
	"sqldecimal":                      types.Decimal,
}
