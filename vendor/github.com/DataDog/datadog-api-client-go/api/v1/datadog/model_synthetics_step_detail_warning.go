// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsStepDetailWarning Object collecting warnings for a given step.
type SyntheticsStepDetailWarning struct {
	// Message for the warning.
	Message string `json:"message"`
	// User locator used.
	Type SyntheticsWarningType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsStepDetailWarning instantiates a new SyntheticsStepDetailWarning object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsStepDetailWarning(message string, typeVar SyntheticsWarningType) *SyntheticsStepDetailWarning {
	this := SyntheticsStepDetailWarning{}
	this.Message = message
	this.Type = typeVar
	return &this
}

// NewSyntheticsStepDetailWarningWithDefaults instantiates a new SyntheticsStepDetailWarning object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsStepDetailWarningWithDefaults() *SyntheticsStepDetailWarning {
	this := SyntheticsStepDetailWarning{}
	return &this
}

// GetMessage returns the Message field value.
func (o *SyntheticsStepDetailWarning) GetMessage() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Message
}

// GetMessageOk returns a tuple with the Message field value
// and a boolean to check if the value has been set.
func (o *SyntheticsStepDetailWarning) GetMessageOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Message, true
}

// SetMessage sets field value.
func (o *SyntheticsStepDetailWarning) SetMessage(v string) {
	o.Message = v
}

// GetType returns the Type field value.
func (o *SyntheticsStepDetailWarning) GetType() SyntheticsWarningType {
	if o == nil {
		var ret SyntheticsWarningType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *SyntheticsStepDetailWarning) GetTypeOk() (*SyntheticsWarningType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *SyntheticsStepDetailWarning) SetType(v SyntheticsWarningType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsStepDetailWarning) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["message"] = o.Message
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsStepDetailWarning) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Message *string                `json:"message"`
		Type    *SyntheticsWarningType `json:"type"`
	}{}
	all := struct {
		Message string                `json:"message"`
		Type    SyntheticsWarningType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Message == nil {
		return fmt.Errorf("Required field message missing")
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
	if v := all.Type; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Message = all.Message
	o.Type = all.Type
	return nil
}
