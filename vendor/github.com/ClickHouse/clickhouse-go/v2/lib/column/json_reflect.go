package column

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2/lib/chcol"
)

// Decoding (Scanning)

// scanIntoStruct will iterate the provided struct and scan JSON data into the matching fields
func (c *JSON) scanIntoStruct(dest any, row int) error {
	val := reflect.ValueOf(dest)
	if val.Kind() != reflect.Pointer {
		return fmt.Errorf("destination must be a pointer")
	}
	val = val.Elem()

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("destination must be a pointer to struct")
	}

	return c.fillStruct(val, "", row)
}

// scanIntoMap converts JSON data into a map
func (c *JSON) scanIntoMap(dest any, row int) error {
	val := reflect.ValueOf(dest)
	if val.Kind() != reflect.Pointer {
		return fmt.Errorf("destination must be a pointer")
	}
	val = val.Elem()

	if val.Kind() != reflect.Map {
		return fmt.Errorf("destination must be a pointer to map")
	}

	if val.Type().Key().Kind() != reflect.String {
		return fmt.Errorf("map key must be string")
	}

	if val.IsNil() {
		val.Set(reflect.MakeMap(val.Type()))
	}

	return c.fillMap(val, "", row)
}

// fillStruct will iterate the provided struct and scan JSON data into the matching fields recursively
func (c *JSON) fillStruct(val reflect.Value, prefix string, row int) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if !field.CanSet() {
			continue
		}

		name := fieldType.Tag.Get("json")
		if name == "" || name[0] == ',' {
			name = fieldType.Name
		} else {
			name = strings.Split(name, ",")[0]
		}

		if name == "-" {
			continue
		}

		path := name
		if prefix != "" {
			path = prefix + "." + name
		}

		if c.hasTypedPath(path) {
			err := c.scanTypedPathToValue(path, row, field)
			if err != nil {
				return fmt.Errorf("fillStruct failed to scan typed path: %w", err)
			}

			continue
		} else if c.hasDynamicPath(path) {
			err := c.scanDynamicPathToValue(path, row, field)
			if err != nil {
				return fmt.Errorf("fillStruct failed to scan dynamic path: %w", err)
			}

			continue
		}

		hasNestedFields := c.pathHasNestedValues(path)
		if !hasNestedFields {
			continue
		}

		switch field.Kind() {
		case reflect.Pointer:
			if field.Type().Elem().Kind() == reflect.Struct {
				if field.IsNil() {
					field.Set(reflect.New(field.Type().Elem()))
				}

				if err := c.fillStruct(field.Elem(), path, row); err != nil {
					return fmt.Errorf("error filling nested struct pointer: %w", err)
				}
			}
		case reflect.Struct:
			if err := c.fillStruct(field, path, row); err != nil {
				return fmt.Errorf("error filling nested struct: %w", err)
			}
		case reflect.Map:
			if err := c.fillMap(field, path, row); err != nil {
				return fmt.Errorf("error filling nested map: %w", err)
			}
		}
	}

	return nil
}

// fillMap will iterate the provided map and scan JSON data in recursively
func (c *JSON) fillMap(val reflect.Value, prefix string, row int) error {
	if val.IsNil() {
		val.Set(reflect.MakeMap(val.Type()))
	}

	var paths []string
	for _, path := range c.typedPaths {
		if strings.HasPrefix(path, prefix) {
			paths = append(paths, path)
		}
	}
	for _, path := range c.dynamicPaths {
		if strings.HasPrefix(path, prefix) {
			paths = append(paths, path)
		}
	}

	children := make(map[string][]string)
	prefixLen := len(prefix)
	if prefixLen > 0 {
		prefixLen++ // splitter
	}

	for _, path := range paths {
		if prefixLen >= len(path) {
			continue
		}

		suffix := path[prefixLen:]
		nextDot := strings.Index(suffix, ".")
		var current string
		if nextDot == -1 {
			current = suffix
		} else {
			current = suffix[:nextDot]
		}
		children[current] = append(children[current], path)
	}

	for key, childPaths := range children {
		noChildNodes := true
		for _, path := range childPaths {
			if strings.Contains(path[prefixLen:], ".") {
				noChildNodes = false
				break
			}
		}

		if noChildNodes {
			fullPath := prefix
			if prefix != "" {
				fullPath += "."
			}
			fullPath += key

			mapValueType := val.Type().Elem()
			newVal := reflect.New(mapValueType).Elem()

			var err error
			if _, isTyped := c.typedPathsIndex[fullPath]; isTyped {
				err = c.scanTypedPathToValue(fullPath, row, newVal)
			} else {
				if mapValueType.Kind() == reflect.Interface {
					value := c.valueAtPath(fullPath, row, false)
					if dyn, ok := value.(chcol.Dynamic); ok {
						value = dyn.Any()
					}

					if value == nil {
						continue
					}

					newVal.Set(reflect.ValueOf(value))
				} else {
					err = c.scanDynamicPathToValue(fullPath, row, newVal)
				}
			}
			if err != nil {
				return fmt.Errorf("failed to scan value at path \"%s\": %w", fullPath, err)
			}

			val.SetMapIndex(reflect.ValueOf(key), newVal)
		} else {
			newPrefix := prefix
			if newPrefix != "" {
				newPrefix += "."
			}
			newPrefix += key

			mapValueType := val.Type().Elem()
			var newMap reflect.Value

			switch mapValueType.Kind() {
			case reflect.Interface:
				newMap = reflect.MakeMap(reflect.TypeOf(map[string]any{}))
			case reflect.Map:
				newMap = reflect.MakeMap(mapValueType)
			default:
				return fmt.Errorf("invalid map value type for nested path \"%s\"", newPrefix)
			}

			err := c.fillMap(newMap, newPrefix, row)
			if err != nil {
				return fmt.Errorf("failed filling nested map at path \"%s\": %w", newPrefix, err)
			}

			if newMap.Len() == 0 {
				continue
			}

			val.SetMapIndex(reflect.ValueOf(key), newMap)
		}
	}

	return nil
}

// Encoding (Append, AppendRow)

// structToJSON converts a struct to JSON data
func structToJSON(v any) (*chcol.JSON, error) {
	json := chcol.NewJSON()
	val := reflect.ValueOf(v)

	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %v", val.Kind())
	}

	err := iterateStruct(val, "", json)
	if err != nil {
		return nil, err
	}

	return json, nil
}

// mapToJSON converts a map to JSON data
func mapToJSON(v any) (*chcol.JSON, error) {
	json := chcol.NewJSON()
	val := reflect.ValueOf(v)

	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	if val.Kind() != reflect.Map {
		return nil, fmt.Errorf("expected map, got %v", val.Kind())
	}

	if val.Type().Key().Kind() != reflect.String {
		return nil, fmt.Errorf("map key must be string, got %v", val.Type().Key().Kind())
	}

	err := iterateMap(val, "", json)
	if err != nil {
		return nil, err
	}

	return json, nil
}

// iterateStruct recursively iterates through a struct and adds its fields to the JSON data
func iterateStruct(val reflect.Value, prefix string, json *chcol.JSON) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if !field.CanInterface() {
			continue
		}

		name := fieldType.Tag.Get("json")
		if name == "" || name[0] == ',' {
			name = fieldType.Name
		} else {
			// handle `json:"name,omitempty"`
			name = strings.Split(name, ",")[0]
		}

		if name == "-" {
			continue
		}

		path := name
		if prefix != "" {
			path = prefix + "." + name
		}

		forcedType := fieldType.Tag.Get("chType")
		err := handleValue(field, path, json, forcedType)
		if err != nil {
			return err
		}
	}

	return nil
}

// iterateStructSkipTypes is a set of struct types that will not be iterated.
// Instead, the value will be assigned directly for use within Dynamic row appending.
var iterateStructSkipTypes = map[reflect.Type]struct{}{
	scanTypeIP:           {},
	scanTypeUUID:         {},
	scanTypeTime:         {},
	scanTypeTime:         {},
	scanTypeRing:         {},
	scanTypePoint:        {},
	scanTypeBigInt:       {},
	scanTypePolygon:      {},
	scanTypeDecimal:      {},
	scanTypeMultiPolygon: {},
	scanTypeVariant:      {},
	scanTypeDynamic:      {},
	scanTypeJSON:         {},
}

// handleValue processes a single value and adds it to the JSON data
func handleValue(val reflect.Value, path string, json *chcol.JSON, forcedType string) error {
	if val.Kind() == reflect.Interface {
		val = val.Elem()
	}

	if !val.IsValid() {
		json.SetValueAtPath(path, nil)
		return nil
	}

	switch val.Kind() {
	case reflect.Pointer:
		if val.IsNil() {
			json.SetValueAtPath(path, nil)
			return nil
		}
		return handleValue(val.Elem(), path, json, forcedType)

	case reflect.Struct:
		if _, ok := iterateStructSkipTypes[val.Type()]; ok {
			json.SetValueAtPath(path, val.Interface())
			return nil
		}

		return iterateStruct(val, path, json)

	case reflect.Map:
		switch {
		case forcedType == "" && val.Type().Elem().Kind() == reflect.Interface:
			// Only iterate maps if they are map[string]interface{}
			return iterateMap(val, path, json)
		case forcedType == "":
			json.SetValueAtPath(path, val.Interface())
			return nil
		default:
			json.SetValueAtPath(path, chcol.NewDynamicWithType(val.Interface(), forcedType))
			return nil
		}
	case reflect.Slice, reflect.Array:
		if forcedType == "" {
			json.SetValueAtPath(path, val.Interface())
		} else {
			json.SetValueAtPath(path, chcol.NewDynamicWithType(val.Interface(), forcedType))
		}
		return nil
	default:
		if forcedType == "" {
			json.SetValueAtPath(path, val.Interface())
		} else {
			json.SetValueAtPath(path, chcol.NewDynamicWithType(val.Interface(), forcedType))
		}
		return nil
	}
}

const MaxMapPathDepth = 32

// iterateMap recursively iterates through a map and adds its values to the JSON data
func iterateMap(val reflect.Value, prefix string, json *chcol.JSON) error {
	depth := len(strings.Split(prefix, "."))
	if depth > MaxMapPathDepth {
		return fmt.Errorf("maximum nesting depth exceeded")
	}

	for _, key := range val.MapKeys() {
		if key.Kind() != reflect.String {
			return fmt.Errorf("map key must be string, got %v", key.Kind())
		}

		path := key.String()
		if prefix != "" {
			path = prefix + "." + path
		}

		mapValue := val.MapIndex(key)

		if mapValue.Kind() == reflect.Interface {
			mapValue = mapValue.Elem()
		}

		if mapValue.Kind() == reflect.Map {
			if err := iterateMap(mapValue, path, json); err != nil {
				return err
			}
		} else {
			if err := handleValue(mapValue, path, json, ""); err != nil {
				return err
			}
		}
	}

	return nil
}
