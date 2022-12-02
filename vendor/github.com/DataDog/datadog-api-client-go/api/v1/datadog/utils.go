// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"
)

// PtrBool is a helper routine that returns a pointer to given boolean value.
func PtrBool(v bool) *bool { return &v }

// PtrInt is a helper routine that returns a pointer to given integer value.
func PtrInt(v int) *int { return &v }

// PtrInt32 is a helper routine that returns a pointer to given integer value.
func PtrInt32(v int32) *int32 { return &v }

// PtrInt64 is a helper routine that returns a pointer to given integer value.
func PtrInt64(v int64) *int64 { return &v }

// PtrFloat32 is a helper routine that returns a pointer to given float value.
func PtrFloat32(v float32) *float32 { return &v }

// PtrFloat64 is a helper routine that returns a pointer to given float value.
func PtrFloat64(v float64) *float64 { return &v }

// PtrString is a helper routine that returns a pointer to given string value.
func PtrString(v string) *string { return &v }

// PtrTime is helper routine that returns a pointer to given Time value.
func PtrTime(v time.Time) *time.Time { return &v }

// NullableBool is a struct to hold a nullable boolean value.
type NullableBool struct {
	value *bool
	isSet bool
}

// Get returns the value associated with the nullable bool.
func (v NullableBool) Get() *bool {
	return v.value
}

// Set sets the value associated with the nullable bool.
func (v *NullableBool) Set(val *bool) {
	v.value = val
	v.isSet = true
}

// IsSet returns true if the value has been set.
func (v NullableBool) IsSet() bool {
	return v.isSet
}

// Unset resets fields of the nullable bool.
func (v *NullableBool) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableBool instantiates a new nullable bool.
func NewNullableBool(val *bool) *NullableBool {
	return &NullableBool{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableBool) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes to the associated value.
func (v *NullableBool) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

// NullableInt is a struct to hold a nullable int value.
type NullableInt struct {
	value *int
	isSet bool
}

// Get returns the value associated with the nullable int.
func (v NullableInt) Get() *int {
	return v.value
}

// Set sets the value associated with the nullable int.
func (v *NullableInt) Set(val *int) {
	v.value = val
	v.isSet = true
}

// IsSet returns true if the value has been set.
func (v NullableInt) IsSet() bool {
	return v.isSet
}

// Unset resets fields of the nullable int.
func (v *NullableInt) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableInt instantiates a new nullable int.
func NewNullableInt(val *int) *NullableInt {
	return &NullableInt{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes to the associated value.
func (v *NullableInt) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

// NullableInt32 is a struct to hold a nullable int32 value.
type NullableInt32 struct {
	value *int32
	isSet bool
}

// Get returns the value associated with the nullable int32.
func (v NullableInt32) Get() *int32 {
	return v.value
}

// Set sets the value associated with the nullable int32.
func (v *NullableInt32) Set(val *int32) {
	v.value = val
	v.isSet = true
}

// IsSet returns true if the value has been set.
func (v NullableInt32) IsSet() bool {
	return v.isSet
}

// Unset resets fields of the nullable int32.
func (v *NullableInt32) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableInt32 instantiates a new nullable int32.
func NewNullableInt32(val *int32) *NullableInt32 {
	return &NullableInt32{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableInt32) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes to the associated value.
func (v *NullableInt32) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

// NullableInt64 is a struct to hold a nullable int64 value.
type NullableInt64 struct {
	value *int64
	isSet bool
}

// Get returns the value associated with the nullable int64.
func (v NullableInt64) Get() *int64 {
	return v.value
}

// Set sets the value associated with the nullable int64.
func (v *NullableInt64) Set(val *int64) {
	v.value = val
	v.isSet = true
}

// IsSet returns true if the value has been set.
func (v NullableInt64) IsSet() bool {
	return v.isSet
}

// Unset resets fields of the nullable int64.
func (v *NullableInt64) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableInt64 instantiates a new nullable int64.
func NewNullableInt64(val *int64) *NullableInt64 {
	return &NullableInt64{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableInt64) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes to the associated value.
func (v *NullableInt64) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

// NullableFloat32 is a struct to hold a nullable float32 value.
type NullableFloat32 struct {
	value *float32
	isSet bool
}

// Get returns the value associated with the nullable float32.
func (v NullableFloat32) Get() *float32 {
	return v.value
}

// Set sets the value associated with the nullable float32.
func (v *NullableFloat32) Set(val *float32) {
	v.value = val
	v.isSet = true
}

// IsSet returns true if the value has been set.
func (v NullableFloat32) IsSet() bool {
	return v.isSet
}

// Unset resets fields of the nullable float32.
func (v *NullableFloat32) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableFloat32 instantiates a new nullable float32.
func NewNullableFloat32(val *float32) *NullableFloat32 {
	return &NullableFloat32{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableFloat32) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes to the associated value.
func (v *NullableFloat32) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

// NullableFloat64 is a struct to hold a nullable float64 value.
type NullableFloat64 struct {
	value *float64
	isSet bool
}

// Get returns the value associated with the nullable float64.
func (v NullableFloat64) Get() *float64 {
	return v.value
}

// Set sets the value associated with the nullable float64.
func (v *NullableFloat64) Set(val *float64) {
	v.value = val
	v.isSet = true
}

// IsSet returns true if the value has been set.
func (v NullableFloat64) IsSet() bool {
	return v.isSet
}

// Unset resets fields of the nullable float64.
func (v *NullableFloat64) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableFloat64 instantiates a new nullable float64.
func NewNullableFloat64(val *float64) *NullableFloat64 {
	return &NullableFloat64{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableFloat64) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes to the associated value.
func (v *NullableFloat64) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

// NullableString is a struct to hold a nullable string value.
type NullableString struct {
	value *string
	isSet bool
}

// Get returns the value associated with the nullable string.
func (v NullableString) Get() *string {
	return v.value
}

// Set sets the value associated with the nullable string.
func (v *NullableString) Set(val *string) {
	v.value = val
	v.isSet = true
}

// IsSet returns true if the value has been set.
func (v NullableString) IsSet() bool {
	return v.isSet
}

// Unset resets fields of the nullable string.
func (v *NullableString) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableString instantiates a new nullable string.
func NewNullableString(val *string) *NullableString {
	return &NullableString{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableString) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes to the associated value.
func (v *NullableString) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

// NullableTime is a struct to hold a nullable Time value.
type NullableTime struct {
	value *time.Time
	isSet bool
}

// Get returns the value associated with the nullable Time.
func (v NullableTime) Get() *time.Time {
	return v.value
}

// Set sets the value associated with the nullable Time.
func (v *NullableTime) Set(val *time.Time) {
	v.value = val
	v.isSet = true
}

// IsSet returns true if the value has been set.
func (v NullableTime) IsSet() bool {
	return v.isSet
}

// Unset resets fields of the nullable Time.
func (v *NullableTime) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableTime instantiates a new nullable Time.
func NewNullableTime(val *time.Time) *NullableTime {
	return &NullableTime{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableTime) MarshalJSON() ([]byte, error) {
	return v.value.MarshalJSON()
}

// UnmarshalJSON deserializes to the associated value.
func (v *NullableTime) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

// ContainsUnparsedObject returns true if the given data contains an unparsed object from the API.
func ContainsUnparsedObject(i interface{}) (bool, interface{}) {
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if n, m := ContainsUnparsedObject(v.Index(i).Interface()); n {
				return n, m
			}
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			if n, m := ContainsUnparsedObject(v.MapIndex(k).Interface()); n {
				return n, m
			}
		}
	case reflect.Struct:
		if u := v.FieldByName("UnparsedObject"); u.IsValid() && !u.IsNil() {
			return true, u.Interface()
		}
		for i := 0; i < v.NumField(); i++ {
			if fn := v.Type().Field(i).Name; string(fn[0]) == strings.ToUpper(string(fn[0])) && fn != "UnparsedObject" {
				if n, m := ContainsUnparsedObject(v.Field(i).Interface()); n {
					return n, m
				}
			} else if fn == "value" { // Special case for Nullables
				if get := v.MethodByName("Get"); get.IsValid() {
					if n, m := ContainsUnparsedObject(get.Call([]reflect.Value{})[0].Interface()); n {
						return n, m
					}
				}
			}
		}
	case reflect.Interface, reflect.Ptr:
		if !v.IsNil() {
			return ContainsUnparsedObject(v.Elem().Interface())
		}
	default:
		if v.IsValid() {
			if m := v.MethodByName("IsValid"); m.IsValid() {
				if !m.Call([]reflect.Value{})[0].Bool() {
					return true, v.Interface()
				}
			}
		}
	}
	return false, nil
}
