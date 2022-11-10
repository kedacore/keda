// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SignalArchiveReason Reason why a signal has been archived.
type SignalArchiveReason string

// List of SignalArchiveReason.
const (
	SIGNALARCHIVEREASON_NONE                   SignalArchiveReason = "none"
	SIGNALARCHIVEREASON_FALSE_POSITIVE         SignalArchiveReason = "false_positive"
	SIGNALARCHIVEREASON_TESTING_OR_MAINTENANCE SignalArchiveReason = "testing_or_maintenance"
	SIGNALARCHIVEREASON_OTHER                  SignalArchiveReason = "other"
)

var allowedSignalArchiveReasonEnumValues = []SignalArchiveReason{
	SIGNALARCHIVEREASON_NONE,
	SIGNALARCHIVEREASON_FALSE_POSITIVE,
	SIGNALARCHIVEREASON_TESTING_OR_MAINTENANCE,
	SIGNALARCHIVEREASON_OTHER,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SignalArchiveReason) GetAllowedValues() []SignalArchiveReason {
	return allowedSignalArchiveReasonEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SignalArchiveReason) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SignalArchiveReason(value)
	return nil
}

// NewSignalArchiveReasonFromValue returns a pointer to a valid SignalArchiveReason
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSignalArchiveReasonFromValue(v string) (*SignalArchiveReason, error) {
	ev := SignalArchiveReason(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SignalArchiveReason: valid values are %v", v, allowedSignalArchiveReasonEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SignalArchiveReason) IsValid() bool {
	for _, existing := range allowedSignalArchiveReasonEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SignalArchiveReason value.
func (v SignalArchiveReason) Ptr() *SignalArchiveReason {
	return &v
}

// NullableSignalArchiveReason handles when a null is used for SignalArchiveReason.
type NullableSignalArchiveReason struct {
	value *SignalArchiveReason
	isSet bool
}

// Get returns the associated value.
func (v NullableSignalArchiveReason) Get() *SignalArchiveReason {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSignalArchiveReason) Set(val *SignalArchiveReason) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSignalArchiveReason) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSignalArchiveReason) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSignalArchiveReason initializes the struct as if Set has been called.
func NewNullableSignalArchiveReason(val *SignalArchiveReason) *NullableSignalArchiveReason {
	return &NullableSignalArchiveReason{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSignalArchiveReason) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSignalArchiveReason) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
