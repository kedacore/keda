// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// NotebookCellResponse The description of a notebook cell response.
type NotebookCellResponse struct {
	// The attributes of a notebook cell response. Valid cell types are `markdown`, `timeseries`, `toplist`, `heatmap`, `distribution`,
	// `log_stream`. [More information on each graph visualization type.](https://docs.datadoghq.com/dashboards/widgets/)
	Attributes NotebookCellResponseAttributes `json:"attributes"`
	// Notebook cell ID.
	Id string `json:"id"`
	// Type of the Notebook Cell resource.
	Type NotebookCellResourceType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewNotebookCellResponse instantiates a new NotebookCellResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewNotebookCellResponse(attributes NotebookCellResponseAttributes, id string, typeVar NotebookCellResourceType) *NotebookCellResponse {
	this := NotebookCellResponse{}
	this.Attributes = attributes
	this.Id = id
	this.Type = typeVar
	return &this
}

// NewNotebookCellResponseWithDefaults instantiates a new NotebookCellResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewNotebookCellResponseWithDefaults() *NotebookCellResponse {
	this := NotebookCellResponse{}
	var typeVar NotebookCellResourceType = NOTEBOOKCELLRESOURCETYPE_NOTEBOOK_CELLS
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value.
func (o *NotebookCellResponse) GetAttributes() NotebookCellResponseAttributes {
	if o == nil {
		var ret NotebookCellResponseAttributes
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value
// and a boolean to check if the value has been set.
func (o *NotebookCellResponse) GetAttributesOk() (*NotebookCellResponseAttributes, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// SetAttributes sets field value.
func (o *NotebookCellResponse) SetAttributes(v NotebookCellResponseAttributes) {
	o.Attributes = v
}

// GetId returns the Id field value.
func (o *NotebookCellResponse) GetId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *NotebookCellResponse) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value.
func (o *NotebookCellResponse) SetId(v string) {
	o.Id = v
}

// GetType returns the Type field value.
func (o *NotebookCellResponse) GetType() NotebookCellResourceType {
	if o == nil {
		var ret NotebookCellResourceType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *NotebookCellResponse) GetTypeOk() (*NotebookCellResourceType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *NotebookCellResponse) SetType(v NotebookCellResourceType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o NotebookCellResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["attributes"] = o.Attributes
	toSerialize["id"] = o.Id
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *NotebookCellResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Attributes *NotebookCellResponseAttributes `json:"attributes"`
		Id         *string                         `json:"id"`
		Type       *NotebookCellResourceType       `json:"type"`
	}{}
	all := struct {
		Attributes NotebookCellResponseAttributes `json:"attributes"`
		Id         string                         `json:"id"`
		Type       NotebookCellResourceType       `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Attributes == nil {
		return fmt.Errorf("Required field attributes missing")
	}
	if required.Id == nil {
		return fmt.Errorf("Required field id missing")
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
	o.Attributes = all.Attributes
	o.Id = all.Id
	o.Type = all.Type
	return nil
}
