package vm

//go:generate sh -c "go run ./func_types > ./generated.go"

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/expr-lang/expr/builtin"
	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/vm/runtime"
)

var MemoryBudget uint = 1e6
var errorType = reflect.TypeOf((*error)(nil)).Elem()

type Function = func(params ...any) (any, error)

func Run(program *Program, env any) (any, error) {
	if program == nil {
		return nil, fmt.Errorf("program is nil")
	}

	vm := VM{}
	return vm.Run(program, env)
}

type VM struct {
	stack        []any
	ip           int
	scopes       []*Scope
	debug        bool
	step         chan struct{}
	curr         chan int
	memory       uint
	memoryBudget uint
}

type Scope struct {
	Array   reflect.Value
	Index   int
	Len     int
	Count   int
	GroupBy map[any][]any
	Acc     any
}

func Debug() *VM {
	vm := &VM{
		debug: true,
		step:  make(chan struct{}, 0),
		curr:  make(chan int, 0),
	}
	return vm
}

func (vm *VM) Run(program *Program, env any) (_ any, err error) {
	defer func() {
		if r := recover(); r != nil {
			var location file.Location
			if vm.ip-1 < len(program.locations) {
				location = program.locations[vm.ip-1]
			}
			f := &file.Error{
				Location: location,
				Message:  fmt.Sprintf("%v", r),
			}
			if err, ok := r.(error); ok {
				f.Wrap(err)
			}
			err = f.Bind(program.source)
		}
	}()

	if vm.stack == nil {
		vm.stack = make([]any, 0, 2)
	} else {
		vm.stack = vm.stack[0:0]
	}

	if vm.scopes != nil {
		vm.scopes = vm.scopes[0:0]
	}

	vm.memoryBudget = MemoryBudget
	vm.memory = 0
	vm.ip = 0

	for vm.ip < len(program.Bytecode) {
		if vm.debug {
			<-vm.step
		}

		op := program.Bytecode[vm.ip]
		arg := program.Arguments[vm.ip]
		vm.ip += 1

		switch op {

		case OpInvalid:
			panic("invalid opcode")

		case OpPush:
			vm.push(program.Constants[arg])

		case OpInt:
			vm.push(arg)

		case OpPop:
			vm.pop()

		case OpStore:
			program.variables[arg] = vm.pop()

		case OpLoadVar:
			vm.push(program.variables[arg])

		case OpLoadConst:
			vm.push(runtime.Fetch(env, program.Constants[arg]))

		case OpLoadField:
			vm.push(runtime.FetchField(env, program.Constants[arg].(*runtime.Field)))

		case OpLoadFast:
			vm.push(env.(map[string]any)[program.Constants[arg].(string)])

		case OpLoadMethod:
			vm.push(runtime.FetchMethod(env, program.Constants[arg].(*runtime.Method)))

		case OpLoadFunc:
			vm.push(program.functions[arg])

		case OpFetch:
			b := vm.pop()
			a := vm.pop()
			vm.push(runtime.Fetch(a, b))

		case OpFetchField:
			a := vm.pop()
			vm.push(runtime.FetchField(a, program.Constants[arg].(*runtime.Field)))

		case OpLoadEnv:
			vm.push(env)

		case OpMethod:
			a := vm.pop()
			vm.push(runtime.FetchMethod(a, program.Constants[arg].(*runtime.Method)))

		case OpTrue:
			vm.push(true)

		case OpFalse:
			vm.push(false)

		case OpNil:
			vm.push(nil)

		case OpNegate:
			v := runtime.Negate(vm.pop())
			vm.push(v)

		case OpNot:
			v := vm.pop().(bool)
			vm.push(!v)

		case OpEqual:
			b := vm.pop()
			a := vm.pop()
			vm.push(runtime.Equal(a, b))

		case OpEqualInt:
			b := vm.pop()
			a := vm.pop()
			vm.push(a.(int) == b.(int))

		case OpEqualString:
			b := vm.pop()
			a := vm.pop()
			vm.push(a.(string) == b.(string))

		case OpJump:
			vm.ip += arg

		case OpJumpIfTrue:
			if vm.current().(bool) {
				vm.ip += arg
			}

		case OpJumpIfFalse:
			if !vm.current().(bool) {
				vm.ip += arg
			}

		case OpJumpIfNil:
			if runtime.IsNil(vm.current()) {
				vm.ip += arg
			}

		case OpJumpIfNotNil:
			if !runtime.IsNil(vm.current()) {
				vm.ip += arg
			}

		case OpJumpIfEnd:
			scope := vm.Scope()
			if scope.Index >= scope.Len {
				vm.ip += arg
			}

		case OpJumpBackward:
			vm.ip -= arg

		case OpIn:
			b := vm.pop()
			a := vm.pop()
			vm.push(runtime.In(a, b))

		case OpLess:
			b := vm.pop()
			a := vm.pop()
			vm.push(runtime.Less(a, b))

		case OpMore:
			b := vm.pop()
			a := vm.pop()
			vm.push(runtime.More(a, b))

		case OpLessOrEqual:
			b := vm.pop()
			a := vm.pop()
			vm.push(runtime.LessOrEqual(a, b))

		case OpMoreOrEqual:
			b := vm.pop()
			a := vm.pop()
			vm.push(runtime.MoreOrEqual(a, b))

		case OpAdd:
			b := vm.pop()
			a := vm.pop()
			vm.push(runtime.Add(a, b))

		case OpSubtract:
			b := vm.pop()
			a := vm.pop()
			vm.push(runtime.Subtract(a, b))

		case OpMultiply:
			b := vm.pop()
			a := vm.pop()
			vm.push(runtime.Multiply(a, b))

		case OpDivide:
			b := vm.pop()
			a := vm.pop()
			vm.push(runtime.Divide(a, b))

		case OpModulo:
			b := vm.pop()
			a := vm.pop()
			vm.push(runtime.Modulo(a, b))

		case OpExponent:
			b := vm.pop()
			a := vm.pop()
			vm.push(runtime.Exponent(a, b))

		case OpRange:
			b := vm.pop()
			a := vm.pop()
			min := runtime.ToInt(a)
			max := runtime.ToInt(b)
			size := max - min + 1
			if size <= 0 {
				size = 0
			}
			vm.memGrow(uint(size))
			vm.push(runtime.MakeRange(min, max))

		case OpMatches:
			b := vm.pop()
			a := vm.pop()
			match, err := regexp.MatchString(b.(string), a.(string))
			if err != nil {
				panic(err)
			}

			vm.push(match)

		case OpMatchesConst:
			a := vm.pop()
			r := program.Constants[arg].(*regexp.Regexp)
			vm.push(r.MatchString(a.(string)))

		case OpContains:
			b := vm.pop()
			a := vm.pop()
			vm.push(strings.Contains(a.(string), b.(string)))

		case OpStartsWith:
			b := vm.pop()
			a := vm.pop()
			vm.push(strings.HasPrefix(a.(string), b.(string)))

		case OpEndsWith:
			b := vm.pop()
			a := vm.pop()
			vm.push(strings.HasSuffix(a.(string), b.(string)))

		case OpSlice:
			from := vm.pop()
			to := vm.pop()
			node := vm.pop()
			vm.push(runtime.Slice(node, from, to))

		case OpCall:
			fn := reflect.ValueOf(vm.pop())
			size := arg
			in := make([]reflect.Value, size)
			for i := int(size) - 1; i >= 0; i-- {
				param := vm.pop()
				if param == nil && reflect.TypeOf(param) == nil {
					// In case of nil value and nil type use this hack,
					// otherwise reflect.Call will panic on zero value.
					in[i] = reflect.ValueOf(&param).Elem()
				} else {
					in[i] = reflect.ValueOf(param)
				}
			}
			out := fn.Call(in)
			if len(out) == 2 && out[1].Type() == errorType && !out[1].IsNil() {
				panic(out[1].Interface().(error))
			}
			vm.push(out[0].Interface())

		case OpCall0:
			out, err := program.functions[arg]()
			if err != nil {
				panic(err)
			}
			vm.push(out)

		case OpCall1:
			a := vm.pop()
			out, err := program.functions[arg](a)
			if err != nil {
				panic(err)
			}
			vm.push(out)

		case OpCall2:
			b := vm.pop()
			a := vm.pop()
			out, err := program.functions[arg](a, b)
			if err != nil {
				panic(err)
			}
			vm.push(out)

		case OpCall3:
			c := vm.pop()
			b := vm.pop()
			a := vm.pop()
			out, err := program.functions[arg](a, b, c)
			if err != nil {
				panic(err)
			}
			vm.push(out)

		case OpCallN:
			fn := vm.pop().(Function)
			size := arg
			in := make([]any, size)
			for i := int(size) - 1; i >= 0; i-- {
				in[i] = vm.pop()
			}
			out, err := fn(in...)
			if err != nil {
				panic(err)
			}
			vm.push(out)

		case OpCallFast:
			fn := vm.pop().(func(...any) any)
			size := arg
			in := make([]any, size)
			for i := int(size) - 1; i >= 0; i-- {
				in[i] = vm.pop()
			}
			vm.push(fn(in...))

		case OpCallTyped:
			vm.push(vm.call(vm.pop(), arg))

		case OpCallBuiltin1:
			vm.push(builtin.Builtins[arg].Fast(vm.pop()))

		case OpValidateArgs:
			fn := vm.pop().(Function)
			mem, err := fn(vm.stack[len(vm.stack)-arg:]...)
			if err != nil {
				panic(err)
			}
			vm.memGrow(mem.(uint))

		case OpArray:
			size := vm.pop().(int)
			vm.memGrow(uint(size))
			array := make([]any, size)
			for i := size - 1; i >= 0; i-- {
				array[i] = vm.pop()
			}
			vm.push(array)

		case OpMap:
			size := vm.pop().(int)
			vm.memGrow(uint(size))
			m := make(map[string]any)
			for i := size - 1; i >= 0; i-- {
				value := vm.pop()
				key := vm.pop()
				m[key.(string)] = value
			}
			vm.push(m)

		case OpLen:
			vm.push(runtime.Len(vm.current()))

		case OpCast:
			switch arg {
			case 0:
				vm.push(runtime.ToInt(vm.pop()))
			case 1:
				vm.push(runtime.ToInt64(vm.pop()))
			case 2:
				vm.push(runtime.ToFloat64(vm.pop()))
			}

		case OpDeref:
			a := vm.pop()
			vm.push(runtime.Deref(a))

		case OpIncrementIndex:
			vm.Scope().Index++

		case OpDecrementIndex:
			scope := vm.Scope()
			scope.Index--

		case OpIncrementCount:
			scope := vm.Scope()
			scope.Count++

		case OpGetIndex:
			vm.push(vm.Scope().Index)

		case OpSetIndex:
			scope := vm.Scope()
			scope.Index = vm.pop().(int)

		case OpGetCount:
			scope := vm.Scope()
			vm.push(scope.Count)

		case OpGetLen:
			scope := vm.Scope()
			vm.push(scope.Len)

		case OpGetGroupBy:
			vm.push(vm.Scope().GroupBy)

		case OpGetAcc:
			vm.push(vm.Scope().Acc)

		case OpSetAcc:
			vm.Scope().Acc = vm.pop()

		case OpPointer:
			scope := vm.Scope()
			vm.push(scope.Array.Index(scope.Index).Interface())

		case OpThrow:
			panic(vm.pop().(error))

		case OpGroupBy:
			scope := vm.Scope()
			if scope.GroupBy == nil {
				scope.GroupBy = make(map[any][]any)
			}
			it := scope.Array.Index(scope.Index).Interface()
			key := vm.pop()
			scope.GroupBy[key] = append(scope.GroupBy[key], it)

		case OpBegin:
			a := vm.pop()
			array := reflect.ValueOf(a)
			vm.scopes = append(vm.scopes, &Scope{
				Array: array,
				Len:   array.Len(),
			})

		case OpEnd:
			vm.scopes = vm.scopes[:len(vm.scopes)-1]

		default:
			panic(fmt.Sprintf("unknown bytecode %#x", op))
		}

		if vm.debug {
			vm.curr <- vm.ip
		}
	}

	if vm.debug {
		close(vm.curr)
		close(vm.step)
	}

	if len(vm.stack) > 0 {
		return vm.pop(), nil
	}

	return nil, nil
}

func (vm *VM) push(value any) {
	vm.stack = append(vm.stack, value)
}

func (vm *VM) current() any {
	return vm.stack[len(vm.stack)-1]
}

func (vm *VM) pop() any {
	value := vm.stack[len(vm.stack)-1]
	vm.stack = vm.stack[:len(vm.stack)-1]
	return value
}

func (vm *VM) memGrow(size uint) {
	vm.memory += size
	if vm.memory >= vm.memoryBudget {
		panic("memory budget exceeded")
	}
}

func (vm *VM) Stack() []any {
	return vm.stack
}

func (vm *VM) Scope() *Scope {
	if len(vm.scopes) > 0 {
		return vm.scopes[len(vm.scopes)-1]
	}
	return nil
}

func (vm *VM) Step() {
	vm.step <- struct{}{}
}

func (vm *VM) Position() chan int {
	return vm.curr
}
