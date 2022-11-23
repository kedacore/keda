// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// FreeTextWidgetDefinitionType Type of the free text widget.
type FreeTextWidgetDefinitionType string

// List of FreeTextWidgetDefinitionType.
const (
	FREETEXTWIDGETDEFINITIONTYPE_FREE_TEXT FreeTextWidgetDefinitionType = "free_text"
)

var allowedFreeTextWidgetDefinitionTypeEnumValues = []FreeTextWidgetDefinitionType{
	FREETEXTWIDGETDEFINITIONTYPE_FREE_TEXT,
}

// GetAllowedValues reeturns the list of possible values.
func (v *FreeTextWidgetDefinitionType) GetAllowedValues() []FreeTextWidgetDefinitionType {
	return allowedFreeTextWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *FreeTextWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = FreeTextWidgetDefinitionType(value)
	return nil
}

// NewFreeTextWidgetDefinitionTypeFromValue returns a pointer to a valid FreeTextWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewFreeTextWidgetDefinitionTypeFromValue(v string) (*FreeTextWidgetDefinitionType, error) {
	ev := FreeTextWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for FreeTextWidgetDefinitionType: valid values are %v", v, allowedFreeTextWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v FreeTextWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedFreeTextWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to FreeTextWidgetDefinitionType value.
func (v FreeTextWidgetDefinitionType) Ptr() *FreeTextWidgetDefinitionType {
	return &v
}

// NullableFreeTextWidgetDefinitionType handles when a null is used for FreeTextWidgetDefinitionType.
type NullableFreeTextWidgetDefinitionType struct {
	value *FreeTextWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableFreeTextWidgetDefinitionType) Get() *FreeTextWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableFreeTextWidgetDefinitionType) Set(val *FreeTextWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableFreeTextWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableFreeTextWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableFreeTextWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableFreeTextWidgetDefinitionType(val *FreeTextWidgetDefinitionType) *NullableFreeTextWidgetDefinitionType {
	return &NullableFreeTextWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableFreeTextWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableFreeTextWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
