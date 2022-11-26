// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBrowserErrorType Error type returned by a browser test.
type SyntheticsBrowserErrorType string

// List of SyntheticsBrowserErrorType.
const (
	SYNTHETICSBROWSERERRORTYPE_NETWORK SyntheticsBrowserErrorType = "network"
	SYNTHETICSBROWSERERRORTYPE_JS      SyntheticsBrowserErrorType = "js"
)

var allowedSyntheticsBrowserErrorTypeEnumValues = []SyntheticsBrowserErrorType{
	SYNTHETICSBROWSERERRORTYPE_NETWORK,
	SYNTHETICSBROWSERERRORTYPE_JS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsBrowserErrorType) GetAllowedValues() []SyntheticsBrowserErrorType {
	return allowedSyntheticsBrowserErrorTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsBrowserErrorType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsBrowserErrorType(value)
	return nil
}

// NewSyntheticsBrowserErrorTypeFromValue returns a pointer to a valid SyntheticsBrowserErrorType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsBrowserErrorTypeFromValue(v string) (*SyntheticsBrowserErrorType, error) {
	ev := SyntheticsBrowserErrorType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsBrowserErrorType: valid values are %v", v, allowedSyntheticsBrowserErrorTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsBrowserErrorType) IsValid() bool {
	for _, existing := range allowedSyntheticsBrowserErrorTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsBrowserErrorType value.
func (v SyntheticsBrowserErrorType) Ptr() *SyntheticsBrowserErrorType {
	return &v
}

// NullableSyntheticsBrowserErrorType handles when a null is used for SyntheticsBrowserErrorType.
type NullableSyntheticsBrowserErrorType struct {
	value *SyntheticsBrowserErrorType
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsBrowserErrorType) Get() *SyntheticsBrowserErrorType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsBrowserErrorType) Set(val *SyntheticsBrowserErrorType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsBrowserErrorType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsBrowserErrorType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsBrowserErrorType initializes the struct as if Set has been called.
func NewNullableSyntheticsBrowserErrorType(val *SyntheticsBrowserErrorType) *NullableSyntheticsBrowserErrorType {
	return &NullableSyntheticsBrowserErrorType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsBrowserErrorType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsBrowserErrorType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
