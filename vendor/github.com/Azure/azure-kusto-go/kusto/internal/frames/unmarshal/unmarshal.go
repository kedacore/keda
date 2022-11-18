// Package unmarshal provides decoding of Kusto row data in a frame into []value.Values representing those rows.
package unmarshal

import (
	"fmt"
	"sync"

	"github.com/Azure/azure-kusto-go/kusto/data/errors"
	"github.com/Azure/azure-kusto-go/kusto/data/table"
	"github.com/Azure/azure-kusto-go/kusto/data/types"
	"github.com/Azure/azure-kusto-go/kusto/data/value"
)

var rowsPool = sync.Pool{
	New: func() interface{} {
		return make([]interface{}, 0)
	},
}

// GetRows is used to pull values that can be used to decode Rows into out of a pool.
func GetRows() []interface{} {
	return rowsPool.Get().([]interface{})
}

// PutRows is used to put values that were used to decode Rows into a pool.
func PutRows(rows []interface{}) {
	rows = rows[0:0]
	//nolint:staticcheck // ignore SA6002, we can't break the API
	rowsPool.Put(rows)
}

// Rows unmarshals a slice of a slice that represents a set of rows and translates them into a set of []value.Values.
func Rows(columns table.Columns, interRows []interface{}, op errors.Op) ([]value.Values, []errors.Error, error) {
	rows := make([]value.Values, 0, len(interRows))
	var errorRows []errors.Error

	for _, rawRow := range interRows {
		interRow, ok := rawRow.([]interface{})
		if !ok && rawRow != nil {
			errorRow, ok := rawRow.(map[string]interface{})
			if !ok {
				errorRows = append(errorRows, *errors.ES(op, errors.KInternal, "Unexpected row error: %v", rawRow))
			}

			errorRows = append(errorRows, *errors.OneToErr(errorRow, op))

			continue
		}

		row := make(value.Values, len(columns))
		for i, col := range columns {
			switch col.Type {
			case types.Bool:
				v := value.Bool{}
				if err := v.Unmarshal(interRow[i]); err != nil {
					return nil, nil, fmt.Errorf("unable to unmarshal column %s into a Bool value: %s", col.Name, err)
				}
				row[i] = v
			case types.DateTime:
				v := value.DateTime{}
				if err := v.Unmarshal(interRow[i]); err != nil {
					return nil, nil, fmt.Errorf("unable to unmarshal column %s into a DateTime value: %s", col.Name, err)
				}
				row[i] = v
			case types.Decimal:
				v := value.Decimal{}
				if err := v.Unmarshal(interRow[i]); err != nil {
					return nil, nil, fmt.Errorf("unable to unmarshal column %s into a Decimal value: %s", col.Name, err)
				}
				row[i] = v
			case types.Dynamic:
				v := value.Dynamic{}
				if err := v.Unmarshal(interRow[i]); err != nil {
					return nil, nil, fmt.Errorf("unable to unmarshal column %s into a Dynamic value: %s", col.Name, err)
				}
				row[i] = v
			case types.GUID:
				v := value.GUID{}
				if err := v.Unmarshal(interRow[i]); err != nil {
					return nil, nil, fmt.Errorf("unable to unmarshal column %s into a GUID value: %s", col.Name, err)
				}
				row[i] = v
			case types.Int:
				v := value.Int{}
				if err := v.Unmarshal(interRow[i]); err != nil {
					return nil, nil, fmt.Errorf("unable to unmarshal column %s into a Int value: %s", col.Name, err)
				}
				row[i] = v
			case types.Long:
				v := value.Long{}
				if err := v.Unmarshal(interRow[i]); err != nil {
					return nil, nil, fmt.Errorf("unable to unmarshal column %s into a Long value: %s", col.Name, err)
				}
				row[i] = v
			case types.Real:
				v := value.Real{}
				if err := v.Unmarshal(interRow[i]); err != nil {
					return nil, nil, fmt.Errorf("unable to unmarshal column %s into a Real value: %s", col.Name, err)
				}
				row[i] = v
			case types.String:
				v := value.String{}
				if err := v.Unmarshal(interRow[i]); err != nil {
					return nil, nil, fmt.Errorf("unable to unmarshal column %s into a String value: %s", col.Name, err)
				}
				row[i] = v
			case types.Timespan:
				v := value.Timespan{}
				if err := v.Unmarshal(interRow[i]); err != nil {
					return nil, nil, fmt.Errorf("unable to unmarshal column %s into a Timespan value: %s", col.Name, err)
				}
				row[i] = v
			default:
				return nil, nil, fmt.Errorf("DataTable had column of type %s, which was unknown", col.Type)
			}
		}
		rows = append(rows, row)
	}
	return rows, errorRows, nil
}
