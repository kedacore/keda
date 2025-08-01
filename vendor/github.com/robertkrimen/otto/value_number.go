package otto

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var stringToNumberParseInteger = regexp.MustCompile(`^(?:0[xX])`)

func parseNumber(value string) float64 {
	value = strings.Trim(value, builtinStringTrimWhitespace)

	if value == "" {
		return 0
	}

	var parseFloat bool
	switch {
	case strings.ContainsRune(value, '.'):
		parseFloat = true
	case stringToNumberParseInteger.MatchString(value):
		parseFloat = false
	default:
		parseFloat = true
	}

	if parseFloat {
		number, err := strconv.ParseFloat(value, 64)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			return math.NaN()
		}
		return number
	}

	number, err := strconv.ParseInt(value, 0, 64)
	if err != nil {
		return math.NaN()
	}
	return float64(number)
}

func (v Value) float64() float64 {
	switch v.kind {
	case valueUndefined:
		return math.NaN()
	case valueNull:
		return 0
	}
	switch value := v.value.(type) {
	case bool:
		if value {
			return 1
		}
		return 0
	case int:
		return float64(value)
	case int8:
		return float64(value)
	case int16:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	case uint:
		return float64(value)
	case uint8:
		return float64(value)
	case uint16:
		return float64(value)
	case uint32:
		return float64(value)
	case uint64:
		return float64(value)
	case float64:
		return value
	case string:
		return parseNumber(value)
	case *object:
		return value.DefaultValue(defaultValueHintNumber).float64()
	}
	panic(fmt.Errorf("toFloat(%T)", v.value))
}

const (
	sqrt1_2 float64 = math.Sqrt2 / 2
)

const (
	maxUint32 = math.MaxUint32
	maxInt    = int(^uint(0) >> 1)

	// int64.
	int64MaxInt8   int64 = math.MaxInt8
	int64MinInt8   int64 = math.MinInt8
	int64MaxInt16  int64 = math.MaxInt16
	int64MinInt16  int64 = math.MinInt16
	int64MaxInt32  int64 = math.MaxInt32
	int64MinInt32  int64 = math.MinInt32
	int64MaxUint8  int64 = math.MaxUint8
	int64MaxUint16 int64 = math.MaxUint16
	int64MaxUint32 int64 = math.MaxUint32

	// float64.
	floatMaxInt    float64 = float64(int(^uint(0) >> 1))
	floatMinInt    float64 = float64(-maxInt - 1)
	floatMaxUint   float64 = float64(^uint(0))
	floatMaxUint64 float64 = math.MaxUint64
	floatMaxInt64  float64 = math.MaxInt64
	floatMinInt64  float64 = math.MinInt64
)

func toIntegerFloat(value Value) float64 {
	float := value.float64()
	switch {
	case math.IsInf(float, 0):
		return float
	case math.IsNaN(float):
		return 0
	case float > 0:
		return math.Floor(float)
	default:
		return math.Ceil(float)
	}
}

type numberKind int

const (
	numberInteger  numberKind = iota // 3.0 => 3.0
	numberFloat                      // 3.14159 => 3.0, 1+2**63 > 2**63-1
	numberInfinity                   // Infinity => 2**63-1
	numberNaN                        // NaN => 0
)

type _number struct {
	kind    numberKind
	int64   int64
	float64 float64
}

// FIXME
// http://www.goinggo.net/2013/08/gustavos-ieee-754-brain-teaser.html
// http://bazaar.launchpad.net/~niemeyer/strepr/trunk/view/6/strepr.go#L160
func (v Value) number() _number {
	var num _number
	switch value := v.value.(type) {
	case int8:
		num.int64 = int64(value)
		return num
	case int16:
		num.int64 = int64(value)
		return num
	case uint8:
		num.int64 = int64(value)
		return num
	case uint16:
		num.int64 = int64(value)
		return num
	case uint32:
		num.int64 = int64(value)
		return num
	case int:
		num.int64 = int64(value)
		return num
	case int64:
		num.int64 = value
		return num
	}

	float := v.float64()
	if float == 0 {
		return num
	}

	num.kind = numberFloat
	num.float64 = float

	if math.IsNaN(float) {
		num.kind = numberNaN
		return num
	}

	if math.IsInf(float, 0) {
		num.kind = numberInfinity
	}

	if float >= floatMaxInt64 {
		num.int64 = math.MaxInt64
		return num
	}

	if float <= floatMinInt64 {
		num.int64 = math.MinInt64
		return num
	}

	var integer float64
	if float > 0 {
		integer = math.Floor(float)
	} else {
		integer = math.Ceil(float)
	}

	if float == integer {
		num.kind = numberInteger
	}
	num.int64 = int64(float)
	return num
}

// ECMA 262: 9.5.
func toInt32(value Value) int32 {
	switch value := value.value.(type) {
	case int8:
		return int32(value)
	case int16:
		return int32(value)
	case int32:
		return value
	}

	floatValue := value.float64()
	if math.IsNaN(floatValue) || math.IsInf(floatValue, 0) || floatValue == 0 {
		return 0
	}

	// Convert to int64 before int32 to force correct wrapping.
	return int32(int64(floatValue))
}

func toUint32(value Value) uint32 {
	switch value := value.value.(type) {
	case int8:
		return uint32(value)
	case int16:
		return uint32(value)
	case uint8:
		return uint32(value)
	case uint16:
		return uint32(value)
	case uint32:
		return value
	}

	floatValue := value.float64()
	if math.IsNaN(floatValue) || math.IsInf(floatValue, 0) || floatValue == 0 {
		return 0
	}

	// Convert to int64 before uint32 to force correct wrapping.
	return uint32(int64(floatValue))
}

// ECMA 262 - 6.0 - 7.1.8.
func toUint16(value Value) uint16 {
	switch value := value.value.(type) {
	case int8:
		return uint16(value)
	case uint8:
		return uint16(value)
	case uint16:
		return value
	}

	floatValue := value.float64()
	if math.IsNaN(floatValue) || math.IsInf(floatValue, 0) || floatValue == 0 {
		return 0
	}

	// Convert to int64 before uint16 to force correct wrapping.
	return uint16(int64(floatValue))
}

// toIntSign returns sign of a number converted to -1, 0 ,1.
func toIntSign(value Value) int {
	switch value := value.value.(type) {
	case int8:
		if value > 0 {
			return 1
		} else if value < 0 {
			return -1
		}

		return 0
	case int16:
		if value > 0 {
			return 1
		} else if value < 0 {
			return -1
		}

		return 0
	case int32:
		if value > 0 {
			return 1
		} else if value < 0 {
			return -1
		}

		return 0
	case uint8:
		if value > 0 {
			return 1
		}

		return 0
	case uint16:
		if value > 0 {
			return 1
		}

		return 0
	case uint32:
		if value > 0 {
			return 1
		}

		return 0
	}
	floatValue := value.float64()
	switch {
	case math.IsNaN(floatValue), math.IsInf(floatValue, 0):
		return 0
	case floatValue == 0:
		return 0
	case floatValue > 0:
		return 1
	default:
		return -1
	}
}
