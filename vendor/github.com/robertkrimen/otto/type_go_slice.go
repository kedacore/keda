package otto

import (
	"reflect"
	"strconv"
)

func (rt *runtime) newGoSliceObject(value reflect.Value) *object {
	o := rt.newObject()
	o.class = classGoSliceName
	o.objectClass = classGoSlice
	o.value = newGoSliceObject(value)
	return o
}

type goSliceObject struct {
	value reflect.Value
}

func newGoSliceObject(value reflect.Value) *goSliceObject {
	return &goSliceObject{
		value: value,
	}
}

func (o goSliceObject) getValue(index int64) (reflect.Value, bool) {
	if index < int64(o.value.Len()) {
		return o.value.Index(int(index)), true
	}
	return reflect.Value{}, false
}

func (o *goSliceObject) setLength(value Value) {
	want, err := value.ToInteger()
	if err != nil {
		panic(err)
	}

	wantInt := int(want)
	switch {
	case wantInt == o.value.Len():
		// No change needed.
	case wantInt < o.value.Cap():
		// Fits in current capacity.
		o.value.SetLen(wantInt)
	default:
		// Needs expanding.
		newSlice := reflect.MakeSlice(o.value.Type(), wantInt, wantInt)
		reflect.Copy(newSlice, o.value)
		o.value = newSlice
	}
}

func (o *goSliceObject) setValue(index int64, value Value) bool {
	reflectValue, err := value.toReflectValue(o.value.Type().Elem())
	if err != nil {
		panic(err)
	}

	indexValue, exists := o.getValue(index)
	if !exists {
		if int64(o.value.Len()) == index {
			// Trying to append e.g. slice.push(...), allow it.
			o.value = reflect.Append(o.value, reflectValue)
			return true
		}
		return false
	}

	indexValue.Set(reflectValue)
	return true
}

func goSliceGetOwnProperty(obj *object, name string) *property {
	// length
	if name == propertyLength {
		return &property{
			value: toValue(obj.value.(*goSliceObject).value.Len()),
			mode:  0o110,
		}
	}

	// .0, .1, .2, ...
	if index := stringToArrayIndex(name); index >= 0 {
		value := Value{}
		reflectValue, exists := obj.value.(*goSliceObject).getValue(index)
		if exists {
			value = obj.runtime.toValue(reflectValue.Interface())
		}
		return &property{
			value: value,
			mode:  0o110,
		}
	}

	// Other methods
	if method := obj.value.(*goSliceObject).value.MethodByName(name); method.IsValid() {
		return &property{
			value: obj.runtime.toValue(method.Interface()),
			mode:  0o110,
		}
	}

	return objectGetOwnProperty(obj, name)
}

func goSliceEnumerate(obj *object, all bool, each func(string) bool) {
	goObj := obj.value.(*goSliceObject)
	// .0, .1, .2, ...

	for index, length := 0, goObj.value.Len(); index < length; index++ {
		name := strconv.FormatInt(int64(index), 10)
		if !each(name) {
			return
		}
	}

	objectEnumerate(obj, all, each)
}

func goSliceDefineOwnProperty(obj *object, name string, descriptor property, throw bool) bool {
	if name == propertyLength {
		obj.value.(*goSliceObject).setLength(descriptor.value.(Value))
		return true
	} else if index := stringToArrayIndex(name); index >= 0 {
		if obj.value.(*goSliceObject).setValue(index, descriptor.value.(Value)) {
			return true
		}
		return obj.runtime.typeErrorResult(throw)
	}
	return objectDefineOwnProperty(obj, name, descriptor, throw)
}

func goSliceDelete(obj *object, name string, throw bool) bool {
	// length
	if name == propertyLength {
		return obj.runtime.typeErrorResult(throw)
	}

	// .0, .1, .2, ...
	index := stringToArrayIndex(name)
	if index >= 0 {
		goObj := obj.value.(*goSliceObject)
		indexValue, exists := goObj.getValue(index)
		if exists {
			indexValue.Set(reflect.Zero(goObj.value.Type().Elem()))
			return true
		}
		return obj.runtime.typeErrorResult(throw)
	}

	return obj.delete(name, throw)
}
