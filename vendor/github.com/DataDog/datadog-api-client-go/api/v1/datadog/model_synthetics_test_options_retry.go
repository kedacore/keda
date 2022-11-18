// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsTestOptionsRetry Object describing the retry strategy to apply to a Synthetic test.
type SyntheticsTestOptionsRetry struct {
	// Number of times a test needs to be retried before marking a
	// location as failed. Defaults to 0.
	Count *int64 `json:"count,omitempty"`
	// Time interval between retries (in milliseconds). Defaults to
	// 300ms.
	Interval *float64 `json:"interval,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsTestOptionsRetry instantiates a new SyntheticsTestOptionsRetry object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsTestOptionsRetry() *SyntheticsTestOptionsRetry {
	this := SyntheticsTestOptionsRetry{}
	return &this
}

// NewSyntheticsTestOptionsRetryWithDefaults instantiates a new SyntheticsTestOptionsRetry object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsTestOptionsRetryWithDefaults() *SyntheticsTestOptionsRetry {
	this := SyntheticsTestOptionsRetry{}
	return &this
}

// GetCount returns the Count field value if set, zero value otherwise.
func (o *SyntheticsTestOptionsRetry) GetCount() int64 {
	if o == nil || o.Count == nil {
		var ret int64
		return ret
	}
	return *o.Count
}

// GetCountOk returns a tuple with the Count field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptionsRetry) GetCountOk() (*int64, bool) {
	if o == nil || o.Count == nil {
		return nil, false
	}
	return o.Count, true
}

// HasCount returns a boolean if a field has been set.
func (o *SyntheticsTestOptionsRetry) HasCount() bool {
	if o != nil && o.Count != nil {
		return true
	}

	return false
}

// SetCount gets a reference to the given int64 and assigns it to the Count field.
func (o *SyntheticsTestOptionsRetry) SetCount(v int64) {
	o.Count = &v
}

// GetInterval returns the Interval field value if set, zero value otherwise.
func (o *SyntheticsTestOptionsRetry) GetInterval() float64 {
	if o == nil || o.Interval == nil {
		var ret float64
		return ret
	}
	return *o.Interval
}

// GetIntervalOk returns a tuple with the Interval field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptionsRetry) GetIntervalOk() (*float64, bool) {
	if o == nil || o.Interval == nil {
		return nil, false
	}
	return o.Interval, true
}

// HasInterval returns a boolean if a field has been set.
func (o *SyntheticsTestOptionsRetry) HasInterval() bool {
	if o != nil && o.Interval != nil {
		return true
	}

	return false
}

// SetInterval gets a reference to the given float64 and assigns it to the Interval field.
func (o *SyntheticsTestOptionsRetry) SetInterval(v float64) {
	o.Interval = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsTestOptionsRetry) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Count != nil {
		toSerialize["count"] = o.Count
	}
	if o.Interval != nil {
		toSerialize["interval"] = o.Interval
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsTestOptionsRetry) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Count    *int64   `json:"count,omitempty"`
		Interval *float64 `json:"interval,omitempty"`
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
	o.Count = all.Count
	o.Interval = all.Interval
	return nil
}
