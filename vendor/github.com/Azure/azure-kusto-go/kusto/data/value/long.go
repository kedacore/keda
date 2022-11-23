package value

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
)

// Long represents a Kusto long type, which is an int64.  Long implements Kusto.
type Long struct {
	// Value holds the value of the type.
	Value int64
	// Valid indicates if this value was set.
	Valid bool
}

func (Long) isKustoVal() {}

// String implements fmt.Stringer.
func (l Long) String() string {
	if !l.Valid {
		return ""
	}
	return strconv.Itoa(int(l.Value))
}

// Unmarshal unmarshals i into Long. i must be an int64 or nil.
func (l *Long) Unmarshal(i interface{}) error {
	if i == nil {
		l.Value = 0
		l.Valid = false
		return nil
	}

	var myInt int64

	switch v := i.(type) {
	case json.Number:
		var err error
		myInt, err = v.Int64()
		if err != nil {
			return fmt.Errorf("Column with type 'long' had value json.Number that had error on .Int64(): %s", err)
		}
	case int:
		myInt = int64(v)
	case float64:
		if v != math.Trunc(v) {
			return fmt.Errorf("Column with type 'int' had value float64(%v) that did not represent a whole number", v)
		}
		myInt = int64(v)
	default:
		return fmt.Errorf("Column with type 'ong' had value that was not a json.Number or int, was %T", i)
	}

	l.Value = myInt
	l.Valid = true
	return nil
}

// Convert Long into reflect value.
func (l Long) Convert(v reflect.Value) error {
	t := v.Type()
	switch {
	case t.Kind() == reflect.Int64:
		if l.Valid {
			v.Set(reflect.ValueOf(l.Value))
		}
		return nil
	case t.ConvertibleTo(reflect.TypeOf(new(int64))):
		if l.Valid {
			i := &l.Value
			fmt.Println(i)
			v.Set(reflect.ValueOf(i))
		}
		return nil
	case t.ConvertibleTo(reflect.TypeOf(Long{})):
		v.Set(reflect.ValueOf(l))
		return nil
	case t.ConvertibleTo(reflect.TypeOf(&Long{})):
		v.Set(reflect.ValueOf(&l))
		return nil
	}
	return fmt.Errorf("Column was type Kusto.Long, receiver had base Kind %s ", t.Kind())
}
