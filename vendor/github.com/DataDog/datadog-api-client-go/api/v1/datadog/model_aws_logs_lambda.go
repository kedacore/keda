// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// AWSLogsLambda Description of the Lambdas.
type AWSLogsLambda struct {
	// Available ARN IDs.
	Arn *string `json:"arn,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewAWSLogsLambda instantiates a new AWSLogsLambda object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAWSLogsLambda() *AWSLogsLambda {
	this := AWSLogsLambda{}
	return &this
}

// NewAWSLogsLambdaWithDefaults instantiates a new AWSLogsLambda object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAWSLogsLambdaWithDefaults() *AWSLogsLambda {
	this := AWSLogsLambda{}
	return &this
}

// GetArn returns the Arn field value if set, zero value otherwise.
func (o *AWSLogsLambda) GetArn() string {
	if o == nil || o.Arn == nil {
		var ret string
		return ret
	}
	return *o.Arn
}

// GetArnOk returns a tuple with the Arn field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSLogsLambda) GetArnOk() (*string, bool) {
	if o == nil || o.Arn == nil {
		return nil, false
	}
	return o.Arn, true
}

// HasArn returns a boolean if a field has been set.
func (o *AWSLogsLambda) HasArn() bool {
	if o != nil && o.Arn != nil {
		return true
	}

	return false
}

// SetArn gets a reference to the given string and assigns it to the Arn field.
func (o *AWSLogsLambda) SetArn(v string) {
	o.Arn = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o AWSLogsLambda) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Arn != nil {
		toSerialize["arn"] = o.Arn
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *AWSLogsLambda) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Arn *string `json:"arn,omitempty"`
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
	o.Arn = all.Arn
	return nil
}
