// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SLOErrorTimeframe The timeframe of the threshold associated with this error
// or "all" if all thresholds are affected.
type SLOErrorTimeframe string

// List of SLOErrorTimeframe.
const (
	SLOERRORTIMEFRAME_SEVEN_DAYS  SLOErrorTimeframe = "7d"
	SLOERRORTIMEFRAME_THIRTY_DAYS SLOErrorTimeframe = "30d"
	SLOERRORTIMEFRAME_NINETY_DAYS SLOErrorTimeframe = "90d"
	SLOERRORTIMEFRAME_ALL         SLOErrorTimeframe = "all"
)

var allowedSLOErrorTimeframeEnumValues = []SLOErrorTimeframe{
	SLOERRORTIMEFRAME_SEVEN_DAYS,
	SLOERRORTIMEFRAME_THIRTY_DAYS,
	SLOERRORTIMEFRAME_NINETY_DAYS,
	SLOERRORTIMEFRAME_ALL,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SLOErrorTimeframe) GetAllowedValues() []SLOErrorTimeframe {
	return allowedSLOErrorTimeframeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SLOErrorTimeframe) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SLOErrorTimeframe(value)
	return nil
}

// NewSLOErrorTimeframeFromValue returns a pointer to a valid SLOErrorTimeframe
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSLOErrorTimeframeFromValue(v string) (*SLOErrorTimeframe, error) {
	ev := SLOErrorTimeframe(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SLOErrorTimeframe: valid values are %v", v, allowedSLOErrorTimeframeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SLOErrorTimeframe) IsValid() bool {
	for _, existing := range allowedSLOErrorTimeframeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SLOErrorTimeframe value.
func (v SLOErrorTimeframe) Ptr() *SLOErrorTimeframe {
	return &v
}

// NullableSLOErrorTimeframe handles when a null is used for SLOErrorTimeframe.
type NullableSLOErrorTimeframe struct {
	value *SLOErrorTimeframe
	isSet bool
}

// Get returns the associated value.
func (v NullableSLOErrorTimeframe) Get() *SLOErrorTimeframe {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSLOErrorTimeframe) Set(val *SLOErrorTimeframe) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSLOErrorTimeframe) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSLOErrorTimeframe) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSLOErrorTimeframe initializes the struct as if Set has been called.
func NewNullableSLOErrorTimeframe(val *SLOErrorTimeframe) *NullableSLOErrorTimeframe {
	return &NullableSLOErrorTimeframe{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSLOErrorTimeframe) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSLOErrorTimeframe) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
