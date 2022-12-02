// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SLOListResponse A response with one or more service level objective.
type SLOListResponse struct {
	// An array of service level objective objects.
	Data []ServiceLevelObjective `json:"data,omitempty"`
	// An array of error messages. Each endpoint documents how/whether this field is
	// used.
	Errors []string `json:"errors,omitempty"`
	// The metadata object containing additional information about the list of SLOs.
	Metadata *SLOListResponseMetadata `json:"metadata,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSLOListResponse instantiates a new SLOListResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOListResponse() *SLOListResponse {
	this := SLOListResponse{}
	return &this
}

// NewSLOListResponseWithDefaults instantiates a new SLOListResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOListResponseWithDefaults() *SLOListResponse {
	this := SLOListResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *SLOListResponse) GetData() []ServiceLevelObjective {
	if o == nil || o.Data == nil {
		var ret []ServiceLevelObjective
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOListResponse) GetDataOk() (*[]ServiceLevelObjective, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return &o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *SLOListResponse) HasData() bool {
	if o != nil && o.Data != nil {
		return true
	}

	return false
}

// SetData gets a reference to the given []ServiceLevelObjective and assigns it to the Data field.
func (o *SLOListResponse) SetData(v []ServiceLevelObjective) {
	o.Data = v
}

// GetErrors returns the Errors field value if set, zero value otherwise.
func (o *SLOListResponse) GetErrors() []string {
	if o == nil || o.Errors == nil {
		var ret []string
		return ret
	}
	return o.Errors
}

// GetErrorsOk returns a tuple with the Errors field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOListResponse) GetErrorsOk() (*[]string, bool) {
	if o == nil || o.Errors == nil {
		return nil, false
	}
	return &o.Errors, true
}

// HasErrors returns a boolean if a field has been set.
func (o *SLOListResponse) HasErrors() bool {
	if o != nil && o.Errors != nil {
		return true
	}

	return false
}

// SetErrors gets a reference to the given []string and assigns it to the Errors field.
func (o *SLOListResponse) SetErrors(v []string) {
	o.Errors = v
}

// GetMetadata returns the Metadata field value if set, zero value otherwise.
func (o *SLOListResponse) GetMetadata() SLOListResponseMetadata {
	if o == nil || o.Metadata == nil {
		var ret SLOListResponseMetadata
		return ret
	}
	return *o.Metadata
}

// GetMetadataOk returns a tuple with the Metadata field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOListResponse) GetMetadataOk() (*SLOListResponseMetadata, bool) {
	if o == nil || o.Metadata == nil {
		return nil, false
	}
	return o.Metadata, true
}

// HasMetadata returns a boolean if a field has been set.
func (o *SLOListResponse) HasMetadata() bool {
	if o != nil && o.Metadata != nil {
		return true
	}

	return false
}

// SetMetadata gets a reference to the given SLOListResponseMetadata and assigns it to the Metadata field.
func (o *SLOListResponse) SetMetadata(v SLOListResponseMetadata) {
	o.Metadata = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOListResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Data != nil {
		toSerialize["data"] = o.Data
	}
	if o.Errors != nil {
		toSerialize["errors"] = o.Errors
	}
	if o.Metadata != nil {
		toSerialize["metadata"] = o.Metadata
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOListResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data     []ServiceLevelObjective  `json:"data,omitempty"`
		Errors   []string                 `json:"errors,omitempty"`
		Metadata *SLOListResponseMetadata `json:"metadata,omitempty"`
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
	o.Data = all.Data
	o.Errors = all.Errors
	if all.Metadata != nil && all.Metadata.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Metadata = all.Metadata
	return nil
}
