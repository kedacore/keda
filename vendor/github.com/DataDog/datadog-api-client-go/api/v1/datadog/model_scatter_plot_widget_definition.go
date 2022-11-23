// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ScatterPlotWidgetDefinition The scatter plot visualization allows you to graph a chosen scope over two different metrics with their respective aggregation.
type ScatterPlotWidgetDefinition struct {
	// List of groups used for colors.
	ColorByGroups []string `json:"color_by_groups,omitempty"`
	// List of custom links.
	CustomLinks []WidgetCustomLink `json:"custom_links,omitempty"`
	// Widget definition.
	Requests ScatterPlotWidgetDefinitionRequests `json:"requests"`
	// Time setting for the widget.
	Time *WidgetTime `json:"time,omitempty"`
	// Title of your widget.
	Title *string `json:"title,omitempty"`
	// How to align the text on the widget.
	TitleAlign *WidgetTextAlign `json:"title_align,omitempty"`
	// Size of the title.
	TitleSize *string `json:"title_size,omitempty"`
	// Type of the scatter plot widget.
	Type ScatterPlotWidgetDefinitionType `json:"type"`
	// Axis controls for the widget.
	Xaxis *WidgetAxis `json:"xaxis,omitempty"`
	// Axis controls for the widget.
	Yaxis *WidgetAxis `json:"yaxis,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewScatterPlotWidgetDefinition instantiates a new ScatterPlotWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewScatterPlotWidgetDefinition(requests ScatterPlotWidgetDefinitionRequests, typeVar ScatterPlotWidgetDefinitionType) *ScatterPlotWidgetDefinition {
	this := ScatterPlotWidgetDefinition{}
	this.Requests = requests
	this.Type = typeVar
	return &this
}

// NewScatterPlotWidgetDefinitionWithDefaults instantiates a new ScatterPlotWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewScatterPlotWidgetDefinitionWithDefaults() *ScatterPlotWidgetDefinition {
	this := ScatterPlotWidgetDefinition{}
	var typeVar ScatterPlotWidgetDefinitionType = SCATTERPLOTWIDGETDEFINITIONTYPE_SCATTERPLOT
	this.Type = typeVar
	return &this
}

// GetColorByGroups returns the ColorByGroups field value if set, zero value otherwise.
func (o *ScatterPlotWidgetDefinition) GetColorByGroups() []string {
	if o == nil || o.ColorByGroups == nil {
		var ret []string
		return ret
	}
	return o.ColorByGroups
}

// GetColorByGroupsOk returns a tuple with the ColorByGroups field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ScatterPlotWidgetDefinition) GetColorByGroupsOk() (*[]string, bool) {
	if o == nil || o.ColorByGroups == nil {
		return nil, false
	}
	return &o.ColorByGroups, true
}

// HasColorByGroups returns a boolean if a field has been set.
func (o *ScatterPlotWidgetDefinition) HasColorByGroups() bool {
	if o != nil && o.ColorByGroups != nil {
		return true
	}

	return false
}

// SetColorByGroups gets a reference to the given []string and assigns it to the ColorByGroups field.
func (o *ScatterPlotWidgetDefinition) SetColorByGroups(v []string) {
	o.ColorByGroups = v
}

// GetCustomLinks returns the CustomLinks field value if set, zero value otherwise.
func (o *ScatterPlotWidgetDefinition) GetCustomLinks() []WidgetCustomLink {
	if o == nil || o.CustomLinks == nil {
		var ret []WidgetCustomLink
		return ret
	}
	return o.CustomLinks
}

// GetCustomLinksOk returns a tuple with the CustomLinks field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ScatterPlotWidgetDefinition) GetCustomLinksOk() (*[]WidgetCustomLink, bool) {
	if o == nil || o.CustomLinks == nil {
		return nil, false
	}
	return &o.CustomLinks, true
}

// HasCustomLinks returns a boolean if a field has been set.
func (o *ScatterPlotWidgetDefinition) HasCustomLinks() bool {
	if o != nil && o.CustomLinks != nil {
		return true
	}

	return false
}

// SetCustomLinks gets a reference to the given []WidgetCustomLink and assigns it to the CustomLinks field.
func (o *ScatterPlotWidgetDefinition) SetCustomLinks(v []WidgetCustomLink) {
	o.CustomLinks = v
}

// GetRequests returns the Requests field value.
func (o *ScatterPlotWidgetDefinition) GetRequests() ScatterPlotWidgetDefinitionRequests {
	if o == nil {
		var ret ScatterPlotWidgetDefinitionRequests
		return ret
	}
	return o.Requests
}

// GetRequestsOk returns a tuple with the Requests field value
// and a boolean to check if the value has been set.
func (o *ScatterPlotWidgetDefinition) GetRequestsOk() (*ScatterPlotWidgetDefinitionRequests, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Requests, true
}

// SetRequests sets field value.
func (o *ScatterPlotWidgetDefinition) SetRequests(v ScatterPlotWidgetDefinitionRequests) {
	o.Requests = v
}

// GetTime returns the Time field value if set, zero value otherwise.
func (o *ScatterPlotWidgetDefinition) GetTime() WidgetTime {
	if o == nil || o.Time == nil {
		var ret WidgetTime
		return ret
	}
	return *o.Time
}

// GetTimeOk returns a tuple with the Time field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ScatterPlotWidgetDefinition) GetTimeOk() (*WidgetTime, bool) {
	if o == nil || o.Time == nil {
		return nil, false
	}
	return o.Time, true
}

// HasTime returns a boolean if a field has been set.
func (o *ScatterPlotWidgetDefinition) HasTime() bool {
	if o != nil && o.Time != nil {
		return true
	}

	return false
}

// SetTime gets a reference to the given WidgetTime and assigns it to the Time field.
func (o *ScatterPlotWidgetDefinition) SetTime(v WidgetTime) {
	o.Time = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *ScatterPlotWidgetDefinition) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ScatterPlotWidgetDefinition) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *ScatterPlotWidgetDefinition) HasTitle() bool {
	if o != nil && o.Title != nil {
		return true
	}

	return false
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *ScatterPlotWidgetDefinition) SetTitle(v string) {
	o.Title = &v
}

// GetTitleAlign returns the TitleAlign field value if set, zero value otherwise.
func (o *ScatterPlotWidgetDefinition) GetTitleAlign() WidgetTextAlign {
	if o == nil || o.TitleAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TitleAlign
}

// GetTitleAlignOk returns a tuple with the TitleAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ScatterPlotWidgetDefinition) GetTitleAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TitleAlign == nil {
		return nil, false
	}
	return o.TitleAlign, true
}

// HasTitleAlign returns a boolean if a field has been set.
func (o *ScatterPlotWidgetDefinition) HasTitleAlign() bool {
	if o != nil && o.TitleAlign != nil {
		return true
	}

	return false
}

// SetTitleAlign gets a reference to the given WidgetTextAlign and assigns it to the TitleAlign field.
func (o *ScatterPlotWidgetDefinition) SetTitleAlign(v WidgetTextAlign) {
	o.TitleAlign = &v
}

// GetTitleSize returns the TitleSize field value if set, zero value otherwise.
func (o *ScatterPlotWidgetDefinition) GetTitleSize() string {
	if o == nil || o.TitleSize == nil {
		var ret string
		return ret
	}
	return *o.TitleSize
}

// GetTitleSizeOk returns a tuple with the TitleSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ScatterPlotWidgetDefinition) GetTitleSizeOk() (*string, bool) {
	if o == nil || o.TitleSize == nil {
		return nil, false
	}
	return o.TitleSize, true
}

// HasTitleSize returns a boolean if a field has been set.
func (o *ScatterPlotWidgetDefinition) HasTitleSize() bool {
	if o != nil && o.TitleSize != nil {
		return true
	}

	return false
}

// SetTitleSize gets a reference to the given string and assigns it to the TitleSize field.
func (o *ScatterPlotWidgetDefinition) SetTitleSize(v string) {
	o.TitleSize = &v
}

// GetType returns the Type field value.
func (o *ScatterPlotWidgetDefinition) GetType() ScatterPlotWidgetDefinitionType {
	if o == nil {
		var ret ScatterPlotWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *ScatterPlotWidgetDefinition) GetTypeOk() (*ScatterPlotWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *ScatterPlotWidgetDefinition) SetType(v ScatterPlotWidgetDefinitionType) {
	o.Type = v
}

// GetXaxis returns the Xaxis field value if set, zero value otherwise.
func (o *ScatterPlotWidgetDefinition) GetXaxis() WidgetAxis {
	if o == nil || o.Xaxis == nil {
		var ret WidgetAxis
		return ret
	}
	return *o.Xaxis
}

// GetXaxisOk returns a tuple with the Xaxis field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ScatterPlotWidgetDefinition) GetXaxisOk() (*WidgetAxis, bool) {
	if o == nil || o.Xaxis == nil {
		return nil, false
	}
	return o.Xaxis, true
}

// HasXaxis returns a boolean if a field has been set.
func (o *ScatterPlotWidgetDefinition) HasXaxis() bool {
	if o != nil && o.Xaxis != nil {
		return true
	}

	return false
}

// SetXaxis gets a reference to the given WidgetAxis and assigns it to the Xaxis field.
func (o *ScatterPlotWidgetDefinition) SetXaxis(v WidgetAxis) {
	o.Xaxis = &v
}

// GetYaxis returns the Yaxis field value if set, zero value otherwise.
func (o *ScatterPlotWidgetDefinition) GetYaxis() WidgetAxis {
	if o == nil || o.Yaxis == nil {
		var ret WidgetAxis
		return ret
	}
	return *o.Yaxis
}

// GetYaxisOk returns a tuple with the Yaxis field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ScatterPlotWidgetDefinition) GetYaxisOk() (*WidgetAxis, bool) {
	if o == nil || o.Yaxis == nil {
		return nil, false
	}
	return o.Yaxis, true
}

// HasYaxis returns a boolean if a field has been set.
func (o *ScatterPlotWidgetDefinition) HasYaxis() bool {
	if o != nil && o.Yaxis != nil {
		return true
	}

	return false
}

// SetYaxis gets a reference to the given WidgetAxis and assigns it to the Yaxis field.
func (o *ScatterPlotWidgetDefinition) SetYaxis(v WidgetAxis) {
	o.Yaxis = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ScatterPlotWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ColorByGroups != nil {
		toSerialize["color_by_groups"] = o.ColorByGroups
	}
	if o.CustomLinks != nil {
		toSerialize["custom_links"] = o.CustomLinks
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
	if o.Xaxis != nil {
		toSerialize["xaxis"] = o.Xaxis
	}
	if o.Yaxis != nil {
		toSerialize["yaxis"] = o.Yaxis
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ScatterPlotWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Requests *ScatterPlotWidgetDefinitionRequests `json:"requests"`
		Type     *ScatterPlotWidgetDefinitionType     `json:"type"`
	}{}
	all := struct {
		ColorByGroups []string                            `json:"color_by_groups,omitempty"`
		CustomLinks   []WidgetCustomLink                  `json:"custom_links,omitempty"`
		Requests      ScatterPlotWidgetDefinitionRequests `json:"requests"`
		Time          *WidgetTime                         `json:"time,omitempty"`
		Title         *string                             `json:"title,omitempty"`
		TitleAlign    *WidgetTextAlign                    `json:"title_align,omitempty"`
		TitleSize     *string                             `json:"title_size,omitempty"`
		Type          ScatterPlotWidgetDefinitionType     `json:"type"`
		Xaxis         *WidgetAxis                         `json:"xaxis,omitempty"`
		Yaxis         *WidgetAxis                         `json:"yaxis,omitempty"`
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
	o.ColorByGroups = all.ColorByGroups
	o.CustomLinks = all.CustomLinks
	if all.Requests.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
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
	if all.Xaxis != nil && all.Xaxis.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Xaxis = all.Xaxis
	if all.Yaxis != nil && all.Yaxis.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Yaxis = all.Yaxis
	return nil
}
