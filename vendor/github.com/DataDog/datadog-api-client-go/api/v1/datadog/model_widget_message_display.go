// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetMessageDisplay Amount of log lines to display
type WidgetMessageDisplay string

// List of WidgetMessageDisplay.
const (
	WIDGETMESSAGEDISPLAY_INLINE          WidgetMessageDisplay = "inline"
	WIDGETMESSAGEDISPLAY_EXPANDED_MEDIUM WidgetMessageDisplay = "expanded-md"
	WIDGETMESSAGEDISPLAY_EXPANDED_LARGE  WidgetMessageDisplay = "expanded-lg"
)

var allowedWidgetMessageDisplayEnumValues = []WidgetMessageDisplay{
	WIDGETMESSAGEDISPLAY_INLINE,
	WIDGETMESSAGEDISPLAY_EXPANDED_MEDIUM,
	WIDGETMESSAGEDISPLAY_EXPANDED_LARGE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *WidgetMessageDisplay) GetAllowedValues() []WidgetMessageDisplay {
	return allowedWidgetMessageDisplayEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *WidgetMessageDisplay) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = WidgetMessageDisplay(value)
	return nil
}

// NewWidgetMessageDisplayFromValue returns a pointer to a valid WidgetMessageDisplay
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewWidgetMessageDisplayFromValue(v string) (*WidgetMessageDisplay, error) {
	ev := WidgetMessageDisplay(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for WidgetMessageDisplay: valid values are %v", v, allowedWidgetMessageDisplayEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v WidgetMessageDisplay) IsValid() bool {
	for _, existing := range allowedWidgetMessageDisplayEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to WidgetMessageDisplay value.
func (v WidgetMessageDisplay) Ptr() *WidgetMessageDisplay {
	return &v
}

// NullableWidgetMessageDisplay handles when a null is used for WidgetMessageDisplay.
type NullableWidgetMessageDisplay struct {
	value *WidgetMessageDisplay
	isSet bool
}

// Get returns the associated value.
func (v NullableWidgetMessageDisplay) Get() *WidgetMessageDisplay {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableWidgetMessageDisplay) Set(val *WidgetMessageDisplay) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableWidgetMessageDisplay) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableWidgetMessageDisplay) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableWidgetMessageDisplay initializes the struct as if Set has been called.
func NewNullableWidgetMessageDisplay(val *WidgetMessageDisplay) *NullableWidgetMessageDisplay {
	return &NullableWidgetMessageDisplay{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableWidgetMessageDisplay) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableWidgetMessageDisplay) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
