// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBasicAuthWebType The type of basic authentication to use when performing the test.
type SyntheticsBasicAuthWebType string

// List of SyntheticsBasicAuthWebType.
const (
	SYNTHETICSBASICAUTHWEBTYPE_WEB SyntheticsBasicAuthWebType = "web"
)

var allowedSyntheticsBasicAuthWebTypeEnumValues = []SyntheticsBasicAuthWebType{
	SYNTHETICSBASICAUTHWEBTYPE_WEB,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsBasicAuthWebType) GetAllowedValues() []SyntheticsBasicAuthWebType {
	return allowedSyntheticsBasicAuthWebTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsBasicAuthWebType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsBasicAuthWebType(value)
	return nil
}

// NewSyntheticsBasicAuthWebTypeFromValue returns a pointer to a valid SyntheticsBasicAuthWebType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsBasicAuthWebTypeFromValue(v string) (*SyntheticsBasicAuthWebType, error) {
	ev := SyntheticsBasicAuthWebType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsBasicAuthWebType: valid values are %v", v, allowedSyntheticsBasicAuthWebTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsBasicAuthWebType) IsValid() bool {
	for _, existing := range allowedSyntheticsBasicAuthWebTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsBasicAuthWebType value.
func (v SyntheticsBasicAuthWebType) Ptr() *SyntheticsBasicAuthWebType {
	return &v
}

// NullableSyntheticsBasicAuthWebType handles when a null is used for SyntheticsBasicAuthWebType.
type NullableSyntheticsBasicAuthWebType struct {
	value *SyntheticsBasicAuthWebType
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsBasicAuthWebType) Get() *SyntheticsBasicAuthWebType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsBasicAuthWebType) Set(val *SyntheticsBasicAuthWebType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsBasicAuthWebType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsBasicAuthWebType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsBasicAuthWebType initializes the struct as if Set has been called.
func NewNullableSyntheticsBasicAuthWebType(val *SyntheticsBasicAuthWebType) *NullableSyntheticsBasicAuthWebType {
	return &NullableSyntheticsBasicAuthWebType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsBasicAuthWebType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsBasicAuthWebType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
