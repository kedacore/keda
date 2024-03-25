package util

import (
	"fmt"
	"strings"
)

// GetValueByPath retrieves a value from a nested map using a dot-separated path
// It also supports .number syntax to access array elements.
//
// This is a helper function for niche use cases.
// Consider using https://pkg.go.dev/k8s.io/apimachinery@v0.29.3/pkg/apis/meta/v1/unstructured#NestedFieldNoCopy instead
//
// Examples:
//
//	data := map[string]interface{}{
//	    "a": map[string]interface{}{
//	        "b": []interface{}{
//	            map[string]interface{}{"c": 1},
//	            map[string]interface{}{"c": 2},
//	        },
//	    },
//	}
//
// GetValueByPath(data, "a.b.0.c") // 1
// GetValueByPath(data, "not.found") // error
func GetValueByPath(data map[string]interface{}, path string) (interface{}, error) {
	keys := strings.Split(path, ".")
	current := data

	for _, key := range keys {
		val, ok := current[key]
		if !ok {
			return nil, fmt.Errorf("key '%s' not found in path '%s'", key, path)
		}

		switch typedValue := val.(type) {
		case map[interface{}]interface{}:
			// Convert map[interface{}]interface{} to map[string]interface{}
			current = make(map[string]interface{})
			for k, v := range typedValue {
				current[fmt.Sprintf("%v", k)] = v
			}
		case []interface{}:
			// Convert map[interface{}]interface{} to map[string]interface{}
			current = make(map[string]interface{})
			for k, v := range typedValue {
				current[fmt.Sprintf("%v", k)] = v
			}
		case map[string]interface{}:
			current = typedValue
		default:
			// Reached the final value
			return val, nil
		}
	}

	return nil, fmt.Errorf("path '%s' does not lead to a value", path)
}
