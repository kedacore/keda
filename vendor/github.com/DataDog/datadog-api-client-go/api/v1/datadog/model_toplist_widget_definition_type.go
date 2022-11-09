// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ToplistWidgetDefinitionType Type of the top list widget.
type ToplistWidgetDefinitionType string

// List of ToplistWidgetDefinitionType.
const (
	TOPLISTWIDGETDEFINITIONTYPE_TOPLIST ToplistWidgetDefinitionType = "toplist"
)

var allowedToplistWidgetDefinitionTypeEnumValues = []ToplistWidgetDefinitionType{
	TOPLISTWIDGETDEFINITIONTYPE_TOPLIST,
}

// GetAllowedValues reeturns the list of possible values.
func (v *ToplistWidgetDefinitionType) GetAllowedValues() []ToplistWidgetDefinitionType {
	return allowedToplistWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *ToplistWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = ToplistWidgetDefinitionType(value)
	return nil
}

// NewToplistWidgetDefinitionTypeFromValue returns a pointer to a valid ToplistWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewToplistWidgetDefinitionTypeFromValue(v string) (*ToplistWidgetDefinitionType, error) {
	ev := ToplistWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for ToplistWidgetDefinitionType: valid values are %v", v, allowedToplistWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v ToplistWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedToplistWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to ToplistWidgetDefinitionType value.
func (v ToplistWidgetDefinitionType) Ptr() *ToplistWidgetDefinitionType {
	return &v
}

// NullableToplistWidgetDefinitionType handles when a null is used for ToplistWidgetDefinitionType.
type NullableToplistWidgetDefinitionType struct {
	value *ToplistWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableToplistWidgetDefinitionType) Get() *ToplistWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableToplistWidgetDefinitionType) Set(val *ToplistWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableToplistWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableToplistWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableToplistWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableToplistWidgetDefinitionType(val *ToplistWidgetDefinitionType) *NullableToplistWidgetDefinitionType {
	return &NullableToplistWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableToplistWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableToplistWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
