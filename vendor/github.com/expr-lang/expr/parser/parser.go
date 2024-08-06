package parser

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	. "github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/builtin"
	"github.com/expr-lang/expr/conf"
	"github.com/expr-lang/expr/file"
	. "github.com/expr-lang/expr/parser/lexer"
	"github.com/expr-lang/expr/parser/operator"
	"github.com/expr-lang/expr/parser/utils"
)

type arg byte

const (
	expr arg = 1 << iota
	closure
)

const optional arg = 1 << 7

var predicates = map[string]struct {
	args []arg
}{
	"all":           {[]arg{expr, closure}},
	"none":          {[]arg{expr, closure}},
	"any":           {[]arg{expr, closure}},
	"one":           {[]arg{expr, closure}},
	"filter":        {[]arg{expr, closure}},
	"map":           {[]arg{expr, closure}},
	"count":         {[]arg{expr, closure | optional}},
	"sum":           {[]arg{expr, closure | optional}},
	"find":          {[]arg{expr, closure}},
	"findIndex":     {[]arg{expr, closure}},
	"findLast":      {[]arg{expr, closure}},
	"findLastIndex": {[]arg{expr, closure}},
	"groupBy":       {[]arg{expr, closure}},
	"sortBy":        {[]arg{expr, closure, expr | optional}},
	"reduce":        {[]arg{expr, closure, expr | optional}},
}

type parser struct {
	tokens  []Token
	current Token
	pos     int
	err     *file.Error
	depth   int // closure call depth
	config  *conf.Config
}

type Tree struct {
	Node   Node
	Source file.Source
}

func Parse(input string) (*Tree, error) {
	return ParseWithConfig(input, &conf.Config{
		Disabled: map[string]bool{},
	})
}

func ParseWithConfig(input string, config *conf.Config) (*Tree, error) {
	source := file.NewSource(input)

	tokens, err := Lex(source)
	if err != nil {
		return nil, err
	}

	p := &parser{
		tokens:  tokens,
		current: tokens[0],
		config:  config,
	}

	node := p.parseExpression(0)

	if !p.current.Is(EOF) {
		p.error("unexpected token %v", p.current)
	}

	tree := &Tree{
		Node:   node,
		Source: source,
	}

	if p.err != nil {
		return tree, p.err.Bind(source)
	}

	return tree, nil
}

func (p *parser) error(format string, args ...any) {
	p.errorAt(p.current, format, args...)
}

func (p *parser) errorAt(token Token, format string, args ...any) {
	if p.err == nil { // show first error
		p.err = &file.Error{
			Location: token.Location,
			Message:  fmt.Sprintf(format, args...),
		}
	}
}

func (p *parser) next() {
	p.pos++
	if p.pos >= len(p.tokens) {
		p.error("unexpected end of expression")
		return
	}
	p.current = p.tokens[p.pos]
}

func (p *parser) expect(kind Kind, values ...string) {
	if p.current.Is(kind, values...) {
		p.next()
		return
	}
	p.error("unexpected token %v", p.current)
}

// parse functions

func (p *parser) parseExpression(precedence int) Node {
	if precedence == 0 && p.current.Is(Operator, "let") {
		return p.parseVariableDeclaration()
	}

	nodeLeft := p.parsePrimary()

	prevOperator := ""
	opToken := p.current
	for opToken.Is(Operator) && p.err == nil {
		negate := opToken.Is(Operator, "not")
		var notToken Token

		// Handle "not *" operator, like "not in" or "not contains".
		if negate {
			currentPos := p.pos
			p.next()
			if operator.AllowedNegateSuffix(p.current.Value) {
				if op, ok := operator.Binary[p.current.Value]; ok && op.Precedence >= precedence {
					notToken = p.current
					opToken = p.current
				} else {
					p.pos = currentPos
					p.current = opToken
					break
				}
			} else {
				p.error("unexpected token %v", p.current)
				break
			}
		}

		if op, ok := operator.Binary[opToken.Value]; ok && op.Precedence >= precedence {
			p.next()

			if opToken.Value == "|" {
				identToken := p.current
				p.expect(Identifier)
				nodeLeft = p.parseCall(identToken, []Node{nodeLeft}, true)
				goto next
			}

			if prevOperator == "??" && opToken.Value != "??" && !opToken.Is(Bracket, "(") {
				p.errorAt(opToken, "Operator (%v) and coalesce expressions (??) cannot be mixed. Wrap either by parentheses.", opToken.Value)
				break
			}

			if operator.IsComparison(opToken.Value) {
				nodeLeft = p.parseComparison(nodeLeft, opToken, op.Precedence)
				goto next
			}

			var nodeRight Node
			if op.Associativity == operator.Left {
				nodeRight = p.parseExpression(op.Precedence + 1)
			} else {
				nodeRight = p.parseExpression(op.Precedence)
			}

			nodeLeft = &BinaryNode{
				Operator: opToken.Value,
				Left:     nodeLeft,
				Right:    nodeRight,
			}
			nodeLeft.SetLocation(opToken.Location)

			if negate {
				nodeLeft = &UnaryNode{
					Operator: "not",
					Node:     nodeLeft,
				}
				nodeLeft.SetLocation(notToken.Location)
			}

			goto next
		}
		break

	next:
		prevOperator = opToken.Value
		opToken = p.current
	}

	if precedence == 0 {
		nodeLeft = p.parseConditional(nodeLeft)
	}

	return nodeLeft
}

func (p *parser) parseVariableDeclaration() Node {
	p.expect(Operator, "let")
	variableName := p.current
	p.expect(Identifier)
	p.expect(Operator, "=")
	value := p.parseExpression(0)
	p.expect(Operator, ";")
	node := p.parseExpression(0)
	let := &VariableDeclaratorNode{
		Name:  variableName.Value,
		Value: value,
		Expr:  node,
	}
	let.SetLocation(variableName.Location)
	return let
}

func (p *parser) parseConditional(node Node) Node {
	var expr1, expr2 Node
	for p.current.Is(Operator, "?") && p.err == nil {
		p.next()

		if !p.current.Is(Operator, ":") {
			expr1 = p.parseExpression(0)
			p.expect(Operator, ":")
			expr2 = p.parseExpression(0)
		} else {
			p.next()
			expr1 = node
			expr2 = p.parseExpression(0)
		}

		node = &ConditionalNode{
			Cond: node,
			Exp1: expr1,
			Exp2: expr2,
		}
	}
	return node
}

func (p *parser) parsePrimary() Node {
	token := p.current

	if token.Is(Operator) {
		if op, ok := operator.Unary[token.Value]; ok {
			p.next()
			expr := p.parseExpression(op.Precedence)
			node := &UnaryNode{
				Operator: token.Value,
				Node:     expr,
			}
			node.SetLocation(token.Location)
			return p.parsePostfixExpression(node)
		}
	}

	if token.Is(Bracket, "(") {
		p.next()
		expr := p.parseExpression(0)
		p.expect(Bracket, ")") // "an opened parenthesis is not properly closed"
		return p.parsePostfixExpression(expr)
	}

	if p.depth > 0 {
		if token.Is(Operator, "#") || token.Is(Operator, ".") {
			name := ""
			if token.Is(Operator, "#") {
				p.next()
				if p.current.Is(Identifier) {
					name = p.current.Value
					p.next()
				}
			}
			node := &PointerNode{Name: name}
			node.SetLocation(token.Location)
			return p.parsePostfixExpression(node)
		}
	} else {
		if token.Is(Operator, "#") || token.Is(Operator, ".") {
			p.error("cannot use pointer accessor outside closure")
		}
	}

	if token.Is(Operator, "::") {
		p.next()
		token = p.current
		p.expect(Identifier)
		return p.parsePostfixExpression(p.parseCall(token, []Node{}, false))
	}

	return p.parseSecondary()
}

func (p *parser) parseSecondary() Node {
	var node Node
	token := p.current

	switch token.Kind {

	case Identifier:
		p.next()
		switch token.Value {
		case "true":
			node := &BoolNode{Value: true}
			node.SetLocation(token.Location)
			return node
		case "false":
			node := &BoolNode{Value: false}
			node.SetLocation(token.Location)
			return node
		case "nil":
			node := &NilNode{}
			node.SetLocation(token.Location)
			return node
		default:
			if p.current.Is(Bracket, "(") {
				node = p.parseCall(token, []Node{}, true)
			} else {
				node = &IdentifierNode{Value: token.Value}
				node.SetLocation(token.Location)
			}
		}

	case Number:
		p.next()
		value := strings.Replace(token.Value, "_", "", -1)
		var node Node
		valueLower := strings.ToLower(value)
		switch {
		case strings.HasPrefix(valueLower, "0x"):
			number, err := strconv.ParseInt(value, 0, 64)
			if err != nil {
				p.error("invalid hex literal: %v", err)
			}
			node = p.toIntegerNode(number)
		case strings.ContainsAny(valueLower, ".e"):
			number, err := strconv.ParseFloat(value, 64)
			if err != nil {
				p.error("invalid float literal: %v", err)
			}
			node = p.toFloatNode(number)
		case strings.HasPrefix(valueLower, "0b"):
			number, err := strconv.ParseInt(value, 0, 64)
			if err != nil {
				p.error("invalid binary literal: %v", err)
			}
			node = p.toIntegerNode(number)
		case strings.HasPrefix(valueLower, "0o"):
			number, err := strconv.ParseInt(value, 0, 64)
			if err != nil {
				p.error("invalid octal literal: %v", err)
			}
			node = p.toIntegerNode(number)
		default:
			number, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				p.error("invalid integer literal: %v", err)
			}
			node = p.toIntegerNode(number)
		}
		if node != nil {
			node.SetLocation(token.Location)
		}
		return node
	case String:
		p.next()
		node = &StringNode{Value: token.Value}
		node.SetLocation(token.Location)

	default:
		if token.Is(Bracket, "[") {
			node = p.parseArrayExpression(token)
		} else if token.Is(Bracket, "{") {
			node = p.parseMapExpression(token)
		} else {
			p.error("unexpected token %v", token)
		}
	}

	return p.parsePostfixExpression(node)
}

func (p *parser) toIntegerNode(number int64) Node {
	if number > math.MaxInt {
		p.error("integer literal is too large")
		return nil
	}
	return &IntegerNode{Value: int(number)}
}

func (p *parser) toFloatNode(number float64) Node {
	if number > math.MaxFloat64 {
		p.error("float literal is too large")
		return nil
	}
	return &FloatNode{Value: number}
}

func (p *parser) parseCall(token Token, arguments []Node, checkOverrides bool) Node {
	var node Node

	isOverridden := p.config.IsOverridden(token.Value)
	isOverridden = isOverridden && checkOverrides

	if b, ok := predicates[token.Value]; ok && !isOverridden {
		p.expect(Bracket, "(")

		// In case of the pipe operator, the first argument is the left-hand side
		// of the operator, so we do not parse it as an argument inside brackets.
		args := b.args[len(arguments):]

		for i, arg := range args {
			if arg&optional == optional {
				if p.current.Is(Bracket, ")") {
					break
				}
			} else {
				if p.current.Is(Bracket, ")") {
					p.error("expected at least %d arguments", len(args))
				}
			}

			if i > 0 {
				p.expect(Operator, ",")
			}
			var node Node
			switch {
			case arg&expr == expr:
				node = p.parseExpression(0)
			case arg&closure == closure:
				node = p.parseClosure()
			}
			arguments = append(arguments, node)
		}

		p.expect(Bracket, ")")

		node = &BuiltinNode{
			Name:      token.Value,
			Arguments: arguments,
		}
		node.SetLocation(token.Location)
	} else if _, ok := builtin.Index[token.Value]; ok && !p.config.Disabled[token.Value] && !isOverridden {
		node = &BuiltinNode{
			Name:      token.Value,
			Arguments: p.parseArguments(arguments),
		}
		node.SetLocation(token.Location)
	} else {
		callee := &IdentifierNode{Value: token.Value}
		callee.SetLocation(token.Location)
		node = &CallNode{
			Callee:    callee,
			Arguments: p.parseArguments(arguments),
		}
		node.SetLocation(token.Location)
	}
	return node
}

func (p *parser) parseArguments(arguments []Node) []Node {
	// If pipe operator is used, the first argument is the left-hand side
	// of the operator, so we do not parse it as an argument inside brackets.
	offset := len(arguments)

	p.expect(Bracket, "(")
	for !p.current.Is(Bracket, ")") && p.err == nil {
		if len(arguments) > offset {
			p.expect(Operator, ",")
		}
		node := p.parseExpression(0)
		arguments = append(arguments, node)
	}
	p.expect(Bracket, ")")

	return arguments
}

func (p *parser) parseClosure() Node {
	startToken := p.current
	expectClosingBracket := false
	if p.current.Is(Bracket, "{") {
		p.next()
		expectClosingBracket = true
	}

	p.depth++
	node := p.parseExpression(0)
	p.depth--

	if expectClosingBracket {
		p.expect(Bracket, "}")
	}
	closure := &ClosureNode{
		Node: node,
	}
	closure.SetLocation(startToken.Location)
	return closure
}

func (p *parser) parseArrayExpression(token Token) Node {
	nodes := make([]Node, 0)

	p.expect(Bracket, "[")
	for !p.current.Is(Bracket, "]") && p.err == nil {
		if len(nodes) > 0 {
			p.expect(Operator, ",")
			if p.current.Is(Bracket, "]") {
				goto end
			}
		}
		node := p.parseExpression(0)
		nodes = append(nodes, node)
	}
end:
	p.expect(Bracket, "]")

	node := &ArrayNode{Nodes: nodes}
	node.SetLocation(token.Location)
	return node
}

func (p *parser) parseMapExpression(token Token) Node {
	p.expect(Bracket, "{")

	nodes := make([]Node, 0)
	for !p.current.Is(Bracket, "}") && p.err == nil {
		if len(nodes) > 0 {
			p.expect(Operator, ",")
			if p.current.Is(Bracket, "}") {
				goto end
			}
			if p.current.Is(Operator, ",") {
				p.error("unexpected token %v", p.current)
			}
		}

		var key Node
		// Map key can be one of:
		//  * number
		//  * string
		//  * identifier, which is equivalent to a string
		//  * expression, which must be enclosed in parentheses -- (1 + 2)
		if p.current.Is(Number) || p.current.Is(String) || p.current.Is(Identifier) {
			key = &StringNode{Value: p.current.Value}
			key.SetLocation(token.Location)
			p.next()
		} else if p.current.Is(Bracket, "(") {
			key = p.parseExpression(0)
		} else {
			p.error("a map key must be a quoted string, a number, a identifier, or an expression enclosed in parentheses (unexpected token %v)", p.current)
		}

		p.expect(Operator, ":")

		node := p.parseExpression(0)
		pair := &PairNode{Key: key, Value: node}
		pair.SetLocation(token.Location)
		nodes = append(nodes, pair)
	}

end:
	p.expect(Bracket, "}")

	node := &MapNode{Pairs: nodes}
	node.SetLocation(token.Location)
	return node
}

func (p *parser) parsePostfixExpression(node Node) Node {
	postfixToken := p.current
	for (postfixToken.Is(Operator) || postfixToken.Is(Bracket)) && p.err == nil {
		optional := postfixToken.Value == "?."
	parseToken:
		if postfixToken.Value == "." || postfixToken.Value == "?." {
			p.next()

			propertyToken := p.current
			if optional && propertyToken.Is(Bracket, "[") {
				postfixToken = propertyToken
				goto parseToken
			}
			p.next()

			if propertyToken.Kind != Identifier &&
				// Operators like "not" and "matches" are valid methods or property names.
				(propertyToken.Kind != Operator || !utils.IsValidIdentifier(propertyToken.Value)) {
				p.error("expected name")
			}

			property := &StringNode{Value: propertyToken.Value}
			property.SetLocation(propertyToken.Location)

			chainNode, isChain := node.(*ChainNode)
			optional := postfixToken.Value == "?."

			if isChain {
				node = chainNode.Node
			}

			memberNode := &MemberNode{
				Node:     node,
				Property: property,
				Optional: optional,
			}
			memberNode.SetLocation(propertyToken.Location)

			if p.current.Is(Bracket, "(") {
				memberNode.Method = true
				node = &CallNode{
					Callee:    memberNode,
					Arguments: p.parseArguments([]Node{}),
				}
				node.SetLocation(propertyToken.Location)
			} else {
				node = memberNode
			}

			if isChain || optional {
				node = &ChainNode{Node: node}
			}

		} else if postfixToken.Value == "[" {
			p.next()
			var from, to Node

			if p.current.Is(Operator, ":") { // slice without from [:1]
				p.next()

				if !p.current.Is(Bracket, "]") { // slice without from and to [:]
					to = p.parseExpression(0)
				}

				node = &SliceNode{
					Node: node,
					To:   to,
				}
				node.SetLocation(postfixToken.Location)
				p.expect(Bracket, "]")

			} else {

				from = p.parseExpression(0)

				if p.current.Is(Operator, ":") {
					p.next()

					if !p.current.Is(Bracket, "]") { // slice without to [1:]
						to = p.parseExpression(0)
					}

					node = &SliceNode{
						Node: node,
						From: from,
						To:   to,
					}
					node.SetLocation(postfixToken.Location)
					p.expect(Bracket, "]")

				} else {
					// Slice operator [:] was not found,
					// it should be just an index node.
					node = &MemberNode{
						Node:     node,
						Property: from,
						Optional: optional,
					}
					node.SetLocation(postfixToken.Location)
					if optional {
						node = &ChainNode{Node: node}
					}
					p.expect(Bracket, "]")
				}
			}
		} else {
			break
		}
		postfixToken = p.current
	}
	return node
}

func (p *parser) parseComparison(left Node, token Token, precedence int) Node {
	var rootNode Node
	for {
		comparator := p.parseExpression(precedence + 1)
		cmpNode := &BinaryNode{
			Operator: token.Value,
			Left:     left,
			Right:    comparator,
		}
		cmpNode.SetLocation(token.Location)
		if rootNode == nil {
			rootNode = cmpNode
		} else {
			rootNode = &BinaryNode{
				Operator: "&&",
				Left:     rootNode,
				Right:    cmpNode,
			}
			rootNode.SetLocation(token.Location)
		}

		left = comparator
		token = p.current
		if !(token.Is(Operator) && operator.IsComparison(token.Value) && p.err == nil) {
			break
		}
		p.next()
	}
	return rootNode
}
