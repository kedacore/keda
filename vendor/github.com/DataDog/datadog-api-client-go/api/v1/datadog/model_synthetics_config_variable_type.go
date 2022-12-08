// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsConfigVariableType Type of the configuration variable.
type SyntheticsConfigVariableType string

// List of SyntheticsConfigVariableType.
const (
	SYNTHETICSCONFIGVARIABLETYPE_GLOBAL SyntheticsConfigVariableType = "global"
	SYNTHETICSCONFIGVARIABLETYPE_TEXT   SyntheticsConfigVariableType = "text"
)

var allowedSyntheticsConfigVariableTypeEnumValues = []SyntheticsConfigVariableType{
	SYNTHETICSCONFIGVARIABLETYPE_GLOBAL,
	SYNTHETICSCONFIGVARIABLETYPE_TEXT,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsConfigVariableType) GetAllowedValues() []SyntheticsConfigVariableType {
	return allowedSyntheticsConfigVariableTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsConfigVariableType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsConfigVariableType(value)
	return nil
}

// NewSyntheticsConfigVariableTypeFromValue returns a pointer to a valid SyntheticsConfigVariableType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsConfigVariableTypeFromValue(v string) (*SyntheticsConfigVariableType, error) {
	ev := SyntheticsConfigVariableType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsConfigVariableType: valid values are %v", v, allowedSyntheticsConfigVariableTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsConfigVariableType) IsValid() bool {
	for _, existing := range allowedSyntheticsConfigVariableTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsConfigVariableType value.
func (v SyntheticsConfigVariableType) Ptr() *SyntheticsConfigVariableType {
	return &v
}

// NullableSyntheticsConfigVariableType handles when a null is used for SyntheticsConfigVariableType.
type NullableSyntheticsConfigVariableType struct {
	value *SyntheticsConfigVariableType
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsConfigVariableType) Get() *SyntheticsConfigVariableType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsConfigVariableType) Set(val *SyntheticsConfigVariableType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsConfigVariableType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsConfigVariableType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsConfigVariableType initializes the struct as if Set has been called.
func NewNullableSyntheticsConfigVariableType(val *SyntheticsConfigVariableType) *NullableSyntheticsConfigVariableType {
	return &NullableSyntheticsConfigVariableType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsConfigVariableType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsConfigVariableType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
