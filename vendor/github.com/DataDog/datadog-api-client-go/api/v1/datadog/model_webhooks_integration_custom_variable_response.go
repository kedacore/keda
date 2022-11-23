// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WebhooksIntegrationCustomVariableResponse Custom variable for Webhook integration.
type WebhooksIntegrationCustomVariableResponse struct {
	// Make custom variable is secret or not.
	// If the custom variable is secret, the value is not returned in the response payload.
	IsSecret bool `json:"is_secret"`
	// The name of the variable. It corresponds with `<CUSTOM_VARIABLE_NAME>`. It must only contains upper-case characters, integers or underscores.
	Name string `json:"name"`
	// Value of the custom variable. It won't be returned if the variable is secret.
	Value *string `json:"value,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewWebhooksIntegrationCustomVariableResponse instantiates a new WebhooksIntegrationCustomVariableResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewWebhooksIntegrationCustomVariableResponse(isSecret bool, name string) *WebhooksIntegrationCustomVariableResponse {
	this := WebhooksIntegrationCustomVariableResponse{}
	this.IsSecret = isSecret
	this.Name = name
	return &this
}

// NewWebhooksIntegrationCustomVariableResponseWithDefaults instantiates a new WebhooksIntegrationCustomVariableResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewWebhooksIntegrationCustomVariableResponseWithDefaults() *WebhooksIntegrationCustomVariableResponse {
	this := WebhooksIntegrationCustomVariableResponse{}
	return &this
}

// GetIsSecret returns the IsSecret field value.
func (o *WebhooksIntegrationCustomVariableResponse) GetIsSecret() bool {
	if o == nil {
		var ret bool
		return ret
	}
	return o.IsSecret
}

// GetIsSecretOk returns a tuple with the IsSecret field value
// and a boolean to check if the value has been set.
func (o *WebhooksIntegrationCustomVariableResponse) GetIsSecretOk() (*bool, bool) {
	if o == nil {
		return nil, false
	}
	return &o.IsSecret, true
}

// SetIsSecret sets field value.
func (o *WebhooksIntegrationCustomVariableResponse) SetIsSecret(v bool) {
	o.IsSecret = v
}

// GetName returns the Name field value.
func (o *WebhooksIntegrationCustomVariableResponse) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *WebhooksIntegrationCustomVariableResponse) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *WebhooksIntegrationCustomVariableResponse) SetName(v string) {
	o.Name = v
}

// GetValue returns the Value field value if set, zero value otherwise.
func (o *WebhooksIntegrationCustomVariableResponse) GetValue() string {
	if o == nil || o.Value == nil {
		var ret string
		return ret
	}
	return *o.Value
}

// GetValueOk returns a tuple with the Value field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WebhooksIntegrationCustomVariableResponse) GetValueOk() (*string, bool) {
	if o == nil || o.Value == nil {
		return nil, false
	}
	return o.Value, true
}

// HasValue returns a boolean if a field has been set.
func (o *WebhooksIntegrationCustomVariableResponse) HasValue() bool {
	if o != nil && o.Value != nil {
		return true
	}

	return false
}

// SetValue gets a reference to the given string and assigns it to the Value field.
func (o *WebhooksIntegrationCustomVariableResponse) SetValue(v string) {
	o.Value = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o WebhooksIntegrationCustomVariableResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["is_secret"] = o.IsSecret
	toSerialize["name"] = o.Name
	if o.Value != nil {
		toSerialize["value"] = o.Value
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *WebhooksIntegrationCustomVariableResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		IsSecret *bool   `json:"is_secret"`
		Name     *string `json:"name"`
	}{}
	all := struct {
		IsSecret bool    `json:"is_secret"`
		Name     string  `json:"name"`
		Value    *string `json:"value,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.IsSecret == nil {
		return fmt.Errorf("Required field is_secret missing")
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
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
	o.IsSecret = all.IsSecret
	o.Name = all.Name
	o.Value = all.Value
	return nil
}
