package conf

import (
	"fmt"
	"reflect"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/builtin"
	"github.com/expr-lang/expr/vm/runtime"
)

type Config struct {
	Env         any
	Types       TypesTable
	MapEnv      bool
	DefaultType reflect.Type
	Operators   OperatorsTable
	Expect      reflect.Kind
	ExpectAny   bool
	Optimize    bool
	Strict      bool
	ConstFns    map[string]reflect.Value
	Visitors    []ast.Visitor
	Functions   map[string]*builtin.Function
	Builtins    map[string]*builtin.Function
	Disabled    map[string]bool // disabled builtins
}

// CreateNew creates new config with default values.
func CreateNew() *Config {
	c := &Config{
		Optimize:  true,
		Operators: make(map[string][]string),
		ConstFns:  make(map[string]reflect.Value),
		Functions: make(map[string]*builtin.Function),
		Builtins:  make(map[string]*builtin.Function),
		Disabled:  make(map[string]bool),
	}
	for _, f := range builtin.Builtins {
		c.Builtins[f.Name] = f
	}
	return c
}

// New creates new config with environment.
func New(env any) *Config {
	c := CreateNew()
	c.WithEnv(env)
	return c
}

func (c *Config) WithEnv(env any) {
	var mapEnv bool
	var mapValueType reflect.Type
	if _, ok := env.(map[string]any); ok {
		mapEnv = true
	} else {
		if reflect.ValueOf(env).Kind() == reflect.Map {
			mapValueType = reflect.TypeOf(env).Elem()
		}
	}

	c.Env = env
	c.Types = CreateTypesTable(env)
	c.MapEnv = mapEnv
	c.DefaultType = mapValueType
	c.Strict = true
}

func (c *Config) Operator(operator string, fns ...string) {
	c.Operators[operator] = append(c.Operators[operator], fns...)
}

func (c *Config) ConstExpr(name string) {
	if c.Env == nil {
		panic("no environment is specified for ConstExpr()")
	}
	fn := reflect.ValueOf(runtime.Fetch(c.Env, name))
	if fn.Kind() != reflect.Func {
		panic(fmt.Errorf("const expression %q must be a function", name))
	}
	c.ConstFns[name] = fn
}

func (c *Config) Check() {
	for operator, fns := range c.Operators {
		for _, fn := range fns {
			fnType, ok := c.Types[fn]
			if !ok || fnType.Type.Kind() != reflect.Func {
				panic(fmt.Errorf("function %s for %s operator does not exist in the environment", fn, operator))
			}
			requiredNumIn := 2
			if fnType.Method {
				requiredNumIn = 3 // As first argument of method is receiver.
			}
			if fnType.Type.NumIn() != requiredNumIn || fnType.Type.NumOut() != 1 {
				panic(fmt.Errorf("function %s for %s operator does not have a correct signature", fn, operator))
			}
		}
	}
	for fnName, t := range c.Types {
		if kind(t.Type) == reflect.Func {
			for _, b := range c.Builtins {
				if b.Name == fnName {
					panic(fmt.Errorf(`cannot override builtin %s(): use expr.DisableBuiltin("%s") to override`, b.Name, b.Name))
				}
			}
		}
	}
	for _, f := range c.Functions {
		for _, b := range c.Builtins {
			if b.Name == f.Name {
				panic(fmt.Errorf(`cannot override builtin %s(); use expr.DisableBuiltin("%s") to override`, f.Name, f.Name))
			}
		}
	}
}
