// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsCIBatchMetadataPipeline Description of the CI pipeline.
type SyntheticsCIBatchMetadataPipeline struct {
	// URL of the pipeline.
	Url *string `json:"url,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsCIBatchMetadataPipeline instantiates a new SyntheticsCIBatchMetadataPipeline object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsCIBatchMetadataPipeline() *SyntheticsCIBatchMetadataPipeline {
	this := SyntheticsCIBatchMetadataPipeline{}
	return &this
}

// NewSyntheticsCIBatchMetadataPipelineWithDefaults instantiates a new SyntheticsCIBatchMetadataPipeline object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsCIBatchMetadataPipelineWithDefaults() *SyntheticsCIBatchMetadataPipeline {
	this := SyntheticsCIBatchMetadataPipeline{}
	return &this
}

// GetUrl returns the Url field value if set, zero value otherwise.
func (o *SyntheticsCIBatchMetadataPipeline) GetUrl() string {
	if o == nil || o.Url == nil {
		var ret string
		return ret
	}
	return *o.Url
}

// GetUrlOk returns a tuple with the Url field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsCIBatchMetadataPipeline) GetUrlOk() (*string, bool) {
	if o == nil || o.Url == nil {
		return nil, false
	}
	return o.Url, true
}

// HasUrl returns a boolean if a field has been set.
func (o *SyntheticsCIBatchMetadataPipeline) HasUrl() bool {
	if o != nil && o.Url != nil {
		return true
	}

	return false
}

// SetUrl gets a reference to the given string and assigns it to the Url field.
func (o *SyntheticsCIBatchMetadataPipeline) SetUrl(v string) {
	o.Url = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsCIBatchMetadataPipeline) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Url != nil {
		toSerialize["url"] = o.Url
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsCIBatchMetadataPipeline) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Url *string `json:"url,omitempty"`
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
	o.Url = all.Url
	return nil
}
