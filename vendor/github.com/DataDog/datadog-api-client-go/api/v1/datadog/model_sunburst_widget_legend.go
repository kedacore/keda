// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SunburstWidgetLegend - Configuration of the legend.
type SunburstWidgetLegend struct {
	SunburstWidgetLegendTable           *SunburstWidgetLegendTable
	SunburstWidgetLegendInlineAutomatic *SunburstWidgetLegendInlineAutomatic

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// SunburstWidgetLegendTableAsSunburstWidgetLegend is a convenience function that returns SunburstWidgetLegendTable wrapped in SunburstWidgetLegend.
func SunburstWidgetLegendTableAsSunburstWidgetLegend(v *SunburstWidgetLegendTable) SunburstWidgetLegend {
	return SunburstWidgetLegend{SunburstWidgetLegendTable: v}
}

// SunburstWidgetLegendInlineAutomaticAsSunburstWidgetLegend is a convenience function that returns SunburstWidgetLegendInlineAutomatic wrapped in SunburstWidgetLegend.
func SunburstWidgetLegendInlineAutomaticAsSunburstWidgetLegend(v *SunburstWidgetLegendInlineAutomatic) SunburstWidgetLegend {
	return SunburstWidgetLegend{SunburstWidgetLegendInlineAutomatic: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *SunburstWidgetLegend) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into SunburstWidgetLegendTable
	err = json.Unmarshal(data, &obj.SunburstWidgetLegendTable)
	if err == nil {
		if obj.SunburstWidgetLegendTable != nil && obj.SunburstWidgetLegendTable.UnparsedObject == nil {
			jsonSunburstWidgetLegendTable, _ := json.Marshal(obj.SunburstWidgetLegendTable)
			if string(jsonSunburstWidgetLegendTable) == "{}" { // empty struct
				obj.SunburstWidgetLegendTable = nil
			} else {
				match++
			}
		} else {
			obj.SunburstWidgetLegendTable = nil
		}
	} else {
		obj.SunburstWidgetLegendTable = nil
	}

	// try to unmarshal data into SunburstWidgetLegendInlineAutomatic
	err = json.Unmarshal(data, &obj.SunburstWidgetLegendInlineAutomatic)
	if err == nil {
		if obj.SunburstWidgetLegendInlineAutomatic != nil && obj.SunburstWidgetLegendInlineAutomatic.UnparsedObject == nil {
			jsonSunburstWidgetLegendInlineAutomatic, _ := json.Marshal(obj.SunburstWidgetLegendInlineAutomatic)
			if string(jsonSunburstWidgetLegendInlineAutomatic) == "{}" { // empty struct
				obj.SunburstWidgetLegendInlineAutomatic = nil
			} else {
				match++
			}
		} else {
			obj.SunburstWidgetLegendInlineAutomatic = nil
		}
	} else {
		obj.SunburstWidgetLegendInlineAutomatic = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.SunburstWidgetLegendTable = nil
		obj.SunburstWidgetLegendInlineAutomatic = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj SunburstWidgetLegend) MarshalJSON() ([]byte, error) {
	if obj.SunburstWidgetLegendTable != nil {
		return json.Marshal(&obj.SunburstWidgetLegendTable)
	}

	if obj.SunburstWidgetLegendInlineAutomatic != nil {
		return json.Marshal(&obj.SunburstWidgetLegendInlineAutomatic)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *SunburstWidgetLegend) GetActualInstance() interface{} {
	if obj.SunburstWidgetLegendTable != nil {
		return obj.SunburstWidgetLegendTable
	}

	if obj.SunburstWidgetLegendInlineAutomatic != nil {
		return obj.SunburstWidgetLegendInlineAutomatic
	}

	// all schemas are nil
	return nil
}

// NullableSunburstWidgetLegend handles when a null is used for SunburstWidgetLegend.
type NullableSunburstWidgetLegend struct {
	value *SunburstWidgetLegend
	isSet bool
}

// Get returns the associated value.
func (v NullableSunburstWidgetLegend) Get() *SunburstWidgetLegend {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSunburstWidgetLegend) Set(val *SunburstWidgetLegend) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSunburstWidgetLegend) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableSunburstWidgetLegend) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSunburstWidgetLegend initializes the struct as if Set has been called.
func NewNullableSunburstWidgetLegend(val *SunburstWidgetLegend) *NullableSunburstWidgetLegend {
	return &NullableSunburstWidgetLegend{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSunburstWidgetLegend) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSunburstWidgetLegend) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
