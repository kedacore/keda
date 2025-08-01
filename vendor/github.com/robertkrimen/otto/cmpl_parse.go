package otto

import (
	"fmt"

	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/file"
	"github.com/robertkrimen/otto/token"
)

var (
	trueLiteral    = &nodeLiteral{value: boolValue(true)}
	falseLiteral   = &nodeLiteral{value: boolValue(false)}
	nullLiteral    = &nodeLiteral{value: nullValue}
	emptyStatement = &nodeEmptyStatement{}
)

func (cmpl *compiler) parseExpression(expr ast.Expression) nodeExpression {
	if expr == nil {
		return nil
	}

	switch expr := expr.(type) {
	case *ast.ArrayLiteral:
		out := &nodeArrayLiteral{
			value: make([]nodeExpression, len(expr.Value)),
		}
		for i, value := range expr.Value {
			out.value[i] = cmpl.parseExpression(value)
		}
		return out

	case *ast.AssignExpression:
		return &nodeAssignExpression{
			operator: expr.Operator,
			left:     cmpl.parseExpression(expr.Left),
			right:    cmpl.parseExpression(expr.Right),
		}

	case *ast.BinaryExpression:
		return &nodeBinaryExpression{
			operator:   expr.Operator,
			left:       cmpl.parseExpression(expr.Left),
			right:      cmpl.parseExpression(expr.Right),
			comparison: expr.Comparison,
		}

	case *ast.BooleanLiteral:
		if expr.Value {
			return trueLiteral
		}
		return falseLiteral

	case *ast.BracketExpression:
		return &nodeBracketExpression{
			idx:    expr.Left.Idx0(),
			left:   cmpl.parseExpression(expr.Left),
			member: cmpl.parseExpression(expr.Member),
		}

	case *ast.CallExpression:
		out := &nodeCallExpression{
			callee:       cmpl.parseExpression(expr.Callee),
			argumentList: make([]nodeExpression, len(expr.ArgumentList)),
		}
		for i, value := range expr.ArgumentList {
			out.argumentList[i] = cmpl.parseExpression(value)
		}
		return out

	case *ast.ConditionalExpression:
		return &nodeConditionalExpression{
			test:       cmpl.parseExpression(expr.Test),
			consequent: cmpl.parseExpression(expr.Consequent),
			alternate:  cmpl.parseExpression(expr.Alternate),
		}

	case *ast.DotExpression:
		return &nodeDotExpression{
			idx:        expr.Left.Idx0(),
			left:       cmpl.parseExpression(expr.Left),
			identifier: expr.Identifier.Name,
		}

	case *ast.EmptyExpression:
		return nil

	case *ast.FunctionLiteral:
		name := ""
		if expr.Name != nil {
			name = expr.Name.Name
		}
		out := &nodeFunctionLiteral{
			name:   name,
			body:   cmpl.parseStatement(expr.Body),
			source: expr.Source,
			file:   cmpl.file,
		}
		if expr.ParameterList != nil {
			list := expr.ParameterList.List
			out.parameterList = make([]string, len(list))
			for i, value := range list {
				out.parameterList[i] = value.Name
			}
		}
		for _, value := range expr.DeclarationList {
			switch value := value.(type) {
			case *ast.FunctionDeclaration:
				out.functionList = append(out.functionList, cmpl.parseExpression(value.Function).(*nodeFunctionLiteral))
			case *ast.VariableDeclaration:
				for _, value := range value.List {
					out.varList = append(out.varList, value.Name)
				}
			default:
				panic(fmt.Sprintf("parse expression unknown function declaration type %T", value))
			}
		}
		return out

	case *ast.Identifier:
		return &nodeIdentifier{
			idx:  expr.Idx,
			name: expr.Name,
		}

	case *ast.NewExpression:
		out := &nodeNewExpression{
			callee:       cmpl.parseExpression(expr.Callee),
			argumentList: make([]nodeExpression, len(expr.ArgumentList)),
		}
		for i, value := range expr.ArgumentList {
			out.argumentList[i] = cmpl.parseExpression(value)
		}
		return out

	case *ast.NullLiteral:
		return nullLiteral

	case *ast.NumberLiteral:
		return &nodeLiteral{
			value: toValue(expr.Value),
		}

	case *ast.ObjectLiteral:
		out := &nodeObjectLiteral{
			value: make([]nodeProperty, len(expr.Value)),
		}
		for i, value := range expr.Value {
			out.value[i] = nodeProperty{
				key:   value.Key,
				kind:  value.Kind,
				value: cmpl.parseExpression(value.Value),
			}
		}
		return out

	case *ast.RegExpLiteral:
		return &nodeRegExpLiteral{
			flags:   expr.Flags,
			pattern: expr.Pattern,
		}

	case *ast.SequenceExpression:
		out := &nodeSequenceExpression{
			sequence: make([]nodeExpression, len(expr.Sequence)),
		}
		for i, value := range expr.Sequence {
			out.sequence[i] = cmpl.parseExpression(value)
		}
		return out

	case *ast.StringLiteral:
		return &nodeLiteral{
			value: stringValue(expr.Value),
		}

	case *ast.ThisExpression:
		return &nodeThisExpression{}

	case *ast.UnaryExpression:
		return &nodeUnaryExpression{
			operator: expr.Operator,
			operand:  cmpl.parseExpression(expr.Operand),
			postfix:  expr.Postfix,
		}

	case *ast.VariableExpression:
		return &nodeVariableExpression{
			idx:         expr.Idx0(),
			name:        expr.Name,
			initializer: cmpl.parseExpression(expr.Initializer),
		}
	default:
		panic(fmt.Errorf("parse expression unknown node type %T", expr))
	}
}

func (cmpl *compiler) parseStatement(stmt ast.Statement) nodeStatement {
	if stmt == nil {
		return nil
	}

	switch stmt := stmt.(type) {
	case *ast.BlockStatement:
		out := &nodeBlockStatement{
			list: make([]nodeStatement, len(stmt.List)),
		}
		for i, value := range stmt.List {
			out.list[i] = cmpl.parseStatement(value)
		}
		return out

	case *ast.BranchStatement:
		out := &nodeBranchStatement{
			branch: stmt.Token,
		}
		if stmt.Label != nil {
			out.label = stmt.Label.Name
		}
		return out

	case *ast.DebuggerStatement:
		return &nodeDebuggerStatement{}

	case *ast.DoWhileStatement:
		out := &nodeDoWhileStatement{
			test: cmpl.parseExpression(stmt.Test),
		}
		body := cmpl.parseStatement(stmt.Body)
		if block, ok := body.(*nodeBlockStatement); ok {
			out.body = block.list
		} else {
			out.body = append(out.body, body)
		}
		return out

	case *ast.EmptyStatement:
		return emptyStatement

	case *ast.ExpressionStatement:
		return &nodeExpressionStatement{
			expression: cmpl.parseExpression(stmt.Expression),
		}

	case *ast.ForInStatement:
		out := &nodeForInStatement{
			into:   cmpl.parseExpression(stmt.Into),
			source: cmpl.parseExpression(stmt.Source),
		}
		body := cmpl.parseStatement(stmt.Body)
		if block, ok := body.(*nodeBlockStatement); ok {
			out.body = block.list
		} else {
			out.body = append(out.body, body)
		}
		return out

	case *ast.ForStatement:
		out := &nodeForStatement{
			initializer: cmpl.parseExpression(stmt.Initializer),
			update:      cmpl.parseExpression(stmt.Update),
			test:        cmpl.parseExpression(stmt.Test),
		}
		body := cmpl.parseStatement(stmt.Body)
		if block, ok := body.(*nodeBlockStatement); ok {
			out.body = block.list
		} else {
			out.body = append(out.body, body)
		}
		return out

	case *ast.FunctionStatement:
		return emptyStatement

	case *ast.IfStatement:
		return &nodeIfStatement{
			test:       cmpl.parseExpression(stmt.Test),
			consequent: cmpl.parseStatement(stmt.Consequent),
			alternate:  cmpl.parseStatement(stmt.Alternate),
		}

	case *ast.LabelledStatement:
		return &nodeLabelledStatement{
			label:     stmt.Label.Name,
			statement: cmpl.parseStatement(stmt.Statement),
		}

	case *ast.ReturnStatement:
		return &nodeReturnStatement{
			argument: cmpl.parseExpression(stmt.Argument),
		}

	case *ast.SwitchStatement:
		out := &nodeSwitchStatement{
			discriminant: cmpl.parseExpression(stmt.Discriminant),
			defaultIdx:   stmt.Default,
			body:         make([]*nodeCaseStatement, len(stmt.Body)),
		}
		for i, clause := range stmt.Body {
			out.body[i] = &nodeCaseStatement{
				test:       cmpl.parseExpression(clause.Test),
				consequent: make([]nodeStatement, len(clause.Consequent)),
			}
			for j, value := range clause.Consequent {
				out.body[i].consequent[j] = cmpl.parseStatement(value)
			}
		}
		return out

	case *ast.ThrowStatement:
		return &nodeThrowStatement{
			argument: cmpl.parseExpression(stmt.Argument),
		}

	case *ast.TryStatement:
		out := &nodeTryStatement{
			body:    cmpl.parseStatement(stmt.Body),
			finally: cmpl.parseStatement(stmt.Finally),
		}
		if stmt.Catch != nil {
			out.catch = &nodeCatchStatement{
				parameter: stmt.Catch.Parameter.Name,
				body:      cmpl.parseStatement(stmt.Catch.Body),
			}
		}
		return out

	case *ast.VariableStatement:
		out := &nodeVariableStatement{
			list: make([]nodeExpression, len(stmt.List)),
		}
		for i, value := range stmt.List {
			out.list[i] = cmpl.parseExpression(value)
		}
		return out

	case *ast.WhileStatement:
		out := &nodeWhileStatement{
			test: cmpl.parseExpression(stmt.Test),
		}
		body := cmpl.parseStatement(stmt.Body)
		if block, ok := body.(*nodeBlockStatement); ok {
			out.body = block.list
		} else {
			out.body = append(out.body, body)
		}
		return out

	case *ast.WithStatement:
		return &nodeWithStatement{
			object: cmpl.parseExpression(stmt.Object),
			body:   cmpl.parseStatement(stmt.Body),
		}
	default:
		panic(fmt.Sprintf("parse statement: unknown type %T", stmt))
	}
}

func cmplParse(in *ast.Program) *nodeProgram {
	cmpl := compiler{
		program: in,
	}
	if cmpl.program != nil {
		cmpl.file = cmpl.program.File
	}

	return cmpl.parse()
}

func (cmpl *compiler) parse() *nodeProgram {
	out := &nodeProgram{
		body: make([]nodeStatement, len(cmpl.program.Body)),
		file: cmpl.program.File,
	}
	for i, value := range cmpl.program.Body {
		out.body[i] = cmpl.parseStatement(value)
	}
	for _, value := range cmpl.program.DeclarationList {
		switch value := value.(type) {
		case *ast.FunctionDeclaration:
			out.functionList = append(out.functionList, cmpl.parseExpression(value.Function).(*nodeFunctionLiteral))
		case *ast.VariableDeclaration:
			for _, value := range value.List {
				out.varList = append(out.varList, value.Name)
			}
		default:
			panic(fmt.Sprintf("Here be dragons: cmpl.parseProgram.DeclarationList(%T)", value))
		}
	}
	return out
}

type nodeProgram struct {
	file         *file.File
	body         []nodeStatement
	varList      []string
	functionList []*nodeFunctionLiteral
}

type node interface{}

type (
	nodeExpression interface {
		node
		expressionNode()
	}

	nodeArrayLiteral struct {
		value []nodeExpression
	}

	nodeAssignExpression struct {
		left     nodeExpression
		right    nodeExpression
		operator token.Token
	}

	nodeBinaryExpression struct {
		left       nodeExpression
		right      nodeExpression
		operator   token.Token
		comparison bool
	}

	nodeBracketExpression struct {
		left   nodeExpression
		member nodeExpression
		idx    file.Idx
	}

	nodeCallExpression struct {
		callee       nodeExpression
		argumentList []nodeExpression
	}

	nodeConditionalExpression struct {
		test       nodeExpression
		consequent nodeExpression
		alternate  nodeExpression
	}

	nodeDotExpression struct {
		left       nodeExpression
		identifier string
		idx        file.Idx
	}

	nodeFunctionLiteral struct {
		body          nodeStatement
		file          *file.File
		name          string
		source        string
		parameterList []string
		varList       []string
		functionList  []*nodeFunctionLiteral
	}

	nodeIdentifier struct {
		name string
		idx  file.Idx
	}

	nodeLiteral struct {
		value Value
	}

	nodeNewExpression struct {
		callee       nodeExpression
		argumentList []nodeExpression
	}

	nodeObjectLiteral struct {
		value []nodeProperty
	}

	nodeProperty struct {
		value nodeExpression
		key   string
		kind  string
	}

	nodeRegExpLiteral struct {
		flags   string
		pattern string // Value?
	}

	nodeSequenceExpression struct {
		sequence []nodeExpression
	}

	nodeThisExpression struct{}

	nodeUnaryExpression struct {
		operand  nodeExpression
		operator token.Token
		postfix  bool
	}

	nodeVariableExpression struct {
		initializer nodeExpression
		name        string
		idx         file.Idx
	}
)

type (
	nodeStatement interface {
		node
		statementNode()
	}

	nodeBlockStatement struct {
		list []nodeStatement
	}

	nodeBranchStatement struct {
		label  string
		branch token.Token
	}

	nodeCaseStatement struct {
		test       nodeExpression
		consequent []nodeStatement
	}

	nodeCatchStatement struct {
		body      nodeStatement
		parameter string
	}

	nodeDebuggerStatement struct{}

	nodeDoWhileStatement struct {
		test nodeExpression
		body []nodeStatement
	}

	nodeEmptyStatement struct{}

	nodeExpressionStatement struct {
		expression nodeExpression
	}

	nodeForInStatement struct {
		into   nodeExpression
		source nodeExpression
		body   []nodeStatement
	}

	nodeForStatement struct {
		initializer nodeExpression
		update      nodeExpression
		test        nodeExpression
		body        []nodeStatement
	}

	nodeIfStatement struct {
		test       nodeExpression
		consequent nodeStatement
		alternate  nodeStatement
	}

	nodeLabelledStatement struct {
		statement nodeStatement
		label     string
	}

	nodeReturnStatement struct {
		argument nodeExpression
	}

	nodeSwitchStatement struct {
		discriminant nodeExpression
		body         []*nodeCaseStatement
		defaultIdx   int
	}

	nodeThrowStatement struct {
		argument nodeExpression
	}

	nodeTryStatement struct {
		body    nodeStatement
		catch   *nodeCatchStatement
		finally nodeStatement
	}

	nodeVariableStatement struct {
		list []nodeExpression
	}

	nodeWhileStatement struct {
		test nodeExpression
		body []nodeStatement
	}

	nodeWithStatement struct {
		object nodeExpression
		body   nodeStatement
	}
)

// expressionNode.
func (*nodeArrayLiteral) expressionNode()          {}
func (*nodeAssignExpression) expressionNode()      {}
func (*nodeBinaryExpression) expressionNode()      {}
func (*nodeBracketExpression) expressionNode()     {}
func (*nodeCallExpression) expressionNode()        {}
func (*nodeConditionalExpression) expressionNode() {}
func (*nodeDotExpression) expressionNode()         {}
func (*nodeFunctionLiteral) expressionNode()       {}
func (*nodeIdentifier) expressionNode()            {}
func (*nodeLiteral) expressionNode()               {}
func (*nodeNewExpression) expressionNode()         {}
func (*nodeObjectLiteral) expressionNode()         {}
func (*nodeRegExpLiteral) expressionNode()         {}
func (*nodeSequenceExpression) expressionNode()    {}
func (*nodeThisExpression) expressionNode()        {}
func (*nodeUnaryExpression) expressionNode()       {}
func (*nodeVariableExpression) expressionNode()    {}

// statementNode

func (*nodeBlockStatement) statementNode()      {}
func (*nodeBranchStatement) statementNode()     {}
func (*nodeCaseStatement) statementNode()       {}
func (*nodeCatchStatement) statementNode()      {}
func (*nodeDebuggerStatement) statementNode()   {}
func (*nodeDoWhileStatement) statementNode()    {}
func (*nodeEmptyStatement) statementNode()      {}
func (*nodeExpressionStatement) statementNode() {}
func (*nodeForInStatement) statementNode()      {}
func (*nodeForStatement) statementNode()        {}
func (*nodeIfStatement) statementNode()         {}
func (*nodeLabelledStatement) statementNode()   {}
func (*nodeReturnStatement) statementNode()     {}
func (*nodeSwitchStatement) statementNode()     {}
func (*nodeThrowStatement) statementNode()      {}
func (*nodeTryStatement) statementNode()        {}
func (*nodeVariableStatement) statementNode()   {}
func (*nodeWhileStatement) statementNode()      {}
func (*nodeWithStatement) statementNode()       {}
