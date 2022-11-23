// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// WebhooksIntegrationCustomVariableUpdateRequest Update request of a custom variable object.
//
// *All properties are optional.*
type WebhooksIntegrationCustomVariableUpdateRequest struct {
	// Make custom variable is secret or not.
	// If the custom variable is secret, the value is not returned in the response payload.
	IsSecret *bool `json:"is_secret,omitempty"`
	// The name of the variable. It corresponds with `<CUSTOM_VARIABLE_NAME>`. It must only contains upper-case characters, integers or underscores.
	Name *string `json:"name,omitempty"`
	// Value of the custom variable.
	Value *string `json:"value,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewWebhooksIntegrationCustomVariableUpdateRequest instantiates a new WebhooksIntegrationCustomVariableUpdateRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewWebhooksIntegrationCustomVariableUpdateRequest() *WebhooksIntegrationCustomVariableUpdateRequest {
	this := WebhooksIntegrationCustomVariableUpdateRequest{}
	return &this
}

// NewWebhooksIntegrationCustomVariableUpdateRequestWithDefaults instantiates a new WebhooksIntegrationCustomVariableUpdateRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewWebhooksIntegrationCustomVariableUpdateRequestWithDefaults() *WebhooksIntegrationCustomVariableUpdateRequest {
	this := WebhooksIntegrationCustomVariableUpdateRequest{}
	return &this
}

// GetIsSecret returns the IsSecret field value if set, zero value otherwise.
func (o *WebhooksIntegrationCustomVariableUpdateRequest) GetIsSecret() bool {
	if o == nil || o.IsSecret == nil {
		var ret bool
		return ret
	}
	return *o.IsSecret
}

// GetIsSecretOk returns a tuple with the IsSecret field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WebhooksIntegrationCustomVariableUpdateRequest) GetIsSecretOk() (*bool, bool) {
	if o == nil || o.IsSecret == nil {
		return nil, false
	}
	return o.IsSecret, true
}

// HasIsSecret returns a boolean if a field has been set.
func (o *WebhooksIntegrationCustomVariableUpdateRequest) HasIsSecret() bool {
	if o != nil && o.IsSecret != nil {
		return true
	}

	return false
}

// SetIsSecret gets a reference to the given bool and assigns it to the IsSecret field.
func (o *WebhooksIntegrationCustomVariableUpdateRequest) SetIsSecret(v bool) {
	o.IsSecret = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *WebhooksIntegrationCustomVariableUpdateRequest) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WebhooksIntegrationCustomVariableUpdateRequest) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *WebhooksIntegrationCustomVariableUpdateRequest) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *WebhooksIntegrationCustomVariableUpdateRequest) SetName(v string) {
	o.Name = &v
}

// GetValue returns the Value field value if set, zero value otherwise.
func (o *WebhooksIntegrationCustomVariableUpdateRequest) GetValue() string {
	if o == nil || o.Value == nil {
		var ret string
		return ret
	}
	return *o.Value
}

// GetValueOk returns a tuple with the Value field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WebhooksIntegrationCustomVariableUpdateRequest) GetValueOk() (*string, bool) {
	if o == nil || o.Value == nil {
		return nil, false
	}
	return o.Value, true
}

// HasValue returns a boolean if a field has been set.
func (o *WebhooksIntegrationCustomVariableUpdateRequest) HasValue() bool {
	if o != nil && o.Value != nil {
		return true
	}

	return false
}

// SetValue gets a reference to the given string and assigns it to the Value field.
func (o *WebhooksIntegrationCustomVariableUpdateRequest) SetValue(v string) {
	o.Value = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o WebhooksIntegrationCustomVariableUpdateRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.IsSecret != nil {
		toSerialize["is_secret"] = o.IsSecret
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Value != nil {
		toSerialize["value"] = o.Value
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *WebhooksIntegrationCustomVariableUpdateRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		IsSecret *bool   `json:"is_secret,omitempty"`
		Name     *string `json:"name,omitempty"`
		Value    *string `json:"value,omitempty"`
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
	o.IsSecret = all.IsSecret
	o.Name = all.Name
	o.Value = all.Value
	return nil
}
