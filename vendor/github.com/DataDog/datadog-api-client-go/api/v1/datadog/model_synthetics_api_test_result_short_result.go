// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsAPITestResultShortResult Result of the last API test run.
type SyntheticsAPITestResultShortResult struct {
	// Describes if the test run has passed or failed.
	Passed *bool `json:"passed,omitempty"`
	// Object containing all metrics and their values collected for a Synthetic API test.
	// Learn more about those metrics in [Synthetics documentation](https://docs.datadoghq.com/synthetics/#metrics).
	Timings *SyntheticsTiming `json:"timings,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsAPITestResultShortResult instantiates a new SyntheticsAPITestResultShortResult object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsAPITestResultShortResult() *SyntheticsAPITestResultShortResult {
	this := SyntheticsAPITestResultShortResult{}
	return &this
}

// NewSyntheticsAPITestResultShortResultWithDefaults instantiates a new SyntheticsAPITestResultShortResult object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsAPITestResultShortResultWithDefaults() *SyntheticsAPITestResultShortResult {
	this := SyntheticsAPITestResultShortResult{}
	return &this
}

// GetPassed returns the Passed field value if set, zero value otherwise.
func (o *SyntheticsAPITestResultShortResult) GetPassed() bool {
	if o == nil || o.Passed == nil {
		var ret bool
		return ret
	}
	return *o.Passed
}

// GetPassedOk returns a tuple with the Passed field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITestResultShortResult) GetPassedOk() (*bool, bool) {
	if o == nil || o.Passed == nil {
		return nil, false
	}
	return o.Passed, true
}

// HasPassed returns a boolean if a field has been set.
func (o *SyntheticsAPITestResultShortResult) HasPassed() bool {
	if o != nil && o.Passed != nil {
		return true
	}

	return false
}

// SetPassed gets a reference to the given bool and assigns it to the Passed field.
func (o *SyntheticsAPITestResultShortResult) SetPassed(v bool) {
	o.Passed = &v
}

// GetTimings returns the Timings field value if set, zero value otherwise.
func (o *SyntheticsAPITestResultShortResult) GetTimings() SyntheticsTiming {
	if o == nil || o.Timings == nil {
		var ret SyntheticsTiming
		return ret
	}
	return *o.Timings
}

// GetTimingsOk returns a tuple with the Timings field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITestResultShortResult) GetTimingsOk() (*SyntheticsTiming, bool) {
	if o == nil || o.Timings == nil {
		return nil, false
	}
	return o.Timings, true
}

// HasTimings returns a boolean if a field has been set.
func (o *SyntheticsAPITestResultShortResult) HasTimings() bool {
	if o != nil && o.Timings != nil {
		return true
	}

	return false
}

// SetTimings gets a reference to the given SyntheticsTiming and assigns it to the Timings field.
func (o *SyntheticsAPITestResultShortResult) SetTimings(v SyntheticsTiming) {
	o.Timings = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsAPITestResultShortResult) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Passed != nil {
		toSerialize["passed"] = o.Passed
	}
	if o.Timings != nil {
		toSerialize["timings"] = o.Timings
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsAPITestResultShortResult) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Passed  *bool             `json:"passed,omitempty"`
		Timings *SyntheticsTiming `json:"timings,omitempty"`
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
	o.Passed = all.Passed
	if all.Timings != nil && all.Timings.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Timings = all.Timings
	return nil
}
