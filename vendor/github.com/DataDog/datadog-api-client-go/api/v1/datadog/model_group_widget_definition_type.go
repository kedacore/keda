// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// GroupWidgetDefinitionType Type of the group widget.
type GroupWidgetDefinitionType string

// List of GroupWidgetDefinitionType.
const (
	GROUPWIDGETDEFINITIONTYPE_GROUP GroupWidgetDefinitionType = "group"
)

var allowedGroupWidgetDefinitionTypeEnumValues = []GroupWidgetDefinitionType{
	GROUPWIDGETDEFINITIONTYPE_GROUP,
}

// GetAllowedValues reeturns the list of possible values.
func (v *GroupWidgetDefinitionType) GetAllowedValues() []GroupWidgetDefinitionType {
	return allowedGroupWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *GroupWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = GroupWidgetDefinitionType(value)
	return nil
}

// NewGroupWidgetDefinitionTypeFromValue returns a pointer to a valid GroupWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewGroupWidgetDefinitionTypeFromValue(v string) (*GroupWidgetDefinitionType, error) {
	ev := GroupWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for GroupWidgetDefinitionType: valid values are %v", v, allowedGroupWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v GroupWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedGroupWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to GroupWidgetDefinitionType value.
func (v GroupWidgetDefinitionType) Ptr() *GroupWidgetDefinitionType {
	return &v
}

// NullableGroupWidgetDefinitionType handles when a null is used for GroupWidgetDefinitionType.
type NullableGroupWidgetDefinitionType struct {
	value *GroupWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableGroupWidgetDefinitionType) Get() *GroupWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableGroupWidgetDefinitionType) Set(val *GroupWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableGroupWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableGroupWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableGroupWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableGroupWidgetDefinitionType(val *GroupWidgetDefinitionType) *NullableGroupWidgetDefinitionType {
	return &NullableGroupWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableGroupWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableGroupWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
