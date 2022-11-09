// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsGlobalVariableParserType Type of parser for a Synthetics global variable from a synthetics test.
type SyntheticsGlobalVariableParserType string

// List of SyntheticsGlobalVariableParserType.
const (
	SYNTHETICSGLOBALVARIABLEPARSERTYPE_RAW       SyntheticsGlobalVariableParserType = "raw"
	SYNTHETICSGLOBALVARIABLEPARSERTYPE_JSON_PATH SyntheticsGlobalVariableParserType = "json_path"
	SYNTHETICSGLOBALVARIABLEPARSERTYPE_REGEX     SyntheticsGlobalVariableParserType = "regex"
	SYNTHETICSGLOBALVARIABLEPARSERTYPE_X_PATH    SyntheticsGlobalVariableParserType = "x_path"
)

var allowedSyntheticsGlobalVariableParserTypeEnumValues = []SyntheticsGlobalVariableParserType{
	SYNTHETICSGLOBALVARIABLEPARSERTYPE_RAW,
	SYNTHETICSGLOBALVARIABLEPARSERTYPE_JSON_PATH,
	SYNTHETICSGLOBALVARIABLEPARSERTYPE_REGEX,
	SYNTHETICSGLOBALVARIABLEPARSERTYPE_X_PATH,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsGlobalVariableParserType) GetAllowedValues() []SyntheticsGlobalVariableParserType {
	return allowedSyntheticsGlobalVariableParserTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsGlobalVariableParserType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsGlobalVariableParserType(value)
	return nil
}

// NewSyntheticsGlobalVariableParserTypeFromValue returns a pointer to a valid SyntheticsGlobalVariableParserType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsGlobalVariableParserTypeFromValue(v string) (*SyntheticsGlobalVariableParserType, error) {
	ev := SyntheticsGlobalVariableParserType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsGlobalVariableParserType: valid values are %v", v, allowedSyntheticsGlobalVariableParserTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsGlobalVariableParserType) IsValid() bool {
	for _, existing := range allowedSyntheticsGlobalVariableParserTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsGlobalVariableParserType value.
func (v SyntheticsGlobalVariableParserType) Ptr() *SyntheticsGlobalVariableParserType {
	return &v
}

// NullableSyntheticsGlobalVariableParserType handles when a null is used for SyntheticsGlobalVariableParserType.
type NullableSyntheticsGlobalVariableParserType struct {
	value *SyntheticsGlobalVariableParserType
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsGlobalVariableParserType) Get() *SyntheticsGlobalVariableParserType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsGlobalVariableParserType) Set(val *SyntheticsGlobalVariableParserType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsGlobalVariableParserType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsGlobalVariableParserType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsGlobalVariableParserType initializes the struct as if Set has been called.
func NewNullableSyntheticsGlobalVariableParserType(val *SyntheticsGlobalVariableParserType) *NullableSyntheticsGlobalVariableParserType {
	return &NullableSyntheticsGlobalVariableParserType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsGlobalVariableParserType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsGlobalVariableParserType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
