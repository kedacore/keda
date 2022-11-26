// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsDateRemapperType Type of logs date remapper.
type LogsDateRemapperType string

// List of LogsDateRemapperType.
const (
	LOGSDATEREMAPPERTYPE_DATE_REMAPPER LogsDateRemapperType = "date-remapper"
)

var allowedLogsDateRemapperTypeEnumValues = []LogsDateRemapperType{
	LOGSDATEREMAPPERTYPE_DATE_REMAPPER,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsDateRemapperType) GetAllowedValues() []LogsDateRemapperType {
	return allowedLogsDateRemapperTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsDateRemapperType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsDateRemapperType(value)
	return nil
}

// NewLogsDateRemapperTypeFromValue returns a pointer to a valid LogsDateRemapperType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsDateRemapperTypeFromValue(v string) (*LogsDateRemapperType, error) {
	ev := LogsDateRemapperType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsDateRemapperType: valid values are %v", v, allowedLogsDateRemapperTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsDateRemapperType) IsValid() bool {
	for _, existing := range allowedLogsDateRemapperTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsDateRemapperType value.
func (v LogsDateRemapperType) Ptr() *LogsDateRemapperType {
	return &v
}

// NullableLogsDateRemapperType handles when a null is used for LogsDateRemapperType.
type NullableLogsDateRemapperType struct {
	value *LogsDateRemapperType
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsDateRemapperType) Get() *LogsDateRemapperType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsDateRemapperType) Set(val *LogsDateRemapperType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsDateRemapperType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsDateRemapperType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsDateRemapperType initializes the struct as if Set has been called.
func NewNullableLogsDateRemapperType(val *LogsDateRemapperType) *NullableLogsDateRemapperType {
	return &NullableLogsDateRemapperType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsDateRemapperType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsDateRemapperType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
