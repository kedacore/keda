// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// NotebookCellResponseAttributes - The attributes of a notebook cell response. Valid cell types are `markdown`, `timeseries`, `toplist`, `heatmap`, `distribution`,
// `log_stream`. [More information on each graph visualization type.](https://docs.datadoghq.com/dashboards/widgets/)
type NotebookCellResponseAttributes struct {
	NotebookMarkdownCellAttributes     *NotebookMarkdownCellAttributes
	NotebookTimeseriesCellAttributes   *NotebookTimeseriesCellAttributes
	NotebookToplistCellAttributes      *NotebookToplistCellAttributes
	NotebookHeatMapCellAttributes      *NotebookHeatMapCellAttributes
	NotebookDistributionCellAttributes *NotebookDistributionCellAttributes
	NotebookLogStreamCellAttributes    *NotebookLogStreamCellAttributes

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// NotebookMarkdownCellAttributesAsNotebookCellResponseAttributes is a convenience function that returns NotebookMarkdownCellAttributes wrapped in NotebookCellResponseAttributes.
func NotebookMarkdownCellAttributesAsNotebookCellResponseAttributes(v *NotebookMarkdownCellAttributes) NotebookCellResponseAttributes {
	return NotebookCellResponseAttributes{NotebookMarkdownCellAttributes: v}
}

// NotebookTimeseriesCellAttributesAsNotebookCellResponseAttributes is a convenience function that returns NotebookTimeseriesCellAttributes wrapped in NotebookCellResponseAttributes.
func NotebookTimeseriesCellAttributesAsNotebookCellResponseAttributes(v *NotebookTimeseriesCellAttributes) NotebookCellResponseAttributes {
	return NotebookCellResponseAttributes{NotebookTimeseriesCellAttributes: v}
}

// NotebookToplistCellAttributesAsNotebookCellResponseAttributes is a convenience function that returns NotebookToplistCellAttributes wrapped in NotebookCellResponseAttributes.
func NotebookToplistCellAttributesAsNotebookCellResponseAttributes(v *NotebookToplistCellAttributes) NotebookCellResponseAttributes {
	return NotebookCellResponseAttributes{NotebookToplistCellAttributes: v}
}

// NotebookHeatMapCellAttributesAsNotebookCellResponseAttributes is a convenience function that returns NotebookHeatMapCellAttributes wrapped in NotebookCellResponseAttributes.
func NotebookHeatMapCellAttributesAsNotebookCellResponseAttributes(v *NotebookHeatMapCellAttributes) NotebookCellResponseAttributes {
	return NotebookCellResponseAttributes{NotebookHeatMapCellAttributes: v}
}

// NotebookDistributionCellAttributesAsNotebookCellResponseAttributes is a convenience function that returns NotebookDistributionCellAttributes wrapped in NotebookCellResponseAttributes.
func NotebookDistributionCellAttributesAsNotebookCellResponseAttributes(v *NotebookDistributionCellAttributes) NotebookCellResponseAttributes {
	return NotebookCellResponseAttributes{NotebookDistributionCellAttributes: v}
}

// NotebookLogStreamCellAttributesAsNotebookCellResponseAttributes is a convenience function that returns NotebookLogStreamCellAttributes wrapped in NotebookCellResponseAttributes.
func NotebookLogStreamCellAttributesAsNotebookCellResponseAttributes(v *NotebookLogStreamCellAttributes) NotebookCellResponseAttributes {
	return NotebookCellResponseAttributes{NotebookLogStreamCellAttributes: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *NotebookCellResponseAttributes) UnmarshalJSON(data []byte) error {
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
func (obj NotebookCellResponseAttributes) MarshalJSON() ([]byte, error) {
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
func (obj *NotebookCellResponseAttributes) GetActualInstance() interface{} {
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

// NullableNotebookCellResponseAttributes handles when a null is used for NotebookCellResponseAttributes.
type NullableNotebookCellResponseAttributes struct {
	value *NotebookCellResponseAttributes
	isSet bool
}

// Get returns the associated value.
func (v NullableNotebookCellResponseAttributes) Get() *NotebookCellResponseAttributes {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableNotebookCellResponseAttributes) Set(val *NotebookCellResponseAttributes) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableNotebookCellResponseAttributes) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableNotebookCellResponseAttributes) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableNotebookCellResponseAttributes initializes the struct as if Set has been called.
func NewNullableNotebookCellResponseAttributes(val *NotebookCellResponseAttributes) *NullableNotebookCellResponseAttributes {
	return &NullableNotebookCellResponseAttributes{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableNotebookCellResponseAttributes) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableNotebookCellResponseAttributes) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
