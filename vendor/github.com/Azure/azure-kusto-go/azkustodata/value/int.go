package value

import (
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-kusto-go/azkustodata/types"
	"math"
	"reflect"
)

// Int represents a Kusto boolean type. Bool implements Kusto.
type Int struct {
	pointerValue[int32]
}

func NewInt(v int32) *Int {
	return &Int{newPointerValue[int32](&v)}
}

func NewNullInt() *Int {
	return &Int{newPointerValue[int32](nil)}
}

// Convert Int into reflect value.
func (in *Int) Convert(v reflect.Value) error {
	if TryConvert[int32](*in, &in.pointerValue, v) {
		return nil
	}

	if v.Type().Kind() == reflect.Int {
		if in.value != nil {
			v.SetInt(int64(*in.value))
		}
		return nil
	}

	return convertError(in, v)
}

// GetType returns the type of the value.
func (in *Int) GetType() types.Column {
	return types.Int
}

func (in *Int) Unmarshal(i interface{}) error {
	if i == nil {
		in.value = nil
		return nil
	}

	var myInt int64

	switch v := i.(type) {
	case json.Number:
		var err error
		myInt, err = v.Int64()
		if err != nil {
			return parseError(in, i, err)
		}
	case float64:
		if v != math.Trunc(v) {
			return parseError(in, i, fmt.Errorf("float64 value was not an integer"))
		}
		myInt = int64(v)
	case int:
		myInt = int64(v)
	default:
		return convertError(in, i)
	}

	if myInt > math.MaxInt32 {
		return parseError(in, i, fmt.Errorf("value was too large for int32"))
	}
	val := int32(myInt)
	in.value = &val
	return nil
}
