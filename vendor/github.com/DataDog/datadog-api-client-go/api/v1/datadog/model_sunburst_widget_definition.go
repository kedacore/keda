// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SunburstWidgetDefinition Sunbursts are spot on to highlight how groups contribute to the total of a query.
type SunburstWidgetDefinition struct {
	// List of custom links.
	CustomLinks []WidgetCustomLink `json:"custom_links,omitempty"`
	// Show the total value in this widget.
	HideTotal *bool `json:"hide_total,omitempty"`
	// Configuration of the legend.
	Legend *SunburstWidgetLegend `json:"legend,omitempty"`
	// List of sunburst widget requests.
	Requests []SunburstWidgetRequest `json:"requests"`
	// Time setting for the widget.
	Time *WidgetTime `json:"time,omitempty"`
	// Title of your widget.
	Title *string `json:"title,omitempty"`
	// How to align the text on the widget.
	TitleAlign *WidgetTextAlign `json:"title_align,omitempty"`
	// Size of the title.
	TitleSize *string `json:"title_size,omitempty"`
	// Type of the Sunburst widget.
	Type SunburstWidgetDefinitionType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSunburstWidgetDefinition instantiates a new SunburstWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSunburstWidgetDefinition(requests []SunburstWidgetRequest, typeVar SunburstWidgetDefinitionType) *SunburstWidgetDefinition {
	this := SunburstWidgetDefinition{}
	this.Requests = requests
	this.Type = typeVar
	return &this
}

// NewSunburstWidgetDefinitionWithDefaults instantiates a new SunburstWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSunburstWidgetDefinitionWithDefaults() *SunburstWidgetDefinition {
	this := SunburstWidgetDefinition{}
	var typeVar SunburstWidgetDefinitionType = SUNBURSTWIDGETDEFINITIONTYPE_SUNBURST
	this.Type = typeVar
	return &this
}

// GetCustomLinks returns the CustomLinks field value if set, zero value otherwise.
func (o *SunburstWidgetDefinition) GetCustomLinks() []WidgetCustomLink {
	if o == nil || o.CustomLinks == nil {
		var ret []WidgetCustomLink
		return ret
	}
	return o.CustomLinks
}

// GetCustomLinksOk returns a tuple with the CustomLinks field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SunburstWidgetDefinition) GetCustomLinksOk() (*[]WidgetCustomLink, bool) {
	if o == nil || o.CustomLinks == nil {
		return nil, false
	}
	return &o.CustomLinks, true
}

// HasCustomLinks returns a boolean if a field has been set.
func (o *SunburstWidgetDefinition) HasCustomLinks() bool {
	if o != nil && o.CustomLinks != nil {
		return true
	}

	return false
}

// SetCustomLinks gets a reference to the given []WidgetCustomLink and assigns it to the CustomLinks field.
func (o *SunburstWidgetDefinition) SetCustomLinks(v []WidgetCustomLink) {
	o.CustomLinks = v
}

// GetHideTotal returns the HideTotal field value if set, zero value otherwise.
func (o *SunburstWidgetDefinition) GetHideTotal() bool {
	if o == nil || o.HideTotal == nil {
		var ret bool
		return ret
	}
	return *o.HideTotal
}

// GetHideTotalOk returns a tuple with the HideTotal field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SunburstWidgetDefinition) GetHideTotalOk() (*bool, bool) {
	if o == nil || o.HideTotal == nil {
		return nil, false
	}
	return o.HideTotal, true
}

// HasHideTotal returns a boolean if a field has been set.
func (o *SunburstWidgetDefinition) HasHideTotal() bool {
	if o != nil && o.HideTotal != nil {
		return true
	}

	return false
}

// SetHideTotal gets a reference to the given bool and assigns it to the HideTotal field.
func (o *SunburstWidgetDefinition) SetHideTotal(v bool) {
	o.HideTotal = &v
}

// GetLegend returns the Legend field value if set, zero value otherwise.
func (o *SunburstWidgetDefinition) GetLegend() SunburstWidgetLegend {
	if o == nil || o.Legend == nil {
		var ret SunburstWidgetLegend
		return ret
	}
	return *o.Legend
}

// GetLegendOk returns a tuple with the Legend field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SunburstWidgetDefinition) GetLegendOk() (*SunburstWidgetLegend, bool) {
	if o == nil || o.Legend == nil {
		return nil, false
	}
	return o.Legend, true
}

// HasLegend returns a boolean if a field has been set.
func (o *SunburstWidgetDefinition) HasLegend() bool {
	if o != nil && o.Legend != nil {
		return true
	}

	return false
}

// SetLegend gets a reference to the given SunburstWidgetLegend and assigns it to the Legend field.
func (o *SunburstWidgetDefinition) SetLegend(v SunburstWidgetLegend) {
	o.Legend = &v
}

// GetRequests returns the Requests field value.
func (o *SunburstWidgetDefinition) GetRequests() []SunburstWidgetRequest {
	if o == nil {
		var ret []SunburstWidgetRequest
		return ret
	}
	return o.Requests
}

// GetRequestsOk returns a tuple with the Requests field value
// and a boolean to check if the value has been set.
func (o *SunburstWidgetDefinition) GetRequestsOk() (*[]SunburstWidgetRequest, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Requests, true
}

// SetRequests sets field value.
func (o *SunburstWidgetDefinition) SetRequests(v []SunburstWidgetRequest) {
	o.Requests = v
}

// GetTime returns the Time field value if set, zero value otherwise.
func (o *SunburstWidgetDefinition) GetTime() WidgetTime {
	if o == nil || o.Time == nil {
		var ret WidgetTime
		return ret
	}
	return *o.Time
}

// GetTimeOk returns a tuple with the Time field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SunburstWidgetDefinition) GetTimeOk() (*WidgetTime, bool) {
	if o == nil || o.Time == nil {
		return nil, false
	}
	return o.Time, true
}

// HasTime returns a boolean if a field has been set.
func (o *SunburstWidgetDefinition) HasTime() bool {
	if o != nil && o.Time != nil {
		return true
	}

	return false
}

// SetTime gets a reference to the given WidgetTime and assigns it to the Time field.
func (o *SunburstWidgetDefinition) SetTime(v WidgetTime) {
	o.Time = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *SunburstWidgetDefinition) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SunburstWidgetDefinition) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *SunburstWidgetDefinition) HasTitle() bool {
	if o != nil && o.Title != nil {
		return true
	}

	return false
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *SunburstWidgetDefinition) SetTitle(v string) {
	o.Title = &v
}

// GetTitleAlign returns the TitleAlign field value if set, zero value otherwise.
func (o *SunburstWidgetDefinition) GetTitleAlign() WidgetTextAlign {
	if o == nil || o.TitleAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TitleAlign
}

// GetTitleAlignOk returns a tuple with the TitleAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SunburstWidgetDefinition) GetTitleAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TitleAlign == nil {
		return nil, false
	}
	return o.TitleAlign, true
}

// HasTitleAlign returns a boolean if a field has been set.
func (o *SunburstWidgetDefinition) HasTitleAlign() bool {
	if o != nil && o.TitleAlign != nil {
		return true
	}

	return false
}

// SetTitleAlign gets a reference to the given WidgetTextAlign and assigns it to the TitleAlign field.
func (o *SunburstWidgetDefinition) SetTitleAlign(v WidgetTextAlign) {
	o.TitleAlign = &v
}

// GetTitleSize returns the TitleSize field value if set, zero value otherwise.
func (o *SunburstWidgetDefinition) GetTitleSize() string {
	if o == nil || o.TitleSize == nil {
		var ret string
		return ret
	}
	return *o.TitleSize
}

// GetTitleSizeOk returns a tuple with the TitleSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SunburstWidgetDefinition) GetTitleSizeOk() (*string, bool) {
	if o == nil || o.TitleSize == nil {
		return nil, false
	}
	return o.TitleSize, true
}

// HasTitleSize returns a boolean if a field has been set.
func (o *SunburstWidgetDefinition) HasTitleSize() bool {
	if o != nil && o.TitleSize != nil {
		return true
	}

	return false
}

// SetTitleSize gets a reference to the given string and assigns it to the TitleSize field.
func (o *SunburstWidgetDefinition) SetTitleSize(v string) {
	o.TitleSize = &v
}

// GetType returns the Type field value.
func (o *SunburstWidgetDefinition) GetType() SunburstWidgetDefinitionType {
	if o == nil {
		var ret SunburstWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *SunburstWidgetDefinition) GetTypeOk() (*SunburstWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *SunburstWidgetDefinition) SetType(v SunburstWidgetDefinitionType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SunburstWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CustomLinks != nil {
		toSerialize["custom_links"] = o.CustomLinks
	}
	if o.HideTotal != nil {
		toSerialize["hide_total"] = o.HideTotal
	}
	if o.Legend != nil {
		toSerialize["legend"] = o.Legend
	}
	toSerialize["requests"] = o.Requests
	if o.Time != nil {
		toSerialize["time"] = o.Time
	}
	if o.Title != nil {
		toSerialize["title"] = o.Title
	}
	if o.TitleAlign != nil {
		toSerialize["title_align"] = o.TitleAlign
	}
	if o.TitleSize != nil {
		toSerialize["title_size"] = o.TitleSize
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SunburstWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Requests *[]SunburstWidgetRequest      `json:"requests"`
		Type     *SunburstWidgetDefinitionType `json:"type"`
	}{}
	all := struct {
		CustomLinks []WidgetCustomLink           `json:"custom_links,omitempty"`
		HideTotal   *bool                        `json:"hide_total,omitempty"`
		Legend      *SunburstWidgetLegend        `json:"legend,omitempty"`
		Requests    []SunburstWidgetRequest      `json:"requests"`
		Time        *WidgetTime                  `json:"time,omitempty"`
		Title       *string                      `json:"title,omitempty"`
		TitleAlign  *WidgetTextAlign             `json:"title_align,omitempty"`
		TitleSize   *string                      `json:"title_size,omitempty"`
		Type        SunburstWidgetDefinitionType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Requests == nil {
		return fmt.Errorf("Required field requests missing")
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
	o.CustomLinks = all.CustomLinks
	o.HideTotal = all.HideTotal
	o.Legend = all.Legend
	o.Requests = all.Requests
	if all.Time != nil && all.Time.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Time = all.Time
	o.Title = all.Title
	o.TitleAlign = all.TitleAlign
	o.TitleSize = all.TitleSize
	o.Type = all.Type
	return nil
}
