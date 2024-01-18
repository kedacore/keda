package ast

import (
	"reflect"

	"github.com/expr-lang/expr/file"
)

// Node represents items of abstract syntax tree.
type Node interface {
	Location() file.Location
	SetLocation(file.Location)
	Type() reflect.Type
	SetType(reflect.Type)
	String() string
}

// Patch replaces the node with a new one.
// Location information is preserved.
// Type information is lost.
func Patch(node *Node, newNode Node) {
	newNode.SetLocation((*node).Location())
	*node = newNode
}

// base is a base struct for all nodes.
type base struct {
	loc      file.Location
	nodeType reflect.Type
}

// Location returns the location of the node in the source code.
func (n *base) Location() file.Location {
	return n.loc
}

// SetLocation sets the location of the node in the source code.
func (n *base) SetLocation(loc file.Location) {
	n.loc = loc
}

// Type returns the type of the node.
func (n *base) Type() reflect.Type {
	return n.nodeType
}

// SetType sets the type of the node.
func (n *base) SetType(t reflect.Type) {
	n.nodeType = t
}

// NilNode represents nil.
type NilNode struct {
	base
}

// IdentifierNode represents an identifier.
type IdentifierNode struct {
	base
	Value string // Name of the identifier. Like "foo" in "foo.bar".
}

// IntegerNode represents an integer.
type IntegerNode struct {
	base
	Value int // Value of the integer.
}

// FloatNode represents a float.
type FloatNode struct {
	base
	Value float64 // Value of the float.
}

// BoolNode represents a boolean.
type BoolNode struct {
	base
	Value bool // Value of the boolean.
}

// StringNode represents a string.
type StringNode struct {
	base
	Value string // Value of the string.
}

// ConstantNode represents a constant.
// Constants are predefined values like nil, true, false, array, map, etc.
// The parser.Parse will never generate ConstantNode, it is only generated
// by the optimizer.
type ConstantNode struct {
	base
	Value any // Value of the constant.
}

// UnaryNode represents a unary operator.
type UnaryNode struct {
	base
	Operator string // Operator of the unary operator. Like "!" in "!foo" or "not" in "not foo".
	Node     Node   // Node of the unary operator. Like "foo" in "!foo".
}

// BinaryNode represents a binary operator.
type BinaryNode struct {
	base
	Operator string // Operator of the binary operator. Like "+" in "foo + bar" or "matches" in "foo matches bar".
	Left     Node   // Left node of the binary operator.
	Right    Node   // Right node of the binary operator.
}

// ChainNode represents an optional chaining group.
// A few MemberNode nodes can be chained together,
// and will be wrapped in a ChainNode. Example:
//
//	foo.bar?.baz?.qux
//
// The whole chain will be wrapped in a ChainNode.
type ChainNode struct {
	base
	Node Node // Node of the chain.
}

// MemberNode represents a member access.
// It can be a field access, a method call,
// or an array element access.
// Example:
//
//	foo.bar or foo["bar"]
//	foo.bar()
//	array[0]
type MemberNode struct {
	base
	Node     Node // Node of the member access. Like "foo" in "foo.bar".
	Property Node // Property of the member access. For property access it is a StringNode.
	Optional bool // If true then the member access is optional. Like "foo?.bar".
}

// SliceNode represents access to a slice of an array.
// Example:
//
//	array[1:4]
type SliceNode struct {
	base
	Node Node // Node of the slice. Like "array" in "array[1:4]".
	From Node // From an index of the array. Like "1" in "array[1:4]".
	To   Node // To an index of the array. Like "4" in "array[1:4]".
}

// CallNode represents a function or a method call.
type CallNode struct {
	base
	Callee    Node   // Node of the call. Like "foo" in "foo()".
	Arguments []Node // Arguments of the call.
}

// BuiltinNode represents a builtin function call.
type BuiltinNode struct {
	base
	Name      string // Name of the builtin function. Like "len" in "len(foo)".
	Arguments []Node // Arguments of the builtin function.
	Throws    bool   // If true then accessing a field or array index can throw an error. Used by optimizer.
	Map       Node   // Used by optimizer to fold filter() and map() builtins.
}

// ClosureNode represents a predicate.
// Example:
//
//	filter(foo, .bar == 1)
//
// The predicate is ".bar == 1".
type ClosureNode struct {
	base
	Node Node // Node of the predicate body.
}

// PointerNode represents a pointer to a current value in predicate.
type PointerNode struct {
	base
	Name string // Name of the pointer. Like "index" in "#index".
}

// ConditionalNode represents a ternary operator.
type ConditionalNode struct {
	base
	Cond Node // Condition of the ternary operator. Like "foo" in "foo ? bar : baz".
	Exp1 Node // Expression 1 of the ternary operator. Like "bar" in "foo ? bar : baz".
	Exp2 Node // Expression 2 of the ternary operator. Like "baz" in "foo ? bar : baz".
}

// VariableDeclaratorNode represents a variable declaration.
type VariableDeclaratorNode struct {
	base
	Name  string // Name of the variable. Like "foo" in "let foo = 1; foo + 1".
	Value Node   // Value of the variable. Like "1" in "let foo = 1; foo + 1".
	Expr  Node   // Expression of the variable. Like "foo + 1" in "let foo = 1; foo + 1".
}

// ArrayNode represents an array.
type ArrayNode struct {
	base
	Nodes []Node // Nodes of the array.
}

// MapNode represents a map.
type MapNode struct {
	base
	Pairs []Node // PairNode nodes.
}

// PairNode represents a key-value pair of a map.
type PairNode struct {
	base
	Key   Node // Key of the pair.
	Value Node // Value of the pair.
}
