// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsTestMonitorStatus The status of your Synthetic monitor.
// * `O` for not triggered
// * `1` for triggered
// * `2` for no data
type SyntheticsTestMonitorStatus int64

// List of SyntheticsTestMonitorStatus.
const (
	SYNTHETICSTESTMONITORSTATUS_UNTRIGGERED SyntheticsTestMonitorStatus = 0
	SYNTHETICSTESTMONITORSTATUS_TRIGGERED   SyntheticsTestMonitorStatus = 1
	SYNTHETICSTESTMONITORSTATUS_NO_DATA     SyntheticsTestMonitorStatus = 2
)

var allowedSyntheticsTestMonitorStatusEnumValues = []SyntheticsTestMonitorStatus{
	SYNTHETICSTESTMONITORSTATUS_UNTRIGGERED,
	SYNTHETICSTESTMONITORSTATUS_TRIGGERED,
	SYNTHETICSTESTMONITORSTATUS_NO_DATA,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsTestMonitorStatus) GetAllowedValues() []SyntheticsTestMonitorStatus {
	return allowedSyntheticsTestMonitorStatusEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsTestMonitorStatus) UnmarshalJSON(src []byte) error {
	var value int64
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsTestMonitorStatus(value)
	return nil
}

// NewSyntheticsTestMonitorStatusFromValue returns a pointer to a valid SyntheticsTestMonitorStatus
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsTestMonitorStatusFromValue(v int64) (*SyntheticsTestMonitorStatus, error) {
	ev := SyntheticsTestMonitorStatus(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsTestMonitorStatus: valid values are %v", v, allowedSyntheticsTestMonitorStatusEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsTestMonitorStatus) IsValid() bool {
	for _, existing := range allowedSyntheticsTestMonitorStatusEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsTestMonitorStatus value.
func (v SyntheticsTestMonitorStatus) Ptr() *SyntheticsTestMonitorStatus {
	return &v
}

// NullableSyntheticsTestMonitorStatus handles when a null is used for SyntheticsTestMonitorStatus.
type NullableSyntheticsTestMonitorStatus struct {
	value *SyntheticsTestMonitorStatus
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsTestMonitorStatus) Get() *SyntheticsTestMonitorStatus {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsTestMonitorStatus) Set(val *SyntheticsTestMonitorStatus) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsTestMonitorStatus) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsTestMonitorStatus) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsTestMonitorStatus initializes the struct as if Set has been called.
func NewNullableSyntheticsTestMonitorStatus(val *SyntheticsTestMonitorStatus) *NullableSyntheticsTestMonitorStatus {
	return &NullableSyntheticsTestMonitorStatus{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsTestMonitorStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsTestMonitorStatus) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
