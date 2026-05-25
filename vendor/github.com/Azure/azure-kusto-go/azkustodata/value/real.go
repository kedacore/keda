package value

import (
	"encoding/json"
	"github.com/Azure/azure-kusto-go/azkustodata/types"
	"reflect"
	"strconv"
)

// Real represents a Kusto real type.  Real implements Kusto.
type Real struct {
	pointerValue[float64]
}

func NewReal(i float64) *Real {
	return &Real{newPointerValue[float64](&i)}
}
func NewNullReal() *Real {
	return &Real{newPointerValue[float64](nil)}
}

// Unmarshal unmarshals i into Real. i must be a json.Number(that is a float64), float64 or nil.
func (r *Real) Unmarshal(i interface{}) error {
	if i == nil {
		r.value = nil
		return nil
	}

	var myFloat float64

	switch v := i.(type) {
	case json.Number:
		var err error
		myFloat, err = v.Float64()
		if err != nil {
			return parseError(r, i, err)
		}
	case float64:
		myFloat = v
	case string:
		var err error
		myFloat, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return parseError(r, i, err)
		}
	default:
		return convertError(r, i)
	}

	r.value = &myFloat
	return nil
}

// Convert Real into reflect value.
func (r *Real) Convert(v reflect.Value) error {
	if TryConvert[float64](*r, &r.pointerValue, v) {
		return nil
	}

	if v.Type().Kind() == reflect.Int || v.Type().Kind() == reflect.Int32 {
		if r.value != nil {
			v.SetInt(int64(*r.value))
		}
		return nil
	}

	return convertError(r, v)
}

// GetType returns the type of the value.
func (r *Real) GetType() types.Column {
	return types.Real
}
