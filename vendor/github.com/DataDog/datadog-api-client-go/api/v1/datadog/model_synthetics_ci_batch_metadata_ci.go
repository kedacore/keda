// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsCIBatchMetadataCI Description of the CI provider.
type SyntheticsCIBatchMetadataCI struct {
	// Description of the CI pipeline.
	Pipeline *SyntheticsCIBatchMetadataPipeline `json:"pipeline,omitempty"`
	// Description of the CI provider.
	Provider *SyntheticsCIBatchMetadataProvider `json:"provider,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsCIBatchMetadataCI instantiates a new SyntheticsCIBatchMetadataCI object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsCIBatchMetadataCI() *SyntheticsCIBatchMetadataCI {
	this := SyntheticsCIBatchMetadataCI{}
	return &this
}

// NewSyntheticsCIBatchMetadataCIWithDefaults instantiates a new SyntheticsCIBatchMetadataCI object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsCIBatchMetadataCIWithDefaults() *SyntheticsCIBatchMetadataCI {
	this := SyntheticsCIBatchMetadataCI{}
	return &this
}

// GetPipeline returns the Pipeline field value if set, zero value otherwise.
func (o *SyntheticsCIBatchMetadataCI) GetPipeline() SyntheticsCIBatchMetadataPipeline {
	if o == nil || o.Pipeline == nil {
		var ret SyntheticsCIBatchMetadataPipeline
		return ret
	}
	return *o.Pipeline
}

// GetPipelineOk returns a tuple with the Pipeline field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsCIBatchMetadataCI) GetPipelineOk() (*SyntheticsCIBatchMetadataPipeline, bool) {
	if o == nil || o.Pipeline == nil {
		return nil, false
	}
	return o.Pipeline, true
}

// HasPipeline returns a boolean if a field has been set.
func (o *SyntheticsCIBatchMetadataCI) HasPipeline() bool {
	if o != nil && o.Pipeline != nil {
		return true
	}

	return false
}

// SetPipeline gets a reference to the given SyntheticsCIBatchMetadataPipeline and assigns it to the Pipeline field.
func (o *SyntheticsCIBatchMetadataCI) SetPipeline(v SyntheticsCIBatchMetadataPipeline) {
	o.Pipeline = &v
}

// GetProvider returns the Provider field value if set, zero value otherwise.
func (o *SyntheticsCIBatchMetadataCI) GetProvider() SyntheticsCIBatchMetadataProvider {
	if o == nil || o.Provider == nil {
		var ret SyntheticsCIBatchMetadataProvider
		return ret
	}
	return *o.Provider
}

// GetProviderOk returns a tuple with the Provider field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsCIBatchMetadataCI) GetProviderOk() (*SyntheticsCIBatchMetadataProvider, bool) {
	if o == nil || o.Provider == nil {
		return nil, false
	}
	return o.Provider, true
}

// HasProvider returns a boolean if a field has been set.
func (o *SyntheticsCIBatchMetadataCI) HasProvider() bool {
	if o != nil && o.Provider != nil {
		return true
	}

	return false
}

// SetProvider gets a reference to the given SyntheticsCIBatchMetadataProvider and assigns it to the Provider field.
func (o *SyntheticsCIBatchMetadataCI) SetProvider(v SyntheticsCIBatchMetadataProvider) {
	o.Provider = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsCIBatchMetadataCI) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Pipeline != nil {
		toSerialize["pipeline"] = o.Pipeline
	}
	if o.Provider != nil {
		toSerialize["provider"] = o.Provider
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsCIBatchMetadataCI) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Pipeline *SyntheticsCIBatchMetadataPipeline `json:"pipeline,omitempty"`
		Provider *SyntheticsCIBatchMetadataProvider `json:"provider,omitempty"`
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
	if all.Pipeline != nil && all.Pipeline.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Pipeline = all.Pipeline
	if all.Provider != nil && all.Provider.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Provider = all.Provider
	return nil
}
