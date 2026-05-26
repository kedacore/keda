package value

import (
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-kusto-go/azkustodata/types"
	"reflect"
)

// Bool represents a Kusto boolean type. Bool implements Kusto.
type Bool struct {
	pointerValue[bool]
}

func NewBool(v bool) *Bool {
	return &Bool{newPointerValue[bool](&v)}
}

func NewNullBool() *Bool {
	return &Bool{newPointerValue[bool](nil)}
}

// Convert Bool into reflect value.
func (bo *Bool) Convert(v reflect.Value) error {
	return Convert[bool](*bo, &bo.pointerValue, v)
}

func (bo *Bool) Unmarshal(i interface{}) error {
	if i == nil {
		bo.value = nil
		return nil
	}

	// Boolean may sometimes be represented as an integer, 0 means false, 1 means true.
	if num, ok := i.(json.Number); ok {
		num, err := num.Int64()
		if err != nil {
			return parseError(bo, i, err)
		}

		bo.value = new(bool)

		if num == 0 {
			*bo.value = false
		} else if num == 1 {
			*bo.value = true
		} else {
			return parseError(bo, i, fmt.Errorf("expected 0 or 1, got %d", num))
		}
		return nil
	}

	return bo.pointerValue.Unmarshal(i)
}

// GetType returns the type of the value.
func (bo *Bool) GetType() types.Column {
	return types.Bool
}
