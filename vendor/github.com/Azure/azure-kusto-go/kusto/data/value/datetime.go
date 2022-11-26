package value

import (
	"fmt"
	"reflect"
	"time"
)

// DateTime represents a Kusto datetime type.  DateTime implements Kusto.
type DateTime struct {
	// Value holds the value of the type.
	Value time.Time
	// Valid indicates if this value was set.
	Valid bool
}

// String implements fmt.Stringer.
func (d DateTime) String() string {
	if !d.Valid {
		return ""
	}
	return fmt.Sprint(d.Value.Format(time.RFC3339Nano))
}

func (DateTime) isKustoVal() {}

// Marshal marshals the DateTime into a Kusto compatible string.
func (d DateTime) Marshal() string {
	if !d.Valid {
		return time.Time{}.Format(time.RFC3339Nano)
	}
	return d.Value.Format(time.RFC3339Nano)
}

// Unmarshal unmarshals i into DateTime. i must be a string representing RFC3339Nano or nil.
func (d *DateTime) Unmarshal(i interface{}) error {
	if i == nil {
		d.Value = time.Time{}
		d.Valid = false
		return nil
	}

	str, ok := i.(string)
	if !ok {
		return fmt.Errorf("Column with type 'datetime' had value that was %T", i)
	}

	t, err := time.Parse(time.RFC3339Nano, str)
	if err != nil {
		return fmt.Errorf("Column with type 'datetime' had value %s which did not parse: %s", str, err)
	}
	d.Value = t
	d.Valid = true

	return nil
}

// Convert DateTime into reflect value.
func (d DateTime) Convert(v reflect.Value) error {
	t := v.Type()
	switch {
	case t.AssignableTo(reflect.TypeOf(time.Time{})):
		if d.Valid {
			v.Set(reflect.ValueOf(d.Value))
		}
		return nil
	case t.ConvertibleTo(reflect.TypeOf(new(time.Time))):
		if d.Valid {
			t := &d.Value
			v.Set(reflect.ValueOf(t))
		}
		return nil
	case t.ConvertibleTo(reflect.TypeOf(DateTime{})):
		v.Set(reflect.ValueOf(d))
		return nil
	case t.ConvertibleTo(reflect.TypeOf(&DateTime{})):
		v.Set(reflect.ValueOf(&d))
		return nil
	}
	return fmt.Errorf("Column was type Kusto.DateTime, receiver had base Kind %s ", t.Kind())
}
