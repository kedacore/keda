// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsBatchDetailsData Wrapper object that contains the details of a batch.
type SyntheticsBatchDetailsData struct {
	// Metadata for the Synthetics tests run.
	Metadata *SyntheticsCIBatchMetadata `json:"metadata,omitempty"`
	// List of results for the batch.
	Results []SyntheticsBatchResult `json:"results,omitempty"`
	// Determines whether or not the batch has passed, failed, or is in progress.
	Status *SyntheticsStatus `json:"status,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsBatchDetailsData instantiates a new SyntheticsBatchDetailsData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsBatchDetailsData() *SyntheticsBatchDetailsData {
	this := SyntheticsBatchDetailsData{}
	return &this
}

// NewSyntheticsBatchDetailsDataWithDefaults instantiates a new SyntheticsBatchDetailsData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsBatchDetailsDataWithDefaults() *SyntheticsBatchDetailsData {
	this := SyntheticsBatchDetailsData{}
	return &this
}

// GetMetadata returns the Metadata field value if set, zero value otherwise.
func (o *SyntheticsBatchDetailsData) GetMetadata() SyntheticsCIBatchMetadata {
	if o == nil || o.Metadata == nil {
		var ret SyntheticsCIBatchMetadata
		return ret
	}
	return *o.Metadata
}

// GetMetadataOk returns a tuple with the Metadata field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBatchDetailsData) GetMetadataOk() (*SyntheticsCIBatchMetadata, bool) {
	if o == nil || o.Metadata == nil {
		return nil, false
	}
	return o.Metadata, true
}

// HasMetadata returns a boolean if a field has been set.
func (o *SyntheticsBatchDetailsData) HasMetadata() bool {
	if o != nil && o.Metadata != nil {
		return true
	}

	return false
}

// SetMetadata gets a reference to the given SyntheticsCIBatchMetadata and assigns it to the Metadata field.
func (o *SyntheticsBatchDetailsData) SetMetadata(v SyntheticsCIBatchMetadata) {
	o.Metadata = &v
}

// GetResults returns the Results field value if set, zero value otherwise.
func (o *SyntheticsBatchDetailsData) GetResults() []SyntheticsBatchResult {
	if o == nil || o.Results == nil {
		var ret []SyntheticsBatchResult
		return ret
	}
	return o.Results
}

// GetResultsOk returns a tuple with the Results field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBatchDetailsData) GetResultsOk() (*[]SyntheticsBatchResult, bool) {
	if o == nil || o.Results == nil {
		return nil, false
	}
	return &o.Results, true
}

// HasResults returns a boolean if a field has been set.
func (o *SyntheticsBatchDetailsData) HasResults() bool {
	if o != nil && o.Results != nil {
		return true
	}

	return false
}

// SetResults gets a reference to the given []SyntheticsBatchResult and assigns it to the Results field.
func (o *SyntheticsBatchDetailsData) SetResults(v []SyntheticsBatchResult) {
	o.Results = v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *SyntheticsBatchDetailsData) GetStatus() SyntheticsStatus {
	if o == nil || o.Status == nil {
		var ret SyntheticsStatus
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBatchDetailsData) GetStatusOk() (*SyntheticsStatus, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *SyntheticsBatchDetailsData) HasStatus() bool {
	if o != nil && o.Status != nil {
		return true
	}

	return false
}

// SetStatus gets a reference to the given SyntheticsStatus and assigns it to the Status field.
func (o *SyntheticsBatchDetailsData) SetStatus(v SyntheticsStatus) {
	o.Status = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsBatchDetailsData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Metadata != nil {
		toSerialize["metadata"] = o.Metadata
	}
	if o.Results != nil {
		toSerialize["results"] = o.Results
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
func (o *SyntheticsBatchDetailsData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Metadata *SyntheticsCIBatchMetadata `json:"metadata,omitempty"`
		Results  []SyntheticsBatchResult    `json:"results,omitempty"`
		Status   *SyntheticsStatus          `json:"status,omitempty"`
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
	if v := all.Status; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if all.Metadata != nil && all.Metadata.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Metadata = all.Metadata
	o.Results = all.Results
	o.Status = all.Status
	return nil
}
