// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// AWSAccountCreateResponse The Response returned by the AWS Create Account call.
type AWSAccountCreateResponse struct {
	// AWS external_id.
	ExternalId *string `json:"external_id,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewAWSAccountCreateResponse instantiates a new AWSAccountCreateResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAWSAccountCreateResponse() *AWSAccountCreateResponse {
	this := AWSAccountCreateResponse{}
	return &this
}

// NewAWSAccountCreateResponseWithDefaults instantiates a new AWSAccountCreateResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAWSAccountCreateResponseWithDefaults() *AWSAccountCreateResponse {
	this := AWSAccountCreateResponse{}
	return &this
}

// GetExternalId returns the ExternalId field value if set, zero value otherwise.
func (o *AWSAccountCreateResponse) GetExternalId() string {
	if o == nil || o.ExternalId == nil {
		var ret string
		return ret
	}
	return *o.ExternalId
}

// GetExternalIdOk returns a tuple with the ExternalId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSAccountCreateResponse) GetExternalIdOk() (*string, bool) {
	if o == nil || o.ExternalId == nil {
		return nil, false
	}
	return o.ExternalId, true
}

// HasExternalId returns a boolean if a field has been set.
func (o *AWSAccountCreateResponse) HasExternalId() bool {
	if o != nil && o.ExternalId != nil {
		return true
	}

	return false
}

// SetExternalId gets a reference to the given string and assigns it to the ExternalId field.
func (o *AWSAccountCreateResponse) SetExternalId(v string) {
	o.ExternalId = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o AWSAccountCreateResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ExternalId != nil {
		toSerialize["external_id"] = o.ExternalId
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *AWSAccountCreateResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		ExternalId *string `json:"external_id,omitempty"`
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
	o.ExternalId = all.ExternalId
	return nil
}
