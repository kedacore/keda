package serialization

import (
	"encoding/json"
	"strconv"
	"strings"
)

// MapStringInterface is used for custom unmarshaling of
// fields that have potentially dynamic types.
// E.g. when a field can be a string or an object/map
type MapStringInterface map[string]interface{}
type mapStringInterfaceProxy MapStringInterface

// UnmarshalJSON is a custom unmarshal method to guard against
// fields that can have more than one type returned from an API.
func (c *MapStringInterface) UnmarshalJSON(data []byte) error {
	var mapStrInterface mapStringInterfaceProxy

	str := string(data)

	// Check for empty JSON string
	if str == `""` {
		return nil
	}

	// Remove quotes if this is a string representation of JSON
	if strings.HasPrefix(str, "\"") && strings.HasSuffix(str, "\"") {
		s, err := strconv.Unquote(str)
		if err != nil {
			return nil
		}

		data = []byte(s)
	}

	err := json.Unmarshal(data, &mapStrInterface)
	if err != nil {
		return err
	}

	*c = MapStringInterface(mapStrInterface)

	return nil
}
