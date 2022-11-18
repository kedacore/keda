// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// HostMapWidgetDefinitionStyle The style to apply to the widget.
type HostMapWidgetDefinitionStyle struct {
	// Max value to use to color the map.
	FillMax *string `json:"fill_max,omitempty"`
	// Min value to use to color the map.
	FillMin *string `json:"fill_min,omitempty"`
	// Color palette to apply to the widget.
	Palette *string `json:"palette,omitempty"`
	// Whether to flip the palette tones.
	PaletteFlip *bool `json:"palette_flip,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewHostMapWidgetDefinitionStyle instantiates a new HostMapWidgetDefinitionStyle object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewHostMapWidgetDefinitionStyle() *HostMapWidgetDefinitionStyle {
	this := HostMapWidgetDefinitionStyle{}
	return &this
}

// NewHostMapWidgetDefinitionStyleWithDefaults instantiates a new HostMapWidgetDefinitionStyle object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewHostMapWidgetDefinitionStyleWithDefaults() *HostMapWidgetDefinitionStyle {
	this := HostMapWidgetDefinitionStyle{}
	return &this
}

// GetFillMax returns the FillMax field value if set, zero value otherwise.
func (o *HostMapWidgetDefinitionStyle) GetFillMax() string {
	if o == nil || o.FillMax == nil {
		var ret string
		return ret
	}
	return *o.FillMax
}

// GetFillMaxOk returns a tuple with the FillMax field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMapWidgetDefinitionStyle) GetFillMaxOk() (*string, bool) {
	if o == nil || o.FillMax == nil {
		return nil, false
	}
	return o.FillMax, true
}

// HasFillMax returns a boolean if a field has been set.
func (o *HostMapWidgetDefinitionStyle) HasFillMax() bool {
	if o != nil && o.FillMax != nil {
		return true
	}

	return false
}

// SetFillMax gets a reference to the given string and assigns it to the FillMax field.
func (o *HostMapWidgetDefinitionStyle) SetFillMax(v string) {
	o.FillMax = &v
}

// GetFillMin returns the FillMin field value if set, zero value otherwise.
func (o *HostMapWidgetDefinitionStyle) GetFillMin() string {
	if o == nil || o.FillMin == nil {
		var ret string
		return ret
	}
	return *o.FillMin
}

// GetFillMinOk returns a tuple with the FillMin field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMapWidgetDefinitionStyle) GetFillMinOk() (*string, bool) {
	if o == nil || o.FillMin == nil {
		return nil, false
	}
	return o.FillMin, true
}

// HasFillMin returns a boolean if a field has been set.
func (o *HostMapWidgetDefinitionStyle) HasFillMin() bool {
	if o != nil && o.FillMin != nil {
		return true
	}

	return false
}

// SetFillMin gets a reference to the given string and assigns it to the FillMin field.
func (o *HostMapWidgetDefinitionStyle) SetFillMin(v string) {
	o.FillMin = &v
}

// GetPalette returns the Palette field value if set, zero value otherwise.
func (o *HostMapWidgetDefinitionStyle) GetPalette() string {
	if o == nil || o.Palette == nil {
		var ret string
		return ret
	}
	return *o.Palette
}

// GetPaletteOk returns a tuple with the Palette field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMapWidgetDefinitionStyle) GetPaletteOk() (*string, bool) {
	if o == nil || o.Palette == nil {
		return nil, false
	}
	return o.Palette, true
}

// HasPalette returns a boolean if a field has been set.
func (o *HostMapWidgetDefinitionStyle) HasPalette() bool {
	if o != nil && o.Palette != nil {
		return true
	}

	return false
}

// SetPalette gets a reference to the given string and assigns it to the Palette field.
func (o *HostMapWidgetDefinitionStyle) SetPalette(v string) {
	o.Palette = &v
}

// GetPaletteFlip returns the PaletteFlip field value if set, zero value otherwise.
func (o *HostMapWidgetDefinitionStyle) GetPaletteFlip() bool {
	if o == nil || o.PaletteFlip == nil {
		var ret bool
		return ret
	}
	return *o.PaletteFlip
}

// GetPaletteFlipOk returns a tuple with the PaletteFlip field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMapWidgetDefinitionStyle) GetPaletteFlipOk() (*bool, bool) {
	if o == nil || o.PaletteFlip == nil {
		return nil, false
	}
	return o.PaletteFlip, true
}

// HasPaletteFlip returns a boolean if a field has been set.
func (o *HostMapWidgetDefinitionStyle) HasPaletteFlip() bool {
	if o != nil && o.PaletteFlip != nil {
		return true
	}

	return false
}

// SetPaletteFlip gets a reference to the given bool and assigns it to the PaletteFlip field.
func (o *HostMapWidgetDefinitionStyle) SetPaletteFlip(v bool) {
	o.PaletteFlip = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o HostMapWidgetDefinitionStyle) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.FillMax != nil {
		toSerialize["fill_max"] = o.FillMax
	}
	if o.FillMin != nil {
		toSerialize["fill_min"] = o.FillMin
	}
	if o.Palette != nil {
		toSerialize["palette"] = o.Palette
	}
	if o.PaletteFlip != nil {
		toSerialize["palette_flip"] = o.PaletteFlip
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *HostMapWidgetDefinitionStyle) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		FillMax     *string `json:"fill_max,omitempty"`
		FillMin     *string `json:"fill_min,omitempty"`
		Palette     *string `json:"palette,omitempty"`
		PaletteFlip *bool   `json:"palette_flip,omitempty"`
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
	o.FillMax = all.FillMax
	o.FillMin = all.FillMin
	o.Palette = all.Palette
	o.PaletteFlip = all.PaletteFlip
	return nil
}
