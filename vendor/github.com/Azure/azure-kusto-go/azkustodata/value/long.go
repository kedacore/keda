package value

import (
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-kusto-go/azkustodata/types"
	"math"
	"reflect"
)

// Long represents a Kusto long type, which is an int64.  Long implements Kusto.
type Long struct {
	pointerValue[int64]
}

func NewLong(i int64) *Long { return &Long{newPointerValue[int64](&i)} }

func NewNullLong() *Long { return &Long{newPointerValue[int64](nil)} }

// Unmarshal unmarshals i into Long. i must be an int64 or nil.
func (l *Long) Unmarshal(i interface{}) error {
	if i == nil {
		l.value = nil
		return nil
	}

	var myInt int64

	switch v := i.(type) {
	case json.Number:
		var err error
		myInt, err = v.Int64()
		if err != nil {
			return parseError(l, i, err)
		}
	case float64:
		if v != math.Trunc(v) {
			return parseError(l, i, fmt.Errorf("float64 value was not an integer"))
		}
		myInt = int64(v)
	case int:
		myInt = int64(v)
	default:
		return convertError(l, i)
	}

	l.value = &myInt
	return nil
}

// Convert Long into reflect value.
func (l *Long) Convert(v reflect.Value) error {
	if TryConvert[int64](*l, &l.pointerValue, v) {
		return nil
	}

	if v.Type().Kind() == reflect.Int || v.Type().Kind() == reflect.Int32 {
		if l.value != nil {
			v.SetInt(*l.value)
		}
		return nil
	}

	return convertError(l, v)
}

// GetType returns the type of the value.
func (l *Long) GetType() types.Column { return types.Long }
