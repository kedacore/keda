package nature

import (
	"reflect"

	"github.com/expr-lang/expr/internal/deref"
)

func fieldName(field reflect.StructField) string {
	if taggedName := field.Tag.Get("expr"); taggedName != "" {
		return taggedName
	}
	return field.Name
}

func fetchField(t reflect.Type, name string) (reflect.StructField, bool) {
	// First check all structs fields.
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Search all fields, even embedded structs.
		if fieldName(field) == name {
			return field, true
		}
	}

	// Second check fields of embedded structs.
	for i := 0; i < t.NumField(); i++ {
		anon := t.Field(i)
		if anon.Anonymous {
			anonType := anon.Type
			if anonType.Kind() == reflect.Pointer {
				anonType = anonType.Elem()
			}
			if field, ok := fetchField(anonType, name); ok {
				field.Index = append(anon.Index, field.Index...)
				return field, true
			}
		}
	}

	return reflect.StructField{}, false
}

func StructFields(t reflect.Type) map[string]Nature {
	table := make(map[string]Nature)

	t = deref.Type(t)
	if t == nil {
		return table
	}

	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)

			if f.Anonymous {
				for name, typ := range StructFields(f.Type) {
					if _, ok := table[name]; ok {
						continue
					}
					typ.FieldIndex = append(f.Index, typ.FieldIndex...)
					table[name] = typ
				}
			}

			table[fieldName(f)] = Nature{
				Type:       f.Type,
				FieldIndex: f.Index,
			}

		}
	}

	return table
}
