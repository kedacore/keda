// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsCIBatchMetadata Metadata for the Synthetics tests run.
type SyntheticsCIBatchMetadata struct {
	// Description of the CI provider.
	Ci *SyntheticsCIBatchMetadataCI `json:"ci,omitempty"`
	// Git information.
	Git *SyntheticsCIBatchMetadataGit `json:"git,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsCIBatchMetadata instantiates a new SyntheticsCIBatchMetadata object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsCIBatchMetadata() *SyntheticsCIBatchMetadata {
	this := SyntheticsCIBatchMetadata{}
	return &this
}

// NewSyntheticsCIBatchMetadataWithDefaults instantiates a new SyntheticsCIBatchMetadata object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsCIBatchMetadataWithDefaults() *SyntheticsCIBatchMetadata {
	this := SyntheticsCIBatchMetadata{}
	return &this
}

// GetCi returns the Ci field value if set, zero value otherwise.
func (o *SyntheticsCIBatchMetadata) GetCi() SyntheticsCIBatchMetadataCI {
	if o == nil || o.Ci == nil {
		var ret SyntheticsCIBatchMetadataCI
		return ret
	}
	return *o.Ci
}

// GetCiOk returns a tuple with the Ci field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsCIBatchMetadata) GetCiOk() (*SyntheticsCIBatchMetadataCI, bool) {
	if o == nil || o.Ci == nil {
		return nil, false
	}
	return o.Ci, true
}

// HasCi returns a boolean if a field has been set.
func (o *SyntheticsCIBatchMetadata) HasCi() bool {
	if o != nil && o.Ci != nil {
		return true
	}

	return false
}

// SetCi gets a reference to the given SyntheticsCIBatchMetadataCI and assigns it to the Ci field.
func (o *SyntheticsCIBatchMetadata) SetCi(v SyntheticsCIBatchMetadataCI) {
	o.Ci = &v
}

// GetGit returns the Git field value if set, zero value otherwise.
func (o *SyntheticsCIBatchMetadata) GetGit() SyntheticsCIBatchMetadataGit {
	if o == nil || o.Git == nil {
		var ret SyntheticsCIBatchMetadataGit
		return ret
	}
	return *o.Git
}

// GetGitOk returns a tuple with the Git field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsCIBatchMetadata) GetGitOk() (*SyntheticsCIBatchMetadataGit, bool) {
	if o == nil || o.Git == nil {
		return nil, false
	}
	return o.Git, true
}

// HasGit returns a boolean if a field has been set.
func (o *SyntheticsCIBatchMetadata) HasGit() bool {
	if o != nil && o.Git != nil {
		return true
	}

	return false
}

// SetGit gets a reference to the given SyntheticsCIBatchMetadataGit and assigns it to the Git field.
func (o *SyntheticsCIBatchMetadata) SetGit(v SyntheticsCIBatchMetadataGit) {
	o.Git = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsCIBatchMetadata) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Ci != nil {
		toSerialize["ci"] = o.Ci
	}
	if o.Git != nil {
		toSerialize["git"] = o.Git
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsCIBatchMetadata) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Ci  *SyntheticsCIBatchMetadataCI  `json:"ci,omitempty"`
		Git *SyntheticsCIBatchMetadataGit `json:"git,omitempty"`
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
	if all.Ci != nil && all.Ci.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Ci = all.Ci
	if all.Git != nil && all.Git.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Git = all.Git
	return nil
}
