package otto

// Boolean

func builtinBoolean(call FunctionCall) Value {
	return boolValue(call.Argument(0).bool())
}

func builtinNewBoolean(obj *object, argumentList []Value) Value {
	return objectValue(obj.runtime.newBoolean(valueOfArrayIndex(argumentList, 0)))
}

func builtinBooleanToString(call FunctionCall) Value {
	value := call.This
	if !value.IsBoolean() {
		// Will throw a TypeError if ThisObject is not a Boolean
		value = call.thisClassObject(classBooleanName).primitiveValue()
	}
	return stringValue(value.string())
}

func builtinBooleanValueOf(call FunctionCall) Value {
	value := call.This
	if !value.IsBoolean() {
		value = call.thisClassObject(classBooleanName).primitiveValue()
	}
	return value
}
