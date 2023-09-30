package checker

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/antonmedv/expr/ast"
	"github.com/antonmedv/expr/builtin"
	"github.com/antonmedv/expr/conf"
	"github.com/antonmedv/expr/file"
	"github.com/antonmedv/expr/parser"
	"github.com/antonmedv/expr/vm"
)

func Check(tree *parser.Tree, config *conf.Config) (t reflect.Type, err error) {
	if config == nil {
		config = conf.New(nil)
	}

	v := &checker{config: config}

	t, _ = v.visit(tree.Node)

	if v.err != nil {
		return t, v.err.Bind(tree.Source)
	}

	if v.config.Expect != reflect.Invalid {
		if v.config.ExpectAny {
			if isAny(t) {
				return t, nil
			}
		}

		switch v.config.Expect {
		case reflect.Int, reflect.Int64, reflect.Float64:
			if !isNumber(t) {
				return nil, fmt.Errorf("expected %v, but got %v", v.config.Expect, t)
			}
		default:
			if t != nil {
				if t.Kind() == v.config.Expect {
					return t, nil
				}
			}
			return nil, fmt.Errorf("expected %v, but got %v", v.config.Expect, t)
		}
	}

	return t, nil
}

type checker struct {
	config          *conf.Config
	predicateScopes []predicateScope
	varScopes       []varScope
	parents         []ast.Node
	err             *file.Error
}

type predicateScope struct {
	vtype reflect.Type
	vars  map[string]reflect.Type
}

type varScope struct {
	name  string
	vtype reflect.Type
	info  info
}

type info struct {
	method bool
	fn     *ast.Function

	// elem is element type of array or map.
	// Arrays created with type []any, but
	// we would like to detect expressions
	// like `42 in ["a"]` as invalid.
	elem reflect.Type
}

func (v *checker) visit(node ast.Node) (reflect.Type, info) {
	var t reflect.Type
	var i info
	v.parents = append(v.parents, node)
	switch n := node.(type) {
	case *ast.NilNode:
		t, i = v.NilNode(n)
	case *ast.IdentifierNode:
		t, i = v.IdentifierNode(n)
	case *ast.IntegerNode:
		t, i = v.IntegerNode(n)
	case *ast.FloatNode:
		t, i = v.FloatNode(n)
	case *ast.BoolNode:
		t, i = v.BoolNode(n)
	case *ast.StringNode:
		t, i = v.StringNode(n)
	case *ast.ConstantNode:
		t, i = v.ConstantNode(n)
	case *ast.UnaryNode:
		t, i = v.UnaryNode(n)
	case *ast.BinaryNode:
		t, i = v.BinaryNode(n)
	case *ast.ChainNode:
		t, i = v.ChainNode(n)
	case *ast.MemberNode:
		t, i = v.MemberNode(n)
	case *ast.SliceNode:
		t, i = v.SliceNode(n)
	case *ast.CallNode:
		t, i = v.CallNode(n)
	case *ast.BuiltinNode:
		t, i = v.BuiltinNode(n)
	case *ast.ClosureNode:
		t, i = v.ClosureNode(n)
	case *ast.PointerNode:
		t, i = v.PointerNode(n)
	case *ast.VariableDeclaratorNode:
		t, i = v.VariableDeclaratorNode(n)
	case *ast.ConditionalNode:
		t, i = v.ConditionalNode(n)
	case *ast.ArrayNode:
		t, i = v.ArrayNode(n)
	case *ast.MapNode:
		t, i = v.MapNode(n)
	case *ast.PairNode:
		t, i = v.PairNode(n)
	default:
		panic(fmt.Sprintf("undefined node type (%T)", node))
	}
	v.parents = v.parents[:len(v.parents)-1]
	node.SetType(t)
	return t, i
}

func (v *checker) error(node ast.Node, format string, args ...any) (reflect.Type, info) {
	if v.err == nil { // show first error
		v.err = &file.Error{
			Location: node.Location(),
			Message:  fmt.Sprintf(format, args...),
		}
	}
	return anyType, info{} // interface represent undefined type
}

func (v *checker) NilNode(*ast.NilNode) (reflect.Type, info) {
	return nilType, info{}
}

func (v *checker) IdentifierNode(node *ast.IdentifierNode) (reflect.Type, info) {
	if s, ok := v.lookupVariable(node.Value); ok {
		return s.vtype, s.info
	}
	if node.Value == "$env" {
		return mapType, info{}
	}
	if fn, ok := v.config.Builtins[node.Value]; ok {
		return functionType, info{fn: fn}
	}
	if fn, ok := v.config.Functions[node.Value]; ok {
		return functionType, info{fn: fn}
	}
	if t, ok := v.config.Types[node.Value]; ok {
		if t.Ambiguous {
			return v.error(node, "ambiguous identifier %v", node.Value)
		}
		node.Method = t.Method
		node.MethodIndex = t.MethodIndex
		node.FieldIndex = t.FieldIndex
		return t.Type, info{method: t.Method}
	}
	if v.config.Strict {
		return v.error(node, "unknown name %v", node.Value)
	}
	if v.config.DefaultType != nil {
		return v.config.DefaultType, info{}
	}
	return anyType, info{}
}

func (v *checker) IntegerNode(*ast.IntegerNode) (reflect.Type, info) {
	return integerType, info{}
}

func (v *checker) FloatNode(*ast.FloatNode) (reflect.Type, info) {
	return floatType, info{}
}

func (v *checker) BoolNode(*ast.BoolNode) (reflect.Type, info) {
	return boolType, info{}
}

func (v *checker) StringNode(*ast.StringNode) (reflect.Type, info) {
	return stringType, info{}
}

func (v *checker) ConstantNode(node *ast.ConstantNode) (reflect.Type, info) {
	return reflect.TypeOf(node.Value), info{}
}

func (v *checker) UnaryNode(node *ast.UnaryNode) (reflect.Type, info) {
	t, _ := v.visit(node.Node)

	t = deref(t)

	switch node.Operator {

	case "!", "not":
		if isBool(t) {
			return boolType, info{}
		}
		if isAny(t) {
			return boolType, info{}
		}

	case "+", "-":
		if isNumber(t) {
			return t, info{}
		}
		if isAny(t) {
			return anyType, info{}
		}

	default:
		return v.error(node, "unknown operator (%v)", node.Operator)
	}

	return v.error(node, `invalid operation: %v (mismatched type %v)`, node.Operator, t)
}

func (v *checker) BinaryNode(node *ast.BinaryNode) (reflect.Type, info) {
	l, _ := v.visit(node.Left)
	r, ri := v.visit(node.Right)

	l = deref(l)
	r = deref(r)

	// check operator overloading
	if fns, ok := v.config.Operators[node.Operator]; ok {
		t, _, ok := conf.FindSuitableOperatorOverload(fns, v.config.Types, l, r)
		if ok {
			return t, info{}
		}
	}

	switch node.Operator {
	case "==", "!=":
		if isComparable(l, r) {
			return boolType, info{}
		}

	case "or", "||", "and", "&&":
		if isBool(l) && isBool(r) {
			return boolType, info{}
		}
		if or(l, r, isBool) {
			return boolType, info{}
		}

	case "<", ">", ">=", "<=":
		if isNumber(l) && isNumber(r) {
			return boolType, info{}
		}
		if isString(l) && isString(r) {
			return boolType, info{}
		}
		if isTime(l) && isTime(r) {
			return boolType, info{}
		}
		if or(l, r, isNumber, isString, isTime) {
			return boolType, info{}
		}

	case "-":
		if isNumber(l) && isNumber(r) {
			return combined(l, r), info{}
		}
		if isTime(l) && isTime(r) {
			return durationType, info{}
		}
		if isTime(l) && isDuration(r) {
			return timeType, info{}
		}
		if or(l, r, isNumber, isTime) {
			return anyType, info{}
		}

	case "*":
		if isNumber(l) && isNumber(r) {
			return combined(l, r), info{}
		}
		if or(l, r, isNumber) {
			return anyType, info{}
		}

	case "/":
		if isNumber(l) && isNumber(r) {
			return floatType, info{}
		}
		if or(l, r, isNumber) {
			return floatType, info{}
		}

	case "**", "^":
		if isNumber(l) && isNumber(r) {
			return floatType, info{}
		}
		if or(l, r, isNumber) {
			return floatType, info{}
		}

	case "%":
		if isInteger(l) && isInteger(r) {
			return combined(l, r), info{}
		}
		if or(l, r, isInteger) {
			return anyType, info{}
		}

	case "+":
		if isNumber(l) && isNumber(r) {
			return combined(l, r), info{}
		}
		if isString(l) && isString(r) {
			return stringType, info{}
		}
		if isTime(l) && isDuration(r) {
			return timeType, info{}
		}
		if isDuration(l) && isTime(r) {
			return timeType, info{}
		}
		if or(l, r, isNumber, isString, isTime, isDuration) {
			return anyType, info{}
		}

	case "in":
		if (isString(l) || isAny(l)) && isStruct(r) {
			return boolType, info{}
		}
		if isMap(r) {
			if l == nil { // It is possible to compare with nil.
				return boolType, info{}
			}
			if !isAny(l) && !l.AssignableTo(r.Key()) {
				return v.error(node, "cannot use %v as type %v in map key", l, r.Key())
			}
			return boolType, info{}
		}
		if isArray(r) {
			if l == nil { // It is possible to compare with nil.
				return boolType, info{}
			}
			if !isComparable(l, r.Elem()) {
				return v.error(node, "cannot use %v as type %v in array", l, r.Elem())
			}
			if !isComparable(l, ri.elem) {
				return v.error(node, "cannot use %v as type %v in array", l, ri.elem)
			}
			return boolType, info{}
		}
		if isAny(l) && anyOf(r, isString, isArray, isMap) {
			return boolType, info{}
		}
		if isAny(r) {
			return boolType, info{}
		}

	case "matches":
		if s, ok := node.Right.(*ast.StringNode); ok {
			r, err := regexp.Compile(s.Value)
			if err != nil {
				return v.error(node, err.Error())
			}
			node.Regexp = r
		}
		if isString(l) && isString(r) {
			return boolType, info{}
		}
		if or(l, r, isString) {
			return boolType, info{}
		}

	case "contains", "startsWith", "endsWith":
		if isString(l) && isString(r) {
			return boolType, info{}
		}
		if or(l, r, isString) {
			return boolType, info{}
		}

	case "..":
		ret := reflect.SliceOf(integerType)
		if isInteger(l) && isInteger(r) {
			return ret, info{}
		}
		if or(l, r, isInteger) {
			return ret, info{}
		}

	case "??":
		if l == nil && r != nil {
			return r, info{}
		}
		if l != nil && r == nil {
			return l, info{}
		}
		if l == nil && r == nil {
			return nilType, info{}
		}
		if r.AssignableTo(l) {
			return l, info{}
		}
		return anyType, info{}

	default:
		return v.error(node, "unknown operator (%v)", node.Operator)

	}

	return v.error(node, `invalid operation: %v (mismatched types %v and %v)`, node.Operator, l, r)
}

func (v *checker) ChainNode(node *ast.ChainNode) (reflect.Type, info) {
	return v.visit(node.Node)
}

func (v *checker) MemberNode(node *ast.MemberNode) (reflect.Type, info) {
	base, _ := v.visit(node.Node)
	prop, _ := v.visit(node.Property)

	if an, ok := node.Node.(*ast.IdentifierNode); ok && an.Value == "$env" {
		// If the index is a constant string, can save some
		// cycles later by finding the type of its referent
		if name, ok := node.Property.(*ast.StringNode); ok {
			if t, ok := v.config.Types[name.Value]; ok {
				return t.Type, info{method: t.Method}
			} // No error if no type found; it may be added to env between compile and run
		}
		return anyType, info{}
	}

	if name, ok := node.Property.(*ast.StringNode); ok {
		if base == nil {
			return v.error(node, "type %v has no field %v", base, name.Value)
		}
		// First, check methods defined on base type itself,
		// independent of which type it is. Without dereferencing.
		if m, ok := base.MethodByName(name.Value); ok {
			if kind(base) == reflect.Interface {
				// In case of interface type method will not have a receiver,
				// and to prevent checker decreasing numbers of in arguments
				// return method type as not method (second argument is false).

				// Also, we can not use m.Index here, because it will be
				// different indexes for different types which implement
				// the same interface.
				return m.Type, info{}
			} else {
				node.Method = true
				node.MethodIndex = m.Index
				node.Name = name.Value
				return m.Type, info{method: true}
			}
		}
	}

	if kind(base) == reflect.Ptr {
		base = base.Elem()
	}

	switch kind(base) {
	case reflect.Interface:
		return anyType, info{}

	case reflect.Map:
		if prop != nil && !prop.AssignableTo(base.Key()) && !isAny(prop) {
			return v.error(node.Property, "cannot use %v to get an element from %v", prop, base)
		}
		return base.Elem(), info{}

	case reflect.Array, reflect.Slice:
		if !isInteger(prop) && !isAny(prop) {
			return v.error(node.Property, "array elements can only be selected using an integer (got %v)", prop)
		}
		return base.Elem(), info{}

	case reflect.Struct:
		if name, ok := node.Property.(*ast.StringNode); ok {
			propertyName := name.Value
			if field, ok := fetchField(base, propertyName); ok {
				node.FieldIndex = field.Index
				node.Name = propertyName
				return field.Type, info{}
			}
			if len(v.parents) > 1 {
				if _, ok := v.parents[len(v.parents)-2].(*ast.CallNode); ok {
					return v.error(node, "type %v has no method %v", base, propertyName)
				}
			}
			return v.error(node, "type %v has no field %v", base, propertyName)
		}
	}

	return v.error(node, "type %v[%v] is undefined", base, prop)
}

func (v *checker) SliceNode(node *ast.SliceNode) (reflect.Type, info) {
	t, _ := v.visit(node.Node)

	switch kind(t) {
	case reflect.Interface:
		// ok
	case reflect.String, reflect.Array, reflect.Slice:
		// ok
	default:
		return v.error(node, "cannot slice %v", t)
	}

	if node.From != nil {
		from, _ := v.visit(node.From)
		if !isInteger(from) && !isAny(from) {
			return v.error(node.From, "non-integer slice index %v", from)
		}
	}
	if node.To != nil {
		to, _ := v.visit(node.To)
		if !isInteger(to) && !isAny(to) {
			return v.error(node.To, "non-integer slice index %v", to)
		}
	}
	return t, info{}
}

func (v *checker) CallNode(node *ast.CallNode) (reflect.Type, info) {
	fn, fnInfo := v.visit(node.Callee)

	if fnInfo.fn != nil {
		node.Func = fnInfo.fn
		return v.checkFunction(fnInfo.fn, node, node.Arguments)
	}

	fnName := "function"
	if identifier, ok := node.Callee.(*ast.IdentifierNode); ok {
		fnName = identifier.Value
	}
	if member, ok := node.Callee.(*ast.MemberNode); ok {
		if name, ok := member.Property.(*ast.StringNode); ok {
			fnName = name.Value
		}
	}
	switch fn.Kind() {
	case reflect.Interface:
		return anyType, info{}
	case reflect.Func:
		inputParamsCount := 1 // for functions
		if fnInfo.method {
			inputParamsCount = 2 // for methods
		}
		// TODO: Deprecate OpCallFast and move fn(...any) any to TypedFunc list.
		// To do this we need add support for variadic arguments in OpCallTyped.
		if !isAny(fn) &&
			fn.IsVariadic() &&
			fn.NumIn() == inputParamsCount &&
			fn.NumOut() == 1 &&
			fn.Out(0).Kind() == reflect.Interface {
			rest := fn.In(fn.NumIn() - 1) // function has only one param for functions and two for methods
			if kind(rest) == reflect.Slice && rest.Elem().Kind() == reflect.Interface {
				node.Fast = true
			}
		}

		outType, err := v.checkArguments(fnName, fn, fnInfo.method, node.Arguments, node)
		if err != nil {
			if v.err == nil {
				v.err = err
			}
			return anyType, info{}
		}

		v.findTypedFunc(node, fn, fnInfo.method)

		return outType, info{}
	}
	return v.error(node, "%v is not callable", fn)
}

func (v *checker) BuiltinNode(node *ast.BuiltinNode) (reflect.Type, info) {
	switch node.Name {
	case "all", "none", "any", "one":
		collection, _ := v.visit(node.Arguments[0])
		if !isArray(collection) && !isAny(collection) {
			return v.error(node.Arguments[0], "builtin %v takes only array (got %v)", node.Name, collection)
		}

		v.begin(collection)
		closure, _ := v.visit(node.Arguments[1])
		v.end()

		if isFunc(closure) &&
			closure.NumOut() == 1 &&
			closure.NumIn() == 1 && isAny(closure.In(0)) {

			if !isBool(closure.Out(0)) && !isAny(closure.Out(0)) {
				return v.error(node.Arguments[1], "predicate should return boolean (got %v)", closure.Out(0).String())
			}
			return boolType, info{}
		}
		return v.error(node.Arguments[1], "predicate should has one input and one output param")

	case "filter":
		collection, _ := v.visit(node.Arguments[0])
		if !isArray(collection) && !isAny(collection) {
			return v.error(node.Arguments[0], "builtin %v takes only array (got %v)", node.Name, collection)
		}

		v.begin(collection)
		closure, _ := v.visit(node.Arguments[1])
		v.end()

		if isFunc(closure) &&
			closure.NumOut() == 1 &&
			closure.NumIn() == 1 && isAny(closure.In(0)) {

			if !isBool(closure.Out(0)) && !isAny(closure.Out(0)) {
				return v.error(node.Arguments[1], "predicate should return boolean (got %v)", closure.Out(0).String())
			}
			if isAny(collection) {
				return arrayType, info{}
			}
			return reflect.SliceOf(collection.Elem()), info{}
		}
		return v.error(node.Arguments[1], "predicate should has one input and one output param")

	case "map":
		collection, _ := v.visit(node.Arguments[0])
		if !isArray(collection) && !isAny(collection) {
			return v.error(node.Arguments[0], "builtin %v takes only array (got %v)", node.Name, collection)
		}

		v.begin(collection, scopeVar{"index", integerType})
		closure, _ := v.visit(node.Arguments[1])
		v.end()

		if isFunc(closure) &&
			closure.NumOut() == 1 &&
			closure.NumIn() == 1 && isAny(closure.In(0)) {

			return reflect.SliceOf(closure.Out(0)), info{}
		}
		return v.error(node.Arguments[1], "predicate should has one input and one output param")

	case "count":
		collection, _ := v.visit(node.Arguments[0])
		if !isArray(collection) && !isAny(collection) {
			return v.error(node.Arguments[0], "builtin %v takes only array (got %v)", node.Name, collection)
		}

		v.begin(collection)
		closure, _ := v.visit(node.Arguments[1])
		v.end()

		if isFunc(closure) &&
			closure.NumOut() == 1 &&
			closure.NumIn() == 1 && isAny(closure.In(0)) {
			if !isBool(closure.Out(0)) && !isAny(closure.Out(0)) {
				return v.error(node.Arguments[1], "predicate should return boolean (got %v)", closure.Out(0).String())
			}

			return integerType, info{}
		}
		return v.error(node.Arguments[1], "predicate should has one input and one output param")

	case "find", "findLast":
		collection, _ := v.visit(node.Arguments[0])
		if !isArray(collection) && !isAny(collection) {
			return v.error(node.Arguments[0], "builtin %v takes only array (got %v)", node.Name, collection)
		}

		v.begin(collection)
		closure, _ := v.visit(node.Arguments[1])
		v.end()

		if isFunc(closure) &&
			closure.NumOut() == 1 &&
			closure.NumIn() == 1 && isAny(closure.In(0)) {

			if !isBool(closure.Out(0)) && !isAny(closure.Out(0)) {
				return v.error(node.Arguments[1], "predicate should return boolean (got %v)", closure.Out(0).String())
			}
			if isAny(collection) {
				return anyType, info{}
			}
			return collection.Elem(), info{}
		}
		return v.error(node.Arguments[1], "predicate should has one input and one output param")

	case "findIndex", "findLastIndex":
		collection, _ := v.visit(node.Arguments[0])
		if !isArray(collection) && !isAny(collection) {
			return v.error(node.Arguments[0], "builtin %v takes only array (got %v)", node.Name, collection)
		}

		v.begin(collection)
		closure, _ := v.visit(node.Arguments[1])
		v.end()

		if isFunc(closure) &&
			closure.NumOut() == 1 &&
			closure.NumIn() == 1 && isAny(closure.In(0)) {

			if !isBool(closure.Out(0)) && !isAny(closure.Out(0)) {
				return v.error(node.Arguments[1], "predicate should return boolean (got %v)", closure.Out(0).String())
			}
			return integerType, info{}
		}
		return v.error(node.Arguments[1], "predicate should has one input and one output param")

	case "groupBy":
		collection, _ := v.visit(node.Arguments[0])
		if !isArray(collection) && !isAny(collection) {
			return v.error(node.Arguments[0], "builtin %v takes only array (got %v)", node.Name, collection)
		}

		v.begin(collection)
		closure, _ := v.visit(node.Arguments[1])
		v.end()

		if isFunc(closure) &&
			closure.NumOut() == 1 &&
			closure.NumIn() == 1 && isAny(closure.In(0)) {

			return reflect.TypeOf(map[any][]any{}), info{}
		}
		return v.error(node.Arguments[1], "predicate should has one input and one output param")

	case "reduce":
		collection, _ := v.visit(node.Arguments[0])
		if !isArray(collection) && !isAny(collection) {
			return v.error(node.Arguments[0], "builtin %v takes only array (got %v)", node.Name, collection)
		}

		v.begin(collection, scopeVar{"index", integerType}, scopeVar{"acc", anyType})
		closure, _ := v.visit(node.Arguments[1])
		v.end()

		if len(node.Arguments) == 3 {
			_, _ = v.visit(node.Arguments[2])
		}

		if isFunc(closure) && closure.NumOut() == 1 {
			return closure.Out(0), info{}
		}
		return v.error(node.Arguments[1], "predicate should has two input and one output param")

	}

	if id, ok := builtin.Index[node.Name]; ok {
		switch node.Name {
		case "get":
			return v.checkBuiltinGet(node)
		}
		return v.checkFunction(builtin.Builtins[id], node, node.Arguments)
	}

	return v.error(node, "unknown builtin %v", node.Name)
}

type scopeVar struct {
	name  string
	vtype reflect.Type
}

func (v *checker) begin(vtype reflect.Type, vars ...scopeVar) {
	scope := predicateScope{vtype: vtype, vars: make(map[string]reflect.Type)}
	for _, v := range vars {
		scope.vars[v.name] = v.vtype
	}
	v.predicateScopes = append(v.predicateScopes, scope)
}

func (v *checker) end() {
	v.predicateScopes = v.predicateScopes[:len(v.predicateScopes)-1]
}

func (v *checker) checkBuiltinGet(node *ast.BuiltinNode) (reflect.Type, info) {
	if len(node.Arguments) != 2 {
		return v.error(node, "invalid number of arguments (expected 2, got %d)", len(node.Arguments))
	}

	val := node.Arguments[0]
	prop := node.Arguments[1]
	if id, ok := val.(*ast.IdentifierNode); ok && id.Value == "$env" {
		if s, ok := prop.(*ast.StringNode); ok {
			return v.config.Types[s.Value].Type, info{}
		}
		return anyType, info{}
	}

	t, _ := v.visit(val)

	switch kind(t) {
	case reflect.Interface:
		return anyType, info{}
	case reflect.Slice, reflect.Array:
		p, _ := v.visit(prop)
		if p == nil {
			return v.error(prop, "cannot use nil as slice index")
		}
		if !isInteger(p) && !isAny(p) {
			return v.error(prop, "non-integer slice index %v", p)
		}
		return t.Elem(), info{}
	case reflect.Map:
		p, _ := v.visit(prop)
		if p == nil {
			return v.error(prop, "cannot use nil as map index")
		}
		if !p.AssignableTo(t.Key()) && !isAny(p) {
			return v.error(prop, "cannot use %v to get an element from %v", p, t)
		}
		return t.Elem(), info{}
	}
	return v.error(val, "type %v does not support indexing", t)
}

func (v *checker) checkFunction(f *ast.Function, node ast.Node, arguments []ast.Node) (reflect.Type, info) {
	if f.Validate != nil {
		args := make([]reflect.Type, len(arguments))
		for i, arg := range arguments {
			args[i], _ = v.visit(arg)
		}
		t, err := f.Validate(args)
		if err != nil {
			return v.error(node, "%v", err)
		}
		return t, info{}
	} else if len(f.Types) == 0 {
		t, err := v.checkArguments(f.Name, functionType, false, arguments, node)
		if err != nil {
			if v.err == nil {
				v.err = err
			}
			return anyType, info{}
		}
		// No type was specified, so we assume the function returns any.
		return t, info{}
	}
	var lastErr *file.Error
	for _, t := range f.Types {
		outType, err := v.checkArguments(f.Name, t, false, arguments, node)
		if err != nil {
			lastErr = err
			continue
		}
		return outType, info{}
	}
	if lastErr != nil {
		if v.err == nil {
			v.err = lastErr
		}
		return anyType, info{}
	}

	return v.error(node, "no matching overload for %v", f.Name)
}

func (v *checker) checkArguments(name string, fn reflect.Type, method bool, arguments []ast.Node, node ast.Node) (reflect.Type, *file.Error) {
	if isAny(fn) {
		return anyType, nil
	}

	if fn.NumOut() == 0 {
		return anyType, &file.Error{
			Location: node.Location(),
			Message:  fmt.Sprintf("func %v doesn't return value", name),
		}
	}
	if numOut := fn.NumOut(); numOut > 2 {
		return anyType, &file.Error{
			Location: node.Location(),
			Message:  fmt.Sprintf("func %v returns more then two values", name),
		}
	}

	// If func is method on an env, first argument should be a receiver,
	// and actual arguments less than fnNumIn by one.
	fnNumIn := fn.NumIn()
	if method {
		fnNumIn--
	}
	// Skip first argument in case of the receiver.
	fnInOffset := 0
	if method {
		fnInOffset = 1
	}

	if fn.IsVariadic() {
		if len(arguments) < fnNumIn-1 {
			return anyType, &file.Error{
				Location: node.Location(),
				Message:  fmt.Sprintf("not enough arguments to call %v", name),
			}
		}
	} else {
		if len(arguments) > fnNumIn {
			return anyType, &file.Error{
				Location: node.Location(),
				Message:  fmt.Sprintf("too many arguments to call %v", name),
			}
		}
		if len(arguments) < fnNumIn {
			return anyType, &file.Error{
				Location: node.Location(),
				Message:  fmt.Sprintf("not enough arguments to call %v", name),
			}
		}
	}

	for i, arg := range arguments {
		t, _ := v.visit(arg)

		var in reflect.Type
		if fn.IsVariadic() && i >= fnNumIn-1 {
			// For variadic arguments fn(xs ...int), go replaces type of xs (int) with ([]int).
			// As we compare arguments one by one, we need underling type.
			in = fn.In(fn.NumIn() - 1).Elem()
		} else {
			in = fn.In(i + fnInOffset)
		}

		if isFloat(in) {
			traverseAndReplaceIntegerNodesWithFloatNodes(&arguments[i], in)
			continue
		}

		if isInteger(in) && isInteger(t) && kind(t) != kind(in) {
			traverseAndReplaceIntegerNodesWithIntegerNodes(&arguments[i], in)
			continue
		}

		if t == nil {
			continue
		}

		if !t.AssignableTo(in) && kind(t) != reflect.Interface {
			return anyType, &file.Error{
				Location: arg.Location(),
				Message:  fmt.Sprintf("cannot use %v as argument (type %v) to call %v ", t, in, name),
			}
		}
	}

	return fn.Out(0), nil
}

func traverseAndReplaceIntegerNodesWithFloatNodes(node *ast.Node, newType reflect.Type) {
	switch (*node).(type) {
	case *ast.IntegerNode:
		*node = &ast.FloatNode{Value: float64((*node).(*ast.IntegerNode).Value)}
		(*node).SetType(newType)
	case *ast.UnaryNode:
		unaryNode := (*node).(*ast.UnaryNode)
		traverseAndReplaceIntegerNodesWithFloatNodes(&unaryNode.Node, newType)
	case *ast.BinaryNode:
		binaryNode := (*node).(*ast.BinaryNode)
		switch binaryNode.Operator {
		case "+", "-", "*":
			traverseAndReplaceIntegerNodesWithFloatNodes(&binaryNode.Left, newType)
			traverseAndReplaceIntegerNodesWithFloatNodes(&binaryNode.Right, newType)
		}
	}
}

func traverseAndReplaceIntegerNodesWithIntegerNodes(node *ast.Node, newType reflect.Type) {
	switch (*node).(type) {
	case *ast.IntegerNode:
		(*node).SetType(newType)
	case *ast.UnaryNode:
		unaryNode := (*node).(*ast.UnaryNode)
		traverseAndReplaceIntegerNodesWithIntegerNodes(&unaryNode.Node, newType)
	case *ast.BinaryNode:
		binaryNode := (*node).(*ast.BinaryNode)
		switch binaryNode.Operator {
		case "+", "-", "*":
			traverseAndReplaceIntegerNodesWithIntegerNodes(&binaryNode.Left, newType)
			traverseAndReplaceIntegerNodesWithIntegerNodes(&binaryNode.Right, newType)
		}
	}
}

func (v *checker) ClosureNode(node *ast.ClosureNode) (reflect.Type, info) {
	t, _ := v.visit(node.Node)
	if t == nil {
		return v.error(node.Node, "closure cannot be nil")
	}
	return reflect.FuncOf([]reflect.Type{anyType}, []reflect.Type{t}, false), info{}
}

func (v *checker) PointerNode(node *ast.PointerNode) (reflect.Type, info) {
	if len(v.predicateScopes) == 0 {
		return v.error(node, "cannot use pointer accessor outside closure")
	}
	scope := v.predicateScopes[len(v.predicateScopes)-1]
	if node.Name == "" {
		switch scope.vtype.Kind() {
		case reflect.Interface:
			return anyType, info{}
		case reflect.Array, reflect.Slice:
			return scope.vtype.Elem(), info{}
		}
		return v.error(node, "cannot use %v as array", scope)
	}
	if scope.vars != nil {
		if t, ok := scope.vars[node.Name]; ok {
			return t, info{}
		}
	}
	return v.error(node, "unknown pointer #%v", node.Name)
}

func (v *checker) VariableDeclaratorNode(node *ast.VariableDeclaratorNode) (reflect.Type, info) {
	if _, ok := v.config.Types[node.Name]; ok {
		return v.error(node, "cannot redeclare %v", node.Name)
	}
	if _, ok := v.config.Functions[node.Name]; ok {
		return v.error(node, "cannot redeclare function %v", node.Name)
	}
	if _, ok := v.config.Builtins[node.Name]; ok {
		return v.error(node, "cannot redeclare builtin %v", node.Name)
	}
	if _, ok := v.lookupVariable(node.Name); ok {
		return v.error(node, "cannot redeclare variable %v", node.Name)
	}
	vtype, vinfo := v.visit(node.Value)
	v.varScopes = append(v.varScopes, varScope{node.Name, vtype, vinfo})
	t, i := v.visit(node.Expr)
	v.varScopes = v.varScopes[:len(v.varScopes)-1]
	return t, i
}

func (v *checker) lookupVariable(name string) (varScope, bool) {
	for i := len(v.varScopes) - 1; i >= 0; i-- {
		if v.varScopes[i].name == name {
			return v.varScopes[i], true
		}
	}
	return varScope{}, false
}

func (v *checker) ConditionalNode(node *ast.ConditionalNode) (reflect.Type, info) {
	c, _ := v.visit(node.Cond)
	if !isBool(c) && !isAny(c) {
		return v.error(node.Cond, "non-bool expression (type %v) used as condition", c)
	}

	t1, _ := v.visit(node.Exp1)
	t2, _ := v.visit(node.Exp2)

	if t1 == nil && t2 != nil {
		return t2, info{}
	}
	if t1 != nil && t2 == nil {
		return t1, info{}
	}
	if t1 == nil && t2 == nil {
		return nilType, info{}
	}
	if t1.AssignableTo(t2) {
		return t1, info{}
	}
	return anyType, info{}
}

func (v *checker) ArrayNode(node *ast.ArrayNode) (reflect.Type, info) {
	var prev reflect.Type
	allElementsAreSameType := true
	for i, node := range node.Nodes {
		curr, _ := v.visit(node)
		if i > 0 {
			if curr == nil || prev == nil {
				allElementsAreSameType = false
			} else if curr.Kind() != prev.Kind() {
				allElementsAreSameType = false
			}
		}
		prev = curr
	}
	if allElementsAreSameType && prev != nil {
		return arrayType, info{elem: prev}
	}
	return arrayType, info{}
}

func (v *checker) MapNode(node *ast.MapNode) (reflect.Type, info) {
	for _, pair := range node.Pairs {
		v.visit(pair)
	}
	return mapType, info{}
}

func (v *checker) PairNode(node *ast.PairNode) (reflect.Type, info) {
	v.visit(node.Key)
	v.visit(node.Value)
	return nilType, info{}
}

func (v *checker) findTypedFunc(node *ast.CallNode, fn reflect.Type, method bool) {
	// OnCallTyped doesn't work for functions with variadic arguments,
	// and doesn't work named function, like `type MyFunc func() int`.
	// In PkgPath() is an empty string, it's unnamed function.
	if !fn.IsVariadic() && fn.PkgPath() == "" {
		fnNumIn := fn.NumIn()
		fnInOffset := 0
		if method {
			fnNumIn--
			fnInOffset = 1
		}
	funcTypes:
		for i := range vm.FuncTypes {
			if i == 0 {
				continue
			}
			typed := reflect.ValueOf(vm.FuncTypes[i]).Elem().Type()
			if typed.Kind() != reflect.Func {
				continue
			}
			if typed.NumOut() != fn.NumOut() {
				continue
			}
			for j := 0; j < typed.NumOut(); j++ {
				if typed.Out(j) != fn.Out(j) {
					continue funcTypes
				}
			}
			if typed.NumIn() != fnNumIn {
				continue
			}
			for j := 0; j < typed.NumIn(); j++ {
				if typed.In(j) != fn.In(j+fnInOffset) {
					continue funcTypes
				}
			}
			node.Typed = i
		}
	}
}
