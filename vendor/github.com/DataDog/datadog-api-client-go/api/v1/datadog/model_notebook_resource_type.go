// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// NotebookResourceType Type of the Notebook resource.
type NotebookResourceType string

// List of NotebookResourceType.
const (
	NOTEBOOKRESOURCETYPE_NOTEBOOKS NotebookResourceType = "notebooks"
)

var allowedNotebookResourceTypeEnumValues = []NotebookResourceType{
	NOTEBOOKRESOURCETYPE_NOTEBOOKS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *NotebookResourceType) GetAllowedValues() []NotebookResourceType {
	return allowedNotebookResourceTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *NotebookResourceType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = NotebookResourceType(value)
	return nil
}

// NewNotebookResourceTypeFromValue returns a pointer to a valid NotebookResourceType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewNotebookResourceTypeFromValue(v string) (*NotebookResourceType, error) {
	ev := NotebookResourceType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for NotebookResourceType: valid values are %v", v, allowedNotebookResourceTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v NotebookResourceType) IsValid() bool {
	for _, existing := range allowedNotebookResourceTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to NotebookResourceType value.
func (v NotebookResourceType) Ptr() *NotebookResourceType {
	return &v
}

// NullableNotebookResourceType handles when a null is used for NotebookResourceType.
type NullableNotebookResourceType struct {
	value *NotebookResourceType
	isSet bool
}

// Get returns the associated value.
func (v NullableNotebookResourceType) Get() *NotebookResourceType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableNotebookResourceType) Set(val *NotebookResourceType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableNotebookResourceType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableNotebookResourceType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableNotebookResourceType initializes the struct as if Set has been called.
func NewNullableNotebookResourceType(val *NotebookResourceType) *NullableNotebookResourceType {
	return &NullableNotebookResourceType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableNotebookResourceType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableNotebookResourceType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
