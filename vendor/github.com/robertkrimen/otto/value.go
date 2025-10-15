package otto

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"unicode/utf16"
)

type valueKind int

const (
	valueUndefined valueKind = iota
	valueNull
	valueNumber
	valueString
	valueBoolean
	valueObject

	// These are invalid outside of the runtime.
	valueEmpty
	valueResult
	valueReference
)

// Value is the representation of a JavaScript value.
type Value struct {
	value interface{}
	kind  valueKind
}

func (v Value) safe() bool {
	return v.kind < valueEmpty
}

var (
	emptyValue = Value{kind: valueEmpty}
	nullValue  = Value{kind: valueNull}
	falseValue = Value{kind: valueBoolean, value: false}
	trueValue  = Value{kind: valueBoolean, value: true}
)

// ToValue will convert an interface{} value to a value digestible by otto/JavaScript
//
// This function will not work for advanced types (struct, map, slice/array, etc.) and
// you should use Otto.ToValue instead.
func ToValue(value interface{}) (Value, error) {
	result := Value{}
	err := catchPanic(func() {
		result = toValue(value)
	})
	return result, err
}

func (v Value) isEmpty() bool {
	return v.kind == valueEmpty
}

// Undefined

// UndefinedValue will return a Value representing undefined.
func UndefinedValue() Value {
	return Value{}
}

// IsDefined will return false if the value is undefined, and true otherwise.
func (v Value) IsDefined() bool {
	return v.kind != valueUndefined
}

// IsUndefined will return true if the value is undefined, and false otherwise.
func (v Value) IsUndefined() bool {
	return v.kind == valueUndefined
}

// NullValue will return a Value representing null.
func NullValue() Value {
	return Value{kind: valueNull}
}

// IsNull will return true if the value is null, and false otherwise.
func (v Value) IsNull() bool {
	return v.kind == valueNull
}

// ---

func (v Value) isCallable() bool {
	o, ok := v.value.(*object)
	return ok && o.isCall()
}

// Call the value as a function with the given this value and argument list and
// return the result of invocation. It is essentially equivalent to:
//
//	value.apply(thisValue, argumentList)
//
// An undefined value and an error will result if:
//
//  1. There is an error during conversion of the argument list
//  2. The value is not actually a function
//  3. An (uncaught) exception is thrown
func (v Value) Call(this Value, argumentList ...interface{}) (Value, error) {
	result := Value{}
	err := catchPanic(func() {
		// FIXME
		result = v.call(nil, this, argumentList...)
	})
	if !v.safe() {
		v = Value{}
	}
	return result, err
}

func (v Value) call(rt *runtime, this Value, argumentList ...interface{}) Value {
	if function, ok := v.value.(*object); ok {
		return function.call(this, function.runtime.toValueArray(argumentList...), false, nativeFrame)
	}
	panic(rt.panicTypeError("call %q is not an object", v.value))
}

func (v Value) constructSafe(rt *runtime, this Value, argumentList ...interface{}) (Value, error) {
	result := Value{}
	err := catchPanic(func() {
		result = v.construct(rt, this, argumentList...)
	})
	return result, err
}

func (v Value) construct(rt *runtime, this Value, argumentList ...interface{}) Value { //nolint:unparam
	if fn, ok := v.value.(*object); ok {
		return fn.construct(fn.runtime.toValueArray(argumentList...))
	}
	panic(rt.panicTypeError("construct %q is not an object", v.value))
}

// IsPrimitive will return true if value is a primitive (any kind of primitive).
func (v Value) IsPrimitive() bool {
	return !v.IsObject()
}

// IsBoolean will return true if value is a boolean (primitive).
func (v Value) IsBoolean() bool {
	return v.kind == valueBoolean
}

// IsNumber will return true if value is a number (primitive).
func (v Value) IsNumber() bool {
	return v.kind == valueNumber
}

// IsNaN will return true if value is NaN (or would convert to NaN).
func (v Value) IsNaN() bool {
	switch value := v.value.(type) {
	case float64:
		return math.IsNaN(value)
	case float32:
		return math.IsNaN(float64(value))
	case int, int8, int32, int64:
		return false
	case uint, uint8, uint32, uint64:
		return false
	}

	return math.IsNaN(v.float64())
}

// IsString will return true if value is a string (primitive).
func (v Value) IsString() bool {
	return v.kind == valueString
}

// IsObject will return true if value is an object.
func (v Value) IsObject() bool {
	return v.kind == valueObject
}

// IsFunction will return true if value is a function.
func (v Value) IsFunction() bool {
	if v.kind != valueObject {
		return false
	}
	return v.value.(*object).class == classFunctionName
}

// Class will return the class string of the value or the empty string if value is not an object.
//
// The return value will (generally) be one of:
//
//	Object
//	Function
//	Array
//	String
//	Number
//	Boolean
//	Date
//	RegExp
func (v Value) Class() string {
	if v.kind != valueObject {
		return ""
	}
	return v.value.(*object).class
}

func (v Value) isArray() bool { //nolint:unused
	if v.kind != valueObject {
		return false
	}
	return isArray(v.value.(*object))
}

func (v Value) isStringObject() bool { //nolint:unused
	if v.kind != valueObject {
		return false
	}
	return v.value.(*object).class == classStringName
}

func (v Value) isBooleanObject() bool { //nolint:unused
	if v.kind != valueObject {
		return false
	}
	return v.value.(*object).class == classBooleanName
}

func (v Value) isNumberObject() bool { //nolint:unused
	if v.kind != valueObject {
		return false
	}
	return v.value.(*object).class == classNumberName
}

func (v Value) isDate() bool { //nolint:unused
	if v.kind != valueObject {
		return false
	}
	return v.value.(*object).class == classDateName
}

func (v Value) isRegExp() bool {
	if v.kind != valueObject {
		return false
	}
	return v.value.(*object).class == classRegExpName
}

func (v Value) isError() bool { //nolint:unused
	if v.kind != valueObject {
		return false
	}
	return v.value.(*object).class == classErrorName
}

// ---

func reflectValuePanic(value interface{}, kind reflect.Kind) {
	// FIXME?
	switch kind {
	case reflect.Struct:
		panic(newError(nil, "TypeError", 0, "invalid value (struct): missing runtime: %v (%T)", value, value))
	case reflect.Map:
		panic(newError(nil, "TypeError", 0, "invalid value (map): missing runtime: %v (%T)", value, value))
	case reflect.Slice:
		panic(newError(nil, "TypeError", 0, "invalid value (slice): missing runtime: %v (%T)", value, value))
	}
}

func toValue(value interface{}) Value {
	switch value := value.(type) {
	case Value:
		return value
	case bool:
		return Value{kind: valueBoolean, value: value}
	case int:
		return Value{kind: valueNumber, value: value}
	case int8:
		return Value{kind: valueNumber, value: value}
	case int16:
		return Value{kind: valueNumber, value: value}
	case int32:
		return Value{kind: valueNumber, value: value}
	case int64:
		return Value{kind: valueNumber, value: value}
	case uint:
		return Value{kind: valueNumber, value: value}
	case uint8:
		return Value{kind: valueNumber, value: value}
	case uint16:
		return Value{kind: valueNumber, value: value}
	case uint32:
		return Value{kind: valueNumber, value: value}
	case uint64:
		return Value{kind: valueNumber, value: value}
	case float32:
		return Value{kind: valueNumber, value: float64(value)}
	case float64:
		return Value{kind: valueNumber, value: value}
	case []uint16:
		return Value{kind: valueString, value: value}
	case string:
		return Value{kind: valueString, value: value}
	// A rune is actually an int32, which is handled above
	case *object:
		return Value{kind: valueObject, value: value}
	case *Object:
		return Value{kind: valueObject, value: value.object}
	case Object:
		return Value{kind: valueObject, value: value.object}
	case referencer: // reference is an interface (already a pointer)
		return Value{kind: valueReference, value: value}
	case result:
		return Value{kind: valueResult, value: value}
	case nil:
		// TODO Ugh.
		return Value{}
	case reflect.Value:
		for value.Kind() == reflect.Ptr {
			// We were given a pointer, so we'll drill down until we get a non-pointer
			//
			// These semantics might change if we want to start supporting pointers to values transparently
			// (It would be best not to depend on this behavior)
			// FIXME: UNDEFINED
			if value.IsNil() {
				return Value{}
			}
			value = value.Elem()
		}
		switch value.Kind() {
		case reflect.Bool:
			return Value{kind: valueBoolean, value: value.Bool()}
		case reflect.Int:
			return Value{kind: valueNumber, value: int(value.Int())}
		case reflect.Int8:
			return Value{kind: valueNumber, value: int8(value.Int())}
		case reflect.Int16:
			return Value{kind: valueNumber, value: int16(value.Int())}
		case reflect.Int32:
			return Value{kind: valueNumber, value: int32(value.Int())}
		case reflect.Int64:
			return Value{kind: valueNumber, value: value.Int()}
		case reflect.Uint:
			return Value{kind: valueNumber, value: uint(value.Uint())}
		case reflect.Uint8:
			return Value{kind: valueNumber, value: uint8(value.Uint())}
		case reflect.Uint16:
			return Value{kind: valueNumber, value: uint16(value.Uint())}
		case reflect.Uint32:
			return Value{kind: valueNumber, value: uint32(value.Uint())}
		case reflect.Uint64:
			return Value{kind: valueNumber, value: value.Uint()}
		case reflect.Float32:
			return Value{kind: valueNumber, value: float32(value.Float())}
		case reflect.Float64:
			return Value{kind: valueNumber, value: value.Float()}
		case reflect.String:
			return Value{kind: valueString, value: value.String()}
		default:
			reflectValuePanic(value.Interface(), value.Kind())
		}
	default:
		return toValue(reflect.ValueOf(value))
	}
	// FIXME?
	panic(newError(nil, "TypeError", 0, "invalid value: %v (%T)", value, value))
}

// String will return the value as a string.
//
// This method will make return the empty string if there is an error.
func (v Value) String() string {
	var result string
	catchPanic(func() { //nolint:errcheck, gosec
		result = v.string()
	})
	return result
}

// ToBoolean will convert the value to a boolean (bool).
//
//	ToValue(0).ToBoolean() => false
//	ToValue("").ToBoolean() => false
//	ToValue(true).ToBoolean() => true
//	ToValue(1).ToBoolean() => true
//	ToValue("Nothing happens").ToBoolean() => true
//
// If there is an error during the conversion process (like an uncaught exception), then the result will be false and an error.
func (v Value) ToBoolean() (bool, error) {
	result := false
	err := catchPanic(func() {
		result = v.bool()
	})
	return result, err
}

func (v Value) numberValue() Value {
	if v.kind == valueNumber {
		return v
	}
	return Value{kind: valueNumber, value: v.float64()}
}

// ToFloat will convert the value to a number (float64).
//
//	ToValue(0).ToFloat() => 0.
//	ToValue(1.1).ToFloat() => 1.1
//	ToValue("11").ToFloat() => 11.
//
// If there is an error during the conversion process (like an uncaught exception), then the result will be 0 and an error.
func (v Value) ToFloat() (float64, error) {
	result := float64(0)
	err := catchPanic(func() {
		result = v.float64()
	})
	return result, err
}

// ToInteger will convert the value to a number (int64).
//
//	ToValue(0).ToInteger() => 0
//	ToValue(1.1).ToInteger() => 1
//	ToValue("11").ToInteger() => 11
//
// If there is an error during the conversion process (like an uncaught exception), then the result will be 0 and an error.
func (v Value) ToInteger() (int64, error) {
	result := int64(0)
	err := catchPanic(func() {
		result = v.number().int64
	})
	return result, err
}

// ToString will convert the value to a string (string).
//
//	ToValue(0).ToString() => "0"
//	ToValue(false).ToString() => "false"
//	ToValue(1.1).ToString() => "1.1"
//	ToValue("11").ToString() => "11"
//	ToValue('Nothing happens.').ToString() => "Nothing happens."
//
// If there is an error during the conversion process (like an uncaught exception), then the result will be the empty string ("") and an error.
func (v Value) ToString() (string, error) {
	result := ""
	err := catchPanic(func() {
		result = v.string()
	})
	return result, err
}

func (v Value) object() *object {
	if v, ok := v.value.(*object); ok {
		return v
	}
	return nil
}

// Object will return the object of the value, or nil if value is not an object.
//
// This method will not do any implicit conversion. For example, calling this method on a string primitive value will not return a String object.
func (v Value) Object() *Object {
	if obj, ok := v.value.(*object); ok {
		return &Object{
			object: obj,
			value:  v,
		}
	}
	return nil
}

func (v Value) reference() referencer {
	value, _ := v.value.(referencer)
	return value
}

func (v Value) resolve() Value {
	if value, ok := v.value.(referencer); ok {
		return value.getValue()
	}
	return v
}

var (
	nan              float64 = math.NaN()
	positiveInfinity float64 = math.Inf(+1)
	negativeInfinity float64 = math.Inf(-1)
	positiveZero     float64 = 0
	negativeZero     float64 = math.Float64frombits(0 | (1 << 63))
)

// NaNValue will return a value representing NaN.
//
// It is equivalent to:
//
//	ToValue(math.NaN())
func NaNValue() Value {
	return Value{kind: valueNumber, value: nan}
}

func positiveInfinityValue() Value {
	return Value{kind: valueNumber, value: positiveInfinity}
}

func negativeInfinityValue() Value {
	return Value{kind: valueNumber, value: negativeInfinity}
}

func positiveZeroValue() Value {
	return Value{kind: valueNumber, value: positiveZero}
}

func negativeZeroValue() Value {
	return Value{kind: valueNumber, value: negativeZero}
}

// TrueValue will return a value representing true.
//
// It is equivalent to:
//
//	ToValue(true)
func TrueValue() Value {
	return Value{kind: valueBoolean, value: true}
}

// FalseValue will return a value representing false.
//
// It is equivalent to:
//
//	ToValue(false)
func FalseValue() Value {
	return Value{kind: valueBoolean, value: false}
}

func sameValue(x Value, y Value) bool {
	if x.kind != y.kind {
		return false
	}

	switch x.kind {
	case valueUndefined, valueNull:
		return true
	case valueNumber:
		x := x.float64()
		y := y.float64()
		if math.IsNaN(x) && math.IsNaN(y) {
			return true
		}

		if x == y {
			if x == 0 {
				// Since +0 != -0
				return math.Signbit(x) == math.Signbit(y)
			}
			return true
		}
		return false
	case valueString:
		return x.string() == y.string()
	case valueBoolean:
		return x.bool() == y.bool()
	case valueObject:
		return x.object() == y.object()
	default:
		panic(hereBeDragons())
	}
}

func strictEqualityComparison(x Value, y Value) bool {
	if x.kind != y.kind {
		return false
	}

	switch x.kind {
	case valueUndefined, valueNull:
		return true
	case valueNumber:
		x := x.float64()
		y := y.float64()
		if math.IsNaN(x) && math.IsNaN(y) {
			return false
		}
		return x == y
	case valueString:
		return x.string() == y.string()
	case valueBoolean:
		return x.bool() == y.bool()
	case valueObject:
		return x.object() == y.object()
	default:
		panic(hereBeDragons())
	}
}

// Export will attempt to convert the value to a Go representation
// and return it via an interface{} kind.
//
// Export returns an error, but it will always be nil. It is present
// for backwards compatibility.
//
// If a reasonable conversion is not possible, then the original
// value is returned.
//
//	undefined   -> nil (FIXME?: Should be Value{})
//	null        -> nil
//	boolean     -> bool
//	number      -> A number type (int, float32, uint64, ...)
//	string      -> string
//	Array       -> []interface{}
//	Object      -> map[string]interface{}
func (v Value) Export() (interface{}, error) {
	return v.export(), nil
}

func (v Value) export() interface{} {
	switch v.kind {
	case valueUndefined:
		return nil
	case valueNull:
		return nil
	case valueNumber, valueBoolean:
		return v.value
	case valueString:
		switch value := v.value.(type) {
		case string:
			return value
		case []uint16:
			return string(utf16.Decode(value))
		}
	case valueObject:
		obj := v.object()
		switch value := obj.value.(type) {
		case *goStructObject:
			return value.value.Interface()
		case *goMapObject:
			return value.value.Interface()
		case *goArrayObject:
			return value.value.Interface()
		case *goSliceObject:
			return value.value.Interface()
		}
		if obj.class == classArrayName {
			result := make([]interface{}, 0)
			lengthValue := obj.get(propertyLength)
			length := lengthValue.value.(uint32)
			kind := reflect.Invalid
			keyKind := reflect.Invalid
			elemKind := reflect.Invalid
			state := 0
			var t reflect.Type
			for index := range length {
				name := strconv.FormatInt(int64(index), 10)
				if !obj.hasProperty(name) {
					continue
				}
				value := obj.get(name).export()

				t = reflect.TypeOf(value)

				var k, kk, ek reflect.Kind
				if t != nil {
					k = t.Kind()
					switch k {
					case reflect.Map:
						kk = t.Key().Kind()
						fallthrough
					case reflect.Array, reflect.Chan, reflect.Ptr, reflect.Slice:
						ek = t.Elem().Kind()
					}
				}

				if state == 0 {
					kind = k
					keyKind = kk
					elemKind = ek
					state = 1
				} else if state == 1 && (kind != k || keyKind != kk || elemKind != ek) {
					state = 2
				}

				result = append(result, value)
			}

			if state != 1 || kind == reflect.Interface || t == nil {
				// No common type
				return result
			}

			// Convert to the common type
			val := reflect.MakeSlice(reflect.SliceOf(t), len(result), len(result))
			for i, v := range result {
				val.Index(i).Set(reflect.ValueOf(v))
			}
			return val.Interface()
		}

		result := make(map[string]interface{})
		// TODO Should we export everything? Or just what is enumerable?
		obj.enumerate(false, func(name string) bool {
			value := obj.get(name)
			if value.IsDefined() {
				result[name] = value.export()
			}
			return true
		})
		return result
	}

	if v.safe() {
		return v
	}

	return Value{}
}

func (v Value) evaluateBreakContinue(labels []string) resultKind {
	result := v.value.(result)
	if result.kind == resultBreak || result.kind == resultContinue {
		for _, label := range labels {
			if label == result.target {
				return result.kind
			}
		}
	}
	return resultReturn
}

func (v Value) evaluateBreak(labels []string) resultKind {
	result := v.value.(result)
	if result.kind == resultBreak {
		for _, label := range labels {
			if label == result.target {
				return result.kind
			}
		}
	}
	return resultReturn
}

// Make a best effort to return a reflect.Value corresponding to reflect.Kind, but
// fallback to just returning the Go value we have handy.
func (v Value) toReflectValue(typ reflect.Type) (reflect.Value, error) {
	kind := typ.Kind()
	switch kind {
	case reflect.Float32, reflect.Float64, reflect.Interface:
	default:
		switch value := v.value.(type) {
		case float32:
			_, frac := math.Modf(float64(value))
			if frac > 0 {
				return reflect.Value{}, fmt.Errorf("RangeError: %v to reflect.Kind: %v", value, kind)
			}
		case float64:
			_, frac := math.Modf(value)
			if frac > 0 {
				return reflect.Value{}, fmt.Errorf("RangeError: %v to reflect.Kind: %v", value, kind)
			}
		}
	}

	switch kind {
	case reflect.Bool: // Bool
		return reflect.ValueOf(v.bool()).Convert(typ), nil
	case reflect.Int: // Int
		// We convert to float64 here because converting to int64 will not tell us
		// if a value is outside the range of int64
		tmp := toIntegerFloat(v)
		if tmp < floatMinInt || tmp > floatMaxInt {
			return reflect.Value{}, fmt.Errorf("RangeError: %f (%v) to int", tmp, v)
		}
		return reflect.ValueOf(int(tmp)).Convert(typ), nil
	case reflect.Int8: // Int8
		tmp := v.number().int64
		if tmp < int64MinInt8 || tmp > int64MaxInt8 {
			return reflect.Value{}, fmt.Errorf("RangeError: %d (%v) to int8", tmp, v)
		}
		return reflect.ValueOf(int8(tmp)).Convert(typ), nil
	case reflect.Int16: // Int16
		tmp := v.number().int64
		if tmp < int64MinInt16 || tmp > int64MaxInt16 {
			return reflect.Value{}, fmt.Errorf("RangeError: %d (%v) to int16", tmp, v)
		}
		return reflect.ValueOf(int16(tmp)).Convert(typ), nil
	case reflect.Int32: // Int32
		tmp := v.number().int64
		if tmp < int64MinInt32 || tmp > int64MaxInt32 {
			return reflect.Value{}, fmt.Errorf("RangeError: %d (%v) to int32", tmp, v)
		}
		return reflect.ValueOf(int32(tmp)).Convert(typ), nil
	case reflect.Int64: // Int64
		// We convert to float64 here because converting to int64 will not tell us
		// if a value is outside the range of int64
		tmp := toIntegerFloat(v)
		if tmp < floatMinInt64 || tmp > floatMaxInt64 {
			return reflect.Value{}, fmt.Errorf("RangeError: %f (%v) to int", tmp, v)
		}
		return reflect.ValueOf(int64(tmp)).Convert(typ), nil
	case reflect.Uint: // Uint
		// We convert to float64 here because converting to int64 will not tell us
		// if a value is outside the range of uint
		tmp := toIntegerFloat(v)
		if tmp < 0 || tmp > floatMaxUint {
			return reflect.Value{}, fmt.Errorf("RangeError: %f (%v) to uint", tmp, v)
		}
		return reflect.ValueOf(uint(tmp)).Convert(typ), nil
	case reflect.Uint8: // Uint8
		tmp := v.number().int64
		if tmp < 0 || tmp > int64MaxUint8 {
			return reflect.Value{}, fmt.Errorf("RangeError: %d (%v) to uint8", tmp, v)
		}
		return reflect.ValueOf(uint8(tmp)).Convert(typ), nil
	case reflect.Uint16: // Uint16
		tmp := v.number().int64
		if tmp < 0 || tmp > int64MaxUint16 {
			return reflect.Value{}, fmt.Errorf("RangeError: %d (%v) to uint16", tmp, v)
		}
		return reflect.ValueOf(uint16(tmp)).Convert(typ), nil
	case reflect.Uint32: // Uint32
		tmp := v.number().int64
		if tmp < 0 || tmp > int64MaxUint32 {
			return reflect.Value{}, fmt.Errorf("RangeError: %d (%v) to uint32", tmp, v)
		}
		return reflect.ValueOf(uint32(tmp)).Convert(typ), nil
	case reflect.Uint64: // Uint64
		// We convert to float64 here because converting to int64 will not tell us
		// if a value is outside the range of uint64
		tmp := toIntegerFloat(v)
		if tmp < 0 || tmp > floatMaxUint64 {
			return reflect.Value{}, fmt.Errorf("RangeError: %f (%v) to uint64", tmp, v)
		}
		return reflect.ValueOf(uint64(tmp)).Convert(typ), nil
	case reflect.Float32: // Float32
		tmp := v.float64()
		tmp1 := tmp
		if 0 > tmp1 {
			tmp1 = -tmp1
		}
		if tmp1 > 0 && (tmp1 < math.SmallestNonzeroFloat32 || tmp1 > math.MaxFloat32) {
			return reflect.Value{}, fmt.Errorf("RangeError: %f (%v) to float32", tmp, v)
		}
		return reflect.ValueOf(float32(tmp)).Convert(typ), nil
	case reflect.Float64: // Float64
		value := v.float64()
		return reflect.ValueOf(value).Convert(typ), nil
	case reflect.String: // String
		return reflect.ValueOf(v.string()).Convert(typ), nil
	case reflect.Invalid: // Invalid
	case reflect.Complex64: // FIXME? Complex64
	case reflect.Complex128: // FIXME? Complex128
	case reflect.Chan: // FIXME? Chan
	case reflect.Func: // FIXME? Func
	case reflect.Ptr: // FIXME? Ptr
	case reflect.UnsafePointer: // FIXME? UnsafePointer
	default:
		switch v.kind {
		case valueObject:
			obj := v.object()
			switch vl := obj.value.(type) {
			case *goStructObject: // Struct
				return reflect.ValueOf(vl.value.Interface()), nil
			case *goMapObject: // Map
				return reflect.ValueOf(vl.value.Interface()), nil
			case *goArrayObject: // Array
				return reflect.ValueOf(vl.value.Interface()), nil
			case *goSliceObject: // Slice
				return reflect.ValueOf(vl.value.Interface()), nil
			}
			exported := reflect.ValueOf(v.export())
			if exported.Type().ConvertibleTo(typ) {
				return exported.Convert(typ), nil
			}
			return reflect.Value{}, fmt.Errorf("TypeError: could not convert %v to reflect.Type: %v", exported, typ)
		case valueEmpty, valueResult, valueReference:
			// These are invalid, and should panic
		default:
			return reflect.ValueOf(v.value), nil
		}
	}

	// FIXME Should this end up as a TypeError?
	panic(fmt.Errorf("invalid conversion of %v (%v) to reflect.Type: %v", v.kind, v, typ))
}

func stringToReflectValue(value string, kind reflect.Kind) (reflect.Value, error) {
	switch kind {
	case reflect.Bool:
		value, err := strconv.ParseBool(value)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(value), nil
	case reflect.Int:
		value, err := strconv.ParseInt(value, 0, 0)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(int(value)), nil
	case reflect.Int8:
		value, err := strconv.ParseInt(value, 0, 8)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(int8(value)), nil
	case reflect.Int16:
		value, err := strconv.ParseInt(value, 0, 16)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(int16(value)), nil
	case reflect.Int32:
		value, err := strconv.ParseInt(value, 0, 32)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(int32(value)), nil
	case reflect.Int64:
		value, err := strconv.ParseInt(value, 0, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(value), nil
	case reflect.Uint:
		value, err := strconv.ParseUint(value, 0, 0)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(uint(value)), nil
	case reflect.Uint8:
		value, err := strconv.ParseUint(value, 0, 8)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(uint8(value)), nil
	case reflect.Uint16:
		value, err := strconv.ParseUint(value, 0, 16)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(uint16(value)), nil
	case reflect.Uint32:
		value, err := strconv.ParseUint(value, 0, 32)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(uint32(value)), nil
	case reflect.Uint64:
		value, err := strconv.ParseUint(value, 0, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(value), nil
	case reflect.Float32:
		value, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(float32(value)), nil
	case reflect.Float64:
		value, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(value), nil
	case reflect.String:
		return reflect.ValueOf(value), nil
	}

	// FIXME This should end up as a TypeError?
	panic(fmt.Errorf("invalid conversion of %q to reflect.Kind: %v", value, kind))
}

// MarshalJSON implements json.Marshaller.
func (v Value) MarshalJSON() ([]byte, error) {
	switch v.kind {
	case valueUndefined, valueNull:
		return []byte("null"), nil
	case valueBoolean, valueNumber:
		return json.Marshal(v.value)
	case valueString:
		return json.Marshal(v.string())
	case valueObject:
		return v.Object().MarshalJSON()
	}
	return nil, fmt.Errorf("invalid type %v", v.kind)
}
