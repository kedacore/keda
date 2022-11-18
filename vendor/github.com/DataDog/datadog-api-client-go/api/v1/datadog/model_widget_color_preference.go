// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetColorPreference Which color to use on the widget.
type WidgetColorPreference string

// List of WidgetColorPreference.
const (
	WIDGETCOLORPREFERENCE_BACKGROUND WidgetColorPreference = "background"
	WIDGETCOLORPREFERENCE_TEXT       WidgetColorPreference = "text"
)

var allowedWidgetColorPreferenceEnumValues = []WidgetColorPreference{
	WIDGETCOLORPREFERENCE_BACKGROUND,
	WIDGETCOLORPREFERENCE_TEXT,
}

// GetAllowedValues reeturns the list of possible values.
func (v *WidgetColorPreference) GetAllowedValues() []WidgetColorPreference {
	return allowedWidgetColorPreferenceEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *WidgetColorPreference) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = WidgetColorPreference(value)
	return nil
}

// NewWidgetColorPreferenceFromValue returns a pointer to a valid WidgetColorPreference
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewWidgetColorPreferenceFromValue(v string) (*WidgetColorPreference, error) {
	ev := WidgetColorPreference(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for WidgetColorPreference: valid values are %v", v, allowedWidgetColorPreferenceEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v WidgetColorPreference) IsValid() bool {
	for _, existing := range allowedWidgetColorPreferenceEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to WidgetColorPreference value.
func (v WidgetColorPreference) Ptr() *WidgetColorPreference {
	return &v
}

// NullableWidgetColorPreference handles when a null is used for WidgetColorPreference.
type NullableWidgetColorPreference struct {
	value *WidgetColorPreference
	isSet bool
}

// Get returns the associated value.
func (v NullableWidgetColorPreference) Get() *WidgetColorPreference {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableWidgetColorPreference) Set(val *WidgetColorPreference) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableWidgetColorPreference) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableWidgetColorPreference) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableWidgetColorPreference initializes the struct as if Set has been called.
func NewNullableWidgetColorPreference(val *WidgetColorPreference) *NullableWidgetColorPreference {
	return &NullableWidgetColorPreference{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableWidgetColorPreference) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableWidgetColorPreference) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
