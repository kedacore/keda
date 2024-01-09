package optimizer

import (
	. "github.com/expr-lang/expr/ast"
)

type filterMap struct{}

func (*filterMap) Visit(node *Node) {
	if mapBuiltin, ok := (*node).(*BuiltinNode); ok &&
		mapBuiltin.Name == "map" &&
		len(mapBuiltin.Arguments) == 2 {
		if closure, ok := mapBuiltin.Arguments[1].(*ClosureNode); ok {
			if filter, ok := mapBuiltin.Arguments[0].(*BuiltinNode); ok &&
				filter.Name == "filter" &&
				filter.Map == nil /* not already optimized */ {
				Patch(node, &BuiltinNode{
					Name:      "filter",
					Arguments: filter.Arguments,
					Map:       closure.Node,
				})
			}
		}
	}
}
