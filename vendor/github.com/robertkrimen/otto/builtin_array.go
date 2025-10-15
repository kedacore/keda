package otto

import (
	"strconv"
	"strings"
)

// Array

func builtinArray(call FunctionCall) Value {
	return objectValue(builtinNewArrayNative(call.runtime, call.ArgumentList))
}

func builtinNewArray(obj *object, argumentList []Value) Value {
	return objectValue(builtinNewArrayNative(obj.runtime, argumentList))
}

func builtinNewArrayNative(rt *runtime, argumentList []Value) *object {
	if len(argumentList) == 1 {
		firstArgument := argumentList[0]
		if firstArgument.IsNumber() {
			return rt.newArray(arrayUint32(rt, firstArgument))
		}
	}
	return rt.newArrayOf(argumentList)
}

func builtinArrayToString(call FunctionCall) Value {
	thisObject := call.thisObject()
	join := thisObject.get("join")
	if join.isCallable() {
		join := join.object()
		return join.call(call.This, call.ArgumentList, false, nativeFrame)
	}
	return builtinObjectToString(call)
}

func builtinArrayToLocaleString(call FunctionCall) Value {
	separator := ","
	thisObject := call.thisObject()
	length := int64(toUint32(thisObject.get(propertyLength)))
	if length == 0 {
		return stringValue("")
	}
	stringList := make([]string, 0, length)
	for index := range length {
		value := thisObject.get(arrayIndexToString(index))
		stringValue := ""
		switch value.kind {
		case valueEmpty, valueUndefined, valueNull:
		default:
			obj := call.runtime.toObject(value)
			toLocaleString := obj.get("toLocaleString")
			if !toLocaleString.isCallable() {
				panic(call.runtime.panicTypeError("Array.toLocaleString index[%d] %q is not callable", index, toLocaleString))
			}
			stringValue = toLocaleString.call(call.runtime, objectValue(obj)).string()
		}
		stringList = append(stringList, stringValue)
	}
	return stringValue(strings.Join(stringList, separator))
}

func builtinArrayConcat(call FunctionCall) Value {
	thisObject := call.thisObject()
	valueArray := []Value{}
	source := append([]Value{objectValue(thisObject)}, call.ArgumentList...)
	for _, item := range source {
		switch item.kind {
		case valueObject:
			obj := item.object()
			if isArray(obj) {
				length := obj.get(propertyLength).number().int64
				for index := range length {
					name := strconv.FormatInt(index, 10)
					if obj.hasProperty(name) {
						valueArray = append(valueArray, obj.get(name))
					} else {
						valueArray = append(valueArray, Value{})
					}
				}
				continue
			}

			fallthrough
		default:
			valueArray = append(valueArray, item)
		}
	}
	return objectValue(call.runtime.newArrayOf(valueArray))
}

func builtinArrayShift(call FunctionCall) Value {
	thisObject := call.thisObject()
	length := int64(toUint32(thisObject.get(propertyLength)))
	if length == 0 {
		thisObject.put(propertyLength, int64Value(0), true)
		return Value{}
	}
	first := thisObject.get("0")
	for index := int64(1); index < length; index++ {
		from := arrayIndexToString(index)
		to := arrayIndexToString(index - 1)
		if thisObject.hasProperty(from) {
			thisObject.put(to, thisObject.get(from), true)
		} else {
			thisObject.delete(to, true)
		}
	}
	thisObject.delete(arrayIndexToString(length-1), true)
	thisObject.put(propertyLength, int64Value(length-1), true)
	return first
}

func builtinArrayPush(call FunctionCall) Value {
	thisObject := call.thisObject()
	itemList := call.ArgumentList
	index := int64(toUint32(thisObject.get(propertyLength)))
	for len(itemList) > 0 {
		thisObject.put(arrayIndexToString(index), itemList[0], true)
		itemList = itemList[1:]
		index++
	}
	length := int64Value(index)
	thisObject.put(propertyLength, length, true)
	return length
}

func builtinArrayPop(call FunctionCall) Value {
	thisObject := call.thisObject()
	length := int64(toUint32(thisObject.get(propertyLength)))
	if length == 0 {
		thisObject.put(propertyLength, uint32Value(0), true)
		return Value{}
	}
	last := thisObject.get(arrayIndexToString(length - 1))
	thisObject.delete(arrayIndexToString(length-1), true)
	thisObject.put(propertyLength, int64Value(length-1), true)
	return last
}

func builtinArrayJoin(call FunctionCall) Value {
	separator := ","
	argument := call.Argument(0)
	if argument.IsDefined() {
		separator = argument.string()
	}
	thisObject := call.thisObject()
	length := int64(toUint32(thisObject.get(propertyLength)))
	if length == 0 {
		return stringValue("")
	}
	stringList := make([]string, 0, length)
	for index := range length {
		value := thisObject.get(arrayIndexToString(index))
		stringValue := ""
		switch value.kind {
		case valueEmpty, valueUndefined, valueNull:
		default:
			stringValue = value.string()
		}
		stringList = append(stringList, stringValue)
	}
	return stringValue(strings.Join(stringList, separator))
}

func builtinArraySplice(call FunctionCall) Value {
	thisObject := call.thisObject()
	length := int64(toUint32(thisObject.get(propertyLength)))

	start := valueToRangeIndex(call.Argument(0), length, false)
	deleteCount := length - start
	if arg, ok := call.getArgument(1); ok {
		deleteCount = valueToRangeIndex(arg, length-start, true)
	}
	valueArray := make([]Value, deleteCount)

	for index := range deleteCount {
		indexString := arrayIndexToString(start + index)
		if thisObject.hasProperty(indexString) {
			valueArray[index] = thisObject.get(indexString)
		}
	}

	// 0, <1, 2, 3, 4>, 5, 6, 7
	// a, b
	// length 8 - delete 4 @ start 1

	itemList := []Value{}
	itemCount := int64(len(call.ArgumentList))
	if itemCount > 2 {
		itemCount -= 2 // Less the first two arguments
		itemList = call.ArgumentList[2:]
	} else {
		itemCount = 0
	}
	if itemCount < deleteCount {
		// The Object/Array is shrinking
		stop := length - deleteCount
		// The new length of the Object/Array before
		// appending the itemList remainder
		// Stopping at the lower bound of the insertion:
		// Move an item from the after the deleted portion
		// to a position after the inserted portion
		for index := start; index < stop; index++ {
			from := arrayIndexToString(index + deleteCount) // Position just after deletion
			to := arrayIndexToString(index + itemCount)     // Position just after splice (insertion)
			if thisObject.hasProperty(from) {
				thisObject.put(to, thisObject.get(from), true)
			} else {
				thisObject.delete(to, true)
			}
		}
		// Delete off the end
		// We don't bother to delete below <stop + itemCount> (if any) since those
		// will be overwritten anyway
		for index := length; index > (stop + itemCount); index-- {
			thisObject.delete(arrayIndexToString(index-1), true)
		}
	} else if itemCount > deleteCount {
		// The Object/Array is growing
		// The itemCount is greater than the deleteCount, so we do
		// not have to worry about overwriting what we should be moving
		// ---
		// Starting from the upper bound of the deletion:
		// Move an item from the after the deleted portion
		// to a position after the inserted portion
		for index := length - deleteCount; index > start; index-- {
			from := arrayIndexToString(index + deleteCount - 1)
			to := arrayIndexToString(index + itemCount - 1)
			if thisObject.hasProperty(from) {
				thisObject.put(to, thisObject.get(from), true)
			} else {
				thisObject.delete(to, true)
			}
		}
	}

	for index := range itemCount {
		thisObject.put(arrayIndexToString(index+start), itemList[index], true)
	}
	thisObject.put(propertyLength, int64Value(length+itemCount-deleteCount), true)

	return objectValue(call.runtime.newArrayOf(valueArray))
}

func builtinArraySlice(call FunctionCall) Value {
	thisObject := call.thisObject()

	length := int64(toUint32(thisObject.get(propertyLength)))
	start, end := rangeStartEnd(call.ArgumentList, length, false)

	if start >= end {
		// Always an empty array
		return objectValue(call.runtime.newArray(0))
	}
	sliceLength := end - start
	sliceValueArray := make([]Value, sliceLength)

	for index := range sliceLength {
		from := arrayIndexToString(index + start)
		if thisObject.hasProperty(from) {
			sliceValueArray[index] = thisObject.get(from)
		}
	}

	return objectValue(call.runtime.newArrayOf(sliceValueArray))
}

func builtinArrayUnshift(call FunctionCall) Value {
	thisObject := call.thisObject()
	length := int64(toUint32(thisObject.get(propertyLength)))
	itemList := call.ArgumentList
	itemCount := int64(len(itemList))

	for index := length; index > 0; index-- {
		from := arrayIndexToString(index - 1)
		to := arrayIndexToString(index + itemCount - 1)
		if thisObject.hasProperty(from) {
			thisObject.put(to, thisObject.get(from), true)
		} else {
			thisObject.delete(to, true)
		}
	}

	for index := range itemCount {
		thisObject.put(arrayIndexToString(index), itemList[index], true)
	}

	newLength := int64Value(length + itemCount)
	thisObject.put(propertyLength, newLength, true)
	return newLength
}

func builtinArrayReverse(call FunctionCall) Value {
	thisObject := call.thisObject()
	length := int64(toUint32(thisObject.get(propertyLength)))

	lower := struct {
		name   string
		index  int64
		exists bool
	}{}
	upper := lower

	lower.index = 0
	middle := length / 2 // Division will floor

	for lower.index != middle {
		lower.name = arrayIndexToString(lower.index)
		upper.index = length - lower.index - 1
		upper.name = arrayIndexToString(upper.index)

		lower.exists = thisObject.hasProperty(lower.name)
		upper.exists = thisObject.hasProperty(upper.name)

		switch {
		case lower.exists && upper.exists:
			lowerValue := thisObject.get(lower.name)
			upperValue := thisObject.get(upper.name)
			thisObject.put(lower.name, upperValue, true)
			thisObject.put(upper.name, lowerValue, true)
		case !lower.exists && upper.exists:
			value := thisObject.get(upper.name)
			thisObject.delete(upper.name, true)
			thisObject.put(lower.name, value, true)
		case lower.exists && !upper.exists:
			value := thisObject.get(lower.name)
			thisObject.delete(lower.name, true)
			thisObject.put(upper.name, value, true)
		}

		lower.index++
	}

	return call.This
}

func sortCompare(thisObject *object, index0, index1 uint, compare *object) int {
	j := struct {
		name    string
		value   string
		exists  bool
		defined bool
	}{}
	k := j
	j.name = arrayIndexToString(int64(index0))
	j.exists = thisObject.hasProperty(j.name)
	k.name = arrayIndexToString(int64(index1))
	k.exists = thisObject.hasProperty(k.name)

	switch {
	case !j.exists && !k.exists:
		return 0
	case !j.exists:
		return 1
	case !k.exists:
		return -1
	}

	x := thisObject.get(j.name)
	y := thisObject.get(k.name)
	j.defined = x.IsDefined()
	k.defined = y.IsDefined()

	switch {
	case !j.defined && !k.defined:
		return 0
	case !j.defined:
		return 1
	case !k.defined:
		return -1
	}

	if compare == nil {
		j.value = x.string()
		k.value = y.string()

		if j.value == k.value {
			return 0
		} else if j.value < k.value {
			return -1
		}

		return 1
	}

	return toIntSign(compare.call(Value{}, []Value{x, y}, false, nativeFrame))
}

func arraySortSwap(thisObject *object, index0, index1 uint) {
	j := struct {
		name   string
		exists bool
	}{}
	k := j

	j.name = arrayIndexToString(int64(index0))
	j.exists = thisObject.hasProperty(j.name)
	k.name = arrayIndexToString(int64(index1))
	k.exists = thisObject.hasProperty(k.name)

	switch {
	case j.exists && k.exists:
		jv := thisObject.get(j.name)
		kv := thisObject.get(k.name)
		thisObject.put(j.name, kv, true)
		thisObject.put(k.name, jv, true)
	case !j.exists && k.exists:
		value := thisObject.get(k.name)
		thisObject.delete(k.name, true)
		thisObject.put(j.name, value, true)
	case j.exists && !k.exists:
		value := thisObject.get(j.name)
		thisObject.delete(j.name, true)
		thisObject.put(k.name, value, true)
	}
}

func arraySortQuickPartition(thisObject *object, left, right, pivot uint, compare *object) (uint, uint) {
	arraySortSwap(thisObject, pivot, right) // Right is now the pivot value
	cursor := left
	cursor2 := left
	for index := left; index < right; index++ {
		comparison := sortCompare(thisObject, index, right, compare) // Compare to the pivot value
		if comparison < 0 {
			arraySortSwap(thisObject, index, cursor)
			if cursor < cursor2 {
				arraySortSwap(thisObject, index, cursor2)
			}
			cursor++
			cursor2++
		} else if comparison == 0 {
			arraySortSwap(thisObject, index, cursor2)
			cursor2++
		}
	}
	arraySortSwap(thisObject, cursor2, right)
	return cursor, cursor2
}

func arraySortQuickSort(thisObject *object, left, right uint, compare *object) {
	if left < right {
		middle := left + (right-left)/2
		pivot, pivot2 := arraySortQuickPartition(thisObject, left, right, middle, compare)
		if pivot > 0 {
			arraySortQuickSort(thisObject, left, pivot-1, compare)
		}
		arraySortQuickSort(thisObject, pivot2+1, right, compare)
	}
}

func builtinArraySort(call FunctionCall) Value {
	thisObject := call.thisObject()
	length := uint(toUint32(thisObject.get(propertyLength)))
	compareValue := call.Argument(0)
	compare := compareValue.object()
	if compareValue.IsUndefined() {
	} else if !compareValue.isCallable() {
		panic(call.runtime.panicTypeError("Array.sort value %q is not callable", compareValue))
	}
	if length > 1 {
		arraySortQuickSort(thisObject, 0, length-1, compare)
	}
	return call.This
}

func builtinArrayIsArray(call FunctionCall) Value {
	return boolValue(isArray(call.Argument(0).object()))
}

func builtinArrayIndexOf(call FunctionCall) Value {
	thisObject, matchValue := call.thisObject(), call.Argument(0)
	if length := int64(toUint32(thisObject.get(propertyLength))); length > 0 {
		index := int64(0)
		if len(call.ArgumentList) > 1 {
			index = call.Argument(1).number().int64
		}
		if index < 0 {
			if index += length; index < 0 {
				index = 0
			}
		} else if index >= length {
			index = -1
		}
		for ; index >= 0 && index < length; index++ {
			name := arrayIndexToString(index)
			if !thisObject.hasProperty(name) {
				continue
			}
			value := thisObject.get(name)
			if strictEqualityComparison(matchValue, value) {
				return uint32Value(uint32(index))
			}
		}
	}
	return intValue(-1)
}

func builtinArrayLastIndexOf(call FunctionCall) Value {
	thisObject, matchValue := call.thisObject(), call.Argument(0)
	length := int64(toUint32(thisObject.get(propertyLength)))
	index := length - 1
	if len(call.ArgumentList) > 1 {
		index = call.Argument(1).number().int64
	}
	if 0 > index {
		index += length
	}
	if index > length {
		index = length - 1
	} else if 0 > index {
		return intValue(-1)
	}
	for ; index >= 0; index-- {
		name := arrayIndexToString(index)
		if !thisObject.hasProperty(name) {
			continue
		}
		value := thisObject.get(name)
		if strictEqualityComparison(matchValue, value) {
			return uint32Value(uint32(index))
		}
	}
	return intValue(-1)
}

func builtinArrayEvery(call FunctionCall) Value {
	thisObject := call.thisObject()
	this := objectValue(thisObject)
	if iterator := call.Argument(0); iterator.isCallable() {
		length := int64(toUint32(thisObject.get(propertyLength)))
		callThis := call.Argument(1)
		for index := range length {
			if key := arrayIndexToString(index); thisObject.hasProperty(key) {
				if value := thisObject.get(key); iterator.call(call.runtime, callThis, value, int64Value(index), this).bool() {
					continue
				}
				return falseValue
			}
		}
		return trueValue
	}
	panic(call.runtime.panicTypeError("Array.every argument %q is not callable", call.Argument(0)))
}

func builtinArraySome(call FunctionCall) Value {
	thisObject := call.thisObject()
	this := objectValue(thisObject)
	if iterator := call.Argument(0); iterator.isCallable() {
		length := int64(toUint32(thisObject.get(propertyLength)))
		callThis := call.Argument(1)
		for index := range length {
			if key := arrayIndexToString(index); thisObject.hasProperty(key) {
				if value := thisObject.get(key); iterator.call(call.runtime, callThis, value, int64Value(index), this).bool() {
					return trueValue
				}
			}
		}
		return falseValue
	}
	panic(call.runtime.panicTypeError("Array.some %q if not callable", call.Argument(0)))
}

func builtinArrayForEach(call FunctionCall) Value {
	thisObject := call.thisObject()
	this := objectValue(thisObject)
	if iterator := call.Argument(0); iterator.isCallable() {
		length := int64(toUint32(thisObject.get(propertyLength)))
		callThis := call.Argument(1)
		for index := range length {
			if key := arrayIndexToString(index); thisObject.hasProperty(key) {
				iterator.call(call.runtime, callThis, thisObject.get(key), int64Value(index), this)
			}
		}
		return Value{}
	}
	panic(call.runtime.panicTypeError("Array.foreach %q if not callable", call.Argument(0)))
}

func builtinArrayMap(call FunctionCall) Value {
	thisObject := call.thisObject()
	this := objectValue(thisObject)
	if iterator := call.Argument(0); iterator.isCallable() {
		length := int64(toUint32(thisObject.get(propertyLength)))
		callThis := call.Argument(1)
		values := make([]Value, length)
		for index := range length {
			if key := arrayIndexToString(index); thisObject.hasProperty(key) {
				values[index] = iterator.call(call.runtime, callThis, thisObject.get(key), index, this)
			} else {
				values[index] = Value{}
			}
		}
		return objectValue(call.runtime.newArrayOf(values))
	}
	panic(call.runtime.panicTypeError("Array.foreach %q if not callable", call.Argument(0)))
}

func builtinArrayFilter(call FunctionCall) Value {
	thisObject := call.thisObject()
	this := objectValue(thisObject)
	if iterator := call.Argument(0); iterator.isCallable() {
		length := int64(toUint32(thisObject.get(propertyLength)))
		callThis := call.Argument(1)
		values := make([]Value, 0)
		for index := range length {
			if key := arrayIndexToString(index); thisObject.hasProperty(key) {
				value := thisObject.get(key)
				if iterator.call(call.runtime, callThis, value, index, this).bool() {
					values = append(values, value)
				}
			}
		}
		return objectValue(call.runtime.newArrayOf(values))
	}
	panic(call.runtime.panicTypeError("Array.filter %q if not callable", call.Argument(0)))
}

func builtinArrayReduce(call FunctionCall) Value {
	thisObject := call.thisObject()
	this := objectValue(thisObject)
	if iterator := call.Argument(0); iterator.isCallable() {
		initial := len(call.ArgumentList) > 1
		start := call.Argument(1)
		length := int64(toUint32(thisObject.get(propertyLength)))
		index := int64(0)
		if length > 0 || initial {
			var accumulator Value
			if !initial {
				for ; index < length; index++ {
					if key := arrayIndexToString(index); thisObject.hasProperty(key) {
						accumulator = thisObject.get(key)
						index++

						break
					}
				}
			} else {
				accumulator = start
			}
			for ; index < length; index++ {
				if key := arrayIndexToString(index); thisObject.hasProperty(key) {
					accumulator = iterator.call(call.runtime, Value{}, accumulator, thisObject.get(key), index, this)
				}
			}
			return accumulator
		}
	}
	panic(call.runtime.panicTypeError("Array.reduce %q if not callable", call.Argument(0)))
}

func builtinArrayReduceRight(call FunctionCall) Value {
	thisObject := call.thisObject()
	this := objectValue(thisObject)
	if iterator := call.Argument(0); iterator.isCallable() {
		initial := len(call.ArgumentList) > 1
		start := call.Argument(1)
		length := int64(toUint32(thisObject.get(propertyLength)))
		if length > 0 || initial {
			index := length - 1
			var accumulator Value
			if !initial {
				for ; index >= 0; index-- {
					if key := arrayIndexToString(index); thisObject.hasProperty(key) {
						accumulator = thisObject.get(key)
						index--
						break
					}
				}
			} else {
				accumulator = start
			}
			for ; index >= 0; index-- {
				if key := arrayIndexToString(index); thisObject.hasProperty(key) {
					accumulator = iterator.call(call.runtime, Value{}, accumulator, thisObject.get(key), key, this)
				}
			}
			return accumulator
		}
	}
	panic(call.runtime.panicTypeError("Array.reduceRight %q if not callable", call.Argument(0)))
}
