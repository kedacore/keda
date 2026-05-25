/*
Package value holds Kusto data value representations. All types provide a Kusto that
stores the native value and Valid which indicates if the value was set or was null.

# Kusto Value

A value.Kusto can hold types that represent Kusto Scalar types that define column data.
We represent that with an interface:

	type Kusto interface

This interface can hold the following values:

	value.Bool
	value.Int
	value.Long
	value.Real
	value.Decimal
	value.String
	value.Dynamic
	value.DateTime
	value.Timespan

Each type defined above has at minimum two fields:

	.Value - The type specific value
	.Valid - True if the value was non-null in the Kusto table

Each provides at minimum the following two methods:

	.String() - Returns the string representation of the value.
	.Unmarshal() - Unmarshals the value into a standard Go type.

The Unmarshal() is for internal use, it should not be needed by an end user. Use .Value or table.Row.ToStruct() instead.
*/
package value

import (
	"fmt"
	"github.com/Azure/azure-kusto-go/azkustodata/errors"
	"github.com/Azure/azure-kusto-go/azkustodata/types"
	"reflect"
)

type pointerValue[T any] struct {
	value *T
}

func newPointerValue[T any](v *T) pointerValue[T] {
	return pointerValue[T]{value: v}
}

func (p *pointerValue[T]) String() string {
	if p.value == nil {
		return ""
	}
	return fmt.Sprintf("%v", *p.value)
}

func (p *pointerValue[T]) GetValue() interface{} {
	return p.value
}

func (p *pointerValue[T]) Ptr() *T {
	return p.value
}

func convertError(expected interface{}, actual interface{}) error {
	if ref, ok := actual.(reflect.Value); ok {
		return errors.ES(errors.OpTableAccess, errors.KWrongColumnType, "column with type '%T' had value that was %v", expected, ref.Type())
	}
	return errors.ES(errors.OpTableAccess, errors.KWrongColumnType, "column with type '%T' had value that was %T", expected, actual)
}

func parseError(expected interface{}, actual interface{}, err error) error {
	return errors.ES(errors.OpTableAccess, errors.KFailedToParse, "column with type '%T' had value %s which did not parse: %s", expected, actual, err)
}

func (p *pointerValue[T]) Unmarshal(i interface{}) error {
	if i == nil {
		p.value = nil
		return nil
	}

	v, ok := i.(T)
	if !ok {
		return convertError(p, i)
	}

	p.value = &v
	return nil
}

func TryConvert[T any](holder interface{}, p *pointerValue[T], v reflect.Value) bool {
	t := v.Type()

	if holder == nil || p.value == nil {
		v.Set(reflect.Zero(t))
		return true
	}

	if reflect.TypeOf(*p.value).ConvertibleTo(t) {
		v.Set(reflect.ValueOf(*p.value).Convert(t))
		return true
	}

	if reflect.TypeOf(p.value).ConvertibleTo(t) {
		v.Set(reflect.ValueOf(p.value).Convert(t))
		return true
	}

	if reflect.TypeOf(holder).ConvertibleTo(t) {
		v.Set(reflect.ValueOf(holder).Convert(t))
		return true
	}

	if reflect.TypeOf(&holder).ConvertibleTo(t) {
		v.Set(reflect.ValueOf(&holder).Convert(t))
		return true
	}

	return false
}

func Convert[T any](holder interface{}, p *pointerValue[T], v reflect.Value) error {
	if !TryConvert[T](holder, p, v) {
		return convertError(holder, v)
	}

	return nil
}

// Kusto represents a Kusto value.
type Kusto interface {
	fmt.Stringer
	Convert(v reflect.Value) error
	GetValue() interface{}
	GetType() types.Column
	Unmarshal(interface{}) error
}

func Default(t types.Column) Kusto {
	switch t {
	case types.Bool:
		return NewNullBool()
	case types.Int:
		return NewNullInt()
	case types.Long:
		return NewNullLong()
	case types.Real:
		return NewNullReal()
	case types.Decimal:
		return NewNullDecimal()
	case types.String:
		return NewString("")
	case types.Dynamic:
		return NewNullDynamic()
	case types.DateTime:
		return NewNullDateTime()
	case types.Timespan:
		return NewNullTimespan()
	case types.GUID:
		return NewNullGUID()
	default:
		return nil
	}
}

// Values is a list of Kusto values, usually an ordered row.
type Values []Kusto
