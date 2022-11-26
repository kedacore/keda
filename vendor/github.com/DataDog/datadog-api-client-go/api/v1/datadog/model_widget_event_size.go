// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetEventSize Size to use to display an event.
type WidgetEventSize string

// List of WidgetEventSize.
const (
	WIDGETEVENTSIZE_SMALL WidgetEventSize = "s"
	WIDGETEVENTSIZE_LARGE WidgetEventSize = "l"
)

var allowedWidgetEventSizeEnumValues = []WidgetEventSize{
	WIDGETEVENTSIZE_SMALL,
	WIDGETEVENTSIZE_LARGE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *WidgetEventSize) GetAllowedValues() []WidgetEventSize {
	return allowedWidgetEventSizeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *WidgetEventSize) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = WidgetEventSize(value)
	return nil
}

// NewWidgetEventSizeFromValue returns a pointer to a valid WidgetEventSize
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewWidgetEventSizeFromValue(v string) (*WidgetEventSize, error) {
	ev := WidgetEventSize(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for WidgetEventSize: valid values are %v", v, allowedWidgetEventSizeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v WidgetEventSize) IsValid() bool {
	for _, existing := range allowedWidgetEventSizeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to WidgetEventSize value.
func (v WidgetEventSize) Ptr() *WidgetEventSize {
	return &v
}

// NullableWidgetEventSize handles when a null is used for WidgetEventSize.
type NullableWidgetEventSize struct {
	value *WidgetEventSize
	isSet bool
}

// Get returns the associated value.
func (v NullableWidgetEventSize) Get() *WidgetEventSize {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableWidgetEventSize) Set(val *WidgetEventSize) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableWidgetEventSize) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableWidgetEventSize) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableWidgetEventSize initializes the struct as if Set has been called.
func NewNullableWidgetEventSize(val *WidgetEventSize) *NullableWidgetEventSize {
	return &NullableWidgetEventSize{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableWidgetEventSize) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableWidgetEventSize) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
