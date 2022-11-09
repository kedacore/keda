// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// NoteWidgetDefinitionType Type of the note widget.
type NoteWidgetDefinitionType string

// List of NoteWidgetDefinitionType.
const (
	NOTEWIDGETDEFINITIONTYPE_NOTE NoteWidgetDefinitionType = "note"
)

var allowedNoteWidgetDefinitionTypeEnumValues = []NoteWidgetDefinitionType{
	NOTEWIDGETDEFINITIONTYPE_NOTE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *NoteWidgetDefinitionType) GetAllowedValues() []NoteWidgetDefinitionType {
	return allowedNoteWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *NoteWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = NoteWidgetDefinitionType(value)
	return nil
}

// NewNoteWidgetDefinitionTypeFromValue returns a pointer to a valid NoteWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewNoteWidgetDefinitionTypeFromValue(v string) (*NoteWidgetDefinitionType, error) {
	ev := NoteWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for NoteWidgetDefinitionType: valid values are %v", v, allowedNoteWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v NoteWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedNoteWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to NoteWidgetDefinitionType value.
func (v NoteWidgetDefinitionType) Ptr() *NoteWidgetDefinitionType {
	return &v
}

// NullableNoteWidgetDefinitionType handles when a null is used for NoteWidgetDefinitionType.
type NullableNoteWidgetDefinitionType struct {
	value *NoteWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableNoteWidgetDefinitionType) Get() *NoteWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableNoteWidgetDefinitionType) Set(val *NoteWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableNoteWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableNoteWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableNoteWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableNoteWidgetDefinitionType(val *NoteWidgetDefinitionType) *NullableNoteWidgetDefinitionType {
	return &NullableNoteWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableNoteWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableNoteWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
