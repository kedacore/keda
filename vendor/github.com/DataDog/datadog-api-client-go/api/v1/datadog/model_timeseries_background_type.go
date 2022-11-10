// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// TimeseriesBackgroundType Timeseries is made using an area or bars.
type TimeseriesBackgroundType string

// List of TimeseriesBackgroundType.
const (
	TIMESERIESBACKGROUNDTYPE_BARS TimeseriesBackgroundType = "bars"
	TIMESERIESBACKGROUNDTYPE_AREA TimeseriesBackgroundType = "area"
)

var allowedTimeseriesBackgroundTypeEnumValues = []TimeseriesBackgroundType{
	TIMESERIESBACKGROUNDTYPE_BARS,
	TIMESERIESBACKGROUNDTYPE_AREA,
}

// GetAllowedValues reeturns the list of possible values.
func (v *TimeseriesBackgroundType) GetAllowedValues() []TimeseriesBackgroundType {
	return allowedTimeseriesBackgroundTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *TimeseriesBackgroundType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = TimeseriesBackgroundType(value)
	return nil
}

// NewTimeseriesBackgroundTypeFromValue returns a pointer to a valid TimeseriesBackgroundType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewTimeseriesBackgroundTypeFromValue(v string) (*TimeseriesBackgroundType, error) {
	ev := TimeseriesBackgroundType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for TimeseriesBackgroundType: valid values are %v", v, allowedTimeseriesBackgroundTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v TimeseriesBackgroundType) IsValid() bool {
	for _, existing := range allowedTimeseriesBackgroundTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to TimeseriesBackgroundType value.
func (v TimeseriesBackgroundType) Ptr() *TimeseriesBackgroundType {
	return &v
}

// NullableTimeseriesBackgroundType handles when a null is used for TimeseriesBackgroundType.
type NullableTimeseriesBackgroundType struct {
	value *TimeseriesBackgroundType
	isSet bool
}

// Get returns the associated value.
func (v NullableTimeseriesBackgroundType) Get() *TimeseriesBackgroundType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableTimeseriesBackgroundType) Set(val *TimeseriesBackgroundType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableTimeseriesBackgroundType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableTimeseriesBackgroundType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableTimeseriesBackgroundType initializes the struct as if Set has been called.
func NewNullableTimeseriesBackgroundType(val *TimeseriesBackgroundType) *NullableTimeseriesBackgroundType {
	return &NullableTimeseriesBackgroundType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableTimeseriesBackgroundType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableTimeseriesBackgroundType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
