package ast

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/expr-lang/expr/parser/operator"
	"github.com/expr-lang/expr/parser/utils"
)

func (n *NilNode) String() string {
	return "nil"
}

func (n *IdentifierNode) String() string {
	return n.Value
}

func (n *IntegerNode) String() string {
	return fmt.Sprintf("%d", n.Value)
}

func (n *FloatNode) String() string {
	return fmt.Sprintf("%v", n.Value)
}

func (n *BoolNode) String() string {
	return fmt.Sprintf("%t", n.Value)
}

func (n *StringNode) String() string {
	return fmt.Sprintf("%q", n.Value)
}

func (n *ConstantNode) String() string {
	if n.Value == nil {
		return "nil"
	}
	b, err := json.Marshal(n.Value)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (n *UnaryNode) String() string {
	op := ""
	if n.Operator == "not" {
		op = fmt.Sprintf("%s ", n.Operator)
	} else {
		op = fmt.Sprintf("%s", n.Operator)
	}
	if _, ok := n.Node.(*BinaryNode); ok {
		return fmt.Sprintf("%s(%s)", op, n.Node.String())
	}
	return fmt.Sprintf("%s%s", op, n.Node.String())
}

func (n *BinaryNode) String() string {
	var lhs, rhs string

	lb, ok := n.Left.(*BinaryNode)
	if ok && (operator.Less(lb.Operator, n.Operator) || lb.Operator == "??") {
		lhs = fmt.Sprintf("(%s)", n.Left.String())
	} else {
		lhs = n.Left.String()
	}

	rb, ok := n.Right.(*BinaryNode)
	if ok && operator.Less(rb.Operator, n.Operator) {
		rhs = fmt.Sprintf("(%s)", n.Right.String())
	} else {
		rhs = n.Right.String()
	}

	return fmt.Sprintf("%s %s %s", lhs, n.Operator, rhs)
}

func (n *ChainNode) String() string {
	return n.Node.String()
}

func (n *MemberNode) String() string {
	if n.Optional {
		if str, ok := n.Property.(*StringNode); ok && utils.IsValidIdentifier(str.Value) {
			return fmt.Sprintf("%s?.%s", n.Node.String(), str.Value)
		} else {
			return fmt.Sprintf("get(%s, %s)", n.Node.String(), n.Property.String())
		}
	}
	if str, ok := n.Property.(*StringNode); ok && utils.IsValidIdentifier(str.Value) {
		if _, ok := n.Node.(*PointerNode); ok {
			return fmt.Sprintf(".%s", str.Value)
		}
		return fmt.Sprintf("%s.%s", n.Node.String(), str.Value)
	}
	return fmt.Sprintf("%s[%s]", n.Node.String(), n.Property.String())
}

func (n *SliceNode) String() string {
	if n.From == nil && n.To == nil {
		return fmt.Sprintf("%s[:]", n.Node.String())
	}
	if n.From == nil {
		return fmt.Sprintf("%s[:%s]", n.Node.String(), n.To.String())
	}
	if n.To == nil {
		return fmt.Sprintf("%s[%s:]", n.Node.String(), n.From.String())
	}
	return fmt.Sprintf("%s[%s:%s]", n.Node.String(), n.From.String(), n.To.String())
}

func (n *CallNode) String() string {
	arguments := make([]string, len(n.Arguments))
	for i, arg := range n.Arguments {
		arguments[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", n.Callee.String(), strings.Join(arguments, ", "))
}

func (n *BuiltinNode) String() string {
	arguments := make([]string, len(n.Arguments))
	for i, arg := range n.Arguments {
		arguments[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", n.Name, strings.Join(arguments, ", "))
}

func (n *ClosureNode) String() string {
	return n.Node.String()
}

func (n *PointerNode) String() string {
	return "#"
}

func (n *VariableDeclaratorNode) String() string {
	return fmt.Sprintf("let %s = %s; %s", n.Name, n.Value.String(), n.Expr.String())
}

func (n *ConditionalNode) String() string {
	var cond, exp1, exp2 string
	if _, ok := n.Cond.(*ConditionalNode); ok {
		cond = fmt.Sprintf("(%s)", n.Cond.String())
	} else {
		cond = n.Cond.String()
	}
	if _, ok := n.Exp1.(*ConditionalNode); ok {
		exp1 = fmt.Sprintf("(%s)", n.Exp1.String())
	} else {
		exp1 = n.Exp1.String()
	}
	if _, ok := n.Exp2.(*ConditionalNode); ok {
		exp2 = fmt.Sprintf("(%s)", n.Exp2.String())
	} else {
		exp2 = n.Exp2.String()
	}
	return fmt.Sprintf("%s ? %s : %s", cond, exp1, exp2)
}

func (n *ArrayNode) String() string {
	nodes := make([]string, len(n.Nodes))
	for i, node := range n.Nodes {
		nodes[i] = node.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(nodes, ", "))
}

func (n *MapNode) String() string {
	pairs := make([]string, len(n.Pairs))
	for i, pair := range n.Pairs {
		pairs[i] = pair.String()
	}
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}

func (n *PairNode) String() string {
	return fmt.Sprintf("%s: %s", n.Key.String(), n.Value.String())
}
