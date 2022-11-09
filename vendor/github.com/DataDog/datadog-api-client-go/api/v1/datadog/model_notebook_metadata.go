// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// NotebookMetadata Metadata associated with the notebook.
type NotebookMetadata struct {
	// Whether or not the notebook is a template.
	IsTemplate *bool `json:"is_template,omitempty"`
	// Whether or not the notebook takes snapshot image backups of the notebook's fixed-time graphs.
	TakeSnapshots *bool `json:"take_snapshots,omitempty"`
	// Metadata type of the notebook.
	Type NullableNotebookMetadataType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewNotebookMetadata instantiates a new NotebookMetadata object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewNotebookMetadata() *NotebookMetadata {
	this := NotebookMetadata{}
	var isTemplate bool = false
	this.IsTemplate = &isTemplate
	var takeSnapshots bool = false
	this.TakeSnapshots = &takeSnapshots
	return &this
}

// NewNotebookMetadataWithDefaults instantiates a new NotebookMetadata object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewNotebookMetadataWithDefaults() *NotebookMetadata {
	this := NotebookMetadata{}
	var isTemplate bool = false
	this.IsTemplate = &isTemplate
	var takeSnapshots bool = false
	this.TakeSnapshots = &takeSnapshots
	return &this
}

// GetIsTemplate returns the IsTemplate field value if set, zero value otherwise.
func (o *NotebookMetadata) GetIsTemplate() bool {
	if o == nil || o.IsTemplate == nil {
		var ret bool
		return ret
	}
	return *o.IsTemplate
}

// GetIsTemplateOk returns a tuple with the IsTemplate field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NotebookMetadata) GetIsTemplateOk() (*bool, bool) {
	if o == nil || o.IsTemplate == nil {
		return nil, false
	}
	return o.IsTemplate, true
}

// HasIsTemplate returns a boolean if a field has been set.
func (o *NotebookMetadata) HasIsTemplate() bool {
	if o != nil && o.IsTemplate != nil {
		return true
	}

	return false
}

// SetIsTemplate gets a reference to the given bool and assigns it to the IsTemplate field.
func (o *NotebookMetadata) SetIsTemplate(v bool) {
	o.IsTemplate = &v
}

// GetTakeSnapshots returns the TakeSnapshots field value if set, zero value otherwise.
func (o *NotebookMetadata) GetTakeSnapshots() bool {
	if o == nil || o.TakeSnapshots == nil {
		var ret bool
		return ret
	}
	return *o.TakeSnapshots
}

// GetTakeSnapshotsOk returns a tuple with the TakeSnapshots field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NotebookMetadata) GetTakeSnapshotsOk() (*bool, bool) {
	if o == nil || o.TakeSnapshots == nil {
		return nil, false
	}
	return o.TakeSnapshots, true
}

// HasTakeSnapshots returns a boolean if a field has been set.
func (o *NotebookMetadata) HasTakeSnapshots() bool {
	if o != nil && o.TakeSnapshots != nil {
		return true
	}

	return false
}

// SetTakeSnapshots gets a reference to the given bool and assigns it to the TakeSnapshots field.
func (o *NotebookMetadata) SetTakeSnapshots(v bool) {
	o.TakeSnapshots = &v
}

// GetType returns the Type field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *NotebookMetadata) GetType() NotebookMetadataType {
	if o == nil || o.Type.Get() == nil {
		var ret NotebookMetadataType
		return ret
	}
	return *o.Type.Get()
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *NotebookMetadata) GetTypeOk() (*NotebookMetadataType, bool) {
	if o == nil {
		return nil, false
	}
	return o.Type.Get(), o.Type.IsSet()
}

// HasType returns a boolean if a field has been set.
func (o *NotebookMetadata) HasType() bool {
	if o != nil && o.Type.IsSet() {
		return true
	}

	return false
}

// SetType gets a reference to the given NullableNotebookMetadataType and assigns it to the Type field.
func (o *NotebookMetadata) SetType(v NotebookMetadataType) {
	o.Type.Set(&v)
}

// SetTypeNil sets the value for Type to be an explicit nil.
func (o *NotebookMetadata) SetTypeNil() {
	o.Type.Set(nil)
}

// UnsetType ensures that no value is present for Type, not even an explicit nil.
func (o *NotebookMetadata) UnsetType() {
	o.Type.Unset()
}

// MarshalJSON serializes the struct using spec logic.
func (o NotebookMetadata) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.IsTemplate != nil {
		toSerialize["is_template"] = o.IsTemplate
	}
	if o.TakeSnapshots != nil {
		toSerialize["take_snapshots"] = o.TakeSnapshots
	}
	if o.Type.IsSet() {
		toSerialize["type"] = o.Type.Get()
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *NotebookMetadata) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		IsTemplate    *bool                        `json:"is_template,omitempty"`
		TakeSnapshots *bool                        `json:"take_snapshots,omitempty"`
		Type          NullableNotebookMetadataType `json:"type,omitempty"`
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
	if v := all.Type; v.Get() != nil && !v.Get().IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.IsTemplate = all.IsTemplate
	o.TakeSnapshots = all.TakeSnapshots
	o.Type = all.Type
	return nil
}
