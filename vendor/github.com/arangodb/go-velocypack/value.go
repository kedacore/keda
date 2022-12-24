//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Ewout Prangsma
//

package velocypack

import (
	"reflect"
	"time"
)

// Value is a helper structure used to build VPack structures.
// It holds a single data value with a specific type.
type Value struct {
	vt        ValueType
	data      interface{}
	unindexed bool
}

// NewValue creates a new Value with type derived from Go type of given value.
// If the given value is not a supported type, a Value of type Illegal is returned.
func NewValue(value interface{}) Value {
	v := reflect.ValueOf(value)
	return NewReflectValue(v)
}

// NewReflectValue creates a new Value with type derived from Go type of given reflect value.
// If the given value is not a supported type, a Value of type Illegal is returned.
func NewReflectValue(v reflect.Value) Value {
	vt := v.Type()
	switch vt.Kind() {
	case reflect.Bool:
		return NewBoolValue(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return NewIntValue(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return NewUIntValue(v.Uint())
	case reflect.Float32, reflect.Float64:
		return NewDoubleValue(v.Float())
	case reflect.String:
		return NewStringValue(v.String())
	case reflect.Slice:
		if vt.Elem().Kind() == reflect.Uint8 {
		}
	}
	if v.CanInterface() {
		raw := v.Interface()
		if v, ok := raw.([]byte); ok {
			return NewBinaryValue(v)
		}
		if v, ok := raw.(Slice); ok {
			return NewSliceValue(v)
		}
		if v, ok := raw.(time.Time); ok {
			return NewUTCDateValue(v)
		}
		if v, ok := raw.(Value); ok {
			return v
		}
	}
	return Value{Illegal, nil, false}
}

// NewBoolValue creates a new Value of type Bool with given value.
func NewBoolValue(value bool) Value {
	return Value{Bool, value, false}
}

// NewIntValue creates a new Value of type Int with given value.
func NewIntValue(value int64) Value {
	if value >= -6 && value <= 9 {
		return Value{SmallInt, value, false}
	}
	return Value{Int, value, false}
}

// NewUIntValue creates a new Value of type UInt with given value.
func NewUIntValue(value uint64) Value {
	return Value{UInt, value, false}
}

// NewDoubleValue creates a new Value of type Double with given value.
func NewDoubleValue(value float64) Value {
	return Value{Double, value, false}
}

// NewStringValue creates a new Value of type String with given value.
func NewStringValue(value string) Value {
	return Value{String, value, false}
}

// NewBinaryValue creates a new Value of type Binary with given value.
func NewBinaryValue(value []byte) Value {
	return Value{Binary, value, false}
}

// NewUTCDateValue creates a new Value of type UTCDate with given value.
func NewUTCDateValue(value time.Time) Value {
	return Value{UTCDate, value, false}
}

// NewSliceValue creates a new Value of from the given slice.
func NewSliceValue(value Slice) Value {
	return Value{value.Type(), value, false}
}

// NewObjectValue creates a new Value that opens a new object.
func NewObjectValue(unindexed ...bool) Value {
	return Value{Object, nil, optionalBool(unindexed, false)}
}

// NewArrayValue creates a new Value that opens a new array.
func NewArrayValue(unindexed ...bool) Value {
	return Value{Array, nil, optionalBool(unindexed, false)}
}

// NewNullValue creates a new Value of type Null.
func NewNullValue() Value {
	return Value{Null, nil, false}
}

// NewMinKeyValue creates a new Value of type MinKey.
func NewMinKeyValue() Value {
	return Value{MinKey, nil, false}
}

// NewMaxKeyValue creates a new Value of type MaxKey.
func NewMaxKeyValue() Value {
	return Value{MaxKey, nil, false}
}

// Type returns the ValueType of this value.
func (v Value) Type() ValueType {
	return v.vt
}

// IsSlice returns true when the value already contains a slice.
func (v Value) IsSlice() bool {
	_, ok := v.data.(Slice)
	return ok
}

// IsIllegal returns true if the type of value is Illegal.
func (v Value) IsIllegal() bool {
	return v.vt == Illegal
}

func (v Value) boolValue() bool {
	return v.data.(bool)
}

func (v Value) intValue() int64 {
	return v.data.(int64)
}

func (v Value) uintValue() uint64 {
	return v.data.(uint64)
}

func (v Value) doubleValue() float64 {
	return v.data.(float64)
}

func (v Value) stringValue() string {
	return v.data.(string)
}

func (v Value) binaryValue() []byte {
	return v.data.([]byte)
}

func (v Value) utcDateValue() int64 {
	time := v.data.(time.Time)
	sec := time.Unix()
	nsec := int64(time.Nanosecond())
	return sec*1000 + nsec/1000000
}

func (v Value) sliceValue() Slice {
	return v.data.(Slice)
}
