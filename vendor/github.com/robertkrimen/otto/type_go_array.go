package otto

import (
	"reflect"
	"strconv"
)

func (rt *runtime) newGoArrayObject(value reflect.Value) *object {
	o := rt.newObject()
	o.class = classGoArrayName
	o.objectClass = classGoArray
	o.value = newGoArrayObject(value)
	return o
}

type goArrayObject struct {
	value        reflect.Value
	writable     bool
	propertyMode propertyMode
}

func newGoArrayObject(value reflect.Value) *goArrayObject {
	writable := value.Kind() == reflect.Ptr || value.CanSet() // The Array is addressable (like a Slice)
	mode := propertyMode(0o010)
	if writable {
		mode = 0o110
	}

	return &goArrayObject{
		value:        value,
		writable:     writable,
		propertyMode: mode,
	}
}

func (o goArrayObject) getValue(name string) (reflect.Value, bool) { //nolint:unused
	if index, err := strconv.ParseInt(name, 10, 64); err != nil {
		v, ok := o.getValueIndex(index)
		if ok {
			return v, ok
		}
	}

	if m := o.value.MethodByName(name); m.IsValid() {
		return m, true
	}

	return reflect.Value{}, false
}

func (o goArrayObject) getValueIndex(index int64) (reflect.Value, bool) {
	value := reflect.Indirect(o.value)
	if index < int64(value.Len()) {
		return value.Index(int(index)), true
	}

	return reflect.Value{}, false
}

func (o goArrayObject) setValue(index int64, value Value) bool {
	indexValue, exists := o.getValueIndex(index)
	if !exists {
		return false
	}
	reflectValue, err := value.toReflectValue(reflect.Indirect(o.value).Type().Elem())
	if err != nil {
		panic(err)
	}
	indexValue.Set(reflectValue)
	return true
}

func goArrayGetOwnProperty(obj *object, name string) *property {
	// length
	if name == propertyLength {
		return &property{
			value: toValue(reflect.Indirect(obj.value.(*goArrayObject).value).Len()),
			mode:  0,
		}
	}

	// .0, .1, .2, ...
	if index := stringToArrayIndex(name); index >= 0 {
		goObj := obj.value.(*goArrayObject)
		value := Value{}
		reflectValue, exists := goObj.getValueIndex(index)
		if exists {
			value = obj.runtime.toValue(reflectValue.Interface())
		}
		return &property{
			value: value,
			mode:  goObj.propertyMode,
		}
	}

	if method := obj.value.(*goArrayObject).value.MethodByName(name); method.IsValid() {
		return &property{
			obj.runtime.toValue(method.Interface()),
			0o110,
		}
	}

	return objectGetOwnProperty(obj, name)
}

func goArrayEnumerate(obj *object, all bool, each func(string) bool) {
	goObj := obj.value.(*goArrayObject)
	// .0, .1, .2, ...

	for index, length := 0, goObj.value.Len(); index < length; index++ {
		name := strconv.FormatInt(int64(index), 10)
		if !each(name) {
			return
		}
	}

	objectEnumerate(obj, all, each)
}

func goArrayDefineOwnProperty(obj *object, name string, descriptor property, throw bool) bool {
	if name == propertyLength {
		return obj.runtime.typeErrorResult(throw)
	} else if index := stringToArrayIndex(name); index >= 0 {
		goObj := obj.value.(*goArrayObject)
		if goObj.writable {
			if obj.value.(*goArrayObject).setValue(index, descriptor.value.(Value)) {
				return true
			}
		}
		return obj.runtime.typeErrorResult(throw)
	}
	return objectDefineOwnProperty(obj, name, descriptor, throw)
}

func goArrayDelete(obj *object, name string, throw bool) bool {
	// length
	if name == propertyLength {
		return obj.runtime.typeErrorResult(throw)
	}

	// .0, .1, .2, ...
	index := stringToArrayIndex(name)
	if index >= 0 {
		goObj := obj.value.(*goArrayObject)
		if goObj.writable {
			indexValue, exists := goObj.getValueIndex(index)
			if exists {
				indexValue.Set(reflect.Zero(reflect.Indirect(goObj.value).Type().Elem()))
				return true
			}
		}
		return obj.runtime.typeErrorResult(throw)
	}

	return obj.delete(name, throw)
}
