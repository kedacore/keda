// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// LogsAPIError Error returned by the Logs API
type LogsAPIError struct {
	// Code identifying the error
	Code *string `json:"code,omitempty"`
	// Additional error details
	Details []LogsAPIError `json:"details,omitempty"`
	// Error message
	Message *string `json:"message,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsAPIError instantiates a new LogsAPIError object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsAPIError() *LogsAPIError {
	this := LogsAPIError{}
	return &this
}

// NewLogsAPIErrorWithDefaults instantiates a new LogsAPIError object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsAPIErrorWithDefaults() *LogsAPIError {
	this := LogsAPIError{}
	return &this
}

// GetCode returns the Code field value if set, zero value otherwise.
func (o *LogsAPIError) GetCode() string {
	if o == nil || o.Code == nil {
		var ret string
		return ret
	}
	return *o.Code
}

// GetCodeOk returns a tuple with the Code field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsAPIError) GetCodeOk() (*string, bool) {
	if o == nil || o.Code == nil {
		return nil, false
	}
	return o.Code, true
}

// HasCode returns a boolean if a field has been set.
func (o *LogsAPIError) HasCode() bool {
	if o != nil && o.Code != nil {
		return true
	}

	return false
}

// SetCode gets a reference to the given string and assigns it to the Code field.
func (o *LogsAPIError) SetCode(v string) {
	o.Code = &v
}

// GetDetails returns the Details field value if set, zero value otherwise.
func (o *LogsAPIError) GetDetails() []LogsAPIError {
	if o == nil || o.Details == nil {
		var ret []LogsAPIError
		return ret
	}
	return o.Details
}

// GetDetailsOk returns a tuple with the Details field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsAPIError) GetDetailsOk() (*[]LogsAPIError, bool) {
	if o == nil || o.Details == nil {
		return nil, false
	}
	return &o.Details, true
}

// HasDetails returns a boolean if a field has been set.
func (o *LogsAPIError) HasDetails() bool {
	if o != nil && o.Details != nil {
		return true
	}

	return false
}

// SetDetails gets a reference to the given []LogsAPIError and assigns it to the Details field.
func (o *LogsAPIError) SetDetails(v []LogsAPIError) {
	o.Details = v
}

// GetMessage returns the Message field value if set, zero value otherwise.
func (o *LogsAPIError) GetMessage() string {
	if o == nil || o.Message == nil {
		var ret string
		return ret
	}
	return *o.Message
}

// GetMessageOk returns a tuple with the Message field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsAPIError) GetMessageOk() (*string, bool) {
	if o == nil || o.Message == nil {
		return nil, false
	}
	return o.Message, true
}

// HasMessage returns a boolean if a field has been set.
func (o *LogsAPIError) HasMessage() bool {
	if o != nil && o.Message != nil {
		return true
	}

	return false
}

// SetMessage gets a reference to the given string and assigns it to the Message field.
func (o *LogsAPIError) SetMessage(v string) {
	o.Message = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsAPIError) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Code != nil {
		toSerialize["code"] = o.Code
	}
	if o.Details != nil {
		toSerialize["details"] = o.Details
	}
	if o.Message != nil {
		toSerialize["message"] = o.Message
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsAPIError) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Code    *string        `json:"code,omitempty"`
		Details []LogsAPIError `json:"details,omitempty"`
		Message *string        `json:"message,omitempty"`
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
	o.Code = all.Code
	o.Details = all.Details
	o.Message = all.Message
	return nil
}
