package patcher

import (
	"reflect"

	"github.com/expr-lang/expr/ast"
)

// WithContext adds WithContext.Name argument to all functions calls with a context.Context argument.
type WithContext struct {
	Name string
}

// Visit adds WithContext.Name argument to all functions calls with a context.Context argument.
func (w WithContext) Visit(node *ast.Node) {
	switch (*node).(type) {
	case *ast.CallNode:
		call := (*node).(*ast.CallNode)
		fn := call.Callee.Type()
		if fn.Kind() != reflect.Func {
			return
		}
		if fn.NumIn() == 0 {
			return
		}
		if fn.In(0).String() != "context.Context" {
			return
		}
		ast.Patch(node, &ast.CallNode{
			Callee: call.Callee,
			Arguments: append([]ast.Node{
				&ast.IdentifierNode{Value: w.Name},
			}, call.Arguments...),
		})
	}
}
