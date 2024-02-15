package util

import (
	"fmt"
	"strings"
)

// GetValueByPath retrieves a value from a nested map using a dot-separated path
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
