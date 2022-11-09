// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsTestPauseStatus Define whether you want to start (`live`) or pause (`paused`) a
// Synthetic test.
type SyntheticsTestPauseStatus string

// List of SyntheticsTestPauseStatus.
const (
	SYNTHETICSTESTPAUSESTATUS_LIVE   SyntheticsTestPauseStatus = "live"
	SYNTHETICSTESTPAUSESTATUS_PAUSED SyntheticsTestPauseStatus = "paused"
)

var allowedSyntheticsTestPauseStatusEnumValues = []SyntheticsTestPauseStatus{
	SYNTHETICSTESTPAUSESTATUS_LIVE,
	SYNTHETICSTESTPAUSESTATUS_PAUSED,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsTestPauseStatus) GetAllowedValues() []SyntheticsTestPauseStatus {
	return allowedSyntheticsTestPauseStatusEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsTestPauseStatus) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsTestPauseStatus(value)
	return nil
}

// NewSyntheticsTestPauseStatusFromValue returns a pointer to a valid SyntheticsTestPauseStatus
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsTestPauseStatusFromValue(v string) (*SyntheticsTestPauseStatus, error) {
	ev := SyntheticsTestPauseStatus(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsTestPauseStatus: valid values are %v", v, allowedSyntheticsTestPauseStatusEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsTestPauseStatus) IsValid() bool {
	for _, existing := range allowedSyntheticsTestPauseStatusEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsTestPauseStatus value.
func (v SyntheticsTestPauseStatus) Ptr() *SyntheticsTestPauseStatus {
	return &v
}

// NullableSyntheticsTestPauseStatus handles when a null is used for SyntheticsTestPauseStatus.
type NullableSyntheticsTestPauseStatus struct {
	value *SyntheticsTestPauseStatus
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsTestPauseStatus) Get() *SyntheticsTestPauseStatus {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsTestPauseStatus) Set(val *SyntheticsTestPauseStatus) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsTestPauseStatus) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsTestPauseStatus) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsTestPauseStatus initializes the struct as if Set has been called.
func NewNullableSyntheticsTestPauseStatus(val *SyntheticsTestPauseStatus) *NullableSyntheticsTestPauseStatus {
	return &NullableSyntheticsTestPauseStatus{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsTestPauseStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsTestPauseStatus) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
