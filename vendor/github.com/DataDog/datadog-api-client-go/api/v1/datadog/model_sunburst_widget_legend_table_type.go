// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SunburstWidgetLegendTableType Whether or not to show a table legend.
type SunburstWidgetLegendTableType string

// List of SunburstWidgetLegendTableType.
const (
	SUNBURSTWIDGETLEGENDTABLETYPE_TABLE SunburstWidgetLegendTableType = "table"
	SUNBURSTWIDGETLEGENDTABLETYPE_NONE  SunburstWidgetLegendTableType = "none"
)

var allowedSunburstWidgetLegendTableTypeEnumValues = []SunburstWidgetLegendTableType{
	SUNBURSTWIDGETLEGENDTABLETYPE_TABLE,
	SUNBURSTWIDGETLEGENDTABLETYPE_NONE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SunburstWidgetLegendTableType) GetAllowedValues() []SunburstWidgetLegendTableType {
	return allowedSunburstWidgetLegendTableTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SunburstWidgetLegendTableType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SunburstWidgetLegendTableType(value)
	return nil
}

// NewSunburstWidgetLegendTableTypeFromValue returns a pointer to a valid SunburstWidgetLegendTableType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSunburstWidgetLegendTableTypeFromValue(v string) (*SunburstWidgetLegendTableType, error) {
	ev := SunburstWidgetLegendTableType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SunburstWidgetLegendTableType: valid values are %v", v, allowedSunburstWidgetLegendTableTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SunburstWidgetLegendTableType) IsValid() bool {
	for _, existing := range allowedSunburstWidgetLegendTableTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SunburstWidgetLegendTableType value.
func (v SunburstWidgetLegendTableType) Ptr() *SunburstWidgetLegendTableType {
	return &v
}

// NullableSunburstWidgetLegendTableType handles when a null is used for SunburstWidgetLegendTableType.
type NullableSunburstWidgetLegendTableType struct {
	value *SunburstWidgetLegendTableType
	isSet bool
}

// Get returns the associated value.
func (v NullableSunburstWidgetLegendTableType) Get() *SunburstWidgetLegendTableType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSunburstWidgetLegendTableType) Set(val *SunburstWidgetLegendTableType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSunburstWidgetLegendTableType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSunburstWidgetLegendTableType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSunburstWidgetLegendTableType initializes the struct as if Set has been called.
func NewNullableSunburstWidgetLegendTableType(val *SunburstWidgetLegendTableType) *NullableSunburstWidgetLegendTableType {
	return &NullableSunburstWidgetLegendTableType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSunburstWidgetLegendTableType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSunburstWidgetLegendTableType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
