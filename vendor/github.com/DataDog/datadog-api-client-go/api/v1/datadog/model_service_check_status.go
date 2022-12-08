// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ServiceCheckStatus The status of a service check.
type ServiceCheckStatus int32

// List of ServiceCheckStatus.
const (
	SERVICECHECKSTATUS_OK       ServiceCheckStatus = 0
	SERVICECHECKSTATUS_WARNING  ServiceCheckStatus = 1
	SERVICECHECKSTATUS_CRITICAL ServiceCheckStatus = 2
	SERVICECHECKSTATUS_UNKNOWN  ServiceCheckStatus = 3
)

var allowedServiceCheckStatusEnumValues = []ServiceCheckStatus{
	SERVICECHECKSTATUS_OK,
	SERVICECHECKSTATUS_WARNING,
	SERVICECHECKSTATUS_CRITICAL,
	SERVICECHECKSTATUS_UNKNOWN,
}

// GetAllowedValues reeturns the list of possible values.
func (v *ServiceCheckStatus) GetAllowedValues() []ServiceCheckStatus {
	return allowedServiceCheckStatusEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *ServiceCheckStatus) UnmarshalJSON(src []byte) error {
	var value int32
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = ServiceCheckStatus(value)
	return nil
}

// NewServiceCheckStatusFromValue returns a pointer to a valid ServiceCheckStatus
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewServiceCheckStatusFromValue(v int32) (*ServiceCheckStatus, error) {
	ev := ServiceCheckStatus(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for ServiceCheckStatus: valid values are %v", v, allowedServiceCheckStatusEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v ServiceCheckStatus) IsValid() bool {
	for _, existing := range allowedServiceCheckStatusEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to ServiceCheckStatus value.
func (v ServiceCheckStatus) Ptr() *ServiceCheckStatus {
	return &v
}

// NullableServiceCheckStatus handles when a null is used for ServiceCheckStatus.
type NullableServiceCheckStatus struct {
	value *ServiceCheckStatus
	isSet bool
}

// Get returns the associated value.
func (v NullableServiceCheckStatus) Get() *ServiceCheckStatus {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableServiceCheckStatus) Set(val *ServiceCheckStatus) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableServiceCheckStatus) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableServiceCheckStatus) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableServiceCheckStatus initializes the struct as if Set has been called.
func NewNullableServiceCheckStatus(val *ServiceCheckStatus) *NullableServiceCheckStatus {
	return &NullableServiceCheckStatus{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableServiceCheckStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableServiceCheckStatus) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
