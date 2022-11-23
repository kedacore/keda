// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetSort Widget sorting methods.
type WidgetSort string

// List of WidgetSort.
const (
	WIDGETSORT_ASCENDING  WidgetSort = "asc"
	WIDGETSORT_DESCENDING WidgetSort = "desc"
)

var allowedWidgetSortEnumValues = []WidgetSort{
	WIDGETSORT_ASCENDING,
	WIDGETSORT_DESCENDING,
}

// GetAllowedValues reeturns the list of possible values.
func (v *WidgetSort) GetAllowedValues() []WidgetSort {
	return allowedWidgetSortEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *WidgetSort) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = WidgetSort(value)
	return nil
}

// NewWidgetSortFromValue returns a pointer to a valid WidgetSort
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewWidgetSortFromValue(v string) (*WidgetSort, error) {
	ev := WidgetSort(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for WidgetSort: valid values are %v", v, allowedWidgetSortEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v WidgetSort) IsValid() bool {
	for _, existing := range allowedWidgetSortEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to WidgetSort value.
func (v WidgetSort) Ptr() *WidgetSort {
	return &v
}

// NullableWidgetSort handles when a null is used for WidgetSort.
type NullableWidgetSort struct {
	value *WidgetSort
	isSet bool
}

// Get returns the associated value.
func (v NullableWidgetSort) Get() *WidgetSort {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableWidgetSort) Set(val *WidgetSort) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableWidgetSort) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableWidgetSort) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableWidgetSort initializes the struct as if Set has been called.
func NewNullableWidgetSort(val *WidgetSort) *NullableWidgetSort {
	return &NullableWidgetSort{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableWidgetSort) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableWidgetSort) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
