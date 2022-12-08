// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// GroupWidgetDefinition The groups widget allows you to keep similar graphs together on your timeboard. Each group has a custom header, can hold one to many graphs, and is collapsible.
type GroupWidgetDefinition struct {
	// Background color of the group title.
	BackgroundColor *string `json:"background_color,omitempty"`
	// URL of image to display as a banner for the group.
	BannerImg *string `json:"banner_img,omitempty"`
	// Layout type of the group.
	LayoutType WidgetLayoutType `json:"layout_type"`
	// Whether to show the title or not.
	ShowTitle *bool `json:"show_title,omitempty"`
	// Title of the widget.
	Title *string `json:"title,omitempty"`
	// How to align the text on the widget.
	TitleAlign *WidgetTextAlign `json:"title_align,omitempty"`
	// Type of the group widget.
	Type GroupWidgetDefinitionType `json:"type"`
	// List of widget groups.
	Widgets []Widget `json:"widgets"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewGroupWidgetDefinition instantiates a new GroupWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewGroupWidgetDefinition(layoutType WidgetLayoutType, typeVar GroupWidgetDefinitionType, widgets []Widget) *GroupWidgetDefinition {
	this := GroupWidgetDefinition{}
	this.LayoutType = layoutType
	var showTitle bool = true
	this.ShowTitle = &showTitle
	this.Type = typeVar
	this.Widgets = widgets
	return &this
}

// NewGroupWidgetDefinitionWithDefaults instantiates a new GroupWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewGroupWidgetDefinitionWithDefaults() *GroupWidgetDefinition {
	this := GroupWidgetDefinition{}
	var showTitle bool = true
	this.ShowTitle = &showTitle
	var typeVar GroupWidgetDefinitionType = GROUPWIDGETDEFINITIONTYPE_GROUP
	this.Type = typeVar
	return &this
}

// GetBackgroundColor returns the BackgroundColor field value if set, zero value otherwise.
func (o *GroupWidgetDefinition) GetBackgroundColor() string {
	if o == nil || o.BackgroundColor == nil {
		var ret string
		return ret
	}
	return *o.BackgroundColor
}

// GetBackgroundColorOk returns a tuple with the BackgroundColor field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GroupWidgetDefinition) GetBackgroundColorOk() (*string, bool) {
	if o == nil || o.BackgroundColor == nil {
		return nil, false
	}
	return o.BackgroundColor, true
}

// HasBackgroundColor returns a boolean if a field has been set.
func (o *GroupWidgetDefinition) HasBackgroundColor() bool {
	if o != nil && o.BackgroundColor != nil {
		return true
	}

	return false
}

// SetBackgroundColor gets a reference to the given string and assigns it to the BackgroundColor field.
func (o *GroupWidgetDefinition) SetBackgroundColor(v string) {
	o.BackgroundColor = &v
}

// GetBannerImg returns the BannerImg field value if set, zero value otherwise.
func (o *GroupWidgetDefinition) GetBannerImg() string {
	if o == nil || o.BannerImg == nil {
		var ret string
		return ret
	}
	return *o.BannerImg
}

// GetBannerImgOk returns a tuple with the BannerImg field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GroupWidgetDefinition) GetBannerImgOk() (*string, bool) {
	if o == nil || o.BannerImg == nil {
		return nil, false
	}
	return o.BannerImg, true
}

// HasBannerImg returns a boolean if a field has been set.
func (o *GroupWidgetDefinition) HasBannerImg() bool {
	if o != nil && o.BannerImg != nil {
		return true
	}

	return false
}

// SetBannerImg gets a reference to the given string and assigns it to the BannerImg field.
func (o *GroupWidgetDefinition) SetBannerImg(v string) {
	o.BannerImg = &v
}

// GetLayoutType returns the LayoutType field value.
func (o *GroupWidgetDefinition) GetLayoutType() WidgetLayoutType {
	if o == nil {
		var ret WidgetLayoutType
		return ret
	}
	return o.LayoutType
}

// GetLayoutTypeOk returns a tuple with the LayoutType field value
// and a boolean to check if the value has been set.
func (o *GroupWidgetDefinition) GetLayoutTypeOk() (*WidgetLayoutType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.LayoutType, true
}

// SetLayoutType sets field value.
func (o *GroupWidgetDefinition) SetLayoutType(v WidgetLayoutType) {
	o.LayoutType = v
}

// GetShowTitle returns the ShowTitle field value if set, zero value otherwise.
func (o *GroupWidgetDefinition) GetShowTitle() bool {
	if o == nil || o.ShowTitle == nil {
		var ret bool
		return ret
	}
	return *o.ShowTitle
}

// GetShowTitleOk returns a tuple with the ShowTitle field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GroupWidgetDefinition) GetShowTitleOk() (*bool, bool) {
	if o == nil || o.ShowTitle == nil {
		return nil, false
	}
	return o.ShowTitle, true
}

// HasShowTitle returns a boolean if a field has been set.
func (o *GroupWidgetDefinition) HasShowTitle() bool {
	if o != nil && o.ShowTitle != nil {
		return true
	}

	return false
}

// SetShowTitle gets a reference to the given bool and assigns it to the ShowTitle field.
func (o *GroupWidgetDefinition) SetShowTitle(v bool) {
	o.ShowTitle = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *GroupWidgetDefinition) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GroupWidgetDefinition) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *GroupWidgetDefinition) HasTitle() bool {
	if o != nil && o.Title != nil {
		return true
	}

	return false
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *GroupWidgetDefinition) SetTitle(v string) {
	o.Title = &v
}

// GetTitleAlign returns the TitleAlign field value if set, zero value otherwise.
func (o *GroupWidgetDefinition) GetTitleAlign() WidgetTextAlign {
	if o == nil || o.TitleAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TitleAlign
}

// GetTitleAlignOk returns a tuple with the TitleAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GroupWidgetDefinition) GetTitleAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TitleAlign == nil {
		return nil, false
	}
	return o.TitleAlign, true
}

// HasTitleAlign returns a boolean if a field has been set.
func (o *GroupWidgetDefinition) HasTitleAlign() bool {
	if o != nil && o.TitleAlign != nil {
		return true
	}

	return false
}

// SetTitleAlign gets a reference to the given WidgetTextAlign and assigns it to the TitleAlign field.
func (o *GroupWidgetDefinition) SetTitleAlign(v WidgetTextAlign) {
	o.TitleAlign = &v
}

// GetType returns the Type field value.
func (o *GroupWidgetDefinition) GetType() GroupWidgetDefinitionType {
	if o == nil {
		var ret GroupWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *GroupWidgetDefinition) GetTypeOk() (*GroupWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *GroupWidgetDefinition) SetType(v GroupWidgetDefinitionType) {
	o.Type = v
}

// GetWidgets returns the Widgets field value.
func (o *GroupWidgetDefinition) GetWidgets() []Widget {
	if o == nil {
		var ret []Widget
		return ret
	}
	return o.Widgets
}

// GetWidgetsOk returns a tuple with the Widgets field value
// and a boolean to check if the value has been set.
func (o *GroupWidgetDefinition) GetWidgetsOk() (*[]Widget, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Widgets, true
}

// SetWidgets sets field value.
func (o *GroupWidgetDefinition) SetWidgets(v []Widget) {
	o.Widgets = v
}

// MarshalJSON serializes the struct using spec logic.
func (o GroupWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.BackgroundColor != nil {
		toSerialize["background_color"] = o.BackgroundColor
	}
	if o.BannerImg != nil {
		toSerialize["banner_img"] = o.BannerImg
	}
	toSerialize["layout_type"] = o.LayoutType
	if o.ShowTitle != nil {
		toSerialize["show_title"] = o.ShowTitle
	}
	if o.Title != nil {
		toSerialize["title"] = o.Title
	}
	if o.TitleAlign != nil {
		toSerialize["title_align"] = o.TitleAlign
	}
	toSerialize["type"] = o.Type
	toSerialize["widgets"] = o.Widgets

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *GroupWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		LayoutType *WidgetLayoutType          `json:"layout_type"`
		Type       *GroupWidgetDefinitionType `json:"type"`
		Widgets    *[]Widget                  `json:"widgets"`
	}{}
	all := struct {
		BackgroundColor *string                   `json:"background_color,omitempty"`
		BannerImg       *string                   `json:"banner_img,omitempty"`
		LayoutType      WidgetLayoutType          `json:"layout_type"`
		ShowTitle       *bool                     `json:"show_title,omitempty"`
		Title           *string                   `json:"title,omitempty"`
		TitleAlign      *WidgetTextAlign          `json:"title_align,omitempty"`
		Type            GroupWidgetDefinitionType `json:"type"`
		Widgets         []Widget                  `json:"widgets"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.LayoutType == nil {
		return fmt.Errorf("Required field layout_type missing")
	}
	if required.Type == nil {
		return fmt.Errorf("Required field type missing")
	}
	if required.Widgets == nil {
		return fmt.Errorf("Required field widgets missing")
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
	if v := all.LayoutType; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.TitleAlign; v != nil && !v.IsValid() {
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
	o.BackgroundColor = all.BackgroundColor
	o.BannerImg = all.BannerImg
	o.LayoutType = all.LayoutType
	o.ShowTitle = all.ShowTitle
	o.Title = all.Title
	o.TitleAlign = all.TitleAlign
	o.Type = all.Type
	o.Widgets = all.Widgets
	return nil
}
