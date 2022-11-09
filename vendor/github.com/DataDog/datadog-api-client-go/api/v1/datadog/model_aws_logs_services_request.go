// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// AWSLogsServicesRequest A list of current AWS services for which Datadog offers automatic log collection.
type AWSLogsServicesRequest struct {
	// Your AWS Account ID without dashes.
	AccountId string `json:"account_id"`
	// Array of services IDs set to enable automatic log collection. Discover the list of available services with the get list of AWS log ready services API endpoint.
	Services []string `json:"services"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewAWSLogsServicesRequest instantiates a new AWSLogsServicesRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAWSLogsServicesRequest(accountId string, services []string) *AWSLogsServicesRequest {
	this := AWSLogsServicesRequest{}
	this.AccountId = accountId
	this.Services = services
	return &this
}

// NewAWSLogsServicesRequestWithDefaults instantiates a new AWSLogsServicesRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAWSLogsServicesRequestWithDefaults() *AWSLogsServicesRequest {
	this := AWSLogsServicesRequest{}
	return &this
}

// GetAccountId returns the AccountId field value.
func (o *AWSLogsServicesRequest) GetAccountId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.AccountId
}

// GetAccountIdOk returns a tuple with the AccountId field value
// and a boolean to check if the value has been set.
func (o *AWSLogsServicesRequest) GetAccountIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.AccountId, true
}

// SetAccountId sets field value.
func (o *AWSLogsServicesRequest) SetAccountId(v string) {
	o.AccountId = v
}

// GetServices returns the Services field value.
func (o *AWSLogsServicesRequest) GetServices() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.Services
}

// GetServicesOk returns a tuple with the Services field value
// and a boolean to check if the value has been set.
func (o *AWSLogsServicesRequest) GetServicesOk() (*[]string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Services, true
}

// SetServices sets field value.
func (o *AWSLogsServicesRequest) SetServices(v []string) {
	o.Services = v
}

// MarshalJSON serializes the struct using spec logic.
func (o AWSLogsServicesRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["account_id"] = o.AccountId
	toSerialize["services"] = o.Services

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *AWSLogsServicesRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		AccountId *string   `json:"account_id"`
		Services  *[]string `json:"services"`
	}{}
	all := struct {
		AccountId string   `json:"account_id"`
		Services  []string `json:"services"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.AccountId == nil {
		return fmt.Errorf("Required field account_id missing")
	}
	if required.Services == nil {
		return fmt.Errorf("Required field services missing")
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
	o.AccountId = all.AccountId
	o.Services = all.Services
	return nil
}
