// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsExclusionFilter Exclusion filter is defined by a query, a sampling rule, and a active/inactive toggle.
type LogsExclusionFilter struct {
	// Default query is `*`, meaning all logs flowing in the index would be excluded.
	// Scope down exclusion filter to only a subset of logs with a log query.
	Query *string `json:"query,omitempty"`
	// Sample rate to apply to logs going through this exclusion filter,
	// a value of 1.0 excludes all logs matching the query.
	SampleRate float64 `json:"sample_rate"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsExclusionFilter instantiates a new LogsExclusionFilter object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsExclusionFilter(sampleRate float64) *LogsExclusionFilter {
	this := LogsExclusionFilter{}
	this.SampleRate = sampleRate
	return &this
}

// NewLogsExclusionFilterWithDefaults instantiates a new LogsExclusionFilter object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsExclusionFilterWithDefaults() *LogsExclusionFilter {
	this := LogsExclusionFilter{}
	return &this
}

// GetQuery returns the Query field value if set, zero value otherwise.
func (o *LogsExclusionFilter) GetQuery() string {
	if o == nil || o.Query == nil {
		var ret string
		return ret
	}
	return *o.Query
}

// GetQueryOk returns a tuple with the Query field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsExclusionFilter) GetQueryOk() (*string, bool) {
	if o == nil || o.Query == nil {
		return nil, false
	}
	return o.Query, true
}

// HasQuery returns a boolean if a field has been set.
func (o *LogsExclusionFilter) HasQuery() bool {
	if o != nil && o.Query != nil {
		return true
	}

	return false
}

// SetQuery gets a reference to the given string and assigns it to the Query field.
func (o *LogsExclusionFilter) SetQuery(v string) {
	o.Query = &v
}

// GetSampleRate returns the SampleRate field value.
func (o *LogsExclusionFilter) GetSampleRate() float64 {
	if o == nil {
		var ret float64
		return ret
	}
	return o.SampleRate
}

// GetSampleRateOk returns a tuple with the SampleRate field value
// and a boolean to check if the value has been set.
func (o *LogsExclusionFilter) GetSampleRateOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.SampleRate, true
}

// SetSampleRate sets field value.
func (o *LogsExclusionFilter) SetSampleRate(v float64) {
	o.SampleRate = v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsExclusionFilter) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Query != nil {
		toSerialize["query"] = o.Query
	}
	toSerialize["sample_rate"] = o.SampleRate

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsExclusionFilter) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		SampleRate *float64 `json:"sample_rate"`
	}{}
	all := struct {
		Query      *string `json:"query,omitempty"`
		SampleRate float64 `json:"sample_rate"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.SampleRate == nil {
		return fmt.Errorf("Required field sample_rate missing")
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
	o.Query = all.Query
	o.SampleRate = all.SampleRate
	return nil
}
