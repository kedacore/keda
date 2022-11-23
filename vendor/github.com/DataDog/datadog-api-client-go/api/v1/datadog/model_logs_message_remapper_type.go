// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsMessageRemapperType Type of logs message remapper.
type LogsMessageRemapperType string

// List of LogsMessageRemapperType.
const (
	LOGSMESSAGEREMAPPERTYPE_MESSAGE_REMAPPER LogsMessageRemapperType = "message-remapper"
)

var allowedLogsMessageRemapperTypeEnumValues = []LogsMessageRemapperType{
	LOGSMESSAGEREMAPPERTYPE_MESSAGE_REMAPPER,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsMessageRemapperType) GetAllowedValues() []LogsMessageRemapperType {
	return allowedLogsMessageRemapperTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsMessageRemapperType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsMessageRemapperType(value)
	return nil
}

// NewLogsMessageRemapperTypeFromValue returns a pointer to a valid LogsMessageRemapperType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsMessageRemapperTypeFromValue(v string) (*LogsMessageRemapperType, error) {
	ev := LogsMessageRemapperType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsMessageRemapperType: valid values are %v", v, allowedLogsMessageRemapperTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsMessageRemapperType) IsValid() bool {
	for _, existing := range allowedLogsMessageRemapperTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsMessageRemapperType value.
func (v LogsMessageRemapperType) Ptr() *LogsMessageRemapperType {
	return &v
}

// NullableLogsMessageRemapperType handles when a null is used for LogsMessageRemapperType.
type NullableLogsMessageRemapperType struct {
	value *LogsMessageRemapperType
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsMessageRemapperType) Get() *LogsMessageRemapperType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsMessageRemapperType) Set(val *LogsMessageRemapperType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsMessageRemapperType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsMessageRemapperType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsMessageRemapperType initializes the struct as if Set has been called.
func NewNullableLogsMessageRemapperType(val *LogsMessageRemapperType) *NullableLogsMessageRemapperType {
	return &NullableLogsMessageRemapperType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsMessageRemapperType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsMessageRemapperType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
