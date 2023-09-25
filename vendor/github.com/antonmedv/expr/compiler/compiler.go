package compiler

import (
	"fmt"
	"reflect"

	"github.com/antonmedv/expr/ast"
	"github.com/antonmedv/expr/builtin"
	"github.com/antonmedv/expr/conf"
	"github.com/antonmedv/expr/file"
	"github.com/antonmedv/expr/parser"
	. "github.com/antonmedv/expr/vm"
	"github.com/antonmedv/expr/vm/runtime"
)

const (
	placeholder = 12345
)

func Compile(tree *parser.Tree, config *conf.Config) (program *Program, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	c := &compiler{
		locations:      make([]file.Location, 0),
		constantsIndex: make(map[any]int),
		functionsIndex: make(map[string]int),
		debugInfo:      make(map[string]string),
	}

	if config != nil {
		c.mapEnv = config.MapEnv
		c.cast = config.Expect
	}

	c.compile(tree.Node)

	switch c.cast {
	case reflect.Int:
		c.emit(OpCast, 0)
	case reflect.Int64:
		c.emit(OpCast, 1)
	case reflect.Float64:
		c.emit(OpCast, 2)
	}

	program = &Program{
		Node:      tree.Node,
		Source:    tree.Source,
		Locations: c.locations,
		Variables: c.variables,
		Constants: c.constants,
		Bytecode:  c.bytecode,
		Arguments: c.arguments,
		Functions: c.functions,
		DebugInfo: c.debugInfo,
	}
	return
}

type compiler struct {
	locations      []file.Location
	bytecode       []Opcode
	variables      []any
	scopes         []scope
	constants      []any
	constantsIndex map[any]int
	functions      []Function
	functionsIndex map[string]int
	debugInfo      map[string]string
	mapEnv         bool
	cast           reflect.Kind
	nodes          []ast.Node
	chains         [][]int
	arguments      []int
}

type scope struct {
	variableName string
	index        int
}

func (c *compiler) emitLocation(loc file.Location, op Opcode, arg int) int {
	c.bytecode = append(c.bytecode, op)
	current := len(c.bytecode)
	c.arguments = append(c.arguments, arg)
	c.locations = append(c.locations, loc)
	return current
}

func (c *compiler) emit(op Opcode, args ...int) int {
	arg := 0
	if len(args) > 1 {
		panic("too many arguments")
	}
	if len(args) == 1 {
		arg = args[0]
	}
	var loc file.Location
	if len(c.nodes) > 0 {
		loc = c.nodes[len(c.nodes)-1].Location()
	}
	return c.emitLocation(loc, op, arg)
}

func (c *compiler) emitPush(value any) int {
	return c.emit(OpPush, c.addConstant(value))
}

func (c *compiler) addConstant(constant any) int {
	indexable := true
	hash := constant
	switch reflect.TypeOf(constant).Kind() {
	case reflect.Slice, reflect.Map, reflect.Struct:
		indexable = false
	}
	if field, ok := constant.(*runtime.Field); ok {
		indexable = true
		hash = fmt.Sprintf("%v", field)
	}
	if method, ok := constant.(*runtime.Method); ok {
		indexable = true
		hash = fmt.Sprintf("%v", method)
	}
	if indexable {
		if p, ok := c.constantsIndex[hash]; ok {
			return p
		}
	}
	c.constants = append(c.constants, constant)
	p := len(c.constants) - 1
	if indexable {
		c.constantsIndex[hash] = p
	}
	return p
}

func (c *compiler) addVariable(name string) int {
	c.variables = append(c.variables, nil)
	p := len(c.variables) - 1
	c.debugInfo[fmt.Sprintf("var_%d", p)] = name
	return p
}

// emitFunction adds builtin.Function.Func to the program.Functions and emits call opcode.
func (c *compiler) emitFunction(fn *ast.Function, argsLen int) {
	switch argsLen {
	case 0:
		c.emit(OpCall0, c.addFunction(fn))
	case 1:
		c.emit(OpCall1, c.addFunction(fn))
	case 2:
		c.emit(OpCall2, c.addFunction(fn))
	case 3:
		c.emit(OpCall3, c.addFunction(fn))
	default:
		c.emit(OpLoadFunc, c.addFunction(fn))
		c.emit(OpCallN, argsLen)
	}
}

// addFunction adds builtin.Function.Func to the program.Functions and returns its index.
func (c *compiler) addFunction(fn *ast.Function) int {
	if fn == nil {
		panic("function is nil")
	}
	if p, ok := c.functionsIndex[fn.Name]; ok {
		return p
	}
	p := len(c.functions)
	c.functions = append(c.functions, fn.Func)
	c.functionsIndex[fn.Name] = p
	c.debugInfo[fmt.Sprintf("func_%d", p)] = fn.Name
	return p
}

func (c *compiler) patchJump(placeholder int) {
	offset := len(c.bytecode) - placeholder
	c.arguments[placeholder-1] = offset
}

func (c *compiler) calcBackwardJump(to int) int {
	return len(c.bytecode) + 1 - to
}

func (c *compiler) compile(node ast.Node) {
	c.nodes = append(c.nodes, node)
	defer func() {
		c.nodes = c.nodes[:len(c.nodes)-1]
	}()

	switch n := node.(type) {
	case *ast.NilNode:
		c.NilNode(n)
	case *ast.IdentifierNode:
		c.IdentifierNode(n)
	case *ast.IntegerNode:
		c.IntegerNode(n)
	case *ast.FloatNode:
		c.FloatNode(n)
	case *ast.BoolNode:
		c.BoolNode(n)
	case *ast.StringNode:
		c.StringNode(n)
	case *ast.ConstantNode:
		c.ConstantNode(n)
	case *ast.UnaryNode:
		c.UnaryNode(n)
	case *ast.BinaryNode:
		c.BinaryNode(n)
	case *ast.ChainNode:
		c.ChainNode(n)
	case *ast.MemberNode:
		c.MemberNode(n)
	case *ast.SliceNode:
		c.SliceNode(n)
	case *ast.CallNode:
		c.CallNode(n)
	case *ast.BuiltinNode:
		c.BuiltinNode(n)
	case *ast.ClosureNode:
		c.ClosureNode(n)
	case *ast.PointerNode:
		c.PointerNode(n)
	case *ast.VariableDeclaratorNode:
		c.VariableDeclaratorNode(n)
	case *ast.ConditionalNode:
		c.ConditionalNode(n)
	case *ast.ArrayNode:
		c.ArrayNode(n)
	case *ast.MapNode:
		c.MapNode(n)
	case *ast.PairNode:
		c.PairNode(n)
	default:
		panic(fmt.Sprintf("undefined node type (%T)", node))
	}
}

func (c *compiler) NilNode(_ *ast.NilNode) {
	c.emit(OpNil)
}

func (c *compiler) IdentifierNode(node *ast.IdentifierNode) {
	if index, ok := c.lookupVariable(node.Value); ok {
		c.emit(OpLoadVar, index)
		return
	}
	if node.Value == "$env" {
		c.emit(OpLoadEnv)
		return
	}
	if c.mapEnv {
		c.emit(OpLoadFast, c.addConstant(node.Value))
	} else if len(node.FieldIndex) > 0 {
		c.emit(OpLoadField, c.addConstant(&runtime.Field{
			Index: node.FieldIndex,
			Path:  []string{node.Value},
		}))
	} else if node.Method {
		c.emit(OpLoadMethod, c.addConstant(&runtime.Method{
			Name:  node.Value,
			Index: node.MethodIndex,
		}))
	} else {
		c.emit(OpLoadConst, c.addConstant(node.Value))
	}
}

func (c *compiler) IntegerNode(node *ast.IntegerNode) {
	t := node.Type()
	if t == nil {
		c.emitPush(node.Value)
		return
	}
	switch t.Kind() {
	case reflect.Float32:
		c.emitPush(float32(node.Value))
	case reflect.Float64:
		c.emitPush(float64(node.Value))
	case reflect.Int:
		c.emitPush(node.Value)
	case reflect.Int8:
		c.emitPush(int8(node.Value))
	case reflect.Int16:
		c.emitPush(int16(node.Value))
	case reflect.Int32:
		c.emitPush(int32(node.Value))
	case reflect.Int64:
		c.emitPush(int64(node.Value))
	case reflect.Uint:
		c.emitPush(uint(node.Value))
	case reflect.Uint8:
		c.emitPush(uint8(node.Value))
	case reflect.Uint16:
		c.emitPush(uint16(node.Value))
	case reflect.Uint32:
		c.emitPush(uint32(node.Value))
	case reflect.Uint64:
		c.emitPush(uint64(node.Value))
	default:
		c.emitPush(node.Value)
	}
}

func (c *compiler) FloatNode(node *ast.FloatNode) {
	t := node.Type()
	if t == nil {
		c.emitPush(node.Value)
		return
	}
	switch t.Kind() {
	case reflect.Float32:
		c.emitPush(float32(node.Value))
	case reflect.Float64:
		c.emitPush(node.Value)
	}
}

func (c *compiler) BoolNode(node *ast.BoolNode) {
	if node.Value {
		c.emit(OpTrue)
	} else {
		c.emit(OpFalse)
	}
}

func (c *compiler) StringNode(node *ast.StringNode) {
	c.emitPush(node.Value)
}

func (c *compiler) ConstantNode(node *ast.ConstantNode) {
	c.emitPush(node.Value)
}

func (c *compiler) UnaryNode(node *ast.UnaryNode) {
	c.compile(node.Node)
	c.derefInNeeded(node.Node)

	switch node.Operator {

	case "!", "not":
		c.emit(OpNot)

	case "+":
		// Do nothing

	case "-":
		c.emit(OpNegate)

	default:
		panic(fmt.Sprintf("unknown operator (%v)", node.Operator))
	}
}

func (c *compiler) BinaryNode(node *ast.BinaryNode) {
	l := kind(node.Left)
	r := kind(node.Right)

	switch node.Operator {
	case "==":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		c.compile(node.Right)
		c.derefInNeeded(node.Left)

		if l == r && l == reflect.Int {
			c.emit(OpEqualInt)
		} else if l == r && l == reflect.String {
			c.emit(OpEqualString)
		} else {
			c.emit(OpEqual)
		}

	case "!=":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		c.compile(node.Right)
		c.derefInNeeded(node.Left)
		c.emit(OpEqual)
		c.emit(OpNot)

	case "or", "||":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		end := c.emit(OpJumpIfTrue, placeholder)
		c.emit(OpPop)
		c.compile(node.Right)
		c.derefInNeeded(node.Right)
		c.patchJump(end)

	case "and", "&&":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		end := c.emit(OpJumpIfFalse, placeholder)
		c.emit(OpPop)
		c.compile(node.Right)
		c.derefInNeeded(node.Right)
		c.patchJump(end)

	case "<":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		c.compile(node.Right)
		c.derefInNeeded(node.Right)
		c.emit(OpLess)

	case ">":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		c.compile(node.Right)
		c.derefInNeeded(node.Right)
		c.emit(OpMore)

	case "<=":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		c.compile(node.Right)
		c.derefInNeeded(node.Right)
		c.emit(OpLessOrEqual)

	case ">=":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		c.compile(node.Right)
		c.derefInNeeded(node.Right)
		c.emit(OpMoreOrEqual)

	case "+":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		c.compile(node.Right)
		c.derefInNeeded(node.Right)
		c.emit(OpAdd)

	case "-":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		c.compile(node.Right)
		c.derefInNeeded(node.Right)
		c.emit(OpSubtract)

	case "*":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		c.compile(node.Right)
		c.derefInNeeded(node.Right)
		c.emit(OpMultiply)

	case "/":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		c.compile(node.Right)
		c.derefInNeeded(node.Right)
		c.emit(OpDivide)

	case "%":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		c.compile(node.Right)
		c.derefInNeeded(node.Right)
		c.emit(OpModulo)

	case "**", "^":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		c.compile(node.Right)
		c.derefInNeeded(node.Right)
		c.emit(OpExponent)

	case "in":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		c.compile(node.Right)
		c.derefInNeeded(node.Right)
		c.emit(OpIn)

	case "matches":
		if node.Regexp != nil {
			c.compile(node.Left)
			c.derefInNeeded(node.Left)
			c.emit(OpMatchesConst, c.addConstant(node.Regexp))
		} else {
			c.compile(node.Left)
			c.derefInNeeded(node.Left)
			c.compile(node.Right)
			c.derefInNeeded(node.Right)
			c.emit(OpMatches)
		}

	case "contains":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		c.compile(node.Right)
		c.derefInNeeded(node.Right)
		c.emit(OpContains)

	case "startsWith":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		c.compile(node.Right)
		c.derefInNeeded(node.Right)
		c.emit(OpStartsWith)

	case "endsWith":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		c.compile(node.Right)
		c.derefInNeeded(node.Right)
		c.emit(OpEndsWith)

	case "..":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		c.compile(node.Right)
		c.derefInNeeded(node.Right)
		c.emit(OpRange)

	case "??":
		c.compile(node.Left)
		c.derefInNeeded(node.Left)
		end := c.emit(OpJumpIfNotNil, placeholder)
		c.emit(OpPop)
		c.compile(node.Right)
		c.derefInNeeded(node.Right)
		c.patchJump(end)

	default:
		panic(fmt.Sprintf("unknown operator (%v)", node.Operator))

	}
}

func (c *compiler) ChainNode(node *ast.ChainNode) {
	c.chains = append(c.chains, []int{})
	c.compile(node.Node)
	// Chain activate (got nit somewhere)
	for _, ph := range c.chains[len(c.chains)-1] {
		c.patchJump(ph)
	}
	c.chains = c.chains[:len(c.chains)-1]
}

func (c *compiler) MemberNode(node *ast.MemberNode) {
	if node.Method {
		c.compile(node.Node)
		c.emit(OpMethod, c.addConstant(&runtime.Method{
			Name:  node.Name,
			Index: node.MethodIndex,
		}))
		return
	}
	op := OpFetch
	index := node.FieldIndex
	path := []string{node.Name}
	base := node.Node
	if len(node.FieldIndex) > 0 {
		op = OpFetchField
		for !node.Optional {
			ident, ok := base.(*ast.IdentifierNode)
			if ok && len(ident.FieldIndex) > 0 {
				index = append(ident.FieldIndex, index...)
				path = append([]string{ident.Value}, path...)
				c.emitLocation(ident.Location(), OpLoadField, c.addConstant(
					&runtime.Field{Index: index, Path: path},
				))
				return
			}
			member, ok := base.(*ast.MemberNode)
			if ok && len(member.FieldIndex) > 0 {
				index = append(member.FieldIndex, index...)
				path = append([]string{member.Name}, path...)
				node = member
				base = member.Node
			} else {
				break
			}
		}
	}

	c.compile(base)
	if node.Optional {
		ph := c.emit(OpJumpIfNil, placeholder)
		c.chains[len(c.chains)-1] = append(c.chains[len(c.chains)-1], ph)
	}

	if op == OpFetch {
		c.compile(node.Property)
		c.emit(OpFetch)
	} else {
		c.emitLocation(node.Location(), op, c.addConstant(
			&runtime.Field{Index: index, Path: path},
		))
	}
}

func (c *compiler) SliceNode(node *ast.SliceNode) {
	c.compile(node.Node)
	if node.To != nil {
		c.compile(node.To)
	} else {
		c.emit(OpLen)
	}
	if node.From != nil {
		c.compile(node.From)
	} else {
		c.emitPush(0)
	}
	c.emit(OpSlice)
}

func (c *compiler) CallNode(node *ast.CallNode) {
	for _, arg := range node.Arguments {
		c.compile(arg)
	}
	if node.Func != nil {
		c.emitFunction(node.Func, len(node.Arguments))
		return
	}
	c.compile(node.Callee)
	if node.Typed > 0 {
		c.emit(OpCallTyped, node.Typed)
		return
	} else if node.Fast {
		c.emit(OpCallFast, len(node.Arguments))
	} else {
		c.emit(OpCall, len(node.Arguments))
	}
}

func (c *compiler) BuiltinNode(node *ast.BuiltinNode) {
	switch node.Name {
	case "all":
		c.compile(node.Arguments[0])
		c.emit(OpBegin)
		var loopBreak int
		c.emitLoop(func() {
			c.compile(node.Arguments[1])
			loopBreak = c.emit(OpJumpIfFalse, placeholder)
			c.emit(OpPop)
		})
		c.emit(OpTrue)
		c.patchJump(loopBreak)
		c.emit(OpEnd)
		return

	case "none":
		c.compile(node.Arguments[0])
		c.emit(OpBegin)
		var loopBreak int
		c.emitLoop(func() {
			c.compile(node.Arguments[1])
			c.emit(OpNot)
			loopBreak = c.emit(OpJumpIfFalse, placeholder)
			c.emit(OpPop)
		})
		c.emit(OpTrue)
		c.patchJump(loopBreak)
		c.emit(OpEnd)
		return

	case "any":
		c.compile(node.Arguments[0])
		c.emit(OpBegin)
		var loopBreak int
		c.emitLoop(func() {
			c.compile(node.Arguments[1])
			loopBreak = c.emit(OpJumpIfTrue, placeholder)
			c.emit(OpPop)
		})
		c.emit(OpFalse)
		c.patchJump(loopBreak)
		c.emit(OpEnd)
		return

	case "one":
		c.compile(node.Arguments[0])
		c.emit(OpBegin)
		c.emitLoop(func() {
			c.compile(node.Arguments[1])
			c.emitCond(func() {
				c.emit(OpIncrementCount)
			})
		})
		c.emit(OpGetCount)
		c.emitPush(1)
		c.emit(OpEqual)
		c.emit(OpEnd)
		return

	case "filter":
		c.compile(node.Arguments[0])
		c.emit(OpBegin)
		c.emitLoop(func() {
			c.compile(node.Arguments[1])
			c.emitCond(func() {
				c.emit(OpIncrementCount)
				if node.Map != nil {
					c.compile(node.Map)
				} else {
					c.emit(OpPointer)
				}
			})
		})
		c.emit(OpGetCount)
		c.emit(OpEnd)
		c.emit(OpArray)
		return

	case "map":
		c.compile(node.Arguments[0])
		c.emit(OpBegin)
		c.emitLoop(func() {
			c.compile(node.Arguments[1])
		})
		c.emit(OpGetLen)
		c.emit(OpEnd)
		c.emit(OpArray)
		return

	case "count":
		c.compile(node.Arguments[0])
		c.emit(OpBegin)
		c.emitLoop(func() {
			c.compile(node.Arguments[1])
			c.emitCond(func() {
				c.emit(OpIncrementCount)
			})
		})
		c.emit(OpGetCount)
		c.emit(OpEnd)
		return

	case "find":
		c.compile(node.Arguments[0])
		c.emit(OpBegin)
		var loopBreak int
		c.emitLoop(func() {
			c.compile(node.Arguments[1])
			noop := c.emit(OpJumpIfFalse, placeholder)
			c.emit(OpPop)
			if node.Map != nil {
				c.compile(node.Map)
			} else {
				c.emit(OpPointer)
			}
			loopBreak = c.emit(OpJump, placeholder)
			c.patchJump(noop)
			c.emit(OpPop)
		})
		if node.Throws {
			c.emit(OpPush, c.addConstant(fmt.Errorf("reflect: slice index out of range")))
			c.emit(OpThrow)
		} else {
			c.emit(OpNil)
		}
		c.patchJump(loopBreak)
		c.emit(OpEnd)
		return

	case "findIndex":
		c.compile(node.Arguments[0])
		c.emit(OpBegin)
		var loopBreak int
		c.emitLoop(func() {
			c.compile(node.Arguments[1])
			noop := c.emit(OpJumpIfFalse, placeholder)
			c.emit(OpPop)
			c.emit(OpGetIndex)
			loopBreak = c.emit(OpJump, placeholder)
			c.patchJump(noop)
			c.emit(OpPop)
		})
		c.emit(OpNil)
		c.patchJump(loopBreak)
		c.emit(OpEnd)
		return

	case "findLast":
		c.compile(node.Arguments[0])
		c.emit(OpBegin)
		var loopBreak int
		c.emitLoopBackwards(func() {
			c.compile(node.Arguments[1])
			noop := c.emit(OpJumpIfFalse, placeholder)
			c.emit(OpPop)
			if node.Map != nil {
				c.compile(node.Map)
			} else {
				c.emit(OpPointer)
			}
			loopBreak = c.emit(OpJump, placeholder)
			c.patchJump(noop)
			c.emit(OpPop)
		})
		if node.Throws {
			c.emit(OpPush, c.addConstant(fmt.Errorf("reflect: slice index out of range")))
			c.emit(OpThrow)
		} else {
			c.emit(OpNil)
		}
		c.patchJump(loopBreak)
		c.emit(OpEnd)
		return

	case "findLastIndex":
		c.compile(node.Arguments[0])
		c.emit(OpBegin)
		var loopBreak int
		c.emitLoopBackwards(func() {
			c.compile(node.Arguments[1])
			noop := c.emit(OpJumpIfFalse, placeholder)
			c.emit(OpPop)
			c.emit(OpGetIndex)
			loopBreak = c.emit(OpJump, placeholder)
			c.patchJump(noop)
			c.emit(OpPop)
		})
		c.emit(OpNil)
		c.patchJump(loopBreak)
		c.emit(OpEnd)
		return

	case "groupBy":
		c.compile(node.Arguments[0])
		c.emit(OpBegin)
		c.emitLoop(func() {
			c.compile(node.Arguments[1])
			c.emit(OpGroupBy)
		})
		c.emit(OpGetGroupBy)
		c.emit(OpEnd)
		return

	case "reduce":
		c.compile(node.Arguments[0])
		c.emit(OpBegin)
		if len(node.Arguments) == 3 {
			c.compile(node.Arguments[2])
			c.emit(OpSetAcc)
		} else {
			c.emit(OpPointer)
			c.emit(OpIncrementIndex)
			c.emit(OpSetAcc)
		}
		c.emitLoop(func() {
			c.compile(node.Arguments[1])
			c.emit(OpSetAcc)
		})
		c.emit(OpGetAcc)
		c.emit(OpEnd)
		return

	}

	if id, ok := builtin.Index[node.Name]; ok {
		f := builtin.Builtins[id]
		for _, arg := range node.Arguments {
			c.compile(arg)
		}
		if f.Fast != nil {
			c.emit(OpCallBuiltin1, id)
		} else if f.Func != nil {
			c.emitFunction(f, len(node.Arguments))
		}
		return
	}

	panic(fmt.Sprintf("unknown builtin %v", node.Name))
}

func (c *compiler) emitCond(body func()) {
	noop := c.emit(OpJumpIfFalse, placeholder)
	c.emit(OpPop)

	body()

	jmp := c.emit(OpJump, placeholder)
	c.patchJump(noop)
	c.emit(OpPop)
	c.patchJump(jmp)
}

func (c *compiler) emitLoop(body func()) {
	begin := len(c.bytecode)
	end := c.emit(OpJumpIfEnd, placeholder)

	body()

	c.emit(OpIncrementIndex)
	c.emit(OpJumpBackward, c.calcBackwardJump(begin))
	c.patchJump(end)
}

func (c *compiler) emitLoopBackwards(body func()) {
	c.emit(OpGetLen)
	c.emit(OpInt, 1)
	c.emit(OpSubtract)
	c.emit(OpSetIndex)
	begin := len(c.bytecode)
	c.emit(OpGetIndex)
	c.emit(OpInt, 0)
	c.emit(OpMoreOrEqual)
	end := c.emit(OpJumpIfFalse, placeholder)

	body()

	c.emit(OpDecrementIndex)
	c.emit(OpJumpBackward, c.calcBackwardJump(begin))
	c.patchJump(end)
}

func (c *compiler) ClosureNode(node *ast.ClosureNode) {
	c.compile(node.Node)
}

func (c *compiler) PointerNode(node *ast.PointerNode) {
	switch node.Name {
	case "index":
		c.emit(OpGetIndex)
	case "acc":
		c.emit(OpGetAcc)
	case "":
		c.emit(OpPointer)
	default:
		panic(fmt.Sprintf("unknown pointer %v", node.Name))
	}
}

func (c *compiler) VariableDeclaratorNode(node *ast.VariableDeclaratorNode) {
	c.compile(node.Value)
	index := c.addVariable(node.Name)
	c.emit(OpStore, index)
	c.beginScope(node.Name, index)
	c.compile(node.Expr)
	c.endScope()
}

func (c *compiler) beginScope(name string, index int) {
	c.scopes = append(c.scopes, scope{name, index})
}

func (c *compiler) endScope() {
	c.scopes = c.scopes[:len(c.scopes)-1]
}

func (c *compiler) lookupVariable(name string) (int, bool) {
	for i := len(c.scopes) - 1; i >= 0; i-- {
		if c.scopes[i].variableName == name {
			return c.scopes[i].index, true
		}
	}
	return 0, false
}

func (c *compiler) ConditionalNode(node *ast.ConditionalNode) {
	c.compile(node.Cond)
	otherwise := c.emit(OpJumpIfFalse, placeholder)

	c.emit(OpPop)
	c.compile(node.Exp1)
	end := c.emit(OpJump, placeholder)

	c.patchJump(otherwise)
	c.emit(OpPop)
	c.compile(node.Exp2)

	c.patchJump(end)
}

func (c *compiler) ArrayNode(node *ast.ArrayNode) {
	for _, node := range node.Nodes {
		c.compile(node)
	}

	c.emitPush(len(node.Nodes))
	c.emit(OpArray)
}

func (c *compiler) MapNode(node *ast.MapNode) {
	for _, pair := range node.Pairs {
		c.compile(pair)
	}

	c.emitPush(len(node.Pairs))
	c.emit(OpMap)
}

func (c *compiler) PairNode(node *ast.PairNode) {
	c.compile(node.Key)
	c.compile(node.Value)
}

func (c *compiler) derefInNeeded(node ast.Node) {
	switch kind(node) {
	case reflect.Ptr, reflect.Interface:
		c.emit(OpDeref)
	}
}

func kind(node ast.Node) reflect.Kind {
	t := node.Type()
	if t == nil {
		return reflect.Invalid
	}
	return t.Kind()
}
