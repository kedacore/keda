// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// AWSLogsListResponse A list of all Datadog-AWS logs integrations available in your Datadog organization.
type AWSLogsListResponse struct {
	// Your AWS Account ID without dashes.
	AccountId *string `json:"account_id,omitempty"`
	// List of ARNs configured in your Datadog account.
	Lambdas []AWSLogsLambda `json:"lambdas,omitempty"`
	// Array of services IDs.
	Services []string `json:"services,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewAWSLogsListResponse instantiates a new AWSLogsListResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAWSLogsListResponse() *AWSLogsListResponse {
	this := AWSLogsListResponse{}
	return &this
}

// NewAWSLogsListResponseWithDefaults instantiates a new AWSLogsListResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAWSLogsListResponseWithDefaults() *AWSLogsListResponse {
	this := AWSLogsListResponse{}
	return &this
}

// GetAccountId returns the AccountId field value if set, zero value otherwise.
func (o *AWSLogsListResponse) GetAccountId() string {
	if o == nil || o.AccountId == nil {
		var ret string
		return ret
	}
	return *o.AccountId
}

// GetAccountIdOk returns a tuple with the AccountId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSLogsListResponse) GetAccountIdOk() (*string, bool) {
	if o == nil || o.AccountId == nil {
		return nil, false
	}
	return o.AccountId, true
}

// HasAccountId returns a boolean if a field has been set.
func (o *AWSLogsListResponse) HasAccountId() bool {
	if o != nil && o.AccountId != nil {
		return true
	}

	return false
}

// SetAccountId gets a reference to the given string and assigns it to the AccountId field.
func (o *AWSLogsListResponse) SetAccountId(v string) {
	o.AccountId = &v
}

// GetLambdas returns the Lambdas field value if set, zero value otherwise.
func (o *AWSLogsListResponse) GetLambdas() []AWSLogsLambda {
	if o == nil || o.Lambdas == nil {
		var ret []AWSLogsLambda
		return ret
	}
	return o.Lambdas
}

// GetLambdasOk returns a tuple with the Lambdas field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSLogsListResponse) GetLambdasOk() (*[]AWSLogsLambda, bool) {
	if o == nil || o.Lambdas == nil {
		return nil, false
	}
	return &o.Lambdas, true
}

// HasLambdas returns a boolean if a field has been set.
func (o *AWSLogsListResponse) HasLambdas() bool {
	if o != nil && o.Lambdas != nil {
		return true
	}

	return false
}

// SetLambdas gets a reference to the given []AWSLogsLambda and assigns it to the Lambdas field.
func (o *AWSLogsListResponse) SetLambdas(v []AWSLogsLambda) {
	o.Lambdas = v
}

// GetServices returns the Services field value if set, zero value otherwise.
func (o *AWSLogsListResponse) GetServices() []string {
	if o == nil || o.Services == nil {
		var ret []string
		return ret
	}
	return o.Services
}

// GetServicesOk returns a tuple with the Services field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSLogsListResponse) GetServicesOk() (*[]string, bool) {
	if o == nil || o.Services == nil {
		return nil, false
	}
	return &o.Services, true
}

// HasServices returns a boolean if a field has been set.
func (o *AWSLogsListResponse) HasServices() bool {
	if o != nil && o.Services != nil {
		return true
	}

	return false
}

// SetServices gets a reference to the given []string and assigns it to the Services field.
func (o *AWSLogsListResponse) SetServices(v []string) {
	o.Services = v
}

// MarshalJSON serializes the struct using spec logic.
func (o AWSLogsListResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AccountId != nil {
		toSerialize["account_id"] = o.AccountId
	}
	if o.Lambdas != nil {
		toSerialize["lambdas"] = o.Lambdas
	}
	if o.Services != nil {
		toSerialize["services"] = o.Services
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *AWSLogsListResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AccountId *string         `json:"account_id,omitempty"`
		Lambdas   []AWSLogsLambda `json:"lambdas,omitempty"`
		Services  []string        `json:"services,omitempty"`
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
	o.AccountId = all.AccountId
	o.Lambdas = all.Lambdas
	o.Services = all.Services
	return nil
}
