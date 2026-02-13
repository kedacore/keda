package otto

import (
	"strconv"
)

func (rt *runtime) cmplEvaluateNodeProgram(node *nodeProgram, eval bool) Value {
	if !eval {
		rt.enterGlobalScope()
		defer rt.leaveScope()
	}
	rt.cmplFunctionDeclaration(node.functionList)
	rt.cmplVariableDeclaration(node.varList)
	rt.scope.frame.file = node.file
	return rt.cmplEvaluateNodeStatementList(node.body)
}

func (rt *runtime) cmplCallNodeFunction(function *object, stash *fnStash, node *nodeFunctionLiteral, argumentList []Value) Value {
	indexOfParameterName := make([]string, len(argumentList))
	// function(abc, def, ghi)
	// indexOfParameterName[0] = "abc"
	// indexOfParameterName[1] = "def"
	// indexOfParameterName[2] = "ghi"
	// ...

	argumentsFound := false
	for index, name := range node.parameterList {
		if name == "arguments" {
			argumentsFound = true
		}
		value := Value{}
		if index < len(argumentList) {
			value = argumentList[index]
			indexOfParameterName[index] = name
		}
		// strict = false
		rt.scope.lexical.setValue(name, value, false)
	}

	if !argumentsFound {
		arguments := rt.newArgumentsObject(indexOfParameterName, stash, len(argumentList))
		arguments.defineProperty("callee", objectValue(function), 0o101, false)
		stash.arguments = arguments
		// strict = false
		rt.scope.lexical.setValue("arguments", objectValue(arguments), false)
		for index := range argumentList {
			if index < len(node.parameterList) {
				continue
			}
			indexAsString := strconv.FormatInt(int64(index), 10)
			arguments.defineProperty(indexAsString, argumentList[index], 0o111, false)
		}
	}

	rt.cmplFunctionDeclaration(node.functionList)
	rt.cmplVariableDeclaration(node.varList)

	result := rt.cmplEvaluateNodeStatement(node.body)
	if result.kind == valueResult {
		return result
	}

	return Value{}
}

func (rt *runtime) cmplFunctionDeclaration(list []*nodeFunctionLiteral) {
	executionContext := rt.scope
	eval := executionContext.eval
	stash := executionContext.variable

	for _, function := range list {
		name := function.name
		value := rt.cmplEvaluateNodeExpression(function)
		if !stash.hasBinding(name) {
			stash.createBinding(name, eval, value)
		} else {
			// TODO 10.5.5.e
			stash.setBinding(name, value, false) // TODO strict
		}
	}
}

func (rt *runtime) cmplVariableDeclaration(list []string) {
	executionContext := rt.scope
	eval := executionContext.eval
	stash := executionContext.variable

	for _, name := range list {
		if !stash.hasBinding(name) {
			stash.createBinding(name, eval, Value{}) // TODO strict?
		}
	}
}
