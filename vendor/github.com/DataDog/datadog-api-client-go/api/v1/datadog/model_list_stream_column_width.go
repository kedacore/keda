// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ListStreamColumnWidth Widget column width.
type ListStreamColumnWidth string

// List of ListStreamColumnWidth.
const (
	LISTSTREAMCOLUMNWIDTH_AUTO    ListStreamColumnWidth = "auto"
	LISTSTREAMCOLUMNWIDTH_COMPACT ListStreamColumnWidth = "compact"
	LISTSTREAMCOLUMNWIDTH_FULL    ListStreamColumnWidth = "full"
)

var allowedListStreamColumnWidthEnumValues = []ListStreamColumnWidth{
	LISTSTREAMCOLUMNWIDTH_AUTO,
	LISTSTREAMCOLUMNWIDTH_COMPACT,
	LISTSTREAMCOLUMNWIDTH_FULL,
}

// GetAllowedValues reeturns the list of possible values.
func (v *ListStreamColumnWidth) GetAllowedValues() []ListStreamColumnWidth {
	return allowedListStreamColumnWidthEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *ListStreamColumnWidth) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = ListStreamColumnWidth(value)
	return nil
}

// NewListStreamColumnWidthFromValue returns a pointer to a valid ListStreamColumnWidth
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewListStreamColumnWidthFromValue(v string) (*ListStreamColumnWidth, error) {
	ev := ListStreamColumnWidth(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for ListStreamColumnWidth: valid values are %v", v, allowedListStreamColumnWidthEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v ListStreamColumnWidth) IsValid() bool {
	for _, existing := range allowedListStreamColumnWidthEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to ListStreamColumnWidth value.
func (v ListStreamColumnWidth) Ptr() *ListStreamColumnWidth {
	return &v
}

// NullableListStreamColumnWidth handles when a null is used for ListStreamColumnWidth.
type NullableListStreamColumnWidth struct {
	value *ListStreamColumnWidth
	isSet bool
}

// Get returns the associated value.
func (v NullableListStreamColumnWidth) Get() *ListStreamColumnWidth {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableListStreamColumnWidth) Set(val *ListStreamColumnWidth) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableListStreamColumnWidth) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableListStreamColumnWidth) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableListStreamColumnWidth initializes the struct as if Set has been called.
func NewNullableListStreamColumnWidth(val *ListStreamColumnWidth) *NullableListStreamColumnWidth {
	return &NullableListStreamColumnWidth{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableListStreamColumnWidth) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableListStreamColumnWidth) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
