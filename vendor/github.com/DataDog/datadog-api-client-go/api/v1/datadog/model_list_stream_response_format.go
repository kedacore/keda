// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ListStreamResponseFormat Widget response format.
type ListStreamResponseFormat string

// List of ListStreamResponseFormat.
const (
	LISTSTREAMRESPONSEFORMAT_EVENT_LIST ListStreamResponseFormat = "event_list"
)

var allowedListStreamResponseFormatEnumValues = []ListStreamResponseFormat{
	LISTSTREAMRESPONSEFORMAT_EVENT_LIST,
}

// GetAllowedValues reeturns the list of possible values.
func (v *ListStreamResponseFormat) GetAllowedValues() []ListStreamResponseFormat {
	return allowedListStreamResponseFormatEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *ListStreamResponseFormat) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = ListStreamResponseFormat(value)
	return nil
}

// NewListStreamResponseFormatFromValue returns a pointer to a valid ListStreamResponseFormat
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewListStreamResponseFormatFromValue(v string) (*ListStreamResponseFormat, error) {
	ev := ListStreamResponseFormat(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for ListStreamResponseFormat: valid values are %v", v, allowedListStreamResponseFormatEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v ListStreamResponseFormat) IsValid() bool {
	for _, existing := range allowedListStreamResponseFormatEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to ListStreamResponseFormat value.
func (v ListStreamResponseFormat) Ptr() *ListStreamResponseFormat {
	return &v
}

// NullableListStreamResponseFormat handles when a null is used for ListStreamResponseFormat.
type NullableListStreamResponseFormat struct {
	value *ListStreamResponseFormat
	isSet bool
}

// Get returns the associated value.
func (v NullableListStreamResponseFormat) Get() *ListStreamResponseFormat {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableListStreamResponseFormat) Set(val *ListStreamResponseFormat) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableListStreamResponseFormat) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableListStreamResponseFormat) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableListStreamResponseFormat initializes the struct as if Set has been called.
func NewNullableListStreamResponseFormat(val *ListStreamResponseFormat) *NullableListStreamResponseFormat {
	return &NullableListStreamResponseFormat{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableListStreamResponseFormat) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableListStreamResponseFormat) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
