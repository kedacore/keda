package otto

import (
	"math"
	"math/rand"
)

// Math

func builtinMathAbs(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Abs(number))
}

func builtinMathAcos(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Acos(number))
}

func builtinMathAcosh(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Acosh(number))
}

func builtinMathAsin(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Asin(number))
}

func builtinMathAsinh(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Asinh(number))
}

func builtinMathAtan(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Atan(number))
}

func builtinMathAtan2(call FunctionCall) Value {
	y := call.Argument(0).float64()
	if math.IsNaN(y) {
		return NaNValue()
	}
	x := call.Argument(1).float64()
	if math.IsNaN(x) {
		return NaNValue()
	}
	return float64Value(math.Atan2(y, x))
}

func builtinMathAtanh(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Atanh(number))
}

func builtinMathCbrt(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Cbrt(number))
}

func builtinMathCos(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Cos(number))
}

func builtinMathCeil(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Ceil(number))
}

func builtinMathCosh(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Cosh(number))
}

func builtinMathExp(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Exp(number))
}

func builtinMathExpm1(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Expm1(number))
}

func builtinMathFloor(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Floor(number))
}

func builtinMathLog(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Log(number))
}

func builtinMathLog10(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Log10(number))
}

func builtinMathLog1p(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Log1p(number))
}

func builtinMathLog2(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Log2(number))
}

func builtinMathMax(call FunctionCall) Value {
	switch len(call.ArgumentList) {
	case 0:
		return negativeInfinityValue()
	case 1:
		return float64Value(call.ArgumentList[0].float64())
	}
	result := call.ArgumentList[0].float64()
	if math.IsNaN(result) {
		return NaNValue()
	}
	for _, value := range call.ArgumentList[1:] {
		value := value.float64()
		if math.IsNaN(value) {
			return NaNValue()
		}
		result = math.Max(result, value)
	}
	return float64Value(result)
}

func builtinMathMin(call FunctionCall) Value {
	switch len(call.ArgumentList) {
	case 0:
		return positiveInfinityValue()
	case 1:
		return float64Value(call.ArgumentList[0].float64())
	}
	result := call.ArgumentList[0].float64()
	if math.IsNaN(result) {
		return NaNValue()
	}
	for _, value := range call.ArgumentList[1:] {
		value := value.float64()
		if math.IsNaN(value) {
			return NaNValue()
		}
		result = math.Min(result, value)
	}
	return float64Value(result)
}

func builtinMathPow(call FunctionCall) Value {
	// TODO Make sure this works according to the specification (15.8.2.13)
	x := call.Argument(0).float64()
	y := call.Argument(1).float64()
	if math.Abs(x) == 1 && math.IsInf(y, 0) {
		return NaNValue()
	}
	return float64Value(math.Pow(x, y))
}

func builtinMathRandom(call FunctionCall) Value {
	var v float64
	if call.runtime.random != nil {
		v = call.runtime.random()
	} else {
		v = rand.Float64() //nolint:gosec
	}
	return float64Value(v)
}

func builtinMathRound(call FunctionCall) Value {
	number := call.Argument(0).float64()
	value := math.Floor(number + 0.5)
	if value == 0 {
		value = math.Copysign(0, number)
	}
	return float64Value(value)
}

func builtinMathSin(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Sin(number))
}

func builtinMathSinh(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Sinh(number))
}

func builtinMathSqrt(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Sqrt(number))
}

func builtinMathTan(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Tan(number))
}

func builtinMathTanh(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Tanh(number))
}

func builtinMathTrunc(call FunctionCall) Value {
	number := call.Argument(0).float64()
	return float64Value(math.Trunc(number))
}
