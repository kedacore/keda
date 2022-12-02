// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// NotebookUpdateCell - Updating a notebook can either insert new cell(s) or update existing cell(s) by including the cell `id`.
// To delete existing cell(s), simply omit it from the list of cells.
type NotebookUpdateCell struct {
	NotebookCellCreateRequest *NotebookCellCreateRequest
	NotebookCellUpdateRequest *NotebookCellUpdateRequest

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// NotebookCellCreateRequestAsNotebookUpdateCell is a convenience function that returns NotebookCellCreateRequest wrapped in NotebookUpdateCell.
func NotebookCellCreateRequestAsNotebookUpdateCell(v *NotebookCellCreateRequest) NotebookUpdateCell {
	return NotebookUpdateCell{NotebookCellCreateRequest: v}
}

// NotebookCellUpdateRequestAsNotebookUpdateCell is a convenience function that returns NotebookCellUpdateRequest wrapped in NotebookUpdateCell.
func NotebookCellUpdateRequestAsNotebookUpdateCell(v *NotebookCellUpdateRequest) NotebookUpdateCell {
	return NotebookUpdateCell{NotebookCellUpdateRequest: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *NotebookUpdateCell) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into NotebookCellCreateRequest
	err = json.Unmarshal(data, &obj.NotebookCellCreateRequest)
	if err == nil {
		if obj.NotebookCellCreateRequest != nil && obj.NotebookCellCreateRequest.UnparsedObject == nil {
			jsonNotebookCellCreateRequest, _ := json.Marshal(obj.NotebookCellCreateRequest)
			if string(jsonNotebookCellCreateRequest) == "{}" { // empty struct
				obj.NotebookCellCreateRequest = nil
			} else {
				match++
			}
		} else {
			obj.NotebookCellCreateRequest = nil
		}
	} else {
		obj.NotebookCellCreateRequest = nil
	}

	// try to unmarshal data into NotebookCellUpdateRequest
	err = json.Unmarshal(data, &obj.NotebookCellUpdateRequest)
	if err == nil {
		if obj.NotebookCellUpdateRequest != nil && obj.NotebookCellUpdateRequest.UnparsedObject == nil {
			jsonNotebookCellUpdateRequest, _ := json.Marshal(obj.NotebookCellUpdateRequest)
			if string(jsonNotebookCellUpdateRequest) == "{}" { // empty struct
				obj.NotebookCellUpdateRequest = nil
			} else {
				match++
			}
		} else {
			obj.NotebookCellUpdateRequest = nil
		}
	} else {
		obj.NotebookCellUpdateRequest = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.NotebookCellCreateRequest = nil
		obj.NotebookCellUpdateRequest = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj NotebookUpdateCell) MarshalJSON() ([]byte, error) {
	if obj.NotebookCellCreateRequest != nil {
		return json.Marshal(&obj.NotebookCellCreateRequest)
	}

	if obj.NotebookCellUpdateRequest != nil {
		return json.Marshal(&obj.NotebookCellUpdateRequest)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *NotebookUpdateCell) GetActualInstance() interface{} {
	if obj.NotebookCellCreateRequest != nil {
		return obj.NotebookCellCreateRequest
	}

	if obj.NotebookCellUpdateRequest != nil {
		return obj.NotebookCellUpdateRequest
	}

	// all schemas are nil
	return nil
}

// NullableNotebookUpdateCell handles when a null is used for NotebookUpdateCell.
type NullableNotebookUpdateCell struct {
	value *NotebookUpdateCell
	isSet bool
}

// Get returns the associated value.
func (v NullableNotebookUpdateCell) Get() *NotebookUpdateCell {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableNotebookUpdateCell) Set(val *NotebookUpdateCell) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableNotebookUpdateCell) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableNotebookUpdateCell) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableNotebookUpdateCell initializes the struct as if Set has been called.
func NewNullableNotebookUpdateCell(val *NotebookUpdateCell) *NullableNotebookUpdateCell {
	return &NullableNotebookUpdateCell{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableNotebookUpdateCell) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableNotebookUpdateCell) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
