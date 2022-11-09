// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// GeomapWidgetDefinitionType Type of the geomap widget.
type GeomapWidgetDefinitionType string

// List of GeomapWidgetDefinitionType.
const (
	GEOMAPWIDGETDEFINITIONTYPE_GEOMAP GeomapWidgetDefinitionType = "geomap"
)

var allowedGeomapWidgetDefinitionTypeEnumValues = []GeomapWidgetDefinitionType{
	GEOMAPWIDGETDEFINITIONTYPE_GEOMAP,
}

// GetAllowedValues reeturns the list of possible values.
func (v *GeomapWidgetDefinitionType) GetAllowedValues() []GeomapWidgetDefinitionType {
	return allowedGeomapWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *GeomapWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = GeomapWidgetDefinitionType(value)
	return nil
}

// NewGeomapWidgetDefinitionTypeFromValue returns a pointer to a valid GeomapWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewGeomapWidgetDefinitionTypeFromValue(v string) (*GeomapWidgetDefinitionType, error) {
	ev := GeomapWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for GeomapWidgetDefinitionType: valid values are %v", v, allowedGeomapWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v GeomapWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedGeomapWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to GeomapWidgetDefinitionType value.
func (v GeomapWidgetDefinitionType) Ptr() *GeomapWidgetDefinitionType {
	return &v
}

// NullableGeomapWidgetDefinitionType handles when a null is used for GeomapWidgetDefinitionType.
type NullableGeomapWidgetDefinitionType struct {
	value *GeomapWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableGeomapWidgetDefinitionType) Get() *GeomapWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableGeomapWidgetDefinitionType) Set(val *GeomapWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableGeomapWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableGeomapWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableGeomapWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableGeomapWidgetDefinitionType(val *GeomapWidgetDefinitionType) *NullableGeomapWidgetDefinitionType {
	return &NullableGeomapWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableGeomapWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableGeomapWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
