package value

import (
	"fmt"
	"reflect"

	"github.com/google/uuid"
)

// GUID represents a Kusto GUID type.  GUID implements Kusto.
type GUID struct {
	// Value holds the value of the type.
	Value uuid.UUID
	// Valid indicates if this value was set.
	Valid bool
}

func (GUID) isKustoVal() {}

// String implements fmt.Stringer.
func (g GUID) String() string {
	if !g.Valid {
		return ""
	}
	return g.Value.String()
}

// Unmarshal unmarshals i into GUID. i must be a string representing a GUID or nil.
func (g *GUID) Unmarshal(i interface{}) error {
	if i == nil {
		g.Value = uuid.UUID{}
		g.Valid = false
		return nil
	}
	str, ok := i.(string)
	if !ok {
		return fmt.Errorf("Column with type 'guid' was not stored as a string, was %T", i)
	}
	u, err := uuid.Parse(str)
	if err != nil {
		return fmt.Errorf("Column with type 'guid' did not store a valid uuid(%s): %s", str, err)
	}
	g.Value = u
	g.Valid = true
	return nil
}

// Convert GUID into reflect value.
func (g GUID) Convert(v reflect.Value) error {
	t := v.Type()
	switch {
	case t.AssignableTo(reflect.TypeOf(uuid.UUID{})):
		if g.Valid {
			v.Set(reflect.ValueOf(g.Value))
		}
		return nil
	case t.ConvertibleTo(reflect.TypeOf(new(uuid.UUID))):
		if g.Valid {
			t := &g.Value
			v.Set(reflect.ValueOf(t))
		}
		return nil
	case t.ConvertibleTo(reflect.TypeOf(GUID{})):
		v.Set(reflect.ValueOf(g))
		return nil
	case t.ConvertibleTo(reflect.TypeOf(&GUID{})):
		v.Set(reflect.ValueOf(&g))
		return nil
	}
	return fmt.Errorf("Column was type Kusto.GUID, receiver had base Kind %s ", t.Kind())
}
