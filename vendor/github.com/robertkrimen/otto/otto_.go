package otto

import (
	"fmt"
	"regexp"
	goruntime "runtime"
	"strconv"
)

var isIdentifierRegexp *regexp.Regexp = regexp.MustCompile(`^[a-zA-Z\$][a-zA-Z0-9\$]*$`)

func isIdentifier(value string) bool {
	return isIdentifierRegexp.MatchString(value)
}

func (rt *runtime) toValueArray(arguments ...interface{}) []Value {
	length := len(arguments)
	if length == 1 {
		if valueArray, ok := arguments[0].([]Value); ok {
			return valueArray
		}
		return []Value{rt.toValue(arguments[0])}
	}

	valueArray := make([]Value, length)
	for index, value := range arguments {
		valueArray[index] = rt.toValue(value)
	}

	return valueArray
}

func stringToArrayIndex(name string) int64 {
	index, err := strconv.ParseInt(name, 10, 64)
	if err != nil {
		return -1
	}
	if index < 0 {
		return -1
	}
	if index >= maxUint32 {
		// The value 2^32 (or above) is not a valid index because
		// you cannot store a uint32 length for an index of uint32
		return -1
	}
	return index
}

func isUint32(value int64) bool {
	return value >= 0 && value <= maxUint32
}

func arrayIndexToString(index int64) string {
	return strconv.FormatInt(index, 10)
}

func valueOfArrayIndex(array []Value, index int) Value {
	value, _ := getValueOfArrayIndex(array, index)
	return value
}

func getValueOfArrayIndex(array []Value, index int) (Value, bool) {
	if index >= 0 && index < len(array) {
		value := array[index]
		if !value.isEmpty() {
			return value, true
		}
	}
	return Value{}, false
}

// A range index can be anything from 0 up to length. It is NOT safe to use as an index
// to an array, but is useful for slicing and in some ECMA algorithms.
func valueToRangeIndex(indexValue Value, length int64, negativeIsZero bool) int64 {
	index := indexValue.number().int64
	if negativeIsZero {
		if index < 0 {
			index = 0
		}
		// minimum(index, length)
		if index >= length {
			index = length
		}
		return index
	}

	if index < 0 {
		index += length
		if index < 0 {
			index = 0
		}
	} else if index > length {
		index = length
	}
	return index
}

func rangeStartEnd(array []Value, size int64, negativeIsZero bool) (start, end int64) { //nolint:nonamedreturns
	start = valueToRangeIndex(valueOfArrayIndex(array, 0), size, negativeIsZero)
	if len(array) == 1 {
		// If there is only the start argument, then end = size
		end = size
		return
	}

	// Assuming the argument is undefined...
	end = size
	endValue := valueOfArrayIndex(array, 1)
	if !endValue.IsUndefined() {
		// Which it is not, so get the value as an array index
		end = valueToRangeIndex(endValue, size, negativeIsZero)
	}
	return
}

func rangeStartLength(source []Value, size int64) (start, length int64) { //nolint:nonamedreturns
	start = valueToRangeIndex(valueOfArrayIndex(source, 0), size, false)

	// Assume the second argument is missing or undefined
	length = size
	if len(source) == 1 {
		// If there is only the start argument, then length = size
		return start, length
	}

	lengthValue := valueOfArrayIndex(source, 1)
	if !lengthValue.IsUndefined() {
		// Which it is not, so get the value as an array index
		length = lengthValue.number().int64
	}
	return start, length
}

func hereBeDragons(arguments ...interface{}) string {
	pc, _, _, _ := goruntime.Caller(1) //nolint:dogsled
	name := goruntime.FuncForPC(pc).Name()
	message := "Here be dragons -- " + name
	if len(arguments) > 0 {
		message += ": "
		argument0 := fmt.Sprintf("%s", arguments[0])
		if len(arguments) == 1 {
			message += argument0
		} else {
			message += fmt.Sprintf(argument0, arguments[1:]...)
		}
	} else {
		message += "."
	}
	return message
}
