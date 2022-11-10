// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// FreeTextWidgetDefinition Free text is a widget that allows you to add headings to your screenboard. Commonly used to state the overall purpose of the dashboard. Only available on FREE layout dashboards.
type FreeTextWidgetDefinition struct {
	// Color of the text.
	Color *string `json:"color,omitempty"`
	// Size of the text.
	FontSize *string `json:"font_size,omitempty"`
	// Text to display.
	Text string `json:"text"`
	// How to align the text on the widget.
	TextAlign *WidgetTextAlign `json:"text_align,omitempty"`
	// Type of the free text widget.
	Type FreeTextWidgetDefinitionType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewFreeTextWidgetDefinition instantiates a new FreeTextWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewFreeTextWidgetDefinition(text string, typeVar FreeTextWidgetDefinitionType) *FreeTextWidgetDefinition {
	this := FreeTextWidgetDefinition{}
	this.Text = text
	this.Type = typeVar
	return &this
}

// NewFreeTextWidgetDefinitionWithDefaults instantiates a new FreeTextWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewFreeTextWidgetDefinitionWithDefaults() *FreeTextWidgetDefinition {
	this := FreeTextWidgetDefinition{}
	var typeVar FreeTextWidgetDefinitionType = FREETEXTWIDGETDEFINITIONTYPE_FREE_TEXT
	this.Type = typeVar
	return &this
}

// GetColor returns the Color field value if set, zero value otherwise.
func (o *FreeTextWidgetDefinition) GetColor() string {
	if o == nil || o.Color == nil {
		var ret string
		return ret
	}
	return *o.Color
}

// GetColorOk returns a tuple with the Color field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FreeTextWidgetDefinition) GetColorOk() (*string, bool) {
	if o == nil || o.Color == nil {
		return nil, false
	}
	return o.Color, true
}

// HasColor returns a boolean if a field has been set.
func (o *FreeTextWidgetDefinition) HasColor() bool {
	if o != nil && o.Color != nil {
		return true
	}

	return false
}

// SetColor gets a reference to the given string and assigns it to the Color field.
func (o *FreeTextWidgetDefinition) SetColor(v string) {
	o.Color = &v
}

// GetFontSize returns the FontSize field value if set, zero value otherwise.
func (o *FreeTextWidgetDefinition) GetFontSize() string {
	if o == nil || o.FontSize == nil {
		var ret string
		return ret
	}
	return *o.FontSize
}

// GetFontSizeOk returns a tuple with the FontSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FreeTextWidgetDefinition) GetFontSizeOk() (*string, bool) {
	if o == nil || o.FontSize == nil {
		return nil, false
	}
	return o.FontSize, true
}

// HasFontSize returns a boolean if a field has been set.
func (o *FreeTextWidgetDefinition) HasFontSize() bool {
	if o != nil && o.FontSize != nil {
		return true
	}

	return false
}

// SetFontSize gets a reference to the given string and assigns it to the FontSize field.
func (o *FreeTextWidgetDefinition) SetFontSize(v string) {
	o.FontSize = &v
}

// GetText returns the Text field value.
func (o *FreeTextWidgetDefinition) GetText() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Text
}

// GetTextOk returns a tuple with the Text field value
// and a boolean to check if the value has been set.
func (o *FreeTextWidgetDefinition) GetTextOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Text, true
}

// SetText sets field value.
func (o *FreeTextWidgetDefinition) SetText(v string) {
	o.Text = v
}

// GetTextAlign returns the TextAlign field value if set, zero value otherwise.
func (o *FreeTextWidgetDefinition) GetTextAlign() WidgetTextAlign {
	if o == nil || o.TextAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TextAlign
}

// GetTextAlignOk returns a tuple with the TextAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FreeTextWidgetDefinition) GetTextAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TextAlign == nil {
		return nil, false
	}
	return o.TextAlign, true
}

// HasTextAlign returns a boolean if a field has been set.
func (o *FreeTextWidgetDefinition) HasTextAlign() bool {
	if o != nil && o.TextAlign != nil {
		return true
	}

	return false
}

// SetTextAlign gets a reference to the given WidgetTextAlign and assigns it to the TextAlign field.
func (o *FreeTextWidgetDefinition) SetTextAlign(v WidgetTextAlign) {
	o.TextAlign = &v
}

// GetType returns the Type field value.
func (o *FreeTextWidgetDefinition) GetType() FreeTextWidgetDefinitionType {
	if o == nil {
		var ret FreeTextWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *FreeTextWidgetDefinition) GetTypeOk() (*FreeTextWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *FreeTextWidgetDefinition) SetType(v FreeTextWidgetDefinitionType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o FreeTextWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Color != nil {
		toSerialize["color"] = o.Color
	}
	if o.FontSize != nil {
		toSerialize["font_size"] = o.FontSize
	}
	toSerialize["text"] = o.Text
	if o.TextAlign != nil {
		toSerialize["text_align"] = o.TextAlign
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *FreeTextWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Text *string                       `json:"text"`
		Type *FreeTextWidgetDefinitionType `json:"type"`
	}{}
	all := struct {
		Color     *string                      `json:"color,omitempty"`
		FontSize  *string                      `json:"font_size,omitempty"`
		Text      string                       `json:"text"`
		TextAlign *WidgetTextAlign             `json:"text_align,omitempty"`
		Type      FreeTextWidgetDefinitionType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Text == nil {
		return fmt.Errorf("Required field text missing")
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
	if v := all.TextAlign; v != nil && !v.IsValid() {
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
	o.Color = all.Color
	o.FontSize = all.FontSize
	o.Text = all.Text
	o.TextAlign = all.TextAlign
	o.Type = all.Type
	return nil
}
