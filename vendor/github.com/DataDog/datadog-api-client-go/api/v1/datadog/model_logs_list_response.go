// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// LogsListResponse Response object with all logs matching the request and pagination information.
type LogsListResponse struct {
	// Array of logs matching the request and the `nextLogId` if sent.
	Logs []Log `json:"logs,omitempty"`
	// Hash identifier of the next log to return in the list.
	// This parameter is used for the pagination feature.
	NextLogId *string `json:"nextLogId,omitempty"`
	// Status of the response.
	Status *string `json:"status,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsListResponse instantiates a new LogsListResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsListResponse() *LogsListResponse {
	this := LogsListResponse{}
	return &this
}

// NewLogsListResponseWithDefaults instantiates a new LogsListResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsListResponseWithDefaults() *LogsListResponse {
	this := LogsListResponse{}
	return &this
}

// GetLogs returns the Logs field value if set, zero value otherwise.
func (o *LogsListResponse) GetLogs() []Log {
	if o == nil || o.Logs == nil {
		var ret []Log
		return ret
	}
	return o.Logs
}

// GetLogsOk returns a tuple with the Logs field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsListResponse) GetLogsOk() (*[]Log, bool) {
	if o == nil || o.Logs == nil {
		return nil, false
	}
	return &o.Logs, true
}

// HasLogs returns a boolean if a field has been set.
func (o *LogsListResponse) HasLogs() bool {
	if o != nil && o.Logs != nil {
		return true
	}

	return false
}

// SetLogs gets a reference to the given []Log and assigns it to the Logs field.
func (o *LogsListResponse) SetLogs(v []Log) {
	o.Logs = v
}

// GetNextLogId returns the NextLogId field value if set, zero value otherwise.
func (o *LogsListResponse) GetNextLogId() string {
	if o == nil || o.NextLogId == nil {
		var ret string
		return ret
	}
	return *o.NextLogId
}

// GetNextLogIdOk returns a tuple with the NextLogId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsListResponse) GetNextLogIdOk() (*string, bool) {
	if o == nil || o.NextLogId == nil {
		return nil, false
	}
	return o.NextLogId, true
}

// HasNextLogId returns a boolean if a field has been set.
func (o *LogsListResponse) HasNextLogId() bool {
	if o != nil && o.NextLogId != nil {
		return true
	}

	return false
}

// SetNextLogId gets a reference to the given string and assigns it to the NextLogId field.
func (o *LogsListResponse) SetNextLogId(v string) {
	o.NextLogId = &v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *LogsListResponse) GetStatus() string {
	if o == nil || o.Status == nil {
		var ret string
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsListResponse) GetStatusOk() (*string, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *LogsListResponse) HasStatus() bool {
	if o != nil && o.Status != nil {
		return true
	}

	return false
}

// SetStatus gets a reference to the given string and assigns it to the Status field.
func (o *LogsListResponse) SetStatus(v string) {
	o.Status = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsListResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Logs != nil {
		toSerialize["logs"] = o.Logs
	}
	if o.NextLogId != nil {
		toSerialize["nextLogId"] = o.NextLogId
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
func (o *LogsListResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Logs      []Log   `json:"logs,omitempty"`
		NextLogId *string `json:"nextLogId,omitempty"`
		Status    *string `json:"status,omitempty"`
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
	o.Logs = all.Logs
	o.NextLogId = all.NextLogId
	o.Status = all.Status
	return nil
}
