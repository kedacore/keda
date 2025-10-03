package otto

import (
	"fmt"
)

// RegExp

func builtinRegExp(call FunctionCall) Value {
	pattern := call.Argument(0)
	flags := call.Argument(1)
	if obj := pattern.object(); obj != nil {
		if obj.class == classRegExpName && flags.IsUndefined() {
			return pattern
		}
	}
	return objectValue(call.runtime.newRegExp(pattern, flags))
}

func builtinNewRegExp(obj *object, argumentList []Value) Value {
	return objectValue(obj.runtime.newRegExp(
		valueOfArrayIndex(argumentList, 0),
		valueOfArrayIndex(argumentList, 1),
	))
}

func builtinRegExpToString(call FunctionCall) Value {
	thisObject := call.thisObject()
	source := thisObject.get("source").string()
	flags := []byte{}
	if thisObject.get("global").bool() {
		flags = append(flags, 'g')
	}
	if thisObject.get("ignoreCase").bool() {
		flags = append(flags, 'i')
	}
	if thisObject.get("multiline").bool() {
		flags = append(flags, 'm')
	}
	return stringValue(fmt.Sprintf("/%s/%s", source, flags))
}

func builtinRegExpExec(call FunctionCall) Value {
	thisObject := call.thisObject()
	target := call.Argument(0).string()
	match, result := execRegExp(thisObject, target)
	if !match {
		return nullValue
	}
	return objectValue(execResultToArray(call.runtime, target, result))
}

func builtinRegExpTest(call FunctionCall) Value {
	thisObject := call.thisObject()
	target := call.Argument(0).string()
	match, result := execRegExp(thisObject, target)

	if !match {
		return boolValue(match)
	}

	// Match extract and assign input, $_ and $1 -> $9 on global RegExp.
	input := stringValue(target)
	call.runtime.global.RegExp.defineProperty("$_", input, 0o100, false)
	call.runtime.global.RegExp.defineProperty("input", input, 0o100, false)

	var start int
	n := 1
	re := call.runtime.global.RegExp
	empty := stringValue("")
	for i, v := range result[2:] {
		if i%2 == 0 {
			start = v
		} else {
			if v == -1 {
				// No match for this part.
				re.defineProperty(fmt.Sprintf("$%d", n), empty, 0o100, false)
			} else {
				re.defineProperty(fmt.Sprintf("$%d", n), stringValue(target[start:v]), 0o100, false)
			}
			n++
			if n == 10 {
				break
			}
		}
	}

	if n <= 9 {
		// Erase remaining.
		for i := n; i <= 9; i++ {
			re.defineProperty(fmt.Sprintf("$%d", i), empty, 0o100, false)
		}
	}

	return boolValue(match)
}

func builtinRegExpCompile(call FunctionCall) Value {
	// This (useless) function is deprecated, but is here to provide some
	// semblance of compatibility.
	// Caveat emptor: it may not be around for long.
	return Value{}
}
