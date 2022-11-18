// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsPrivateLocation Object containing information about the private location to create.
type SyntheticsPrivateLocation struct {
	// Description of the private location.
	Description string `json:"description"`
	// Unique identifier of the private location.
	Id *string `json:"id,omitempty"`
	// Object containing metadata about the private location.
	Metadata *SyntheticsPrivateLocationMetadata `json:"metadata,omitempty"`
	// Name of the private location.
	Name string `json:"name"`
	// Secrets for the private location. Only present in the response when creating the private location.
	Secrets *SyntheticsPrivateLocationSecrets `json:"secrets,omitempty"`
	// Array of tags attached to the private location.
	Tags []string `json:"tags"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsPrivateLocation instantiates a new SyntheticsPrivateLocation object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsPrivateLocation(description string, name string, tags []string) *SyntheticsPrivateLocation {
	this := SyntheticsPrivateLocation{}
	this.Description = description
	this.Name = name
	this.Tags = tags
	return &this
}

// NewSyntheticsPrivateLocationWithDefaults instantiates a new SyntheticsPrivateLocation object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsPrivateLocationWithDefaults() *SyntheticsPrivateLocation {
	this := SyntheticsPrivateLocation{}
	return &this
}

// GetDescription returns the Description field value.
func (o *SyntheticsPrivateLocation) GetDescription() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Description
}

// GetDescriptionOk returns a tuple with the Description field value
// and a boolean to check if the value has been set.
func (o *SyntheticsPrivateLocation) GetDescriptionOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Description, true
}

// SetDescription sets field value.
func (o *SyntheticsPrivateLocation) SetDescription(v string) {
	o.Description = v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *SyntheticsPrivateLocation) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsPrivateLocation) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *SyntheticsPrivateLocation) HasId() bool {
	if o != nil && o.Id != nil {
		return true
	}

	return false
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *SyntheticsPrivateLocation) SetId(v string) {
	o.Id = &v
}

// GetMetadata returns the Metadata field value if set, zero value otherwise.
func (o *SyntheticsPrivateLocation) GetMetadata() SyntheticsPrivateLocationMetadata {
	if o == nil || o.Metadata == nil {
		var ret SyntheticsPrivateLocationMetadata
		return ret
	}
	return *o.Metadata
}

// GetMetadataOk returns a tuple with the Metadata field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsPrivateLocation) GetMetadataOk() (*SyntheticsPrivateLocationMetadata, bool) {
	if o == nil || o.Metadata == nil {
		return nil, false
	}
	return o.Metadata, true
}

// HasMetadata returns a boolean if a field has been set.
func (o *SyntheticsPrivateLocation) HasMetadata() bool {
	if o != nil && o.Metadata != nil {
		return true
	}

	return false
}

// SetMetadata gets a reference to the given SyntheticsPrivateLocationMetadata and assigns it to the Metadata field.
func (o *SyntheticsPrivateLocation) SetMetadata(v SyntheticsPrivateLocationMetadata) {
	o.Metadata = &v
}

// GetName returns the Name field value.
func (o *SyntheticsPrivateLocation) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *SyntheticsPrivateLocation) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *SyntheticsPrivateLocation) SetName(v string) {
	o.Name = v
}

// GetSecrets returns the Secrets field value if set, zero value otherwise.
func (o *SyntheticsPrivateLocation) GetSecrets() SyntheticsPrivateLocationSecrets {
	if o == nil || o.Secrets == nil {
		var ret SyntheticsPrivateLocationSecrets
		return ret
	}
	return *o.Secrets
}

// GetSecretsOk returns a tuple with the Secrets field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsPrivateLocation) GetSecretsOk() (*SyntheticsPrivateLocationSecrets, bool) {
	if o == nil || o.Secrets == nil {
		return nil, false
	}
	return o.Secrets, true
}

// HasSecrets returns a boolean if a field has been set.
func (o *SyntheticsPrivateLocation) HasSecrets() bool {
	if o != nil && o.Secrets != nil {
		return true
	}

	return false
}

// SetSecrets gets a reference to the given SyntheticsPrivateLocationSecrets and assigns it to the Secrets field.
func (o *SyntheticsPrivateLocation) SetSecrets(v SyntheticsPrivateLocationSecrets) {
	o.Secrets = &v
}

// GetTags returns the Tags field value.
func (o *SyntheticsPrivateLocation) GetTags() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value
// and a boolean to check if the value has been set.
func (o *SyntheticsPrivateLocation) GetTagsOk() (*[]string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Tags, true
}

// SetTags sets field value.
func (o *SyntheticsPrivateLocation) SetTags(v []string) {
	o.Tags = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsPrivateLocation) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["description"] = o.Description
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
	if o.Metadata != nil {
		toSerialize["metadata"] = o.Metadata
	}
	toSerialize["name"] = o.Name
	if o.Secrets != nil {
		toSerialize["secrets"] = o.Secrets
	}
	toSerialize["tags"] = o.Tags

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsPrivateLocation) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Description *string   `json:"description"`
		Name        *string   `json:"name"`
		Tags        *[]string `json:"tags"`
	}{}
	all := struct {
		Description string                             `json:"description"`
		Id          *string                            `json:"id,omitempty"`
		Metadata    *SyntheticsPrivateLocationMetadata `json:"metadata,omitempty"`
		Name        string                             `json:"name"`
		Secrets     *SyntheticsPrivateLocationSecrets  `json:"secrets,omitempty"`
		Tags        []string                           `json:"tags"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Description == nil {
		return fmt.Errorf("Required field description missing")
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
	}
	if required.Tags == nil {
		return fmt.Errorf("Required field tags missing")
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
	o.Description = all.Description
	o.Id = all.Id
	if all.Metadata != nil && all.Metadata.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Metadata = all.Metadata
	o.Name = all.Name
	if all.Secrets != nil && all.Secrets.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Secrets = all.Secrets
	o.Tags = all.Tags
	return nil
}
