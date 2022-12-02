// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// UsageSort The field to sort by.
type UsageSort string

// List of UsageSort.
const (
	USAGESORT_COMPUTED_ON UsageSort = "computed_on"
	USAGESORT_SIZE        UsageSort = "size"
	USAGESORT_START_DATE  UsageSort = "start_date"
	USAGESORT_END_DATE    UsageSort = "end_date"
)

var allowedUsageSortEnumValues = []UsageSort{
	USAGESORT_COMPUTED_ON,
	USAGESORT_SIZE,
	USAGESORT_START_DATE,
	USAGESORT_END_DATE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *UsageSort) GetAllowedValues() []UsageSort {
	return allowedUsageSortEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *UsageSort) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = UsageSort(value)
	return nil
}

// NewUsageSortFromValue returns a pointer to a valid UsageSort
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewUsageSortFromValue(v string) (*UsageSort, error) {
	ev := UsageSort(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for UsageSort: valid values are %v", v, allowedUsageSortEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v UsageSort) IsValid() bool {
	for _, existing := range allowedUsageSortEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to UsageSort value.
func (v UsageSort) Ptr() *UsageSort {
	return &v
}

// NullableUsageSort handles when a null is used for UsageSort.
type NullableUsageSort struct {
	value *UsageSort
	isSet bool
}

// Get returns the associated value.
func (v NullableUsageSort) Get() *UsageSort {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableUsageSort) Set(val *UsageSort) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableUsageSort) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableUsageSort) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableUsageSort initializes the struct as if Set has been called.
func NewNullableUsageSort(val *UsageSort) *NullableUsageSort {
	return &NullableUsageSort{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableUsageSort) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableUsageSort) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
