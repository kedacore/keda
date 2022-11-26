// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// LogsIndexListResponse Object with all Index configurations for a given organization.
type LogsIndexListResponse struct {
	// Array of Log index configurations.
	Indexes []LogsIndex `json:"indexes,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsIndexListResponse instantiates a new LogsIndexListResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsIndexListResponse() *LogsIndexListResponse {
	this := LogsIndexListResponse{}
	return &this
}

// NewLogsIndexListResponseWithDefaults instantiates a new LogsIndexListResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsIndexListResponseWithDefaults() *LogsIndexListResponse {
	this := LogsIndexListResponse{}
	return &this
}

// GetIndexes returns the Indexes field value if set, zero value otherwise.
func (o *LogsIndexListResponse) GetIndexes() []LogsIndex {
	if o == nil || o.Indexes == nil {
		var ret []LogsIndex
		return ret
	}
	return o.Indexes
}

// GetIndexesOk returns a tuple with the Indexes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsIndexListResponse) GetIndexesOk() (*[]LogsIndex, bool) {
	if o == nil || o.Indexes == nil {
		return nil, false
	}
	return &o.Indexes, true
}

// HasIndexes returns a boolean if a field has been set.
func (o *LogsIndexListResponse) HasIndexes() bool {
	if o != nil && o.Indexes != nil {
		return true
	}

	return false
}

// SetIndexes gets a reference to the given []LogsIndex and assigns it to the Indexes field.
func (o *LogsIndexListResponse) SetIndexes(v []LogsIndex) {
	o.Indexes = v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsIndexListResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Indexes != nil {
		toSerialize["indexes"] = o.Indexes
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsIndexListResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Indexes []LogsIndex `json:"indexes,omitempty"`
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
	o.Indexes = all.Indexes
	return nil
}
