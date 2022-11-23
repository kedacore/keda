// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsTriggerCITestsResponse Object containing information about the tests triggered.
type SyntheticsTriggerCITestsResponse struct {
	// The public ID of the batch triggered.
	BatchId NullableString `json:"batch_id,omitempty"`
	// List of Synthetics locations.
	Locations []SyntheticsTriggerCITestLocation `json:"locations,omitempty"`
	// Information about the tests runs.
	Results []SyntheticsTriggerCITestRunResult `json:"results,omitempty"`
	// The public IDs of the Synthetics test triggered.
	TriggeredCheckIds []string `json:"triggered_check_ids,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsTriggerCITestsResponse instantiates a new SyntheticsTriggerCITestsResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsTriggerCITestsResponse() *SyntheticsTriggerCITestsResponse {
	this := SyntheticsTriggerCITestsResponse{}
	return &this
}

// NewSyntheticsTriggerCITestsResponseWithDefaults instantiates a new SyntheticsTriggerCITestsResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsTriggerCITestsResponseWithDefaults() *SyntheticsTriggerCITestsResponse {
	this := SyntheticsTriggerCITestsResponse{}
	return &this
}

// GetBatchId returns the BatchId field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SyntheticsTriggerCITestsResponse) GetBatchId() string {
	if o == nil || o.BatchId.Get() == nil {
		var ret string
		return ret
	}
	return *o.BatchId.Get()
}

// GetBatchIdOk returns a tuple with the BatchId field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SyntheticsTriggerCITestsResponse) GetBatchIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.BatchId.Get(), o.BatchId.IsSet()
}

// HasBatchId returns a boolean if a field has been set.
func (o *SyntheticsTriggerCITestsResponse) HasBatchId() bool {
	if o != nil && o.BatchId.IsSet() {
		return true
	}

	return false
}

// SetBatchId gets a reference to the given NullableString and assigns it to the BatchId field.
func (o *SyntheticsTriggerCITestsResponse) SetBatchId(v string) {
	o.BatchId.Set(&v)
}

// SetBatchIdNil sets the value for BatchId to be an explicit nil.
func (o *SyntheticsTriggerCITestsResponse) SetBatchIdNil() {
	o.BatchId.Set(nil)
}

// UnsetBatchId ensures that no value is present for BatchId, not even an explicit nil.
func (o *SyntheticsTriggerCITestsResponse) UnsetBatchId() {
	o.BatchId.Unset()
}

// GetLocations returns the Locations field value if set, zero value otherwise.
func (o *SyntheticsTriggerCITestsResponse) GetLocations() []SyntheticsTriggerCITestLocation {
	if o == nil || o.Locations == nil {
		var ret []SyntheticsTriggerCITestLocation
		return ret
	}
	return o.Locations
}

// GetLocationsOk returns a tuple with the Locations field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTriggerCITestsResponse) GetLocationsOk() (*[]SyntheticsTriggerCITestLocation, bool) {
	if o == nil || o.Locations == nil {
		return nil, false
	}
	return &o.Locations, true
}

// HasLocations returns a boolean if a field has been set.
func (o *SyntheticsTriggerCITestsResponse) HasLocations() bool {
	if o != nil && o.Locations != nil {
		return true
	}

	return false
}

// SetLocations gets a reference to the given []SyntheticsTriggerCITestLocation and assigns it to the Locations field.
func (o *SyntheticsTriggerCITestsResponse) SetLocations(v []SyntheticsTriggerCITestLocation) {
	o.Locations = v
}

// GetResults returns the Results field value if set, zero value otherwise.
func (o *SyntheticsTriggerCITestsResponse) GetResults() []SyntheticsTriggerCITestRunResult {
	if o == nil || o.Results == nil {
		var ret []SyntheticsTriggerCITestRunResult
		return ret
	}
	return o.Results
}

// GetResultsOk returns a tuple with the Results field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTriggerCITestsResponse) GetResultsOk() (*[]SyntheticsTriggerCITestRunResult, bool) {
	if o == nil || o.Results == nil {
		return nil, false
	}
	return &o.Results, true
}

// HasResults returns a boolean if a field has been set.
func (o *SyntheticsTriggerCITestsResponse) HasResults() bool {
	if o != nil && o.Results != nil {
		return true
	}

	return false
}

// SetResults gets a reference to the given []SyntheticsTriggerCITestRunResult and assigns it to the Results field.
func (o *SyntheticsTriggerCITestsResponse) SetResults(v []SyntheticsTriggerCITestRunResult) {
	o.Results = v
}

// GetTriggeredCheckIds returns the TriggeredCheckIds field value if set, zero value otherwise.
func (o *SyntheticsTriggerCITestsResponse) GetTriggeredCheckIds() []string {
	if o == nil || o.TriggeredCheckIds == nil {
		var ret []string
		return ret
	}
	return o.TriggeredCheckIds
}

// GetTriggeredCheckIdsOk returns a tuple with the TriggeredCheckIds field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTriggerCITestsResponse) GetTriggeredCheckIdsOk() (*[]string, bool) {
	if o == nil || o.TriggeredCheckIds == nil {
		return nil, false
	}
	return &o.TriggeredCheckIds, true
}

// HasTriggeredCheckIds returns a boolean if a field has been set.
func (o *SyntheticsTriggerCITestsResponse) HasTriggeredCheckIds() bool {
	if o != nil && o.TriggeredCheckIds != nil {
		return true
	}

	return false
}

// SetTriggeredCheckIds gets a reference to the given []string and assigns it to the TriggeredCheckIds field.
func (o *SyntheticsTriggerCITestsResponse) SetTriggeredCheckIds(v []string) {
	o.TriggeredCheckIds = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsTriggerCITestsResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.BatchId.IsSet() {
		toSerialize["batch_id"] = o.BatchId.Get()
	}
	if o.Locations != nil {
		toSerialize["locations"] = o.Locations
	}
	if o.Results != nil {
		toSerialize["results"] = o.Results
	}
	if o.TriggeredCheckIds != nil {
		toSerialize["triggered_check_ids"] = o.TriggeredCheckIds
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsTriggerCITestsResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		BatchId           NullableString                     `json:"batch_id,omitempty"`
		Locations         []SyntheticsTriggerCITestLocation  `json:"locations,omitempty"`
		Results           []SyntheticsTriggerCITestRunResult `json:"results,omitempty"`
		TriggeredCheckIds []string                           `json:"triggered_check_ids,omitempty"`
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
	o.BatchId = all.BatchId
	o.Locations = all.Locations
	o.Results = all.Results
	o.TriggeredCheckIds = all.TriggeredCheckIds
	return nil
}
