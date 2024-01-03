package ast

import "reflect"

type Function struct {
	Name      string
	Func      func(args ...any) (any, error)
	Fast      func(arg any) any
	Types     []reflect.Type
	Validate  func(args []reflect.Type) (reflect.Type, error)
	Predicate bool
}
