// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// TreeMapWidgetDefinitionType Type of the treemap widget.
type TreeMapWidgetDefinitionType string

// List of TreeMapWidgetDefinitionType.
const (
	TREEMAPWIDGETDEFINITIONTYPE_TREEMAP TreeMapWidgetDefinitionType = "treemap"
)

var allowedTreeMapWidgetDefinitionTypeEnumValues = []TreeMapWidgetDefinitionType{
	TREEMAPWIDGETDEFINITIONTYPE_TREEMAP,
}

// GetAllowedValues reeturns the list of possible values.
func (v *TreeMapWidgetDefinitionType) GetAllowedValues() []TreeMapWidgetDefinitionType {
	return allowedTreeMapWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *TreeMapWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = TreeMapWidgetDefinitionType(value)
	return nil
}

// NewTreeMapWidgetDefinitionTypeFromValue returns a pointer to a valid TreeMapWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewTreeMapWidgetDefinitionTypeFromValue(v string) (*TreeMapWidgetDefinitionType, error) {
	ev := TreeMapWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for TreeMapWidgetDefinitionType: valid values are %v", v, allowedTreeMapWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v TreeMapWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedTreeMapWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to TreeMapWidgetDefinitionType value.
func (v TreeMapWidgetDefinitionType) Ptr() *TreeMapWidgetDefinitionType {
	return &v
}

// NullableTreeMapWidgetDefinitionType handles when a null is used for TreeMapWidgetDefinitionType.
type NullableTreeMapWidgetDefinitionType struct {
	value *TreeMapWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableTreeMapWidgetDefinitionType) Get() *TreeMapWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableTreeMapWidgetDefinitionType) Set(val *TreeMapWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableTreeMapWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableTreeMapWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableTreeMapWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableTreeMapWidgetDefinitionType(val *TreeMapWidgetDefinitionType) *NullableTreeMapWidgetDefinitionType {
	return &NullableTreeMapWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableTreeMapWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableTreeMapWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
