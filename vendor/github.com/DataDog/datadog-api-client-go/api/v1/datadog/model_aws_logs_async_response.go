// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// AWSLogsAsyncResponse A list of all Datadog-AWS logs integrations available in your Datadog organization.
type AWSLogsAsyncResponse struct {
	// List of errors.
	Errors []AWSLogsAsyncError `json:"errors,omitempty"`
	// Status of the properties.
	Status *string `json:"status,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewAWSLogsAsyncResponse instantiates a new AWSLogsAsyncResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAWSLogsAsyncResponse() *AWSLogsAsyncResponse {
	this := AWSLogsAsyncResponse{}
	return &this
}

// NewAWSLogsAsyncResponseWithDefaults instantiates a new AWSLogsAsyncResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAWSLogsAsyncResponseWithDefaults() *AWSLogsAsyncResponse {
	this := AWSLogsAsyncResponse{}
	return &this
}

// GetErrors returns the Errors field value if set, zero value otherwise.
func (o *AWSLogsAsyncResponse) GetErrors() []AWSLogsAsyncError {
	if o == nil || o.Errors == nil {
		var ret []AWSLogsAsyncError
		return ret
	}
	return o.Errors
}

// GetErrorsOk returns a tuple with the Errors field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSLogsAsyncResponse) GetErrorsOk() (*[]AWSLogsAsyncError, bool) {
	if o == nil || o.Errors == nil {
		return nil, false
	}
	return &o.Errors, true
}

// HasErrors returns a boolean if a field has been set.
func (o *AWSLogsAsyncResponse) HasErrors() bool {
	if o != nil && o.Errors != nil {
		return true
	}

	return false
}

// SetErrors gets a reference to the given []AWSLogsAsyncError and assigns it to the Errors field.
func (o *AWSLogsAsyncResponse) SetErrors(v []AWSLogsAsyncError) {
	o.Errors = v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *AWSLogsAsyncResponse) GetStatus() string {
	if o == nil || o.Status == nil {
		var ret string
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSLogsAsyncResponse) GetStatusOk() (*string, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *AWSLogsAsyncResponse) HasStatus() bool {
	if o != nil && o.Status != nil {
		return true
	}

	return false
}

// SetStatus gets a reference to the given string and assigns it to the Status field.
func (o *AWSLogsAsyncResponse) SetStatus(v string) {
	o.Status = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o AWSLogsAsyncResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Errors != nil {
		toSerialize["errors"] = o.Errors
	}
	if o.Status != nil {
		toSerialize["status"] = o.Status
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *AWSLogsAsyncResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Errors []AWSLogsAsyncError `json:"errors,omitempty"`
		Status *string             `json:"status,omitempty"`
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
	o.Errors = all.Errors
	o.Status = all.Status
	return nil
}
