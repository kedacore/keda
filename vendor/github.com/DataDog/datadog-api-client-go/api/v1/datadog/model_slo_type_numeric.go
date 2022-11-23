// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SLOTypeNumeric A numeric representation of the type of the service level objective (`0` for
// monitor, `1` for metric). Always included in service level objective responses.
// Ignored in create/update requests.
type SLOTypeNumeric int32

// List of SLOTypeNumeric.
const (
	SLOTYPENUMERIC_MONITOR SLOTypeNumeric = 0
	SLOTYPENUMERIC_METRIC  SLOTypeNumeric = 1
)

var allowedSLOTypeNumericEnumValues = []SLOTypeNumeric{
	SLOTYPENUMERIC_MONITOR,
	SLOTYPENUMERIC_METRIC,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SLOTypeNumeric) GetAllowedValues() []SLOTypeNumeric {
	return allowedSLOTypeNumericEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SLOTypeNumeric) UnmarshalJSON(src []byte) error {
	var value int32
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SLOTypeNumeric(value)
	return nil
}

// NewSLOTypeNumericFromValue returns a pointer to a valid SLOTypeNumeric
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSLOTypeNumericFromValue(v int32) (*SLOTypeNumeric, error) {
	ev := SLOTypeNumeric(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SLOTypeNumeric: valid values are %v", v, allowedSLOTypeNumericEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SLOTypeNumeric) IsValid() bool {
	for _, existing := range allowedSLOTypeNumericEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SLOTypeNumeric value.
func (v SLOTypeNumeric) Ptr() *SLOTypeNumeric {
	return &v
}

// NullableSLOTypeNumeric handles when a null is used for SLOTypeNumeric.
type NullableSLOTypeNumeric struct {
	value *SLOTypeNumeric
	isSet bool
}

// Get returns the associated value.
func (v NullableSLOTypeNumeric) Get() *SLOTypeNumeric {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSLOTypeNumeric) Set(val *SLOTypeNumeric) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSLOTypeNumeric) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSLOTypeNumeric) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSLOTypeNumeric initializes the struct as if Set has been called.
func NewNullableSLOTypeNumeric(val *SLOTypeNumeric) *NullableSLOTypeNumeric {
	return &NullableSLOTypeNumeric{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSLOTypeNumeric) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSLOTypeNumeric) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
