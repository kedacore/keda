package builtin

import (
	"fmt"
	"reflect"
)

var (
	anyType     = reflect.TypeOf(new(any)).Elem()
	integerType = reflect.TypeOf(0)
	floatType   = reflect.TypeOf(float64(0))
	arrayType   = reflect.TypeOf([]any{})
	mapType     = reflect.TypeOf(map[any]any{})
)

func kind(t reflect.Type) reflect.Kind {
	if t == nil {
		return reflect.Invalid
	}
	return t.Kind()
}

func types(types ...any) []reflect.Type {
	ts := make([]reflect.Type, len(types))
	for i, t := range types {
		t := reflect.TypeOf(t)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() != reflect.Func {
			panic("not a function")
		}
		ts[i] = t
	}
	return ts
}

func deref(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return v
		}
		v = v.Elem()
	}

loop:
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return v
		}
		indirect := reflect.Indirect(v)
		switch indirect.Kind() {
		case reflect.Struct, reflect.Map, reflect.Array, reflect.Slice:
			break loop
		default:
			v = v.Elem()
		}
	}

	if v.IsValid() {
		return v
	}

	panic(fmt.Sprintf("cannot deref %s", v))
}
