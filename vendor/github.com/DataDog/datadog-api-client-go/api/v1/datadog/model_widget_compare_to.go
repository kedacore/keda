// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetCompareTo Timeframe used for the change comparison.
type WidgetCompareTo string

// List of WidgetCompareTo.
const (
	WIDGETCOMPARETO_HOUR_BEFORE  WidgetCompareTo = "hour_before"
	WIDGETCOMPARETO_DAY_BEFORE   WidgetCompareTo = "day_before"
	WIDGETCOMPARETO_WEEK_BEFORE  WidgetCompareTo = "week_before"
	WIDGETCOMPARETO_MONTH_BEFORE WidgetCompareTo = "month_before"
)

var allowedWidgetCompareToEnumValues = []WidgetCompareTo{
	WIDGETCOMPARETO_HOUR_BEFORE,
	WIDGETCOMPARETO_DAY_BEFORE,
	WIDGETCOMPARETO_WEEK_BEFORE,
	WIDGETCOMPARETO_MONTH_BEFORE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *WidgetCompareTo) GetAllowedValues() []WidgetCompareTo {
	return allowedWidgetCompareToEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *WidgetCompareTo) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = WidgetCompareTo(value)
	return nil
}

// NewWidgetCompareToFromValue returns a pointer to a valid WidgetCompareTo
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewWidgetCompareToFromValue(v string) (*WidgetCompareTo, error) {
	ev := WidgetCompareTo(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for WidgetCompareTo: valid values are %v", v, allowedWidgetCompareToEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v WidgetCompareTo) IsValid() bool {
	for _, existing := range allowedWidgetCompareToEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to WidgetCompareTo value.
func (v WidgetCompareTo) Ptr() *WidgetCompareTo {
	return &v
}

// NullableWidgetCompareTo handles when a null is used for WidgetCompareTo.
type NullableWidgetCompareTo struct {
	value *WidgetCompareTo
	isSet bool
}

// Get returns the associated value.
func (v NullableWidgetCompareTo) Get() *WidgetCompareTo {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableWidgetCompareTo) Set(val *WidgetCompareTo) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableWidgetCompareTo) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableWidgetCompareTo) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableWidgetCompareTo initializes the struct as if Set has been called.
func NewNullableWidgetCompareTo(val *WidgetCompareTo) *NullableWidgetCompareTo {
	return &NullableWidgetCompareTo{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableWidgetCompareTo) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableWidgetCompareTo) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
