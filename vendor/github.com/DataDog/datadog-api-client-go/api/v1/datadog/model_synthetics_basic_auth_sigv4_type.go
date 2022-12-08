// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBasicAuthSigv4Type The type of authentication to use when performing the test.
type SyntheticsBasicAuthSigv4Type string

// List of SyntheticsBasicAuthSigv4Type.
const (
	SYNTHETICSBASICAUTHSIGV4TYPE_SIGV4 SyntheticsBasicAuthSigv4Type = "sigv4"
)

var allowedSyntheticsBasicAuthSigv4TypeEnumValues = []SyntheticsBasicAuthSigv4Type{
	SYNTHETICSBASICAUTHSIGV4TYPE_SIGV4,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsBasicAuthSigv4Type) GetAllowedValues() []SyntheticsBasicAuthSigv4Type {
	return allowedSyntheticsBasicAuthSigv4TypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsBasicAuthSigv4Type) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsBasicAuthSigv4Type(value)
	return nil
}

// NewSyntheticsBasicAuthSigv4TypeFromValue returns a pointer to a valid SyntheticsBasicAuthSigv4Type
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsBasicAuthSigv4TypeFromValue(v string) (*SyntheticsBasicAuthSigv4Type, error) {
	ev := SyntheticsBasicAuthSigv4Type(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsBasicAuthSigv4Type: valid values are %v", v, allowedSyntheticsBasicAuthSigv4TypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsBasicAuthSigv4Type) IsValid() bool {
	for _, existing := range allowedSyntheticsBasicAuthSigv4TypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsBasicAuthSigv4Type value.
func (v SyntheticsBasicAuthSigv4Type) Ptr() *SyntheticsBasicAuthSigv4Type {
	return &v
}

// NullableSyntheticsBasicAuthSigv4Type handles when a null is used for SyntheticsBasicAuthSigv4Type.
type NullableSyntheticsBasicAuthSigv4Type struct {
	value *SyntheticsBasicAuthSigv4Type
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsBasicAuthSigv4Type) Get() *SyntheticsBasicAuthSigv4Type {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsBasicAuthSigv4Type) Set(val *SyntheticsBasicAuthSigv4Type) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsBasicAuthSigv4Type) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsBasicAuthSigv4Type) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsBasicAuthSigv4Type initializes the struct as if Set has been called.
func NewNullableSyntheticsBasicAuthSigv4Type(val *SyntheticsBasicAuthSigv4Type) *NullableSyntheticsBasicAuthSigv4Type {
	return &NullableSyntheticsBasicAuthSigv4Type{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsBasicAuthSigv4Type) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsBasicAuthSigv4Type) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
