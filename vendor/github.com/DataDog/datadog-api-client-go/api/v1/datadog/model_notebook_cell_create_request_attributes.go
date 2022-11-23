// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// NotebookCellCreateRequestAttributes - The attributes of a notebook cell in create cell request. Valid cell types are `markdown`, `timeseries`, `toplist`, `heatmap`, `distribution`,
// `log_stream`. [More information on each graph visualization type.](https://docs.datadoghq.com/dashboards/widgets/)
type NotebookCellCreateRequestAttributes struct {
	NotebookMarkdownCellAttributes     *NotebookMarkdownCellAttributes
	NotebookTimeseriesCellAttributes   *NotebookTimeseriesCellAttributes
	NotebookToplistCellAttributes      *NotebookToplistCellAttributes
	NotebookHeatMapCellAttributes      *NotebookHeatMapCellAttributes
	NotebookDistributionCellAttributes *NotebookDistributionCellAttributes
	NotebookLogStreamCellAttributes    *NotebookLogStreamCellAttributes

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// NotebookMarkdownCellAttributesAsNotebookCellCreateRequestAttributes is a convenience function that returns NotebookMarkdownCellAttributes wrapped in NotebookCellCreateRequestAttributes.
func NotebookMarkdownCellAttributesAsNotebookCellCreateRequestAttributes(v *NotebookMarkdownCellAttributes) NotebookCellCreateRequestAttributes {
	return NotebookCellCreateRequestAttributes{NotebookMarkdownCellAttributes: v}
}

// NotebookTimeseriesCellAttributesAsNotebookCellCreateRequestAttributes is a convenience function that returns NotebookTimeseriesCellAttributes wrapped in NotebookCellCreateRequestAttributes.
func NotebookTimeseriesCellAttributesAsNotebookCellCreateRequestAttributes(v *NotebookTimeseriesCellAttributes) NotebookCellCreateRequestAttributes {
	return NotebookCellCreateRequestAttributes{NotebookTimeseriesCellAttributes: v}
}

// NotebookToplistCellAttributesAsNotebookCellCreateRequestAttributes is a convenience function that returns NotebookToplistCellAttributes wrapped in NotebookCellCreateRequestAttributes.
func NotebookToplistCellAttributesAsNotebookCellCreateRequestAttributes(v *NotebookToplistCellAttributes) NotebookCellCreateRequestAttributes {
	return NotebookCellCreateRequestAttributes{NotebookToplistCellAttributes: v}
}

// NotebookHeatMapCellAttributesAsNotebookCellCreateRequestAttributes is a convenience function that returns NotebookHeatMapCellAttributes wrapped in NotebookCellCreateRequestAttributes.
func NotebookHeatMapCellAttributesAsNotebookCellCreateRequestAttributes(v *NotebookHeatMapCellAttributes) NotebookCellCreateRequestAttributes {
	return NotebookCellCreateRequestAttributes{NotebookHeatMapCellAttributes: v}
}

// NotebookDistributionCellAttributesAsNotebookCellCreateRequestAttributes is a convenience function that returns NotebookDistributionCellAttributes wrapped in NotebookCellCreateRequestAttributes.
func NotebookDistributionCellAttributesAsNotebookCellCreateRequestAttributes(v *NotebookDistributionCellAttributes) NotebookCellCreateRequestAttributes {
	return NotebookCellCreateRequestAttributes{NotebookDistributionCellAttributes: v}
}

// NotebookLogStreamCellAttributesAsNotebookCellCreateRequestAttributes is a convenience function that returns NotebookLogStreamCellAttributes wrapped in NotebookCellCreateRequestAttributes.
func NotebookLogStreamCellAttributesAsNotebookCellCreateRequestAttributes(v *NotebookLogStreamCellAttributes) NotebookCellCreateRequestAttributes {
	return NotebookCellCreateRequestAttributes{NotebookLogStreamCellAttributes: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *NotebookCellCreateRequestAttributes) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into NotebookMarkdownCellAttributes
	err = json.Unmarshal(data, &obj.NotebookMarkdownCellAttributes)
	if err == nil {
		if obj.NotebookMarkdownCellAttributes != nil && obj.NotebookMarkdownCellAttributes.UnparsedObject == nil {
			jsonNotebookMarkdownCellAttributes, _ := json.Marshal(obj.NotebookMarkdownCellAttributes)
			if string(jsonNotebookMarkdownCellAttributes) == "{}" { // empty struct
				obj.NotebookMarkdownCellAttributes = nil
			} else {
				match++
			}
		} else {
			obj.NotebookMarkdownCellAttributes = nil
		}
	} else {
		obj.NotebookMarkdownCellAttributes = nil
	}

	// try to unmarshal data into NotebookTimeseriesCellAttributes
	err = json.Unmarshal(data, &obj.NotebookTimeseriesCellAttributes)
	if err == nil {
		if obj.NotebookTimeseriesCellAttributes != nil && obj.NotebookTimeseriesCellAttributes.UnparsedObject == nil {
			jsonNotebookTimeseriesCellAttributes, _ := json.Marshal(obj.NotebookTimeseriesCellAttributes)
			if string(jsonNotebookTimeseriesCellAttributes) == "{}" { // empty struct
				obj.NotebookTimeseriesCellAttributes = nil
			} else {
				match++
			}
		} else {
			obj.NotebookTimeseriesCellAttributes = nil
		}
	} else {
		obj.NotebookTimeseriesCellAttributes = nil
	}

	// try to unmarshal data into NotebookToplistCellAttributes
	err = json.Unmarshal(data, &obj.NotebookToplistCellAttributes)
	if err == nil {
		if obj.NotebookToplistCellAttributes != nil && obj.NotebookToplistCellAttributes.UnparsedObject == nil {
			jsonNotebookToplistCellAttributes, _ := json.Marshal(obj.NotebookToplistCellAttributes)
			if string(jsonNotebookToplistCellAttributes) == "{}" { // empty struct
				obj.NotebookToplistCellAttributes = nil
			} else {
				match++
			}
		} else {
			obj.NotebookToplistCellAttributes = nil
		}
	} else {
		obj.NotebookToplistCellAttributes = nil
	}

	// try to unmarshal data into NotebookHeatMapCellAttributes
	err = json.Unmarshal(data, &obj.NotebookHeatMapCellAttributes)
	if err == nil {
		if obj.NotebookHeatMapCellAttributes != nil && obj.NotebookHeatMapCellAttributes.UnparsedObject == nil {
			jsonNotebookHeatMapCellAttributes, _ := json.Marshal(obj.NotebookHeatMapCellAttributes)
			if string(jsonNotebookHeatMapCellAttributes) == "{}" { // empty struct
				obj.NotebookHeatMapCellAttributes = nil
			} else {
				match++
			}
		} else {
			obj.NotebookHeatMapCellAttributes = nil
		}
	} else {
		obj.NotebookHeatMapCellAttributes = nil
	}

	// try to unmarshal data into NotebookDistributionCellAttributes
	err = json.Unmarshal(data, &obj.NotebookDistributionCellAttributes)
	if err == nil {
		if obj.NotebookDistributionCellAttributes != nil && obj.NotebookDistributionCellAttributes.UnparsedObject == nil {
			jsonNotebookDistributionCellAttributes, _ := json.Marshal(obj.NotebookDistributionCellAttributes)
			if string(jsonNotebookDistributionCellAttributes) == "{}" { // empty struct
				obj.NotebookDistributionCellAttributes = nil
			} else {
				match++
			}
		} else {
			obj.NotebookDistributionCellAttributes = nil
		}
	} else {
		obj.NotebookDistributionCellAttributes = nil
	}

	// try to unmarshal data into NotebookLogStreamCellAttributes
	err = json.Unmarshal(data, &obj.NotebookLogStreamCellAttributes)
	if err == nil {
		if obj.NotebookLogStreamCellAttributes != nil && obj.NotebookLogStreamCellAttributes.UnparsedObject == nil {
			jsonNotebookLogStreamCellAttributes, _ := json.Marshal(obj.NotebookLogStreamCellAttributes)
			if string(jsonNotebookLogStreamCellAttributes) == "{}" { // empty struct
				obj.NotebookLogStreamCellAttributes = nil
			} else {
				match++
			}
		} else {
			obj.NotebookLogStreamCellAttributes = nil
		}
	} else {
		obj.NotebookLogStreamCellAttributes = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.NotebookMarkdownCellAttributes = nil
		obj.NotebookTimeseriesCellAttributes = nil
		obj.NotebookToplistCellAttributes = nil
		obj.NotebookHeatMapCellAttributes = nil
		obj.NotebookDistributionCellAttributes = nil
		obj.NotebookLogStreamCellAttributes = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj NotebookCellCreateRequestAttributes) MarshalJSON() ([]byte, error) {
	if obj.NotebookMarkdownCellAttributes != nil {
		return json.Marshal(&obj.NotebookMarkdownCellAttributes)
	}

	if obj.NotebookTimeseriesCellAttributes != nil {
		return json.Marshal(&obj.NotebookTimeseriesCellAttributes)
	}

	if obj.NotebookToplistCellAttributes != nil {
		return json.Marshal(&obj.NotebookToplistCellAttributes)
	}

	if obj.NotebookHeatMapCellAttributes != nil {
		return json.Marshal(&obj.NotebookHeatMapCellAttributes)
	}

	if obj.NotebookDistributionCellAttributes != nil {
		return json.Marshal(&obj.NotebookDistributionCellAttributes)
	}

	if obj.NotebookLogStreamCellAttributes != nil {
		return json.Marshal(&obj.NotebookLogStreamCellAttributes)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *NotebookCellCreateRequestAttributes) GetActualInstance() interface{} {
	if obj.NotebookMarkdownCellAttributes != nil {
		return obj.NotebookMarkdownCellAttributes
	}

	if obj.NotebookTimeseriesCellAttributes != nil {
		return obj.NotebookTimeseriesCellAttributes
	}

	if obj.NotebookToplistCellAttributes != nil {
		return obj.NotebookToplistCellAttributes
	}

	if obj.NotebookHeatMapCellAttributes != nil {
		return obj.NotebookHeatMapCellAttributes
	}

	if obj.NotebookDistributionCellAttributes != nil {
		return obj.NotebookDistributionCellAttributes
	}

	if obj.NotebookLogStreamCellAttributes != nil {
		return obj.NotebookLogStreamCellAttributes
	}

	// all schemas are nil
	return nil
}

// NullableNotebookCellCreateRequestAttributes handles when a null is used for NotebookCellCreateRequestAttributes.
type NullableNotebookCellCreateRequestAttributes struct {
	value *NotebookCellCreateRequestAttributes
	isSet bool
}

// Get returns the associated value.
func (v NullableNotebookCellCreateRequestAttributes) Get() *NotebookCellCreateRequestAttributes {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableNotebookCellCreateRequestAttributes) Set(val *NotebookCellCreateRequestAttributes) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableNotebookCellCreateRequestAttributes) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableNotebookCellCreateRequestAttributes) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableNotebookCellCreateRequestAttributes initializes the struct as if Set has been called.
func NewNullableNotebookCellCreateRequestAttributes(val *NotebookCellCreateRequestAttributes) *NullableNotebookCellCreateRequestAttributes {
	return &NullableNotebookCellCreateRequestAttributes{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableNotebookCellCreateRequestAttributes) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableNotebookCellCreateRequestAttributes) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
