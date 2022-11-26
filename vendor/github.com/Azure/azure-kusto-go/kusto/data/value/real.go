package value

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

// Real represents a Kusto real type.  Real implements Kusto.
type Real struct {
	// Value holds the value of the type.
	Value float64
	// Valid indicates if this value was set.
	Valid bool
}

func (Real) isKustoVal() {}

// String implements fmt.Stringer.
func (r Real) String() string {
	if !r.Valid {
		return ""
	}
	return strconv.FormatFloat(r.Value, 'e', -1, 64)
}

// Unmarshal unmarshals i into Real. i must be a json.Number(that is a float64), float64 or nil.
func (r *Real) Unmarshal(i interface{}) error {
	if i == nil {
		r.Value = 0.0
		r.Valid = false
		return nil
	}

	var myFloat float64

	switch v := i.(type) {
	case json.Number:
		var err error
		myFloat, err = v.Float64()
		if err != nil {
			return fmt.Errorf("Column with type 'real' had value json.Number that had error on .Float64(): %s", err)
		}
	case float64:
		myFloat = v
	default:
		return fmt.Errorf("Column with type 'real' had value that was not a json.Number or float64, was %T", i)
	}

	r.Value = myFloat
	r.Valid = true
	return nil
}

// Convert Real into reflect value.
func (r Real) Convert(v reflect.Value) error {
	t := v.Type()
	switch {
	case t.Kind() == reflect.Float64:
		if r.Valid {
			v.Set(reflect.ValueOf(r.Value))
		}
		return nil
	case t.ConvertibleTo(reflect.TypeOf(new(float64))):
		if r.Valid {
			i := &r.Value
			v.Set(reflect.ValueOf(i))
		}
		return nil
	case t.ConvertibleTo(reflect.TypeOf(Real{})):
		v.Set(reflect.ValueOf(r))
		return nil
	case t.ConvertibleTo(reflect.TypeOf(&Real{})):
		v.Set(reflect.ValueOf(&r))
		return nil
	}
	return fmt.Errorf("Column was type Kusto.Real, receiver had base Kind %s ", t.Kind())
}
