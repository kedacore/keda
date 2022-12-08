// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// ApiKeyListResponse List of API and application keys available for a given organization.
type ApiKeyListResponse struct {
	// Array of API keys.
	ApiKeys []ApiKey `json:"api_keys,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewApiKeyListResponse instantiates a new ApiKeyListResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewApiKeyListResponse() *ApiKeyListResponse {
	this := ApiKeyListResponse{}
	return &this
}

// NewApiKeyListResponseWithDefaults instantiates a new ApiKeyListResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewApiKeyListResponseWithDefaults() *ApiKeyListResponse {
	this := ApiKeyListResponse{}
	return &this
}

// GetApiKeys returns the ApiKeys field value if set, zero value otherwise.
func (o *ApiKeyListResponse) GetApiKeys() []ApiKey {
	if o == nil || o.ApiKeys == nil {
		var ret []ApiKey
		return ret
	}
	return o.ApiKeys
}

// GetApiKeysOk returns a tuple with the ApiKeys field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ApiKeyListResponse) GetApiKeysOk() (*[]ApiKey, bool) {
	if o == nil || o.ApiKeys == nil {
		return nil, false
	}
	return &o.ApiKeys, true
}

// HasApiKeys returns a boolean if a field has been set.
func (o *ApiKeyListResponse) HasApiKeys() bool {
	if o != nil && o.ApiKeys != nil {
		return true
	}

	return false
}

// SetApiKeys gets a reference to the given []ApiKey and assigns it to the ApiKeys field.
func (o *ApiKeyListResponse) SetApiKeys(v []ApiKey) {
	o.ApiKeys = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ApiKeyListResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ApiKeys != nil {
		toSerialize["api_keys"] = o.ApiKeys
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ApiKeyListResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		ApiKeys []ApiKey `json:"api_keys,omitempty"`
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
	o.ApiKeys = all.ApiKeys
	return nil
}
