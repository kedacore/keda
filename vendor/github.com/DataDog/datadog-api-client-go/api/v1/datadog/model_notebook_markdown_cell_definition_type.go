// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// NotebookMarkdownCellDefinitionType Type of the markdown cell.
type NotebookMarkdownCellDefinitionType string

// List of NotebookMarkdownCellDefinitionType.
const (
	NOTEBOOKMARKDOWNCELLDEFINITIONTYPE_MARKDOWN NotebookMarkdownCellDefinitionType = "markdown"
)

var allowedNotebookMarkdownCellDefinitionTypeEnumValues = []NotebookMarkdownCellDefinitionType{
	NOTEBOOKMARKDOWNCELLDEFINITIONTYPE_MARKDOWN,
}

// GetAllowedValues reeturns the list of possible values.
func (v *NotebookMarkdownCellDefinitionType) GetAllowedValues() []NotebookMarkdownCellDefinitionType {
	return allowedNotebookMarkdownCellDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *NotebookMarkdownCellDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = NotebookMarkdownCellDefinitionType(value)
	return nil
}

// NewNotebookMarkdownCellDefinitionTypeFromValue returns a pointer to a valid NotebookMarkdownCellDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewNotebookMarkdownCellDefinitionTypeFromValue(v string) (*NotebookMarkdownCellDefinitionType, error) {
	ev := NotebookMarkdownCellDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for NotebookMarkdownCellDefinitionType: valid values are %v", v, allowedNotebookMarkdownCellDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v NotebookMarkdownCellDefinitionType) IsValid() bool {
	for _, existing := range allowedNotebookMarkdownCellDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to NotebookMarkdownCellDefinitionType value.
func (v NotebookMarkdownCellDefinitionType) Ptr() *NotebookMarkdownCellDefinitionType {
	return &v
}

// NullableNotebookMarkdownCellDefinitionType handles when a null is used for NotebookMarkdownCellDefinitionType.
type NullableNotebookMarkdownCellDefinitionType struct {
	value *NotebookMarkdownCellDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableNotebookMarkdownCellDefinitionType) Get() *NotebookMarkdownCellDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableNotebookMarkdownCellDefinitionType) Set(val *NotebookMarkdownCellDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableNotebookMarkdownCellDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableNotebookMarkdownCellDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableNotebookMarkdownCellDefinitionType initializes the struct as if Set has been called.
func NewNullableNotebookMarkdownCellDefinitionType(val *NotebookMarkdownCellDefinitionType) *NullableNotebookMarkdownCellDefinitionType {
	return &NullableNotebookMarkdownCellDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableNotebookMarkdownCellDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableNotebookMarkdownCellDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
