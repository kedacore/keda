// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetDisplayType Type of display to use for the request.
type WidgetDisplayType string

// List of WidgetDisplayType.
const (
	WIDGETDISPLAYTYPE_AREA WidgetDisplayType = "area"
	WIDGETDISPLAYTYPE_BARS WidgetDisplayType = "bars"
	WIDGETDISPLAYTYPE_LINE WidgetDisplayType = "line"
)

var allowedWidgetDisplayTypeEnumValues = []WidgetDisplayType{
	WIDGETDISPLAYTYPE_AREA,
	WIDGETDISPLAYTYPE_BARS,
	WIDGETDISPLAYTYPE_LINE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *WidgetDisplayType) GetAllowedValues() []WidgetDisplayType {
	return allowedWidgetDisplayTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *WidgetDisplayType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = WidgetDisplayType(value)
	return nil
}

// NewWidgetDisplayTypeFromValue returns a pointer to a valid WidgetDisplayType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewWidgetDisplayTypeFromValue(v string) (*WidgetDisplayType, error) {
	ev := WidgetDisplayType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for WidgetDisplayType: valid values are %v", v, allowedWidgetDisplayTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v WidgetDisplayType) IsValid() bool {
	for _, existing := range allowedWidgetDisplayTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to WidgetDisplayType value.
func (v WidgetDisplayType) Ptr() *WidgetDisplayType {
	return &v
}

// NullableWidgetDisplayType handles when a null is used for WidgetDisplayType.
type NullableWidgetDisplayType struct {
	value *WidgetDisplayType
	isSet bool
}

// Get returns the associated value.
func (v NullableWidgetDisplayType) Get() *WidgetDisplayType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableWidgetDisplayType) Set(val *WidgetDisplayType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableWidgetDisplayType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableWidgetDisplayType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableWidgetDisplayType initializes the struct as if Set has been called.
func NewNullableWidgetDisplayType(val *WidgetDisplayType) *NullableWidgetDisplayType {
	return &NullableWidgetDisplayType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableWidgetDisplayType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableWidgetDisplayType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
