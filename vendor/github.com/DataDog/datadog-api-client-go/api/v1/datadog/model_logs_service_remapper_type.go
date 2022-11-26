// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsServiceRemapperType Type of logs service remapper.
type LogsServiceRemapperType string

// List of LogsServiceRemapperType.
const (
	LOGSSERVICEREMAPPERTYPE_SERVICE_REMAPPER LogsServiceRemapperType = "service-remapper"
)

var allowedLogsServiceRemapperTypeEnumValues = []LogsServiceRemapperType{
	LOGSSERVICEREMAPPERTYPE_SERVICE_REMAPPER,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsServiceRemapperType) GetAllowedValues() []LogsServiceRemapperType {
	return allowedLogsServiceRemapperTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsServiceRemapperType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsServiceRemapperType(value)
	return nil
}

// NewLogsServiceRemapperTypeFromValue returns a pointer to a valid LogsServiceRemapperType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsServiceRemapperTypeFromValue(v string) (*LogsServiceRemapperType, error) {
	ev := LogsServiceRemapperType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsServiceRemapperType: valid values are %v", v, allowedLogsServiceRemapperTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsServiceRemapperType) IsValid() bool {
	for _, existing := range allowedLogsServiceRemapperTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsServiceRemapperType value.
func (v LogsServiceRemapperType) Ptr() *LogsServiceRemapperType {
	return &v
}

// NullableLogsServiceRemapperType handles when a null is used for LogsServiceRemapperType.
type NullableLogsServiceRemapperType struct {
	value *LogsServiceRemapperType
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsServiceRemapperType) Get() *LogsServiceRemapperType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsServiceRemapperType) Set(val *LogsServiceRemapperType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsServiceRemapperType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsServiceRemapperType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsServiceRemapperType initializes the struct as if Set has been called.
func NewNullableLogsServiceRemapperType(val *LogsServiceRemapperType) *NullableLogsServiceRemapperType {
	return &NullableLogsServiceRemapperType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsServiceRemapperType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsServiceRemapperType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
