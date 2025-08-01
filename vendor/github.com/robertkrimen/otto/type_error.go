package otto

func (rt *runtime) newErrorObject(name string, message Value, stackFramesToPop int) *object {
	obj := rt.newClassObject(classErrorName)
	if message.IsDefined() {
		err := newError(rt, name, stackFramesToPop, "%s", message.string())
		obj.defineProperty("message", err.messageValue(), 0o111, false)
		obj.value = err
	} else {
		obj.value = newError(rt, name, stackFramesToPop)
	}

	obj.defineOwnProperty("stack", property{
		value: propertyGetSet{
			rt.newNativeFunction("get", "internal", 0, func(FunctionCall) Value {
				return stringValue(obj.value.(ottoError).formatWithStack())
			}),
			&nilGetSetObject,
		},
		mode: modeConfigureMask & modeOnMask,
	}, false)

	return obj
}

func (rt *runtime) newErrorObjectError(err ottoError) *object {
	obj := rt.newClassObject(classErrorName)
	obj.defineProperty("message", err.messageValue(), 0o111, false)
	obj.value = err
	switch err.name {
	case "EvalError":
		obj.prototype = rt.global.EvalErrorPrototype
	case "TypeError":
		obj.prototype = rt.global.TypeErrorPrototype
	case "RangeError":
		obj.prototype = rt.global.RangeErrorPrototype
	case "ReferenceError":
		obj.prototype = rt.global.ReferenceErrorPrototype
	case "SyntaxError":
		obj.prototype = rt.global.SyntaxErrorPrototype
	case "URIError":
		obj.prototype = rt.global.URIErrorPrototype
	default:
		obj.prototype = rt.global.ErrorPrototype
	}

	obj.defineOwnProperty("stack", property{
		value: propertyGetSet{
			rt.newNativeFunction("get", "internal", 0, func(FunctionCall) Value {
				return stringValue(obj.value.(ottoError).formatWithStack())
			}),
			&nilGetSetObject,
		},
		mode: modeConfigureMask & modeOnMask,
	}, false)

	return obj
}
