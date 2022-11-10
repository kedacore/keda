// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ApmStatsQueryRowType The level of detail for the request.
type ApmStatsQueryRowType string

// List of ApmStatsQueryRowType.
const (
	APMSTATSQUERYROWTYPE_SERVICE  ApmStatsQueryRowType = "service"
	APMSTATSQUERYROWTYPE_RESOURCE ApmStatsQueryRowType = "resource"
	APMSTATSQUERYROWTYPE_SPAN     ApmStatsQueryRowType = "span"
)

var allowedApmStatsQueryRowTypeEnumValues = []ApmStatsQueryRowType{
	APMSTATSQUERYROWTYPE_SERVICE,
	APMSTATSQUERYROWTYPE_RESOURCE,
	APMSTATSQUERYROWTYPE_SPAN,
}

// GetAllowedValues reeturns the list of possible values.
func (v *ApmStatsQueryRowType) GetAllowedValues() []ApmStatsQueryRowType {
	return allowedApmStatsQueryRowTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *ApmStatsQueryRowType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = ApmStatsQueryRowType(value)
	return nil
}

// NewApmStatsQueryRowTypeFromValue returns a pointer to a valid ApmStatsQueryRowType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewApmStatsQueryRowTypeFromValue(v string) (*ApmStatsQueryRowType, error) {
	ev := ApmStatsQueryRowType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for ApmStatsQueryRowType: valid values are %v", v, allowedApmStatsQueryRowTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v ApmStatsQueryRowType) IsValid() bool {
	for _, existing := range allowedApmStatsQueryRowTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to ApmStatsQueryRowType value.
func (v ApmStatsQueryRowType) Ptr() *ApmStatsQueryRowType {
	return &v
}

// NullableApmStatsQueryRowType handles when a null is used for ApmStatsQueryRowType.
type NullableApmStatsQueryRowType struct {
	value *ApmStatsQueryRowType
	isSet bool
}

// Get returns the associated value.
func (v NullableApmStatsQueryRowType) Get() *ApmStatsQueryRowType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableApmStatsQueryRowType) Set(val *ApmStatsQueryRowType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableApmStatsQueryRowType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableApmStatsQueryRowType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableApmStatsQueryRowType initializes the struct as if Set has been called.
func NewNullableApmStatsQueryRowType(val *ApmStatsQueryRowType) *NullableApmStatsQueryRowType {
	return &NullableApmStatsQueryRowType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableApmStatsQueryRowType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableApmStatsQueryRowType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
