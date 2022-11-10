// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SLOType The type of the service level objective.
type SLOType string

// List of SLOType.
const (
	SLOTYPE_METRIC  SLOType = "metric"
	SLOTYPE_MONITOR SLOType = "monitor"
)

var allowedSLOTypeEnumValues = []SLOType{
	SLOTYPE_METRIC,
	SLOTYPE_MONITOR,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SLOType) GetAllowedValues() []SLOType {
	return allowedSLOTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SLOType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SLOType(value)
	return nil
}

// NewSLOTypeFromValue returns a pointer to a valid SLOType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSLOTypeFromValue(v string) (*SLOType, error) {
	ev := SLOType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SLOType: valid values are %v", v, allowedSLOTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SLOType) IsValid() bool {
	for _, existing := range allowedSLOTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SLOType value.
func (v SLOType) Ptr() *SLOType {
	return &v
}

// NullableSLOType handles when a null is used for SLOType.
type NullableSLOType struct {
	value *SLOType
	isSet bool
}

// Get returns the associated value.
func (v NullableSLOType) Get() *SLOType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSLOType) Set(val *SLOType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSLOType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSLOType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSLOType initializes the struct as if Set has been called.
func NewNullableSLOType(val *SLOType) *NullableSLOType {
	return &NullableSLOType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSLOType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSLOType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
