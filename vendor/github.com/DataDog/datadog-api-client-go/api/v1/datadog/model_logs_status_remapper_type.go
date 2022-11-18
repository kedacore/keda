// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsStatusRemapperType Type of logs status remapper.
type LogsStatusRemapperType string

// List of LogsStatusRemapperType.
const (
	LOGSSTATUSREMAPPERTYPE_STATUS_REMAPPER LogsStatusRemapperType = "status-remapper"
)

var allowedLogsStatusRemapperTypeEnumValues = []LogsStatusRemapperType{
	LOGSSTATUSREMAPPERTYPE_STATUS_REMAPPER,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsStatusRemapperType) GetAllowedValues() []LogsStatusRemapperType {
	return allowedLogsStatusRemapperTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsStatusRemapperType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsStatusRemapperType(value)
	return nil
}

// NewLogsStatusRemapperTypeFromValue returns a pointer to a valid LogsStatusRemapperType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsStatusRemapperTypeFromValue(v string) (*LogsStatusRemapperType, error) {
	ev := LogsStatusRemapperType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsStatusRemapperType: valid values are %v", v, allowedLogsStatusRemapperTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsStatusRemapperType) IsValid() bool {
	for _, existing := range allowedLogsStatusRemapperTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsStatusRemapperType value.
func (v LogsStatusRemapperType) Ptr() *LogsStatusRemapperType {
	return &v
}

// NullableLogsStatusRemapperType handles when a null is used for LogsStatusRemapperType.
type NullableLogsStatusRemapperType struct {
	value *LogsStatusRemapperType
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsStatusRemapperType) Get() *LogsStatusRemapperType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsStatusRemapperType) Set(val *LogsStatusRemapperType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsStatusRemapperType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsStatusRemapperType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsStatusRemapperType initializes the struct as if Set has been called.
func NewNullableLogsStatusRemapperType(val *LogsStatusRemapperType) *NullableLogsStatusRemapperType {
	return &NullableLogsStatusRemapperType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsStatusRemapperType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsStatusRemapperType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
