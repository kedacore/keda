package parser

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	. "github.com/antonmedv/expr/ast"
	"github.com/antonmedv/expr/builtin"
	"github.com/antonmedv/expr/conf"
	"github.com/antonmedv/expr/file"
	. "github.com/antonmedv/expr/parser/lexer"
	"github.com/antonmedv/expr/parser/operator"
	"github.com/antonmedv/expr/parser/utils"
)

var predicates = map[string]struct {
	arity int
}{
	"all":           {2},
	"none":          {2},
	"any":           {2},
	"one":           {2},
	"filter":        {2},
	"map":           {2},
	"count":         {2},
	"find":          {2},
	"findIndex":     {2},
	"findLast":      {2},
	"findLastIndex": {2},
	"groupBy":       {2},
	"reduce":        {3},
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
	Source *file.Source
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

	if p.err != nil {
		return nil, p.err.Bind(source)
	}

	return &Tree{
		Node:   node,
		Source: source,
	}, nil
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
	if precedence == 0 {
		if p.current.Is(Operator, "let") {
			return p.parseVariableDeclaration()
		}
	}

	nodeLeft := p.parsePrimary()

	prevOperator := ""
	opToken := p.current
	for opToken.Is(Operator) && p.err == nil {
		negate := false
		var notToken Token

		// Handle "not *" operator, like "not in" or "not contains".
		if opToken.Is(Operator, "not") {
			p.next()
			notToken = p.current
			negate = true
			opToken = p.current
		}

		if op, ok := operator.Binary[opToken.Value]; ok {
			if op.Precedence >= precedence {
				p.next()

				if opToken.Value == "|" {
					nodeLeft = p.parsePipe(nodeLeft)
					goto next
				}

				if prevOperator == "??" && opToken.Value != "??" && !opToken.Is(Bracket, "(") {
					p.errorAt(opToken, "Operator (%v) and coalesce expressions (??) cannot be mixed. Wrap either by parentheses.", opToken.Value)
					break
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
			node = p.parseCall(token)
		}

	case Number:
		p.next()
		value := strings.Replace(token.Value, "_", "", -1)
		if strings.Contains(value, "x") {
			number, err := strconv.ParseInt(value, 0, 64)
			if err != nil {
				p.error("invalid hex literal: %v", err)
			}
			if number > math.MaxInt {
				p.error("integer literal is too large")
				return nil
			}
			node := &IntegerNode{Value: int(number)}
			node.SetLocation(token.Location)
			return node
		} else if strings.ContainsAny(value, ".eE") {
			number, err := strconv.ParseFloat(value, 64)
			if err != nil {
				p.error("invalid float literal: %v", err)
			}
			node := &FloatNode{Value: number}
			node.SetLocation(token.Location)
			return node
		} else {
			number, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				p.error("invalid integer literal: %v", err)
			}
			if number > math.MaxInt {
				p.error("integer literal is too large")
				return nil
			}
			node := &IntegerNode{Value: int(number)}
			node.SetLocation(token.Location)
			return node
		}

	case String:
		p.next()
		node := &StringNode{Value: token.Value}
		node.SetLocation(token.Location)
		return node

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

func (p *parser) parseCall(token Token) Node {
	var node Node
	if p.current.Is(Bracket, "(") {
		var arguments []Node

		if b, ok := predicates[token.Value]; ok {
			p.expect(Bracket, "(")

			// TODO: Refactor parser to use builtin.Builtins instead of predicates map.

			if b.arity == 1 {
				arguments = make([]Node, 1)
				arguments[0] = p.parseExpression(0)
			} else if b.arity == 2 {
				arguments = make([]Node, 2)
				arguments[0] = p.parseExpression(0)
				p.expect(Operator, ",")
				arguments[1] = p.parseClosure()
			}

			if token.Value == "reduce" {
				arguments = make([]Node, 2)
				arguments[0] = p.parseExpression(0)
				p.expect(Operator, ",")
				arguments[1] = p.parseClosure()
				if p.current.Is(Operator, ",") {
					p.next()
					arguments = append(arguments, p.parseExpression(0))
				}
			}

			p.expect(Bracket, ")")

			node = &BuiltinNode{
				Name:      token.Value,
				Arguments: arguments,
			}
			node.SetLocation(token.Location)
		} else if _, ok := builtin.Index[token.Value]; ok && !p.config.Disabled[token.Value] {
			node = &BuiltinNode{
				Name:      token.Value,
				Arguments: p.parseArguments(),
			}
			node.SetLocation(token.Location)
		} else {
			callee := &IdentifierNode{Value: token.Value}
			callee.SetLocation(token.Location)
			node = &CallNode{
				Callee:    callee,
				Arguments: p.parseArguments(),
			}
			node.SetLocation(token.Location)
		}
	} else {
		node = &IdentifierNode{Value: token.Value}
		node.SetLocation(token.Location)
	}
	return node
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
		if postfixToken.Value == "." || postfixToken.Value == "?." {
			p.next()

			propertyToken := p.current
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
				node = &CallNode{
					Callee:    memberNode,
					Arguments: p.parseArguments(),
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
					}
					node.SetLocation(postfixToken.Location)
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

func (p *parser) parsePipe(node Node) Node {
	identifier := p.current
	p.expect(Identifier)

	arguments := []Node{node}

	if b, ok := predicates[identifier.Value]; ok {
		p.expect(Bracket, "(")

		// TODO: Refactor parser to use builtin.Builtins instead of predicates map.

		if b.arity == 2 {
			arguments = append(arguments, p.parseClosure())
		}

		if identifier.Value == "reduce" {
			arguments = append(arguments, p.parseClosure())
			if p.current.Is(Operator, ",") {
				p.next()
				arguments = append(arguments, p.parseExpression(0))
			}
		}

		p.expect(Bracket, ")")

		node = &BuiltinNode{
			Name:      identifier.Value,
			Arguments: arguments,
		}
		node.SetLocation(identifier.Location)
	} else if _, ok := builtin.Index[identifier.Value]; ok {
		arguments = append(arguments, p.parseArguments()...)

		node = &BuiltinNode{
			Name:      identifier.Value,
			Arguments: arguments,
		}
		node.SetLocation(identifier.Location)
	} else {
		callee := &IdentifierNode{Value: identifier.Value}
		callee.SetLocation(identifier.Location)

		arguments = append(arguments, p.parseArguments()...)

		node = &CallNode{
			Callee:    callee,
			Arguments: arguments,
		}
		node.SetLocation(identifier.Location)
	}

	return node
}

func (p *parser) parseArguments() []Node {
	p.expect(Bracket, "(")
	nodes := make([]Node, 0)
	for !p.current.Is(Bracket, ")") && p.err == nil {
		if len(nodes) > 0 {
			p.expect(Operator, ",")
		}
		node := p.parseExpression(0)
		nodes = append(nodes, node)
	}
	p.expect(Bracket, ")")

	return nodes
}
