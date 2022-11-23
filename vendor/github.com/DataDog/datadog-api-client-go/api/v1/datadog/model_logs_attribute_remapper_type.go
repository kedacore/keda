// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsAttributeRemapperType Type of logs attribute remapper.
type LogsAttributeRemapperType string

// List of LogsAttributeRemapperType.
const (
	LOGSATTRIBUTEREMAPPERTYPE_ATTRIBUTE_REMAPPER LogsAttributeRemapperType = "attribute-remapper"
)

var allowedLogsAttributeRemapperTypeEnumValues = []LogsAttributeRemapperType{
	LOGSATTRIBUTEREMAPPERTYPE_ATTRIBUTE_REMAPPER,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsAttributeRemapperType) GetAllowedValues() []LogsAttributeRemapperType {
	return allowedLogsAttributeRemapperTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsAttributeRemapperType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsAttributeRemapperType(value)
	return nil
}

// NewLogsAttributeRemapperTypeFromValue returns a pointer to a valid LogsAttributeRemapperType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsAttributeRemapperTypeFromValue(v string) (*LogsAttributeRemapperType, error) {
	ev := LogsAttributeRemapperType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsAttributeRemapperType: valid values are %v", v, allowedLogsAttributeRemapperTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsAttributeRemapperType) IsValid() bool {
	for _, existing := range allowedLogsAttributeRemapperTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsAttributeRemapperType value.
func (v LogsAttributeRemapperType) Ptr() *LogsAttributeRemapperType {
	return &v
}

// NullableLogsAttributeRemapperType handles when a null is used for LogsAttributeRemapperType.
type NullableLogsAttributeRemapperType struct {
	value *LogsAttributeRemapperType
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsAttributeRemapperType) Get() *LogsAttributeRemapperType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsAttributeRemapperType) Set(val *LogsAttributeRemapperType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsAttributeRemapperType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsAttributeRemapperType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsAttributeRemapperType initializes the struct as if Set has been called.
func NewNullableLogsAttributeRemapperType(val *LogsAttributeRemapperType) *NullableLogsAttributeRemapperType {
	return &NullableLogsAttributeRemapperType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsAttributeRemapperType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsAttributeRemapperType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
