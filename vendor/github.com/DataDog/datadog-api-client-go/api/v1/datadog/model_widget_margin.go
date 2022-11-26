// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetMargin Size of the margins around the image.
// **Note**: `small` and `large` values are deprecated.
type WidgetMargin string

// List of WidgetMargin.
const (
	WIDGETMARGIN_SM    WidgetMargin = "sm"
	WIDGETMARGIN_MD    WidgetMargin = "md"
	WIDGETMARGIN_LG    WidgetMargin = "lg"
	WIDGETMARGIN_SMALL WidgetMargin = "small"
	WIDGETMARGIN_LARGE WidgetMargin = "large"
)

var allowedWidgetMarginEnumValues = []WidgetMargin{
	WIDGETMARGIN_SM,
	WIDGETMARGIN_MD,
	WIDGETMARGIN_LG,
	WIDGETMARGIN_SMALL,
	WIDGETMARGIN_LARGE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *WidgetMargin) GetAllowedValues() []WidgetMargin {
	return allowedWidgetMarginEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *WidgetMargin) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = WidgetMargin(value)
	return nil
}

// NewWidgetMarginFromValue returns a pointer to a valid WidgetMargin
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewWidgetMarginFromValue(v string) (*WidgetMargin, error) {
	ev := WidgetMargin(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for WidgetMargin: valid values are %v", v, allowedWidgetMarginEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v WidgetMargin) IsValid() bool {
	for _, existing := range allowedWidgetMarginEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to WidgetMargin value.
func (v WidgetMargin) Ptr() *WidgetMargin {
	return &v
}

// NullableWidgetMargin handles when a null is used for WidgetMargin.
type NullableWidgetMargin struct {
	value *WidgetMargin
	isSet bool
}

// Get returns the associated value.
func (v NullableWidgetMargin) Get() *WidgetMargin {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableWidgetMargin) Set(val *WidgetMargin) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableWidgetMargin) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableWidgetMargin) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableWidgetMargin initializes the struct as if Set has been called.
func NewNullableWidgetMargin(val *WidgetMargin) *NullableWidgetMargin {
	return &NullableWidgetMargin{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableWidgetMargin) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableWidgetMargin) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
