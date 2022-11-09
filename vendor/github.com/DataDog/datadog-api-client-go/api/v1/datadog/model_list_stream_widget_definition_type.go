// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ListStreamWidgetDefinitionType Type of the list stream widget.
type ListStreamWidgetDefinitionType string

// List of ListStreamWidgetDefinitionType.
const (
	LISTSTREAMWIDGETDEFINITIONTYPE_LIST_STREAM ListStreamWidgetDefinitionType = "list_stream"
)

var allowedListStreamWidgetDefinitionTypeEnumValues = []ListStreamWidgetDefinitionType{
	LISTSTREAMWIDGETDEFINITIONTYPE_LIST_STREAM,
}

// GetAllowedValues reeturns the list of possible values.
func (v *ListStreamWidgetDefinitionType) GetAllowedValues() []ListStreamWidgetDefinitionType {
	return allowedListStreamWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *ListStreamWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = ListStreamWidgetDefinitionType(value)
	return nil
}

// NewListStreamWidgetDefinitionTypeFromValue returns a pointer to a valid ListStreamWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewListStreamWidgetDefinitionTypeFromValue(v string) (*ListStreamWidgetDefinitionType, error) {
	ev := ListStreamWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for ListStreamWidgetDefinitionType: valid values are %v", v, allowedListStreamWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v ListStreamWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedListStreamWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to ListStreamWidgetDefinitionType value.
func (v ListStreamWidgetDefinitionType) Ptr() *ListStreamWidgetDefinitionType {
	return &v
}

// NullableListStreamWidgetDefinitionType handles when a null is used for ListStreamWidgetDefinitionType.
type NullableListStreamWidgetDefinitionType struct {
	value *ListStreamWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableListStreamWidgetDefinitionType) Get() *ListStreamWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableListStreamWidgetDefinitionType) Set(val *ListStreamWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableListStreamWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableListStreamWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableListStreamWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableListStreamWidgetDefinitionType(val *ListStreamWidgetDefinitionType) *NullableListStreamWidgetDefinitionType {
	return &NullableListStreamWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableListStreamWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableListStreamWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
