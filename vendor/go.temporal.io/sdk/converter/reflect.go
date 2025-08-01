package converter

import (
	"reflect"
)

func pointerTo(val interface{}) reflect.Value {
	valPtr := reflect.New(reflect.TypeOf(val))
	valPtr.Elem().Set(reflect.ValueOf(val))
	return valPtr
}

func newOfSameType(val reflect.Value) reflect.Value {
	valType := val.Type().Elem()     // is value type (i.e. commonpb.WorkflowType)
	newValue := reflect.New(valType) // is of pointer type (i.e. *commonpb.WorkflowType)
	val.Set(newValue)                // set newly created value back to passed value
	return newValue
}

func isInterfaceNil(i interface{}) bool {
	v := reflect.ValueOf(i)
	return i == nil || (v.Kind() == reflect.Ptr && v.IsNil())
}
