// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// MonitorFormulaAndFunctionQueryDefinition - A formula and function query.
type MonitorFormulaAndFunctionQueryDefinition struct {
	MonitorFormulaAndFunctionEventQueryDefinition *MonitorFormulaAndFunctionEventQueryDefinition

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// MonitorFormulaAndFunctionEventQueryDefinitionAsMonitorFormulaAndFunctionQueryDefinition is a convenience function that returns MonitorFormulaAndFunctionEventQueryDefinition wrapped in MonitorFormulaAndFunctionQueryDefinition.
func MonitorFormulaAndFunctionEventQueryDefinitionAsMonitorFormulaAndFunctionQueryDefinition(v *MonitorFormulaAndFunctionEventQueryDefinition) MonitorFormulaAndFunctionQueryDefinition {
	return MonitorFormulaAndFunctionQueryDefinition{MonitorFormulaAndFunctionEventQueryDefinition: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *MonitorFormulaAndFunctionQueryDefinition) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into MonitorFormulaAndFunctionEventQueryDefinition
	err = json.Unmarshal(data, &obj.MonitorFormulaAndFunctionEventQueryDefinition)
	if err == nil {
		if obj.MonitorFormulaAndFunctionEventQueryDefinition != nil && obj.MonitorFormulaAndFunctionEventQueryDefinition.UnparsedObject == nil {
			jsonMonitorFormulaAndFunctionEventQueryDefinition, _ := json.Marshal(obj.MonitorFormulaAndFunctionEventQueryDefinition)
			if string(jsonMonitorFormulaAndFunctionEventQueryDefinition) == "{}" { // empty struct
				obj.MonitorFormulaAndFunctionEventQueryDefinition = nil
			} else {
				match++
			}
		} else {
			obj.MonitorFormulaAndFunctionEventQueryDefinition = nil
		}
	} else {
		obj.MonitorFormulaAndFunctionEventQueryDefinition = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.MonitorFormulaAndFunctionEventQueryDefinition = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj MonitorFormulaAndFunctionQueryDefinition) MarshalJSON() ([]byte, error) {
	if obj.MonitorFormulaAndFunctionEventQueryDefinition != nil {
		return json.Marshal(&obj.MonitorFormulaAndFunctionEventQueryDefinition)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *MonitorFormulaAndFunctionQueryDefinition) GetActualInstance() interface{} {
	if obj.MonitorFormulaAndFunctionEventQueryDefinition != nil {
		return obj.MonitorFormulaAndFunctionEventQueryDefinition
	}

	// all schemas are nil
	return nil
}

// NullableMonitorFormulaAndFunctionQueryDefinition handles when a null is used for MonitorFormulaAndFunctionQueryDefinition.
type NullableMonitorFormulaAndFunctionQueryDefinition struct {
	value *MonitorFormulaAndFunctionQueryDefinition
	isSet bool
}

// Get returns the associated value.
func (v NullableMonitorFormulaAndFunctionQueryDefinition) Get() *MonitorFormulaAndFunctionQueryDefinition {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMonitorFormulaAndFunctionQueryDefinition) Set(val *MonitorFormulaAndFunctionQueryDefinition) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMonitorFormulaAndFunctionQueryDefinition) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableMonitorFormulaAndFunctionQueryDefinition) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMonitorFormulaAndFunctionQueryDefinition initializes the struct as if Set has been called.
func NewNullableMonitorFormulaAndFunctionQueryDefinition(val *MonitorFormulaAndFunctionQueryDefinition) *NullableMonitorFormulaAndFunctionQueryDefinition {
	return &NullableMonitorFormulaAndFunctionQueryDefinition{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMonitorFormulaAndFunctionQueryDefinition) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMonitorFormulaAndFunctionQueryDefinition) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
