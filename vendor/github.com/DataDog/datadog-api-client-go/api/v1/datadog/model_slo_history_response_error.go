// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SLOHistoryResponseError A list of errors while querying the history data for the service level objective.
type SLOHistoryResponseError struct {
	// Human readable error.
	Error *string `json:"error,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSLOHistoryResponseError instantiates a new SLOHistoryResponseError object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOHistoryResponseError() *SLOHistoryResponseError {
	this := SLOHistoryResponseError{}
	return &this
}

// NewSLOHistoryResponseErrorWithDefaults instantiates a new SLOHistoryResponseError object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOHistoryResponseErrorWithDefaults() *SLOHistoryResponseError {
	this := SLOHistoryResponseError{}
	return &this
}

// GetError returns the Error field value if set, zero value otherwise.
func (o *SLOHistoryResponseError) GetError() string {
	if o == nil || o.Error == nil {
		var ret string
		return ret
	}
	return *o.Error
}

// GetErrorOk returns a tuple with the Error field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryResponseError) GetErrorOk() (*string, bool) {
	if o == nil || o.Error == nil {
		return nil, false
	}
	return o.Error, true
}

// HasError returns a boolean if a field has been set.
func (o *SLOHistoryResponseError) HasError() bool {
	if o != nil && o.Error != nil {
		return true
	}

	return false
}

// SetError gets a reference to the given string and assigns it to the Error field.
func (o *SLOHistoryResponseError) SetError(v string) {
	o.Error = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOHistoryResponseError) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Error != nil {
		toSerialize["error"] = o.Error
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOHistoryResponseError) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Error *string `json:"error,omitempty"`
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
	o.Error = all.Error
	return nil
}
