// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsPrivateLocationSecretsConfigDecryption Private key for the private location.
type SyntheticsPrivateLocationSecretsConfigDecryption struct {
	// Private key for the private location.
	Key *string `json:"key,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsPrivateLocationSecretsConfigDecryption instantiates a new SyntheticsPrivateLocationSecretsConfigDecryption object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsPrivateLocationSecretsConfigDecryption() *SyntheticsPrivateLocationSecretsConfigDecryption {
	this := SyntheticsPrivateLocationSecretsConfigDecryption{}
	return &this
}

// NewSyntheticsPrivateLocationSecretsConfigDecryptionWithDefaults instantiates a new SyntheticsPrivateLocationSecretsConfigDecryption object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsPrivateLocationSecretsConfigDecryptionWithDefaults() *SyntheticsPrivateLocationSecretsConfigDecryption {
	this := SyntheticsPrivateLocationSecretsConfigDecryption{}
	return &this
}

// GetKey returns the Key field value if set, zero value otherwise.
func (o *SyntheticsPrivateLocationSecretsConfigDecryption) GetKey() string {
	if o == nil || o.Key == nil {
		var ret string
		return ret
	}
	return *o.Key
}

// GetKeyOk returns a tuple with the Key field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsPrivateLocationSecretsConfigDecryption) GetKeyOk() (*string, bool) {
	if o == nil || o.Key == nil {
		return nil, false
	}
	return o.Key, true
}

// HasKey returns a boolean if a field has been set.
func (o *SyntheticsPrivateLocationSecretsConfigDecryption) HasKey() bool {
	if o != nil && o.Key != nil {
		return true
	}

	return false
}

// SetKey gets a reference to the given string and assigns it to the Key field.
func (o *SyntheticsPrivateLocationSecretsConfigDecryption) SetKey(v string) {
	o.Key = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsPrivateLocationSecretsConfigDecryption) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Key != nil {
		toSerialize["key"] = o.Key
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsPrivateLocationSecretsConfigDecryption) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Key *string `json:"key,omitempty"`
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
	o.Key = all.Key
	return nil
}
