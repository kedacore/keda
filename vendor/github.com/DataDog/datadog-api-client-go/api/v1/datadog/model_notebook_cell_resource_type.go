// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// NotebookCellResourceType Type of the Notebook Cell resource.
type NotebookCellResourceType string

// List of NotebookCellResourceType.
const (
	NOTEBOOKCELLRESOURCETYPE_NOTEBOOK_CELLS NotebookCellResourceType = "notebook_cells"
)

var allowedNotebookCellResourceTypeEnumValues = []NotebookCellResourceType{
	NOTEBOOKCELLRESOURCETYPE_NOTEBOOK_CELLS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *NotebookCellResourceType) GetAllowedValues() []NotebookCellResourceType {
	return allowedNotebookCellResourceTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *NotebookCellResourceType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = NotebookCellResourceType(value)
	return nil
}

// NewNotebookCellResourceTypeFromValue returns a pointer to a valid NotebookCellResourceType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewNotebookCellResourceTypeFromValue(v string) (*NotebookCellResourceType, error) {
	ev := NotebookCellResourceType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for NotebookCellResourceType: valid values are %v", v, allowedNotebookCellResourceTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v NotebookCellResourceType) IsValid() bool {
	for _, existing := range allowedNotebookCellResourceTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to NotebookCellResourceType value.
func (v NotebookCellResourceType) Ptr() *NotebookCellResourceType {
	return &v
}

// NullableNotebookCellResourceType handles when a null is used for NotebookCellResourceType.
type NullableNotebookCellResourceType struct {
	value *NotebookCellResourceType
	isSet bool
}

// Get returns the associated value.
func (v NullableNotebookCellResourceType) Get() *NotebookCellResourceType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableNotebookCellResourceType) Set(val *NotebookCellResourceType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableNotebookCellResourceType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableNotebookCellResourceType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableNotebookCellResourceType initializes the struct as if Set has been called.
func NewNullableNotebookCellResourceType(val *NotebookCellResourceType) *NullableNotebookCellResourceType {
	return &NullableNotebookCellResourceType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableNotebookCellResourceType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableNotebookCellResourceType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
