// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WebhooksIntegrationCustomVariable Custom variable for Webhook integration.
type WebhooksIntegrationCustomVariable struct {
	// Make custom variable is secret or not.
	// If the custom variable is secret, the value is not returned in the response payload.
	IsSecret bool `json:"is_secret"`
	// The name of the variable. It corresponds with `<CUSTOM_VARIABLE_NAME>`.
	Name string `json:"name"`
	// Value of the custom variable.
	Value string `json:"value"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewWebhooksIntegrationCustomVariable instantiates a new WebhooksIntegrationCustomVariable object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewWebhooksIntegrationCustomVariable(isSecret bool, name string, value string) *WebhooksIntegrationCustomVariable {
	this := WebhooksIntegrationCustomVariable{}
	this.IsSecret = isSecret
	this.Name = name
	this.Value = value
	return &this
}

// NewWebhooksIntegrationCustomVariableWithDefaults instantiates a new WebhooksIntegrationCustomVariable object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewWebhooksIntegrationCustomVariableWithDefaults() *WebhooksIntegrationCustomVariable {
	this := WebhooksIntegrationCustomVariable{}
	return &this
}

// GetIsSecret returns the IsSecret field value.
func (o *WebhooksIntegrationCustomVariable) GetIsSecret() bool {
	if o == nil {
		var ret bool
		return ret
	}
	return o.IsSecret
}

// GetIsSecretOk returns a tuple with the IsSecret field value
// and a boolean to check if the value has been set.
func (o *WebhooksIntegrationCustomVariable) GetIsSecretOk() (*bool, bool) {
	if o == nil {
		return nil, false
	}
	return &o.IsSecret, true
}

// SetIsSecret sets field value.
func (o *WebhooksIntegrationCustomVariable) SetIsSecret(v bool) {
	o.IsSecret = v
}

// GetName returns the Name field value.
func (o *WebhooksIntegrationCustomVariable) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *WebhooksIntegrationCustomVariable) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *WebhooksIntegrationCustomVariable) SetName(v string) {
	o.Name = v
}

// GetValue returns the Value field value.
func (o *WebhooksIntegrationCustomVariable) GetValue() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Value
}

// GetValueOk returns a tuple with the Value field value
// and a boolean to check if the value has been set.
func (o *WebhooksIntegrationCustomVariable) GetValueOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Value, true
}

// SetValue sets field value.
func (o *WebhooksIntegrationCustomVariable) SetValue(v string) {
	o.Value = v
}

// MarshalJSON serializes the struct using spec logic.
func (o WebhooksIntegrationCustomVariable) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["is_secret"] = o.IsSecret
	toSerialize["name"] = o.Name
	toSerialize["value"] = o.Value

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *WebhooksIntegrationCustomVariable) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		IsSecret *bool   `json:"is_secret"`
		Name     *string `json:"name"`
		Value    *string `json:"value"`
	}{}
	all := struct {
		IsSecret bool   `json:"is_secret"`
		Name     string `json:"name"`
		Value    string `json:"value"`
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
	if required.Value == nil {
		return fmt.Errorf("Required field value missing")
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
