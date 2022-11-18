// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// DistributionWidgetDefinitionType Type of the distribution widget.
type DistributionWidgetDefinitionType string

// List of DistributionWidgetDefinitionType.
const (
	DISTRIBUTIONWIDGETDEFINITIONTYPE_DISTRIBUTION DistributionWidgetDefinitionType = "distribution"
)

var allowedDistributionWidgetDefinitionTypeEnumValues = []DistributionWidgetDefinitionType{
	DISTRIBUTIONWIDGETDEFINITIONTYPE_DISTRIBUTION,
}

// GetAllowedValues reeturns the list of possible values.
func (v *DistributionWidgetDefinitionType) GetAllowedValues() []DistributionWidgetDefinitionType {
	return allowedDistributionWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *DistributionWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = DistributionWidgetDefinitionType(value)
	return nil
}

// NewDistributionWidgetDefinitionTypeFromValue returns a pointer to a valid DistributionWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewDistributionWidgetDefinitionTypeFromValue(v string) (*DistributionWidgetDefinitionType, error) {
	ev := DistributionWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for DistributionWidgetDefinitionType: valid values are %v", v, allowedDistributionWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v DistributionWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedDistributionWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to DistributionWidgetDefinitionType value.
func (v DistributionWidgetDefinitionType) Ptr() *DistributionWidgetDefinitionType {
	return &v
}

// NullableDistributionWidgetDefinitionType handles when a null is used for DistributionWidgetDefinitionType.
type NullableDistributionWidgetDefinitionType struct {
	value *DistributionWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableDistributionWidgetDefinitionType) Get() *DistributionWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableDistributionWidgetDefinitionType) Set(val *DistributionWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableDistributionWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableDistributionWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableDistributionWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableDistributionWidgetDefinitionType(val *DistributionWidgetDefinitionType) *NullableDistributionWidgetDefinitionType {
	return &NullableDistributionWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableDistributionWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableDistributionWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
