package otto

import (
	"fmt"
	goruntime "runtime"

	"github.com/robertkrimen/otto/token"
)

func (rt *runtime) cmplEvaluateNodeStatement(node nodeStatement) Value {
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
	case *nodeBlockStatement:
		labels := rt.labels
		rt.labels = nil

		value := rt.cmplEvaluateNodeStatementList(node.list)
		if value.kind == valueResult {
			if value.evaluateBreak(labels) == resultBreak {
				return emptyValue
			}
		}
		return value

	case *nodeBranchStatement:
		target := node.label
		switch node.branch { // FIXME Maybe node.kind? node.operator?
		case token.BREAK:
			return toValue(newBreakResult(target))
		case token.CONTINUE:
			return toValue(newContinueResult(target))
		default:
			panic(fmt.Errorf("unknown node branch token %T", node))
		}

	case *nodeDebuggerStatement:
		if rt.debugger != nil {
			rt.debugger(rt.otto)
		}
		return emptyValue // Nothing happens.

	case *nodeDoWhileStatement:
		return rt.cmplEvaluateNodeDoWhileStatement(node)

	case *nodeEmptyStatement:
		return emptyValue

	case *nodeExpressionStatement:
		return rt.cmplEvaluateNodeExpression(node.expression)

	case *nodeForInStatement:
		return rt.cmplEvaluateNodeForInStatement(node)

	case *nodeForStatement:
		return rt.cmplEvaluateNodeForStatement(node)

	case *nodeIfStatement:
		return rt.cmplEvaluateNodeIfStatement(node)

	case *nodeLabelledStatement:
		rt.labels = append(rt.labels, node.label)
		defer func() {
			if len(rt.labels) > 0 {
				rt.labels = rt.labels[:len(rt.labels)-1] // Pop the label
			} else {
				rt.labels = nil
			}
		}()
		return rt.cmplEvaluateNodeStatement(node.statement)

	case *nodeReturnStatement:
		if node.argument != nil {
			return toValue(newReturnResult(rt.cmplEvaluateNodeExpression(node.argument).resolve()))
		}
		return toValue(newReturnResult(Value{}))

	case *nodeSwitchStatement:
		return rt.cmplEvaluateNodeSwitchStatement(node)

	case *nodeThrowStatement:
		value := rt.cmplEvaluateNodeExpression(node.argument).resolve()
		panic(newException(value))

	case *nodeTryStatement:
		return rt.cmplEvaluateNodeTryStatement(node)

	case *nodeVariableStatement:
		// Variables are already defined, this is initialization only
		for _, variable := range node.list {
			rt.cmplEvaluateNodeVariableExpression(variable.(*nodeVariableExpression))
		}
		return emptyValue

	case *nodeWhileStatement:
		return rt.cmplEvaluateModeWhileStatement(node)

	case *nodeWithStatement:
		return rt.cmplEvaluateNodeWithStatement(node)
	default:
		panic(fmt.Errorf("unknown node statement type %T", node))
	}
}

func (rt *runtime) cmplEvaluateNodeStatementList(list []nodeStatement) Value {
	var result Value
	for _, node := range list {
		value := rt.cmplEvaluateNodeStatement(node)
		switch value.kind {
		case valueResult:
			return value
		case valueEmpty:
		default:
			// We have getValue here to (for example) trigger a
			// ReferenceError (of the not defined variety)
			// Not sure if this is the best way to error out early
			// for such errors or if there is a better way
			// TODO Do we still need this?
			result = value.resolve()
		}
	}
	return result
}

func (rt *runtime) cmplEvaluateNodeDoWhileStatement(node *nodeDoWhileStatement) Value {
	labels := append(rt.labels, "") //nolint:gocritic
	rt.labels = nil

	test := node.test

	result := emptyValue
resultBreak:
	for {
		for _, node := range node.body {
			value := rt.cmplEvaluateNodeStatement(node)
			switch value.kind {
			case valueResult:
				switch value.evaluateBreakContinue(labels) {
				case resultReturn:
					return value
				case resultBreak:
					break resultBreak
				case resultContinue:
					goto resultContinue
				}
			case valueEmpty:
			default:
				result = value
			}
		}
	resultContinue:
		if !rt.cmplEvaluateNodeExpression(test).resolve().bool() {
			// Stahp: do ... while (false)
			break
		}
	}
	return result
}

func (rt *runtime) cmplEvaluateNodeForInStatement(node *nodeForInStatement) Value {
	labels := append(rt.labels, "") //nolint:gocritic
	rt.labels = nil

	source := rt.cmplEvaluateNodeExpression(node.source)
	sourceValue := source.resolve()

	switch sourceValue.kind {
	case valueUndefined, valueNull:
		return emptyValue
	}

	sourceObject := rt.toObject(sourceValue)

	into := node.into
	body := node.body

	result := emptyValue
	obj := sourceObject
	for obj != nil {
		enumerateValue := emptyValue
		obj.enumerate(false, func(name string) bool {
			into := rt.cmplEvaluateNodeExpression(into)
			// In the case of: for (var abc in def) ...
			if into.reference() == nil {
				identifier := into.string()
				// TODO Should be true or false (strictness) depending on context
				into = toValue(getIdentifierReference(rt, rt.scope.lexical, identifier, false, -1))
			}
			rt.putValue(into.reference(), stringValue(name))
			for _, node := range body {
				value := rt.cmplEvaluateNodeStatement(node)
				switch value.kind {
				case valueResult:
					switch value.evaluateBreakContinue(labels) {
					case resultReturn:
						enumerateValue = value
						return false
					case resultBreak:
						obj = nil
						return false
					case resultContinue:
						return true
					}
				case valueEmpty:
				default:
					enumerateValue = value
				}
			}
			return true
		})
		if obj == nil {
			break
		}
		obj = obj.prototype
		if !enumerateValue.isEmpty() {
			result = enumerateValue
		}
	}
	return result
}

func (rt *runtime) cmplEvaluateNodeForStatement(node *nodeForStatement) Value {
	labels := append(rt.labels, "") //nolint:gocritic
	rt.labels = nil

	initializer := node.initializer
	test := node.test
	update := node.update
	body := node.body

	if initializer != nil {
		initialResult := rt.cmplEvaluateNodeExpression(initializer)
		initialResult.resolve() // Side-effect trigger
	}

	result := emptyValue
resultBreak:
	for {
		if test != nil {
			testResult := rt.cmplEvaluateNodeExpression(test)
			testResultValue := testResult.resolve()
			if !testResultValue.bool() {
				break
			}
		}

		// this is to prevent for cycles with no body from running forever
		if len(body) == 0 && rt.otto.Interrupt != nil {
			goruntime.Gosched()
			select {
			case value := <-rt.otto.Interrupt:
				value()
			default:
			}
		}

		for _, node := range body {
			value := rt.cmplEvaluateNodeStatement(node)
			switch value.kind {
			case valueResult:
				switch value.evaluateBreakContinue(labels) {
				case resultReturn:
					return value
				case resultBreak:
					break resultBreak
				case resultContinue:
					goto resultContinue
				}
			case valueEmpty:
			default:
				result = value
			}
		}
	resultContinue:
		if update != nil {
			updateResult := rt.cmplEvaluateNodeExpression(update)
			updateResult.resolve() // Side-effect trigger
		}
	}
	return result
}

func (rt *runtime) cmplEvaluateNodeIfStatement(node *nodeIfStatement) Value {
	test := rt.cmplEvaluateNodeExpression(node.test)
	testValue := test.resolve()
	if testValue.bool() {
		return rt.cmplEvaluateNodeStatement(node.consequent)
	} else if node.alternate != nil {
		return rt.cmplEvaluateNodeStatement(node.alternate)
	}

	return emptyValue
}

func (rt *runtime) cmplEvaluateNodeSwitchStatement(node *nodeSwitchStatement) Value {
	labels := append(rt.labels, "") //nolint:gocritic
	rt.labels = nil

	discriminantResult := rt.cmplEvaluateNodeExpression(node.discriminant)
	target := node.defaultIdx

	for index, clause := range node.body {
		test := clause.test
		if test != nil {
			if rt.calculateComparison(token.STRICT_EQUAL, discriminantResult, rt.cmplEvaluateNodeExpression(test)) {
				target = index
				break
			}
		}
	}

	result := emptyValue
	if target != -1 {
		for _, clause := range node.body[target:] {
			for _, statement := range clause.consequent {
				value := rt.cmplEvaluateNodeStatement(statement)
				switch value.kind {
				case valueResult:
					switch value.evaluateBreak(labels) {
					case resultReturn:
						return value
					case resultBreak:
						return emptyValue
					}
				case valueEmpty:
				default:
					result = value
				}
			}
		}
	}

	return result
}

func (rt *runtime) cmplEvaluateNodeTryStatement(node *nodeTryStatement) Value {
	tryCatchValue, exep := rt.tryCatchEvaluate(func() Value {
		return rt.cmplEvaluateNodeStatement(node.body)
	})

	if exep && node.catch != nil {
		outer := rt.scope.lexical
		rt.scope.lexical = rt.newDeclarationStash(outer)
		defer func() {
			rt.scope.lexical = outer
		}()
		// TODO If necessary, convert TypeError<runtime> => TypeError
		// That, is, such errors can be thrown despite not being JavaScript "native"
		// strict = false
		rt.scope.lexical.setValue(node.catch.parameter, tryCatchValue, false)

		// FIXME node.CatchParameter
		// FIXME node.Catch
		tryCatchValue, exep = rt.tryCatchEvaluate(func() Value {
			return rt.cmplEvaluateNodeStatement(node.catch.body)
		})
	}

	if node.finally != nil {
		finallyValue := rt.cmplEvaluateNodeStatement(node.finally)
		if finallyValue.kind == valueResult {
			return finallyValue
		}
	}

	if exep {
		panic(newException(tryCatchValue))
	}

	return tryCatchValue
}

func (rt *runtime) cmplEvaluateModeWhileStatement(node *nodeWhileStatement) Value {
	test := node.test
	body := node.body
	labels := append(rt.labels, "") //nolint:gocritic
	rt.labels = nil

	result := emptyValue
resultBreakContinue:
	for {
		if !rt.cmplEvaluateNodeExpression(test).resolve().bool() {
			// Stahp: while (false) ...
			break
		}
		for _, node := range body {
			value := rt.cmplEvaluateNodeStatement(node)
			switch value.kind {
			case valueResult:
				switch value.evaluateBreakContinue(labels) {
				case resultReturn:
					return value
				case resultBreak:
					break resultBreakContinue
				case resultContinue:
					continue resultBreakContinue
				}
			case valueEmpty:
			default:
				result = value
			}
		}
	}
	return result
}

func (rt *runtime) cmplEvaluateNodeWithStatement(node *nodeWithStatement) Value {
	obj := rt.cmplEvaluateNodeExpression(node.object)
	outer := rt.scope.lexical
	lexical := rt.newObjectStash(rt.toObject(obj.resolve()), outer)
	rt.scope.lexical = lexical
	defer func() {
		rt.scope.lexical = outer
	}()

	return rt.cmplEvaluateNodeStatement(node.body)
}
