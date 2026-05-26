package value

import (
	"fmt"
	"github.com/Azure/azure-kusto-go/azkustodata/types"
	"reflect"
)

// String represents a Kusto string type.  String implements Kusto.
type String struct {
	// Value holds the value of the type.
	Value string
}

func NewString(v string) *String {
	return &String{Value: v}
}

// String implements fmt.Stringer.
func (s *String) String() string {
	return s.Value
}

// Unmarshal unmarshals i into String. i must be a string or nil.
func (s *String) Unmarshal(i interface{}) error {
	if i == nil {
		s.Value = ""
		return nil
	}

	v, ok := i.(string)
	if !ok {
		return convertError(s, i)
	}

	s.Value = v
	return nil
}

// Convert String into reflect value.
func (s *String) Convert(v reflect.Value) error {
	t := v.Type()
	switch {
	case t.Kind() == reflect.String:
		v.SetString(s.Value)
		return nil
	case t.ConvertibleTo(reflect.TypeOf(new(string))):
		i := &s.Value
		v.Set(reflect.ValueOf(i))
		return nil
	case t.ConvertibleTo(reflect.TypeOf(String{})):
		v.Set(reflect.ValueOf(*s))
		return nil
	case t.ConvertibleTo(reflect.TypeOf(&String{})):
		v.Set(reflect.ValueOf(s))
		return nil
	}
	return fmt.Errorf("Column was type Kusto.String, receiver had base Kind %s ", t.Kind())
}

// GetValue returns the value of the type.
func (s *String) GetValue() interface{} {
	return s.Value
}

// GetType returns the type of the value.
func (s *String) GetType() types.Column {
	return types.String
}
