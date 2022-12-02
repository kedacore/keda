// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// FunnelRequestType Widget request type.
type FunnelRequestType string

// List of FunnelRequestType.
const (
	FUNNELREQUESTTYPE_FUNNEL FunnelRequestType = "funnel"
)

var allowedFunnelRequestTypeEnumValues = []FunnelRequestType{
	FUNNELREQUESTTYPE_FUNNEL,
}

// GetAllowedValues reeturns the list of possible values.
func (v *FunnelRequestType) GetAllowedValues() []FunnelRequestType {
	return allowedFunnelRequestTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *FunnelRequestType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = FunnelRequestType(value)
	return nil
}

// NewFunnelRequestTypeFromValue returns a pointer to a valid FunnelRequestType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewFunnelRequestTypeFromValue(v string) (*FunnelRequestType, error) {
	ev := FunnelRequestType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for FunnelRequestType: valid values are %v", v, allowedFunnelRequestTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v FunnelRequestType) IsValid() bool {
	for _, existing := range allowedFunnelRequestTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to FunnelRequestType value.
func (v FunnelRequestType) Ptr() *FunnelRequestType {
	return &v
}

// NullableFunnelRequestType handles when a null is used for FunnelRequestType.
type NullableFunnelRequestType struct {
	value *FunnelRequestType
	isSet bool
}

// Get returns the associated value.
func (v NullableFunnelRequestType) Get() *FunnelRequestType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableFunnelRequestType) Set(val *FunnelRequestType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableFunnelRequestType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableFunnelRequestType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableFunnelRequestType initializes the struct as if Set has been called.
func NewNullableFunnelRequestType(val *FunnelRequestType) *NullableFunnelRequestType {
	return &NullableFunnelRequestType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableFunnelRequestType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableFunnelRequestType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
