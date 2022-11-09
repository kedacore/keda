// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WebhooksIntegration Datadog-Webhooks integration.
type WebhooksIntegration struct {
	// If `null`, uses no header.
	// If given a JSON payload, these will be headers attached to your webhook.
	CustomHeaders NullableString `json:"custom_headers,omitempty"`
	// Encoding type. Can be given either `json` or `form`.
	EncodeAs *WebhooksIntegrationEncoding `json:"encode_as,omitempty"`
	// The name of the webhook. It corresponds with `<WEBHOOK_NAME>`.
	// Learn more on how to use it in
	// [monitor notifications](https://docs.datadoghq.com/monitors/notify).
	Name string `json:"name"`
	// If `null`, uses the default payload.
	// If given a JSON payload, the webhook returns the payload
	// specified by the given payload.
	// [Webhooks variable usage](https://docs.datadoghq.com/integrations/webhooks/#usage).
	Payload NullableString `json:"payload,omitempty"`
	// URL of the webhook.
	Url string `json:"url"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewWebhooksIntegration instantiates a new WebhooksIntegration object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewWebhooksIntegration(name string, url string) *WebhooksIntegration {
	this := WebhooksIntegration{}
	var encodeAs WebhooksIntegrationEncoding = WEBHOOKSINTEGRATIONENCODING_JSON
	this.EncodeAs = &encodeAs
	this.Name = name
	this.Url = url
	return &this
}

// NewWebhooksIntegrationWithDefaults instantiates a new WebhooksIntegration object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewWebhooksIntegrationWithDefaults() *WebhooksIntegration {
	this := WebhooksIntegration{}
	var encodeAs WebhooksIntegrationEncoding = WEBHOOKSINTEGRATIONENCODING_JSON
	this.EncodeAs = &encodeAs
	return &this
}

// GetCustomHeaders returns the CustomHeaders field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *WebhooksIntegration) GetCustomHeaders() string {
	if o == nil || o.CustomHeaders.Get() == nil {
		var ret string
		return ret
	}
	return *o.CustomHeaders.Get()
}

// GetCustomHeadersOk returns a tuple with the CustomHeaders field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *WebhooksIntegration) GetCustomHeadersOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.CustomHeaders.Get(), o.CustomHeaders.IsSet()
}

// HasCustomHeaders returns a boolean if a field has been set.
func (o *WebhooksIntegration) HasCustomHeaders() bool {
	if o != nil && o.CustomHeaders.IsSet() {
		return true
	}

	return false
}

// SetCustomHeaders gets a reference to the given NullableString and assigns it to the CustomHeaders field.
func (o *WebhooksIntegration) SetCustomHeaders(v string) {
	o.CustomHeaders.Set(&v)
}

// SetCustomHeadersNil sets the value for CustomHeaders to be an explicit nil.
func (o *WebhooksIntegration) SetCustomHeadersNil() {
	o.CustomHeaders.Set(nil)
}

// UnsetCustomHeaders ensures that no value is present for CustomHeaders, not even an explicit nil.
func (o *WebhooksIntegration) UnsetCustomHeaders() {
	o.CustomHeaders.Unset()
}

// GetEncodeAs returns the EncodeAs field value if set, zero value otherwise.
func (o *WebhooksIntegration) GetEncodeAs() WebhooksIntegrationEncoding {
	if o == nil || o.EncodeAs == nil {
		var ret WebhooksIntegrationEncoding
		return ret
	}
	return *o.EncodeAs
}

// GetEncodeAsOk returns a tuple with the EncodeAs field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WebhooksIntegration) GetEncodeAsOk() (*WebhooksIntegrationEncoding, bool) {
	if o == nil || o.EncodeAs == nil {
		return nil, false
	}
	return o.EncodeAs, true
}

// HasEncodeAs returns a boolean if a field has been set.
func (o *WebhooksIntegration) HasEncodeAs() bool {
	if o != nil && o.EncodeAs != nil {
		return true
	}

	return false
}

// SetEncodeAs gets a reference to the given WebhooksIntegrationEncoding and assigns it to the EncodeAs field.
func (o *WebhooksIntegration) SetEncodeAs(v WebhooksIntegrationEncoding) {
	o.EncodeAs = &v
}

// GetName returns the Name field value.
func (o *WebhooksIntegration) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *WebhooksIntegration) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *WebhooksIntegration) SetName(v string) {
	o.Name = v
}

// GetPayload returns the Payload field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *WebhooksIntegration) GetPayload() string {
	if o == nil || o.Payload.Get() == nil {
		var ret string
		return ret
	}
	return *o.Payload.Get()
}

// GetPayloadOk returns a tuple with the Payload field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *WebhooksIntegration) GetPayloadOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Payload.Get(), o.Payload.IsSet()
}

// HasPayload returns a boolean if a field has been set.
func (o *WebhooksIntegration) HasPayload() bool {
	if o != nil && o.Payload.IsSet() {
		return true
	}

	return false
}

// SetPayload gets a reference to the given NullableString and assigns it to the Payload field.
func (o *WebhooksIntegration) SetPayload(v string) {
	o.Payload.Set(&v)
}

// SetPayloadNil sets the value for Payload to be an explicit nil.
func (o *WebhooksIntegration) SetPayloadNil() {
	o.Payload.Set(nil)
}

// UnsetPayload ensures that no value is present for Payload, not even an explicit nil.
func (o *WebhooksIntegration) UnsetPayload() {
	o.Payload.Unset()
}

// GetUrl returns the Url field value.
func (o *WebhooksIntegration) GetUrl() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Url
}

// GetUrlOk returns a tuple with the Url field value
// and a boolean to check if the value has been set.
func (o *WebhooksIntegration) GetUrlOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Url, true
}

// SetUrl sets field value.
func (o *WebhooksIntegration) SetUrl(v string) {
	o.Url = v
}

// MarshalJSON serializes the struct using spec logic.
func (o WebhooksIntegration) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CustomHeaders.IsSet() {
		toSerialize["custom_headers"] = o.CustomHeaders.Get()
	}
	if o.EncodeAs != nil {
		toSerialize["encode_as"] = o.EncodeAs
	}
	toSerialize["name"] = o.Name
	if o.Payload.IsSet() {
		toSerialize["payload"] = o.Payload.Get()
	}
	toSerialize["url"] = o.Url

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *WebhooksIntegration) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Name *string `json:"name"`
		Url  *string `json:"url"`
	}{}
	all := struct {
		CustomHeaders NullableString               `json:"custom_headers,omitempty"`
		EncodeAs      *WebhooksIntegrationEncoding `json:"encode_as,omitempty"`
		Name          string                       `json:"name"`
		Payload       NullableString               `json:"payload,omitempty"`
		Url           string                       `json:"url"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
	}
	if required.Url == nil {
		return fmt.Errorf("Required field url missing")
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
	if v := all.EncodeAs; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.CustomHeaders = all.CustomHeaders
	o.EncodeAs = all.EncodeAs
	o.Name = all.Name
	o.Payload = all.Payload
	o.Url = all.Url
	return nil
}
