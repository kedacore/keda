// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsTestDetailsType Type of the Synthetic test, either `api` or `browser`.
type SyntheticsTestDetailsType string

// List of SyntheticsTestDetailsType.
const (
	SYNTHETICSTESTDETAILSTYPE_API     SyntheticsTestDetailsType = "api"
	SYNTHETICSTESTDETAILSTYPE_BROWSER SyntheticsTestDetailsType = "browser"
)

var allowedSyntheticsTestDetailsTypeEnumValues = []SyntheticsTestDetailsType{
	SYNTHETICSTESTDETAILSTYPE_API,
	SYNTHETICSTESTDETAILSTYPE_BROWSER,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsTestDetailsType) GetAllowedValues() []SyntheticsTestDetailsType {
	return allowedSyntheticsTestDetailsTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsTestDetailsType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsTestDetailsType(value)
	return nil
}

// NewSyntheticsTestDetailsTypeFromValue returns a pointer to a valid SyntheticsTestDetailsType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsTestDetailsTypeFromValue(v string) (*SyntheticsTestDetailsType, error) {
	ev := SyntheticsTestDetailsType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsTestDetailsType: valid values are %v", v, allowedSyntheticsTestDetailsTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsTestDetailsType) IsValid() bool {
	for _, existing := range allowedSyntheticsTestDetailsTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsTestDetailsType value.
func (v SyntheticsTestDetailsType) Ptr() *SyntheticsTestDetailsType {
	return &v
}

// NullableSyntheticsTestDetailsType handles when a null is used for SyntheticsTestDetailsType.
type NullableSyntheticsTestDetailsType struct {
	value *SyntheticsTestDetailsType
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsTestDetailsType) Get() *SyntheticsTestDetailsType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsTestDetailsType) Set(val *SyntheticsTestDetailsType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsTestDetailsType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsTestDetailsType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsTestDetailsType initializes the struct as if Set has been called.
func NewNullableSyntheticsTestDetailsType(val *SyntheticsTestDetailsType) *NullableSyntheticsTestDetailsType {
	return &NullableSyntheticsTestDetailsType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsTestDetailsType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsTestDetailsType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
