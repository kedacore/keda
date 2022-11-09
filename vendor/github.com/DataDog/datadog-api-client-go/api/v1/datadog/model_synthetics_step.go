// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsStep The steps used in a Synthetics browser test.
type SyntheticsStep struct {
	// A boolean set to allow this step to fail.
	AllowFailure *bool `json:"allowFailure,omitempty"`
	// A boolean to use in addition to `allowFailure` to determine if the test should be marked as failed when the step fails.
	IsCritical *bool `json:"isCritical,omitempty"`
	// The name of the step.
	Name *string `json:"name,omitempty"`
	// The parameters of the step.
	Params interface{} `json:"params,omitempty"`
	// The time before declaring a step failed.
	Timeout *int64 `json:"timeout,omitempty"`
	// Step type used in your Synthetic test.
	Type *SyntheticsStepType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsStep instantiates a new SyntheticsStep object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsStep() *SyntheticsStep {
	this := SyntheticsStep{}
	return &this
}

// NewSyntheticsStepWithDefaults instantiates a new SyntheticsStep object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsStepWithDefaults() *SyntheticsStep {
	this := SyntheticsStep{}
	return &this
}

// GetAllowFailure returns the AllowFailure field value if set, zero value otherwise.
func (o *SyntheticsStep) GetAllowFailure() bool {
	if o == nil || o.AllowFailure == nil {
		var ret bool
		return ret
	}
	return *o.AllowFailure
}

// GetAllowFailureOk returns a tuple with the AllowFailure field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsStep) GetAllowFailureOk() (*bool, bool) {
	if o == nil || o.AllowFailure == nil {
		return nil, false
	}
	return o.AllowFailure, true
}

// HasAllowFailure returns a boolean if a field has been set.
func (o *SyntheticsStep) HasAllowFailure() bool {
	if o != nil && o.AllowFailure != nil {
		return true
	}

	return false
}

// SetAllowFailure gets a reference to the given bool and assigns it to the AllowFailure field.
func (o *SyntheticsStep) SetAllowFailure(v bool) {
	o.AllowFailure = &v
}

// GetIsCritical returns the IsCritical field value if set, zero value otherwise.
func (o *SyntheticsStep) GetIsCritical() bool {
	if o == nil || o.IsCritical == nil {
		var ret bool
		return ret
	}
	return *o.IsCritical
}

// GetIsCriticalOk returns a tuple with the IsCritical field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsStep) GetIsCriticalOk() (*bool, bool) {
	if o == nil || o.IsCritical == nil {
		return nil, false
	}
	return o.IsCritical, true
}

// HasIsCritical returns a boolean if a field has been set.
func (o *SyntheticsStep) HasIsCritical() bool {
	if o != nil && o.IsCritical != nil {
		return true
	}

	return false
}

// SetIsCritical gets a reference to the given bool and assigns it to the IsCritical field.
func (o *SyntheticsStep) SetIsCritical(v bool) {
	o.IsCritical = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *SyntheticsStep) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsStep) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *SyntheticsStep) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *SyntheticsStep) SetName(v string) {
	o.Name = &v
}

// GetParams returns the Params field value if set, zero value otherwise.
func (o *SyntheticsStep) GetParams() interface{} {
	if o == nil || o.Params == nil {
		var ret interface{}
		return ret
	}
	return o.Params
}

// GetParamsOk returns a tuple with the Params field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsStep) GetParamsOk() (*interface{}, bool) {
	if o == nil || o.Params == nil {
		return nil, false
	}
	return &o.Params, true
}

// HasParams returns a boolean if a field has been set.
func (o *SyntheticsStep) HasParams() bool {
	if o != nil && o.Params != nil {
		return true
	}

	return false
}

// SetParams gets a reference to the given interface{} and assigns it to the Params field.
func (o *SyntheticsStep) SetParams(v interface{}) {
	o.Params = v
}

// GetTimeout returns the Timeout field value if set, zero value otherwise.
func (o *SyntheticsStep) GetTimeout() int64 {
	if o == nil || o.Timeout == nil {
		var ret int64
		return ret
	}
	return *o.Timeout
}

// GetTimeoutOk returns a tuple with the Timeout field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsStep) GetTimeoutOk() (*int64, bool) {
	if o == nil || o.Timeout == nil {
		return nil, false
	}
	return o.Timeout, true
}

// HasTimeout returns a boolean if a field has been set.
func (o *SyntheticsStep) HasTimeout() bool {
	if o != nil && o.Timeout != nil {
		return true
	}

	return false
}

// SetTimeout gets a reference to the given int64 and assigns it to the Timeout field.
func (o *SyntheticsStep) SetTimeout(v int64) {
	o.Timeout = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *SyntheticsStep) GetType() SyntheticsStepType {
	if o == nil || o.Type == nil {
		var ret SyntheticsStepType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsStep) GetTypeOk() (*SyntheticsStepType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *SyntheticsStep) HasType() bool {
	if o != nil && o.Type != nil {
		return true
	}

	return false
}

// SetType gets a reference to the given SyntheticsStepType and assigns it to the Type field.
func (o *SyntheticsStep) SetType(v SyntheticsStepType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsStep) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AllowFailure != nil {
		toSerialize["allowFailure"] = o.AllowFailure
	}
	if o.IsCritical != nil {
		toSerialize["isCritical"] = o.IsCritical
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Params != nil {
		toSerialize["params"] = o.Params
	}
	if o.Timeout != nil {
		toSerialize["timeout"] = o.Timeout
	}
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsStep) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AllowFailure *bool               `json:"allowFailure,omitempty"`
		IsCritical   *bool               `json:"isCritical,omitempty"`
		Name         *string             `json:"name,omitempty"`
		Params       interface{}         `json:"params,omitempty"`
		Timeout      *int64              `json:"timeout,omitempty"`
		Type         *SyntheticsStepType `json:"type,omitempty"`
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
	if v := all.Type; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.AllowFailure = all.AllowFailure
	o.IsCritical = all.IsCritical
	o.Name = all.Name
	o.Params = all.Params
	o.Timeout = all.Timeout
	o.Type = all.Type
	return nil
}
