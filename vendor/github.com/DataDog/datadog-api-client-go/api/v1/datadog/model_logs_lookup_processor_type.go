// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsLookupProcessorType Type of logs lookup processor.
type LogsLookupProcessorType string

// List of LogsLookupProcessorType.
const (
	LOGSLOOKUPPROCESSORTYPE_LOOKUP_PROCESSOR LogsLookupProcessorType = "lookup-processor"
)

var allowedLogsLookupProcessorTypeEnumValues = []LogsLookupProcessorType{
	LOGSLOOKUPPROCESSORTYPE_LOOKUP_PROCESSOR,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsLookupProcessorType) GetAllowedValues() []LogsLookupProcessorType {
	return allowedLogsLookupProcessorTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsLookupProcessorType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsLookupProcessorType(value)
	return nil
}

// NewLogsLookupProcessorTypeFromValue returns a pointer to a valid LogsLookupProcessorType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsLookupProcessorTypeFromValue(v string) (*LogsLookupProcessorType, error) {
	ev := LogsLookupProcessorType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsLookupProcessorType: valid values are %v", v, allowedLogsLookupProcessorTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsLookupProcessorType) IsValid() bool {
	for _, existing := range allowedLogsLookupProcessorTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsLookupProcessorType value.
func (v LogsLookupProcessorType) Ptr() *LogsLookupProcessorType {
	return &v
}

// NullableLogsLookupProcessorType handles when a null is used for LogsLookupProcessorType.
type NullableLogsLookupProcessorType struct {
	value *LogsLookupProcessorType
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsLookupProcessorType) Get() *LogsLookupProcessorType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsLookupProcessorType) Set(val *LogsLookupProcessorType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsLookupProcessorType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsLookupProcessorType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsLookupProcessorType initializes the struct as if Set has been called.
func NewNullableLogsLookupProcessorType(val *LogsLookupProcessorType) *NullableLogsLookupProcessorType {
	return &NullableLogsLookupProcessorType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsLookupProcessorType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsLookupProcessorType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
