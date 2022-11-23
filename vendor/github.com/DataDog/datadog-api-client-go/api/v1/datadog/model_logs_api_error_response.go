// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// LogsAPIErrorResponse Response returned by the Logs API when errors occur.
type LogsAPIErrorResponse struct {
	// Error returned by the Logs API
	Error *LogsAPIError `json:"error,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsAPIErrorResponse instantiates a new LogsAPIErrorResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsAPIErrorResponse() *LogsAPIErrorResponse {
	this := LogsAPIErrorResponse{}
	return &this
}

// NewLogsAPIErrorResponseWithDefaults instantiates a new LogsAPIErrorResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsAPIErrorResponseWithDefaults() *LogsAPIErrorResponse {
	this := LogsAPIErrorResponse{}
	return &this
}

// GetError returns the Error field value if set, zero value otherwise.
func (o *LogsAPIErrorResponse) GetError() LogsAPIError {
	if o == nil || o.Error == nil {
		var ret LogsAPIError
		return ret
	}
	return *o.Error
}

// GetErrorOk returns a tuple with the Error field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsAPIErrorResponse) GetErrorOk() (*LogsAPIError, bool) {
	if o == nil || o.Error == nil {
		return nil, false
	}
	return o.Error, true
}

// HasError returns a boolean if a field has been set.
func (o *LogsAPIErrorResponse) HasError() bool {
	if o != nil && o.Error != nil {
		return true
	}

	return false
}

// SetError gets a reference to the given LogsAPIError and assigns it to the Error field.
func (o *LogsAPIErrorResponse) SetError(v LogsAPIError) {
	o.Error = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsAPIErrorResponse) MarshalJSON() ([]byte, error) {
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
func (o *LogsAPIErrorResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Error *LogsAPIError `json:"error,omitempty"`
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
	if all.Error != nil && all.Error.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Error = all.Error
	return nil
}
