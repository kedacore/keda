// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ListStreamWidgetDefinition The list stream visualization displays a table of recent events in your application that
// match a search criteria using user-defined columns.
//
type ListStreamWidgetDefinition struct {
	// Available legend sizes for a widget. Should be one of "0", "2", "4", "8", "16", or "auto".
	LegendSize *string `json:"legend_size,omitempty"`
	// Request payload used to query items.
	Requests []ListStreamWidgetRequest `json:"requests"`
	// Whether or not to display the legend on this widget.
	ShowLegend *bool `json:"show_legend,omitempty"`
	// Time setting for the widget.
	Time *WidgetTime `json:"time,omitempty"`
	// Title of the widget.
	Title *string `json:"title,omitempty"`
	// How to align the text on the widget.
	TitleAlign *WidgetTextAlign `json:"title_align,omitempty"`
	// Size of the title.
	TitleSize *string `json:"title_size,omitempty"`
	// Type of the list stream widget.
	Type ListStreamWidgetDefinitionType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewListStreamWidgetDefinition instantiates a new ListStreamWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewListStreamWidgetDefinition(requests []ListStreamWidgetRequest, typeVar ListStreamWidgetDefinitionType) *ListStreamWidgetDefinition {
	this := ListStreamWidgetDefinition{}
	this.Requests = requests
	this.Type = typeVar
	return &this
}

// NewListStreamWidgetDefinitionWithDefaults instantiates a new ListStreamWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewListStreamWidgetDefinitionWithDefaults() *ListStreamWidgetDefinition {
	this := ListStreamWidgetDefinition{}
	var typeVar ListStreamWidgetDefinitionType = LISTSTREAMWIDGETDEFINITIONTYPE_LIST_STREAM
	this.Type = typeVar
	return &this
}

// GetLegendSize returns the LegendSize field value if set, zero value otherwise.
func (o *ListStreamWidgetDefinition) GetLegendSize() string {
	if o == nil || o.LegendSize == nil {
		var ret string
		return ret
	}
	return *o.LegendSize
}

// GetLegendSizeOk returns a tuple with the LegendSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ListStreamWidgetDefinition) GetLegendSizeOk() (*string, bool) {
	if o == nil || o.LegendSize == nil {
		return nil, false
	}
	return o.LegendSize, true
}

// HasLegendSize returns a boolean if a field has been set.
func (o *ListStreamWidgetDefinition) HasLegendSize() bool {
	if o != nil && o.LegendSize != nil {
		return true
	}

	return false
}

// SetLegendSize gets a reference to the given string and assigns it to the LegendSize field.
func (o *ListStreamWidgetDefinition) SetLegendSize(v string) {
	o.LegendSize = &v
}

// GetRequests returns the Requests field value.
func (o *ListStreamWidgetDefinition) GetRequests() []ListStreamWidgetRequest {
	if o == nil {
		var ret []ListStreamWidgetRequest
		return ret
	}
	return o.Requests
}

// GetRequestsOk returns a tuple with the Requests field value
// and a boolean to check if the value has been set.
func (o *ListStreamWidgetDefinition) GetRequestsOk() (*[]ListStreamWidgetRequest, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Requests, true
}

// SetRequests sets field value.
func (o *ListStreamWidgetDefinition) SetRequests(v []ListStreamWidgetRequest) {
	o.Requests = v
}

// GetShowLegend returns the ShowLegend field value if set, zero value otherwise.
func (o *ListStreamWidgetDefinition) GetShowLegend() bool {
	if o == nil || o.ShowLegend == nil {
		var ret bool
		return ret
	}
	return *o.ShowLegend
}

// GetShowLegendOk returns a tuple with the ShowLegend field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ListStreamWidgetDefinition) GetShowLegendOk() (*bool, bool) {
	if o == nil || o.ShowLegend == nil {
		return nil, false
	}
	return o.ShowLegend, true
}

// HasShowLegend returns a boolean if a field has been set.
func (o *ListStreamWidgetDefinition) HasShowLegend() bool {
	if o != nil && o.ShowLegend != nil {
		return true
	}

	return false
}

// SetShowLegend gets a reference to the given bool and assigns it to the ShowLegend field.
func (o *ListStreamWidgetDefinition) SetShowLegend(v bool) {
	o.ShowLegend = &v
}

// GetTime returns the Time field value if set, zero value otherwise.
func (o *ListStreamWidgetDefinition) GetTime() WidgetTime {
	if o == nil || o.Time == nil {
		var ret WidgetTime
		return ret
	}
	return *o.Time
}

// GetTimeOk returns a tuple with the Time field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ListStreamWidgetDefinition) GetTimeOk() (*WidgetTime, bool) {
	if o == nil || o.Time == nil {
		return nil, false
	}
	return o.Time, true
}

// HasTime returns a boolean if a field has been set.
func (o *ListStreamWidgetDefinition) HasTime() bool {
	if o != nil && o.Time != nil {
		return true
	}

	return false
}

// SetTime gets a reference to the given WidgetTime and assigns it to the Time field.
func (o *ListStreamWidgetDefinition) SetTime(v WidgetTime) {
	o.Time = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *ListStreamWidgetDefinition) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ListStreamWidgetDefinition) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *ListStreamWidgetDefinition) HasTitle() bool {
	if o != nil && o.Title != nil {
		return true
	}

	return false
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *ListStreamWidgetDefinition) SetTitle(v string) {
	o.Title = &v
}

// GetTitleAlign returns the TitleAlign field value if set, zero value otherwise.
func (o *ListStreamWidgetDefinition) GetTitleAlign() WidgetTextAlign {
	if o == nil || o.TitleAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TitleAlign
}

// GetTitleAlignOk returns a tuple with the TitleAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ListStreamWidgetDefinition) GetTitleAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TitleAlign == nil {
		return nil, false
	}
	return o.TitleAlign, true
}

// HasTitleAlign returns a boolean if a field has been set.
func (o *ListStreamWidgetDefinition) HasTitleAlign() bool {
	if o != nil && o.TitleAlign != nil {
		return true
	}

	return false
}

// SetTitleAlign gets a reference to the given WidgetTextAlign and assigns it to the TitleAlign field.
func (o *ListStreamWidgetDefinition) SetTitleAlign(v WidgetTextAlign) {
	o.TitleAlign = &v
}

// GetTitleSize returns the TitleSize field value if set, zero value otherwise.
func (o *ListStreamWidgetDefinition) GetTitleSize() string {
	if o == nil || o.TitleSize == nil {
		var ret string
		return ret
	}
	return *o.TitleSize
}

// GetTitleSizeOk returns a tuple with the TitleSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ListStreamWidgetDefinition) GetTitleSizeOk() (*string, bool) {
	if o == nil || o.TitleSize == nil {
		return nil, false
	}
	return o.TitleSize, true
}

// HasTitleSize returns a boolean if a field has been set.
func (o *ListStreamWidgetDefinition) HasTitleSize() bool {
	if o != nil && o.TitleSize != nil {
		return true
	}

	return false
}

// SetTitleSize gets a reference to the given string and assigns it to the TitleSize field.
func (o *ListStreamWidgetDefinition) SetTitleSize(v string) {
	o.TitleSize = &v
}

// GetType returns the Type field value.
func (o *ListStreamWidgetDefinition) GetType() ListStreamWidgetDefinitionType {
	if o == nil {
		var ret ListStreamWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *ListStreamWidgetDefinition) GetTypeOk() (*ListStreamWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *ListStreamWidgetDefinition) SetType(v ListStreamWidgetDefinitionType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ListStreamWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.LegendSize != nil {
		toSerialize["legend_size"] = o.LegendSize
	}
	toSerialize["requests"] = o.Requests
	if o.ShowLegend != nil {
		toSerialize["show_legend"] = o.ShowLegend
	}
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
func (o *ListStreamWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Requests *[]ListStreamWidgetRequest      `json:"requests"`
		Type     *ListStreamWidgetDefinitionType `json:"type"`
	}{}
	all := struct {
		LegendSize *string                        `json:"legend_size,omitempty"`
		Requests   []ListStreamWidgetRequest      `json:"requests"`
		ShowLegend *bool                          `json:"show_legend,omitempty"`
		Time       *WidgetTime                    `json:"time,omitempty"`
		Title      *string                        `json:"title,omitempty"`
		TitleAlign *WidgetTextAlign               `json:"title_align,omitempty"`
		TitleSize  *string                        `json:"title_size,omitempty"`
		Type       ListStreamWidgetDefinitionType `json:"type"`
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
	o.LegendSize = all.LegendSize
	o.Requests = all.Requests
	o.ShowLegend = all.ShowLegend
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
