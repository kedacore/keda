// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// FormulaAndFunctionQueryDefinition - A formula and function query.
type FormulaAndFunctionQueryDefinition struct {
	FormulaAndFunctionMetricQueryDefinition             *FormulaAndFunctionMetricQueryDefinition
	FormulaAndFunctionEventQueryDefinition              *FormulaAndFunctionEventQueryDefinition
	FormulaAndFunctionProcessQueryDefinition            *FormulaAndFunctionProcessQueryDefinition
	FormulaAndFunctionApmDependencyStatsQueryDefinition *FormulaAndFunctionApmDependencyStatsQueryDefinition
	FormulaAndFunctionApmResourceStatsQueryDefinition   *FormulaAndFunctionApmResourceStatsQueryDefinition

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// FormulaAndFunctionMetricQueryDefinitionAsFormulaAndFunctionQueryDefinition is a convenience function that returns FormulaAndFunctionMetricQueryDefinition wrapped in FormulaAndFunctionQueryDefinition.
func FormulaAndFunctionMetricQueryDefinitionAsFormulaAndFunctionQueryDefinition(v *FormulaAndFunctionMetricQueryDefinition) FormulaAndFunctionQueryDefinition {
	return FormulaAndFunctionQueryDefinition{FormulaAndFunctionMetricQueryDefinition: v}
}

// FormulaAndFunctionEventQueryDefinitionAsFormulaAndFunctionQueryDefinition is a convenience function that returns FormulaAndFunctionEventQueryDefinition wrapped in FormulaAndFunctionQueryDefinition.
func FormulaAndFunctionEventQueryDefinitionAsFormulaAndFunctionQueryDefinition(v *FormulaAndFunctionEventQueryDefinition) FormulaAndFunctionQueryDefinition {
	return FormulaAndFunctionQueryDefinition{FormulaAndFunctionEventQueryDefinition: v}
}

// FormulaAndFunctionProcessQueryDefinitionAsFormulaAndFunctionQueryDefinition is a convenience function that returns FormulaAndFunctionProcessQueryDefinition wrapped in FormulaAndFunctionQueryDefinition.
func FormulaAndFunctionProcessQueryDefinitionAsFormulaAndFunctionQueryDefinition(v *FormulaAndFunctionProcessQueryDefinition) FormulaAndFunctionQueryDefinition {
	return FormulaAndFunctionQueryDefinition{FormulaAndFunctionProcessQueryDefinition: v}
}

// FormulaAndFunctionApmDependencyStatsQueryDefinitionAsFormulaAndFunctionQueryDefinition is a convenience function that returns FormulaAndFunctionApmDependencyStatsQueryDefinition wrapped in FormulaAndFunctionQueryDefinition.
func FormulaAndFunctionApmDependencyStatsQueryDefinitionAsFormulaAndFunctionQueryDefinition(v *FormulaAndFunctionApmDependencyStatsQueryDefinition) FormulaAndFunctionQueryDefinition {
	return FormulaAndFunctionQueryDefinition{FormulaAndFunctionApmDependencyStatsQueryDefinition: v}
}

// FormulaAndFunctionApmResourceStatsQueryDefinitionAsFormulaAndFunctionQueryDefinition is a convenience function that returns FormulaAndFunctionApmResourceStatsQueryDefinition wrapped in FormulaAndFunctionQueryDefinition.
func FormulaAndFunctionApmResourceStatsQueryDefinitionAsFormulaAndFunctionQueryDefinition(v *FormulaAndFunctionApmResourceStatsQueryDefinition) FormulaAndFunctionQueryDefinition {
	return FormulaAndFunctionQueryDefinition{FormulaAndFunctionApmResourceStatsQueryDefinition: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *FormulaAndFunctionQueryDefinition) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into FormulaAndFunctionMetricQueryDefinition
	err = json.Unmarshal(data, &obj.FormulaAndFunctionMetricQueryDefinition)
	if err == nil {
		if obj.FormulaAndFunctionMetricQueryDefinition != nil && obj.FormulaAndFunctionMetricQueryDefinition.UnparsedObject == nil {
			jsonFormulaAndFunctionMetricQueryDefinition, _ := json.Marshal(obj.FormulaAndFunctionMetricQueryDefinition)
			if string(jsonFormulaAndFunctionMetricQueryDefinition) == "{}" { // empty struct
				obj.FormulaAndFunctionMetricQueryDefinition = nil
			} else {
				match++
			}
		} else {
			obj.FormulaAndFunctionMetricQueryDefinition = nil
		}
	} else {
		obj.FormulaAndFunctionMetricQueryDefinition = nil
	}

	// try to unmarshal data into FormulaAndFunctionEventQueryDefinition
	err = json.Unmarshal(data, &obj.FormulaAndFunctionEventQueryDefinition)
	if err == nil {
		if obj.FormulaAndFunctionEventQueryDefinition != nil && obj.FormulaAndFunctionEventQueryDefinition.UnparsedObject == nil {
			jsonFormulaAndFunctionEventQueryDefinition, _ := json.Marshal(obj.FormulaAndFunctionEventQueryDefinition)
			if string(jsonFormulaAndFunctionEventQueryDefinition) == "{}" { // empty struct
				obj.FormulaAndFunctionEventQueryDefinition = nil
			} else {
				match++
			}
		} else {
			obj.FormulaAndFunctionEventQueryDefinition = nil
		}
	} else {
		obj.FormulaAndFunctionEventQueryDefinition = nil
	}

	// try to unmarshal data into FormulaAndFunctionProcessQueryDefinition
	err = json.Unmarshal(data, &obj.FormulaAndFunctionProcessQueryDefinition)
	if err == nil {
		if obj.FormulaAndFunctionProcessQueryDefinition != nil && obj.FormulaAndFunctionProcessQueryDefinition.UnparsedObject == nil {
			jsonFormulaAndFunctionProcessQueryDefinition, _ := json.Marshal(obj.FormulaAndFunctionProcessQueryDefinition)
			if string(jsonFormulaAndFunctionProcessQueryDefinition) == "{}" { // empty struct
				obj.FormulaAndFunctionProcessQueryDefinition = nil
			} else {
				match++
			}
		} else {
			obj.FormulaAndFunctionProcessQueryDefinition = nil
		}
	} else {
		obj.FormulaAndFunctionProcessQueryDefinition = nil
	}

	// try to unmarshal data into FormulaAndFunctionApmDependencyStatsQueryDefinition
	err = json.Unmarshal(data, &obj.FormulaAndFunctionApmDependencyStatsQueryDefinition)
	if err == nil {
		if obj.FormulaAndFunctionApmDependencyStatsQueryDefinition != nil && obj.FormulaAndFunctionApmDependencyStatsQueryDefinition.UnparsedObject == nil {
			jsonFormulaAndFunctionApmDependencyStatsQueryDefinition, _ := json.Marshal(obj.FormulaAndFunctionApmDependencyStatsQueryDefinition)
			if string(jsonFormulaAndFunctionApmDependencyStatsQueryDefinition) == "{}" { // empty struct
				obj.FormulaAndFunctionApmDependencyStatsQueryDefinition = nil
			} else {
				match++
			}
		} else {
			obj.FormulaAndFunctionApmDependencyStatsQueryDefinition = nil
		}
	} else {
		obj.FormulaAndFunctionApmDependencyStatsQueryDefinition = nil
	}

	// try to unmarshal data into FormulaAndFunctionApmResourceStatsQueryDefinition
	err = json.Unmarshal(data, &obj.FormulaAndFunctionApmResourceStatsQueryDefinition)
	if err == nil {
		if obj.FormulaAndFunctionApmResourceStatsQueryDefinition != nil && obj.FormulaAndFunctionApmResourceStatsQueryDefinition.UnparsedObject == nil {
			jsonFormulaAndFunctionApmResourceStatsQueryDefinition, _ := json.Marshal(obj.FormulaAndFunctionApmResourceStatsQueryDefinition)
			if string(jsonFormulaAndFunctionApmResourceStatsQueryDefinition) == "{}" { // empty struct
				obj.FormulaAndFunctionApmResourceStatsQueryDefinition = nil
			} else {
				match++
			}
		} else {
			obj.FormulaAndFunctionApmResourceStatsQueryDefinition = nil
		}
	} else {
		obj.FormulaAndFunctionApmResourceStatsQueryDefinition = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.FormulaAndFunctionMetricQueryDefinition = nil
		obj.FormulaAndFunctionEventQueryDefinition = nil
		obj.FormulaAndFunctionProcessQueryDefinition = nil
		obj.FormulaAndFunctionApmDependencyStatsQueryDefinition = nil
		obj.FormulaAndFunctionApmResourceStatsQueryDefinition = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj FormulaAndFunctionQueryDefinition) MarshalJSON() ([]byte, error) {
	if obj.FormulaAndFunctionMetricQueryDefinition != nil {
		return json.Marshal(&obj.FormulaAndFunctionMetricQueryDefinition)
	}

	if obj.FormulaAndFunctionEventQueryDefinition != nil {
		return json.Marshal(&obj.FormulaAndFunctionEventQueryDefinition)
	}

	if obj.FormulaAndFunctionProcessQueryDefinition != nil {
		return json.Marshal(&obj.FormulaAndFunctionProcessQueryDefinition)
	}

	if obj.FormulaAndFunctionApmDependencyStatsQueryDefinition != nil {
		return json.Marshal(&obj.FormulaAndFunctionApmDependencyStatsQueryDefinition)
	}

	if obj.FormulaAndFunctionApmResourceStatsQueryDefinition != nil {
		return json.Marshal(&obj.FormulaAndFunctionApmResourceStatsQueryDefinition)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *FormulaAndFunctionQueryDefinition) GetActualInstance() interface{} {
	if obj.FormulaAndFunctionMetricQueryDefinition != nil {
		return obj.FormulaAndFunctionMetricQueryDefinition
	}

	if obj.FormulaAndFunctionEventQueryDefinition != nil {
		return obj.FormulaAndFunctionEventQueryDefinition
	}

	if obj.FormulaAndFunctionProcessQueryDefinition != nil {
		return obj.FormulaAndFunctionProcessQueryDefinition
	}

	if obj.FormulaAndFunctionApmDependencyStatsQueryDefinition != nil {
		return obj.FormulaAndFunctionApmDependencyStatsQueryDefinition
	}

	if obj.FormulaAndFunctionApmResourceStatsQueryDefinition != nil {
		return obj.FormulaAndFunctionApmResourceStatsQueryDefinition
	}

	// all schemas are nil
	return nil
}

// NullableFormulaAndFunctionQueryDefinition handles when a null is used for FormulaAndFunctionQueryDefinition.
type NullableFormulaAndFunctionQueryDefinition struct {
	value *FormulaAndFunctionQueryDefinition
	isSet bool
}

// Get returns the associated value.
func (v NullableFormulaAndFunctionQueryDefinition) Get() *FormulaAndFunctionQueryDefinition {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableFormulaAndFunctionQueryDefinition) Set(val *FormulaAndFunctionQueryDefinition) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableFormulaAndFunctionQueryDefinition) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableFormulaAndFunctionQueryDefinition) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableFormulaAndFunctionQueryDefinition initializes the struct as if Set has been called.
func NewNullableFormulaAndFunctionQueryDefinition(val *FormulaAndFunctionQueryDefinition) *NullableFormulaAndFunctionQueryDefinition {
	return &NullableFormulaAndFunctionQueryDefinition{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableFormulaAndFunctionQueryDefinition) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableFormulaAndFunctionQueryDefinition) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
