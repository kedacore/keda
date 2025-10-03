package otto

import (
	"reflect"
)

func (rt *runtime) newGoMapObject(value reflect.Value) *object {
	obj := rt.newObject()
	obj.class = classObjectName // TODO Should this be something else?
	obj.objectClass = classGoMap
	obj.value = newGoMapObject(value)
	return obj
}

type goMapObject struct {
	keyType   reflect.Type
	valueType reflect.Type
	value     reflect.Value
}

func newGoMapObject(value reflect.Value) *goMapObject {
	if value.Kind() != reflect.Map {
		dbgf("%/panic//%@: %v != reflect.Map", value.Kind())
	}
	return &goMapObject{
		value:     value,
		keyType:   value.Type().Key(),
		valueType: value.Type().Elem(),
	}
}

func (o goMapObject) toKey(name string) reflect.Value {
	reflectValue, err := stringToReflectValue(name, o.keyType.Kind())
	if err != nil {
		panic(err)
	}
	return reflectValue
}

func (o goMapObject) toValue(value Value) reflect.Value {
	reflectValue, err := value.toReflectValue(o.valueType)
	if err != nil {
		panic(err)
	}
	return reflectValue
}

func goMapGetOwnProperty(obj *object, name string) *property {
	goObj := obj.value.(*goMapObject)

	// an error here means that the key referenced by `name` could not possibly
	// be a property of this object, so it should be safe to ignore this error
	//
	// TODO: figure out if any cases from
	// https://go.dev/ref/spec#Comparison_operators meet the criteria of 1)
	// being possible to represent as a string, 2) being possible to reconstruct
	// from a string, and 3) having a meaningful failure case in this context
	// other than "key does not exist"
	key, err := stringToReflectValue(name, goObj.keyType.Kind())
	if err != nil {
		return nil
	}

	value := goObj.value.MapIndex(key)
	if value.IsValid() {
		return &property{obj.runtime.toValue(value.Interface()), 0o111}
	}

	// Other methods
	if method := obj.value.(*goMapObject).value.MethodByName(name); method.IsValid() {
		return &property{
			value: obj.runtime.toValue(method.Interface()),
			mode:  0o110,
		}
	}

	return nil
}

func goMapEnumerate(obj *object, all bool, each func(string) bool) {
	goObj := obj.value.(*goMapObject)
	keys := goObj.value.MapKeys()
	for _, key := range keys {
		if !each(toValue(key).String()) {
			return
		}
	}
}

func goMapDefineOwnProperty(obj *object, name string, descriptor property, throw bool) bool {
	goObj := obj.value.(*goMapObject)
	// TODO ...or 0222
	if descriptor.mode != 0o111 {
		return obj.runtime.typeErrorResult(throw)
	}
	if !descriptor.isDataDescriptor() {
		return obj.runtime.typeErrorResult(throw)
	}
	goObj.value.SetMapIndex(goObj.toKey(name), goObj.toValue(descriptor.value.(Value)))
	return true
}

func goMapDelete(obj *object, name string, throw bool) bool {
	goObj := obj.value.(*goMapObject)
	goObj.value.SetMapIndex(goObj.toKey(name), reflect.Value{})
	// FIXME
	return true
}
