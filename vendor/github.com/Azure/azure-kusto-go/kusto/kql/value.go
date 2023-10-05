package kql

import (
	"fmt"
	"github.com/Azure/azure-kusto-go/kusto/data/types"
	"github.com/Azure/azure-kusto-go/kusto/data/value"
	"time"
)

type Value interface {
	fmt.Stringer
	Value() interface{}
	Type() types.Column
}

type kqlValue struct {
	value     interface{}
	kustoType types.Column
}

func (v *kqlValue) Value() interface{} {
	return v.value
}

func (v *kqlValue) Type() types.Column {
	return v.kustoType
}

func (v *kqlValue) String() string {
	val := v.value
	switch v.kustoType {
	case types.String:
		return QuoteString(val.(string), false)
	case types.DateTime:
		val = FormatDatetime(val.(time.Time))
	case types.Timespan:
		val = FormatTimespan(val.(time.Duration))
	case types.Dynamic:
		got := value.Dynamic{}
		_ = got.Unmarshal(val)
		val = got
	}

	return fmt.Sprintf("%v(%v)", v.kustoType, val)
}

func newValue(value interface{}, kustoType types.Column) Value {
	return &kqlValue{
		value:     value,
		kustoType: kustoType,
	}
}
