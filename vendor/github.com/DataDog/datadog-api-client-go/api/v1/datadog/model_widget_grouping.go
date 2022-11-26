// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetGrouping The kind of grouping to use.
type WidgetGrouping string

// List of WidgetGrouping.
const (
	WIDGETGROUPING_CHECK   WidgetGrouping = "check"
	WIDGETGROUPING_CLUSTER WidgetGrouping = "cluster"
)

var allowedWidgetGroupingEnumValues = []WidgetGrouping{
	WIDGETGROUPING_CHECK,
	WIDGETGROUPING_CLUSTER,
}

// GetAllowedValues reeturns the list of possible values.
func (v *WidgetGrouping) GetAllowedValues() []WidgetGrouping {
	return allowedWidgetGroupingEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *WidgetGrouping) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = WidgetGrouping(value)
	return nil
}

// NewWidgetGroupingFromValue returns a pointer to a valid WidgetGrouping
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewWidgetGroupingFromValue(v string) (*WidgetGrouping, error) {
	ev := WidgetGrouping(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for WidgetGrouping: valid values are %v", v, allowedWidgetGroupingEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v WidgetGrouping) IsValid() bool {
	for _, existing := range allowedWidgetGroupingEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to WidgetGrouping value.
func (v WidgetGrouping) Ptr() *WidgetGrouping {
	return &v
}

// NullableWidgetGrouping handles when a null is used for WidgetGrouping.
type NullableWidgetGrouping struct {
	value *WidgetGrouping
	isSet bool
}

// Get returns the associated value.
func (v NullableWidgetGrouping) Get() *WidgetGrouping {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableWidgetGrouping) Set(val *WidgetGrouping) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableWidgetGrouping) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableWidgetGrouping) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableWidgetGrouping initializes the struct as if Set has been called.
func NewNullableWidgetGrouping(val *WidgetGrouping) *NullableWidgetGrouping {
	return &NullableWidgetGrouping{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableWidgetGrouping) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableWidgetGrouping) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
