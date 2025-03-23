package checker

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/builtin"
	. "github.com/expr-lang/expr/checker/nature"
	"github.com/expr-lang/expr/conf"
	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser"
)

// Run visitors in a given config over the given tree
// runRepeatable controls whether to filter for only vistors that require multiple passes or not
func runVisitors(tree *parser.Tree, config *conf.Config, runRepeatable bool) {
	for {
		more := false
		for _, v := range config.Visitors {
			// We need to perform types check, because some visitors may rely on
			// types information available in the tree.
			_, _ = Check(tree, config)

			r, repeatable := v.(interface {
				Reset()
				ShouldRepeat() bool
			})

			if repeatable {
				if runRepeatable {
					r.Reset()
					ast.Walk(&tree.Node, v)
					more = more || r.ShouldRepeat()
				}
			} else {
				if !runRepeatable {
					ast.Walk(&tree.Node, v)
				}
			}
		}

		if !more {
			break
		}
	}
}

// ParseCheck parses input expression and checks its types. Also, it applies
// all provided patchers. In case of error, it returns error with a tree.
func ParseCheck(input string, config *conf.Config) (*parser.Tree, error) {
	tree, err := parser.ParseWithConfig(input, config)
	if err != nil {
		return tree, err
	}

	if len(config.Visitors) > 0 {
		// Run all patchers that dont support being run repeatedly first
		runVisitors(tree, config, false)

		// Run patchers that require multiple passes next (currently only Operator patching)
		runVisitors(tree, config, true)
	}
	_, err = Check(tree, config)
	if err != nil {
		return tree, err
	}

	return tree, nil
}

// Check checks types of the expression tree. It returns type of the expression
// and error if any. If config is nil, then default configuration will be used.
func Check(tree *parser.Tree, config *conf.Config) (reflect.Type, error) {
	if config == nil {
		config = conf.New(nil)
	}

	v := &checker{config: config}

	nt := v.visit(tree.Node)

	// To keep compatibility with previous versions, we should return any, if nature is unknown.
	t := nt.Type
	if t == nil {
		t = anyType
	}

	if v.err != nil {
		return t, v.err.Bind(tree.Source)
	}

	if v.config.Expect != reflect.Invalid {
		if v.config.ExpectAny {
			if isUnknown(nt) {
				return t, nil
			}
		}

		switch v.config.Expect {
		case reflect.Int, reflect.Int64, reflect.Float64:
			if !isNumber(nt) {
				return nil, fmt.Errorf("expected %v, but got %v", v.config.Expect, nt)
			}
		default:
			if nt.Kind() != v.config.Expect {
				return nil, fmt.Errorf("expected %v, but got %s", v.config.Expect, nt)
			}
		}
	}

	return t, nil
}

type checker struct {
	config          *conf.Config
	predicateScopes []predicateScope
	varScopes       []varScope
	err             *file.Error
}

type predicateScope struct {
	collection Nature
	vars       map[string]Nature
}

type varScope struct {
	name   string
	nature Nature
}

type info struct {
	method bool
	fn     *builtin.Function

	// elem is element type of array or map.
	// Arrays created with type []any, but
	// we would like to detect expressions
	// like `42 in ["a"]` as invalid.
	elem reflect.Type
}

func (v *checker) visit(node ast.Node) Nature {
	var nt Nature
	switch n := node.(type) {
	case *ast.NilNode:
		nt = v.NilNode(n)
	case *ast.IdentifierNode:
		nt = v.IdentifierNode(n)
	case *ast.IntegerNode:
		nt = v.IntegerNode(n)
	case *ast.FloatNode:
		nt = v.FloatNode(n)
	case *ast.BoolNode:
		nt = v.BoolNode(n)
	case *ast.StringNode:
		nt = v.StringNode(n)
	case *ast.ConstantNode:
		nt = v.ConstantNode(n)
	case *ast.UnaryNode:
		nt = v.UnaryNode(n)
	case *ast.BinaryNode:
		nt = v.BinaryNode(n)
	case *ast.ChainNode:
		nt = v.ChainNode(n)
	case *ast.MemberNode:
		nt = v.MemberNode(n)
	case *ast.SliceNode:
		nt = v.SliceNode(n)
	case *ast.CallNode:
		nt = v.CallNode(n)
	case *ast.BuiltinNode:
		nt = v.BuiltinNode(n)
	case *ast.PredicateNode:
		nt = v.PredicateNode(n)
	case *ast.PointerNode:
		nt = v.PointerNode(n)
	case *ast.VariableDeclaratorNode:
		nt = v.VariableDeclaratorNode(n)
	case *ast.SequenceNode:
		nt = v.SequenceNode(n)
	case *ast.ConditionalNode:
		nt = v.ConditionalNode(n)
	case *ast.ArrayNode:
		nt = v.ArrayNode(n)
	case *ast.MapNode:
		nt = v.MapNode(n)
	case *ast.PairNode:
		nt = v.PairNode(n)
	default:
		panic(fmt.Sprintf("undefined node type (%T)", node))
	}
	node.SetNature(nt)
	return nt
}

func (v *checker) error(node ast.Node, format string, args ...any) Nature {
	if v.err == nil { // show first error
		v.err = &file.Error{
			Location: node.Location(),
			Message:  fmt.Sprintf(format, args...),
		}
	}
	return unknown
}

func (v *checker) NilNode(*ast.NilNode) Nature {
	return nilNature
}

func (v *checker) IdentifierNode(node *ast.IdentifierNode) Nature {
	if variable, ok := v.lookupVariable(node.Value); ok {
		return variable.nature
	}
	if node.Value == "$env" {
		return unknown
	}

	return v.ident(node, node.Value, v.config.Strict, true)
}

// ident method returns type of environment variable, builtin or function.
func (v *checker) ident(node ast.Node, name string, strict, builtins bool) Nature {
	if nt, ok := v.config.Env.Get(name); ok {
		return nt
	}
	if builtins {
		if fn, ok := v.config.Functions[name]; ok {
			return Nature{Type: fn.Type(), Func: fn}
		}
		if fn, ok := v.config.Builtins[name]; ok {
			return Nature{Type: fn.Type(), Func: fn}
		}
	}
	if v.config.Strict && strict {
		return v.error(node, "unknown name %v", name)
	}
	return unknown
}

func (v *checker) IntegerNode(*ast.IntegerNode) Nature {
	return integerNature
}

func (v *checker) FloatNode(*ast.FloatNode) Nature {
	return floatNature
}

func (v *checker) BoolNode(*ast.BoolNode) Nature {
	return boolNature
}

func (v *checker) StringNode(*ast.StringNode) Nature {
	return stringNature
}

func (v *checker) ConstantNode(node *ast.ConstantNode) Nature {
	return Nature{Type: reflect.TypeOf(node.Value)}
}

func (v *checker) UnaryNode(node *ast.UnaryNode) Nature {
	nt := v.visit(node.Node)
	nt = nt.Deref()

	switch node.Operator {

	case "!", "not":
		if isBool(nt) {
			return boolNature
		}
		if isUnknown(nt) {
			return boolNature
		}

	case "+", "-":
		if isNumber(nt) {
			return nt
		}
		if isUnknown(nt) {
			return unknown
		}

	default:
		return v.error(node, "unknown operator (%v)", node.Operator)
	}

	return v.error(node, `invalid operation: %v (mismatched type %s)`, node.Operator, nt)
}

func (v *checker) BinaryNode(node *ast.BinaryNode) Nature {
	l := v.visit(node.Left)
	r := v.visit(node.Right)

	l = l.Deref()
	r = r.Deref()

	switch node.Operator {
	case "==", "!=":
		if isComparable(l, r) {
			return boolNature
		}

	case "or", "||", "and", "&&":
		if isBool(l) && isBool(r) {
			return boolNature
		}
		if or(l, r, isBool) {
			return boolNature
		}

	case "<", ">", ">=", "<=":
		if isNumber(l) && isNumber(r) {
			return boolNature
		}
		if isString(l) && isString(r) {
			return boolNature
		}
		if isTime(l) && isTime(r) {
			return boolNature
		}
		if isDuration(l) && isDuration(r) {
			return boolNature
		}
		if or(l, r, isNumber, isString, isTime, isDuration) {
			return boolNature
		}

	case "-":
		if isNumber(l) && isNumber(r) {
			return combined(l, r)
		}
		if isTime(l) && isTime(r) {
			return durationNature
		}
		if isTime(l) && isDuration(r) {
			return timeNature
		}
		if isDuration(l) && isDuration(r) {
			return durationNature
		}
		if or(l, r, isNumber, isTime, isDuration) {
			return unknown
		}

	case "*":
		if isNumber(l) && isNumber(r) {
			return combined(l, r)
		}
		if isNumber(l) && isDuration(r) {
			return durationNature
		}
		if isDuration(l) && isNumber(r) {
			return durationNature
		}
		if isDuration(l) && isDuration(r) {
			return durationNature
		}
		if or(l, r, isNumber, isDuration) {
			return unknown
		}

	case "/":
		if isNumber(l) && isNumber(r) {
			return floatNature
		}
		if or(l, r, isNumber) {
			return floatNature
		}

	case "**", "^":
		if isNumber(l) && isNumber(r) {
			return floatNature
		}
		if or(l, r, isNumber) {
			return floatNature
		}

	case "%":
		if isInteger(l) && isInteger(r) {
			return integerNature
		}
		if or(l, r, isInteger) {
			return integerNature
		}

	case "+":
		if isNumber(l) && isNumber(r) {
			return combined(l, r)
		}
		if isString(l) && isString(r) {
			return stringNature
		}
		if isTime(l) && isDuration(r) {
			return timeNature
		}
		if isDuration(l) && isTime(r) {
			return timeNature
		}
		if isDuration(l) && isDuration(r) {
			return durationNature
		}
		if or(l, r, isNumber, isString, isTime, isDuration) {
			return unknown
		}

	case "in":
		if (isString(l) || isUnknown(l)) && isStruct(r) {
			return boolNature
		}
		if isMap(r) {
			if !isUnknown(l) && !l.AssignableTo(r.Key()) {
				return v.error(node, "cannot use %v as type %v in map key", l, r.Key())
			}
			return boolNature
		}
		if isArray(r) {
			if !isComparable(l, r.Elem()) {
				return v.error(node, "cannot use %v as type %v in array", l, r.Elem())
			}
			return boolNature
		}
		if isUnknown(l) && anyOf(r, isString, isArray, isMap) {
			return boolNature
		}
		if isUnknown(r) {
			return boolNature
		}

	case "matches":
		if s, ok := node.Right.(*ast.StringNode); ok {
			_, err := regexp.Compile(s.Value)
			if err != nil {
				return v.error(node, err.Error())
			}
		}
		if isString(l) && isString(r) {
			return boolNature
		}
		if or(l, r, isString) {
			return boolNature
		}

	case "contains", "startsWith", "endsWith":
		if isString(l) && isString(r) {
			return boolNature
		}
		if or(l, r, isString) {
			return boolNature
		}

	case "..":
		if isInteger(l) && isInteger(r) {
			return arrayOf(integerNature)
		}
		if or(l, r, isInteger) {
			return arrayOf(integerNature)
		}

	case "??":
		if isNil(l) && !isNil(r) {
			return r
		}
		if !isNil(l) && isNil(r) {
			return l
		}
		if isNil(l) && isNil(r) {
			return nilNature
		}
		if r.AssignableTo(l) {
			return l
		}
		return unknown

	default:
		return v.error(node, "unknown operator (%v)", node.Operator)

	}

	return v.error(node, `invalid operation: %v (mismatched types %v and %v)`, node.Operator, l, r)
}

func (v *checker) ChainNode(node *ast.ChainNode) Nature {
	return v.visit(node.Node)
}

func (v *checker) MemberNode(node *ast.MemberNode) Nature {
	// $env variable
	if an, ok := node.Node.(*ast.IdentifierNode); ok && an.Value == "$env" {
		if name, ok := node.Property.(*ast.StringNode); ok {
			strict := v.config.Strict
			if node.Optional {
				// If user explicitly set optional flag, then we should not
				// throw error if field is not found (as user trying to handle
				// this case). But if user did not set optional flag, then we
				// should throw error if field is not found & v.config.Strict.
				strict = false
			}
			return v.ident(node, name.Value, strict, false /* no builtins and no functions */)
		}
		return unknown
	}

	base := v.visit(node.Node)
	prop := v.visit(node.Property)

	if isUnknown(base) {
		return unknown
	}

	if name, ok := node.Property.(*ast.StringNode); ok {
		if isNil(base) {
			return v.error(node, "type nil has no field %v", name.Value)
		}

		// First, check methods defined on base type itself,
		// independent of which type it is. Without dereferencing.
		if m, ok := base.MethodByName(name.Value); ok {
			return m
		}
	}

	base = base.Deref()

	switch base.Kind() {
	case reflect.Map:
		if !prop.AssignableTo(base.Key()) && !isUnknown(prop) {
			return v.error(node.Property, "cannot use %v to get an element from %v", prop, base)
		}
		if prop, ok := node.Property.(*ast.StringNode); ok {
			if field, ok := base.Fields[prop.Value]; ok {
				return field
			} else if base.Strict {
				return v.error(node.Property, "unknown field %v", prop.Value)
			}
		}
		return base.Elem()

	case reflect.Array, reflect.Slice:
		if !isInteger(prop) && !isUnknown(prop) {
			return v.error(node.Property, "array elements can only be selected using an integer (got %v)", prop)
		}
		return base.Elem()

	case reflect.Struct:
		if name, ok := node.Property.(*ast.StringNode); ok {
			propertyName := name.Value
			if field, ok := base.FieldByName(propertyName); ok {
				return Nature{Type: field.Type}
			}
			if node.Method {
				return v.error(node, "type %v has no method %v", base, propertyName)
			}
			return v.error(node, "type %v has no field %v", base, propertyName)
		}
	}

	// Not found.

	if name, ok := node.Property.(*ast.StringNode); ok {
		if node.Method {
			return v.error(node, "type %v has no method %v", base, name.Value)
		}
		return v.error(node, "type %v has no field %v", base, name.Value)
	}
	return v.error(node, "type %v[%v] is undefined", base, prop)
}

func (v *checker) SliceNode(node *ast.SliceNode) Nature {
	nt := v.visit(node.Node)

	if isUnknown(nt) {
		return unknown
	}

	switch nt.Kind() {
	case reflect.String, reflect.Array, reflect.Slice:
		// ok
	default:
		return v.error(node, "cannot slice %s", nt)
	}

	if node.From != nil {
		from := v.visit(node.From)
		if !isInteger(from) && !isUnknown(from) {
			return v.error(node.From, "non-integer slice index %v", from)
		}
	}

	if node.To != nil {
		to := v.visit(node.To)
		if !isInteger(to) && !isUnknown(to) {
			return v.error(node.To, "non-integer slice index %v", to)
		}
	}

	return nt
}

func (v *checker) CallNode(node *ast.CallNode) Nature {
	nt := v.functionReturnType(node)

	// Check if type was set on node (for example, by patcher)
	// and use node type instead of function return type.
	//
	// If node type is anyType, then we should use function
	// return type. For example, on error we return anyType
	// for a call `errCall().Method()` and method will be
	// evaluated on `anyType.Method()`, so return type will
	// be anyType `anyType.Method(): anyType`. Patcher can
	// fix `errCall()` to return proper type, so on second
	// checker pass we should replace anyType on method node
	// with new correct function return type.
	if node.Type() != nil && node.Type() != anyType {
		return node.Nature()
	}

	return nt
}

func (v *checker) functionReturnType(node *ast.CallNode) Nature {
	nt := v.visit(node.Callee)

	if nt.Func != nil {
		return v.checkFunction(nt.Func, node, node.Arguments)
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

	if isUnknown(nt) {
		return unknown
	}

	if isNil(nt) {
		return v.error(node, "%v is nil; cannot call nil as function", fnName)
	}

	switch nt.Kind() {
	case reflect.Func:
		outType, err := v.checkArguments(fnName, nt, node.Arguments, node)
		if err != nil {
			if v.err == nil {
				v.err = err
			}
			return unknown
		}
		return outType
	}
	return v.error(node, "%s is not callable", nt)
}

func (v *checker) BuiltinNode(node *ast.BuiltinNode) Nature {
	switch node.Name {
	case "all", "none", "any", "one":
		collection := v.visit(node.Arguments[0]).Deref()
		if !isArray(collection) && !isUnknown(collection) {
			return v.error(node.Arguments[0], "builtin %v takes only array (got %v)", node.Name, collection)
		}

		v.begin(collection)
		predicate := v.visit(node.Arguments[1])
		v.end()

		if isFunc(predicate) &&
			predicate.NumOut() == 1 &&
			predicate.NumIn() == 1 && isUnknown(predicate.In(0)) {

			if !isBool(predicate.Out(0)) && !isUnknown(predicate.Out(0)) {
				return v.error(node.Arguments[1], "predicate should return boolean (got %v)", predicate.Out(0).String())
			}
			return boolNature
		}
		return v.error(node.Arguments[1], "predicate should has one input and one output param")

	case "filter":
		collection := v.visit(node.Arguments[0]).Deref()
		if !isArray(collection) && !isUnknown(collection) {
			return v.error(node.Arguments[0], "builtin %v takes only array (got %v)", node.Name, collection)
		}

		v.begin(collection)
		predicate := v.visit(node.Arguments[1])
		v.end()

		if isFunc(predicate) &&
			predicate.NumOut() == 1 &&
			predicate.NumIn() == 1 && isUnknown(predicate.In(0)) {

			if !isBool(predicate.Out(0)) && !isUnknown(predicate.Out(0)) {
				return v.error(node.Arguments[1], "predicate should return boolean (got %v)", predicate.Out(0).String())
			}
			if isUnknown(collection) {
				return arrayNature
			}
			return arrayOf(collection.Elem())
		}
		return v.error(node.Arguments[1], "predicate should has one input and one output param")

	case "map":
		collection := v.visit(node.Arguments[0]).Deref()
		if !isArray(collection) && !isUnknown(collection) {
			return v.error(node.Arguments[0], "builtin %v takes only array (got %v)", node.Name, collection)
		}

		v.begin(collection, scopeVar{"index", integerNature})
		predicate := v.visit(node.Arguments[1])
		v.end()

		if isFunc(predicate) &&
			predicate.NumOut() == 1 &&
			predicate.NumIn() == 1 && isUnknown(predicate.In(0)) {

			return arrayOf(*predicate.PredicateOut)
		}
		return v.error(node.Arguments[1], "predicate should has one input and one output param")

	case "count":
		collection := v.visit(node.Arguments[0]).Deref()
		if !isArray(collection) && !isUnknown(collection) {
			return v.error(node.Arguments[0], "builtin %v takes only array (got %v)", node.Name, collection)
		}

		if len(node.Arguments) == 1 {
			return integerNature
		}

		v.begin(collection)
		predicate := v.visit(node.Arguments[1])
		v.end()

		if isFunc(predicate) &&
			predicate.NumOut() == 1 &&
			predicate.NumIn() == 1 && isUnknown(predicate.In(0)) {
			if !isBool(predicate.Out(0)) && !isUnknown(predicate.Out(0)) {
				return v.error(node.Arguments[1], "predicate should return boolean (got %v)", predicate.Out(0).String())
			}

			return integerNature
		}
		return v.error(node.Arguments[1], "predicate should has one input and one output param")

	case "sum":
		collection := v.visit(node.Arguments[0]).Deref()
		if !isArray(collection) && !isUnknown(collection) {
			return v.error(node.Arguments[0], "builtin %v takes only array (got %v)", node.Name, collection)
		}

		if len(node.Arguments) == 2 {
			v.begin(collection)
			predicate := v.visit(node.Arguments[1])
			v.end()

			if isFunc(predicate) &&
				predicate.NumOut() == 1 &&
				predicate.NumIn() == 1 && isUnknown(predicate.In(0)) {
				return predicate.Out(0)
			}
		} else {
			if isUnknown(collection) {
				return unknown
			}
			return collection.Elem()
		}

	case "find", "findLast":
		collection := v.visit(node.Arguments[0]).Deref()
		if !isArray(collection) && !isUnknown(collection) {
			return v.error(node.Arguments[0], "builtin %v takes only array (got %v)", node.Name, collection)
		}

		v.begin(collection)
		predicate := v.visit(node.Arguments[1])
		v.end()

		if isFunc(predicate) &&
			predicate.NumOut() == 1 &&
			predicate.NumIn() == 1 && isUnknown(predicate.In(0)) {

			if !isBool(predicate.Out(0)) && !isUnknown(predicate.Out(0)) {
				return v.error(node.Arguments[1], "predicate should return boolean (got %v)", predicate.Out(0).String())
			}
			if isUnknown(collection) {
				return unknown
			}
			return collection.Elem()
		}
		return v.error(node.Arguments[1], "predicate should has one input and one output param")

	case "findIndex", "findLastIndex":
		collection := v.visit(node.Arguments[0]).Deref()
		if !isArray(collection) && !isUnknown(collection) {
			return v.error(node.Arguments[0], "builtin %v takes only array (got %v)", node.Name, collection)
		}

		v.begin(collection)
		predicate := v.visit(node.Arguments[1])
		v.end()

		if isFunc(predicate) &&
			predicate.NumOut() == 1 &&
			predicate.NumIn() == 1 && isUnknown(predicate.In(0)) {

			if !isBool(predicate.Out(0)) && !isUnknown(predicate.Out(0)) {
				return v.error(node.Arguments[1], "predicate should return boolean (got %v)", predicate.Out(0).String())
			}
			return integerNature
		}
		return v.error(node.Arguments[1], "predicate should has one input and one output param")

	case "groupBy":
		collection := v.visit(node.Arguments[0]).Deref()
		if !isArray(collection) && !isUnknown(collection) {
			return v.error(node.Arguments[0], "builtin %v takes only array (got %v)", node.Name, collection)
		}

		v.begin(collection)
		predicate := v.visit(node.Arguments[1])
		v.end()

		if isFunc(predicate) &&
			predicate.NumOut() == 1 &&
			predicate.NumIn() == 1 && isUnknown(predicate.In(0)) {

			groups := arrayOf(collection.Elem())
			return Nature{Type: reflect.TypeOf(map[any][]any{}), ArrayOf: &groups}
		}
		return v.error(node.Arguments[1], "predicate should has one input and one output param")

	case "sortBy":
		collection := v.visit(node.Arguments[0]).Deref()
		if !isArray(collection) && !isUnknown(collection) {
			return v.error(node.Arguments[0], "builtin %v takes only array (got %v)", node.Name, collection)
		}

		v.begin(collection)
		predicate := v.visit(node.Arguments[1])
		v.end()

		if len(node.Arguments) == 3 {
			_ = v.visit(node.Arguments[2])
		}

		if isFunc(predicate) &&
			predicate.NumOut() == 1 &&
			predicate.NumIn() == 1 && isUnknown(predicate.In(0)) {

			return collection
		}
		return v.error(node.Arguments[1], "predicate should has one input and one output param")

	case "reduce":
		collection := v.visit(node.Arguments[0]).Deref()
		if !isArray(collection) && !isUnknown(collection) {
			return v.error(node.Arguments[0], "builtin %v takes only array (got %v)", node.Name, collection)
		}

		v.begin(collection, scopeVar{"index", integerNature}, scopeVar{"acc", unknown})
		predicate := v.visit(node.Arguments[1])
		v.end()

		if len(node.Arguments) == 3 {
			_ = v.visit(node.Arguments[2])
		}

		if isFunc(predicate) && predicate.NumOut() == 1 {
			return *predicate.PredicateOut
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
	varName   string
	varNature Nature
}

func (v *checker) begin(collectionNature Nature, vars ...scopeVar) {
	scope := predicateScope{collection: collectionNature, vars: make(map[string]Nature)}
	for _, v := range vars {
		scope.vars[v.varName] = v.varNature
	}
	v.predicateScopes = append(v.predicateScopes, scope)
}

func (v *checker) end() {
	v.predicateScopes = v.predicateScopes[:len(v.predicateScopes)-1]
}

func (v *checker) checkBuiltinGet(node *ast.BuiltinNode) Nature {
	if len(node.Arguments) != 2 {
		return v.error(node, "invalid number of arguments (expected 2, got %d)", len(node.Arguments))
	}

	base := v.visit(node.Arguments[0])
	prop := v.visit(node.Arguments[1])

	if id, ok := node.Arguments[0].(*ast.IdentifierNode); ok && id.Value == "$env" {
		if s, ok := node.Arguments[1].(*ast.StringNode); ok {
			if nt, ok := v.config.Env.Get(s.Value); ok {
				return nt
			}
		}
		return unknown
	}

	if isUnknown(base) {
		return unknown
	}

	switch base.Kind() {
	case reflect.Slice, reflect.Array:
		if !isInteger(prop) && !isUnknown(prop) {
			return v.error(node.Arguments[1], "non-integer slice index %s", prop)
		}
		return base.Elem()
	case reflect.Map:
		if !prop.AssignableTo(base.Key()) && !isUnknown(prop) {
			return v.error(node.Arguments[1], "cannot use %s to get an element from %s", prop, base)
		}
		return base.Elem()
	}
	return v.error(node.Arguments[0], "type %v does not support indexing", base)
}

func (v *checker) checkFunction(f *builtin.Function, node ast.Node, arguments []ast.Node) Nature {
	if f.Validate != nil {
		args := make([]reflect.Type, len(arguments))
		for i, arg := range arguments {
			argNature := v.visit(arg)
			if isUnknown(argNature) {
				args[i] = anyType
			} else {
				args[i] = argNature.Type
			}
		}
		t, err := f.Validate(args)
		if err != nil {
			return v.error(node, "%v", err)
		}
		return Nature{Type: t}
	} else if len(f.Types) == 0 {
		nt, err := v.checkArguments(f.Name, Nature{Type: f.Type()}, arguments, node)
		if err != nil {
			if v.err == nil {
				v.err = err
			}
			return unknown
		}
		// No type was specified, so we assume the function returns any.
		return nt
	}
	var lastErr *file.Error
	for _, t := range f.Types {
		outNature, err := v.checkArguments(f.Name, Nature{Type: t}, arguments, node)
		if err != nil {
			lastErr = err
			continue
		}

		// As we found the correct function overload, we can stop the loop.
		// Also, we need to set the correct nature of the callee so compiler,
		// can correctly handle OpDeref opcode.
		if callNode, ok := node.(*ast.CallNode); ok {
			callNode.Callee.SetType(t)
		}

		return outNature
	}
	if lastErr != nil {
		if v.err == nil {
			v.err = lastErr
		}
		return unknown
	}

	return v.error(node, "no matching overload for %v", f.Name)
}

func (v *checker) checkArguments(
	name string,
	fn Nature,
	arguments []ast.Node,
	node ast.Node,
) (Nature, *file.Error) {
	if isUnknown(fn) {
		return unknown, nil
	}

	if fn.NumOut() == 0 {
		return unknown, &file.Error{
			Location: node.Location(),
			Message:  fmt.Sprintf("func %v doesn't return value", name),
		}
	}
	if numOut := fn.NumOut(); numOut > 2 {
		return unknown, &file.Error{
			Location: node.Location(),
			Message:  fmt.Sprintf("func %v returns more then two values", name),
		}
	}

	// If func is method on an env, first argument should be a receiver,
	// and actual arguments less than fnNumIn by one.
	fnNumIn := fn.NumIn()
	if fn.Method { // TODO: Move subtraction to the Nature.NumIn() and Nature.In() methods.
		fnNumIn--
	}
	// Skip first argument in case of the receiver.
	fnInOffset := 0
	if fn.Method {
		fnInOffset = 1
	}

	var err *file.Error
	if fn.IsVariadic() {
		if len(arguments) < fnNumIn-1 {
			err = &file.Error{
				Location: node.Location(),
				Message:  fmt.Sprintf("not enough arguments to call %v", name),
			}
		}
	} else {
		if len(arguments) > fnNumIn {
			err = &file.Error{
				Location: node.Location(),
				Message:  fmt.Sprintf("too many arguments to call %v", name),
			}
		}
		if len(arguments) < fnNumIn {
			err = &file.Error{
				Location: node.Location(),
				Message:  fmt.Sprintf("not enough arguments to call %v", name),
			}
		}
	}

	if err != nil {
		// If we have an error, we should still visit all arguments to
		// type check them, as a patch can fix the error later.
		for _, arg := range arguments {
			_ = v.visit(arg)
		}
		return fn.Out(0), err
	}

	for i, arg := range arguments {
		argNature := v.visit(arg)

		var in Nature
		if fn.IsVariadic() && i >= fnNumIn-1 {
			// For variadic arguments fn(xs ...int), go replaces type of xs (int) with ([]int).
			// As we compare arguments one by one, we need underling type.
			in = fn.In(fn.NumIn() - 1).Elem()
		} else {
			in = fn.In(i + fnInOffset)
		}

		if isFloat(in) && isInteger(argNature) {
			traverseAndReplaceIntegerNodesWithFloatNodes(&arguments[i], in)
			continue
		}

		if isInteger(in) && isInteger(argNature) && argNature.Kind() != in.Kind() {
			traverseAndReplaceIntegerNodesWithIntegerNodes(&arguments[i], in)
			continue
		}

		if isNil(argNature) {
			if in.Kind() == reflect.Ptr || in.Kind() == reflect.Interface {
				continue
			}
			return unknown, &file.Error{
				Location: arg.Location(),
				Message:  fmt.Sprintf("cannot use nil as argument (type %s) to call %v", in, name),
			}
		}

		// Check if argument is assignable to the function input type.
		// We check original type (like *time.Time), not dereferenced type,
		// as function input type can be pointer to a struct.
		assignable := argNature.AssignableTo(in)

		// We also need to check if dereference arg type is assignable to the function input type.
		// For example, func(int) and argument *int. In this case we will add OpDeref to the argument,
		// so we can call the function with *int argument.
		assignable = assignable || argNature.Deref().AssignableTo(in)

		if !assignable && !isUnknown(argNature) {
			return unknown, &file.Error{
				Location: arg.Location(),
				Message:  fmt.Sprintf("cannot use %s as argument (type %s) to call %v ", argNature, in, name),
			}
		}
	}

	return fn.Out(0), nil
}

func traverseAndReplaceIntegerNodesWithFloatNodes(node *ast.Node, newNature Nature) {
	switch (*node).(type) {
	case *ast.IntegerNode:
		*node = &ast.FloatNode{Value: float64((*node).(*ast.IntegerNode).Value)}
		(*node).SetType(newNature.Type)
	case *ast.UnaryNode:
		unaryNode := (*node).(*ast.UnaryNode)
		traverseAndReplaceIntegerNodesWithFloatNodes(&unaryNode.Node, newNature)
	case *ast.BinaryNode:
		binaryNode := (*node).(*ast.BinaryNode)
		switch binaryNode.Operator {
		case "+", "-", "*":
			traverseAndReplaceIntegerNodesWithFloatNodes(&binaryNode.Left, newNature)
			traverseAndReplaceIntegerNodesWithFloatNodes(&binaryNode.Right, newNature)
		}
	}
}

func traverseAndReplaceIntegerNodesWithIntegerNodes(node *ast.Node, newNature Nature) {
	switch (*node).(type) {
	case *ast.IntegerNode:
		(*node).SetType(newNature.Type)
	case *ast.UnaryNode:
		(*node).SetType(newNature.Type)
		unaryNode := (*node).(*ast.UnaryNode)
		traverseAndReplaceIntegerNodesWithIntegerNodes(&unaryNode.Node, newNature)
	case *ast.BinaryNode:
		// TODO: Binary node return type is dependent on the type of the operands. We can't just change the type of the node.
		binaryNode := (*node).(*ast.BinaryNode)
		switch binaryNode.Operator {
		case "+", "-", "*":
			traverseAndReplaceIntegerNodesWithIntegerNodes(&binaryNode.Left, newNature)
			traverseAndReplaceIntegerNodesWithIntegerNodes(&binaryNode.Right, newNature)
		}
	}
}

func (v *checker) PredicateNode(node *ast.PredicateNode) Nature {
	nt := v.visit(node.Node)
	var out []reflect.Type
	if isUnknown(nt) {
		out = append(out, anyType)
	} else if !isNil(nt) {
		out = append(out, nt.Type)
	}
	return Nature{
		Type:         reflect.FuncOf([]reflect.Type{anyType}, out, false),
		PredicateOut: &nt,
	}
}

func (v *checker) PointerNode(node *ast.PointerNode) Nature {
	if len(v.predicateScopes) == 0 {
		return v.error(node, "cannot use pointer accessor outside predicate")
	}
	scope := v.predicateScopes[len(v.predicateScopes)-1]
	if node.Name == "" {
		if isUnknown(scope.collection) {
			return unknown
		}
		switch scope.collection.Kind() {
		case reflect.Array, reflect.Slice:
			return scope.collection.Elem()
		}
		return v.error(node, "cannot use %v as array", scope)
	}
	if scope.vars != nil {
		if t, ok := scope.vars[node.Name]; ok {
			return t
		}
	}
	return v.error(node, "unknown pointer #%v", node.Name)
}

func (v *checker) VariableDeclaratorNode(node *ast.VariableDeclaratorNode) Nature {
	if _, ok := v.config.Env.Get(node.Name); ok {
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
	varNature := v.visit(node.Value)
	v.varScopes = append(v.varScopes, varScope{node.Name, varNature})
	exprNature := v.visit(node.Expr)
	v.varScopes = v.varScopes[:len(v.varScopes)-1]
	return exprNature
}

func (v *checker) SequenceNode(node *ast.SequenceNode) Nature {
	if len(node.Nodes) == 0 {
		return v.error(node, "empty sequence expression")
	}
	var last Nature
	for _, node := range node.Nodes {
		last = v.visit(node)
	}
	return last
}

func (v *checker) lookupVariable(name string) (varScope, bool) {
	for i := len(v.varScopes) - 1; i >= 0; i-- {
		if v.varScopes[i].name == name {
			return v.varScopes[i], true
		}
	}
	return varScope{}, false
}

func (v *checker) ConditionalNode(node *ast.ConditionalNode) Nature {
	c := v.visit(node.Cond)
	if !isBool(c) && !isUnknown(c) {
		return v.error(node.Cond, "non-bool expression (type %v) used as condition", c)
	}

	t1 := v.visit(node.Exp1)
	t2 := v.visit(node.Exp2)

	if isNil(t1) && !isNil(t2) {
		return t2
	}
	if !isNil(t1) && isNil(t2) {
		return t1
	}
	if isNil(t1) && isNil(t2) {
		return nilNature
	}
	if t1.AssignableTo(t2) {
		return t1
	}
	return unknown
}

func (v *checker) ArrayNode(node *ast.ArrayNode) Nature {
	var prev Nature
	allElementsAreSameType := true
	for i, node := range node.Nodes {
		curr := v.visit(node)
		if i > 0 {
			if curr.Kind() != prev.Kind() {
				allElementsAreSameType = false
			}
		}
		prev = curr
	}
	if allElementsAreSameType {
		return arrayOf(prev)
	}
	return arrayNature
}

func (v *checker) MapNode(node *ast.MapNode) Nature {
	for _, pair := range node.Pairs {
		v.visit(pair)
	}
	return mapNature
}

func (v *checker) PairNode(node *ast.PairNode) Nature {
	v.visit(node.Key)
	v.visit(node.Value)
	return nilNature
}
