package value

import (
	"fmt"
	"reflect"
)

// Bool represents a Kusto boolean type. Bool implements Kusto.
type Bool struct {
	// Value holds the value of the type.
	Value bool
	// Valid indicates if this value was set.
	Valid bool
}

func (Bool) isKustoVal() {}

// String implements fmt.Stringer.
func (bo Bool) String() string {
	if !bo.Valid {
		return ""
	}
	if bo.Value {
		return "true"
	}
	return "false"
}

// Unmarshal unmarshals i into Bool. i must be a bool or nil.
func (bo *Bool) Unmarshal(i interface{}) error {
	if i == nil {
		bo.Value = false
		bo.Valid = false
		return nil
	}
	v, ok := i.(bool)
	if !ok {
		return fmt.Errorf("Column with type 'bool' had value that was %T", i)
	}
	bo.Value = v
	bo.Valid = true
	return nil
}

// Convert Bool into reflect value.
func (bo Bool) Convert(v reflect.Value) error {
	t := v.Type()
	switch {
	case t.Kind() == reflect.Bool:
		if bo.Valid {
			v.SetBool(bo.Value)
		}
		return nil
	case t.ConvertibleTo(reflect.TypeOf(new(bool))):
		if bo.Valid {
			b := new(bool)
			if bo.Value {
				*b = true
			}
			v.Set(reflect.ValueOf(b))
		}
		return nil
	case t.ConvertibleTo(reflect.TypeOf(Bool{})):
		v.Set(reflect.ValueOf(bo))
		return nil
	case t.ConvertibleTo(reflect.TypeOf(&Bool{})):
		v.Set(reflect.ValueOf(&bo))
		return nil
	}
	return fmt.Errorf("Column was type Kusto.Bool, receiver had base Kind %s ", t.Kind())
}
