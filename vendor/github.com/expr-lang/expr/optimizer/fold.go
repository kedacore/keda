package optimizer

import (
	"fmt"
	"math"
	"reflect"

	. "github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/file"
)

var (
	integerType = reflect.TypeOf(0)
	floatType   = reflect.TypeOf(float64(0))
	stringType  = reflect.TypeOf("")
)

type fold struct {
	applied bool
	err     *file.Error
}

func (fold *fold) Visit(node *Node) {
	patch := func(newNode Node) {
		fold.applied = true
		Patch(node, newNode)
	}
	patchWithType := func(newNode Node) {
		patch(newNode)
		switch newNode.(type) {
		case *IntegerNode:
			newNode.SetType(integerType)
		case *FloatNode:
			newNode.SetType(floatType)
		case *StringNode:
			newNode.SetType(stringType)
		default:
			panic(fmt.Sprintf("unknown type %T", newNode))
		}
	}

	switch n := (*node).(type) {
	case *UnaryNode:
		switch n.Operator {
		case "-":
			if i, ok := n.Node.(*IntegerNode); ok {
				patchWithType(&IntegerNode{Value: -i.Value})
			}
			if i, ok := n.Node.(*FloatNode); ok {
				patchWithType(&FloatNode{Value: -i.Value})
			}
		case "+":
			if i, ok := n.Node.(*IntegerNode); ok {
				patchWithType(&IntegerNode{Value: i.Value})
			}
			if i, ok := n.Node.(*FloatNode); ok {
				patchWithType(&FloatNode{Value: i.Value})
			}
		case "!", "not":
			if a := toBool(n.Node); a != nil {
				patch(&BoolNode{Value: !a.Value})
			}
		}

	case *BinaryNode:
		switch n.Operator {
		case "+":
			{
				a := toInteger(n.Left)
				b := toInteger(n.Right)
				if a != nil && b != nil {
					patchWithType(&IntegerNode{Value: a.Value + b.Value})
				}
			}
			{
				a := toInteger(n.Left)
				b := toFloat(n.Right)
				if a != nil && b != nil {
					patchWithType(&FloatNode{Value: float64(a.Value) + b.Value})
				}
			}
			{
				a := toFloat(n.Left)
				b := toInteger(n.Right)
				if a != nil && b != nil {
					patchWithType(&FloatNode{Value: a.Value + float64(b.Value)})
				}
			}
			{
				a := toFloat(n.Left)
				b := toFloat(n.Right)
				if a != nil && b != nil {
					patchWithType(&FloatNode{Value: a.Value + b.Value})
				}
			}
			{
				a := toString(n.Left)
				b := toString(n.Right)
				if a != nil && b != nil {
					patch(&StringNode{Value: a.Value + b.Value})
				}
			}
		case "-":
			{
				a := toInteger(n.Left)
				b := toInteger(n.Right)
				if a != nil && b != nil {
					patchWithType(&IntegerNode{Value: a.Value - b.Value})
				}
			}
			{
				a := toInteger(n.Left)
				b := toFloat(n.Right)
				if a != nil && b != nil {
					patchWithType(&FloatNode{Value: float64(a.Value) - b.Value})
				}
			}
			{
				a := toFloat(n.Left)
				b := toInteger(n.Right)
				if a != nil && b != nil {
					patchWithType(&FloatNode{Value: a.Value - float64(b.Value)})
				}
			}
			{
				a := toFloat(n.Left)
				b := toFloat(n.Right)
				if a != nil && b != nil {
					patchWithType(&FloatNode{Value: a.Value - b.Value})
				}
			}
		case "*":
			{
				a := toInteger(n.Left)
				b := toInteger(n.Right)
				if a != nil && b != nil {
					patchWithType(&IntegerNode{Value: a.Value * b.Value})
				}
			}
			{
				a := toInteger(n.Left)
				b := toFloat(n.Right)
				if a != nil && b != nil {
					patchWithType(&FloatNode{Value: float64(a.Value) * b.Value})
				}
			}
			{
				a := toFloat(n.Left)
				b := toInteger(n.Right)
				if a != nil && b != nil {
					patchWithType(&FloatNode{Value: a.Value * float64(b.Value)})
				}
			}
			{
				a := toFloat(n.Left)
				b := toFloat(n.Right)
				if a != nil && b != nil {
					patchWithType(&FloatNode{Value: a.Value * b.Value})
				}
			}
		case "/":
			{
				a := toInteger(n.Left)
				b := toInteger(n.Right)
				if a != nil && b != nil {
					patchWithType(&FloatNode{Value: float64(a.Value) / float64(b.Value)})
				}
			}
			{
				a := toInteger(n.Left)
				b := toFloat(n.Right)
				if a != nil && b != nil {
					patchWithType(&FloatNode{Value: float64(a.Value) / b.Value})
				}
			}
			{
				a := toFloat(n.Left)
				b := toInteger(n.Right)
				if a != nil && b != nil {
					patchWithType(&FloatNode{Value: a.Value / float64(b.Value)})
				}
			}
			{
				a := toFloat(n.Left)
				b := toFloat(n.Right)
				if a != nil && b != nil {
					patchWithType(&FloatNode{Value: a.Value / b.Value})
				}
			}
		case "%":
			if a, ok := n.Left.(*IntegerNode); ok {
				if b, ok := n.Right.(*IntegerNode); ok {
					if b.Value == 0 {
						fold.err = &file.Error{
							Location: (*node).Location(),
							Message:  "integer divide by zero",
						}
						return
					}
					patch(&IntegerNode{Value: a.Value % b.Value})
				}
			}
		case "**", "^":
			{
				a := toInteger(n.Left)
				b := toInteger(n.Right)
				if a != nil && b != nil {
					patchWithType(&FloatNode{Value: math.Pow(float64(a.Value), float64(b.Value))})
				}
			}
			{
				a := toInteger(n.Left)
				b := toFloat(n.Right)
				if a != nil && b != nil {
					patchWithType(&FloatNode{Value: math.Pow(float64(a.Value), b.Value)})
				}
			}
			{
				a := toFloat(n.Left)
				b := toInteger(n.Right)
				if a != nil && b != nil {
					patchWithType(&FloatNode{Value: math.Pow(a.Value, float64(b.Value))})
				}
			}
			{
				a := toFloat(n.Left)
				b := toFloat(n.Right)
				if a != nil && b != nil {
					patchWithType(&FloatNode{Value: math.Pow(a.Value, b.Value)})
				}
			}
		case "and", "&&":
			a := toBool(n.Left)
			b := toBool(n.Right)

			if a != nil && a.Value { // true and x
				patch(n.Right)
			} else if b != nil && b.Value { // x and true
				patch(n.Left)
			} else if (a != nil && !a.Value) || (b != nil && !b.Value) { // "x and false" or "false and x"
				patch(&BoolNode{Value: false})
			}
		case "or", "||":
			a := toBool(n.Left)
			b := toBool(n.Right)

			if a != nil && !a.Value { // false or x
				patch(n.Right)
			} else if b != nil && !b.Value { // x or false
				patch(n.Left)
			} else if (a != nil && a.Value) || (b != nil && b.Value) { // "x or true" or "true or x"
				patch(&BoolNode{Value: true})
			}
		case "==":
			{
				a := toInteger(n.Left)
				b := toInteger(n.Right)
				if a != nil && b != nil {
					patch(&BoolNode{Value: a.Value == b.Value})
				}
			}
			{
				a := toString(n.Left)
				b := toString(n.Right)
				if a != nil && b != nil {
					patch(&BoolNode{Value: a.Value == b.Value})
				}
			}
			{
				a := toBool(n.Left)
				b := toBool(n.Right)
				if a != nil && b != nil {
					patch(&BoolNode{Value: a.Value == b.Value})
				}
			}
		}

	case *ArrayNode:
		if len(n.Nodes) > 0 {
			for _, a := range n.Nodes {
				switch a.(type) {
				case *IntegerNode, *FloatNode, *StringNode, *BoolNode:
					continue
				default:
					return
				}
			}
			value := make([]any, len(n.Nodes))
			for i, a := range n.Nodes {
				switch b := a.(type) {
				case *IntegerNode:
					value[i] = b.Value
				case *FloatNode:
					value[i] = b.Value
				case *StringNode:
					value[i] = b.Value
				case *BoolNode:
					value[i] = b.Value
				}
			}
			patch(&ConstantNode{Value: value})
		}

	case *BuiltinNode:
		switch n.Name {
		case "filter":
			if len(n.Arguments) != 2 {
				return
			}
			if base, ok := n.Arguments[0].(*BuiltinNode); ok && base.Name == "filter" {
				patch(&BuiltinNode{
					Name: "filter",
					Arguments: []Node{
						base.Arguments[0],
						&BinaryNode{
							Operator: "&&",
							Left:     base.Arguments[1],
							Right:    n.Arguments[1],
						},
					},
				})
			}
		}
	}
}

func toString(n Node) *StringNode {
	switch a := n.(type) {
	case *StringNode:
		return a
	}
	return nil
}

func toInteger(n Node) *IntegerNode {
	switch a := n.(type) {
	case *IntegerNode:
		return a
	}
	return nil
}

func toFloat(n Node) *FloatNode {
	switch a := n.(type) {
	case *FloatNode:
		return a
	}
	return nil
}

func toBool(n Node) *BoolNode {
	switch a := n.(type) {
	case *BoolNode:
		return a
	}
	return nil
}
