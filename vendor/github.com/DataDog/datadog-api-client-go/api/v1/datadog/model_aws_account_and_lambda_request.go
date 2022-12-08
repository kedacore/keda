// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// AWSAccountAndLambdaRequest AWS account ID and Lambda ARN.
type AWSAccountAndLambdaRequest struct {
	// Your AWS Account ID without dashes.
	AccountId string `json:"account_id"`
	// ARN of the Datadog Lambda created during the Datadog-Amazon Web services Log collection setup.
	LambdaArn string `json:"lambda_arn"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewAWSAccountAndLambdaRequest instantiates a new AWSAccountAndLambdaRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAWSAccountAndLambdaRequest(accountId string, lambdaArn string) *AWSAccountAndLambdaRequest {
	this := AWSAccountAndLambdaRequest{}
	this.AccountId = accountId
	this.LambdaArn = lambdaArn
	return &this
}

// NewAWSAccountAndLambdaRequestWithDefaults instantiates a new AWSAccountAndLambdaRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAWSAccountAndLambdaRequestWithDefaults() *AWSAccountAndLambdaRequest {
	this := AWSAccountAndLambdaRequest{}
	return &this
}

// GetAccountId returns the AccountId field value.
func (o *AWSAccountAndLambdaRequest) GetAccountId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.AccountId
}

// GetAccountIdOk returns a tuple with the AccountId field value
// and a boolean to check if the value has been set.
func (o *AWSAccountAndLambdaRequest) GetAccountIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.AccountId, true
}

// SetAccountId sets field value.
func (o *AWSAccountAndLambdaRequest) SetAccountId(v string) {
	o.AccountId = v
}

// GetLambdaArn returns the LambdaArn field value.
func (o *AWSAccountAndLambdaRequest) GetLambdaArn() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.LambdaArn
}

// GetLambdaArnOk returns a tuple with the LambdaArn field value
// and a boolean to check if the value has been set.
func (o *AWSAccountAndLambdaRequest) GetLambdaArnOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.LambdaArn, true
}

// SetLambdaArn sets field value.
func (o *AWSAccountAndLambdaRequest) SetLambdaArn(v string) {
	o.LambdaArn = v
}

// MarshalJSON serializes the struct using spec logic.
func (o AWSAccountAndLambdaRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["account_id"] = o.AccountId
	toSerialize["lambda_arn"] = o.LambdaArn

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *AWSAccountAndLambdaRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		AccountId *string `json:"account_id"`
		LambdaArn *string `json:"lambda_arn"`
	}{}
	all := struct {
		AccountId string `json:"account_id"`
		LambdaArn string `json:"lambda_arn"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.AccountId == nil {
		return fmt.Errorf("Required field account_id missing")
	}
	if required.LambdaArn == nil {
		return fmt.Errorf("Required field lambda_arn missing")
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
	o.LambdaArn = all.LambdaArn
	return nil
}
