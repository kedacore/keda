package otto

import (
	"fmt"
	"math"
	"reflect"
	"unicode/utf16"
)

func (v Value) bool() bool {
	if v.kind == valueBoolean {
		return v.value.(bool)
	}
	if v.IsUndefined() || v.IsNull() {
		return false
	}
	switch value := v.value.(type) {
	case bool:
		return value
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(value).Int() != 0
	case uint, uint8, uint16, uint32, uint64:
		return reflect.ValueOf(value).Uint() != 0
	case float32:
		return value != 0
	case float64:
		if math.IsNaN(value) || value == 0 {
			return false
		}
		return true
	case string:
		return len(value) != 0
	case []uint16:
		return len(utf16.Decode(value)) != 0
	}
	if v.IsObject() {
		return true
	}
	panic(fmt.Sprintf("unexpected boolean type %T", v.value))
}
