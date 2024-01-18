package vm

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/expr-lang/expr/builtin"
	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/vm/runtime"
)

// Program represents a compiled expression.
type Program struct {
	Bytecode  []Opcode
	Arguments []int
	Constants []any

	source    *file.Source
	locations []file.Location
	variables []any
	functions []Function
	debugInfo map[string]string
}

// NewProgram returns a new Program. It's used by the compiler.
func NewProgram(
	source *file.Source,
	locations []file.Location,
	variables []any,
	constants []any,
	bytecode []Opcode,
	arguments []int,
	functions []Function,
	debugInfo map[string]string,
) *Program {
	return &Program{
		source:    source,
		locations: locations,
		variables: variables,
		Constants: constants,
		Bytecode:  bytecode,
		Arguments: arguments,
		functions: functions,
		debugInfo: debugInfo,
	}
}

// Disassemble returns opcodes as a string.
func (program *Program) Disassemble() string {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	program.DisassembleWriter(w)
	_ = w.Flush()
	return buf.String()
}

// DisassembleWriter takes a writer and writes opcodes to it.
func (program *Program) DisassembleWriter(w io.Writer) {
	ip := 0
	for ip < len(program.Bytecode) {
		pp := ip
		op := program.Bytecode[ip]
		arg := program.Arguments[ip]
		ip += 1

		code := func(label string) {
			_, _ = fmt.Fprintf(w, "%v\t%v\n", pp, label)
		}
		jump := func(label string) {
			_, _ = fmt.Fprintf(w, "%v\t%v\t<%v>\t(%v)\n", pp, label, arg, ip+arg)
		}
		jumpBack := func(label string) {
			_, _ = fmt.Fprintf(w, "%v\t%v\t<%v>\t(%v)\n", pp, label, arg, ip-arg)
		}
		argument := func(label string) {
			_, _ = fmt.Fprintf(w, "%v\t%v\t<%v>\n", pp, label, arg)
		}
		argumentWithInfo := func(label string, prefix string) {
			_, _ = fmt.Fprintf(w, "%v\t%v\t<%v>\t%v\n", pp, label, arg, program.debugInfo[fmt.Sprintf("%s_%d", prefix, arg)])
		}
		constant := func(label string) {
			var c any
			if arg < len(program.Constants) {
				c = program.Constants[arg]
			} else {
				c = "out of range"
			}
			if r, ok := c.(*regexp.Regexp); ok {
				c = r.String()
			}
			if field, ok := c.(*runtime.Field); ok {
				c = fmt.Sprintf("{%v %v}", strings.Join(field.Path, "."), field.Index)
			}
			if method, ok := c.(*runtime.Method); ok {
				c = fmt.Sprintf("{%v %v}", method.Name, method.Index)
			}
			_, _ = fmt.Fprintf(w, "%v\t%v\t<%v>\t%v\n", pp, label, arg, c)
		}
		builtinArg := func(label string) {
			_, _ = fmt.Fprintf(w, "%v\t%v\t<%v>\t%v\n", pp, label, arg, builtin.Builtins[arg].Name)
		}

		switch op {
		case OpInvalid:
			code("OpInvalid")

		case OpPush:
			constant("OpPush")

		case OpInt:
			argument("OpInt")

		case OpPop:
			code("OpPop")

		case OpStore:
			argumentWithInfo("OpStore", "var")

		case OpLoadVar:
			argumentWithInfo("OpLoadVar", "var")

		case OpLoadConst:
			constant("OpLoadConst")

		case OpLoadField:
			constant("OpLoadField")

		case OpLoadFast:
			constant("OpLoadFast")

		case OpLoadMethod:
			constant("OpLoadMethod")

		case OpLoadFunc:
			argumentWithInfo("OpLoadFunc", "func")

		case OpLoadEnv:
			code("OpLoadEnv")

		case OpFetch:
			code("OpFetch")

		case OpFetchField:
			constant("OpFetchField")

		case OpMethod:
			constant("OpMethod")

		case OpTrue:
			code("OpTrue")

		case OpFalse:
			code("OpFalse")

		case OpNil:
			code("OpNil")

		case OpNegate:
			code("OpNegate")

		case OpNot:
			code("OpNot")

		case OpEqual:
			code("OpEqual")

		case OpEqualInt:
			code("OpEqualInt")

		case OpEqualString:
			code("OpEqualString")

		case OpJump:
			jump("OpJump")

		case OpJumpIfTrue:
			jump("OpJumpIfTrue")

		case OpJumpIfFalse:
			jump("OpJumpIfFalse")

		case OpJumpIfNil:
			jump("OpJumpIfNil")

		case OpJumpIfNotNil:
			jump("OpJumpIfNotNil")

		case OpJumpIfEnd:
			jump("OpJumpIfEnd")

		case OpJumpBackward:
			jumpBack("OpJumpBackward")

		case OpIn:
			code("OpIn")

		case OpLess:
			code("OpLess")

		case OpMore:
			code("OpMore")

		case OpLessOrEqual:
			code("OpLessOrEqual")

		case OpMoreOrEqual:
			code("OpMoreOrEqual")

		case OpAdd:
			code("OpAdd")

		case OpSubtract:
			code("OpSubtract")

		case OpMultiply:
			code("OpMultiply")

		case OpDivide:
			code("OpDivide")

		case OpModulo:
			code("OpModulo")

		case OpExponent:
			code("OpExponent")

		case OpRange:
			code("OpRange")

		case OpMatches:
			code("OpMatches")

		case OpMatchesConst:
			constant("OpMatchesConst")

		case OpContains:
			code("OpContains")

		case OpStartsWith:
			code("OpStartsWith")

		case OpEndsWith:
			code("OpEndsWith")

		case OpSlice:
			code("OpSlice")

		case OpCall:
			argument("OpCall")

		case OpCall0:
			argumentWithInfo("OpCall0", "func")

		case OpCall1:
			argumentWithInfo("OpCall1", "func")

		case OpCall2:
			argumentWithInfo("OpCall2", "func")

		case OpCall3:
			argumentWithInfo("OpCall3", "func")

		case OpCallN:
			argument("OpCallN")

		case OpCallFast:
			argument("OpCallFast")

		case OpCallTyped:
			signature := reflect.TypeOf(FuncTypes[arg]).Elem().String()
			_, _ = fmt.Fprintf(w, "%v\t%v\t<%v>\t%v\n", pp, "OpCallTyped", arg, signature)

		case OpCallBuiltin1:
			builtinArg("OpCallBuiltin1")

		case OpValidateArgs:
			argument("OpValidateArgs")

		case OpArray:
			code("OpArray")

		case OpMap:
			code("OpMap")

		case OpLen:
			code("OpLen")

		case OpCast:
			argument("OpCast")

		case OpDeref:
			code("OpDeref")

		case OpIncrementIndex:
			code("OpIncrementIndex")

		case OpDecrementIndex:
			code("OpDecrementIndex")

		case OpIncrementCount:
			code("OpIncrementCount")

		case OpGetIndex:
			code("OpGetIndex")

		case OpSetIndex:
			code("OpSetIndex")

		case OpGetCount:
			code("OpGetCount")

		case OpGetLen:
			code("OpGetLen")

		case OpGetGroupBy:
			code("OpGetGroupBy")

		case OpGetAcc:
			code("OpGetAcc")

		case OpPointer:
			code("OpPointer")

		case OpThrow:
			code("OpThrow")

		case OpGroupBy:
			code("OpGroupBy")

		case OpSetAcc:
			code("OpSetAcc")

		case OpBegin:
			code("OpBegin")

		case OpEnd:
			code("OpEnd")

		default:
			_, _ = fmt.Fprintf(w, "%v\t%#x (unknown)\n", ip, op)
		}
	}
}

// Source returns origin file.Source.
func (program *Program) Source() *file.Source {
	return program.source
}
