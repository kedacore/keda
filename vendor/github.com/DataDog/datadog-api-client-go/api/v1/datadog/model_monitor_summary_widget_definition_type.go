// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// MonitorSummaryWidgetDefinitionType Type of the monitor summary widget.
type MonitorSummaryWidgetDefinitionType string

// List of MonitorSummaryWidgetDefinitionType.
const (
	MONITORSUMMARYWIDGETDEFINITIONTYPE_MANAGE_STATUS MonitorSummaryWidgetDefinitionType = "manage_status"
)

var allowedMonitorSummaryWidgetDefinitionTypeEnumValues = []MonitorSummaryWidgetDefinitionType{
	MONITORSUMMARYWIDGETDEFINITIONTYPE_MANAGE_STATUS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *MonitorSummaryWidgetDefinitionType) GetAllowedValues() []MonitorSummaryWidgetDefinitionType {
	return allowedMonitorSummaryWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *MonitorSummaryWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = MonitorSummaryWidgetDefinitionType(value)
	return nil
}

// NewMonitorSummaryWidgetDefinitionTypeFromValue returns a pointer to a valid MonitorSummaryWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewMonitorSummaryWidgetDefinitionTypeFromValue(v string) (*MonitorSummaryWidgetDefinitionType, error) {
	ev := MonitorSummaryWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for MonitorSummaryWidgetDefinitionType: valid values are %v", v, allowedMonitorSummaryWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v MonitorSummaryWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedMonitorSummaryWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to MonitorSummaryWidgetDefinitionType value.
func (v MonitorSummaryWidgetDefinitionType) Ptr() *MonitorSummaryWidgetDefinitionType {
	return &v
}

// NullableMonitorSummaryWidgetDefinitionType handles when a null is used for MonitorSummaryWidgetDefinitionType.
type NullableMonitorSummaryWidgetDefinitionType struct {
	value *MonitorSummaryWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableMonitorSummaryWidgetDefinitionType) Get() *MonitorSummaryWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMonitorSummaryWidgetDefinitionType) Set(val *MonitorSummaryWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMonitorSummaryWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableMonitorSummaryWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMonitorSummaryWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableMonitorSummaryWidgetDefinitionType(val *MonitorSummaryWidgetDefinitionType) *NullableMonitorSummaryWidgetDefinitionType {
	return &NullableMonitorSummaryWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMonitorSummaryWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMonitorSummaryWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
