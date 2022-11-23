// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// NotebookStatus Publication status of the notebook. For now, always "published".
type NotebookStatus string

// List of NotebookStatus.
const (
	NOTEBOOKSTATUS_PUBLISHED NotebookStatus = "published"
)

var allowedNotebookStatusEnumValues = []NotebookStatus{
	NOTEBOOKSTATUS_PUBLISHED,
}

// GetAllowedValues reeturns the list of possible values.
func (v *NotebookStatus) GetAllowedValues() []NotebookStatus {
	return allowedNotebookStatusEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *NotebookStatus) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = NotebookStatus(value)
	return nil
}

// NewNotebookStatusFromValue returns a pointer to a valid NotebookStatus
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewNotebookStatusFromValue(v string) (*NotebookStatus, error) {
	ev := NotebookStatus(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for NotebookStatus: valid values are %v", v, allowedNotebookStatusEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v NotebookStatus) IsValid() bool {
	for _, existing := range allowedNotebookStatusEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to NotebookStatus value.
func (v NotebookStatus) Ptr() *NotebookStatus {
	return &v
}

// NullableNotebookStatus handles when a null is used for NotebookStatus.
type NullableNotebookStatus struct {
	value *NotebookStatus
	isSet bool
}

// Get returns the associated value.
func (v NullableNotebookStatus) Get() *NotebookStatus {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableNotebookStatus) Set(val *NotebookStatus) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableNotebookStatus) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableNotebookStatus) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableNotebookStatus initializes the struct as if Set has been called.
func NewNullableNotebookStatus(val *NotebookStatus) *NullableNotebookStatus {
	return &NullableNotebookStatus{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableNotebookStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableNotebookStatus) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
