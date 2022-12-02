// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// AWSAccountDeleteRequest List of AWS accounts to delete.
type AWSAccountDeleteRequest struct {
	// Your AWS access key ID. Only required if your AWS account is a GovCloud or China account.
	AccessKeyId *string `json:"access_key_id,omitempty"`
	// Your AWS Account ID without dashes.
	AccountId *string `json:"account_id,omitempty"`
	// Your Datadog role delegation name.
	RoleName *string `json:"role_name,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewAWSAccountDeleteRequest instantiates a new AWSAccountDeleteRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAWSAccountDeleteRequest() *AWSAccountDeleteRequest {
	this := AWSAccountDeleteRequest{}
	return &this
}

// NewAWSAccountDeleteRequestWithDefaults instantiates a new AWSAccountDeleteRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAWSAccountDeleteRequestWithDefaults() *AWSAccountDeleteRequest {
	this := AWSAccountDeleteRequest{}
	return &this
}

// GetAccessKeyId returns the AccessKeyId field value if set, zero value otherwise.
func (o *AWSAccountDeleteRequest) GetAccessKeyId() string {
	if o == nil || o.AccessKeyId == nil {
		var ret string
		return ret
	}
	return *o.AccessKeyId
}

// GetAccessKeyIdOk returns a tuple with the AccessKeyId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSAccountDeleteRequest) GetAccessKeyIdOk() (*string, bool) {
	if o == nil || o.AccessKeyId == nil {
		return nil, false
	}
	return o.AccessKeyId, true
}

// HasAccessKeyId returns a boolean if a field has been set.
func (o *AWSAccountDeleteRequest) HasAccessKeyId() bool {
	if o != nil && o.AccessKeyId != nil {
		return true
	}

	return false
}

// SetAccessKeyId gets a reference to the given string and assigns it to the AccessKeyId field.
func (o *AWSAccountDeleteRequest) SetAccessKeyId(v string) {
	o.AccessKeyId = &v
}

// GetAccountId returns the AccountId field value if set, zero value otherwise.
func (o *AWSAccountDeleteRequest) GetAccountId() string {
	if o == nil || o.AccountId == nil {
		var ret string
		return ret
	}
	return *o.AccountId
}

// GetAccountIdOk returns a tuple with the AccountId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSAccountDeleteRequest) GetAccountIdOk() (*string, bool) {
	if o == nil || o.AccountId == nil {
		return nil, false
	}
	return o.AccountId, true
}

// HasAccountId returns a boolean if a field has been set.
func (o *AWSAccountDeleteRequest) HasAccountId() bool {
	if o != nil && o.AccountId != nil {
		return true
	}

	return false
}

// SetAccountId gets a reference to the given string and assigns it to the AccountId field.
func (o *AWSAccountDeleteRequest) SetAccountId(v string) {
	o.AccountId = &v
}

// GetRoleName returns the RoleName field value if set, zero value otherwise.
func (o *AWSAccountDeleteRequest) GetRoleName() string {
	if o == nil || o.RoleName == nil {
		var ret string
		return ret
	}
	return *o.RoleName
}

// GetRoleNameOk returns a tuple with the RoleName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSAccountDeleteRequest) GetRoleNameOk() (*string, bool) {
	if o == nil || o.RoleName == nil {
		return nil, false
	}
	return o.RoleName, true
}

// HasRoleName returns a boolean if a field has been set.
func (o *AWSAccountDeleteRequest) HasRoleName() bool {
	if o != nil && o.RoleName != nil {
		return true
	}

	return false
}

// SetRoleName gets a reference to the given string and assigns it to the RoleName field.
func (o *AWSAccountDeleteRequest) SetRoleName(v string) {
	o.RoleName = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o AWSAccountDeleteRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AccessKeyId != nil {
		toSerialize["access_key_id"] = o.AccessKeyId
	}
	if o.AccountId != nil {
		toSerialize["account_id"] = o.AccountId
	}
	if o.RoleName != nil {
		toSerialize["role_name"] = o.RoleName
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *AWSAccountDeleteRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AccessKeyId *string `json:"access_key_id,omitempty"`
		AccountId   *string `json:"account_id,omitempty"`
		RoleName    *string `json:"role_name,omitempty"`
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
	o.AccessKeyId = all.AccessKeyId
	o.AccountId = all.AccountId
	o.RoleName = all.RoleName
	return nil
}
