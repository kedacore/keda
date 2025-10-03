package otto

import (
	"fmt"
	"regexp"

	"github.com/robertkrimen/otto/parser"
)

type regExpObject struct {
	regularExpression *regexp.Regexp
	source            string
	flags             string
	global            bool
	ignoreCase        bool
	multiline         bool
}

func (rt *runtime) newRegExpObject(pattern string, flags string) *object {
	o := rt.newObject()
	o.class = classRegExpName

	global := false
	ignoreCase := false
	multiline := false
	re2flags := ""

	// TODO Maybe clean up the panicking here... TypeError, SyntaxError, ?

	for _, chr := range flags {
		switch chr {
		case 'g':
			if global {
				panic(rt.panicSyntaxError("newRegExpObject: %s %s", pattern, flags))
			}
			global = true
		case 'm':
			if multiline {
				panic(rt.panicSyntaxError("newRegExpObject: %s %s", pattern, flags))
			}
			multiline = true
			re2flags += "m"
		case 'i':
			if ignoreCase {
				panic(rt.panicSyntaxError("newRegExpObject: %s %s", pattern, flags))
			}
			ignoreCase = true
			re2flags += "i"
		}
	}

	re2pattern, err := parser.TransformRegExp(pattern)
	if err != nil {
		panic(rt.panicTypeError("Invalid regular expression: %s", err.Error()))
	}
	if len(re2flags) > 0 {
		re2pattern = fmt.Sprintf("(?%s:%s)", re2flags, re2pattern)
	}

	regularExpression, err := regexp.Compile(re2pattern)
	if err != nil {
		panic(rt.panicSyntaxError("Invalid regular expression: %s", err.Error()[22:]))
	}

	o.value = regExpObject{
		regularExpression: regularExpression,
		global:            global,
		ignoreCase:        ignoreCase,
		multiline:         multiline,
		source:            pattern,
		flags:             flags,
	}
	o.defineProperty("global", boolValue(global), 0, false)
	o.defineProperty("ignoreCase", boolValue(ignoreCase), 0, false)
	o.defineProperty("multiline", boolValue(multiline), 0, false)
	o.defineProperty("lastIndex", intValue(0), 0o100, false)
	o.defineProperty("source", stringValue(pattern), 0, false)
	return o
}

func (o *object) regExpValue() regExpObject {
	value, _ := o.value.(regExpObject)
	return value
}

func execRegExp(this *object, target string) (bool, []int) {
	if this.class != classRegExpName {
		panic(this.runtime.panicTypeError("Calling RegExp.exec on a non-RegExp object"))
	}
	lastIndex := this.get("lastIndex").number().int64
	index := lastIndex
	global := this.get("global").bool()
	if !global {
		index = 0
	}

	var result []int
	if 0 > index || index > int64(len(target)) {
	} else {
		result = this.regExpValue().regularExpression.FindStringSubmatchIndex(target[index:])
	}

	if result == nil {
		this.put("lastIndex", intValue(0), true)
		return false, nil
	}

	startIndex := index
	endIndex := int(lastIndex) + result[1]
	// We do this shift here because the .FindStringSubmatchIndex above
	// was done on a local subordinate slice of the string, not the whole string
	for index, offset := range result {
		if offset != -1 {
			result[index] += int(startIndex)
		}
	}
	if global {
		this.put("lastIndex", intValue(endIndex), true)
	}

	return true, result
}

func execResultToArray(rt *runtime, target string, result []int) *object {
	captureCount := len(result) / 2
	valueArray := make([]Value, captureCount)
	for index := range captureCount {
		offset := 2 * index
		if result[offset] != -1 {
			valueArray[index] = stringValue(target[result[offset]:result[offset+1]])
		} else {
			valueArray[index] = Value{}
		}
	}
	matchIndex := result[0]
	if matchIndex != 0 {
		// Find the utf16 index in the string, not the byte index.
		matchIndex = utf16Length(target[:matchIndex])
	}
	match := rt.newArrayOf(valueArray)
	match.defineProperty("input", stringValue(target), 0o111, false)
	match.defineProperty("index", intValue(matchIndex), 0o111, false)
	return match
}
