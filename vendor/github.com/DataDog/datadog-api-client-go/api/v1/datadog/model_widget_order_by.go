// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetOrderBy What to order by.
type WidgetOrderBy string

// List of WidgetOrderBy.
const (
	WIDGETORDERBY_CHANGE  WidgetOrderBy = "change"
	WIDGETORDERBY_NAME    WidgetOrderBy = "name"
	WIDGETORDERBY_PRESENT WidgetOrderBy = "present"
	WIDGETORDERBY_PAST    WidgetOrderBy = "past"
)

var allowedWidgetOrderByEnumValues = []WidgetOrderBy{
	WIDGETORDERBY_CHANGE,
	WIDGETORDERBY_NAME,
	WIDGETORDERBY_PRESENT,
	WIDGETORDERBY_PAST,
}

// GetAllowedValues reeturns the list of possible values.
func (v *WidgetOrderBy) GetAllowedValues() []WidgetOrderBy {
	return allowedWidgetOrderByEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *WidgetOrderBy) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = WidgetOrderBy(value)
	return nil
}

// NewWidgetOrderByFromValue returns a pointer to a valid WidgetOrderBy
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewWidgetOrderByFromValue(v string) (*WidgetOrderBy, error) {
	ev := WidgetOrderBy(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for WidgetOrderBy: valid values are %v", v, allowedWidgetOrderByEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v WidgetOrderBy) IsValid() bool {
	for _, existing := range allowedWidgetOrderByEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to WidgetOrderBy value.
func (v WidgetOrderBy) Ptr() *WidgetOrderBy {
	return &v
}

// NullableWidgetOrderBy handles when a null is used for WidgetOrderBy.
type NullableWidgetOrderBy struct {
	value *WidgetOrderBy
	isSet bool
}

// Get returns the associated value.
func (v NullableWidgetOrderBy) Get() *WidgetOrderBy {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableWidgetOrderBy) Set(val *WidgetOrderBy) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableWidgetOrderBy) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableWidgetOrderBy) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableWidgetOrderBy initializes the struct as if Set has been called.
func NewNullableWidgetOrderBy(val *WidgetOrderBy) *NullableWidgetOrderBy {
	return &NullableWidgetOrderBy{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableWidgetOrderBy) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableWidgetOrderBy) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
