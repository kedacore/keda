// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsTestCiOptions CI/CD options for a Synthetic test.
type SyntheticsTestCiOptions struct {
	// Execution rule for a Synthetics test.
	ExecutionRule *SyntheticsTestExecutionRule `json:"executionRule,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsTestCiOptions instantiates a new SyntheticsTestCiOptions object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsTestCiOptions() *SyntheticsTestCiOptions {
	this := SyntheticsTestCiOptions{}
	return &this
}

// NewSyntheticsTestCiOptionsWithDefaults instantiates a new SyntheticsTestCiOptions object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsTestCiOptionsWithDefaults() *SyntheticsTestCiOptions {
	this := SyntheticsTestCiOptions{}
	return &this
}

// GetExecutionRule returns the ExecutionRule field value if set, zero value otherwise.
func (o *SyntheticsTestCiOptions) GetExecutionRule() SyntheticsTestExecutionRule {
	if o == nil || o.ExecutionRule == nil {
		var ret SyntheticsTestExecutionRule
		return ret
	}
	return *o.ExecutionRule
}

// GetExecutionRuleOk returns a tuple with the ExecutionRule field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestCiOptions) GetExecutionRuleOk() (*SyntheticsTestExecutionRule, bool) {
	if o == nil || o.ExecutionRule == nil {
		return nil, false
	}
	return o.ExecutionRule, true
}

// HasExecutionRule returns a boolean if a field has been set.
func (o *SyntheticsTestCiOptions) HasExecutionRule() bool {
	if o != nil && o.ExecutionRule != nil {
		return true
	}

	return false
}

// SetExecutionRule gets a reference to the given SyntheticsTestExecutionRule and assigns it to the ExecutionRule field.
func (o *SyntheticsTestCiOptions) SetExecutionRule(v SyntheticsTestExecutionRule) {
	o.ExecutionRule = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsTestCiOptions) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ExecutionRule != nil {
		toSerialize["executionRule"] = o.ExecutionRule
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsTestCiOptions) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		ExecutionRule *SyntheticsTestExecutionRule `json:"executionRule,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.ExecutionRule; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.ExecutionRule = all.ExecutionRule
	return nil
}
