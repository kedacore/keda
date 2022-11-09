// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsAssertion - Object describing the assertions type, their associated operator,
// which property they apply, and upon which target.
type SyntheticsAssertion struct {
	SyntheticsAssertionTarget         *SyntheticsAssertionTarget
	SyntheticsAssertionJSONPathTarget *SyntheticsAssertionJSONPathTarget

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// SyntheticsAssertionTargetAsSyntheticsAssertion is a convenience function that returns SyntheticsAssertionTarget wrapped in SyntheticsAssertion.
func SyntheticsAssertionTargetAsSyntheticsAssertion(v *SyntheticsAssertionTarget) SyntheticsAssertion {
	return SyntheticsAssertion{SyntheticsAssertionTarget: v}
}

// SyntheticsAssertionJSONPathTargetAsSyntheticsAssertion is a convenience function that returns SyntheticsAssertionJSONPathTarget wrapped in SyntheticsAssertion.
func SyntheticsAssertionJSONPathTargetAsSyntheticsAssertion(v *SyntheticsAssertionJSONPathTarget) SyntheticsAssertion {
	return SyntheticsAssertion{SyntheticsAssertionJSONPathTarget: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *SyntheticsAssertion) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into SyntheticsAssertionTarget
	err = json.Unmarshal(data, &obj.SyntheticsAssertionTarget)
	if err == nil {
		if obj.SyntheticsAssertionTarget != nil && obj.SyntheticsAssertionTarget.UnparsedObject == nil {
			jsonSyntheticsAssertionTarget, _ := json.Marshal(obj.SyntheticsAssertionTarget)
			if string(jsonSyntheticsAssertionTarget) == "{}" { // empty struct
				obj.SyntheticsAssertionTarget = nil
			} else {
				match++
			}
		} else {
			obj.SyntheticsAssertionTarget = nil
		}
	} else {
		obj.SyntheticsAssertionTarget = nil
	}

	// try to unmarshal data into SyntheticsAssertionJSONPathTarget
	err = json.Unmarshal(data, &obj.SyntheticsAssertionJSONPathTarget)
	if err == nil {
		if obj.SyntheticsAssertionJSONPathTarget != nil && obj.SyntheticsAssertionJSONPathTarget.UnparsedObject == nil {
			jsonSyntheticsAssertionJSONPathTarget, _ := json.Marshal(obj.SyntheticsAssertionJSONPathTarget)
			if string(jsonSyntheticsAssertionJSONPathTarget) == "{}" { // empty struct
				obj.SyntheticsAssertionJSONPathTarget = nil
			} else {
				match++
			}
		} else {
			obj.SyntheticsAssertionJSONPathTarget = nil
		}
	} else {
		obj.SyntheticsAssertionJSONPathTarget = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.SyntheticsAssertionTarget = nil
		obj.SyntheticsAssertionJSONPathTarget = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj SyntheticsAssertion) MarshalJSON() ([]byte, error) {
	if obj.SyntheticsAssertionTarget != nil {
		return json.Marshal(&obj.SyntheticsAssertionTarget)
	}

	if obj.SyntheticsAssertionJSONPathTarget != nil {
		return json.Marshal(&obj.SyntheticsAssertionJSONPathTarget)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *SyntheticsAssertion) GetActualInstance() interface{} {
	if obj.SyntheticsAssertionTarget != nil {
		return obj.SyntheticsAssertionTarget
	}

	if obj.SyntheticsAssertionJSONPathTarget != nil {
		return obj.SyntheticsAssertionJSONPathTarget
	}

	// all schemas are nil
	return nil
}

// NullableSyntheticsAssertion handles when a null is used for SyntheticsAssertion.
type NullableSyntheticsAssertion struct {
	value *SyntheticsAssertion
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsAssertion) Get() *SyntheticsAssertion {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsAssertion) Set(val *SyntheticsAssertion) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsAssertion) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableSyntheticsAssertion) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsAssertion initializes the struct as if Set has been called.
func NewNullableSyntheticsAssertion(val *SyntheticsAssertion) *NullableSyntheticsAssertion {
	return &NullableSyntheticsAssertion{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsAssertion) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsAssertion) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
