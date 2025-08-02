package otto

import (
	"fmt"
	"math"
	"strings"

	"github.com/robertkrimen/otto/token"
)

func (rt *runtime) evaluateMultiply(left float64, right float64) Value { //nolint:unused
	// TODO 11.5.1
	return Value{}
}

func (rt *runtime) evaluateDivide(left float64, right float64) Value {
	if math.IsNaN(left) || math.IsNaN(right) {
		return NaNValue()
	}
	if math.IsInf(left, 0) && math.IsInf(right, 0) {
		return NaNValue()
	}
	if left == 0 && right == 0 {
		return NaNValue()
	}
	if math.IsInf(left, 0) {
		if math.Signbit(left) == math.Signbit(right) {
			return positiveInfinityValue()
		}
		return negativeInfinityValue()
	}
	if math.IsInf(right, 0) {
		if math.Signbit(left) == math.Signbit(right) {
			return positiveZeroValue()
		}
		return negativeZeroValue()
	}
	if right == 0 {
		if math.Signbit(left) == math.Signbit(right) {
			return positiveInfinityValue()
		}
		return negativeInfinityValue()
	}
	return float64Value(left / right)
}

func (rt *runtime) evaluateModulo(left float64, right float64) Value { //nolint:unused
	// TODO 11.5.3
	return Value{}
}

func (rt *runtime) calculateBinaryExpression(operator token.Token, left Value, right Value) Value {
	leftValue := left.resolve()

	switch operator {
	// Additive
	case token.PLUS:
		leftValue = toPrimitiveValue(leftValue)
		rightValue := right.resolve()
		rightValue = toPrimitiveValue(rightValue)

		if leftValue.IsString() || rightValue.IsString() {
			return stringValue(strings.Join([]string{leftValue.string(), rightValue.string()}, ""))
		}
		return float64Value(leftValue.float64() + rightValue.float64())
	case token.MINUS:
		rightValue := right.resolve()
		return float64Value(leftValue.float64() - rightValue.float64())

		// Multiplicative
	case token.MULTIPLY:
		rightValue := right.resolve()
		return float64Value(leftValue.float64() * rightValue.float64())
	case token.SLASH:
		rightValue := right.resolve()
		return rt.evaluateDivide(leftValue.float64(), rightValue.float64())
	case token.REMAINDER:
		rightValue := right.resolve()
		return float64Value(math.Mod(leftValue.float64(), rightValue.float64()))

		// Logical
	case token.LOGICAL_AND:
		left := leftValue.bool()
		if !left {
			return falseValue
		}
		return boolValue(right.resolve().bool())
	case token.LOGICAL_OR:
		left := leftValue.bool()
		if left {
			return trueValue
		}
		return boolValue(right.resolve().bool())

		// Bitwise
	case token.AND:
		rightValue := right.resolve()
		return int32Value(toInt32(leftValue) & toInt32(rightValue))
	case token.OR:
		rightValue := right.resolve()
		return int32Value(toInt32(leftValue) | toInt32(rightValue))
	case token.EXCLUSIVE_OR:
		rightValue := right.resolve()
		return int32Value(toInt32(leftValue) ^ toInt32(rightValue))

		// Shift
		// (Masking of 0x1f is to restrict the shift to a maximum of 31 places)
	case token.SHIFT_LEFT:
		rightValue := right.resolve()
		return int32Value(toInt32(leftValue) << (toUint32(rightValue) & 0x1f))
	case token.SHIFT_RIGHT:
		rightValue := right.resolve()
		return int32Value(toInt32(leftValue) >> (toUint32(rightValue) & 0x1f))
	case token.UNSIGNED_SHIFT_RIGHT:
		rightValue := right.resolve()
		// Shifting an unsigned integer is a logical shift
		return uint32Value(toUint32(leftValue) >> (toUint32(rightValue) & 0x1f))

	case token.INSTANCEOF:
		rightValue := right.resolve()
		if !rightValue.IsObject() {
			panic(rt.panicTypeError("invalid kind %s for instanceof (expected object)", rightValue.kind))
		}
		return boolValue(rightValue.object().hasInstance(leftValue))

	case token.IN:
		rightValue := right.resolve()
		if !rightValue.IsObject() {
			panic(rt.panicTypeError("invalid kind %s for in (expected object)", rightValue.kind))
		}
		return boolValue(rightValue.object().hasProperty(leftValue.string()))
	}

	panic(hereBeDragons(operator))
}

type lessThanResult int

const (
	lessThanFalse lessThanResult = iota
	lessThanTrue
	lessThanUndefined
)

func calculateLessThan(left Value, right Value, leftFirst bool) lessThanResult {
	var x, y Value
	if leftFirst {
		x = toNumberPrimitive(left)
		y = toNumberPrimitive(right)
	} else {
		y = toNumberPrimitive(right)
		x = toNumberPrimitive(left)
	}

	var result bool
	if x.kind != valueString || y.kind != valueString {
		x, y := x.float64(), y.float64()
		if math.IsNaN(x) || math.IsNaN(y) {
			return lessThanUndefined
		}
		result = x < y
	} else {
		x, y := x.string(), y.string()
		result = x < y
	}

	if result {
		return lessThanTrue
	}

	return lessThanFalse
}

// FIXME Probably a map is not the most efficient way to do this.
var lessThanTable [4](map[lessThanResult]bool) = [4](map[lessThanResult]bool){
	// <
	map[lessThanResult]bool{
		lessThanFalse:     false,
		lessThanTrue:      true,
		lessThanUndefined: false,
	},

	// >
	map[lessThanResult]bool{
		lessThanFalse:     false,
		lessThanTrue:      true,
		lessThanUndefined: false,
	},

	// <=
	map[lessThanResult]bool{
		lessThanFalse:     true,
		lessThanTrue:      false,
		lessThanUndefined: false,
	},

	// >=
	map[lessThanResult]bool{
		lessThanFalse:     true,
		lessThanTrue:      false,
		lessThanUndefined: false,
	},
}

func (rt *runtime) calculateComparison(comparator token.Token, left Value, right Value) bool {
	// FIXME Use strictEqualityComparison?
	// TODO This might be redundant now (with regards to evaluateComparison)
	x := left.resolve()
	y := right.resolve()

	var kindEqualKind bool
	var negate bool
	result := true

	switch comparator {
	case token.LESS:
		result = lessThanTable[0][calculateLessThan(x, y, true)]
	case token.GREATER:
		result = lessThanTable[1][calculateLessThan(y, x, false)]
	case token.LESS_OR_EQUAL:
		result = lessThanTable[2][calculateLessThan(y, x, false)]
	case token.GREATER_OR_EQUAL:
		result = lessThanTable[3][calculateLessThan(x, y, true)]
	case token.STRICT_NOT_EQUAL:
		negate = true
		fallthrough
	case token.STRICT_EQUAL:
		if x.kind != y.kind {
			result = false
		} else {
			kindEqualKind = true
		}
	case token.NOT_EQUAL:
		negate = true
		fallthrough
	case token.EQUAL:
		switch {
		case x.kind == y.kind:
			kindEqualKind = true
		case x.kind <= valueNull && y.kind <= valueNull:
			result = true
		case x.kind <= valueNull || y.kind <= valueNull:
			result = false
		case x.kind <= valueString && y.kind <= valueString:
			result = x.float64() == y.float64()
		case x.kind == valueBoolean:
			result = rt.calculateComparison(token.EQUAL, float64Value(x.float64()), y)
		case y.kind == valueBoolean:
			result = rt.calculateComparison(token.EQUAL, x, float64Value(y.float64()))
		case x.kind == valueObject:
			result = rt.calculateComparison(token.EQUAL, toPrimitiveValue(x), y)
		case y.kind == valueObject:
			result = rt.calculateComparison(token.EQUAL, x, toPrimitiveValue(y))
		default:
			panic(fmt.Sprintf("unknown types for equal: %v ==? %v", x, y))
		}
	default:
		panic("unknown comparator " + comparator.String())
	}

	if kindEqualKind {
		switch x.kind {
		case valueUndefined, valueNull:
			result = true
		case valueNumber:
			x := x.float64()
			y := y.float64()
			if math.IsNaN(x) || math.IsNaN(y) {
				result = false
			} else {
				result = x == y
			}
		case valueString:
			result = x.string() == y.string()
		case valueBoolean:
			result = x.bool() == y.bool()
		case valueObject:
			result = x.object() == y.object()
		default:
			goto ERROR
		}
	}

	if negate {
		result = !result
	}

	return result

ERROR:
	panic(hereBeDragons("%v (%v) %s %v (%v)", x, x.kind, comparator, y, y.kind))
}
