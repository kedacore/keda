// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsStatus Determines whether or not the batch has passed, failed, or is in progress.
type SyntheticsStatus string

// List of SyntheticsStatus.
const (
	SYNTHETICSSTATUS_PASSED  SyntheticsStatus = "passed"
	SYNTHETICSSTATUS_skipped SyntheticsStatus = "skipped"
	SYNTHETICSSTATUS_failed  SyntheticsStatus = "failed"
)

var allowedSyntheticsStatusEnumValues = []SyntheticsStatus{
	SYNTHETICSSTATUS_PASSED,
	SYNTHETICSSTATUS_skipped,
	SYNTHETICSSTATUS_failed,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsStatus) GetAllowedValues() []SyntheticsStatus {
	return allowedSyntheticsStatusEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsStatus) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsStatus(value)
	return nil
}

// NewSyntheticsStatusFromValue returns a pointer to a valid SyntheticsStatus
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsStatusFromValue(v string) (*SyntheticsStatus, error) {
	ev := SyntheticsStatus(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsStatus: valid values are %v", v, allowedSyntheticsStatusEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsStatus) IsValid() bool {
	for _, existing := range allowedSyntheticsStatusEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsStatus value.
func (v SyntheticsStatus) Ptr() *SyntheticsStatus {
	return &v
}

// NullableSyntheticsStatus handles when a null is used for SyntheticsStatus.
type NullableSyntheticsStatus struct {
	value *SyntheticsStatus
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsStatus) Get() *SyntheticsStatus {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsStatus) Set(val *SyntheticsStatus) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsStatus) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsStatus) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsStatus initializes the struct as if Set has been called.
func NewNullableSyntheticsStatus(val *SyntheticsStatus) *NullableSyntheticsStatus {
	return &NullableSyntheticsStatus{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsStatus) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
