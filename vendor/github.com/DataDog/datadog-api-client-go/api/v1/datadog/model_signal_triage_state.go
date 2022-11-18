// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SignalTriageState The new triage state of the signal.
type SignalTriageState string

// List of SignalTriageState.
const (
	SIGNALTRIAGESTATE_OPEN         SignalTriageState = "open"
	SIGNALTRIAGESTATE_ARCHIVED     SignalTriageState = "archived"
	SIGNALTRIAGESTATE_UNDER_REVIEW SignalTriageState = "under_review"
)

var allowedSignalTriageStateEnumValues = []SignalTriageState{
	SIGNALTRIAGESTATE_OPEN,
	SIGNALTRIAGESTATE_ARCHIVED,
	SIGNALTRIAGESTATE_UNDER_REVIEW,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SignalTriageState) GetAllowedValues() []SignalTriageState {
	return allowedSignalTriageStateEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SignalTriageState) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SignalTriageState(value)
	return nil
}

// NewSignalTriageStateFromValue returns a pointer to a valid SignalTriageState
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSignalTriageStateFromValue(v string) (*SignalTriageState, error) {
	ev := SignalTriageState(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SignalTriageState: valid values are %v", v, allowedSignalTriageStateEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SignalTriageState) IsValid() bool {
	for _, existing := range allowedSignalTriageStateEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SignalTriageState value.
func (v SignalTriageState) Ptr() *SignalTriageState {
	return &v
}

// NullableSignalTriageState handles when a null is used for SignalTriageState.
type NullableSignalTriageState struct {
	value *SignalTriageState
	isSet bool
}

// Get returns the associated value.
func (v NullableSignalTriageState) Get() *SignalTriageState {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSignalTriageState) Set(val *SignalTriageState) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSignalTriageState) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSignalTriageState) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSignalTriageState initializes the struct as if Set has been called.
func NewNullableSignalTriageState(val *SignalTriageState) *NullableSignalTriageState {
	return &NullableSignalTriageState{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSignalTriageState) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSignalTriageState) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
