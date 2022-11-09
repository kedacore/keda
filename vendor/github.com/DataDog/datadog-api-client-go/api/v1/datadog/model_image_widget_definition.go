// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ImageWidgetDefinition The image widget allows you to embed an image on your dashboard. An image can be a PNG, JPG, or animated GIF. Only available on FREE layout dashboards.
type ImageWidgetDefinition struct {
	// Whether to display a background or not.
	HasBackground *bool `json:"has_background,omitempty"`
	// Whether to display a border or not.
	HasBorder *bool `json:"has_border,omitempty"`
	// Horizontal alignment.
	HorizontalAlign *WidgetHorizontalAlign `json:"horizontal_align,omitempty"`
	// Size of the margins around the image.
	// **Note**: `small` and `large` values are deprecated.
	Margin *WidgetMargin `json:"margin,omitempty"`
	// How to size the image on the widget. The values are based on the image `object-fit` CSS properties.
	// **Note**: `zoom`, `fit` and `center` values are deprecated.
	Sizing *WidgetImageSizing `json:"sizing,omitempty"`
	// Type of the image widget.
	Type ImageWidgetDefinitionType `json:"type"`
	// URL of the image.
	Url string `json:"url"`
	// URL of the image in dark mode.
	UrlDarkTheme *string `json:"url_dark_theme,omitempty"`
	// Vertical alignment.
	VerticalAlign *WidgetVerticalAlign `json:"vertical_align,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewImageWidgetDefinition instantiates a new ImageWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewImageWidgetDefinition(typeVar ImageWidgetDefinitionType, url string) *ImageWidgetDefinition {
	this := ImageWidgetDefinition{}
	var hasBackground bool = true
	this.HasBackground = &hasBackground
	var hasBorder bool = true
	this.HasBorder = &hasBorder
	this.Type = typeVar
	this.Url = url
	return &this
}

// NewImageWidgetDefinitionWithDefaults instantiates a new ImageWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewImageWidgetDefinitionWithDefaults() *ImageWidgetDefinition {
	this := ImageWidgetDefinition{}
	var hasBackground bool = true
	this.HasBackground = &hasBackground
	var hasBorder bool = true
	this.HasBorder = &hasBorder
	var typeVar ImageWidgetDefinitionType = IMAGEWIDGETDEFINITIONTYPE_IMAGE
	this.Type = typeVar
	return &this
}

// GetHasBackground returns the HasBackground field value if set, zero value otherwise.
func (o *ImageWidgetDefinition) GetHasBackground() bool {
	if o == nil || o.HasBackground == nil {
		var ret bool
		return ret
	}
	return *o.HasBackground
}

// GetHasBackgroundOk returns a tuple with the HasBackground field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ImageWidgetDefinition) GetHasBackgroundOk() (*bool, bool) {
	if o == nil || o.HasBackground == nil {
		return nil, false
	}
	return o.HasBackground, true
}

// HasHasBackground returns a boolean if a field has been set.
func (o *ImageWidgetDefinition) HasHasBackground() bool {
	if o != nil && o.HasBackground != nil {
		return true
	}

	return false
}

// SetHasBackground gets a reference to the given bool and assigns it to the HasBackground field.
func (o *ImageWidgetDefinition) SetHasBackground(v bool) {
	o.HasBackground = &v
}

// GetHasBorder returns the HasBorder field value if set, zero value otherwise.
func (o *ImageWidgetDefinition) GetHasBorder() bool {
	if o == nil || o.HasBorder == nil {
		var ret bool
		return ret
	}
	return *o.HasBorder
}

// GetHasBorderOk returns a tuple with the HasBorder field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ImageWidgetDefinition) GetHasBorderOk() (*bool, bool) {
	if o == nil || o.HasBorder == nil {
		return nil, false
	}
	return o.HasBorder, true
}

// HasHasBorder returns a boolean if a field has been set.
func (o *ImageWidgetDefinition) HasHasBorder() bool {
	if o != nil && o.HasBorder != nil {
		return true
	}

	return false
}

// SetHasBorder gets a reference to the given bool and assigns it to the HasBorder field.
func (o *ImageWidgetDefinition) SetHasBorder(v bool) {
	o.HasBorder = &v
}

// GetHorizontalAlign returns the HorizontalAlign field value if set, zero value otherwise.
func (o *ImageWidgetDefinition) GetHorizontalAlign() WidgetHorizontalAlign {
	if o == nil || o.HorizontalAlign == nil {
		var ret WidgetHorizontalAlign
		return ret
	}
	return *o.HorizontalAlign
}

// GetHorizontalAlignOk returns a tuple with the HorizontalAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ImageWidgetDefinition) GetHorizontalAlignOk() (*WidgetHorizontalAlign, bool) {
	if o == nil || o.HorizontalAlign == nil {
		return nil, false
	}
	return o.HorizontalAlign, true
}

// HasHorizontalAlign returns a boolean if a field has been set.
func (o *ImageWidgetDefinition) HasHorizontalAlign() bool {
	if o != nil && o.HorizontalAlign != nil {
		return true
	}

	return false
}

// SetHorizontalAlign gets a reference to the given WidgetHorizontalAlign and assigns it to the HorizontalAlign field.
func (o *ImageWidgetDefinition) SetHorizontalAlign(v WidgetHorizontalAlign) {
	o.HorizontalAlign = &v
}

// GetMargin returns the Margin field value if set, zero value otherwise.
func (o *ImageWidgetDefinition) GetMargin() WidgetMargin {
	if o == nil || o.Margin == nil {
		var ret WidgetMargin
		return ret
	}
	return *o.Margin
}

// GetMarginOk returns a tuple with the Margin field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ImageWidgetDefinition) GetMarginOk() (*WidgetMargin, bool) {
	if o == nil || o.Margin == nil {
		return nil, false
	}
	return o.Margin, true
}

// HasMargin returns a boolean if a field has been set.
func (o *ImageWidgetDefinition) HasMargin() bool {
	if o != nil && o.Margin != nil {
		return true
	}

	return false
}

// SetMargin gets a reference to the given WidgetMargin and assigns it to the Margin field.
func (o *ImageWidgetDefinition) SetMargin(v WidgetMargin) {
	o.Margin = &v
}

// GetSizing returns the Sizing field value if set, zero value otherwise.
func (o *ImageWidgetDefinition) GetSizing() WidgetImageSizing {
	if o == nil || o.Sizing == nil {
		var ret WidgetImageSizing
		return ret
	}
	return *o.Sizing
}

// GetSizingOk returns a tuple with the Sizing field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ImageWidgetDefinition) GetSizingOk() (*WidgetImageSizing, bool) {
	if o == nil || o.Sizing == nil {
		return nil, false
	}
	return o.Sizing, true
}

// HasSizing returns a boolean if a field has been set.
func (o *ImageWidgetDefinition) HasSizing() bool {
	if o != nil && o.Sizing != nil {
		return true
	}

	return false
}

// SetSizing gets a reference to the given WidgetImageSizing and assigns it to the Sizing field.
func (o *ImageWidgetDefinition) SetSizing(v WidgetImageSizing) {
	o.Sizing = &v
}

// GetType returns the Type field value.
func (o *ImageWidgetDefinition) GetType() ImageWidgetDefinitionType {
	if o == nil {
		var ret ImageWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *ImageWidgetDefinition) GetTypeOk() (*ImageWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *ImageWidgetDefinition) SetType(v ImageWidgetDefinitionType) {
	o.Type = v
}

// GetUrl returns the Url field value.
func (o *ImageWidgetDefinition) GetUrl() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Url
}

// GetUrlOk returns a tuple with the Url field value
// and a boolean to check if the value has been set.
func (o *ImageWidgetDefinition) GetUrlOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Url, true
}

// SetUrl sets field value.
func (o *ImageWidgetDefinition) SetUrl(v string) {
	o.Url = v
}

// GetUrlDarkTheme returns the UrlDarkTheme field value if set, zero value otherwise.
func (o *ImageWidgetDefinition) GetUrlDarkTheme() string {
	if o == nil || o.UrlDarkTheme == nil {
		var ret string
		return ret
	}
	return *o.UrlDarkTheme
}

// GetUrlDarkThemeOk returns a tuple with the UrlDarkTheme field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ImageWidgetDefinition) GetUrlDarkThemeOk() (*string, bool) {
	if o == nil || o.UrlDarkTheme == nil {
		return nil, false
	}
	return o.UrlDarkTheme, true
}

// HasUrlDarkTheme returns a boolean if a field has been set.
func (o *ImageWidgetDefinition) HasUrlDarkTheme() bool {
	if o != nil && o.UrlDarkTheme != nil {
		return true
	}

	return false
}

// SetUrlDarkTheme gets a reference to the given string and assigns it to the UrlDarkTheme field.
func (o *ImageWidgetDefinition) SetUrlDarkTheme(v string) {
	o.UrlDarkTheme = &v
}

// GetVerticalAlign returns the VerticalAlign field value if set, zero value otherwise.
func (o *ImageWidgetDefinition) GetVerticalAlign() WidgetVerticalAlign {
	if o == nil || o.VerticalAlign == nil {
		var ret WidgetVerticalAlign
		return ret
	}
	return *o.VerticalAlign
}

// GetVerticalAlignOk returns a tuple with the VerticalAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ImageWidgetDefinition) GetVerticalAlignOk() (*WidgetVerticalAlign, bool) {
	if o == nil || o.VerticalAlign == nil {
		return nil, false
	}
	return o.VerticalAlign, true
}

// HasVerticalAlign returns a boolean if a field has been set.
func (o *ImageWidgetDefinition) HasVerticalAlign() bool {
	if o != nil && o.VerticalAlign != nil {
		return true
	}

	return false
}

// SetVerticalAlign gets a reference to the given WidgetVerticalAlign and assigns it to the VerticalAlign field.
func (o *ImageWidgetDefinition) SetVerticalAlign(v WidgetVerticalAlign) {
	o.VerticalAlign = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ImageWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.HasBackground != nil {
		toSerialize["has_background"] = o.HasBackground
	}
	if o.HasBorder != nil {
		toSerialize["has_border"] = o.HasBorder
	}
	if o.HorizontalAlign != nil {
		toSerialize["horizontal_align"] = o.HorizontalAlign
	}
	if o.Margin != nil {
		toSerialize["margin"] = o.Margin
	}
	if o.Sizing != nil {
		toSerialize["sizing"] = o.Sizing
	}
	toSerialize["type"] = o.Type
	toSerialize["url"] = o.Url
	if o.UrlDarkTheme != nil {
		toSerialize["url_dark_theme"] = o.UrlDarkTheme
	}
	if o.VerticalAlign != nil {
		toSerialize["vertical_align"] = o.VerticalAlign
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ImageWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Type *ImageWidgetDefinitionType `json:"type"`
		Url  *string                    `json:"url"`
	}{}
	all := struct {
		HasBackground   *bool                     `json:"has_background,omitempty"`
		HasBorder       *bool                     `json:"has_border,omitempty"`
		HorizontalAlign *WidgetHorizontalAlign    `json:"horizontal_align,omitempty"`
		Margin          *WidgetMargin             `json:"margin,omitempty"`
		Sizing          *WidgetImageSizing        `json:"sizing,omitempty"`
		Type            ImageWidgetDefinitionType `json:"type"`
		Url             string                    `json:"url"`
		UrlDarkTheme    *string                   `json:"url_dark_theme,omitempty"`
		VerticalAlign   *WidgetVerticalAlign      `json:"vertical_align,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Type == nil {
		return fmt.Errorf("Required field type missing")
	}
	if required.Url == nil {
		return fmt.Errorf("Required field url missing")
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
	if v := all.HorizontalAlign; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Margin; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Sizing; v != nil && !v.IsValid() {
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
	if v := all.VerticalAlign; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.HasBackground = all.HasBackground
	o.HasBorder = all.HasBorder
	o.HorizontalAlign = all.HorizontalAlign
	o.Margin = all.Margin
	o.Sizing = all.Sizing
	o.Type = all.Type
	o.Url = all.Url
	o.UrlDarkTheme = all.UrlDarkTheme
	o.VerticalAlign = all.VerticalAlign
	return nil
}
