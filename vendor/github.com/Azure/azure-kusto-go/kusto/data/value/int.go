package value

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
)

// Int represents a Kusto int type. Values int type's are int32 values.  Int implements Kusto.
type Int struct {
	// Value holds the value of the type.
	Value int32
	// Valid indicates if this value was set.
	Valid bool
}

func (Int) isKustoVal() {}

// String implements fmt.Stringer.
func (in Int) String() string {
	if !in.Valid {
		return ""
	}
	return strconv.Itoa(int(in.Value))
}

// Unmarshal unmarshals i into Int. i must be an int32 or nil.
func (in *Int) Unmarshal(i interface{}) error {
	if i == nil {
		in.Value = 0
		in.Valid = false
		return nil
	}

	var myInt int64

	switch v := i.(type) {
	case json.Number:
		var err error
		myInt, err = v.Int64()
		if err != nil {
			return fmt.Errorf("Column with type 'int' had value json.Number that had error on .Int64(): %s", err)
		}
	case float64:
		if v != math.Trunc(v) {
			return fmt.Errorf("Column with type 'int' had value float64(%v) that did not represent a whole number", v)
		}
		myInt = int64(v)
	case int:
		myInt = int64(v)
	default:
		return fmt.Errorf("Column with type 'int' had value that was not a json.Number or int, was %T", i)
	}

	if myInt > math.MaxInt32 {
		return fmt.Errorf("Column with type 'int' had value that was greater than an int32 can hold, was %d", myInt)
	}
	in.Value = int32(myInt)
	in.Valid = true
	return nil
}

// Convert Int into reflect value.
func (in Int) Convert(v reflect.Value) error {
	t := v.Type()
	switch {
	case t.Kind() == reflect.Int32:
		if in.Valid {
			v.Set(reflect.ValueOf(in.Value))
		}
		return nil
	case t.ConvertibleTo(reflect.TypeOf(new(int32))):
		if in.Valid {
			i := &in.Value
			v.Set(reflect.ValueOf(i))
		}
		return nil
	case t.ConvertibleTo(reflect.TypeOf(Int{})):
		v.Set(reflect.ValueOf(in))
		return nil
	case t.ConvertibleTo(reflect.TypeOf(&Int{})):
		v.Set(reflect.ValueOf(&in))
		return nil
	}
	return fmt.Errorf("Column was type Kusto.Int, receiver had base Kind %s ", t.Kind())
}
