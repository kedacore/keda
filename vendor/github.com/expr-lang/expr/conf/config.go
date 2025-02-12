package conf

import (
	"fmt"
	"reflect"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/builtin"
	"github.com/expr-lang/expr/vm/runtime"
)

type FunctionsTable map[string]*builtin.Function

type Config struct {
	Env         any
	Types       TypesTable
	MapEnv      bool
	DefaultType reflect.Type
	Expect      reflect.Kind
	ExpectAny   bool
	Optimize    bool
	Strict      bool
	Profile     bool
	ConstFns    map[string]reflect.Value
	Visitors    []ast.Visitor
	Functions   FunctionsTable
	Builtins    FunctionsTable
	Disabled    map[string]bool // disabled builtins
}

// CreateNew creates new config with default values.
func CreateNew() *Config {
	c := &Config{
		Optimize:  true,
		Types:     make(TypesTable),
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
	types := CreateTypesTable(env)
	for name, t := range types {
		c.Types[name] = t
	}
	c.MapEnv = mapEnv
	c.DefaultType = mapValueType
	c.Strict = true
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

type Checker interface {
	Check()
}

func (c *Config) Check() {
	for _, v := range c.Visitors {
		if c, ok := v.(Checker); ok {
			c.Check()
		}
	}
}

func (c *Config) IsOverridden(name string) bool {
	if _, ok := c.Functions[name]; ok {
		return true
	}
	if _, ok := c.Types[name]; ok {
		return true
	}
	return false
}
