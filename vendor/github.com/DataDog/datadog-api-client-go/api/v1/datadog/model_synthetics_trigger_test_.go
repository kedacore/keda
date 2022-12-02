// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsTriggerTest Test configuration for Synthetics
type SyntheticsTriggerTest struct {
	// Metadata for the Synthetics tests run.
	Metadata *SyntheticsCIBatchMetadata `json:"metadata,omitempty"`
	// The public ID of the Synthetics test to trigger.
	PublicId string `json:"public_id"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsTriggerTest instantiates a new SyntheticsTriggerTest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsTriggerTest(publicId string) *SyntheticsTriggerTest {
	this := SyntheticsTriggerTest{}
	this.PublicId = publicId
	return &this
}

// NewSyntheticsTriggerTestWithDefaults instantiates a new SyntheticsTriggerTest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsTriggerTestWithDefaults() *SyntheticsTriggerTest {
	this := SyntheticsTriggerTest{}
	return &this
}

// GetMetadata returns the Metadata field value if set, zero value otherwise.
func (o *SyntheticsTriggerTest) GetMetadata() SyntheticsCIBatchMetadata {
	if o == nil || o.Metadata == nil {
		var ret SyntheticsCIBatchMetadata
		return ret
	}
	return *o.Metadata
}

// GetMetadataOk returns a tuple with the Metadata field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTriggerTest) GetMetadataOk() (*SyntheticsCIBatchMetadata, bool) {
	if o == nil || o.Metadata == nil {
		return nil, false
	}
	return o.Metadata, true
}

// HasMetadata returns a boolean if a field has been set.
func (o *SyntheticsTriggerTest) HasMetadata() bool {
	if o != nil && o.Metadata != nil {
		return true
	}

	return false
}

// SetMetadata gets a reference to the given SyntheticsCIBatchMetadata and assigns it to the Metadata field.
func (o *SyntheticsTriggerTest) SetMetadata(v SyntheticsCIBatchMetadata) {
	o.Metadata = &v
}

// GetPublicId returns the PublicId field value.
func (o *SyntheticsTriggerTest) GetPublicId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value
// and a boolean to check if the value has been set.
func (o *SyntheticsTriggerTest) GetPublicIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.PublicId, true
}

// SetPublicId sets field value.
func (o *SyntheticsTriggerTest) SetPublicId(v string) {
	o.PublicId = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsTriggerTest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Metadata != nil {
		toSerialize["metadata"] = o.Metadata
	}
	toSerialize["public_id"] = o.PublicId

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsTriggerTest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		PublicId *string `json:"public_id"`
	}{}
	all := struct {
		Metadata *SyntheticsCIBatchMetadata `json:"metadata,omitempty"`
		PublicId string                     `json:"public_id"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.PublicId == nil {
		return fmt.Errorf("Required field public_id missing")
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
	if all.Metadata != nil && all.Metadata.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Metadata = all.Metadata
	o.PublicId = all.PublicId
	return nil
}
