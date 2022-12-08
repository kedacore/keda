// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsAssertionJSONPathTarget An assertion for the `validatesJSONPath` operator.
type SyntheticsAssertionJSONPathTarget struct {
	// Assertion operator to apply.
	Operator SyntheticsAssertionJSONPathOperator `json:"operator"`
	// The associated assertion property.
	Property *string `json:"property,omitempty"`
	// Composed target for `validatesJSONPath` operator.
	Target *SyntheticsAssertionJSONPathTargetTarget `json:"target,omitempty"`
	// Type of the assertion.
	Type SyntheticsAssertionType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsAssertionJSONPathTarget instantiates a new SyntheticsAssertionJSONPathTarget object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsAssertionJSONPathTarget(operator SyntheticsAssertionJSONPathOperator, typeVar SyntheticsAssertionType) *SyntheticsAssertionJSONPathTarget {
	this := SyntheticsAssertionJSONPathTarget{}
	this.Operator = operator
	this.Type = typeVar
	return &this
}

// NewSyntheticsAssertionJSONPathTargetWithDefaults instantiates a new SyntheticsAssertionJSONPathTarget object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsAssertionJSONPathTargetWithDefaults() *SyntheticsAssertionJSONPathTarget {
	this := SyntheticsAssertionJSONPathTarget{}
	return &this
}

// GetOperator returns the Operator field value.
func (o *SyntheticsAssertionJSONPathTarget) GetOperator() SyntheticsAssertionJSONPathOperator {
	if o == nil {
		var ret SyntheticsAssertionJSONPathOperator
		return ret
	}
	return o.Operator
}

// GetOperatorOk returns a tuple with the Operator field value
// and a boolean to check if the value has been set.
func (o *SyntheticsAssertionJSONPathTarget) GetOperatorOk() (*SyntheticsAssertionJSONPathOperator, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Operator, true
}

// SetOperator sets field value.
func (o *SyntheticsAssertionJSONPathTarget) SetOperator(v SyntheticsAssertionJSONPathOperator) {
	o.Operator = v
}

// GetProperty returns the Property field value if set, zero value otherwise.
func (o *SyntheticsAssertionJSONPathTarget) GetProperty() string {
	if o == nil || o.Property == nil {
		var ret string
		return ret
	}
	return *o.Property
}

// GetPropertyOk returns a tuple with the Property field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAssertionJSONPathTarget) GetPropertyOk() (*string, bool) {
	if o == nil || o.Property == nil {
		return nil, false
	}
	return o.Property, true
}

// HasProperty returns a boolean if a field has been set.
func (o *SyntheticsAssertionJSONPathTarget) HasProperty() bool {
	if o != nil && o.Property != nil {
		return true
	}

	return false
}

// SetProperty gets a reference to the given string and assigns it to the Property field.
func (o *SyntheticsAssertionJSONPathTarget) SetProperty(v string) {
	o.Property = &v
}

// GetTarget returns the Target field value if set, zero value otherwise.
func (o *SyntheticsAssertionJSONPathTarget) GetTarget() SyntheticsAssertionJSONPathTargetTarget {
	if o == nil || o.Target == nil {
		var ret SyntheticsAssertionJSONPathTargetTarget
		return ret
	}
	return *o.Target
}

// GetTargetOk returns a tuple with the Target field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAssertionJSONPathTarget) GetTargetOk() (*SyntheticsAssertionJSONPathTargetTarget, bool) {
	if o == nil || o.Target == nil {
		return nil, false
	}
	return o.Target, true
}

// HasTarget returns a boolean if a field has been set.
func (o *SyntheticsAssertionJSONPathTarget) HasTarget() bool {
	if o != nil && o.Target != nil {
		return true
	}

	return false
}

// SetTarget gets a reference to the given SyntheticsAssertionJSONPathTargetTarget and assigns it to the Target field.
func (o *SyntheticsAssertionJSONPathTarget) SetTarget(v SyntheticsAssertionJSONPathTargetTarget) {
	o.Target = &v
}

// GetType returns the Type field value.
func (o *SyntheticsAssertionJSONPathTarget) GetType() SyntheticsAssertionType {
	if o == nil {
		var ret SyntheticsAssertionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *SyntheticsAssertionJSONPathTarget) GetTypeOk() (*SyntheticsAssertionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *SyntheticsAssertionJSONPathTarget) SetType(v SyntheticsAssertionType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsAssertionJSONPathTarget) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["operator"] = o.Operator
	if o.Property != nil {
		toSerialize["property"] = o.Property
	}
	if o.Target != nil {
		toSerialize["target"] = o.Target
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsAssertionJSONPathTarget) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Operator *SyntheticsAssertionJSONPathOperator `json:"operator"`
		Type     *SyntheticsAssertionType             `json:"type"`
	}{}
	all := struct {
		Operator SyntheticsAssertionJSONPathOperator      `json:"operator"`
		Property *string                                  `json:"property,omitempty"`
		Target   *SyntheticsAssertionJSONPathTargetTarget `json:"target,omitempty"`
		Type     SyntheticsAssertionType                  `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Operator == nil {
		return fmt.Errorf("Required field operator missing")
	}
	if required.Type == nil {
		return fmt.Errorf("Required field type missing")
	}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Operator; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Type; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Operator = all.Operator
	o.Property = all.Property
	if all.Target != nil && all.Target.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Target = all.Target
	o.Type = all.Type
	return nil
}
