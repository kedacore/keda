// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsAssertionTarget An assertion which uses a simple target.
type SyntheticsAssertionTarget struct {
	// Assertion operator to apply.
	Operator SyntheticsAssertionOperator `json:"operator"`
	// The associated assertion property.
	Property *string `json:"property,omitempty"`
	// Value used by the operator.
	Target interface{} `json:"target"`
	// Type of the assertion.
	Type SyntheticsAssertionType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsAssertionTarget instantiates a new SyntheticsAssertionTarget object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsAssertionTarget(operator SyntheticsAssertionOperator, target interface{}, typeVar SyntheticsAssertionType) *SyntheticsAssertionTarget {
	this := SyntheticsAssertionTarget{}
	this.Operator = operator
	this.Target = target
	this.Type = typeVar
	return &this
}

// NewSyntheticsAssertionTargetWithDefaults instantiates a new SyntheticsAssertionTarget object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsAssertionTargetWithDefaults() *SyntheticsAssertionTarget {
	this := SyntheticsAssertionTarget{}
	return &this
}

// GetOperator returns the Operator field value.
func (o *SyntheticsAssertionTarget) GetOperator() SyntheticsAssertionOperator {
	if o == nil {
		var ret SyntheticsAssertionOperator
		return ret
	}
	return o.Operator
}

// GetOperatorOk returns a tuple with the Operator field value
// and a boolean to check if the value has been set.
func (o *SyntheticsAssertionTarget) GetOperatorOk() (*SyntheticsAssertionOperator, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Operator, true
}

// SetOperator sets field value.
func (o *SyntheticsAssertionTarget) SetOperator(v SyntheticsAssertionOperator) {
	o.Operator = v
}

// GetProperty returns the Property field value if set, zero value otherwise.
func (o *SyntheticsAssertionTarget) GetProperty() string {
	if o == nil || o.Property == nil {
		var ret string
		return ret
	}
	return *o.Property
}

// GetPropertyOk returns a tuple with the Property field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAssertionTarget) GetPropertyOk() (*string, bool) {
	if o == nil || o.Property == nil {
		return nil, false
	}
	return o.Property, true
}

// HasProperty returns a boolean if a field has been set.
func (o *SyntheticsAssertionTarget) HasProperty() bool {
	if o != nil && o.Property != nil {
		return true
	}

	return false
}

// SetProperty gets a reference to the given string and assigns it to the Property field.
func (o *SyntheticsAssertionTarget) SetProperty(v string) {
	o.Property = &v
}

// GetTarget returns the Target field value.
func (o *SyntheticsAssertionTarget) GetTarget() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}
	return o.Target
}

// GetTargetOk returns a tuple with the Target field value
// and a boolean to check if the value has been set.
func (o *SyntheticsAssertionTarget) GetTargetOk() (*interface{}, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Target, true
}

// SetTarget sets field value.
func (o *SyntheticsAssertionTarget) SetTarget(v interface{}) {
	o.Target = v
}

// GetType returns the Type field value.
func (o *SyntheticsAssertionTarget) GetType() SyntheticsAssertionType {
	if o == nil {
		var ret SyntheticsAssertionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *SyntheticsAssertionTarget) GetTypeOk() (*SyntheticsAssertionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *SyntheticsAssertionTarget) SetType(v SyntheticsAssertionType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsAssertionTarget) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["operator"] = o.Operator
	if o.Property != nil {
		toSerialize["property"] = o.Property
	}
	toSerialize["target"] = o.Target
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsAssertionTarget) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Operator *SyntheticsAssertionOperator `json:"operator"`
		Target   *interface{}                 `json:"target"`
		Type     *SyntheticsAssertionType     `json:"type"`
	}{}
	all := struct {
		Operator SyntheticsAssertionOperator `json:"operator"`
		Property *string                     `json:"property,omitempty"`
		Target   interface{}                 `json:"target"`
		Type     SyntheticsAssertionType     `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Operator == nil {
		return fmt.Errorf("Required field operator missing")
	}
	if required.Target == nil {
		return fmt.Errorf("Required field target missing")
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
	o.Target = all.Target
	o.Type = all.Type
	return nil
}
