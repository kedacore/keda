// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// DistributionWidgetHistogramRequestType Request type for the histogram request.
type DistributionWidgetHistogramRequestType string

// List of DistributionWidgetHistogramRequestType.
const (
	DISTRIBUTIONWIDGETHISTOGRAMREQUESTTYPE_HISTOGRAM DistributionWidgetHistogramRequestType = "histogram"
)

var allowedDistributionWidgetHistogramRequestTypeEnumValues = []DistributionWidgetHistogramRequestType{
	DISTRIBUTIONWIDGETHISTOGRAMREQUESTTYPE_HISTOGRAM,
}

// GetAllowedValues reeturns the list of possible values.
func (v *DistributionWidgetHistogramRequestType) GetAllowedValues() []DistributionWidgetHistogramRequestType {
	return allowedDistributionWidgetHistogramRequestTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *DistributionWidgetHistogramRequestType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = DistributionWidgetHistogramRequestType(value)
	return nil
}

// NewDistributionWidgetHistogramRequestTypeFromValue returns a pointer to a valid DistributionWidgetHistogramRequestType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewDistributionWidgetHistogramRequestTypeFromValue(v string) (*DistributionWidgetHistogramRequestType, error) {
	ev := DistributionWidgetHistogramRequestType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for DistributionWidgetHistogramRequestType: valid values are %v", v, allowedDistributionWidgetHistogramRequestTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v DistributionWidgetHistogramRequestType) IsValid() bool {
	for _, existing := range allowedDistributionWidgetHistogramRequestTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to DistributionWidgetHistogramRequestType value.
func (v DistributionWidgetHistogramRequestType) Ptr() *DistributionWidgetHistogramRequestType {
	return &v
}

// NullableDistributionWidgetHistogramRequestType handles when a null is used for DistributionWidgetHistogramRequestType.
type NullableDistributionWidgetHistogramRequestType struct {
	value *DistributionWidgetHistogramRequestType
	isSet bool
}

// Get returns the associated value.
func (v NullableDistributionWidgetHistogramRequestType) Get() *DistributionWidgetHistogramRequestType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableDistributionWidgetHistogramRequestType) Set(val *DistributionWidgetHistogramRequestType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableDistributionWidgetHistogramRequestType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableDistributionWidgetHistogramRequestType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableDistributionWidgetHistogramRequestType initializes the struct as if Set has been called.
func NewNullableDistributionWidgetHistogramRequestType(val *DistributionWidgetHistogramRequestType) *NullableDistributionWidgetHistogramRequestType {
	return &NullableDistributionWidgetHistogramRequestType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableDistributionWidgetHistogramRequestType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableDistributionWidgetHistogramRequestType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
