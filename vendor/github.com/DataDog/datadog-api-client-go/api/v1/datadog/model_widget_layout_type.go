// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetLayoutType Layout type of the group.
type WidgetLayoutType string

// List of WidgetLayoutType.
const (
	WIDGETLAYOUTTYPE_ORDERED WidgetLayoutType = "ordered"
)

var allowedWidgetLayoutTypeEnumValues = []WidgetLayoutType{
	WIDGETLAYOUTTYPE_ORDERED,
}

// GetAllowedValues reeturns the list of possible values.
func (v *WidgetLayoutType) GetAllowedValues() []WidgetLayoutType {
	return allowedWidgetLayoutTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *WidgetLayoutType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = WidgetLayoutType(value)
	return nil
}

// NewWidgetLayoutTypeFromValue returns a pointer to a valid WidgetLayoutType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewWidgetLayoutTypeFromValue(v string) (*WidgetLayoutType, error) {
	ev := WidgetLayoutType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for WidgetLayoutType: valid values are %v", v, allowedWidgetLayoutTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v WidgetLayoutType) IsValid() bool {
	for _, existing := range allowedWidgetLayoutTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to WidgetLayoutType value.
func (v WidgetLayoutType) Ptr() *WidgetLayoutType {
	return &v
}

// NullableWidgetLayoutType handles when a null is used for WidgetLayoutType.
type NullableWidgetLayoutType struct {
	value *WidgetLayoutType
	isSet bool
}

// Get returns the associated value.
func (v NullableWidgetLayoutType) Get() *WidgetLayoutType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableWidgetLayoutType) Set(val *WidgetLayoutType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableWidgetLayoutType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableWidgetLayoutType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableWidgetLayoutType initializes the struct as if Set has been called.
func NewNullableWidgetLayoutType(val *WidgetLayoutType) *NullableWidgetLayoutType {
	return &NullableWidgetLayoutType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableWidgetLayoutType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableWidgetLayoutType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
