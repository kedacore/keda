package parser

import (
	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/token"
)

func (p *parser) parseBlockStatement() *ast.BlockStatement {
	node := &ast.BlockStatement{}

	// Find comments before the leading brace
	if p.mode&StoreComments != 0 {
		p.comments.CommentMap.AddComments(node, p.comments.FetchAll(), ast.LEADING)
		p.comments.Unset()
	}

	node.LeftBrace = p.expect(token.LEFT_BRACE)
	node.List = p.parseStatementList()

	if p.mode&StoreComments != 0 {
		p.comments.Unset()
		p.comments.CommentMap.AddComments(node, p.comments.FetchAll(), ast.FINAL)
		p.comments.AfterBlock()
	}

	node.RightBrace = p.expect(token.RIGHT_BRACE)

	// Find comments after the trailing brace
	if p.mode&StoreComments != 0 {
		p.comments.ResetLineBreak()
		p.comments.CommentMap.AddComments(node, p.comments.Fetch(), ast.TRAILING)
	}

	return node
}

func (p *parser) parseEmptyStatement() ast.Statement {
	idx := p.expect(token.SEMICOLON)
	return &ast.EmptyStatement{Semicolon: idx}
}

func (p *parser) parseStatementList() (list []ast.Statement) { //nolint:nonamedreturns
	for p.token != token.RIGHT_BRACE && p.token != token.EOF {
		statement := p.parseStatement()
		list = append(list, statement)
	}

	return list
}

func (p *parser) parseStatement() ast.Statement {
	if p.token == token.EOF {
		p.errorUnexpectedToken(p.token)
		return &ast.BadStatement{From: p.idx, To: p.idx + 1}
	}

	if p.mode&StoreComments != 0 {
		p.comments.ResetLineBreak()
	}

	switch p.token {
	case token.SEMICOLON:
		return p.parseEmptyStatement()
	case token.LEFT_BRACE:
		return p.parseBlockStatement()
	case token.IF:
		return p.parseIfStatement()
	case token.DO:
		statement := p.parseDoWhileStatement()
		p.comments.PostProcessNode(statement)
		return statement
	case token.WHILE:
		return p.parseWhileStatement()
	case token.FOR:
		return p.parseForOrForInStatement()
	case token.BREAK:
		return p.parseBreakStatement()
	case token.CONTINUE:
		return p.parseContinueStatement()
	case token.DEBUGGER:
		return p.parseDebuggerStatement()
	case token.WITH:
		return p.parseWithStatement()
	case token.VAR:
		return p.parseVariableStatement()
	case token.FUNCTION:
		return p.parseFunctionStatement()
	case token.SWITCH:
		return p.parseSwitchStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.THROW:
		return p.parseThrowStatement()
	case token.TRY:
		return p.parseTryStatement()
	}

	var comments []*ast.Comment
	if p.mode&StoreComments != 0 {
		comments = p.comments.FetchAll()
	}

	expression := p.parseExpression()

	if identifier, isIdentifier := expression.(*ast.Identifier); isIdentifier && p.token == token.COLON {
		// LabelledStatement
		colon := p.idx
		if p.mode&StoreComments != 0 {
			p.comments.Unset()
		}
		p.next() // :

		label := identifier.Name
		for _, value := range p.scope.labels {
			if label == value {
				p.error(identifier.Idx0(), "Label '%s' already exists", label)
			}
		}
		var labelComments []*ast.Comment
		if p.mode&StoreComments != 0 {
			labelComments = p.comments.FetchAll()
		}
		p.scope.labels = append(p.scope.labels, label) // Push the label
		statement := p.parseStatement()
		p.scope.labels = p.scope.labels[:len(p.scope.labels)-1] // Pop the label
		exp := &ast.LabelledStatement{
			Label:     identifier,
			Colon:     colon,
			Statement: statement,
		}
		if p.mode&StoreComments != 0 {
			p.comments.CommentMap.AddComments(exp, labelComments, ast.LEADING)
		}

		return exp
	}

	p.optionalSemicolon()

	statement := &ast.ExpressionStatement{
		Expression: expression,
	}

	if p.mode&StoreComments != 0 {
		p.comments.CommentMap.AddComments(statement, comments, ast.LEADING)
	}
	return statement
}

func (p *parser) parseTryStatement() ast.Statement {
	var tryComments []*ast.Comment
	if p.mode&StoreComments != 0 {
		tryComments = p.comments.FetchAll()
	}
	node := &ast.TryStatement{
		Try:  p.expect(token.TRY),
		Body: p.parseBlockStatement(),
	}
	if p.mode&StoreComments != 0 {
		p.comments.CommentMap.AddComments(node, tryComments, ast.LEADING)
		p.comments.CommentMap.AddComments(node.Body, p.comments.FetchAll(), ast.TRAILING)
	}

	if p.token == token.CATCH {
		catch := p.idx
		if p.mode&StoreComments != 0 {
			p.comments.Unset()
		}
		p.next()
		p.expect(token.LEFT_PARENTHESIS)
		if p.token != token.IDENTIFIER {
			p.expect(token.IDENTIFIER)
			p.nextStatement()
			return &ast.BadStatement{From: catch, To: p.idx}
		}

		identifier := p.parseIdentifier()
		p.expect(token.RIGHT_PARENTHESIS)
		node.Catch = &ast.CatchStatement{
			Catch:     catch,
			Parameter: identifier,
			Body:      p.parseBlockStatement(),
		}

		if p.mode&StoreComments != 0 {
			p.comments.CommentMap.AddComments(node.Catch.Body, p.comments.FetchAll(), ast.TRAILING)
		}
	}

	if p.token == token.FINALLY {
		if p.mode&StoreComments != 0 {
			p.comments.Unset()
		}
		p.next()
		if p.mode&StoreComments != 0 {
			tryComments = p.comments.FetchAll()
		}

		node.Finally = p.parseBlockStatement()

		if p.mode&StoreComments != 0 {
			p.comments.CommentMap.AddComments(node.Finally, tryComments, ast.LEADING)
		}
	}

	if node.Catch == nil && node.Finally == nil {
		p.error(node.Try, "Missing catch or finally after try")
		return &ast.BadStatement{From: node.Try, To: node.Body.Idx1()}
	}

	return node
}

func (p *parser) parseFunctionParameterList() *ast.ParameterList {
	opening := p.expect(token.LEFT_PARENTHESIS)
	if p.mode&StoreComments != 0 {
		p.comments.Unset()
	}
	var list []*ast.Identifier
	for p.token != token.RIGHT_PARENTHESIS && p.token != token.EOF {
		if p.token != token.IDENTIFIER {
			p.expect(token.IDENTIFIER)
		} else {
			identifier := p.parseIdentifier()
			list = append(list, identifier)
		}
		if p.token != token.RIGHT_PARENTHESIS {
			if p.mode&StoreComments != 0 {
				p.comments.Unset()
			}
			p.expect(token.COMMA)
		}
	}
	closing := p.expect(token.RIGHT_PARENTHESIS)

	return &ast.ParameterList{
		Opening: opening,
		List:    list,
		Closing: closing,
	}
}

func (p *parser) parseFunctionStatement() *ast.FunctionStatement {
	var comments []*ast.Comment
	if p.mode&StoreComments != 0 {
		comments = p.comments.FetchAll()
	}
	function := &ast.FunctionStatement{
		Function: p.parseFunction(true),
	}
	if p.mode&StoreComments != 0 {
		p.comments.CommentMap.AddComments(function, comments, ast.LEADING)
	}

	return function
}

func (p *parser) parseFunction(declaration bool) *ast.FunctionLiteral {
	node := &ast.FunctionLiteral{
		Function: p.expect(token.FUNCTION),
	}

	var name *ast.Identifier
	if p.token == token.IDENTIFIER {
		name = p.parseIdentifier()
		if declaration {
			p.scope.declare(&ast.FunctionDeclaration{
				Function: node,
			})
		}
	} else if declaration {
		// Use expect error handling
		p.expect(token.IDENTIFIER)
	}
	if p.mode&StoreComments != 0 {
		p.comments.Unset()
	}
	node.Name = name
	node.ParameterList = p.parseFunctionParameterList()
	p.parseFunctionBlock(node)
	node.Source = p.slice(node.Idx0(), node.Idx1())

	return node
}

func (p *parser) parseFunctionBlock(node *ast.FunctionLiteral) {
	p.openScope()
	inFunction := p.scope.inFunction
	p.scope.inFunction = true
	defer func() {
		p.scope.inFunction = inFunction
		p.closeScope()
	}()
	node.Body = p.parseBlockStatement()
	node.DeclarationList = p.scope.declarationList
}

func (p *parser) parseDebuggerStatement() ast.Statement {
	idx := p.expect(token.DEBUGGER)

	node := &ast.DebuggerStatement{
		Debugger: idx,
	}
	if p.mode&StoreComments != 0 {
		p.comments.CommentMap.AddComments(node, p.comments.FetchAll(), ast.TRAILING)
	}

	p.semicolon()
	return node
}

func (p *parser) parseReturnStatement() ast.Statement {
	idx := p.expect(token.RETURN)
	var comments []*ast.Comment
	if p.mode&StoreComments != 0 {
		comments = p.comments.FetchAll()
	}

	if !p.scope.inFunction {
		p.error(idx, "Illegal return statement")
		p.nextStatement()
		return &ast.BadStatement{From: idx, To: p.idx}
	}

	node := &ast.ReturnStatement{
		Return: idx,
	}

	if !p.implicitSemicolon && p.token != token.SEMICOLON && p.token != token.RIGHT_BRACE && p.token != token.EOF {
		node.Argument = p.parseExpression()
	}
	if p.mode&StoreComments != 0 {
		p.comments.CommentMap.AddComments(node, comments, ast.LEADING)
	}

	p.semicolon()

	return node
}

func (p *parser) parseThrowStatement() ast.Statement {
	var comments []*ast.Comment
	if p.mode&StoreComments != 0 {
		comments = p.comments.FetchAll()
	}
	idx := p.expect(token.THROW)

	if p.implicitSemicolon {
		if p.chr == -1 { // Hackish
			p.error(idx, "Unexpected end of input")
		} else {
			p.error(idx, "Illegal newline after throw")
		}
		p.nextStatement()
		return &ast.BadStatement{From: idx, To: p.idx}
	}

	node := &ast.ThrowStatement{
		Throw:    idx,
		Argument: p.parseExpression(),
	}
	if p.mode&StoreComments != 0 {
		p.comments.CommentMap.AddComments(node, comments, ast.LEADING)
	}

	p.semicolon()

	return node
}

func (p *parser) parseSwitchStatement() ast.Statement {
	var comments []*ast.Comment
	if p.mode&StoreComments != 0 {
		comments = p.comments.FetchAll()
	}
	idx := p.expect(token.SWITCH)
	if p.mode&StoreComments != 0 {
		comments = append(comments, p.comments.FetchAll()...)
	}
	p.expect(token.LEFT_PARENTHESIS)
	node := &ast.SwitchStatement{
		Switch:       idx,
		Discriminant: p.parseExpression(),
		Default:      -1,
	}
	p.expect(token.RIGHT_PARENTHESIS)
	if p.mode&StoreComments != 0 {
		comments = append(comments, p.comments.FetchAll()...)
	}

	p.expect(token.LEFT_BRACE)

	inSwitch := p.scope.inSwitch
	p.scope.inSwitch = true
	defer func() {
		p.scope.inSwitch = inSwitch
	}()

	for index := 0; p.token != token.EOF; index++ {
		if p.token == token.RIGHT_BRACE {
			node.RightBrace = p.idx
			p.next()
			break
		}

		clause := p.parseCaseStatement()
		if clause.Test == nil {
			if node.Default != -1 {
				p.error(clause.Case, "Already saw a default in switch")
			}
			node.Default = index
		}
		node.Body = append(node.Body, clause)
	}

	if p.mode&StoreComments != 0 {
		p.comments.CommentMap.AddComments(node, comments, ast.LEADING)
	}

	return node
}

func (p *parser) parseWithStatement() ast.Statement {
	var comments []*ast.Comment
	if p.mode&StoreComments != 0 {
		comments = p.comments.FetchAll()
	}
	idx := p.expect(token.WITH)
	var withComments []*ast.Comment
	if p.mode&StoreComments != 0 {
		withComments = p.comments.FetchAll()
	}

	p.expect(token.LEFT_PARENTHESIS)

	node := &ast.WithStatement{
		With:   idx,
		Object: p.parseExpression(),
	}
	p.expect(token.RIGHT_PARENTHESIS)

	if p.mode&StoreComments != 0 {
		p.comments.CommentMap.AddComments(node, comments, ast.LEADING)
		p.comments.CommentMap.AddComments(node, withComments, ast.WITH)
	}

	node.Body = p.parseStatement()

	return node
}

func (p *parser) parseCaseStatement() *ast.CaseStatement {
	node := &ast.CaseStatement{
		Case: p.idx,
	}

	var comments []*ast.Comment
	if p.mode&StoreComments != 0 {
		comments = p.comments.FetchAll()
		p.comments.Unset()
	}

	if p.token == token.DEFAULT {
		p.next()
	} else {
		p.expect(token.CASE)
		node.Test = p.parseExpression()
	}

	if p.mode&StoreComments != 0 {
		p.comments.Unset()
	}
	p.expect(token.COLON)

	for {
		if p.token == token.EOF ||
			p.token == token.RIGHT_BRACE ||
			p.token == token.CASE ||
			p.token == token.DEFAULT {
			break
		}
		consequent := p.parseStatement()
		node.Consequent = append(node.Consequent, consequent)
	}

	// Link the comments to the case statement
	if p.mode&StoreComments != 0 {
		p.comments.CommentMap.AddComments(node, comments, ast.LEADING)
	}

	return node
}

func (p *parser) parseIterationStatement() ast.Statement {
	inIteration := p.scope.inIteration
	p.scope.inIteration = true
	defer func() {
		p.scope.inIteration = inIteration
	}()
	return p.parseStatement()
}

func (p *parser) parseForIn(into ast.Expression) *ast.ForInStatement {
	// Already have consumed "<into> in"

	source := p.parseExpression()
	p.expect(token.RIGHT_PARENTHESIS)
	body := p.parseIterationStatement()

	forin := &ast.ForInStatement{
		Into:   into,
		Source: source,
		Body:   body,
	}

	return forin
}

func (p *parser) parseFor(initializer ast.Expression) *ast.ForStatement {
	// Already have consumed "<initializer> ;"

	var test, update ast.Expression

	if p.token != token.SEMICOLON {
		test = p.parseExpression()
	}
	if p.mode&StoreComments != 0 {
		p.comments.Unset()
	}
	p.expect(token.SEMICOLON)

	if p.token != token.RIGHT_PARENTHESIS {
		update = p.parseExpression()
	}
	p.expect(token.RIGHT_PARENTHESIS)
	body := p.parseIterationStatement()

	forstatement := &ast.ForStatement{
		Initializer: initializer,
		Test:        test,
		Update:      update,
		Body:        body,
	}

	return forstatement
}

func (p *parser) parseForOrForInStatement() ast.Statement {
	var comments []*ast.Comment
	if p.mode&StoreComments != 0 {
		comments = p.comments.FetchAll()
	}
	idx := p.expect(token.FOR)
	var forComments []*ast.Comment
	if p.mode&StoreComments != 0 {
		forComments = p.comments.FetchAll()
	}
	p.expect(token.LEFT_PARENTHESIS)

	var left []ast.Expression

	forIn := false
	if p.token != token.SEMICOLON {
		allowIn := p.scope.allowIn
		p.scope.allowIn = false
		if p.token == token.VAR {
			tokenIdx := p.idx
			var varComments []*ast.Comment
			if p.mode&StoreComments != 0 {
				varComments = p.comments.FetchAll()
				p.comments.Unset()
			}
			p.next()
			list := p.parseVariableDeclarationList(tokenIdx)
			if len(list) == 1 && p.token == token.IN {
				if p.mode&StoreComments != 0 {
					p.comments.Unset()
				}
				p.next() // in
				forIn = true
				left = []ast.Expression{list[0]} // There is only one declaration
			} else {
				left = list
			}
			if p.mode&StoreComments != 0 {
				p.comments.CommentMap.AddComments(left[0], varComments, ast.LEADING)
			}
		} else {
			left = append(left, p.parseExpression())
			if p.token == token.IN {
				p.next()
				forIn = true
			}
		}
		p.scope.allowIn = allowIn
	}

	if forIn {
		switch left[0].(type) {
		case *ast.Identifier, *ast.DotExpression, *ast.BracketExpression, *ast.VariableExpression:
			// These are all acceptable
		default:
			p.error(idx, "Invalid left-hand side in for-in")
			p.nextStatement()
			return &ast.BadStatement{From: idx, To: p.idx}
		}

		forin := p.parseForIn(left[0])
		forin.For = idx
		if p.mode&StoreComments != 0 {
			p.comments.CommentMap.AddComments(forin, comments, ast.LEADING)
			p.comments.CommentMap.AddComments(forin, forComments, ast.FOR)
		}
		return forin
	}

	if p.mode&StoreComments != 0 {
		p.comments.Unset()
	}
	p.expect(token.SEMICOLON)
	initializer := &ast.SequenceExpression{Sequence: left}
	forstatement := p.parseFor(initializer)
	forstatement.For = idx
	if p.mode&StoreComments != 0 {
		p.comments.CommentMap.AddComments(forstatement, comments, ast.LEADING)
		p.comments.CommentMap.AddComments(forstatement, forComments, ast.FOR)
	}
	return forstatement
}

func (p *parser) parseVariableStatement() *ast.VariableStatement {
	var comments []*ast.Comment
	if p.mode&StoreComments != 0 {
		comments = p.comments.FetchAll()
	}
	idx := p.expect(token.VAR)

	list := p.parseVariableDeclarationList(idx)

	statement := &ast.VariableStatement{
		Var:  idx,
		List: list,
	}
	if p.mode&StoreComments != 0 {
		p.comments.CommentMap.AddComments(statement, comments, ast.LEADING)
		p.comments.Unset()
	}
	p.semicolon()

	return statement
}

func (p *parser) parseDoWhileStatement() ast.Statement {
	inIteration := p.scope.inIteration
	p.scope.inIteration = true
	defer func() {
		p.scope.inIteration = inIteration
	}()

	var comments []*ast.Comment
	if p.mode&StoreComments != 0 {
		comments = p.comments.FetchAll()
	}
	idx := p.expect(token.DO)
	var doComments []*ast.Comment
	if p.mode&StoreComments != 0 {
		doComments = p.comments.FetchAll()
	}

	node := &ast.DoWhileStatement{Do: idx}
	if p.token == token.LEFT_BRACE {
		node.Body = p.parseBlockStatement()
	} else {
		node.Body = p.parseStatement()
	}

	p.expect(token.WHILE)
	var whileComments []*ast.Comment
	if p.mode&StoreComments != 0 {
		whileComments = p.comments.FetchAll()
	}
	p.expect(token.LEFT_PARENTHESIS)
	node.Test = p.parseExpression()
	node.RightParenthesis = p.expect(token.RIGHT_PARENTHESIS)

	p.implicitSemicolon = true
	p.optionalSemicolon()

	if p.mode&StoreComments != 0 {
		p.comments.CommentMap.AddComments(node, comments, ast.LEADING)
		p.comments.CommentMap.AddComments(node, doComments, ast.DO)
		p.comments.CommentMap.AddComments(node, whileComments, ast.WHILE)
	}

	return node
}

func (p *parser) parseWhileStatement() ast.Statement {
	var comments []*ast.Comment
	if p.mode&StoreComments != 0 {
		comments = p.comments.FetchAll()
	}
	idx := p.expect(token.WHILE)

	var whileComments []*ast.Comment
	if p.mode&StoreComments != 0 {
		whileComments = p.comments.FetchAll()
	}

	p.expect(token.LEFT_PARENTHESIS)
	node := &ast.WhileStatement{
		While: idx,
		Test:  p.parseExpression(),
	}
	p.expect(token.RIGHT_PARENTHESIS)
	node.Body = p.parseIterationStatement()

	if p.mode&StoreComments != 0 {
		p.comments.CommentMap.AddComments(node, comments, ast.LEADING)
		p.comments.CommentMap.AddComments(node, whileComments, ast.WHILE)
	}

	return node
}

func (p *parser) parseIfStatement() ast.Statement {
	var comments []*ast.Comment
	if p.mode&StoreComments != 0 {
		comments = p.comments.FetchAll()
	}
	pos := p.expect(token.IF)
	var ifComments []*ast.Comment
	if p.mode&StoreComments != 0 {
		ifComments = p.comments.FetchAll()
	}

	p.expect(token.LEFT_PARENTHESIS)
	node := &ast.IfStatement{
		If:   pos,
		Test: p.parseExpression(),
	}
	p.expect(token.RIGHT_PARENTHESIS)
	if p.token == token.LEFT_BRACE {
		node.Consequent = p.parseBlockStatement()
	} else {
		node.Consequent = p.parseStatement()
	}

	if p.token == token.ELSE {
		p.next()
		node.Alternate = p.parseStatement()
	}

	if p.mode&StoreComments != 0 {
		p.comments.CommentMap.AddComments(node, comments, ast.LEADING)
		p.comments.CommentMap.AddComments(node, ifComments, ast.IF)
	}

	return node
}

func (p *parser) parseSourceElement() ast.Statement {
	statement := p.parseStatement()
	return statement
}

func (p *parser) parseSourceElements() []ast.Statement {
	body := []ast.Statement(nil)

	for {
		if p.token != token.STRING {
			break
		}
		body = append(body, p.parseSourceElement())
	}

	for p.token != token.EOF {
		body = append(body, p.parseSourceElement())
	}

	return body
}

func (p *parser) parseProgram() *ast.Program {
	p.openScope()
	defer p.closeScope()
	return &ast.Program{
		Body:            p.parseSourceElements(),
		DeclarationList: p.scope.declarationList,
		File:            p.file,
	}
}

func (p *parser) parseBreakStatement() ast.Statement {
	var comments []*ast.Comment
	if p.mode&StoreComments != 0 {
		comments = p.comments.FetchAll()
	}
	idx := p.expect(token.BREAK)
	semicolon := p.implicitSemicolon
	if p.token == token.SEMICOLON {
		semicolon = true
		p.next()
	}

	if semicolon || p.token == token.RIGHT_BRACE {
		p.implicitSemicolon = false
		if !p.scope.inIteration && !p.scope.inSwitch {
			goto illegal
		}
		breakStatement := &ast.BranchStatement{
			Idx:   idx,
			Token: token.BREAK,
		}

		if p.mode&StoreComments != 0 {
			p.comments.CommentMap.AddComments(breakStatement, comments, ast.LEADING)
			p.comments.CommentMap.AddComments(breakStatement, p.comments.FetchAll(), ast.TRAILING)
		}

		return breakStatement
	}

	if p.token == token.IDENTIFIER {
		identifier := p.parseIdentifier()
		if !p.scope.hasLabel(identifier.Name) {
			p.error(idx, "Undefined label '%s'", identifier.Name)
			return &ast.BadStatement{From: idx, To: identifier.Idx1()}
		}
		p.semicolon()
		breakStatement := &ast.BranchStatement{
			Idx:   idx,
			Token: token.BREAK,
			Label: identifier,
		}
		if p.mode&StoreComments != 0 {
			p.comments.CommentMap.AddComments(breakStatement, comments, ast.LEADING)
		}

		return breakStatement
	}

	p.expect(token.IDENTIFIER)

illegal:
	p.error(idx, "Illegal break statement")
	p.nextStatement()
	return &ast.BadStatement{From: idx, To: p.idx}
}

func (p *parser) parseContinueStatement() ast.Statement {
	idx := p.expect(token.CONTINUE)
	semicolon := p.implicitSemicolon
	if p.token == token.SEMICOLON {
		semicolon = true
		p.next()
	}

	if semicolon || p.token == token.RIGHT_BRACE {
		p.implicitSemicolon = false
		if !p.scope.inIteration {
			goto illegal
		}
		return &ast.BranchStatement{
			Idx:   idx,
			Token: token.CONTINUE,
		}
	}

	if p.token == token.IDENTIFIER {
		identifier := p.parseIdentifier()
		if !p.scope.hasLabel(identifier.Name) {
			p.error(idx, "Undefined label '%s'", identifier.Name)
			return &ast.BadStatement{From: idx, To: identifier.Idx1()}
		}
		if !p.scope.inIteration {
			goto illegal
		}
		p.semicolon()
		return &ast.BranchStatement{
			Idx:   idx,
			Token: token.CONTINUE,
			Label: identifier,
		}
	}

	p.expect(token.IDENTIFIER)

illegal:
	p.error(idx, "Illegal continue statement")
	p.nextStatement()
	return &ast.BadStatement{From: idx, To: p.idx}
}

// Find the next statement after an error (recover).
func (p *parser) nextStatement() {
	for {
		switch p.token {
		case token.BREAK, token.CONTINUE,
			token.FOR, token.IF, token.RETURN, token.SWITCH,
			token.VAR, token.DO, token.TRY, token.WITH,
			token.WHILE, token.THROW, token.CATCH, token.FINALLY:
			// Return only if parser made some progress since last
			// sync or if it has not reached 10 next calls without
			// progress. Otherwise consume at least one token to
			// avoid an endless parser loop
			if p.idx == p.recover.idx && p.recover.count < 10 {
				p.recover.count++
				return
			}
			if p.idx > p.recover.idx {
				p.recover.idx = p.idx
				p.recover.count = 0
				return
			}
			// Reaching here indicates a parser bug, likely an
			// incorrect token list in this function, but it only
			// leads to skipping of possibly correct code if a
			// previous error is present, and thus is preferred
			// over a non-terminating parse.
		case token.EOF:
			return
		}
		p.next()
	}
}
