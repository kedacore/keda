// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsSort Time-ascending `asc` or time-descending `desc` results.
type LogsSort string

// List of LogsSort.
const (
	LOGSSORT_TIME_ASCENDING  LogsSort = "asc"
	LOGSSORT_TIME_DESCENDING LogsSort = "desc"
)

var allowedLogsSortEnumValues = []LogsSort{
	LOGSSORT_TIME_ASCENDING,
	LOGSSORT_TIME_DESCENDING,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsSort) GetAllowedValues() []LogsSort {
	return allowedLogsSortEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsSort) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsSort(value)
	return nil
}

// NewLogsSortFromValue returns a pointer to a valid LogsSort
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsSortFromValue(v string) (*LogsSort, error) {
	ev := LogsSort(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsSort: valid values are %v", v, allowedLogsSortEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsSort) IsValid() bool {
	for _, existing := range allowedLogsSortEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsSort value.
func (v LogsSort) Ptr() *LogsSort {
	return &v
}

// NullableLogsSort handles when a null is used for LogsSort.
type NullableLogsSort struct {
	value *LogsSort
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsSort) Get() *LogsSort {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsSort) Set(val *LogsSort) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsSort) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsSort) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsSort initializes the struct as if Set has been called.
func NewNullableLogsSort(val *LogsSort) *NullableLogsSort {
	return &NullableLogsSort{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsSort) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsSort) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
