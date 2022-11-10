// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// DistributionPointsType The type of the distribution point.
type DistributionPointsType string

// List of DistributionPointsType.
const (
	DISTRIBUTIONPOINTSTYPE_DISTRIBUTION DistributionPointsType = "distribution"
)

var allowedDistributionPointsTypeEnumValues = []DistributionPointsType{
	DISTRIBUTIONPOINTSTYPE_DISTRIBUTION,
}

// GetAllowedValues reeturns the list of possible values.
func (v *DistributionPointsType) GetAllowedValues() []DistributionPointsType {
	return allowedDistributionPointsTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *DistributionPointsType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = DistributionPointsType(value)
	return nil
}

// NewDistributionPointsTypeFromValue returns a pointer to a valid DistributionPointsType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewDistributionPointsTypeFromValue(v string) (*DistributionPointsType, error) {
	ev := DistributionPointsType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for DistributionPointsType: valid values are %v", v, allowedDistributionPointsTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v DistributionPointsType) IsValid() bool {
	for _, existing := range allowedDistributionPointsTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to DistributionPointsType value.
func (v DistributionPointsType) Ptr() *DistributionPointsType {
	return &v
}

// NullableDistributionPointsType handles when a null is used for DistributionPointsType.
type NullableDistributionPointsType struct {
	value *DistributionPointsType
	isSet bool
}

// Get returns the associated value.
func (v NullableDistributionPointsType) Get() *DistributionPointsType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableDistributionPointsType) Set(val *DistributionPointsType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableDistributionPointsType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableDistributionPointsType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableDistributionPointsType initializes the struct as if Set has been called.
func NewNullableDistributionPointsType(val *DistributionPointsType) *NullableDistributionPointsType {
	return &NullableDistributionPointsType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableDistributionPointsType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableDistributionPointsType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
