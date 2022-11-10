// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsTraceRemapperType Type of logs trace remapper.
type LogsTraceRemapperType string

// List of LogsTraceRemapperType.
const (
	LOGSTRACEREMAPPERTYPE_TRACE_ID_REMAPPER LogsTraceRemapperType = "trace-id-remapper"
)

var allowedLogsTraceRemapperTypeEnumValues = []LogsTraceRemapperType{
	LOGSTRACEREMAPPERTYPE_TRACE_ID_REMAPPER,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsTraceRemapperType) GetAllowedValues() []LogsTraceRemapperType {
	return allowedLogsTraceRemapperTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsTraceRemapperType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsTraceRemapperType(value)
	return nil
}

// NewLogsTraceRemapperTypeFromValue returns a pointer to a valid LogsTraceRemapperType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsTraceRemapperTypeFromValue(v string) (*LogsTraceRemapperType, error) {
	ev := LogsTraceRemapperType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsTraceRemapperType: valid values are %v", v, allowedLogsTraceRemapperTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsTraceRemapperType) IsValid() bool {
	for _, existing := range allowedLogsTraceRemapperTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsTraceRemapperType value.
func (v LogsTraceRemapperType) Ptr() *LogsTraceRemapperType {
	return &v
}

// NullableLogsTraceRemapperType handles when a null is used for LogsTraceRemapperType.
type NullableLogsTraceRemapperType struct {
	value *LogsTraceRemapperType
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsTraceRemapperType) Get() *LogsTraceRemapperType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsTraceRemapperType) Set(val *LogsTraceRemapperType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsTraceRemapperType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsTraceRemapperType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsTraceRemapperType initializes the struct as if Set has been called.
func NewNullableLogsTraceRemapperType(val *LogsTraceRemapperType) *NullableLogsTraceRemapperType {
	return &NullableLogsTraceRemapperType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsTraceRemapperType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsTraceRemapperType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
