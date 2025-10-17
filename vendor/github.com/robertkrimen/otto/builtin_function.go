package otto

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/robertkrimen/otto/parser"
)

// Function

func builtinFunction(call FunctionCall) Value {
	return objectValue(builtinNewFunctionNative(call.runtime, call.ArgumentList))
}

func builtinNewFunction(obj *object, argumentList []Value) Value {
	return objectValue(builtinNewFunctionNative(obj.runtime, argumentList))
}

func argumentList2parameterList(argumentList []Value) []string {
	parameterList := make([]string, 0, len(argumentList))
	for _, value := range argumentList {
		tmp := strings.FieldsFunc(value.string(), func(chr rune) bool {
			return chr == ',' || unicode.IsSpace(chr)
		})
		parameterList = append(parameterList, tmp...)
	}
	return parameterList
}

func builtinNewFunctionNative(rt *runtime, argumentList []Value) *object {
	var parameterList, body string
	if count := len(argumentList); count > 0 {
		tmp := make([]string, 0, count-1)
		for _, value := range argumentList[0 : count-1] {
			tmp = append(tmp, value.string())
		}
		parameterList = strings.Join(tmp, ",")
		body = argumentList[count-1].string()
	}

	// FIXME
	function, err := parser.ParseFunction(parameterList, body)
	rt.parseThrow(err) // Will panic/throw appropriately
	cmpl := compiler{}
	cmplFunction := cmpl.parseExpression(function)

	return rt.newNodeFunction(cmplFunction.(*nodeFunctionLiteral), rt.globalStash)
}

func builtinFunctionToString(call FunctionCall) Value {
	obj := call.thisClassObject(classFunctionName) // Should throw a TypeError unless Function
	switch fn := obj.value.(type) {
	case nativeFunctionObject:
		return stringValue(fmt.Sprintf("function %s() { [native code] }", fn.name))
	case nodeFunctionObject:
		return stringValue(fn.node.source)
	case bindFunctionObject:
		return stringValue("function () { [native code] }")
	default:
		panic(call.runtime.panicTypeError("Function.toString unknown type %T", obj.value))
	}
}

func builtinFunctionApply(call FunctionCall) Value {
	if !call.This.isCallable() {
		panic(call.runtime.panicTypeError("Function.apply %q is not callable", call.This))
	}
	this := call.Argument(0)
	if this.IsUndefined() {
		// FIXME Not ECMA5
		this = objectValue(call.runtime.globalObject)
	}
	argumentList := call.Argument(1)
	switch argumentList.kind {
	case valueUndefined, valueNull:
		return call.thisObject().call(this, nil, false, nativeFrame)
	case valueObject:
	default:
		panic(call.runtime.panicTypeError("Function.apply unknown type %T for second argument"))
	}

	arrayObject := argumentList.object()
	thisObject := call.thisObject()
	length := int64(toUint32(arrayObject.get(propertyLength)))
	valueArray := make([]Value, length)
	for index := range length {
		valueArray[index] = arrayObject.get(arrayIndexToString(index))
	}
	return thisObject.call(this, valueArray, false, nativeFrame)
}

func builtinFunctionCall(call FunctionCall) Value {
	if !call.This.isCallable() {
		panic(call.runtime.panicTypeError("Function.call %q is not callable", call.This))
	}
	thisObject := call.thisObject()
	this := call.Argument(0)
	if this.IsUndefined() {
		// FIXME Not ECMA5
		this = objectValue(call.runtime.globalObject)
	}
	if len(call.ArgumentList) >= 1 {
		return thisObject.call(this, call.ArgumentList[1:], false, nativeFrame)
	}
	return thisObject.call(this, nil, false, nativeFrame)
}

func builtinFunctionBind(call FunctionCall) Value {
	target := call.This
	if !target.isCallable() {
		panic(call.runtime.panicTypeError("Function.bind %q is not callable", call.This))
	}
	targetObject := target.object()

	this := call.Argument(0)
	argumentList := call.slice(1)
	if this.IsUndefined() {
		// FIXME Do this elsewhere?
		this = objectValue(call.runtime.globalObject)
	}

	return objectValue(call.runtime.newBoundFunction(targetObject, this, argumentList))
}
