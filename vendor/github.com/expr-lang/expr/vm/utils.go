package vm

import (
	"reflect"
)

type Function = func(params ...any) (any, error)

var MemoryBudget uint = 1e6

var errorType = reflect.TypeOf((*error)(nil)).Elem()
