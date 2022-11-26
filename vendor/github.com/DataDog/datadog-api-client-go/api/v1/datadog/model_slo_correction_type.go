// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SLOCorrectionType SLO correction resource type.
type SLOCorrectionType string

// List of SLOCorrectionType.
const (
	SLOCORRECTIONTYPE_CORRECTION SLOCorrectionType = "correction"
)

var allowedSLOCorrectionTypeEnumValues = []SLOCorrectionType{
	SLOCORRECTIONTYPE_CORRECTION,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SLOCorrectionType) GetAllowedValues() []SLOCorrectionType {
	return allowedSLOCorrectionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SLOCorrectionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SLOCorrectionType(value)
	return nil
}

// NewSLOCorrectionTypeFromValue returns a pointer to a valid SLOCorrectionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSLOCorrectionTypeFromValue(v string) (*SLOCorrectionType, error) {
	ev := SLOCorrectionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SLOCorrectionType: valid values are %v", v, allowedSLOCorrectionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SLOCorrectionType) IsValid() bool {
	for _, existing := range allowedSLOCorrectionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SLOCorrectionType value.
func (v SLOCorrectionType) Ptr() *SLOCorrectionType {
	return &v
}

// NullableSLOCorrectionType handles when a null is used for SLOCorrectionType.
type NullableSLOCorrectionType struct {
	value *SLOCorrectionType
	isSet bool
}

// Get returns the associated value.
func (v NullableSLOCorrectionType) Get() *SLOCorrectionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSLOCorrectionType) Set(val *SLOCorrectionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSLOCorrectionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSLOCorrectionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSLOCorrectionType initializes the struct as if Set has been called.
func NewNullableSLOCorrectionType(val *SLOCorrectionType) *NullableSLOCorrectionType {
	return &NullableSLOCorrectionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSLOCorrectionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSLOCorrectionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
