package otto

import (
	"fmt"
	"math"
	goruntime "runtime"

	"github.com/robertkrimen/otto/token"
)

func (rt *runtime) cmplEvaluateNodeExpression(node nodeExpression) Value {
	// Allow interpreter interruption
	// If the Interrupt channel is nil, then
	// we avoid runtime.Gosched() overhead (if any)
	// FIXME: Test this
	if rt.otto.Interrupt != nil {
		goruntime.Gosched()
		select {
		case value := <-rt.otto.Interrupt:
			value()
		default:
		}
	}

	switch node := node.(type) {
	case *nodeArrayLiteral:
		return rt.cmplEvaluateNodeArrayLiteral(node)

	case *nodeAssignExpression:
		return rt.cmplEvaluateNodeAssignExpression(node)

	case *nodeBinaryExpression:
		if node.comparison {
			return rt.cmplEvaluateNodeBinaryExpressionComparison(node)
		}
		return rt.cmplEvaluateNodeBinaryExpression(node)

	case *nodeBracketExpression:
		return rt.cmplEvaluateNodeBracketExpression(node)

	case *nodeCallExpression:
		return rt.cmplEvaluateNodeCallExpression(node, nil)

	case *nodeConditionalExpression:
		return rt.cmplEvaluateNodeConditionalExpression(node)

	case *nodeDotExpression:
		return rt.cmplEvaluateNodeDotExpression(node)

	case *nodeFunctionLiteral:
		local := rt.scope.lexical
		if node.name != "" {
			local = rt.newDeclarationStash(local)
		}

		value := objectValue(rt.newNodeFunction(node, local))
		if node.name != "" {
			local.createBinding(node.name, false, value)
		}
		return value

	case *nodeIdentifier:
		name := node.name
		// TODO Should be true or false (strictness) depending on context
		// getIdentifierReference should not return nil, but we check anyway and panic
		// so as not to propagate the nil into something else
		reference := getIdentifierReference(rt, rt.scope.lexical, name, false, at(node.idx))
		if reference == nil {
			// Should never get here!
			panic(hereBeDragons("referenceError == nil: " + name))
		}
		return toValue(reference)

	case *nodeLiteral:
		return node.value

	case *nodeNewExpression:
		return rt.cmplEvaluateNodeNewExpression(node)

	case *nodeObjectLiteral:
		return rt.cmplEvaluateNodeObjectLiteral(node)

	case *nodeRegExpLiteral:
		return objectValue(rt.newRegExpDirect(node.pattern, node.flags))

	case *nodeSequenceExpression:
		return rt.cmplEvaluateNodeSequenceExpression(node)

	case *nodeThisExpression:
		return objectValue(rt.scope.this)

	case *nodeUnaryExpression:
		return rt.cmplEvaluateNodeUnaryExpression(node)

	case *nodeVariableExpression:
		return rt.cmplEvaluateNodeVariableExpression(node)
	default:
		panic(fmt.Sprintf("unknown node type: %T", node))
	}
}

func (rt *runtime) cmplEvaluateNodeArrayLiteral(node *nodeArrayLiteral) Value {
	valueArray := []Value{}

	for _, node := range node.value {
		if node == nil {
			valueArray = append(valueArray, emptyValue)
		} else {
			valueArray = append(valueArray, rt.cmplEvaluateNodeExpression(node).resolve())
		}
	}

	result := rt.newArrayOf(valueArray)

	return objectValue(result)
}

func (rt *runtime) cmplEvaluateNodeAssignExpression(node *nodeAssignExpression) Value {
	left := rt.cmplEvaluateNodeExpression(node.left)
	right := rt.cmplEvaluateNodeExpression(node.right)
	rightValue := right.resolve()

	result := rightValue
	if node.operator != token.ASSIGN {
		result = rt.calculateBinaryExpression(node.operator, left, rightValue)
	}

	rt.putValue(left.reference(), result)

	return result
}

func (rt *runtime) cmplEvaluateNodeBinaryExpression(node *nodeBinaryExpression) Value {
	left := rt.cmplEvaluateNodeExpression(node.left)
	leftValue := left.resolve()

	switch node.operator {
	// Logical
	case token.LOGICAL_AND:
		if !leftValue.bool() {
			return leftValue
		}
		right := rt.cmplEvaluateNodeExpression(node.right)
		return right.resolve()
	case token.LOGICAL_OR:
		if leftValue.bool() {
			return leftValue
		}
		right := rt.cmplEvaluateNodeExpression(node.right)
		return right.resolve()
	}

	return rt.calculateBinaryExpression(node.operator, leftValue, rt.cmplEvaluateNodeExpression(node.right))
}

func (rt *runtime) cmplEvaluateNodeBinaryExpressionComparison(node *nodeBinaryExpression) Value {
	left := rt.cmplEvaluateNodeExpression(node.left).resolve()
	right := rt.cmplEvaluateNodeExpression(node.right).resolve()

	return boolValue(rt.calculateComparison(node.operator, left, right))
}

func (rt *runtime) cmplEvaluateNodeBracketExpression(node *nodeBracketExpression) Value {
	target := rt.cmplEvaluateNodeExpression(node.left)
	targetValue := target.resolve()
	member := rt.cmplEvaluateNodeExpression(node.member)
	memberValue := member.resolve()

	// TODO Pass in base value as-is, and defer toObject till later?
	obj, err := rt.objectCoerce(targetValue)
	if err != nil {
		panic(rt.panicTypeError("Cannot access member %q of %s", memberValue.string(), err, at(node.idx)))
	}
	return toValue(newPropertyReference(rt, obj, memberValue.string(), false, at(node.idx)))
}

func (rt *runtime) cmplEvaluateNodeCallExpression(node *nodeCallExpression, withArgumentList []interface{}) Value {
	this := Value{}
	callee := rt.cmplEvaluateNodeExpression(node.callee)

	argumentList := []Value{}
	if withArgumentList != nil {
		argumentList = rt.toValueArray(withArgumentList...)
	} else {
		for _, argumentNode := range node.argumentList {
			argumentList = append(argumentList, rt.cmplEvaluateNodeExpression(argumentNode).resolve())
		}
	}

	eval := false // Whether this call is a (candidate for) direct call to eval
	name := ""
	if rf := callee.reference(); rf != nil {
		switch rf := rf.(type) {
		case *propertyReference:
			name = rf.name
			this = objectValue(rf.base)
			eval = rf.name == "eval" // Possible direct eval
		case *stashReference:
			// TODO ImplicitThisValue
			name = rf.name
			eval = rf.name == "eval" // Possible direct eval
		default:
			// FIXME?
			panic(rt.panicTypeError("unexpected callee type %T to node call expression", rf))
		}
	}

	atv := at(-1)
	switch callee := node.callee.(type) {
	case *nodeIdentifier:
		atv = at(callee.idx)
	case *nodeDotExpression:
		atv = at(callee.idx)
	case *nodeBracketExpression:
		atv = at(callee.idx)
	}

	frm := frame{
		callee: name,
		file:   rt.scope.frame.file,
	}

	vl := callee.resolve()
	if !vl.IsFunction() {
		if name == "" {
			// FIXME Maybe typeof?
			panic(rt.panicTypeError("%v is not a function", vl, atv))
		}
		panic(rt.panicTypeError("%q is not a function", name, atv))
	}

	rt.scope.frame.offset = int(atv)

	return vl.object().call(this, argumentList, eval, frm)
}

func (rt *runtime) cmplEvaluateNodeConditionalExpression(node *nodeConditionalExpression) Value {
	test := rt.cmplEvaluateNodeExpression(node.test)
	testValue := test.resolve()
	if testValue.bool() {
		return rt.cmplEvaluateNodeExpression(node.consequent)
	}
	return rt.cmplEvaluateNodeExpression(node.alternate)
}

func (rt *runtime) cmplEvaluateNodeDotExpression(node *nodeDotExpression) Value {
	target := rt.cmplEvaluateNodeExpression(node.left)
	targetValue := target.resolve()
	// TODO Pass in base value as-is, and defer toObject till later?
	obj, err := rt.objectCoerce(targetValue)
	if err != nil {
		panic(rt.panicTypeError("Cannot access member %q of %s", node.identifier, err, at(node.idx)))
	}
	return toValue(newPropertyReference(rt, obj, node.identifier, false, at(node.idx)))
}

func (rt *runtime) cmplEvaluateNodeNewExpression(node *nodeNewExpression) Value {
	callee := rt.cmplEvaluateNodeExpression(node.callee)

	argumentList := []Value{}
	for _, argumentNode := range node.argumentList {
		argumentList = append(argumentList, rt.cmplEvaluateNodeExpression(argumentNode).resolve())
	}

	var name string
	if rf := callee.reference(); rf != nil {
		switch rf := rf.(type) {
		case *propertyReference:
			name = rf.name
		case *stashReference:
			name = rf.name
		default:
			panic(rt.panicTypeError("node new expression unexpected callee type %T", rf))
		}
	}

	atv := at(-1)
	switch callee := node.callee.(type) {
	case *nodeIdentifier:
		atv = at(callee.idx)
	case *nodeDotExpression:
		atv = at(callee.idx)
	case *nodeBracketExpression:
		atv = at(callee.idx)
	}

	vl := callee.resolve()
	if !vl.IsFunction() {
		if name == "" {
			// FIXME Maybe typeof?
			panic(rt.panicTypeError("%v is not a function", vl, atv))
		}
		panic(rt.panicTypeError("'%s' is not a function", name, atv))
	}

	rt.scope.frame.offset = int(atv)

	return vl.object().construct(argumentList)
}

func (rt *runtime) cmplEvaluateNodeObjectLiteral(node *nodeObjectLiteral) Value {
	result := rt.newObject()
	for _, prop := range node.value {
		switch prop.kind {
		case "value":
			result.defineProperty(prop.key, rt.cmplEvaluateNodeExpression(prop.value).resolve(), 0o111, false)
		case "get":
			getter := rt.newNodeFunction(prop.value.(*nodeFunctionLiteral), rt.scope.lexical)
			descriptor := property{}
			descriptor.mode = 0o211
			descriptor.value = propertyGetSet{getter, nil}
			result.defineOwnProperty(prop.key, descriptor, false)
		case "set":
			setter := rt.newNodeFunction(prop.value.(*nodeFunctionLiteral), rt.scope.lexical)
			descriptor := property{}
			descriptor.mode = 0o211
			descriptor.value = propertyGetSet{nil, setter}
			result.defineOwnProperty(prop.key, descriptor, false)
		default:
			panic(fmt.Sprintf("unknown node object literal property kind %T", prop.kind))
		}
	}

	return objectValue(result)
}

func (rt *runtime) cmplEvaluateNodeSequenceExpression(node *nodeSequenceExpression) Value {
	var result Value
	for _, node := range node.sequence {
		result = rt.cmplEvaluateNodeExpression(node)
		result = result.resolve()
	}
	return result
}

func (rt *runtime) cmplEvaluateNodeUnaryExpression(node *nodeUnaryExpression) Value {
	target := rt.cmplEvaluateNodeExpression(node.operand)
	switch node.operator {
	case token.TYPEOF, token.DELETE:
		if target.kind == valueReference && target.reference().invalid() {
			if node.operator == token.TYPEOF {
				return stringValue("undefined")
			}
			return trueValue
		}
	}

	switch node.operator {
	case token.NOT:
		targetValue := target.resolve()
		if targetValue.bool() {
			return falseValue
		}
		return trueValue
	case token.BITWISE_NOT:
		targetValue := target.resolve()
		integerValue := toInt32(targetValue)
		return int32Value(^integerValue)
	case token.PLUS:
		targetValue := target.resolve()
		return float64Value(targetValue.float64())
	case token.MINUS:
		targetValue := target.resolve()
		value := targetValue.float64()
		// TODO Test this
		sign := float64(-1)
		if math.Signbit(value) {
			sign = 1
		}
		return float64Value(math.Copysign(value, sign))
	case token.INCREMENT:
		targetValue := target.resolve()
		if node.postfix {
			// Postfix++
			oldValue := targetValue.float64()
			newValue := float64Value(+1 + oldValue)
			rt.putValue(target.reference(), newValue)
			return float64Value(oldValue)
		}

		// ++Prefix
		newValue := float64Value(+1 + targetValue.float64())
		rt.putValue(target.reference(), newValue)
		return newValue
	case token.DECREMENT:
		targetValue := target.resolve()
		if node.postfix {
			// Postfix--
			oldValue := targetValue.float64()
			newValue := float64Value(-1 + oldValue)
			rt.putValue(target.reference(), newValue)
			return float64Value(oldValue)
		}

		// --Prefix
		newValue := float64Value(-1 + targetValue.float64())
		rt.putValue(target.reference(), newValue)
		return newValue
	case token.VOID:
		target.resolve() // FIXME Side effect?
		return Value{}
	case token.DELETE:
		reference := target.reference()
		if reference == nil {
			return trueValue
		}
		return boolValue(target.reference().delete())
	case token.TYPEOF:
		targetValue := target.resolve()
		switch targetValue.kind {
		case valueUndefined:
			return stringValue("undefined")
		case valueNull:
			return stringValue("object")
		case valueBoolean:
			return stringValue("boolean")
		case valueNumber:
			return stringValue("number")
		case valueString:
			return stringValue("string")
		case valueObject:
			if targetValue.object().isCall() {
				return stringValue("function")
			}
			return stringValue("object")
		default:
			// FIXME ?
		}
	}

	panic(hereBeDragons())
}

func (rt *runtime) cmplEvaluateNodeVariableExpression(node *nodeVariableExpression) Value {
	if node.initializer != nil {
		// FIXME If reference is nil
		left := getIdentifierReference(rt, rt.scope.lexical, node.name, false, at(node.idx))
		right := rt.cmplEvaluateNodeExpression(node.initializer)
		rightValue := right.resolve()

		rt.putValue(left, rightValue)
	}
	return stringValue(node.name)
}
