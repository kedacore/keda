package otto

import (
	"encoding/json"
)

type objectClass struct {
	getOwnProperty    func(*object, string) *property
	getProperty       func(*object, string) *property
	get               func(*object, string) Value
	canPut            func(*object, string) bool
	put               func(*object, string, Value, bool)
	hasProperty       func(*object, string) bool
	hasOwnProperty    func(*object, string) bool
	defineOwnProperty func(*object, string, property, bool) bool
	delete            func(*object, string, bool) bool
	enumerate         func(*object, bool, func(string) bool)
	clone             func(*object, *object, *cloner) *object
	marshalJSON       func(*object) json.Marshaler
}

func objectEnumerate(obj *object, all bool, each func(string) bool) {
	for _, name := range obj.propertyOrder {
		if all || obj.property[name].enumerable() {
			if !each(name) {
				return
			}
		}
	}
}

var classObject,
	classArray,
	classString,
	classArguments,
	classGoStruct,
	classGoMap,
	classGoArray,
	classGoSlice *objectClass

func init() {
	classObject = &objectClass{
		objectGetOwnProperty,
		objectGetProperty,
		objectGet,
		objectCanPut,
		objectPut,
		objectHasProperty,
		objectHasOwnProperty,
		objectDefineOwnProperty,
		objectDelete,
		objectEnumerate,
		objectClone,
		nil,
	}

	classArray = &objectClass{
		objectGetOwnProperty,
		objectGetProperty,
		objectGet,
		objectCanPut,
		objectPut,
		objectHasProperty,
		objectHasOwnProperty,
		arrayDefineOwnProperty,
		objectDelete,
		objectEnumerate,
		objectClone,
		nil,
	}

	classString = &objectClass{
		stringGetOwnProperty,
		objectGetProperty,
		objectGet,
		objectCanPut,
		objectPut,
		objectHasProperty,
		objectHasOwnProperty,
		objectDefineOwnProperty,
		objectDelete,
		stringEnumerate,
		objectClone,
		nil,
	}

	classArguments = &objectClass{
		argumentsGetOwnProperty,
		objectGetProperty,
		argumentsGet,
		objectCanPut,
		objectPut,
		objectHasProperty,
		objectHasOwnProperty,
		argumentsDefineOwnProperty,
		argumentsDelete,
		objectEnumerate,
		objectClone,
		nil,
	}

	classGoStruct = &objectClass{
		goStructGetOwnProperty,
		objectGetProperty,
		objectGet,
		goStructCanPut,
		goStructPut,
		objectHasProperty,
		objectHasOwnProperty,
		objectDefineOwnProperty,
		objectDelete,
		goStructEnumerate,
		objectClone,
		goStructMarshalJSON,
	}

	classGoMap = &objectClass{
		goMapGetOwnProperty,
		objectGetProperty,
		objectGet,
		objectCanPut,
		objectPut,
		objectHasProperty,
		objectHasOwnProperty,
		goMapDefineOwnProperty,
		goMapDelete,
		goMapEnumerate,
		objectClone,
		nil,
	}

	classGoArray = &objectClass{
		goArrayGetOwnProperty,
		objectGetProperty,
		objectGet,
		objectCanPut,
		objectPut,
		objectHasProperty,
		objectHasOwnProperty,
		goArrayDefineOwnProperty,
		goArrayDelete,
		goArrayEnumerate,
		objectClone,
		nil,
	}

	classGoSlice = &objectClass{
		goSliceGetOwnProperty,
		objectGetProperty,
		objectGet,
		objectCanPut,
		objectPut,
		objectHasProperty,
		objectHasOwnProperty,
		goSliceDefineOwnProperty,
		goSliceDelete,
		goSliceEnumerate,
		objectClone,
		nil,
	}
}

// Allons-y

// 8.12.1.
func objectGetOwnProperty(obj *object, name string) *property {
	// Return a _copy_ of the prop
	prop, exists := obj.readProperty(name)
	if !exists {
		return nil
	}
	return &prop
}

// 8.12.2.
func objectGetProperty(obj *object, name string) *property {
	prop := obj.getOwnProperty(name)
	if prop != nil {
		return prop
	}
	if obj.prototype != nil {
		return obj.prototype.getProperty(name)
	}
	return nil
}

// 8.12.3.
func objectGet(obj *object, name string) Value {
	if prop := obj.getProperty(name); prop != nil {
		return prop.get(obj)
	}
	return Value{}
}

// 8.12.4.
func objectCanPut(obj *object, name string) bool {
	canPut, _, _ := objectCanPutDetails(obj, name)
	return canPut
}

func objectCanPutDetails(obj *object, name string) (canPut bool, prop *property, setter *object) { //nolint:nonamedreturns
	prop = obj.getOwnProperty(name)
	if prop != nil {
		switch propertyValue := prop.value.(type) {
		case Value:
			return prop.writable(), prop, nil
		case propertyGetSet:
			setter = propertyValue[1]
			return setter != nil, prop, setter
		default:
			panic(obj.runtime.panicTypeError("unexpected type %T to Object.CanPutDetails", prop.value))
		}
	}

	if obj.prototype == nil {
		return obj.extensible, nil, nil
	}

	prop = obj.prototype.getProperty(name)
	if prop == nil {
		return obj.extensible, nil, nil
	}

	switch propertyValue := prop.value.(type) {
	case Value:
		if !obj.extensible {
			return false, nil, nil
		}
		return prop.writable(), nil, nil
	case propertyGetSet:
		setter = propertyValue[1]
		return setter != nil, prop, setter
	default:
		panic(obj.runtime.panicTypeError("unexpected type %T to Object.CanPutDetails", prop.value))
	}
}

// 8.12.5.
func objectPut(obj *object, name string, value Value, throw bool) {
	if true {
		// Shortcut...
		//
		// So, right now, every class is using objectCanPut and every class
		// is using objectPut.
		//
		// If that were to no longer be the case, we would have to have
		// something to detect that here, so that we do not use an
		// incompatible canPut routine
		canPut, prop, setter := objectCanPutDetails(obj, name)
		switch {
		case !canPut:
			obj.runtime.typeErrorResult(throw)
		case setter != nil:
			setter.call(toValue(obj), []Value{value}, false, nativeFrame)
		case prop != nil:
			prop.value = value
			obj.defineOwnProperty(name, *prop, throw)
		default:
			obj.defineProperty(name, value, 0o111, throw)
		}
		return
	}

	// The long way...
	//
	// Right now, code should never get here, see above
	if !obj.canPut(name) {
		obj.runtime.typeErrorResult(throw)
		return
	}

	prop := obj.getOwnProperty(name)
	if prop == nil {
		prop = obj.getProperty(name)
		if prop != nil {
			if getSet, isAccessor := prop.value.(propertyGetSet); isAccessor {
				getSet[1].call(toValue(obj), []Value{value}, false, nativeFrame)
				return
			}
		}
		obj.defineProperty(name, value, 0o111, throw)
		return
	}

	switch propertyValue := prop.value.(type) {
	case Value:
		prop.value = value
		obj.defineOwnProperty(name, *prop, throw)
	case propertyGetSet:
		if propertyValue[1] != nil {
			propertyValue[1].call(toValue(obj), []Value{value}, false, nativeFrame)
			return
		}
		if throw {
			panic(obj.runtime.panicTypeError("Object.Put nil second parameter to propertyGetSet"))
		}
	default:
		panic(obj.runtime.panicTypeError("Object.Put unexpected type %T", prop.value))
	}
}

// 8.12.6.
func objectHasProperty(obj *object, name string) bool {
	return obj.getProperty(name) != nil
}

func objectHasOwnProperty(obj *object, name string) bool {
	return obj.getOwnProperty(name) != nil
}

// 8.12.9.
func objectDefineOwnProperty(obj *object, name string, descriptor property, throw bool) bool {
	reject := func(reason string) bool {
		if throw {
			panic(obj.runtime.panicTypeError("Object.DefineOwnProperty: %s", reason))
		}
		return false
	}

	prop, exists := obj.readProperty(name)
	if !exists {
		if !obj.extensible {
			return reject("not exists and not extensible")
		}
		if newGetSet, isAccessor := descriptor.value.(propertyGetSet); isAccessor {
			if newGetSet[0] == &nilGetSetObject {
				newGetSet[0] = nil
			}
			if newGetSet[1] == &nilGetSetObject {
				newGetSet[1] = nil
			}
			descriptor.value = newGetSet
		}
		obj.writeProperty(name, descriptor.value, descriptor.mode)
		return true
	}

	if descriptor.isEmpty() {
		return true
	}

	// TODO Per 8.12.9.6 - We should shortcut here (returning true) if
	// the current and new (define) properties are the same

	configurable := prop.configurable()
	if !configurable {
		if descriptor.configurable() {
			return reject("property and descriptor not configurable")
		}
		// Test that, if enumerable is set on the property descriptor, then it should
		// be the same as the existing property
		if descriptor.enumerateSet() && descriptor.enumerable() != prop.enumerable() {
			return reject("property not configurable and enumerable miss match")
		}
	}

	value, isDataDescriptor := prop.value.(Value)
	getSet, _ := prop.value.(propertyGetSet)
	switch {
	case descriptor.isGenericDescriptor():
		// GenericDescriptor
	case isDataDescriptor != descriptor.isDataDescriptor():
		// DataDescriptor <=> AccessorDescriptor
		if !configurable {
			return reject("property descriptor not configurable")
		}
	case isDataDescriptor && descriptor.isDataDescriptor():
		// DataDescriptor <=> DataDescriptor
		if !configurable {
			if !prop.writable() && descriptor.writable() {
				return reject("property not configurable or writeable and descriptor not writeable")
			}
			if !prop.writable() {
				if descriptor.value != nil && !sameValue(value, descriptor.value.(Value)) {
					return reject("property not configurable or writeable and descriptor not the same")
				}
			}
		}
	default:
		// AccessorDescriptor <=> AccessorDescriptor
		newGetSet, _ := descriptor.value.(propertyGetSet)
		presentGet, presentSet := true, true
		if newGetSet[0] == &nilGetSetObject {
			// Present, but nil
			newGetSet[0] = nil
		} else if newGetSet[0] == nil {
			// Missing, not even nil
			newGetSet[0] = getSet[0]
			presentGet = false
		}
		if newGetSet[1] == &nilGetSetObject {
			// Present, but nil
			newGetSet[1] = nil
		} else if newGetSet[1] == nil {
			// Missing, not even nil
			newGetSet[1] = getSet[1]
			presentSet = false
		}
		if !configurable {
			if (presentGet && (getSet[0] != newGetSet[0])) || (presentSet && (getSet[1] != newGetSet[1])) {
				return reject("access descriptor not configurable")
			}
		}
		descriptor.value = newGetSet
	}

	// This section will preserve attributes of
	// the original property, if necessary
	value1 := descriptor.value
	if value1 == nil {
		value1 = prop.value
	} else if newGetSet, isAccessor := descriptor.value.(propertyGetSet); isAccessor {
		if newGetSet[0] == &nilGetSetObject {
			newGetSet[0] = nil
		}
		if newGetSet[1] == &nilGetSetObject {
			newGetSet[1] = nil
		}
		value1 = newGetSet
	}
	mode1 := descriptor.mode
	if mode1&0o222 != 0 {
		// TODO Factor this out into somewhere testable
		// (Maybe put into switch ...)
		mode0 := prop.mode
		if mode1&0o200 != 0 {
			if descriptor.isDataDescriptor() {
				mode1 &= ^0o200 // Turn off "writable" missing
				mode1 |= (mode0 & 0o100)
			}
		}
		if mode1&0o20 != 0 {
			mode1 |= (mode0 & 0o10)
		}
		if mode1&0o2 != 0 {
			mode1 |= (mode0 & 0o1)
		}
		mode1 &= 0o311 // 0311 to preserve the non-setting on "writable"
	}
	obj.writeProperty(name, value1, mode1)

	return true
}

func objectDelete(obj *object, name string, throw bool) bool {
	prop := obj.getOwnProperty(name)
	if prop == nil {
		return true
	}
	if prop.configurable() {
		obj.deleteProperty(name)
		return true
	}
	return obj.runtime.typeErrorResult(throw)
}

func objectClone(in *object, out *object, clone *cloner) *object {
	*out = *in

	out.runtime = clone.runtime
	if out.prototype != nil {
		out.prototype = clone.object(in.prototype)
	}
	out.property = make(map[string]property, len(in.property))
	out.propertyOrder = make([]string, len(in.propertyOrder))
	copy(out.propertyOrder, in.propertyOrder)
	for index, prop := range in.property {
		out.property[index] = clone.property(prop)
	}

	switch value := in.value.(type) {
	case nativeFunctionObject:
		out.value = value
	case bindFunctionObject:
		out.value = bindFunctionObject{
			target:       clone.object(value.target),
			this:         clone.value(value.this),
			argumentList: clone.valueArray(value.argumentList),
		}
	case nodeFunctionObject:
		out.value = nodeFunctionObject{
			node:  value.node,
			stash: clone.stash(value.stash),
		}
	case argumentsObject:
		out.value = value.clone(clone)
	}

	return out
}
