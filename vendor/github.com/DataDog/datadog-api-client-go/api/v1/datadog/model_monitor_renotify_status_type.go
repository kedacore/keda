// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// MonitorRenotifyStatusType The different statuses for which renotification is supported.
type MonitorRenotifyStatusType string

// List of MonitorRenotifyStatusType.
const (
	MONITORRENOTIFYSTATUSTYPE_ALERT   MonitorRenotifyStatusType = "alert"
	MONITORRENOTIFYSTATUSTYPE_WARN    MonitorRenotifyStatusType = "warn"
	MONITORRENOTIFYSTATUSTYPE_NO_DATA MonitorRenotifyStatusType = "no data"
)

var allowedMonitorRenotifyStatusTypeEnumValues = []MonitorRenotifyStatusType{
	MONITORRENOTIFYSTATUSTYPE_ALERT,
	MONITORRENOTIFYSTATUSTYPE_WARN,
	MONITORRENOTIFYSTATUSTYPE_NO_DATA,
}

// GetAllowedValues reeturns the list of possible values.
func (v *MonitorRenotifyStatusType) GetAllowedValues() []MonitorRenotifyStatusType {
	return allowedMonitorRenotifyStatusTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *MonitorRenotifyStatusType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = MonitorRenotifyStatusType(value)
	return nil
}

// NewMonitorRenotifyStatusTypeFromValue returns a pointer to a valid MonitorRenotifyStatusType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewMonitorRenotifyStatusTypeFromValue(v string) (*MonitorRenotifyStatusType, error) {
	ev := MonitorRenotifyStatusType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for MonitorRenotifyStatusType: valid values are %v", v, allowedMonitorRenotifyStatusTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v MonitorRenotifyStatusType) IsValid() bool {
	for _, existing := range allowedMonitorRenotifyStatusTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to MonitorRenotifyStatusType value.
func (v MonitorRenotifyStatusType) Ptr() *MonitorRenotifyStatusType {
	return &v
}

// NullableMonitorRenotifyStatusType handles when a null is used for MonitorRenotifyStatusType.
type NullableMonitorRenotifyStatusType struct {
	value *MonitorRenotifyStatusType
	isSet bool
}

// Get returns the associated value.
func (v NullableMonitorRenotifyStatusType) Get() *MonitorRenotifyStatusType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMonitorRenotifyStatusType) Set(val *MonitorRenotifyStatusType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMonitorRenotifyStatusType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableMonitorRenotifyStatusType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMonitorRenotifyStatusType initializes the struct as if Set has been called.
func NewNullableMonitorRenotifyStatusType(val *MonitorRenotifyStatusType) *NullableMonitorRenotifyStatusType {
	return &NullableMonitorRenotifyStatusType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMonitorRenotifyStatusType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMonitorRenotifyStatusType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
