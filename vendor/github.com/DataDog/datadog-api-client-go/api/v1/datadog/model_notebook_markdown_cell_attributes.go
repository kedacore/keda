// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// NotebookMarkdownCellAttributes The attributes of a notebook `markdown` cell.
type NotebookMarkdownCellAttributes struct {
	// Text in a notebook is formatted with [Markdown](https://daringfireball.net/projects/markdown/), which enables the use of headings, subheadings, links, images, lists, and code blocks.
	Definition NotebookMarkdownCellDefinition `json:"definition"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewNotebookMarkdownCellAttributes instantiates a new NotebookMarkdownCellAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewNotebookMarkdownCellAttributes(definition NotebookMarkdownCellDefinition) *NotebookMarkdownCellAttributes {
	this := NotebookMarkdownCellAttributes{}
	this.Definition = definition
	return &this
}

// NewNotebookMarkdownCellAttributesWithDefaults instantiates a new NotebookMarkdownCellAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewNotebookMarkdownCellAttributesWithDefaults() *NotebookMarkdownCellAttributes {
	this := NotebookMarkdownCellAttributes{}
	return &this
}

// GetDefinition returns the Definition field value.
func (o *NotebookMarkdownCellAttributes) GetDefinition() NotebookMarkdownCellDefinition {
	if o == nil {
		var ret NotebookMarkdownCellDefinition
		return ret
	}
	return o.Definition
}

// GetDefinitionOk returns a tuple with the Definition field value
// and a boolean to check if the value has been set.
func (o *NotebookMarkdownCellAttributes) GetDefinitionOk() (*NotebookMarkdownCellDefinition, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Definition, true
}

// SetDefinition sets field value.
func (o *NotebookMarkdownCellAttributes) SetDefinition(v NotebookMarkdownCellDefinition) {
	o.Definition = v
}

// MarshalJSON serializes the struct using spec logic.
func (o NotebookMarkdownCellAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["definition"] = o.Definition

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *NotebookMarkdownCellAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Definition *NotebookMarkdownCellDefinition `json:"definition"`
	}{}
	all := struct {
		Definition NotebookMarkdownCellDefinition `json:"definition"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Definition == nil {
		return fmt.Errorf("Required field definition missing")
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
	if all.Definition.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Definition = all.Definition
	return nil
}
