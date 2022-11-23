// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// WidgetRequestStyle Define request widget style.
type WidgetRequestStyle struct {
	// Type of lines displayed.
	LineType *WidgetLineType `json:"line_type,omitempty"`
	// Width of line displayed.
	LineWidth *WidgetLineWidth `json:"line_width,omitempty"`
	// Color palette to apply to the widget.
	Palette *string `json:"palette,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewWidgetRequestStyle instantiates a new WidgetRequestStyle object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewWidgetRequestStyle() *WidgetRequestStyle {
	this := WidgetRequestStyle{}
	return &this
}

// NewWidgetRequestStyleWithDefaults instantiates a new WidgetRequestStyle object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewWidgetRequestStyleWithDefaults() *WidgetRequestStyle {
	this := WidgetRequestStyle{}
	return &this
}

// GetLineType returns the LineType field value if set, zero value otherwise.
func (o *WidgetRequestStyle) GetLineType() WidgetLineType {
	if o == nil || o.LineType == nil {
		var ret WidgetLineType
		return ret
	}
	return *o.LineType
}

// GetLineTypeOk returns a tuple with the LineType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetRequestStyle) GetLineTypeOk() (*WidgetLineType, bool) {
	if o == nil || o.LineType == nil {
		return nil, false
	}
	return o.LineType, true
}

// HasLineType returns a boolean if a field has been set.
func (o *WidgetRequestStyle) HasLineType() bool {
	if o != nil && o.LineType != nil {
		return true
	}

	return false
}

// SetLineType gets a reference to the given WidgetLineType and assigns it to the LineType field.
func (o *WidgetRequestStyle) SetLineType(v WidgetLineType) {
	o.LineType = &v
}

// GetLineWidth returns the LineWidth field value if set, zero value otherwise.
func (o *WidgetRequestStyle) GetLineWidth() WidgetLineWidth {
	if o == nil || o.LineWidth == nil {
		var ret WidgetLineWidth
		return ret
	}
	return *o.LineWidth
}

// GetLineWidthOk returns a tuple with the LineWidth field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetRequestStyle) GetLineWidthOk() (*WidgetLineWidth, bool) {
	if o == nil || o.LineWidth == nil {
		return nil, false
	}
	return o.LineWidth, true
}

// HasLineWidth returns a boolean if a field has been set.
func (o *WidgetRequestStyle) HasLineWidth() bool {
	if o != nil && o.LineWidth != nil {
		return true
	}

	return false
}

// SetLineWidth gets a reference to the given WidgetLineWidth and assigns it to the LineWidth field.
func (o *WidgetRequestStyle) SetLineWidth(v WidgetLineWidth) {
	o.LineWidth = &v
}

// GetPalette returns the Palette field value if set, zero value otherwise.
func (o *WidgetRequestStyle) GetPalette() string {
	if o == nil || o.Palette == nil {
		var ret string
		return ret
	}
	return *o.Palette
}

// GetPaletteOk returns a tuple with the Palette field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetRequestStyle) GetPaletteOk() (*string, bool) {
	if o == nil || o.Palette == nil {
		return nil, false
	}
	return o.Palette, true
}

// HasPalette returns a boolean if a field has been set.
func (o *WidgetRequestStyle) HasPalette() bool {
	if o != nil && o.Palette != nil {
		return true
	}

	return false
}

// SetPalette gets a reference to the given string and assigns it to the Palette field.
func (o *WidgetRequestStyle) SetPalette(v string) {
	o.Palette = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o WidgetRequestStyle) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.LineType != nil {
		toSerialize["line_type"] = o.LineType
	}
	if o.LineWidth != nil {
		toSerialize["line_width"] = o.LineWidth
	}
	if o.Palette != nil {
		toSerialize["palette"] = o.Palette
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *WidgetRequestStyle) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		LineType  *WidgetLineType  `json:"line_type,omitempty"`
		LineWidth *WidgetLineWidth `json:"line_width,omitempty"`
		Palette   *string          `json:"palette,omitempty"`
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
	if v := all.LineType; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.LineWidth; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.LineType = all.LineType
	o.LineWidth = all.LineWidth
	o.Palette = all.Palette
	return nil
}
