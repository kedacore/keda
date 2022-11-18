// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// HeatMapWidgetDefinitionType Type of the heat map widget.
type HeatMapWidgetDefinitionType string

// List of HeatMapWidgetDefinitionType.
const (
	HEATMAPWIDGETDEFINITIONTYPE_HEATMAP HeatMapWidgetDefinitionType = "heatmap"
)

var allowedHeatMapWidgetDefinitionTypeEnumValues = []HeatMapWidgetDefinitionType{
	HEATMAPWIDGETDEFINITIONTYPE_HEATMAP,
}

// GetAllowedValues reeturns the list of possible values.
func (v *HeatMapWidgetDefinitionType) GetAllowedValues() []HeatMapWidgetDefinitionType {
	return allowedHeatMapWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *HeatMapWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = HeatMapWidgetDefinitionType(value)
	return nil
}

// NewHeatMapWidgetDefinitionTypeFromValue returns a pointer to a valid HeatMapWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewHeatMapWidgetDefinitionTypeFromValue(v string) (*HeatMapWidgetDefinitionType, error) {
	ev := HeatMapWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for HeatMapWidgetDefinitionType: valid values are %v", v, allowedHeatMapWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v HeatMapWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedHeatMapWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to HeatMapWidgetDefinitionType value.
func (v HeatMapWidgetDefinitionType) Ptr() *HeatMapWidgetDefinitionType {
	return &v
}

// NullableHeatMapWidgetDefinitionType handles when a null is used for HeatMapWidgetDefinitionType.
type NullableHeatMapWidgetDefinitionType struct {
	value *HeatMapWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableHeatMapWidgetDefinitionType) Get() *HeatMapWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableHeatMapWidgetDefinitionType) Set(val *HeatMapWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableHeatMapWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableHeatMapWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableHeatMapWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableHeatMapWidgetDefinitionType(val *HeatMapWidgetDefinitionType) *NullableHeatMapWidgetDefinitionType {
	return &NullableHeatMapWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableHeatMapWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableHeatMapWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
