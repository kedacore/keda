// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// GeomapWidgetDefinitionStyle The style to apply to the widget.
type GeomapWidgetDefinitionStyle struct {
	// The color palette to apply to the widget.
	Palette string `json:"palette"`
	// Whether to flip the palette tones.
	PaletteFlip bool `json:"palette_flip"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewGeomapWidgetDefinitionStyle instantiates a new GeomapWidgetDefinitionStyle object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewGeomapWidgetDefinitionStyle(palette string, paletteFlip bool) *GeomapWidgetDefinitionStyle {
	this := GeomapWidgetDefinitionStyle{}
	this.Palette = palette
	this.PaletteFlip = paletteFlip
	return &this
}

// NewGeomapWidgetDefinitionStyleWithDefaults instantiates a new GeomapWidgetDefinitionStyle object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewGeomapWidgetDefinitionStyleWithDefaults() *GeomapWidgetDefinitionStyle {
	this := GeomapWidgetDefinitionStyle{}
	return &this
}

// GetPalette returns the Palette field value.
func (o *GeomapWidgetDefinitionStyle) GetPalette() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Palette
}

// GetPaletteOk returns a tuple with the Palette field value
// and a boolean to check if the value has been set.
func (o *GeomapWidgetDefinitionStyle) GetPaletteOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Palette, true
}

// SetPalette sets field value.
func (o *GeomapWidgetDefinitionStyle) SetPalette(v string) {
	o.Palette = v
}

// GetPaletteFlip returns the PaletteFlip field value.
func (o *GeomapWidgetDefinitionStyle) GetPaletteFlip() bool {
	if o == nil {
		var ret bool
		return ret
	}
	return o.PaletteFlip
}

// GetPaletteFlipOk returns a tuple with the PaletteFlip field value
// and a boolean to check if the value has been set.
func (o *GeomapWidgetDefinitionStyle) GetPaletteFlipOk() (*bool, bool) {
	if o == nil {
		return nil, false
	}
	return &o.PaletteFlip, true
}

// SetPaletteFlip sets field value.
func (o *GeomapWidgetDefinitionStyle) SetPaletteFlip(v bool) {
	o.PaletteFlip = v
}

// MarshalJSON serializes the struct using spec logic.
func (o GeomapWidgetDefinitionStyle) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["palette"] = o.Palette
	toSerialize["palette_flip"] = o.PaletteFlip

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *GeomapWidgetDefinitionStyle) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Palette     *string `json:"palette"`
		PaletteFlip *bool   `json:"palette_flip"`
	}{}
	all := struct {
		Palette     string `json:"palette"`
		PaletteFlip bool   `json:"palette_flip"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Palette == nil {
		return fmt.Errorf("Required field palette missing")
	}
	if required.PaletteFlip == nil {
		return fmt.Errorf("Required field palette_flip missing")
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
	o.Palette = all.Palette
	o.PaletteFlip = all.PaletteFlip
	return nil
}
