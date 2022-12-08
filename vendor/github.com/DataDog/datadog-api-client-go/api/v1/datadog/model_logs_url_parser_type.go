// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsURLParserType Type of logs URL parser.
type LogsURLParserType string

// List of LogsURLParserType.
const (
	LOGSURLPARSERTYPE_URL_PARSER LogsURLParserType = "url-parser"
)

var allowedLogsURLParserTypeEnumValues = []LogsURLParserType{
	LOGSURLPARSERTYPE_URL_PARSER,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsURLParserType) GetAllowedValues() []LogsURLParserType {
	return allowedLogsURLParserTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsURLParserType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsURLParserType(value)
	return nil
}

// NewLogsURLParserTypeFromValue returns a pointer to a valid LogsURLParserType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsURLParserTypeFromValue(v string) (*LogsURLParserType, error) {
	ev := LogsURLParserType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsURLParserType: valid values are %v", v, allowedLogsURLParserTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsURLParserType) IsValid() bool {
	for _, existing := range allowedLogsURLParserTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsURLParserType value.
func (v LogsURLParserType) Ptr() *LogsURLParserType {
	return &v
}

// NullableLogsURLParserType handles when a null is used for LogsURLParserType.
type NullableLogsURLParserType struct {
	value *LogsURLParserType
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsURLParserType) Get() *LogsURLParserType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsURLParserType) Set(val *LogsURLParserType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsURLParserType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsURLParserType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsURLParserType initializes the struct as if Set has been called.
func NewNullableLogsURLParserType(val *LogsURLParserType) *NullableLogsURLParserType {
	return &NullableLogsURLParserType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsURLParserType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsURLParserType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
