package otto

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path"
	"reflect"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/parser"
)

type global struct {
	Object         *object // Object( ... ), new Object( ... ) - 1 (length)
	Function       *object // Function( ... ), new Function( ... ) - 1
	Array          *object // Array( ... ), new Array( ... ) - 1
	String         *object // String( ... ), new String( ... ) - 1
	Boolean        *object // Boolean( ... ), new Boolean( ... ) - 1
	Number         *object // Number( ... ), new Number( ... ) - 1
	Math           *object
	Date           *object // Date( ... ), new Date( ... ) - 7
	RegExp         *object // RegExp( ... ), new RegExp( ... ) - 2
	Error          *object // Error( ... ), new Error( ... ) - 1
	EvalError      *object
	TypeError      *object
	RangeError     *object
	ReferenceError *object
	SyntaxError    *object
	URIError       *object
	JSON           *object

	ObjectPrototype         *object // Object.prototype
	FunctionPrototype       *object // Function.prototype
	ArrayPrototype          *object // Array.prototype
	StringPrototype         *object // String.prototype
	BooleanPrototype        *object // Boolean.prototype
	NumberPrototype         *object // Number.prototype
	DatePrototype           *object // Date.prototype
	RegExpPrototype         *object // RegExp.prototype
	ErrorPrototype          *object // Error.prototype
	EvalErrorPrototype      *object
	TypeErrorPrototype      *object
	RangeErrorPrototype     *object
	ReferenceErrorPrototype *object
	SyntaxErrorPrototype    *object
	URIErrorPrototype       *object
}

type runtime struct {
	global       global
	globalObject *object
	globalStash  *objectStash
	scope        *scope
	otto         *Otto
	eval         *object
	debugger     func(*Otto)
	random       func() float64
	labels       []string
	stackLimit   int
	traceLimit   int
	lck          sync.Mutex
}

func (rt *runtime) enterScope(scop *scope) {
	scop.outer = rt.scope
	if rt.scope != nil {
		if rt.stackLimit != 0 && rt.scope.depth+1 >= rt.stackLimit {
			panic(rt.panicRangeError("Maximum call stack size exceeded"))
		}

		scop.depth = rt.scope.depth + 1
	}

	rt.scope = scop
}

func (rt *runtime) leaveScope() {
	rt.scope = rt.scope.outer
}

// FIXME This is used in two places (cloning).
func (rt *runtime) enterGlobalScope() {
	rt.enterScope(newScope(rt.globalStash, rt.globalStash, rt.globalObject))
}

func (rt *runtime) enterFunctionScope(outer stasher, this Value) *fnStash {
	if outer == nil {
		outer = rt.globalStash
	}
	stash := rt.newFunctionStash(outer)
	var thisObject *object
	switch this.kind {
	case valueUndefined, valueNull:
		thisObject = rt.globalObject
	default:
		thisObject = rt.toObject(this)
	}
	rt.enterScope(newScope(stash, stash, thisObject))
	return stash
}

func (rt *runtime) putValue(reference referencer, value Value) {
	name := reference.putValue(value)
	if name != "" {
		// Why? -- If reference.base == nil
		// strict = false
		rt.globalObject.defineProperty(name, value, 0o111, false)
	}
}

func (rt *runtime) tryCatchEvaluate(inner func() Value) (tryValue Value, isException bool) { //nolint:nonamedreturns
	// resultValue = The value of the block (e.g. the last statement)
	// throw = Something was thrown
	// throwValue = The value of what was thrown
	// other = Something that changes flow (return, break, continue) that is not a throw
	// Otherwise, some sort of unknown panic happened, we'll just propagate it.
	defer func() {
		if caught := recover(); caught != nil {
			if excep, ok := caught.(*exception); ok {
				caught = excep.eject()
			}
			switch caught := caught.(type) {
			case ottoError:
				isException = true
				tryValue = objectValue(rt.newErrorObjectError(caught))
			case Value:
				isException = true
				tryValue = caught
			default:
				isException = true
				tryValue = toValue(caught)
			}
		}
	}()

	return inner(), false
}

func (rt *runtime) toObject(value Value) *object {
	switch value.kind {
	case valueEmpty, valueUndefined, valueNull:
		panic(rt.panicTypeError("toObject unsupported kind %s", value.kind))
	case valueBoolean:
		return rt.newBoolean(value)
	case valueString:
		return rt.newString(value)
	case valueNumber:
		return rt.newNumber(value)
	case valueObject:
		return value.object()
	default:
		panic(rt.panicTypeError("toObject unknown kind %s", value.kind))
	}
}

func (rt *runtime) objectCoerce(value Value) (*object, error) {
	switch value.kind {
	case valueUndefined:
		return nil, errors.New("undefined")
	case valueNull:
		return nil, errors.New("null")
	case valueBoolean:
		return rt.newBoolean(value), nil
	case valueString:
		return rt.newString(value), nil
	case valueNumber:
		return rt.newNumber(value), nil
	case valueObject:
		return value.object(), nil
	default:
		panic(rt.panicTypeError("objectCoerce unknown kind %s", value.kind))
	}
}

func checkObjectCoercible(rt *runtime, value Value) {
	isObject, mustCoerce := testObjectCoercible(value)
	if !isObject && !mustCoerce {
		panic(rt.panicTypeError("checkObjectCoercible not object or mustCoerce"))
	}
}

// testObjectCoercible.
func testObjectCoercible(value Value) (isObject, mustCoerce bool) { //nolint:nonamedreturns
	switch value.kind {
	case valueReference, valueEmpty, valueNull, valueUndefined:
		return false, false
	case valueNumber, valueString, valueBoolean:
		return false, true
	case valueObject:
		return true, false
	default:
		panic(fmt.Sprintf("testObjectCoercible unknown kind %s", value.kind))
	}
}

func (rt *runtime) safeToValue(value interface{}) (Value, error) {
	result := Value{}
	err := catchPanic(func() {
		result = rt.toValue(value)
	})
	return result, err
}

// convertNumeric converts numeric parameter val from js to that of type t if it is safe to do so, otherwise it panics.
// This allows literals (int64), bitwise values (int32) and the general form (float64) of javascript numerics to be passed as parameters to go functions easily.
func (rt *runtime) convertNumeric(v Value, t reflect.Type) reflect.Value {
	val := reflect.ValueOf(v.export())

	if val.Kind() == t.Kind() {
		return val
	}

	if val.Kind() == reflect.Interface {
		val = reflect.ValueOf(val.Interface())
	}

	switch val.Kind() {
	case reflect.Float32, reflect.Float64:
		f64 := val.Float()
		switch t.Kind() {
		case reflect.Float64:
			return reflect.ValueOf(f64)
		case reflect.Float32:
			if reflect.Zero(t).OverflowFloat(f64) {
				panic(rt.panicRangeError("converting float64 to float32 would overflow"))
			}

			return val.Convert(t)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			i64 := int64(f64)
			if float64(i64) != f64 {
				panic(rt.panicRangeError(fmt.Sprintf("converting %v to %v would cause loss of precision", val.Type(), t)))
			}

			// The float represents an integer
			val = reflect.ValueOf(i64)
		default:
			panic(rt.panicTypeError(fmt.Sprintf("cannot convert %v to %v", val.Type(), t)))
		}
	}

	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i64 := val.Int()
		switch t.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if reflect.Zero(t).OverflowInt(i64) {
				panic(rt.panicRangeError(fmt.Sprintf("converting %v to %v would overflow", val.Type(), t)))
			}
			return val.Convert(t)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if i64 < 0 {
				panic(rt.panicRangeError(fmt.Sprintf("converting %v to %v would underflow", val.Type(), t)))
			}
			if reflect.Zero(t).OverflowUint(uint64(i64)) {
				panic(rt.panicRangeError(fmt.Sprintf("converting %v to %v would overflow", val.Type(), t)))
			}
			return val.Convert(t)
		case reflect.Float32, reflect.Float64:
			return val.Convert(t)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u64 := val.Uint()
		switch t.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if u64 > math.MaxInt64 || reflect.Zero(t).OverflowInt(int64(u64)) {
				panic(rt.panicRangeError(fmt.Sprintf("converting %v to %v would overflow", val.Type(), t)))
			}
			return val.Convert(t)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if reflect.Zero(t).OverflowUint(u64) {
				panic(rt.panicRangeError(fmt.Sprintf("converting %v to %v would overflow", val.Type(), t)))
			}
			return val.Convert(t)
		case reflect.Float32, reflect.Float64:
			return val.Convert(t)
		}
	}

	panic(rt.panicTypeError(fmt.Sprintf("unsupported type %v -> %v for numeric conversion", val.Type(), t)))
}

func fieldIndexByName(t reflect.Type, name string) []int {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for i := range t.NumField() {
		f := t.Field(i)

		if !validGoStructName(f.Name) {
			continue
		}

		if f.Anonymous {
			for t.Kind() == reflect.Ptr {
				t = t.Elem()
			}

			if f.Type.Kind() == reflect.Struct {
				if a := fieldIndexByName(f.Type, name); a != nil {
					return append([]int{i}, a...)
				}
			}
		}

		if a := strings.SplitN(f.Tag.Get("json"), ",", 2); a[0] != "" {
			if a[0] == "-" {
				continue
			}

			if a[0] == name {
				return []int{i}
			}
		}

		if f.Name == name {
			return []int{i}
		}
	}

	return nil
}

var (
	typeOfValue          = reflect.TypeOf(Value{})
	typeOfJSONRawMessage = reflect.TypeOf(json.RawMessage{})
)

// convertCallParameter converts request val to type t if possible.
// If the conversion fails due to overflow or type miss-match then it panics.
// If no conversion is known then the original value is returned.
func (rt *runtime) convertCallParameter(v Value, t reflect.Type) (reflect.Value, error) {
	if t == typeOfValue {
		return reflect.ValueOf(v), nil
	}

	if t == typeOfJSONRawMessage {
		if d, err := json.Marshal(v.export()); err == nil {
			return reflect.ValueOf(d), nil
		}
	}

	if v.kind == valueObject {
		if gso, ok := v.object().value.(*goStructObject); ok {
			if gso.value.Type().AssignableTo(t) {
				// please see TestDynamicFunctionReturningInterface for why this exists
				if t.Kind() == reflect.Interface && gso.value.Type().ConvertibleTo(t) {
					return gso.value.Convert(t), nil
				}
				return gso.value, nil
			}
		}

		if gao, ok := v.object().value.(*goArrayObject); ok {
			if gao.value.Type().AssignableTo(t) {
				// please see TestDynamicFunctionReturningInterface for why this exists
				if t.Kind() == reflect.Interface && gao.value.Type().ConvertibleTo(t) {
					return gao.value.Convert(t), nil
				}
				return gao.value, nil
			}
		}
	}

	tk := t.Kind()

	if tk == reflect.Interface {
		e := v.export()
		if e == nil {
			return reflect.Zero(t), nil
		}
		iv := reflect.ValueOf(e)
		if iv.Type().AssignableTo(t) {
			return iv, nil
		}
	}

	if tk == reflect.Ptr {
		switch v.kind {
		case valueEmpty, valueNull, valueUndefined:
			return reflect.Zero(t), nil
		default:
			var vv reflect.Value
			vv, err := rt.convertCallParameter(v, t.Elem())
			if err != nil {
				return reflect.Zero(t), fmt.Errorf("can't convert to %s: %w", t, err)
			}

			if vv.CanAddr() {
				return vv.Addr(), nil
			}

			pv := reflect.New(vv.Type())
			pv.Elem().Set(vv)
			return pv, nil
		}
	}

	switch tk {
	case reflect.Bool:
		return reflect.ValueOf(v.bool()), nil
	case reflect.String:
		switch v.kind {
		case valueString:
			return reflect.ValueOf(v.value), nil
		case valueNumber:
			return reflect.ValueOf(fmt.Sprintf("%v", v.value)), nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		if v.kind == valueNumber {
			return rt.convertNumeric(v, t), nil
		}
	case reflect.Slice:
		if o := v.object(); o != nil {
			if lv := o.get(propertyLength); lv.IsNumber() {
				l := lv.number().int64

				s := reflect.MakeSlice(t, int(l), int(l))

				tt := t.Elem()

				switch o.class {
				case classArrayName:
					for i := range l {
						p, ok := o.property[strconv.FormatInt(i, 10)]
						if !ok {
							continue
						}

						e, ok := p.value.(Value)
						if !ok {
							continue
						}

						ev, err := rt.convertCallParameter(e, tt)
						if err != nil {
							return reflect.Zero(t), fmt.Errorf("couldn't convert element %d of %s: %w", i, t, err)
						}

						s.Index(int(i)).Set(ev)
					}
				case classGoArrayName, classGoSliceName:
					var gslice bool
					switch o.value.(type) {
					case *goSliceObject:
						gslice = true
					case *goArrayObject:
						gslice = false
					}

					for i := range l {
						var p *property
						if gslice {
							p = goSliceGetOwnProperty(o, strconv.FormatInt(i, 10))
						} else {
							p = goArrayGetOwnProperty(o, strconv.FormatInt(i, 10))
						}
						if p == nil {
							continue
						}

						e, ok := p.value.(Value)
						if !ok {
							continue
						}

						ev, err := rt.convertCallParameter(e, tt)
						if err != nil {
							return reflect.Zero(t), fmt.Errorf("couldn't convert element %d of %s: %w", i, t, err)
						}

						s.Index(int(i)).Set(ev)
					}
				}

				return s, nil
			}
		}
	case reflect.Map:
		if o := v.object(); o != nil && t.Key().Kind() == reflect.String {
			m := reflect.MakeMap(t)

			var err error

			o.enumerate(false, func(k string) bool {
				v, verr := rt.convertCallParameter(o.get(k), t.Elem())
				if verr != nil {
					err = fmt.Errorf("couldn't convert property %q of %s: %w", k, t, verr)
					return false
				}
				m.SetMapIndex(reflect.ValueOf(k), v)
				return true
			})

			if err != nil {
				return reflect.Zero(t), err
			}

			return m, nil
		}
	case reflect.Func:
		if t.NumOut() > 1 {
			return reflect.Zero(t), errors.New("converting JavaScript values to Go functions with more than one return value is currently not supported")
		}

		if o := v.object(); o != nil && o.class == classFunctionName {
			return reflect.MakeFunc(t, func(args []reflect.Value) []reflect.Value {
				l := make([]interface{}, len(args))
				for i, a := range args {
					if a.CanInterface() {
						l[i] = a.Interface()
					}
				}

				rv, err := v.Call(nullValue, l...)
				if err != nil {
					panic(err)
				}

				if t.NumOut() == 0 {
					return nil
				}

				r, err := rt.convertCallParameter(rv, t.Out(0))
				if err != nil {
					panic(rt.panicTypeError("convertCallParameter Func: %s", err))
				}

				return []reflect.Value{r}
			}), nil
		}
	case reflect.Struct:
		if o := v.object(); o != nil && o.class == classObjectName {
			s := reflect.New(t)

			for _, k := range o.propertyOrder {
				idx := fieldIndexByName(t, k)

				if idx == nil {
					return reflect.Zero(t), fmt.Errorf("can't convert property %q of %s: field does not exist", k, t)
				}

				ss := s

				for _, i := range idx {
					if ss.Kind() == reflect.Ptr {
						if ss.IsNil() {
							if !ss.CanSet() {
								return reflect.Zero(t), fmt.Errorf("can't convert property %q of %s: %s is unexported", k, t, ss.Type().Elem())
							}

							ss.Set(reflect.New(ss.Type().Elem()))
						}

						ss = ss.Elem()
					}

					ss = ss.Field(i)
				}

				v, err := rt.convertCallParameter(o.get(k), ss.Type())
				if err != nil {
					return reflect.Zero(t), fmt.Errorf("couldn't convert property %q of %s: %w", k, t, err)
				}

				ss.Set(v)
			}

			return s.Elem(), nil
		}
	}

	if tk == reflect.String {
		if o := v.object(); o != nil && o.hasProperty("toString") {
			if fn := o.get("toString"); fn.IsFunction() {
				sv, err := fn.Call(v)
				if err != nil {
					return reflect.Zero(t), fmt.Errorf("couldn't call toString: %w", err)
				}

				r, err := rt.convertCallParameter(sv, t)
				if err != nil {
					return reflect.Zero(t), fmt.Errorf("couldn't convert toString result: %w", err)
				}
				return r, nil
			}
		}

		return reflect.ValueOf(v.String()), nil
	}

	if v.kind == valueString {
		var s encoding.TextUnmarshaler

		if reflect.PointerTo(t).Implements(reflect.TypeOf(&s).Elem()) {
			r := reflect.New(t)

			if err := r.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(v.string())); err != nil {
				return reflect.Zero(t), fmt.Errorf("can't convert to %s as TextUnmarshaller: %w", t.String(), err)
			}

			return r.Elem(), nil
		}
	}

	s := "OTTO DOES NOT UNDERSTAND THIS TYPE"
	switch v.kind {
	case valueBoolean:
		s = "boolean"
	case valueNull:
		s = "null"
	case valueNumber:
		s = "number"
	case valueString:
		s = "string"
	case valueUndefined:
		s = "undefined"
	case valueObject:
		s = v.Class()
	}

	return reflect.Zero(t), fmt.Errorf("can't convert from %q to %q", s, t)
}

func (rt *runtime) toValue(value interface{}) Value {
	rv, ok := value.(reflect.Value)
	if ok {
		value = rv.Interface()
	}

	switch value := value.(type) {
	case Value:
		return value
	case func(FunctionCall) Value:
		var name, file string
		var line int
		pc := reflect.ValueOf(value).Pointer()
		fn := goruntime.FuncForPC(pc)
		if fn != nil {
			name = fn.Name()
			file, line = fn.FileLine(pc)
			file = path.Base(file)
		}
		return objectValue(rt.newNativeFunction(name, file, line, value))
	case nativeFunction:
		var name, file string
		var line int
		pc := reflect.ValueOf(value).Pointer()
		fn := goruntime.FuncForPC(pc)
		if fn != nil {
			name = fn.Name()
			file, line = fn.FileLine(pc)
			file = path.Base(file)
		}
		return objectValue(rt.newNativeFunction(name, file, line, value))
	case Object, *Object, object, *object:
		// Nothing happens.
		// FIXME We should really figure out what can come here.
		// This catch-all is ugly.
	default:
		val := reflect.ValueOf(value)
		if ok && val.Kind() == rv.Kind() {
			// Use passed in rv which may be writable.
			val = rv
		}

		switch val.Kind() {
		case reflect.Ptr:
			switch reflect.Indirect(val).Kind() {
			case reflect.Struct:
				return objectValue(rt.newGoStructObject(val))
			case reflect.Array:
				return objectValue(rt.newGoArray(val))
			}
		case reflect.Struct:
			return objectValue(rt.newGoStructObject(val))
		case reflect.Map:
			return objectValue(rt.newGoMapObject(val))
		case reflect.Slice:
			return objectValue(rt.newGoSlice(val))
		case reflect.Array:
			return objectValue(rt.newGoArray(val))
		case reflect.Func:
			var name, file string
			var line int
			if v := reflect.ValueOf(val); v.Kind() == reflect.Ptr {
				pc := v.Pointer()
				fn := goruntime.FuncForPC(pc)
				if fn != nil {
					name = fn.Name()
					file, line = fn.FileLine(pc)
					file = path.Base(file)
				}
			}

			typ := val.Type()

			return objectValue(rt.newNativeFunction(name, file, line, func(c FunctionCall) Value {
				nargs := typ.NumIn()

				if len(c.ArgumentList) != nargs {
					if typ.IsVariadic() {
						if len(c.ArgumentList) < nargs-1 {
							panic(rt.panicRangeError(fmt.Sprintf("expected at least %d arguments; got %d", nargs-1, len(c.ArgumentList))))
						}
					} else {
						panic(rt.panicRangeError(fmt.Sprintf("expected %d argument(s); got %d", nargs, len(c.ArgumentList))))
					}
				}

				in := make([]reflect.Value, len(c.ArgumentList))

				callSlice := false

				for i, a := range c.ArgumentList {
					var t reflect.Type

					n := i
					if n >= nargs-1 && typ.IsVariadic() {
						if n > nargs-1 {
							n = nargs - 1
						}

						t = typ.In(n).Elem()
					} else {
						t = typ.In(n)
					}

					// if this is a variadic Go function, and the caller has supplied
					// exactly the number of JavaScript arguments required, and this
					// is the last JavaScript argument, try treating the it as the
					// actual set of variadic Go arguments. if that succeeds, break
					// out of the loop.
					if typ.IsVariadic() && len(c.ArgumentList) == nargs && i == nargs-1 {
						if v, err := rt.convertCallParameter(a, typ.In(n)); err == nil {
							in[i] = v
							callSlice = true
							break
						}
					}

					v, err := rt.convertCallParameter(a, t)
					if err != nil {
						panic(rt.panicTypeError(err.Error()))
					}

					in[i] = v
				}

				var out []reflect.Value
				if callSlice {
					out = val.CallSlice(in)
				} else {
					out = val.Call(in)
				}

				switch len(out) {
				case 0:
					return Value{}
				case 1:
					return rt.toValue(out[0].Interface())
				default:
					s := make([]interface{}, len(out))
					for i, v := range out {
						s[i] = rt.toValue(v.Interface())
					}

					return rt.toValue(s)
				}
			}))
		}
	}

	return toValue(value)
}

func (rt *runtime) newGoSlice(value reflect.Value) *object {
	obj := rt.newGoSliceObject(value)
	obj.prototype = rt.global.ArrayPrototype
	return obj
}

func (rt *runtime) newGoArray(value reflect.Value) *object {
	obj := rt.newGoArrayObject(value)
	obj.prototype = rt.global.ArrayPrototype
	return obj
}

func (rt *runtime) parse(filename string, src, sm interface{}) (*ast.Program, error) {
	return parser.ParseFileWithSourceMap(nil, filename, src, sm, 0)
}

func (rt *runtime) cmplParse(filename string, src, sm interface{}) (*nodeProgram, error) {
	program, err := parser.ParseFileWithSourceMap(nil, filename, src, sm, 0)
	if err != nil {
		return nil, err
	}

	return cmplParse(program), nil
}

func (rt *runtime) parseSource(src, sm interface{}) (*nodeProgram, *ast.Program, error) {
	switch src := src.(type) {
	case *ast.Program:
		return nil, src, nil
	case *Script:
		return src.program, nil, nil
	}

	program, err := rt.parse("", src, sm)

	return nil, program, err
}

func (rt *runtime) cmplRunOrEval(src, sm interface{}, eval bool) (Value, error) {
	result := Value{}
	node, program, err := rt.parseSource(src, sm)
	if err != nil {
		return result, err
	}
	if node == nil {
		node = cmplParse(program)
	}
	err = catchPanic(func() {
		result = rt.cmplEvaluateNodeProgram(node, eval)
	})
	switch result.kind {
	case valueEmpty:
		result = Value{}
	case valueReference:
		result = result.resolve()
	}
	return result, err
}

func (rt *runtime) cmplRun(src, sm interface{}) (Value, error) {
	return rt.cmplRunOrEval(src, sm, false)
}

func (rt *runtime) cmplEval(src, sm interface{}) (Value, error) {
	return rt.cmplRunOrEval(src, sm, true)
}

func (rt *runtime) parseThrow(err error) {
	if err == nil {
		return
	}

	var errl parser.ErrorList
	if errors.Is(err, &errl) {
		err := errl[0]
		if err.Message == "invalid left-hand side in assignment" {
			panic(rt.panicReferenceError(err.Message))
		}
		panic(rt.panicSyntaxError(err.Message))
	}
	panic(rt.panicSyntaxError(err.Error()))
}

func (rt *runtime) cmplParseOrThrow(src, sm interface{}) *nodeProgram {
	program, err := rt.cmplParse("", src, sm)
	rt.parseThrow(err) // Will panic/throw appropriately
	return program
}
