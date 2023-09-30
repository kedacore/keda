package builtin

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/antonmedv/expr/ast"
	"github.com/antonmedv/expr/vm/runtime"
)

var (
	Index map[string]int
	Names []string
)

func init() {
	Index = make(map[string]int)
	Names = make([]string, len(Builtins))
	for i, fn := range Builtins {
		Index[fn.Name] = i
		Names[i] = fn.Name
	}
}

var Builtins = []*ast.Function{
	{
		Name:      "all",
		Predicate: true,
		Types:     types(new(func([]any, func(any) bool) bool)),
	},
	{
		Name:      "none",
		Predicate: true,
		Types:     types(new(func([]any, func(any) bool) bool)),
	},
	{
		Name:      "any",
		Predicate: true,
		Types:     types(new(func([]any, func(any) bool) bool)),
	},
	{
		Name:      "one",
		Predicate: true,
		Types:     types(new(func([]any, func(any) bool) bool)),
	},
	{
		Name:      "filter",
		Predicate: true,
		Types:     types(new(func([]any, func(any) bool) []any)),
	},
	{
		Name:      "map",
		Predicate: true,
		Types:     types(new(func([]any, func(any) any) []any)),
	},
	{
		Name:      "find",
		Predicate: true,
		Types:     types(new(func([]any, func(any) bool) any)),
	},
	{
		Name:      "findIndex",
		Predicate: true,
		Types:     types(new(func([]any, func(any) bool) int)),
	},
	{
		Name:      "findLast",
		Predicate: true,
		Types:     types(new(func([]any, func(any) bool) any)),
	},
	{
		Name:      "findLastIndex",
		Predicate: true,
		Types:     types(new(func([]any, func(any) bool) int)),
	},
	{
		Name:      "count",
		Predicate: true,
		Types:     types(new(func([]any, func(any) bool) int)),
	},
	{
		Name:      "groupBy",
		Predicate: true,
		Types:     types(new(func([]any, func(any) any) map[any][]any)),
	},
	{
		Name:      "reduce",
		Predicate: true,
		Types:     types(new(func([]any, func(any, any) any, any) any)),
	},
	{
		Name: "len",
		Fast: Len,
		Validate: func(args []reflect.Type) (reflect.Type, error) {
			if len(args) != 1 {
				return anyType, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			switch kind(args[0]) {
			case reflect.Array, reflect.Map, reflect.Slice, reflect.String, reflect.Interface:
				return integerType, nil
			}
			return anyType, fmt.Errorf("invalid argument for len (type %s)", args[0])
		},
	},
	{
		Name:  "type",
		Fast:  Type,
		Types: types(new(func(any) string)),
	},
	{
		Name: "abs",
		Fast: Abs,
		Validate: func(args []reflect.Type) (reflect.Type, error) {
			if len(args) != 1 {
				return anyType, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			switch kind(args[0]) {
			case reflect.Float32, reflect.Float64, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Interface:
				return args[0], nil
			}
			return anyType, fmt.Errorf("invalid argument for abs (type %s)", args[0])
		},
	},
	{
		Name: "int",
		Fast: Int,
		Validate: func(args []reflect.Type) (reflect.Type, error) {
			if len(args) != 1 {
				return anyType, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			switch kind(args[0]) {
			case reflect.Interface:
				return integerType, nil
			case reflect.Float32, reflect.Float64, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return integerType, nil
			case reflect.String:
				return integerType, nil
			}
			return anyType, fmt.Errorf("invalid argument for int (type %s)", args[0])
		},
	},
	{
		Name: "float",
		Fast: Float,
		Validate: func(args []reflect.Type) (reflect.Type, error) {
			if len(args) != 1 {
				return anyType, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			switch kind(args[0]) {
			case reflect.Interface:
				return floatType, nil
			case reflect.Float32, reflect.Float64, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return floatType, nil
			case reflect.String:
				return floatType, nil
			}
			return anyType, fmt.Errorf("invalid argument for float (type %s)", args[0])
		},
	},
	{
		Name:  "string",
		Fast:  String,
		Types: types(new(func(any any) string)),
	},
	{
		Name: "trim",
		Func: func(args ...any) (any, error) {
			if len(args) == 1 {
				return strings.TrimSpace(args[0].(string)), nil
			} else if len(args) == 2 {
				return strings.Trim(args[0].(string), args[1].(string)), nil
			} else {
				return nil, fmt.Errorf("invalid number of arguments for trim (expected 1 or 2, got %d)", len(args))
			}
		},
		Types: types(
			strings.TrimSpace,
			strings.Trim,
		),
	},
	{
		Name: "trimPrefix",
		Func: func(args ...any) (any, error) {
			s := " "
			if len(args) == 2 {
				s = args[1].(string)
			}
			return strings.TrimPrefix(args[0].(string), s), nil
		},
		Types: types(
			strings.TrimPrefix,
			new(func(string) string),
		),
	},
	{
		Name: "trimSuffix",
		Func: func(args ...any) (any, error) {
			s := " "
			if len(args) == 2 {
				s = args[1].(string)
			}
			return strings.TrimSuffix(args[0].(string), s), nil
		},
		Types: types(
			strings.TrimSuffix,
			new(func(string) string),
		),
	},
	{
		Name: "upper",
		Fast: func(arg any) any {
			return strings.ToUpper(arg.(string))
		},
		Types: types(strings.ToUpper),
	},
	{
		Name: "lower",
		Fast: func(arg any) any {
			return strings.ToLower(arg.(string))
		},
		Types: types(strings.ToLower),
	},
	{
		Name: "split",
		Func: func(args ...any) (any, error) {
			if len(args) == 2 {
				return strings.Split(args[0].(string), args[1].(string)), nil
			} else if len(args) == 3 {
				return strings.SplitN(args[0].(string), args[1].(string), runtime.ToInt(args[2])), nil
			} else {
				return nil, fmt.Errorf("invalid number of arguments for split (expected 2 or 3, got %d)", len(args))
			}
		},
		Types: types(
			strings.Split,
			strings.SplitN,
		),
	},
	{
		Name: "splitAfter",
		Func: func(args ...any) (any, error) {
			if len(args) == 2 {
				return strings.SplitAfter(args[0].(string), args[1].(string)), nil
			} else if len(args) == 3 {
				return strings.SplitAfterN(args[0].(string), args[1].(string), runtime.ToInt(args[2])), nil
			} else {
				return nil, fmt.Errorf("invalid number of arguments for splitAfter (expected 2 or 3, got %d)", len(args))
			}
		},
		Types: types(
			strings.SplitAfter,
			strings.SplitAfterN,
		),
	},
	{
		Name: "replace",
		Func: func(args ...any) (any, error) {
			if len(args) == 4 {
				return strings.Replace(args[0].(string), args[1].(string), args[2].(string), runtime.ToInt(args[3])), nil
			} else if len(args) == 3 {
				return strings.ReplaceAll(args[0].(string), args[1].(string), args[2].(string)), nil
			} else {
				return nil, fmt.Errorf("invalid number of arguments for replace (expected 3 or 4, got %d)", len(args))
			}
		},
		Types: types(
			strings.Replace,
			strings.ReplaceAll,
		),
	},
	{
		Name: "repeat",
		Func: func(args ...any) (any, error) {
			n := runtime.ToInt(args[1])
			if n > 1e6 {
				panic("memory budget exceeded")
			}
			return strings.Repeat(args[0].(string), n), nil
		},
		Types: types(strings.Repeat),
	},
	{
		Name: "join",
		Func: func(args ...any) (any, error) {
			glue := ""
			if len(args) == 2 {
				glue = args[1].(string)
			}
			switch args[0].(type) {
			case []string:
				return strings.Join(args[0].([]string), glue), nil
			case []any:
				var s []string
				for _, arg := range args[0].([]any) {
					s = append(s, arg.(string))
				}
				return strings.Join(s, glue), nil
			}
			return nil, fmt.Errorf("invalid argument for join (type %s)", reflect.TypeOf(args[0]))
		},
		Types: types(
			strings.Join,
			new(func([]any, string) string),
			new(func([]any) string),
			new(func([]string, string) string),
			new(func([]string) string),
		),
	},
	{
		Name: "indexOf",
		Func: func(args ...any) (any, error) {
			return strings.Index(args[0].(string), args[1].(string)), nil
		},
		Types: types(strings.Index),
	},
	{
		Name: "lastIndexOf",
		Func: func(args ...any) (any, error) {
			return strings.LastIndex(args[0].(string), args[1].(string)), nil
		},
		Types: types(strings.LastIndex),
	},
	{
		Name: "hasPrefix",
		Func: func(args ...any) (any, error) {
			return strings.HasPrefix(args[0].(string), args[1].(string)), nil
		},
		Types: types(strings.HasPrefix),
	},
	{
		Name: "hasSuffix",
		Func: func(args ...any) (any, error) {
			return strings.HasSuffix(args[0].(string), args[1].(string)), nil
		},
		Types: types(strings.HasSuffix),
	},
	{
		Name: "max",
		Func: Max,
		Validate: func(args []reflect.Type) (reflect.Type, error) {
			if len(args) == 0 {
				return anyType, fmt.Errorf("not enough arguments to call max")
			}
			for _, arg := range args {
				switch kind(arg) {
				case reflect.Interface:
					return anyType, nil
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
				default:
					return anyType, fmt.Errorf("invalid argument for max (type %s)", arg)
				}
			}
			return args[0], nil
		},
	},
	{
		Name: "min",
		Func: Min,
		Validate: func(args []reflect.Type) (reflect.Type, error) {
			if len(args) == 0 {
				return anyType, fmt.Errorf("not enough arguments to call min")
			}
			for _, arg := range args {
				switch kind(arg) {
				case reflect.Interface:
					return anyType, nil
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
				default:
					return anyType, fmt.Errorf("invalid argument for min (type %s)", arg)
				}
			}
			return args[0], nil
		},
	},
	{
		Name: "sum",
		Func: func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			v := reflect.ValueOf(args[0])
			if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
				return nil, fmt.Errorf("cannot sum %s", v.Kind())
			}
			sum := int64(0)
			i := 0
			for ; i < v.Len(); i++ {
				it := deref(v.Index(i))
				if it.CanInt() {
					sum += it.Int()
				} else if it.CanFloat() {
					goto float
				} else {
					return nil, fmt.Errorf("cannot sum %s", it.Kind())
				}
			}
			return int(sum), nil
		float:
			fSum := float64(sum)
			for ; i < v.Len(); i++ {
				it := deref(v.Index(i))
				if it.CanInt() {
					fSum += float64(it.Int())
				} else if it.CanFloat() {
					fSum += it.Float()
				} else {
					return nil, fmt.Errorf("cannot sum %s", it.Kind())
				}
			}
			return fSum, nil
		},
		Validate: func(args []reflect.Type) (reflect.Type, error) {
			if len(args) != 1 {
				return anyType, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			switch kind(args[0]) {
			case reflect.Interface, reflect.Slice, reflect.Array:
			default:
				return anyType, fmt.Errorf("cannot sum %s", args[0])
			}
			return anyType, nil
		},
	},
	{
		Name: "mean",
		Func: func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			v := reflect.ValueOf(args[0])
			if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
				return nil, fmt.Errorf("cannot mean %s", v.Kind())
			}
			if v.Len() == 0 {
				return 0.0, nil
			}
			sum := float64(0)
			i := 0
			for ; i < v.Len(); i++ {
				it := deref(v.Index(i))
				if it.CanInt() {
					sum += float64(it.Int())
				} else if it.CanFloat() {
					sum += it.Float()
				} else {
					return nil, fmt.Errorf("cannot mean %s", it.Kind())
				}
			}
			return sum / float64(i), nil
		},
		Validate: func(args []reflect.Type) (reflect.Type, error) {
			if len(args) != 1 {
				return anyType, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			switch kind(args[0]) {
			case reflect.Interface, reflect.Slice, reflect.Array:
			default:
				return anyType, fmt.Errorf("cannot avg %s", args[0])
			}
			return floatType, nil
		},
	},
	{
		Name: "median",
		Func: func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			v := reflect.ValueOf(args[0])
			if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
				return nil, fmt.Errorf("cannot median %s", v.Kind())
			}
			if v.Len() == 0 {
				return 0.0, nil
			}
			s := make([]float64, v.Len())
			for i := 0; i < v.Len(); i++ {
				it := deref(v.Index(i))
				if it.CanInt() {
					s[i] = float64(it.Int())
				} else if it.CanFloat() {
					s[i] = it.Float()
				} else {
					return nil, fmt.Errorf("cannot median %s", it.Kind())
				}
			}
			sort.Float64s(s)
			if len(s)%2 == 0 {
				return (s[len(s)/2-1] + s[len(s)/2]) / 2, nil
			}
			return s[len(s)/2], nil
		},
		Validate: func(args []reflect.Type) (reflect.Type, error) {
			if len(args) != 1 {
				return anyType, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			switch kind(args[0]) {
			case reflect.Interface, reflect.Slice, reflect.Array:
			default:
				return anyType, fmt.Errorf("cannot median %s", args[0])
			}
			return floatType, nil
		},
	},
	{
		Name: "toJSON",
		Func: func(args ...any) (any, error) {
			b, err := json.MarshalIndent(args[0], "", "  ")
			if err != nil {
				return nil, err
			}
			return string(b), nil
		},
		Types: types(new(func(any) string)),
	},
	{
		Name: "fromJSON",
		Func: func(args ...any) (any, error) {
			var v any
			err := json.Unmarshal([]byte(args[0].(string)), &v)
			if err != nil {
				return nil, err
			}
			return v, nil
		},
		Types: types(new(func(string) any)),
	},
	{
		Name: "toBase64",
		Func: func(args ...any) (any, error) {
			return base64.StdEncoding.EncodeToString([]byte(args[0].(string))), nil
		},
		Types: types(new(func(string) string)),
	},
	{
		Name: "fromBase64",
		Func: func(args ...any) (any, error) {
			b, err := base64.StdEncoding.DecodeString(args[0].(string))
			if err != nil {
				return nil, err
			}
			return string(b), nil
		},
		Types: types(new(func(string) string)),
	},
	{
		Name: "now",
		Func: func(args ...any) (any, error) {
			return time.Now(), nil
		},
		Types: types(new(func() time.Time)),
	},
	{
		Name: "duration",
		Func: func(args ...any) (any, error) {
			return time.ParseDuration(args[0].(string))
		},
		Types: types(time.ParseDuration),
	},
	{
		Name: "date",
		Func: func(args ...any) (any, error) {
			date := args[0].(string)
			if len(args) == 2 {
				layout := args[1].(string)
				return time.Parse(layout, date)
			}
			if len(args) == 3 {
				layout := args[1].(string)
				timeZone := args[2].(string)
				tz, err := time.LoadLocation(timeZone)
				if err != nil {
					return nil, err
				}
				t, err := time.ParseInLocation(layout, date, tz)
				if err != nil {
					return nil, err
				}
				return t, nil
			}

			layouts := []string{
				"2006-01-02",
				"15:04:05",
				"2006-01-02 15:04:05",
				time.RFC3339,
				time.RFC822,
				time.RFC850,
				time.RFC1123,
			}
			for _, layout := range layouts {
				t, err := time.Parse(layout, date)
				if err == nil {
					return t, nil
				}
			}
			return nil, fmt.Errorf("invalid date %s", date)
		},
		Types: types(
			new(func(string) time.Time),
			new(func(string, string) time.Time),
			new(func(string, string, string) time.Time),
		),
	},
	{
		Name: "first",
		Func: func(args ...any) (any, error) {
			defer func() {
				if r := recover(); r != nil {
					return
				}
			}()
			return runtime.Fetch(args[0], 0), nil
		},
		Validate: func(args []reflect.Type) (reflect.Type, error) {
			if len(args) != 1 {
				return anyType, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			switch kind(args[0]) {
			case reflect.Interface:
				return anyType, nil
			case reflect.Slice, reflect.Array:
				return args[0].Elem(), nil
			}
			return anyType, fmt.Errorf("cannot get first element from %s", args[0])
		},
	},
	{
		Name: "last",
		Func: func(args ...any) (any, error) {
			defer func() {
				if r := recover(); r != nil {
					return
				}
			}()
			return runtime.Fetch(args[0], -1), nil
		},
		Validate: func(args []reflect.Type) (reflect.Type, error) {
			if len(args) != 1 {
				return anyType, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			switch kind(args[0]) {
			case reflect.Interface:
				return anyType, nil
			case reflect.Slice, reflect.Array:
				return args[0].Elem(), nil
			}
			return anyType, fmt.Errorf("cannot get last element from %s", args[0])
		},
	},
	{
		Name: "get",
		Func: func(args ...any) (out any, err error) {
			defer func() {
				if r := recover(); r != nil {
					return
				}
			}()
			return runtime.Fetch(args[0], args[1]), nil
		},
	},
	{
		Name: "take",
		Func: func(args ...any) (any, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("invalid number of arguments (expected 2, got %d)", len(args))
			}
			v := reflect.ValueOf(args[0])
			if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
				return nil, fmt.Errorf("cannot take from %s", v.Kind())
			}
			n := reflect.ValueOf(args[1])
			if !n.CanInt() {
				return nil, fmt.Errorf("cannot take %s elements", n.Kind())
			}
			if n.Int() > int64(v.Len()) {
				return args[0], nil
			}
			return v.Slice(0, int(n.Int())).Interface(), nil
		},
		Validate: func(args []reflect.Type) (reflect.Type, error) {
			if len(args) != 2 {
				return anyType, fmt.Errorf("invalid number of arguments (expected 2, got %d)", len(args))
			}
			switch kind(args[0]) {
			case reflect.Interface, reflect.Slice, reflect.Array:
			default:
				return anyType, fmt.Errorf("cannot take from %s", args[0])
			}
			switch kind(args[1]) {
			case reflect.Interface, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			default:
				return anyType, fmt.Errorf("cannot take %s elements", args[1])
			}
			return args[0], nil
		},
	},
	{
		Name: "keys",
		Func: func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			v := reflect.ValueOf(args[0])
			if v.Kind() != reflect.Map {
				return nil, fmt.Errorf("cannot get keys from %s", v.Kind())
			}
			keys := v.MapKeys()
			out := make([]any, len(keys))
			for i, key := range keys {
				out[i] = key.Interface()
			}
			return out, nil
		},
		Validate: func(args []reflect.Type) (reflect.Type, error) {
			if len(args) != 1 {
				return anyType, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			switch kind(args[0]) {
			case reflect.Interface:
				return arrayType, nil
			case reflect.Map:
				return arrayType, nil
			}
			return anyType, fmt.Errorf("cannot get keys from %s", args[0])
		},
	},
	{
		Name: "values",
		Func: func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			v := reflect.ValueOf(args[0])
			if v.Kind() != reflect.Map {
				return nil, fmt.Errorf("cannot get values from %s", v.Kind())
			}
			keys := v.MapKeys()
			out := make([]any, len(keys))
			for i, key := range keys {
				out[i] = v.MapIndex(key).Interface()
			}
			return out, nil
		},
		Validate: func(args []reflect.Type) (reflect.Type, error) {
			if len(args) != 1 {
				return anyType, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			switch kind(args[0]) {
			case reflect.Interface:
				return arrayType, nil
			case reflect.Map:
				return arrayType, nil
			}
			return anyType, fmt.Errorf("cannot get values from %s", args[0])
		},
	},
	{
		Name: "toPairs",
		Func: func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			v := reflect.ValueOf(args[0])
			if v.Kind() != reflect.Map {
				return nil, fmt.Errorf("cannot transform %s to pairs", v.Kind())
			}
			keys := v.MapKeys()
			out := make([][2]any, len(keys))
			for i, key := range keys {
				out[i] = [2]any{key.Interface(), v.MapIndex(key).Interface()}
			}
			return out, nil
		},
		Validate: func(args []reflect.Type) (reflect.Type, error) {
			if len(args) != 1 {
				return anyType, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			switch kind(args[0]) {
			case reflect.Interface, reflect.Map:
				return arrayType, nil
			}
			return anyType, fmt.Errorf("cannot transform %s to pairs", args[0])
		},
	},
	{
		Name: "fromPairs",
		Func: func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			v := reflect.ValueOf(args[0])
			if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
				return nil, fmt.Errorf("cannot transform %s from pairs", v)
			}
			out := reflect.MakeMap(mapType)
			for i := 0; i < v.Len(); i++ {
				pair := deref(v.Index(i))
				if pair.Kind() != reflect.Array && pair.Kind() != reflect.Slice {
					return nil, fmt.Errorf("invalid pair %v", pair)
				}
				if pair.Len() != 2 {
					return nil, fmt.Errorf("invalid pair length %v", pair)
				}
				key := pair.Index(0)
				value := pair.Index(1)
				out.SetMapIndex(key, value)
			}
			return out.Interface(), nil
		},
		Validate: func(args []reflect.Type) (reflect.Type, error) {
			if len(args) != 1 {
				return anyType, fmt.Errorf("invalid number of arguments (expected 1, got %d)", len(args))
			}
			switch kind(args[0]) {
			case reflect.Interface, reflect.Slice, reflect.Array:
				return mapType, nil
			}
			return anyType, fmt.Errorf("cannot transform %s from pairs", args[0])
		},
	},
	{
		Name: "sort",
		Func: func(args ...any) (any, error) {
			if len(args) != 1 && len(args) != 2 {
				return nil, fmt.Errorf("invalid number of arguments (expected 1 or 2, got %d)", len(args))
			}

			v := reflect.ValueOf(args[0])
			if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
				return nil, fmt.Errorf("cannot sort %s", v.Kind())
			}

			orderBy := OrderBy{}
			if len(args) == 2 {
				dir, err := ascOrDesc(args[1])
				if err != nil {
					return nil, err
				}
				orderBy.Desc = dir
			}

			sortable, err := copyArray(v, orderBy)
			if err != nil {
				return nil, err
			}
			sort.Sort(sortable)
			return sortable.Array, nil
		},
		Validate: func(args []reflect.Type) (reflect.Type, error) {
			if len(args) != 1 && len(args) != 2 {
				return anyType, fmt.Errorf("invalid number of arguments (expected 1 or 2, got %d)", len(args))
			}
			switch kind(args[0]) {
			case reflect.Interface, reflect.Slice, reflect.Array:
			default:
				return anyType, fmt.Errorf("cannot sort %s", args[0])
			}
			if len(args) == 2 {
				switch kind(args[1]) {
				case reflect.String, reflect.Interface:
				default:
					return anyType, fmt.Errorf("invalid argument for sort (expected string, got %s)", args[1])
				}
			}
			return arrayType, nil
		},
	},
	{
		Name: "sortBy",
		Func: func(args ...any) (any, error) {
			if len(args) != 2 && len(args) != 3 {
				return nil, fmt.Errorf("invalid number of arguments (expected 2 or 3, got %d)", len(args))
			}

			v := reflect.ValueOf(args[0])
			if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
				return nil, fmt.Errorf("cannot sort %s", v.Kind())
			}

			orderBy := OrderBy{}

			field, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("invalid argument for sort (expected string, got %s)", reflect.TypeOf(args[1]))
			}
			orderBy.Field = field

			if len(args) == 3 {
				dir, err := ascOrDesc(args[2])
				if err != nil {
					return nil, err
				}
				orderBy.Desc = dir
			}

			sortable, err := copyArray(v, orderBy)
			if err != nil {
				return nil, err
			}
			sort.Sort(sortable)
			return sortable.Array, nil
		},
		Validate: func(args []reflect.Type) (reflect.Type, error) {
			if len(args) != 2 && len(args) != 3 {
				return anyType, fmt.Errorf("invalid number of arguments (expected 2 or 3, got %d)", len(args))
			}
			switch kind(args[0]) {
			case reflect.Interface, reflect.Slice, reflect.Array:
			default:
				return anyType, fmt.Errorf("cannot sort %s", args[0])
			}
			switch kind(args[1]) {
			case reflect.String, reflect.Interface:
			default:
				return anyType, fmt.Errorf("invalid argument for sort (expected string, got %s)", args[1])
			}
			if len(args) == 3 {
				switch kind(args[2]) {
				case reflect.String, reflect.Interface:
				default:
					return anyType, fmt.Errorf("invalid argument for sort (expected string, got %s)", args[1])
				}
			}
			return arrayType, nil
		},
	},
}
