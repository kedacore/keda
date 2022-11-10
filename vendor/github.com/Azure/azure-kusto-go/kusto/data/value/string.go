package value

import (
	"fmt"
	"reflect"
)

// String represents a Kusto string type.  String implements Kusto.
type String struct {
	// Value holds the value of the type.
	Value string
	// Valid indicates if this value was set.
	Valid bool
}

func (String) isKustoVal() {}

// String implements fmt.Stringer.
func (s String) String() string {
	if !s.Valid {
		return ""
	}
	return s.Value
}

// Unmarshal unmarshals i into String. i must be a string or nil.
func (s *String) Unmarshal(i interface{}) error {
	if i == nil {
		s.Value = ""
		s.Valid = false
		return nil
	}

	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("Column with type 'string' had type %T", i)
	}

	s.Value = v
	s.Valid = true
	return nil
}

// Convert String into reflect value.
func (s String) Convert(v reflect.Value) error {
	t := v.Type()
	switch {
	case t.Kind() == reflect.String:
		if s.Valid {
			v.Set(reflect.ValueOf(s.Value))
		}
		return nil
	case t.ConvertibleTo(reflect.TypeOf(new(string))):
		if s.Valid {
			i := &s.Value
			v.Set(reflect.ValueOf(i))
		}
		return nil
	case t.ConvertibleTo(reflect.TypeOf(String{})):
		v.Set(reflect.ValueOf(s))
		return nil
	case t.ConvertibleTo(reflect.TypeOf(&String{})):
		v.Set(reflect.ValueOf(&s))
		return nil
	}
	return fmt.Errorf("Column was type Kusto.String, receiver had base Kind %s ", t.Kind())
}
