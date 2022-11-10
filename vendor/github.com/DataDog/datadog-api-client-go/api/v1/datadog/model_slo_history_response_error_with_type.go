// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SLOHistoryResponseErrorWithType An object describing the error with error type and error message.
type SLOHistoryResponseErrorWithType struct {
	// A message with more details about the error.
	ErrorMessage string `json:"error_message"`
	// Type of the error.
	ErrorType string `json:"error_type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSLOHistoryResponseErrorWithType instantiates a new SLOHistoryResponseErrorWithType object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOHistoryResponseErrorWithType(errorMessage string, errorType string) *SLOHistoryResponseErrorWithType {
	this := SLOHistoryResponseErrorWithType{}
	this.ErrorMessage = errorMessage
	this.ErrorType = errorType
	return &this
}

// NewSLOHistoryResponseErrorWithTypeWithDefaults instantiates a new SLOHistoryResponseErrorWithType object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOHistoryResponseErrorWithTypeWithDefaults() *SLOHistoryResponseErrorWithType {
	this := SLOHistoryResponseErrorWithType{}
	return &this
}

// GetErrorMessage returns the ErrorMessage field value.
func (o *SLOHistoryResponseErrorWithType) GetErrorMessage() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.ErrorMessage
}

// GetErrorMessageOk returns a tuple with the ErrorMessage field value
// and a boolean to check if the value has been set.
func (o *SLOHistoryResponseErrorWithType) GetErrorMessageOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ErrorMessage, true
}

// SetErrorMessage sets field value.
func (o *SLOHistoryResponseErrorWithType) SetErrorMessage(v string) {
	o.ErrorMessage = v
}

// GetErrorType returns the ErrorType field value.
func (o *SLOHistoryResponseErrorWithType) GetErrorType() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.ErrorType
}

// GetErrorTypeOk returns a tuple with the ErrorType field value
// and a boolean to check if the value has been set.
func (o *SLOHistoryResponseErrorWithType) GetErrorTypeOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ErrorType, true
}

// SetErrorType sets field value.
func (o *SLOHistoryResponseErrorWithType) SetErrorType(v string) {
	o.ErrorType = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOHistoryResponseErrorWithType) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["error_message"] = o.ErrorMessage
	toSerialize["error_type"] = o.ErrorType

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOHistoryResponseErrorWithType) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		ErrorMessage *string `json:"error_message"`
		ErrorType    *string `json:"error_type"`
	}{}
	all := struct {
		ErrorMessage string `json:"error_message"`
		ErrorType    string `json:"error_type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.ErrorMessage == nil {
		return fmt.Errorf("Required field error_message missing")
	}
	if required.ErrorType == nil {
		return fmt.Errorf("Required field error_type missing")
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
	o.ErrorMessage = all.ErrorMessage
	o.ErrorType = all.ErrorType
	return nil
}
