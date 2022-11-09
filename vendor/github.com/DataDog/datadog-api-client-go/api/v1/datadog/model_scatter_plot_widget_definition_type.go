// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ScatterPlotWidgetDefinitionType Type of the scatter plot widget.
type ScatterPlotWidgetDefinitionType string

// List of ScatterPlotWidgetDefinitionType.
const (
	SCATTERPLOTWIDGETDEFINITIONTYPE_SCATTERPLOT ScatterPlotWidgetDefinitionType = "scatterplot"
)

var allowedScatterPlotWidgetDefinitionTypeEnumValues = []ScatterPlotWidgetDefinitionType{
	SCATTERPLOTWIDGETDEFINITIONTYPE_SCATTERPLOT,
}

// GetAllowedValues reeturns the list of possible values.
func (v *ScatterPlotWidgetDefinitionType) GetAllowedValues() []ScatterPlotWidgetDefinitionType {
	return allowedScatterPlotWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *ScatterPlotWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = ScatterPlotWidgetDefinitionType(value)
	return nil
}

// NewScatterPlotWidgetDefinitionTypeFromValue returns a pointer to a valid ScatterPlotWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewScatterPlotWidgetDefinitionTypeFromValue(v string) (*ScatterPlotWidgetDefinitionType, error) {
	ev := ScatterPlotWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for ScatterPlotWidgetDefinitionType: valid values are %v", v, allowedScatterPlotWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v ScatterPlotWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedScatterPlotWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to ScatterPlotWidgetDefinitionType value.
func (v ScatterPlotWidgetDefinitionType) Ptr() *ScatterPlotWidgetDefinitionType {
	return &v
}

// NullableScatterPlotWidgetDefinitionType handles when a null is used for ScatterPlotWidgetDefinitionType.
type NullableScatterPlotWidgetDefinitionType struct {
	value *ScatterPlotWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableScatterPlotWidgetDefinitionType) Get() *ScatterPlotWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableScatterPlotWidgetDefinitionType) Set(val *ScatterPlotWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableScatterPlotWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableScatterPlotWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableScatterPlotWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableScatterPlotWidgetDefinitionType(val *ScatterPlotWidgetDefinitionType) *NullableScatterPlotWidgetDefinitionType {
	return &NullableScatterPlotWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableScatterPlotWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableScatterPlotWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
