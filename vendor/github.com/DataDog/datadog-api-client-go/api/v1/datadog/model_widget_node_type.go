// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetNodeType Which type of node to use in the map.
type WidgetNodeType string

// List of WidgetNodeType.
const (
	WIDGETNODETYPE_HOST      WidgetNodeType = "host"
	WIDGETNODETYPE_CONTAINER WidgetNodeType = "container"
)

var allowedWidgetNodeTypeEnumValues = []WidgetNodeType{
	WIDGETNODETYPE_HOST,
	WIDGETNODETYPE_CONTAINER,
}

// GetAllowedValues reeturns the list of possible values.
func (v *WidgetNodeType) GetAllowedValues() []WidgetNodeType {
	return allowedWidgetNodeTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *WidgetNodeType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = WidgetNodeType(value)
	return nil
}

// NewWidgetNodeTypeFromValue returns a pointer to a valid WidgetNodeType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewWidgetNodeTypeFromValue(v string) (*WidgetNodeType, error) {
	ev := WidgetNodeType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for WidgetNodeType: valid values are %v", v, allowedWidgetNodeTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v WidgetNodeType) IsValid() bool {
	for _, existing := range allowedWidgetNodeTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to WidgetNodeType value.
func (v WidgetNodeType) Ptr() *WidgetNodeType {
	return &v
}

// NullableWidgetNodeType handles when a null is used for WidgetNodeType.
type NullableWidgetNodeType struct {
	value *WidgetNodeType
	isSet bool
}

// Get returns the associated value.
func (v NullableWidgetNodeType) Get() *WidgetNodeType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableWidgetNodeType) Set(val *WidgetNodeType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableWidgetNodeType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableWidgetNodeType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableWidgetNodeType initializes the struct as if Set has been called.
func NewNullableWidgetNodeType(val *WidgetNodeType) *NullableWidgetNodeType {
	return &NullableWidgetNodeType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableWidgetNodeType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableWidgetNodeType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
