package otto

import (
	"strconv"
)

func (rt *runtime) newArgumentsObject(indexOfParameterName []string, stash stasher, length int) *object {
	obj := rt.newClassObject("Arguments")

	for index := range indexOfParameterName {
		name := strconv.FormatInt(int64(index), 10)
		objectDefineOwnProperty(obj, name, property{Value{}, 0o111}, false)
	}

	obj.objectClass = classArguments
	obj.value = argumentsObject{
		indexOfParameterName: indexOfParameterName,
		stash:                stash,
	}

	obj.prototype = rt.global.ObjectPrototype

	obj.defineProperty(propertyLength, intValue(length), 0o101, false)

	return obj
}

type argumentsObject struct {
	stash                stasher
	indexOfParameterName []string
}

func (o argumentsObject) clone(c *cloner) argumentsObject {
	indexOfParameterName := make([]string, len(o.indexOfParameterName))
	copy(indexOfParameterName, o.indexOfParameterName)
	return argumentsObject{
		indexOfParameterName: indexOfParameterName,
		stash:                c.stash(o.stash),
	}
}

func (o argumentsObject) get(name string) (Value, bool) {
	index := stringToArrayIndex(name)
	if index >= 0 && index < int64(len(o.indexOfParameterName)) {
		if name = o.indexOfParameterName[index]; name == "" {
			return Value{}, false
		}
		return o.stash.getBinding(name, false), true
	}
	return Value{}, false
}

func (o argumentsObject) put(name string, value Value) {
	index := stringToArrayIndex(name)
	name = o.indexOfParameterName[index]
	o.stash.setBinding(name, value, false)
}

func (o argumentsObject) delete(name string) {
	index := stringToArrayIndex(name)
	o.indexOfParameterName[index] = ""
}

func argumentsGet(obj *object, name string) Value {
	if value, exists := obj.value.(argumentsObject).get(name); exists {
		return value
	}
	return objectGet(obj, name)
}

func argumentsGetOwnProperty(obj *object, name string) *property {
	prop := objectGetOwnProperty(obj, name)
	if value, exists := obj.value.(argumentsObject).get(name); exists {
		prop.value = value
	}
	return prop
}

func argumentsDefineOwnProperty(obj *object, name string, descriptor property, throw bool) bool {
	if _, exists := obj.value.(argumentsObject).get(name); exists {
		if !objectDefineOwnProperty(obj, name, descriptor, false) {
			return obj.runtime.typeErrorResult(throw)
		}
		if value, valid := descriptor.value.(Value); valid {
			obj.value.(argumentsObject).put(name, value)
		}
		return true
	}
	return objectDefineOwnProperty(obj, name, descriptor, throw)
}

func argumentsDelete(obj *object, name string, throw bool) bool {
	if !objectDelete(obj, name, throw) {
		return false
	}
	if _, exists := obj.value.(argumentsObject).get(name); exists {
		obj.value.(argumentsObject).delete(name)
	}
	return true
}
