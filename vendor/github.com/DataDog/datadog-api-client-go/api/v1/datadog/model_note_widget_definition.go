// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// NoteWidgetDefinition The notes and links widget is similar to free text widget, but allows for more formatting options.
type NoteWidgetDefinition struct {
	// Background color of the note.
	BackgroundColor *string `json:"background_color,omitempty"`
	// Content of the note.
	Content string `json:"content"`
	// Size of the text.
	FontSize *string `json:"font_size,omitempty"`
	// Whether to add padding or not.
	HasPadding *bool `json:"has_padding,omitempty"`
	// Whether to show a tick or not.
	ShowTick *bool `json:"show_tick,omitempty"`
	// How to align the text on the widget.
	TextAlign *WidgetTextAlign `json:"text_align,omitempty"`
	// Define how you want to align the text on the widget.
	TickEdge *WidgetTickEdge `json:"tick_edge,omitempty"`
	// Where to position the tick on an edge.
	TickPos *string `json:"tick_pos,omitempty"`
	// Type of the note widget.
	Type NoteWidgetDefinitionType `json:"type"`
	// Vertical alignment.
	VerticalAlign *WidgetVerticalAlign `json:"vertical_align,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewNoteWidgetDefinition instantiates a new NoteWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewNoteWidgetDefinition(content string, typeVar NoteWidgetDefinitionType) *NoteWidgetDefinition {
	this := NoteWidgetDefinition{}
	this.Content = content
	var hasPadding bool = true
	this.HasPadding = &hasPadding
	this.Type = typeVar
	return &this
}

// NewNoteWidgetDefinitionWithDefaults instantiates a new NoteWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewNoteWidgetDefinitionWithDefaults() *NoteWidgetDefinition {
	this := NoteWidgetDefinition{}
	var hasPadding bool = true
	this.HasPadding = &hasPadding
	var typeVar NoteWidgetDefinitionType = NOTEWIDGETDEFINITIONTYPE_NOTE
	this.Type = typeVar
	return &this
}

// GetBackgroundColor returns the BackgroundColor field value if set, zero value otherwise.
func (o *NoteWidgetDefinition) GetBackgroundColor() string {
	if o == nil || o.BackgroundColor == nil {
		var ret string
		return ret
	}
	return *o.BackgroundColor
}

// GetBackgroundColorOk returns a tuple with the BackgroundColor field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NoteWidgetDefinition) GetBackgroundColorOk() (*string, bool) {
	if o == nil || o.BackgroundColor == nil {
		return nil, false
	}
	return o.BackgroundColor, true
}

// HasBackgroundColor returns a boolean if a field has been set.
func (o *NoteWidgetDefinition) HasBackgroundColor() bool {
	if o != nil && o.BackgroundColor != nil {
		return true
	}

	return false
}

// SetBackgroundColor gets a reference to the given string and assigns it to the BackgroundColor field.
func (o *NoteWidgetDefinition) SetBackgroundColor(v string) {
	o.BackgroundColor = &v
}

// GetContent returns the Content field value.
func (o *NoteWidgetDefinition) GetContent() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Content
}

// GetContentOk returns a tuple with the Content field value
// and a boolean to check if the value has been set.
func (o *NoteWidgetDefinition) GetContentOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Content, true
}

// SetContent sets field value.
func (o *NoteWidgetDefinition) SetContent(v string) {
	o.Content = v
}

// GetFontSize returns the FontSize field value if set, zero value otherwise.
func (o *NoteWidgetDefinition) GetFontSize() string {
	if o == nil || o.FontSize == nil {
		var ret string
		return ret
	}
	return *o.FontSize
}

// GetFontSizeOk returns a tuple with the FontSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NoteWidgetDefinition) GetFontSizeOk() (*string, bool) {
	if o == nil || o.FontSize == nil {
		return nil, false
	}
	return o.FontSize, true
}

// HasFontSize returns a boolean if a field has been set.
func (o *NoteWidgetDefinition) HasFontSize() bool {
	if o != nil && o.FontSize != nil {
		return true
	}

	return false
}

// SetFontSize gets a reference to the given string and assigns it to the FontSize field.
func (o *NoteWidgetDefinition) SetFontSize(v string) {
	o.FontSize = &v
}

// GetHasPadding returns the HasPadding field value if set, zero value otherwise.
func (o *NoteWidgetDefinition) GetHasPadding() bool {
	if o == nil || o.HasPadding == nil {
		var ret bool
		return ret
	}
	return *o.HasPadding
}

// GetHasPaddingOk returns a tuple with the HasPadding field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NoteWidgetDefinition) GetHasPaddingOk() (*bool, bool) {
	if o == nil || o.HasPadding == nil {
		return nil, false
	}
	return o.HasPadding, true
}

// HasHasPadding returns a boolean if a field has been set.
func (o *NoteWidgetDefinition) HasHasPadding() bool {
	if o != nil && o.HasPadding != nil {
		return true
	}

	return false
}

// SetHasPadding gets a reference to the given bool and assigns it to the HasPadding field.
func (o *NoteWidgetDefinition) SetHasPadding(v bool) {
	o.HasPadding = &v
}

// GetShowTick returns the ShowTick field value if set, zero value otherwise.
func (o *NoteWidgetDefinition) GetShowTick() bool {
	if o == nil || o.ShowTick == nil {
		var ret bool
		return ret
	}
	return *o.ShowTick
}

// GetShowTickOk returns a tuple with the ShowTick field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NoteWidgetDefinition) GetShowTickOk() (*bool, bool) {
	if o == nil || o.ShowTick == nil {
		return nil, false
	}
	return o.ShowTick, true
}

// HasShowTick returns a boolean if a field has been set.
func (o *NoteWidgetDefinition) HasShowTick() bool {
	if o != nil && o.ShowTick != nil {
		return true
	}

	return false
}

// SetShowTick gets a reference to the given bool and assigns it to the ShowTick field.
func (o *NoteWidgetDefinition) SetShowTick(v bool) {
	o.ShowTick = &v
}

// GetTextAlign returns the TextAlign field value if set, zero value otherwise.
func (o *NoteWidgetDefinition) GetTextAlign() WidgetTextAlign {
	if o == nil || o.TextAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TextAlign
}

// GetTextAlignOk returns a tuple with the TextAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NoteWidgetDefinition) GetTextAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TextAlign == nil {
		return nil, false
	}
	return o.TextAlign, true
}

// HasTextAlign returns a boolean if a field has been set.
func (o *NoteWidgetDefinition) HasTextAlign() bool {
	if o != nil && o.TextAlign != nil {
		return true
	}

	return false
}

// SetTextAlign gets a reference to the given WidgetTextAlign and assigns it to the TextAlign field.
func (o *NoteWidgetDefinition) SetTextAlign(v WidgetTextAlign) {
	o.TextAlign = &v
}

// GetTickEdge returns the TickEdge field value if set, zero value otherwise.
func (o *NoteWidgetDefinition) GetTickEdge() WidgetTickEdge {
	if o == nil || o.TickEdge == nil {
		var ret WidgetTickEdge
		return ret
	}
	return *o.TickEdge
}

// GetTickEdgeOk returns a tuple with the TickEdge field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NoteWidgetDefinition) GetTickEdgeOk() (*WidgetTickEdge, bool) {
	if o == nil || o.TickEdge == nil {
		return nil, false
	}
	return o.TickEdge, true
}

// HasTickEdge returns a boolean if a field has been set.
func (o *NoteWidgetDefinition) HasTickEdge() bool {
	if o != nil && o.TickEdge != nil {
		return true
	}

	return false
}

// SetTickEdge gets a reference to the given WidgetTickEdge and assigns it to the TickEdge field.
func (o *NoteWidgetDefinition) SetTickEdge(v WidgetTickEdge) {
	o.TickEdge = &v
}

// GetTickPos returns the TickPos field value if set, zero value otherwise.
func (o *NoteWidgetDefinition) GetTickPos() string {
	if o == nil || o.TickPos == nil {
		var ret string
		return ret
	}
	return *o.TickPos
}

// GetTickPosOk returns a tuple with the TickPos field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NoteWidgetDefinition) GetTickPosOk() (*string, bool) {
	if o == nil || o.TickPos == nil {
		return nil, false
	}
	return o.TickPos, true
}

// HasTickPos returns a boolean if a field has been set.
func (o *NoteWidgetDefinition) HasTickPos() bool {
	if o != nil && o.TickPos != nil {
		return true
	}

	return false
}

// SetTickPos gets a reference to the given string and assigns it to the TickPos field.
func (o *NoteWidgetDefinition) SetTickPos(v string) {
	o.TickPos = &v
}

// GetType returns the Type field value.
func (o *NoteWidgetDefinition) GetType() NoteWidgetDefinitionType {
	if o == nil {
		var ret NoteWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *NoteWidgetDefinition) GetTypeOk() (*NoteWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *NoteWidgetDefinition) SetType(v NoteWidgetDefinitionType) {
	o.Type = v
}

// GetVerticalAlign returns the VerticalAlign field value if set, zero value otherwise.
func (o *NoteWidgetDefinition) GetVerticalAlign() WidgetVerticalAlign {
	if o == nil || o.VerticalAlign == nil {
		var ret WidgetVerticalAlign
		return ret
	}
	return *o.VerticalAlign
}

// GetVerticalAlignOk returns a tuple with the VerticalAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NoteWidgetDefinition) GetVerticalAlignOk() (*WidgetVerticalAlign, bool) {
	if o == nil || o.VerticalAlign == nil {
		return nil, false
	}
	return o.VerticalAlign, true
}

// HasVerticalAlign returns a boolean if a field has been set.
func (o *NoteWidgetDefinition) HasVerticalAlign() bool {
	if o != nil && o.VerticalAlign != nil {
		return true
	}

	return false
}

// SetVerticalAlign gets a reference to the given WidgetVerticalAlign and assigns it to the VerticalAlign field.
func (o *NoteWidgetDefinition) SetVerticalAlign(v WidgetVerticalAlign) {
	o.VerticalAlign = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o NoteWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.BackgroundColor != nil {
		toSerialize["background_color"] = o.BackgroundColor
	}
	toSerialize["content"] = o.Content
	if o.FontSize != nil {
		toSerialize["font_size"] = o.FontSize
	}
	if o.HasPadding != nil {
		toSerialize["has_padding"] = o.HasPadding
	}
	if o.ShowTick != nil {
		toSerialize["show_tick"] = o.ShowTick
	}
	if o.TextAlign != nil {
		toSerialize["text_align"] = o.TextAlign
	}
	if o.TickEdge != nil {
		toSerialize["tick_edge"] = o.TickEdge
	}
	if o.TickPos != nil {
		toSerialize["tick_pos"] = o.TickPos
	}
	toSerialize["type"] = o.Type
	if o.VerticalAlign != nil {
		toSerialize["vertical_align"] = o.VerticalAlign
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *NoteWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Content *string                   `json:"content"`
		Type    *NoteWidgetDefinitionType `json:"type"`
	}{}
	all := struct {
		BackgroundColor *string                  `json:"background_color,omitempty"`
		Content         string                   `json:"content"`
		FontSize        *string                  `json:"font_size,omitempty"`
		HasPadding      *bool                    `json:"has_padding,omitempty"`
		ShowTick        *bool                    `json:"show_tick,omitempty"`
		TextAlign       *WidgetTextAlign         `json:"text_align,omitempty"`
		TickEdge        *WidgetTickEdge          `json:"tick_edge,omitempty"`
		TickPos         *string                  `json:"tick_pos,omitempty"`
		Type            NoteWidgetDefinitionType `json:"type"`
		VerticalAlign   *WidgetVerticalAlign     `json:"vertical_align,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Content == nil {
		return fmt.Errorf("Required field content missing")
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
	if v := all.TickEdge; v != nil && !v.IsValid() {
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
	o.BackgroundColor = all.BackgroundColor
	o.Content = all.Content
	o.FontSize = all.FontSize
	o.HasPadding = all.HasPadding
	o.ShowTick = all.ShowTick
	o.TextAlign = all.TextAlign
	o.TickEdge = all.TickEdge
	o.TickPos = all.TickPos
	o.Type = all.Type
	o.VerticalAlign = all.VerticalAlign
	return nil
}
