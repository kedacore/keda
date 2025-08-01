package otto

import (
	"strconv"
)

func (rt *runtime) newArrayObject(length uint32) *object {
	obj := rt.newObject()
	obj.class = classArrayName
	obj.defineProperty(propertyLength, uint32Value(length), 0o100, false)
	obj.objectClass = classArray
	return obj
}

func isArray(obj *object) bool {
	if obj == nil {
		return false
	}

	switch obj.class {
	case classArrayName, classGoArrayName, classGoSliceName:
		return true
	default:
		return false
	}
}

func objectLength(obj *object) uint32 {
	if obj == nil {
		return 0
	}
	switch obj.class {
	case classArrayName:
		return obj.get(propertyLength).value.(uint32)
	case classStringName:
		return uint32(obj.get(propertyLength).value.(int))
	case classGoArrayName, classGoSliceName:
		return uint32(obj.get(propertyLength).value.(int))
	}
	return 0
}

func arrayUint32(rt *runtime, value Value) uint32 {
	nm := value.number()
	if nm.kind != numberInteger || !isUint32(nm.int64) {
		// FIXME
		panic(rt.panicRangeError())
	}
	return uint32(nm.int64)
}

func arrayDefineOwnProperty(obj *object, name string, descriptor property, throw bool) bool {
	lengthProperty := obj.getOwnProperty(propertyLength)
	lengthValue, valid := lengthProperty.value.(Value)
	if !valid {
		panic("Array.length != Value{}")
	}

	reject := func(reason string) bool {
		if throw {
			panic(obj.runtime.panicTypeError("Array.DefineOwnProperty %s", reason))
		}
		return false
	}
	length := lengthValue.value.(uint32)
	if name == propertyLength {
		if descriptor.value == nil {
			return objectDefineOwnProperty(obj, name, descriptor, throw)
		}
		newLengthValue, isValue := descriptor.value.(Value)
		if !isValue {
			panic(obj.runtime.panicTypeError("Array.DefineOwnProperty %q is not a value", descriptor.value))
		}
		newLength := arrayUint32(obj.runtime, newLengthValue)
		descriptor.value = uint32Value(newLength)
		if newLength > length {
			return objectDefineOwnProperty(obj, name, descriptor, throw)
		}
		if !lengthProperty.writable() {
			return reject("property length for not writable")
		}
		newWritable := true
		if descriptor.mode&0o700 == 0 {
			// If writable is off
			newWritable = false
			descriptor.mode |= 0o100
		}
		if !objectDefineOwnProperty(obj, name, descriptor, throw) {
			return false
		}
		for newLength < length {
			length--
			if !obj.delete(strconv.FormatInt(int64(length), 10), false) {
				descriptor.value = uint32Value(length + 1)
				if !newWritable {
					descriptor.mode &= 0o077
				}
				objectDefineOwnProperty(obj, name, descriptor, false)
				return reject("delete failed")
			}
		}
		if !newWritable {
			descriptor.mode &= 0o077
			objectDefineOwnProperty(obj, name, descriptor, false)
		}
	} else if index := stringToArrayIndex(name); index >= 0 {
		if index >= int64(length) && !lengthProperty.writable() {
			return reject("property length not writable")
		}
		if !objectDefineOwnProperty(obj, strconv.FormatInt(index, 10), descriptor, false) {
			return reject("Object.DefineOwnProperty failed")
		}
		if index >= int64(length) {
			lengthProperty.value = uint32Value(uint32(index + 1))
			objectDefineOwnProperty(obj, propertyLength, *lengthProperty, false)
			return true
		}
	}
	return objectDefineOwnProperty(obj, name, descriptor, throw)
}
