package conf

import (
	"fmt"
	"reflect"

	. "github.com/expr-lang/expr/checker/nature"
	"github.com/expr-lang/expr/internal/deref"
	"github.com/expr-lang/expr/types"
)

func Env(env any) Nature {
	if env == nil {
		return Nature{
			Type:   reflect.TypeOf(map[string]any{}),
			Strict: true,
		}
	}

	switch env := env.(type) {
	case types.Map:
		return env.Nature()
	}

	v := reflect.ValueOf(env)
	d := deref.Value(v)

	switch d.Kind() {
	case reflect.Struct:
		return Nature{
			Type:   v.Type(),
			Strict: true,
		}

	case reflect.Map:
		n := Nature{
			Type:   v.Type(),
			Fields: make(map[string]Nature, v.Len()),
			Strict: true,
		}

		for _, key := range v.MapKeys() {
			elem := v.MapIndex(key)
			if !elem.IsValid() || !elem.CanInterface() {
				panic(fmt.Sprintf("invalid map value: %s", key))
			}

			face := elem.Interface()

			switch face := face.(type) {
			case types.Map:
				n.Fields[key.String()] = face.Nature()

			default:
				if face == nil {
					n.Fields[key.String()] = Nature{Nil: true}
					continue
				}
				n.Fields[key.String()] = Nature{Type: reflect.TypeOf(face)}
			}

		}

		return n
	}

	panic(fmt.Sprintf("unknown type %T", env))
}
