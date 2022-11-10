// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// GeomapWidgetDefinitionView The view of the world that the map should render.
type GeomapWidgetDefinitionView struct {
	// The 2-letter ISO code of a country to focus the map on. Or `WORLD`.
	Focus string `json:"focus"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewGeomapWidgetDefinitionView instantiates a new GeomapWidgetDefinitionView object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewGeomapWidgetDefinitionView(focus string) *GeomapWidgetDefinitionView {
	this := GeomapWidgetDefinitionView{}
	this.Focus = focus
	return &this
}

// NewGeomapWidgetDefinitionViewWithDefaults instantiates a new GeomapWidgetDefinitionView object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewGeomapWidgetDefinitionViewWithDefaults() *GeomapWidgetDefinitionView {
	this := GeomapWidgetDefinitionView{}
	return &this
}

// GetFocus returns the Focus field value.
func (o *GeomapWidgetDefinitionView) GetFocus() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Focus
}

// GetFocusOk returns a tuple with the Focus field value
// and a boolean to check if the value has been set.
func (o *GeomapWidgetDefinitionView) GetFocusOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Focus, true
}

// SetFocus sets field value.
func (o *GeomapWidgetDefinitionView) SetFocus(v string) {
	o.Focus = v
}

// MarshalJSON serializes the struct using spec logic.
func (o GeomapWidgetDefinitionView) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["focus"] = o.Focus

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *GeomapWidgetDefinitionView) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Focus *string `json:"focus"`
	}{}
	all := struct {
		Focus string `json:"focus"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Focus == nil {
		return fmt.Errorf("Required field focus missing")
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
	o.Focus = all.Focus
	return nil
}
