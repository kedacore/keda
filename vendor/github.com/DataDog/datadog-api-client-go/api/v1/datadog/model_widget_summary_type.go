// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetSummaryType Which summary type should be used.
type WidgetSummaryType string

// List of WidgetSummaryType.
const (
	WIDGETSUMMARYTYPE_MONITORS WidgetSummaryType = "monitors"
	WIDGETSUMMARYTYPE_GROUPS   WidgetSummaryType = "groups"
	WIDGETSUMMARYTYPE_COMBINED WidgetSummaryType = "combined"
)

var allowedWidgetSummaryTypeEnumValues = []WidgetSummaryType{
	WIDGETSUMMARYTYPE_MONITORS,
	WIDGETSUMMARYTYPE_GROUPS,
	WIDGETSUMMARYTYPE_COMBINED,
}

// GetAllowedValues reeturns the list of possible values.
func (v *WidgetSummaryType) GetAllowedValues() []WidgetSummaryType {
	return allowedWidgetSummaryTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *WidgetSummaryType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = WidgetSummaryType(value)
	return nil
}

// NewWidgetSummaryTypeFromValue returns a pointer to a valid WidgetSummaryType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewWidgetSummaryTypeFromValue(v string) (*WidgetSummaryType, error) {
	ev := WidgetSummaryType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for WidgetSummaryType: valid values are %v", v, allowedWidgetSummaryTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v WidgetSummaryType) IsValid() bool {
	for _, existing := range allowedWidgetSummaryTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to WidgetSummaryType value.
func (v WidgetSummaryType) Ptr() *WidgetSummaryType {
	return &v
}

// NullableWidgetSummaryType handles when a null is used for WidgetSummaryType.
type NullableWidgetSummaryType struct {
	value *WidgetSummaryType
	isSet bool
}

// Get returns the associated value.
func (v NullableWidgetSummaryType) Get() *WidgetSummaryType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableWidgetSummaryType) Set(val *WidgetSummaryType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableWidgetSummaryType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableWidgetSummaryType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableWidgetSummaryType initializes the struct as if Set has been called.
func NewNullableWidgetSummaryType(val *WidgetSummaryType) *NullableWidgetSummaryType {
	return &NullableWidgetSummaryType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableWidgetSummaryType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableWidgetSummaryType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
