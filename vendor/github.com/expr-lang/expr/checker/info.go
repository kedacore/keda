package checker

import (
	"reflect"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/conf"
	"github.com/expr-lang/expr/vm"
)

func FieldIndex(types conf.TypesTable, node ast.Node) (bool, []int, string) {
	switch n := node.(type) {
	case *ast.IdentifierNode:
		if t, ok := types[n.Value]; ok && len(t.FieldIndex) > 0 {
			return true, t.FieldIndex, n.Value
		}
	case *ast.MemberNode:
		base := n.Node.Type()
		if kind(base) == reflect.Ptr {
			base = base.Elem()
		}
		if kind(base) == reflect.Struct {
			if prop, ok := n.Property.(*ast.StringNode); ok {
				name := prop.Value
				if field, ok := fetchField(base, name); ok {
					return true, field.Index, name
				}
			}
		}
	}
	return false, nil, ""
}

func MethodIndex(types conf.TypesTable, node ast.Node) (bool, int, string) {
	switch n := node.(type) {
	case *ast.IdentifierNode:
		if t, ok := types[n.Value]; ok {
			return t.Method, t.MethodIndex, n.Value
		}
	case *ast.MemberNode:
		if name, ok := n.Property.(*ast.StringNode); ok {
			base := n.Node.Type()
			if base != nil && base.Kind() != reflect.Interface {
				if m, ok := base.MethodByName(name.Value); ok {
					return true, m.Index, name.Value
				}
			}
		}
	}
	return false, 0, ""
}

func TypedFuncIndex(fn reflect.Type, method bool) (int, bool) {
	if fn == nil {
		return 0, false
	}
	if fn.Kind() != reflect.Func {
		return 0, false
	}
	// OnCallTyped doesn't work for functions with variadic arguments.
	if fn.IsVariadic() {
		return 0, false
	}
	// OnCallTyped doesn't work named function, like `type MyFunc func() int`.
	if fn.PkgPath() != "" { // If PkgPath() is not empty, it means that function is named.
		return 0, false
	}

	fnNumIn := fn.NumIn()
	fnInOffset := 0
	if method {
		fnNumIn--
		fnInOffset = 1
	}

funcTypes:
	for i := range vm.FuncTypes {
		if i == 0 {
			continue
		}
		typed := reflect.ValueOf(vm.FuncTypes[i]).Elem().Type()
		if typed.Kind() != reflect.Func {
			continue
		}
		if typed.NumOut() != fn.NumOut() {
			continue
		}
		for j := 0; j < typed.NumOut(); j++ {
			if typed.Out(j) != fn.Out(j) {
				continue funcTypes
			}
		}
		if typed.NumIn() != fnNumIn {
			continue
		}
		for j := 0; j < typed.NumIn(); j++ {
			if typed.In(j) != fn.In(j+fnInOffset) {
				continue funcTypes
			}
		}
		return i, true
	}
	return 0, false
}

func IsFastFunc(fn reflect.Type, method bool) bool {
	if fn == nil {
		return false
	}
	if fn.Kind() != reflect.Func {
		return false
	}
	numIn := 1
	if method {
		numIn = 2
	}
	if !isAny(fn) &&
		fn.IsVariadic() &&
		fn.NumIn() == numIn &&
		fn.NumOut() == 1 &&
		fn.Out(0).Kind() == reflect.Interface {
		rest := fn.In(fn.NumIn() - 1) // function has only one param for functions and two for methods
		if kind(rest) == reflect.Slice && rest.Elem().Kind() == reflect.Interface {
			return true
		}
	}
	return false
}
