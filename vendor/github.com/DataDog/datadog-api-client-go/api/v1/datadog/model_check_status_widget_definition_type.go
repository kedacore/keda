// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// CheckStatusWidgetDefinitionType Type of the check status widget.
type CheckStatusWidgetDefinitionType string

// List of CheckStatusWidgetDefinitionType.
const (
	CHECKSTATUSWIDGETDEFINITIONTYPE_CHECK_STATUS CheckStatusWidgetDefinitionType = "check_status"
)

var allowedCheckStatusWidgetDefinitionTypeEnumValues = []CheckStatusWidgetDefinitionType{
	CHECKSTATUSWIDGETDEFINITIONTYPE_CHECK_STATUS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *CheckStatusWidgetDefinitionType) GetAllowedValues() []CheckStatusWidgetDefinitionType {
	return allowedCheckStatusWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *CheckStatusWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = CheckStatusWidgetDefinitionType(value)
	return nil
}

// NewCheckStatusWidgetDefinitionTypeFromValue returns a pointer to a valid CheckStatusWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewCheckStatusWidgetDefinitionTypeFromValue(v string) (*CheckStatusWidgetDefinitionType, error) {
	ev := CheckStatusWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for CheckStatusWidgetDefinitionType: valid values are %v", v, allowedCheckStatusWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v CheckStatusWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedCheckStatusWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to CheckStatusWidgetDefinitionType value.
func (v CheckStatusWidgetDefinitionType) Ptr() *CheckStatusWidgetDefinitionType {
	return &v
}

// NullableCheckStatusWidgetDefinitionType handles when a null is used for CheckStatusWidgetDefinitionType.
type NullableCheckStatusWidgetDefinitionType struct {
	value *CheckStatusWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableCheckStatusWidgetDefinitionType) Get() *CheckStatusWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableCheckStatusWidgetDefinitionType) Set(val *CheckStatusWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableCheckStatusWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableCheckStatusWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableCheckStatusWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableCheckStatusWidgetDefinitionType(val *CheckStatusWidgetDefinitionType) *NullableCheckStatusWidgetDefinitionType {
	return &NullableCheckStatusWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableCheckStatusWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableCheckStatusWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
