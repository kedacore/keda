// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// DistributionPointItem - List of distribution point.
type DistributionPointItem struct {
	DistributionPointTimestamp *float64
	DistributionPointData      *[]float64

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// DistributionPointTimestampAsDistributionPointItem is a convenience function that returns float64 wrapped in DistributionPointItem.
func DistributionPointTimestampAsDistributionPointItem(v *float64) DistributionPointItem {
	return DistributionPointItem{DistributionPointTimestamp: v}
}

// DistributionPointDataAsDistributionPointItem is a convenience function that returns []float64 wrapped in DistributionPointItem.
func DistributionPointDataAsDistributionPointItem(v *[]float64) DistributionPointItem {
	return DistributionPointItem{DistributionPointData: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *DistributionPointItem) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into DistributionPointTimestamp
	err = json.Unmarshal(data, &obj.DistributionPointTimestamp)
	if err == nil {
		if obj.DistributionPointTimestamp != nil {
			jsonDistributionPointTimestamp, _ := json.Marshal(obj.DistributionPointTimestamp)
			if string(jsonDistributionPointTimestamp) == "{}" { // empty struct
				obj.DistributionPointTimestamp = nil
			} else {
				match++
			}
		} else {
			obj.DistributionPointTimestamp = nil
		}
	} else {
		obj.DistributionPointTimestamp = nil
	}

	// try to unmarshal data into DistributionPointData
	err = json.Unmarshal(data, &obj.DistributionPointData)
	if err == nil {
		if obj.DistributionPointData != nil {
			jsonDistributionPointData, _ := json.Marshal(obj.DistributionPointData)
			if string(jsonDistributionPointData) == "{}" { // empty struct
				obj.DistributionPointData = nil
			} else {
				match++
			}
		} else {
			obj.DistributionPointData = nil
		}
	} else {
		obj.DistributionPointData = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.DistributionPointTimestamp = nil
		obj.DistributionPointData = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj DistributionPointItem) MarshalJSON() ([]byte, error) {
	if obj.DistributionPointTimestamp != nil {
		return json.Marshal(&obj.DistributionPointTimestamp)
	}

	if obj.DistributionPointData != nil {
		return json.Marshal(&obj.DistributionPointData)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *DistributionPointItem) GetActualInstance() interface{} {
	if obj.DistributionPointTimestamp != nil {
		return obj.DistributionPointTimestamp
	}

	if obj.DistributionPointData != nil {
		return obj.DistributionPointData
	}

	// all schemas are nil
	return nil
}

// NullableDistributionPointItem handles when a null is used for DistributionPointItem.
type NullableDistributionPointItem struct {
	value *DistributionPointItem
	isSet bool
}

// Get returns the associated value.
func (v NullableDistributionPointItem) Get() *DistributionPointItem {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableDistributionPointItem) Set(val *DistributionPointItem) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableDistributionPointItem) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableDistributionPointItem) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableDistributionPointItem initializes the struct as if Set has been called.
func NewNullableDistributionPointItem(val *DistributionPointItem) *NullableDistributionPointItem {
	return &NullableDistributionPointItem{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableDistributionPointItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableDistributionPointItem) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
