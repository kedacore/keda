// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsGeoIPParserType Type of GeoIP parser.
type LogsGeoIPParserType string

// List of LogsGeoIPParserType.
const (
	LOGSGEOIPPARSERTYPE_GEO_IP_PARSER LogsGeoIPParserType = "geo-ip-parser"
)

var allowedLogsGeoIPParserTypeEnumValues = []LogsGeoIPParserType{
	LOGSGEOIPPARSERTYPE_GEO_IP_PARSER,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsGeoIPParserType) GetAllowedValues() []LogsGeoIPParserType {
	return allowedLogsGeoIPParserTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsGeoIPParserType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsGeoIPParserType(value)
	return nil
}

// NewLogsGeoIPParserTypeFromValue returns a pointer to a valid LogsGeoIPParserType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsGeoIPParserTypeFromValue(v string) (*LogsGeoIPParserType, error) {
	ev := LogsGeoIPParserType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsGeoIPParserType: valid values are %v", v, allowedLogsGeoIPParserTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsGeoIPParserType) IsValid() bool {
	for _, existing := range allowedLogsGeoIPParserTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsGeoIPParserType value.
func (v LogsGeoIPParserType) Ptr() *LogsGeoIPParserType {
	return &v
}

// NullableLogsGeoIPParserType handles when a null is used for LogsGeoIPParserType.
type NullableLogsGeoIPParserType struct {
	value *LogsGeoIPParserType
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsGeoIPParserType) Get() *LogsGeoIPParserType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsGeoIPParserType) Set(val *LogsGeoIPParserType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsGeoIPParserType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsGeoIPParserType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsGeoIPParserType initializes the struct as if Set has been called.
func NewNullableLogsGeoIPParserType(val *LogsGeoIPParserType) *NullableLogsGeoIPParserType {
	return &NullableLogsGeoIPParserType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsGeoIPParserType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsGeoIPParserType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
