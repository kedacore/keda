package builtin

import (
	"fmt"
	"math"
	"reflect"
	"strconv"

	"github.com/expr-lang/expr/vm/runtime"
)

func Len(x any) any {
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return v.Len()
	default:
		panic(fmt.Sprintf("invalid argument for len (type %T)", x))
	}
}

func Type(arg any) any {
	if arg == nil {
		return "nil"
	}
	v := reflect.ValueOf(arg)
	for {
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		} else if v.Kind() == reflect.Interface {
			v = v.Elem()
		} else {
			break
		}
	}
	if v.Type().Name() != "" && v.Type().PkgPath() != "" {
		return fmt.Sprintf("%s.%s", v.Type().PkgPath(), v.Type().Name())
	}
	switch v.Type().Kind() {
	case reflect.Invalid:
		return "invalid"
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "int"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "uint"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.String:
		return "string"
	case reflect.Array, reflect.Slice:
		return "array"
	case reflect.Map:
		return "map"
	case reflect.Func:
		return "func"
	case reflect.Struct:
		return "struct"
	}
	return "unknown"
}

func Abs(x any) any {
	switch x.(type) {
	case float32:
		if x.(float32) < 0 {
			return -x.(float32)
		} else {
			return x
		}
	case float64:
		if x.(float64) < 0 {
			return -x.(float64)
		} else {
			return x
		}
	case int:
		if x.(int) < 0 {
			return -x.(int)
		} else {
			return x
		}
	case int8:
		if x.(int8) < 0 {
			return -x.(int8)
		} else {
			return x
		}
	case int16:
		if x.(int16) < 0 {
			return -x.(int16)
		} else {
			return x
		}
	case int32:
		if x.(int32) < 0 {
			return -x.(int32)
		} else {
			return x
		}
	case int64:
		if x.(int64) < 0 {
			return -x.(int64)
		} else {
			return x
		}
	case uint:
		if x.(uint) < 0 {
			return -x.(uint)
		} else {
			return x
		}
	case uint8:
		if x.(uint8) < 0 {
			return -x.(uint8)
		} else {
			return x
		}
	case uint16:
		if x.(uint16) < 0 {
			return -x.(uint16)
		} else {
			return x
		}
	case uint32:
		if x.(uint32) < 0 {
			return -x.(uint32)
		} else {
			return x
		}
	case uint64:
		if x.(uint64) < 0 {
			return -x.(uint64)
		} else {
			return x
		}
	}
	panic(fmt.Sprintf("invalid argument for abs (type %T)", x))
}

func Ceil(x any) any {
	switch x := x.(type) {
	case float32:
		return math.Ceil(float64(x))
	case float64:
		return math.Ceil(x)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return Float(x)
	}
	panic(fmt.Sprintf("invalid argument for ceil (type %T)", x))
}

func Floor(x any) any {
	switch x := x.(type) {
	case float32:
		return math.Floor(float64(x))
	case float64:
		return math.Floor(x)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return Float(x)
	}
	panic(fmt.Sprintf("invalid argument for floor (type %T)", x))
}

func Round(x any) any {
	switch x := x.(type) {
	case float32:
		return math.Round(float64(x))
	case float64:
		return math.Round(x)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return Float(x)
	}
	panic(fmt.Sprintf("invalid argument for round (type %T)", x))
}

func Int(x any) any {
	switch x := x.(type) {
	case float32:
		return int(x)
	case float64:
		return int(x)
	case int:
		return x
	case int8:
		return int(x)
	case int16:
		return int(x)
	case int32:
		return int(x)
	case int64:
		return int(x)
	case uint:
		return int(x)
	case uint8:
		return int(x)
	case uint16:
		return int(x)
	case uint32:
		return int(x)
	case uint64:
		return int(x)
	case string:
		i, err := strconv.Atoi(x)
		if err != nil {
			panic(fmt.Sprintf("invalid operation: int(%s)", x))
		}
		return i
	default:
		panic(fmt.Sprintf("invalid operation: int(%T)", x))
	}
}

func Float(x any) any {
	switch x := x.(type) {
	case float32:
		return float64(x)
	case float64:
		return x
	case int:
		return float64(x)
	case int8:
		return float64(x)
	case int16:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case uint:
		return float64(x)
	case uint8:
		return float64(x)
	case uint16:
		return float64(x)
	case uint32:
		return float64(x)
	case uint64:
		return float64(x)
	case string:
		f, err := strconv.ParseFloat(x, 64)
		if err != nil {
			panic(fmt.Sprintf("invalid operation: float(%s)", x))
		}
		return f
	default:
		panic(fmt.Sprintf("invalid operation: float(%T)", x))
	}
}

func String(arg any) any {
	return fmt.Sprintf("%v", arg)
}

func Max(args ...any) (any, error) {
	var max any
	for _, arg := range args {
		if max == nil || runtime.Less(max, arg) {
			max = arg
		}
	}
	return max, nil
}

func Min(args ...any) (any, error) {
	var min any
	for _, arg := range args {
		if min == nil || runtime.More(min, arg) {
			min = arg
		}
	}
	return min, nil
}

func bitFunc(name string, fn func(x, y int) (any, error)) *Function {
	return &Function{
		Name: name,
		Func: func(args ...any) (any, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("invalid number of arguments for %s (expected 2, got %d)", name, len(args))
			}
			x, err := toInt(args[0])
			if err != nil {
				return nil, fmt.Errorf("%v to call %s", err, name)
			}
			y, err := toInt(args[1])
			if err != nil {
				return nil, fmt.Errorf("%v to call %s", err, name)
			}
			return fn(x, y)
		},
		Types: types(new(func(int, int) int)),
	}
}
