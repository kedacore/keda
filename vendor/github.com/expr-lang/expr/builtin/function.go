package builtin

import (
	"reflect"
)

type Function struct {
	Name         string
	Func         func(args ...any) (any, error)
	Fast         func(arg any) any
	ValidateArgs func(args ...any) (any, error)
	Types        []reflect.Type
	Validate     func(args []reflect.Type) (reflect.Type, error)
	Predicate    bool
}

func (f *Function) Type() reflect.Type {
	if len(f.Types) > 0 {
		return f.Types[0]
	}
	return reflect.TypeOf(f.Func)
}
