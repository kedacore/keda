// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// AlertValueWidgetDefinitionType Type of the alert value widget.
type AlertValueWidgetDefinitionType string

// List of AlertValueWidgetDefinitionType.
const (
	ALERTVALUEWIDGETDEFINITIONTYPE_ALERT_VALUE AlertValueWidgetDefinitionType = "alert_value"
)

var allowedAlertValueWidgetDefinitionTypeEnumValues = []AlertValueWidgetDefinitionType{
	ALERTVALUEWIDGETDEFINITIONTYPE_ALERT_VALUE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *AlertValueWidgetDefinitionType) GetAllowedValues() []AlertValueWidgetDefinitionType {
	return allowedAlertValueWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *AlertValueWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = AlertValueWidgetDefinitionType(value)
	return nil
}

// NewAlertValueWidgetDefinitionTypeFromValue returns a pointer to a valid AlertValueWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewAlertValueWidgetDefinitionTypeFromValue(v string) (*AlertValueWidgetDefinitionType, error) {
	ev := AlertValueWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for AlertValueWidgetDefinitionType: valid values are %v", v, allowedAlertValueWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v AlertValueWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedAlertValueWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to AlertValueWidgetDefinitionType value.
func (v AlertValueWidgetDefinitionType) Ptr() *AlertValueWidgetDefinitionType {
	return &v
}

// NullableAlertValueWidgetDefinitionType handles when a null is used for AlertValueWidgetDefinitionType.
type NullableAlertValueWidgetDefinitionType struct {
	value *AlertValueWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableAlertValueWidgetDefinitionType) Get() *AlertValueWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableAlertValueWidgetDefinitionType) Set(val *AlertValueWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableAlertValueWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableAlertValueWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableAlertValueWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableAlertValueWidgetDefinitionType(val *AlertValueWidgetDefinitionType) *NullableAlertValueWidgetDefinitionType {
	return &NullableAlertValueWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableAlertValueWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableAlertValueWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
