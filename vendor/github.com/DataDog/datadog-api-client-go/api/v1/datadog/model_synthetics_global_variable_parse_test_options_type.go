// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsGlobalVariableParseTestOptionsType Property of the Synthetics Test Response to use for a Synthetics global variable.
type SyntheticsGlobalVariableParseTestOptionsType string

// List of SyntheticsGlobalVariableParseTestOptionsType.
const (
	SYNTHETICSGLOBALVARIABLEPARSETESTOPTIONSTYPE_HTTP_BODY   SyntheticsGlobalVariableParseTestOptionsType = "http_body"
	SYNTHETICSGLOBALVARIABLEPARSETESTOPTIONSTYPE_HTTP_HEADER SyntheticsGlobalVariableParseTestOptionsType = "http_header"
)

var allowedSyntheticsGlobalVariableParseTestOptionsTypeEnumValues = []SyntheticsGlobalVariableParseTestOptionsType{
	SYNTHETICSGLOBALVARIABLEPARSETESTOPTIONSTYPE_HTTP_BODY,
	SYNTHETICSGLOBALVARIABLEPARSETESTOPTIONSTYPE_HTTP_HEADER,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsGlobalVariableParseTestOptionsType) GetAllowedValues() []SyntheticsGlobalVariableParseTestOptionsType {
	return allowedSyntheticsGlobalVariableParseTestOptionsTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsGlobalVariableParseTestOptionsType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsGlobalVariableParseTestOptionsType(value)
	return nil
}

// NewSyntheticsGlobalVariableParseTestOptionsTypeFromValue returns a pointer to a valid SyntheticsGlobalVariableParseTestOptionsType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsGlobalVariableParseTestOptionsTypeFromValue(v string) (*SyntheticsGlobalVariableParseTestOptionsType, error) {
	ev := SyntheticsGlobalVariableParseTestOptionsType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsGlobalVariableParseTestOptionsType: valid values are %v", v, allowedSyntheticsGlobalVariableParseTestOptionsTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsGlobalVariableParseTestOptionsType) IsValid() bool {
	for _, existing := range allowedSyntheticsGlobalVariableParseTestOptionsTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsGlobalVariableParseTestOptionsType value.
func (v SyntheticsGlobalVariableParseTestOptionsType) Ptr() *SyntheticsGlobalVariableParseTestOptionsType {
	return &v
}

// NullableSyntheticsGlobalVariableParseTestOptionsType handles when a null is used for SyntheticsGlobalVariableParseTestOptionsType.
type NullableSyntheticsGlobalVariableParseTestOptionsType struct {
	value *SyntheticsGlobalVariableParseTestOptionsType
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsGlobalVariableParseTestOptionsType) Get() *SyntheticsGlobalVariableParseTestOptionsType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsGlobalVariableParseTestOptionsType) Set(val *SyntheticsGlobalVariableParseTestOptionsType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsGlobalVariableParseTestOptionsType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsGlobalVariableParseTestOptionsType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsGlobalVariableParseTestOptionsType initializes the struct as if Set has been called.
func NewNullableSyntheticsGlobalVariableParseTestOptionsType(val *SyntheticsGlobalVariableParseTestOptionsType) *NullableSyntheticsGlobalVariableParseTestOptionsType {
	return &NullableSyntheticsGlobalVariableParseTestOptionsType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsGlobalVariableParseTestOptionsType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsGlobalVariableParseTestOptionsType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
