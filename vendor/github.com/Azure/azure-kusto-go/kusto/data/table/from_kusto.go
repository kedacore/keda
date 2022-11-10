package table

// value.go provides methods for converting a row to a *struct and for converting KustoValue into Go types
// or in the reverse.

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Azure/azure-kusto-go/kusto/data/value"
)

// decodeToStruct takes a list of columns and a row to decode into "p" which will be a pointer
// to a struct (enforce in the decoder).
func decodeToStruct(cols Columns, row value.Values, p interface{}) error {
	t := reflect.TypeOf(p)
	v := reflect.ValueOf(p)
	fields := newFields(cols, t)

	for i, col := range cols {
		if err := fields.convert(col, row[i], t, v); err != nil {
			return err
		}
	}
	return nil
}

// fields represents the fields inside a struct.
type fields struct {
	colNameToFieldName map[string]string
}

// newFields takes in the Columns from our row and the reflect.Type of our *struct.
func newFields(cols Columns, ptr reflect.Type) fields {
	nFields := fields{colNameToFieldName: map[string]string{}}
	for i := 0; i < ptr.Elem().NumField(); i++ {
		field := ptr.Elem().Field(i)
		if tag := field.Tag.Get("kusto"); strings.TrimSpace(tag) != "" {
			nFields.colNameToFieldName[tag] = field.Name
		} else {
			nFields.colNameToFieldName[field.Name] = field.Name
		}
	}

	return nFields
}

// convert converts a KustoValue that is for Column col into "v" reflect.Value with reflect.Type "t".
func (f fields) convert(col Column, k value.Kusto, t reflect.Type, v reflect.Value) error {
	fieldName, ok := f.colNameToFieldName[col.Name]
	if !ok {
		return nil
	}

	if fieldName == "-" {
		return nil
	}

	err := k.Convert(v.Elem().FieldByName(fieldName))
	if err != nil {
		return fmt.Errorf("column %s could not store in struct.%s: %s", col.Name, fieldName, err.Error())
	}

	return nil
}
