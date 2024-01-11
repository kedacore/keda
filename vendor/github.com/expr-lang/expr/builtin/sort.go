package builtin

import (
	"fmt"
	"reflect"
)

type Sortable struct {
	Array  []any
	Values []reflect.Value
	OrderBy
}

type OrderBy struct {
	Field string
	Desc  bool
}

func (s *Sortable) Len() int {
	return len(s.Array)
}

func (s *Sortable) Swap(i, j int) {
	s.Array[i], s.Array[j] = s.Array[j], s.Array[i]
	s.Values[i], s.Values[j] = s.Values[j], s.Values[i]
}

func (s *Sortable) Less(i, j int) bool {
	a, b := s.Values[i], s.Values[j]
	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if s.Desc {
			return a.Int() > b.Int()
		}
		return a.Int() < b.Int()
	case reflect.Float64, reflect.Float32:
		if s.Desc {
			return a.Float() > b.Float()
		}
		return a.Float() < b.Float()
	case reflect.String:
		if s.Desc {
			return a.String() > b.String()
		}
		return a.String() < b.String()
	default:
		panic(fmt.Sprintf("sort: unsupported type %s", a.Kind()))
	}
}

func copyArray(v reflect.Value, orderBy OrderBy) (*Sortable, error) {
	s := &Sortable{
		Array:   make([]any, v.Len()),
		Values:  make([]reflect.Value, v.Len()),
		OrderBy: orderBy,
	}
	var prev reflect.Value
	for i := 0; i < s.Len(); i++ {
		elem := deref(v.Index(i))
		var value reflect.Value
		switch elem.Kind() {
		case reflect.Struct:
			value = elem.FieldByName(s.Field)
		case reflect.Map:
			value = elem.MapIndex(reflect.ValueOf(s.Field))
		default:
			value = elem
		}
		value = deref(value)

		s.Array[i] = elem.Interface()
		s.Values[i] = value

		if i == 0 {
			prev = value
		} else if value.Type() != prev.Type() {
			return nil, fmt.Errorf("cannot sort array of different types (%s and %s)", value.Type(), prev.Type())
		}
	}
	return s, nil
}

func ascOrDesc(arg any) (bool, error) {
	dir, ok := arg.(string)
	if !ok {
		return false, fmt.Errorf("invalid argument for sort (expected string, got %s)", reflect.TypeOf(arg))
	}
	switch dir {
	case "desc":
		return true, nil
	case "asc":
		return false, nil
	default:
		return false, fmt.Errorf(`invalid argument for sort (expected "asc" or "desc", got %q)`, dir)
	}
}
