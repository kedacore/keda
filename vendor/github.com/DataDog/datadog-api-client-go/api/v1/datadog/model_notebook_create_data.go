// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// NotebookCreateData The data for a notebook create request.
type NotebookCreateData struct {
	// The data attributes of a notebook.
	Attributes NotebookCreateDataAttributes `json:"attributes"`
	// Type of the Notebook resource.
	Type NotebookResourceType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewNotebookCreateData instantiates a new NotebookCreateData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewNotebookCreateData(attributes NotebookCreateDataAttributes, typeVar NotebookResourceType) *NotebookCreateData {
	this := NotebookCreateData{}
	this.Attributes = attributes
	this.Type = typeVar
	return &this
}

// NewNotebookCreateDataWithDefaults instantiates a new NotebookCreateData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewNotebookCreateDataWithDefaults() *NotebookCreateData {
	this := NotebookCreateData{}
	var typeVar NotebookResourceType = NOTEBOOKRESOURCETYPE_NOTEBOOKS
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value.
func (o *NotebookCreateData) GetAttributes() NotebookCreateDataAttributes {
	if o == nil {
		var ret NotebookCreateDataAttributes
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value
// and a boolean to check if the value has been set.
func (o *NotebookCreateData) GetAttributesOk() (*NotebookCreateDataAttributes, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// SetAttributes sets field value.
func (o *NotebookCreateData) SetAttributes(v NotebookCreateDataAttributes) {
	o.Attributes = v
}

// GetType returns the Type field value.
func (o *NotebookCreateData) GetType() NotebookResourceType {
	if o == nil {
		var ret NotebookResourceType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *NotebookCreateData) GetTypeOk() (*NotebookResourceType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *NotebookCreateData) SetType(v NotebookResourceType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o NotebookCreateData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["attributes"] = o.Attributes
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *NotebookCreateData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Attributes *NotebookCreateDataAttributes `json:"attributes"`
		Type       *NotebookResourceType         `json:"type"`
	}{}
	all := struct {
		Attributes NotebookCreateDataAttributes `json:"attributes"`
		Type       NotebookResourceType         `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Attributes == nil {
		return fmt.Errorf("Required field attributes missing")
	}
	if required.Type == nil {
		return fmt.Errorf("Required field type missing")
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
	if v := all.Type; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if all.Attributes.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Attributes = all.Attributes
	o.Type = all.Type
	return nil
}
