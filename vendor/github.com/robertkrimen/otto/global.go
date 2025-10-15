package otto

import (
	"strconv"
	"time"
)

var (
	prototypeValueObject   = interface{}(nil)
	prototypeValueFunction = nativeFunctionObject{
		call: func(_ FunctionCall) Value {
			return Value{}
		},
	}
	prototypeValueString = stringASCII("")
	// TODO Make this just false?
	prototypeValueBoolean = Value{
		kind:  valueBoolean,
		value: false,
	}
	prototypeValueNumber = Value{
		kind:  valueNumber,
		value: 0,
	}
	prototypeValueDate = dateObject{
		epoch: 0,
		isNaN: false,
		time:  time.Unix(0, 0).UTC(),
		value: Value{
			kind:  valueNumber,
			value: 0,
		},
	}
	prototypeValueRegExp = regExpObject{
		regularExpression: nil,
		global:            false,
		ignoreCase:        false,
		multiline:         false,
		source:            "",
		flags:             "",
	}
)

func newContext() *runtime {
	rt := &runtime{}

	rt.globalStash = rt.newObjectStash(nil, nil)
	rt.globalObject = rt.globalStash.object

	rt.newContext()

	rt.eval = rt.globalObject.property["eval"].value.(Value).value.(*object)
	rt.globalObject.prototype = rt.global.ObjectPrototype

	return rt
}

func (rt *runtime) newBaseObject() *object {
	return newObject(rt, "")
}

func (rt *runtime) newClassObject(class string) *object {
	return newObject(rt, class)
}

func (rt *runtime) newPrimitiveObject(class string, value Value) *object {
	o := rt.newClassObject(class)
	o.value = value
	return o
}

func (o *object) primitiveValue() Value {
	switch value := o.value.(type) {
	case Value:
		return value
	case stringObjecter:
		return stringValue(value.String())
	}
	return Value{}
}

func (o *object) hasPrimitive() bool { //nolint:unused
	switch o.value.(type) {
	case Value, stringObjecter:
		return true
	}
	return false
}

func (rt *runtime) newObject() *object {
	o := rt.newClassObject(classObjectName)
	o.prototype = rt.global.ObjectPrototype
	return o
}

func (rt *runtime) newArray(length uint32) *object {
	o := rt.newArrayObject(length)
	o.prototype = rt.global.ArrayPrototype
	return o
}

func (rt *runtime) newArrayOf(valueArray []Value) *object {
	o := rt.newArray(uint32(len(valueArray)))
	for index, value := range valueArray {
		if value.isEmpty() {
			continue
		}
		o.defineProperty(strconv.FormatInt(int64(index), 10), value, 0o111, false)
	}
	return o
}

func (rt *runtime) newString(value Value) *object {
	o := rt.newStringObject(value)
	o.prototype = rt.global.StringPrototype
	return o
}

func (rt *runtime) newBoolean(value Value) *object {
	o := rt.newBooleanObject(value)
	o.prototype = rt.global.BooleanPrototype
	return o
}

func (rt *runtime) newNumber(value Value) *object {
	o := rt.newNumberObject(value)
	o.prototype = rt.global.NumberPrototype
	return o
}

func (rt *runtime) newRegExp(patternValue Value, flagsValue Value) *object {
	pattern := ""
	flags := ""
	if obj := patternValue.object(); obj != nil && obj.class == classRegExpName {
		if flagsValue.IsDefined() {
			panic(rt.panicTypeError("Cannot supply flags when constructing one RegExp from another"))
		}
		regExp := obj.regExpValue()
		pattern = regExp.source
		flags = regExp.flags
	} else {
		if patternValue.IsDefined() {
			pattern = patternValue.string()
		}
		if flagsValue.IsDefined() {
			flags = flagsValue.string()
		}
	}

	return rt.newRegExpDirect(pattern, flags)
}

func (rt *runtime) newRegExpDirect(pattern string, flags string) *object {
	o := rt.newRegExpObject(pattern, flags)
	o.prototype = rt.global.RegExpPrototype
	return o
}

// TODO Should (probably) be one argument, right? This is redundant.
func (rt *runtime) newDate(epoch float64) *object {
	o := rt.newDateObject(epoch)
	o.prototype = rt.global.DatePrototype
	return o
}

func (rt *runtime) newError(name string, message Value, stackFramesToPop int) *object {
	switch name {
	case "EvalError":
		return rt.newEvalError(message)
	case "TypeError":
		return rt.newTypeError(message)
	case "RangeError":
		return rt.newRangeError(message)
	case "ReferenceError":
		return rt.newReferenceError(message)
	case "SyntaxError":
		return rt.newSyntaxError(message)
	case "URIError":
		return rt.newURIError(message)
	}

	obj := rt.newErrorObject(name, message, stackFramesToPop)
	obj.prototype = rt.global.ErrorPrototype
	if name != "" {
		obj.defineProperty("name", stringValue(name), 0o111, false)
	}
	return obj
}

func (rt *runtime) newNativeFunction(name, file string, line int, fn nativeFunction) *object {
	o := rt.newNativeFunctionObject(name, file, line, fn, 0)
	o.prototype = rt.global.FunctionPrototype
	prototype := rt.newObject()
	o.defineProperty("prototype", objectValue(prototype), 0o100, false)
	prototype.defineProperty("constructor", objectValue(o), 0o100, false)
	return o
}

func (rt *runtime) newNodeFunction(node *nodeFunctionLiteral, scopeEnvironment stasher) *object {
	// TODO Implement 13.2 fully
	o := rt.newNodeFunctionObject(node, scopeEnvironment)
	o.prototype = rt.global.FunctionPrototype
	prototype := rt.newObject()
	o.defineProperty("prototype", objectValue(prototype), 0o100, false)
	prototype.defineProperty("constructor", objectValue(o), 0o101, false)
	return o
}

// FIXME Only in one place...
func (rt *runtime) newBoundFunction(target *object, this Value, argumentList []Value) *object {
	o := rt.newBoundFunctionObject(target, this, argumentList)
	o.prototype = rt.global.FunctionPrototype
	prototype := rt.newObject()
	o.defineProperty("prototype", objectValue(prototype), 0o100, false)
	prototype.defineProperty("constructor", objectValue(o), 0o100, false)
	return o
}
