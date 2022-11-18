package kusto

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Azure/azure-kusto-go/kusto/data/table"
	"github.com/Azure/azure-kusto-go/kusto/data/types"
	"github.com/Azure/azure-kusto-go/kusto/data/value"

	"github.com/google/uuid"
)

// structToKustoValues takes a *struct and encodes to value.Values. At least one column must get set.
func structToKustoValues(cols table.Columns, p interface{}) (value.Values, error) {
	t := reflect.TypeOf(p).Elem()
	v := reflect.ValueOf(p).Elem()

	m := newColumnMap(cols)

	row, err := defaultRow(cols)
	if err != nil {
		return nil, err
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if tag := field.Tag.Get("kusto"); strings.TrimSpace(tag) != "" {
			colData, ok := m[tag]
			if !ok {
				continue
			}
			if err := fieldConvert(colData, v.Field(i), row); err != nil {
				return nil, err
			}
		} else {
			colData, ok := m[field.Name]
			if !ok {
				continue
			}

			if err := fieldConvert(colData, v.Field(i), row); err != nil {
				return nil, err
			}
		}
	}

	return row, nil
}

// fieldConvert will attempt to take the value held in v and convert it to the appropriate types.KustoValue
// that is described in colData in the correct location in row.
func fieldConvert(colData columnData, v reflect.Value, row value.Values) error {
	switch colData.column.Type {
	case types.Bool:
		c, err := convertBool(v)
		if err != nil {
			return err
		}
		row[colData.position] = c
	case types.DateTime:
		c, err := convertDateTime(v)
		if err != nil {
			return err
		}
		row[colData.position] = c
	case types.Dynamic:
		c, err := convertDynamic(v)
		if err != nil {
			return err
		}
		row[colData.position] = c
	case types.GUID:
		c, err := convertGUID(v)
		if err != nil {
			return err
		}
		row[colData.position] = c
	case types.Int:
		c, err := convertInt(v)
		if err != nil {
			return err
		}
		row[colData.position] = c
	case types.Long:
		c, err := convertLong(v)
		if err != nil {
			return err
		}
		row[colData.position] = c
	case types.Real:
		c, err := convertReal(v)
		if err != nil {
			return err
		}
		row[colData.position] = c
	case types.String:
		c, err := convertString(v)
		if err != nil {
			return err
		}
		row[colData.position] = c
	case types.Timespan:
		c, err := convertTimespan(v)
		if err != nil {
			return err
		}
		row[colData.position] = c
	case types.Decimal:
		c, err := convertDecimal(v)
		if err != nil {
			return err
		}
		row[colData.position] = c
	default:
		return fmt.Errorf("column[%d] was for a column type that we don't understand(%s)", colData.position, colData.column.Type)
	}
	return nil
}

// defaultRow creates a complete row of KustoValues set to types outlined with cols. Useful for having
// default values for fields that are not set.
func defaultRow(cols table.Columns) (value.Values, error) {
	var row = make(value.Values, len(cols))
	for i, col := range cols {
		switch col.Type {
		case types.Bool:
			row[i] = value.Bool{}
		case types.DateTime:
			row[i] = value.DateTime{}
		case types.Dynamic:
			row[i] = value.Dynamic{}
		case types.GUID:
			row[i] = value.GUID{}
		case types.Int:
			row[i] = value.Int{}
		case types.Long:
			row[i] = value.Long{}
		case types.Real:
			row[i] = value.Real{}
		case types.String:
			row[i] = value.String{}
		case types.Timespan:
			row[i] = value.Timespan{}
		case types.Decimal:
			row[i] = value.Decimal{}
		default:
			return nil, fmt.Errorf("column[%d] was for a column type that we don't understand(%s)", i, col.Type)
		}
	}
	return row, nil
}

func colToValueCheck(cols table.Columns, values value.Values) error {
	if len(cols) != len(values) {
		return fmt.Errorf("the length of columns(%d) is not the same as the length of the row(%d)", len(cols), len(values))
	}

	for i, v := range values {
		col := cols[i]

		switch col.Type {
		case types.Bool:
			if reflect.TypeOf(v) != reflect.TypeOf(value.Bool{}) {
				return fmt.Errorf("value[%d] was expected to be of a value.Bool, was %T", i, v)
			}
		case types.DateTime:
			if reflect.TypeOf(v) != reflect.TypeOf(value.DateTime{}) {
				return fmt.Errorf("value[%d] was expected to be of a value.DateTime, was %T", i, v)
			}
		case types.Dynamic:
			if reflect.TypeOf(v) != reflect.TypeOf(value.Dynamic{}) {
				return fmt.Errorf("value[%d] was expected to be of a value.Dynamic, was %T", i, v)
			}
		case types.GUID:
			if reflect.TypeOf(v) != reflect.TypeOf(value.GUID{}) {
				return fmt.Errorf("value[%d] was expected to be of a value.GUID, was %T", i, v)
			}
		case types.Int:
			if reflect.TypeOf(v) != reflect.TypeOf(value.Int{}) {
				return fmt.Errorf("value[%d] was expected to be of a value.Int, was %T", i, v)
			}
		case types.Long:
			if reflect.TypeOf(v) != reflect.TypeOf(value.Long{}) {
				return fmt.Errorf("value[%d] was expected to be of a value.Long, was %T", i, v)
			}
		case types.Real:
			if reflect.TypeOf(v) != reflect.TypeOf(value.Real{}) {
				return fmt.Errorf("value[%d] was expected to be of a value.Real, was %T", i, v)
			}
		case types.String:
			if reflect.TypeOf(v) != reflect.TypeOf(value.String{}) {
				return fmt.Errorf("value[%d] was expected to be of a value.String, was %T", i, v)
			}
		case types.Timespan:
			if reflect.TypeOf(v) != reflect.TypeOf(value.Timespan{}) {
				return fmt.Errorf("value[%d] was expected to be of a value.Timespan, was %T", i, v)
			}
		case types.Decimal:
			if reflect.TypeOf(v) != reflect.TypeOf(value.Decimal{}) {
				return fmt.Errorf("value[%d] was expected to be of a value.Decimal, was %T", i, v)
			}
		default:
			return fmt.Errorf("value[%d] was for a column type that MockRow doesn't understand(%s)", i, col.Type)
		}
	}
	return nil
}

func convertBool(v reflect.Value) (value.Bool, error) {
	t := v.Type()

	// If it is a pointer, dereference it.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	// Was a value.Bool{}, so return it.
	if t == reflect.TypeOf(value.Bool{}) {
		return v.Interface().(value.Bool), nil
	}

	// Was a Bool, so return its value.
	if t == reflect.TypeOf(true) {
		return value.Bool{Value: v.Interface().(bool), Valid: true}, nil
	}

	return value.Bool{}, fmt.Errorf("value was expected to be either a value.Bool, *bool or bool, was %T", v.Interface())
}

func convertDateTime(v reflect.Value) (value.DateTime, error) {
	t := v.Type()

	// If it is a pointer, dereference it.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	// Was a value.DateTime{}, so return it.
	if t == reflect.TypeOf(value.DateTime{}) {
		return v.Interface().(value.DateTime), nil
	}

	// Was a time.Time, so return its value.
	if t == reflect.TypeOf(time.Time{}) {
		return value.DateTime{Value: v.Interface().(time.Time), Valid: true}, nil
	}

	return value.DateTime{}, fmt.Errorf("value was expected to be either a value.DateTime, *time.Time or time.Time, was %T", v.Interface())
}

func convertTimespan(v reflect.Value) (value.Timespan, error) {
	t := v.Type()

	// If it is a pointer, dereference it.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	// Was a value.Timespan{}, so return it.
	if t == reflect.TypeOf(value.Timespan{}) {
		return v.Interface().(value.Timespan), nil
	}

	// Was a time.Duration, so return its value.
	if t == reflect.TypeOf(time.Second) {
		return value.Timespan{Value: v.Interface().(time.Duration), Valid: true}, nil
	}

	return value.Timespan{}, fmt.Errorf("value was expected to be either a value.Timespan, *time.Duration or time.Duration, was %T", v.Interface())
}

func convertDynamic(v reflect.Value) (value.Dynamic, error) {
	t := v.Type()

	// If it is a pointer, dereference it.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	// Was a value.Dynamic{}, so return it.
	if t == reflect.TypeOf(value.Dynamic{}) {
		return v.Interface().(value.Dynamic), nil
	}

	// Was a string, so return it as []byte.
	if t == reflect.TypeOf("") {
		return value.Dynamic{Value: []byte(v.Interface().(string)), Valid: true}, nil
	}

	// Was a []byte, so return it.
	if t == reflect.TypeOf([]byte{}) {
		return value.Dynamic{Value: v.Interface().([]byte), Valid: true}, nil
	}

	// Anything else, try to marshal it.
	b, err := json.Marshal(v.Interface())
	if err != nil {
		return value.Dynamic{}, fmt.Errorf("the type *%T used in a value.Dynamic could not be JSON encoded: %s", v.Interface(), err)
	}
	return value.Dynamic{Value: b, Valid: true}, nil
}

func convertGUID(v reflect.Value) (value.GUID, error) {
	t := v.Type()

	// If it is a pointer, dereference it.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	// Was a value.GUID{}, so return it.
	if t == reflect.TypeOf(value.GUID{}) {
		return v.Interface().(value.GUID), nil
	}

	// Was a uuid.UUID, so return its value.
	if t == reflect.TypeOf(uuid.UUID{}) {
		return value.GUID{Value: v.Interface().(uuid.UUID), Valid: true}, nil
	}

	return value.GUID{}, fmt.Errorf("value was expected to be either a value.BUID, *uuid.UUID or uuid.UUID, was %T", v.Interface())
}

func convertInt(v reflect.Value) (value.Int, error) {
	t := v.Type()

	// If it is a pointer, dereference it.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	// Was a value.Int{}, so return it.
	if t == reflect.TypeOf(value.Int{}) {
		return v.Interface().(value.Int), nil
	}

	// Was a int32, so return its value.
	if t == reflect.TypeOf(int32(1)) {
		return value.Int{Value: v.Interface().(int32), Valid: true}, nil
	}

	return value.Int{}, fmt.Errorf("value was expected to be either a value.Int, *int32 or int32, was %T", v.Interface())
}

func convertLong(v reflect.Value) (value.Long, error) {
	t := v.Type()

	// If it is a pointer, dereference it.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	// Was a value.Long{}, so return it.
	if t == reflect.TypeOf(value.Long{}) {
		return v.Interface().(value.Long), nil
	}

	// Was a int64, so return its value.
	if t == reflect.TypeOf(int64(1)) {
		return value.Long{Value: v.Interface().(int64), Valid: true}, nil
	}

	return value.Long{}, fmt.Errorf("value was expected to be either a value.Long, *int64 or int64, was %T", v.Interface())
}

func convertReal(v reflect.Value) (value.Real, error) {
	t := v.Type()

	// If it is a pointer, dereference it.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	// Was a value.Real{}, so return it.
	if t == reflect.TypeOf(value.Real{}) {
		return v.Interface().(value.Real), nil
	}

	// Was a float64, so return its value.
	if t == reflect.TypeOf(float64(1.0)) {
		return value.Real{Value: v.Interface().(float64), Valid: true}, nil
	}

	return value.Real{}, fmt.Errorf("value was expected to be either a value.Real, *float64 or float64, was %T", v.Interface())
}

func convertString(v reflect.Value) (value.String, error) {
	t := v.Type()

	// If it is a pointer, dereference it.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	// Was a types.String{}, so return it.
	if t == reflect.TypeOf(value.String{}) {
		return v.Interface().(value.String), nil
	}

	// Was a string, so return its value.
	if t == reflect.TypeOf("") {
		return value.String{Value: v.Interface().(string), Valid: true}, nil
	}

	return value.String{}, fmt.Errorf("value was expected to be either a types.String, *string or string, was %T", v.Interface())
}

func convertDecimal(v reflect.Value) (value.Decimal, error) {
	t := v.Type()

	// If it is a pointer, dereference it.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	// Was a types.Decimal{}, so return it.
	if t == reflect.TypeOf(value.Decimal{}) {
		return v.Interface().(value.Decimal), nil
	}

	// Was a string, so return its value.
	if t == reflect.TypeOf("") {
		return value.Decimal{Value: v.Interface().(string), Valid: true}, nil
	}

	return value.Decimal{}, fmt.Errorf("value was expected to be either a types.Decimal, *string or string, was %T", v.Interface())
}
