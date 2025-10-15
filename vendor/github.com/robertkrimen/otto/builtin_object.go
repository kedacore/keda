package otto

import (
	"fmt"
)

// Object

func builtinObject(call FunctionCall) Value {
	value := call.Argument(0)
	switch value.kind {
	case valueUndefined, valueNull:
		return objectValue(call.runtime.newObject())
	}

	return objectValue(call.runtime.toObject(value))
}

func builtinNewObject(obj *object, argumentList []Value) Value {
	value := valueOfArrayIndex(argumentList, 0)
	switch value.kind {
	case valueNull, valueUndefined:
	case valueNumber, valueString, valueBoolean:
		return objectValue(obj.runtime.toObject(value))
	case valueObject:
		return value
	default:
	}
	return objectValue(obj.runtime.newObject())
}

func builtinObjectValueOf(call FunctionCall) Value {
	return objectValue(call.thisObject())
}

func builtinObjectHasOwnProperty(call FunctionCall) Value {
	propertyName := call.Argument(0).string()
	thisObject := call.thisObject()
	return boolValue(thisObject.hasOwnProperty(propertyName))
}

func builtinObjectIsPrototypeOf(call FunctionCall) Value {
	value := call.Argument(0)
	if !value.IsObject() {
		return falseValue
	}
	prototype := call.toObject(value).prototype
	thisObject := call.thisObject()
	for prototype != nil {
		if thisObject == prototype {
			return trueValue
		}
		prototype = prototype.prototype
	}
	return falseValue
}

func builtinObjectPropertyIsEnumerable(call FunctionCall) Value {
	propertyName := call.Argument(0).string()
	thisObject := call.thisObject()
	prop := thisObject.getOwnProperty(propertyName)
	if prop != nil && prop.enumerable() {
		return trueValue
	}
	return falseValue
}

func builtinObjectToString(call FunctionCall) Value {
	var result string
	switch {
	case call.This.IsUndefined():
		result = "[object Undefined]"
	case call.This.IsNull():
		result = "[object Null]"
	default:
		result = fmt.Sprintf("[object %s]", call.thisObject().class)
	}
	return stringValue(result)
}

func builtinObjectToLocaleString(call FunctionCall) Value {
	toString := call.thisObject().get("toString")
	if !toString.isCallable() {
		panic(call.runtime.panicTypeError("Object.toLocaleString %q is not callable", toString))
	}
	return toString.call(call.runtime, call.This)
}

func builtinObjectGetPrototypeOf(call FunctionCall) Value {
	val := call.Argument(0)
	obj := val.object()
	if obj == nil {
		panic(call.runtime.panicTypeError("Object.GetPrototypeOf is nil"))
	}

	if obj.prototype == nil {
		return nullValue
	}

	return objectValue(obj.prototype)
}

func builtinObjectGetOwnPropertyDescriptor(call FunctionCall) Value {
	val := call.Argument(0)
	obj := val.object()
	if obj == nil {
		panic(call.runtime.panicTypeError("Object.GetOwnPropertyDescriptor is nil"))
	}

	name := call.Argument(1).string()
	descriptor := obj.getOwnProperty(name)
	if descriptor == nil {
		return Value{}
	}
	return objectValue(call.runtime.fromPropertyDescriptor(*descriptor))
}

func builtinObjectDefineProperty(call FunctionCall) Value {
	val := call.Argument(0)
	obj := val.object()
	if obj == nil {
		panic(call.runtime.panicTypeError("Object.DefineProperty is nil"))
	}
	name := call.Argument(1).string()
	descriptor := toPropertyDescriptor(call.runtime, call.Argument(2))
	obj.defineOwnProperty(name, descriptor, true)
	return val
}

func builtinObjectDefineProperties(call FunctionCall) Value {
	val := call.Argument(0)
	obj := val.object()
	if obj == nil {
		panic(call.runtime.panicTypeError("Object.DefineProperties is nil"))
	}

	properties := call.runtime.toObject(call.Argument(1))
	properties.enumerate(false, func(name string) bool {
		descriptor := toPropertyDescriptor(call.runtime, properties.get(name))
		obj.defineOwnProperty(name, descriptor, true)
		return true
	})

	return val
}

func builtinObjectCreate(call FunctionCall) Value {
	prototypeValue := call.Argument(0)
	if !prototypeValue.IsNull() && !prototypeValue.IsObject() {
		panic(call.runtime.panicTypeError("Object.Create is nil"))
	}

	obj := call.runtime.newObject()
	obj.prototype = prototypeValue.object()

	propertiesValue := call.Argument(1)
	if propertiesValue.IsDefined() {
		properties := call.runtime.toObject(propertiesValue)
		properties.enumerate(false, func(name string) bool {
			descriptor := toPropertyDescriptor(call.runtime, properties.get(name))
			obj.defineOwnProperty(name, descriptor, true)
			return true
		})
	}

	return objectValue(obj)
}

func builtinObjectIsExtensible(call FunctionCall) Value {
	val := call.Argument(0)
	if obj := val.object(); obj != nil {
		return boolValue(obj.extensible)
	}
	panic(call.runtime.panicTypeError("Object.IsExtensible is nil"))
}

func builtinObjectPreventExtensions(call FunctionCall) Value {
	val := call.Argument(0)
	if obj := val.object(); obj != nil {
		obj.extensible = false
		return val
	}
	panic(call.runtime.panicTypeError("Object.PreventExtensions is nil"))
}

func builtinObjectIsSealed(call FunctionCall) Value {
	val := call.Argument(0)
	if obj := val.object(); obj != nil {
		if obj.extensible {
			return boolValue(false)
		}
		result := true
		obj.enumerate(true, func(name string) bool {
			prop := obj.getProperty(name)
			if prop.configurable() {
				result = false
			}
			return true
		})
		return boolValue(result)
	}
	panic(call.runtime.panicTypeError("Object.IsSealed is nil"))
}

func builtinObjectSeal(call FunctionCall) Value {
	val := call.Argument(0)
	if obj := val.object(); obj != nil {
		obj.enumerate(true, func(name string) bool {
			if prop := obj.getOwnProperty(name); nil != prop && prop.configurable() {
				prop.configureOff()
				obj.defineOwnProperty(name, *prop, true)
			}
			return true
		})
		obj.extensible = false
		return val
	}
	panic(call.runtime.panicTypeError("Object.Seal is nil"))
}

func builtinObjectIsFrozen(call FunctionCall) Value {
	val := call.Argument(0)
	if obj := val.object(); obj != nil {
		if obj.extensible {
			return boolValue(false)
		}
		result := true
		obj.enumerate(true, func(name string) bool {
			prop := obj.getProperty(name)
			if prop.configurable() || prop.writable() {
				result = false
			}
			return true
		})
		return boolValue(result)
	}
	panic(call.runtime.panicTypeError("Object.IsFrozen is nil"))
}

func builtinObjectFreeze(call FunctionCall) Value {
	val := call.Argument(0)
	if obj := val.object(); obj != nil {
		obj.enumerate(true, func(name string) bool {
			if prop, update := obj.getOwnProperty(name), false; nil != prop {
				if prop.isDataDescriptor() && prop.writable() {
					prop.writeOff()
					update = true
				}
				if prop.configurable() {
					prop.configureOff()
					update = true
				}
				if update {
					obj.defineOwnProperty(name, *prop, true)
				}
			}
			return true
		})
		obj.extensible = false
		return val
	}
	panic(call.runtime.panicTypeError("Object.Freeze is nil"))
}

func builtinObjectKeys(call FunctionCall) Value {
	if obj, keys := call.Argument(0).object(), []Value(nil); nil != obj {
		obj.enumerate(false, func(name string) bool {
			keys = append(keys, stringValue(name))
			return true
		})
		return objectValue(call.runtime.newArrayOf(keys))
	}
	panic(call.runtime.panicTypeError("Object.Keys is nil"))
}

func builtinObjectValues(call FunctionCall) Value {
	if obj, values := call.Argument(0).object(), []Value(nil); nil != obj {
		obj.enumerate(false, func(name string) bool {
			values = append(values, obj.get(name))
			return true
		})
		return objectValue(call.runtime.newArrayOf(values))
	}
	panic(call.runtime.panicTypeError("Object.Values is nil"))
}

func builtinObjectGetOwnPropertyNames(call FunctionCall) Value {
	if obj, propertyNames := call.Argument(0).object(), []Value(nil); nil != obj {
		obj.enumerate(true, func(name string) bool {
			if obj.hasOwnProperty(name) {
				propertyNames = append(propertyNames, stringValue(name))
			}
			return true
		})
		return objectValue(call.runtime.newArrayOf(propertyNames))
	}

	// Default to empty array for non object types.
	return objectValue(call.runtime.newArray(0))
}
