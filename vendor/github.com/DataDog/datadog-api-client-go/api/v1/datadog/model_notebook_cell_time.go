// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// NotebookCellTime - Timeframe for the notebook cell. When 'null', the notebook global time is used.
type NotebookCellTime struct {
	NotebookRelativeTime *NotebookRelativeTime
	NotebookAbsoluteTime *NotebookAbsoluteTime

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// NotebookRelativeTimeAsNotebookCellTime is a convenience function that returns NotebookRelativeTime wrapped in NotebookCellTime.
func NotebookRelativeTimeAsNotebookCellTime(v *NotebookRelativeTime) NotebookCellTime {
	return NotebookCellTime{NotebookRelativeTime: v}
}

// NotebookAbsoluteTimeAsNotebookCellTime is a convenience function that returns NotebookAbsoluteTime wrapped in NotebookCellTime.
func NotebookAbsoluteTimeAsNotebookCellTime(v *NotebookAbsoluteTime) NotebookCellTime {
	return NotebookCellTime{NotebookAbsoluteTime: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *NotebookCellTime) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into NotebookRelativeTime
	err = json.Unmarshal(data, &obj.NotebookRelativeTime)
	if err == nil {
		if obj.NotebookRelativeTime != nil && obj.NotebookRelativeTime.UnparsedObject == nil {
			jsonNotebookRelativeTime, _ := json.Marshal(obj.NotebookRelativeTime)
			if string(jsonNotebookRelativeTime) == "{}" { // empty struct
				obj.NotebookRelativeTime = nil
			} else {
				match++
			}
		} else {
			obj.NotebookRelativeTime = nil
		}
	} else {
		obj.NotebookRelativeTime = nil
	}

	// try to unmarshal data into NotebookAbsoluteTime
	err = json.Unmarshal(data, &obj.NotebookAbsoluteTime)
	if err == nil {
		if obj.NotebookAbsoluteTime != nil && obj.NotebookAbsoluteTime.UnparsedObject == nil {
			jsonNotebookAbsoluteTime, _ := json.Marshal(obj.NotebookAbsoluteTime)
			if string(jsonNotebookAbsoluteTime) == "{}" { // empty struct
				obj.NotebookAbsoluteTime = nil
			} else {
				match++
			}
		} else {
			obj.NotebookAbsoluteTime = nil
		}
	} else {
		obj.NotebookAbsoluteTime = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.NotebookRelativeTime = nil
		obj.NotebookAbsoluteTime = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj NotebookCellTime) MarshalJSON() ([]byte, error) {
	if obj.NotebookRelativeTime != nil {
		return json.Marshal(&obj.NotebookRelativeTime)
	}

	if obj.NotebookAbsoluteTime != nil {
		return json.Marshal(&obj.NotebookAbsoluteTime)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *NotebookCellTime) GetActualInstance() interface{} {
	if obj.NotebookRelativeTime != nil {
		return obj.NotebookRelativeTime
	}

	if obj.NotebookAbsoluteTime != nil {
		return obj.NotebookAbsoluteTime
	}

	// all schemas are nil
	return nil
}

// NullableNotebookCellTime handles when a null is used for NotebookCellTime.
type NullableNotebookCellTime struct {
	value *NotebookCellTime
	isSet bool
}

// Get returns the associated value.
func (v NullableNotebookCellTime) Get() *NotebookCellTime {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableNotebookCellTime) Set(val *NotebookCellTime) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableNotebookCellTime) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableNotebookCellTime) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableNotebookCellTime initializes the struct as if Set has been called.
func NewNullableNotebookCellTime(val *NotebookCellTime) *NullableNotebookCellTime {
	return &NullableNotebookCellTime{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableNotebookCellTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableNotebookCellTime) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
