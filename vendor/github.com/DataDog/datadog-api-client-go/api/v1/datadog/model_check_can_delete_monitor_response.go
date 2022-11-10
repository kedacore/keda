// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// CheckCanDeleteMonitorResponse Response of monitor IDs that can or can't be safely deleted.
type CheckCanDeleteMonitorResponse struct {
	// Wrapper object with the list of monitor IDs.
	Data CheckCanDeleteMonitorResponseData `json:"data"`
	// A mapping of Monitor ID to strings denoting where it's used.
	Errors map[string][]string `json:"errors,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewCheckCanDeleteMonitorResponse instantiates a new CheckCanDeleteMonitorResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCheckCanDeleteMonitorResponse(data CheckCanDeleteMonitorResponseData) *CheckCanDeleteMonitorResponse {
	this := CheckCanDeleteMonitorResponse{}
	this.Data = data
	return &this
}

// NewCheckCanDeleteMonitorResponseWithDefaults instantiates a new CheckCanDeleteMonitorResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCheckCanDeleteMonitorResponseWithDefaults() *CheckCanDeleteMonitorResponse {
	this := CheckCanDeleteMonitorResponse{}
	return &this
}

// GetData returns the Data field value.
func (o *CheckCanDeleteMonitorResponse) GetData() CheckCanDeleteMonitorResponseData {
	if o == nil {
		var ret CheckCanDeleteMonitorResponseData
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value
// and a boolean to check if the value has been set.
func (o *CheckCanDeleteMonitorResponse) GetDataOk() (*CheckCanDeleteMonitorResponseData, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Data, true
}

// SetData sets field value.
func (o *CheckCanDeleteMonitorResponse) SetData(v CheckCanDeleteMonitorResponseData) {
	o.Data = v
}

// GetErrors returns the Errors field value if set, zero value otherwise.
func (o *CheckCanDeleteMonitorResponse) GetErrors() map[string][]string {
	if o == nil || o.Errors == nil {
		var ret map[string][]string
		return ret
	}
	return o.Errors
}

// GetErrorsOk returns a tuple with the Errors field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CheckCanDeleteMonitorResponse) GetErrorsOk() (*map[string][]string, bool) {
	if o == nil || o.Errors == nil {
		return nil, false
	}
	return &o.Errors, true
}

// HasErrors returns a boolean if a field has been set.
func (o *CheckCanDeleteMonitorResponse) HasErrors() bool {
	if o != nil && o.Errors != nil {
		return true
	}

	return false
}

// SetErrors gets a reference to the given map[string][]string and assigns it to the Errors field.
func (o *CheckCanDeleteMonitorResponse) SetErrors(v map[string][]string) {
	o.Errors = v
}

// MarshalJSON serializes the struct using spec logic.
func (o CheckCanDeleteMonitorResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["data"] = o.Data
	if o.Errors != nil {
		toSerialize["errors"] = o.Errors
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CheckCanDeleteMonitorResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Data *CheckCanDeleteMonitorResponseData `json:"data"`
	}{}
	all := struct {
		Data   CheckCanDeleteMonitorResponseData `json:"data"`
		Errors map[string][]string               `json:"errors,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Data == nil {
		return fmt.Errorf("Required field data missing")
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
	if all.Data.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Data = all.Data
	o.Errors = all.Errors
	return nil
}
