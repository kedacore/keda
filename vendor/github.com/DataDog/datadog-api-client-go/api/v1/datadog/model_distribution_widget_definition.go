// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// DistributionWidgetDefinition The Distribution visualization is another way of showing metrics
// aggregated across one or several tags, such as hosts.
// Unlike the heat map, a distribution graph’s x-axis is quantity rather than time.
type DistributionWidgetDefinition struct {
	// (Deprecated) The widget legend was replaced by a tooltip and sidebar.
	// Deprecated
	LegendSize *string `json:"legend_size,omitempty"`
	// List of markers.
	Markers []WidgetMarker `json:"markers,omitempty"`
	// Array of one request object to display in the widget.
	//
	// See the dedicated [Request JSON schema documentation](https://docs.datadoghq.com/dashboards/graphing_json/request_json)
	//  to learn how to build the `REQUEST_SCHEMA`.
	Requests []DistributionWidgetRequest `json:"requests"`
	// (Deprecated) The widget legend was replaced by a tooltip and sidebar.
	// Deprecated
	ShowLegend *bool `json:"show_legend,omitempty"`
	// Time setting for the widget.
	Time *WidgetTime `json:"time,omitempty"`
	// Title of the widget.
	Title *string `json:"title,omitempty"`
	// How to align the text on the widget.
	TitleAlign *WidgetTextAlign `json:"title_align,omitempty"`
	// Size of the title.
	TitleSize *string `json:"title_size,omitempty"`
	// Type of the distribution widget.
	Type DistributionWidgetDefinitionType `json:"type"`
	// X Axis controls for the distribution widget.
	Xaxis *DistributionWidgetXAxis `json:"xaxis,omitempty"`
	// Y Axis controls for the distribution widget.
	Yaxis *DistributionWidgetYAxis `json:"yaxis,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewDistributionWidgetDefinition instantiates a new DistributionWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewDistributionWidgetDefinition(requests []DistributionWidgetRequest, typeVar DistributionWidgetDefinitionType) *DistributionWidgetDefinition {
	this := DistributionWidgetDefinition{}
	this.Requests = requests
	this.Type = typeVar
	return &this
}

// NewDistributionWidgetDefinitionWithDefaults instantiates a new DistributionWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewDistributionWidgetDefinitionWithDefaults() *DistributionWidgetDefinition {
	this := DistributionWidgetDefinition{}
	var typeVar DistributionWidgetDefinitionType = DISTRIBUTIONWIDGETDEFINITIONTYPE_DISTRIBUTION
	this.Type = typeVar
	return &this
}

// GetLegendSize returns the LegendSize field value if set, zero value otherwise.
// Deprecated
func (o *DistributionWidgetDefinition) GetLegendSize() string {
	if o == nil || o.LegendSize == nil {
		var ret string
		return ret
	}
	return *o.LegendSize
}

// GetLegendSizeOk returns a tuple with the LegendSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
// Deprecated
func (o *DistributionWidgetDefinition) GetLegendSizeOk() (*string, bool) {
	if o == nil || o.LegendSize == nil {
		return nil, false
	}
	return o.LegendSize, true
}

// HasLegendSize returns a boolean if a field has been set.
func (o *DistributionWidgetDefinition) HasLegendSize() bool {
	if o != nil && o.LegendSize != nil {
		return true
	}

	return false
}

// SetLegendSize gets a reference to the given string and assigns it to the LegendSize field.
// Deprecated
func (o *DistributionWidgetDefinition) SetLegendSize(v string) {
	o.LegendSize = &v
}

// GetMarkers returns the Markers field value if set, zero value otherwise.
func (o *DistributionWidgetDefinition) GetMarkers() []WidgetMarker {
	if o == nil || o.Markers == nil {
		var ret []WidgetMarker
		return ret
	}
	return o.Markers
}

// GetMarkersOk returns a tuple with the Markers field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DistributionWidgetDefinition) GetMarkersOk() (*[]WidgetMarker, bool) {
	if o == nil || o.Markers == nil {
		return nil, false
	}
	return &o.Markers, true
}

// HasMarkers returns a boolean if a field has been set.
func (o *DistributionWidgetDefinition) HasMarkers() bool {
	if o != nil && o.Markers != nil {
		return true
	}

	return false
}

// SetMarkers gets a reference to the given []WidgetMarker and assigns it to the Markers field.
func (o *DistributionWidgetDefinition) SetMarkers(v []WidgetMarker) {
	o.Markers = v
}

// GetRequests returns the Requests field value.
func (o *DistributionWidgetDefinition) GetRequests() []DistributionWidgetRequest {
	if o == nil {
		var ret []DistributionWidgetRequest
		return ret
	}
	return o.Requests
}

// GetRequestsOk returns a tuple with the Requests field value
// and a boolean to check if the value has been set.
func (o *DistributionWidgetDefinition) GetRequestsOk() (*[]DistributionWidgetRequest, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Requests, true
}

// SetRequests sets field value.
func (o *DistributionWidgetDefinition) SetRequests(v []DistributionWidgetRequest) {
	o.Requests = v
}

// GetShowLegend returns the ShowLegend field value if set, zero value otherwise.
// Deprecated
func (o *DistributionWidgetDefinition) GetShowLegend() bool {
	if o == nil || o.ShowLegend == nil {
		var ret bool
		return ret
	}
	return *o.ShowLegend
}

// GetShowLegendOk returns a tuple with the ShowLegend field value if set, nil otherwise
// and a boolean to check if the value has been set.
// Deprecated
func (o *DistributionWidgetDefinition) GetShowLegendOk() (*bool, bool) {
	if o == nil || o.ShowLegend == nil {
		return nil, false
	}
	return o.ShowLegend, true
}

// HasShowLegend returns a boolean if a field has been set.
func (o *DistributionWidgetDefinition) HasShowLegend() bool {
	if o != nil && o.ShowLegend != nil {
		return true
	}

	return false
}

// SetShowLegend gets a reference to the given bool and assigns it to the ShowLegend field.
// Deprecated
func (o *DistributionWidgetDefinition) SetShowLegend(v bool) {
	o.ShowLegend = &v
}

// GetTime returns the Time field value if set, zero value otherwise.
func (o *DistributionWidgetDefinition) GetTime() WidgetTime {
	if o == nil || o.Time == nil {
		var ret WidgetTime
		return ret
	}
	return *o.Time
}

// GetTimeOk returns a tuple with the Time field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DistributionWidgetDefinition) GetTimeOk() (*WidgetTime, bool) {
	if o == nil || o.Time == nil {
		return nil, false
	}
	return o.Time, true
}

// HasTime returns a boolean if a field has been set.
func (o *DistributionWidgetDefinition) HasTime() bool {
	if o != nil && o.Time != nil {
		return true
	}

	return false
}

// SetTime gets a reference to the given WidgetTime and assigns it to the Time field.
func (o *DistributionWidgetDefinition) SetTime(v WidgetTime) {
	o.Time = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *DistributionWidgetDefinition) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DistributionWidgetDefinition) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *DistributionWidgetDefinition) HasTitle() bool {
	if o != nil && o.Title != nil {
		return true
	}

	return false
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *DistributionWidgetDefinition) SetTitle(v string) {
	o.Title = &v
}

// GetTitleAlign returns the TitleAlign field value if set, zero value otherwise.
func (o *DistributionWidgetDefinition) GetTitleAlign() WidgetTextAlign {
	if o == nil || o.TitleAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TitleAlign
}

// GetTitleAlignOk returns a tuple with the TitleAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DistributionWidgetDefinition) GetTitleAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TitleAlign == nil {
		return nil, false
	}
	return o.TitleAlign, true
}

// HasTitleAlign returns a boolean if a field has been set.
func (o *DistributionWidgetDefinition) HasTitleAlign() bool {
	if o != nil && o.TitleAlign != nil {
		return true
	}

	return false
}

// SetTitleAlign gets a reference to the given WidgetTextAlign and assigns it to the TitleAlign field.
func (o *DistributionWidgetDefinition) SetTitleAlign(v WidgetTextAlign) {
	o.TitleAlign = &v
}

// GetTitleSize returns the TitleSize field value if set, zero value otherwise.
func (o *DistributionWidgetDefinition) GetTitleSize() string {
	if o == nil || o.TitleSize == nil {
		var ret string
		return ret
	}
	return *o.TitleSize
}

// GetTitleSizeOk returns a tuple with the TitleSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DistributionWidgetDefinition) GetTitleSizeOk() (*string, bool) {
	if o == nil || o.TitleSize == nil {
		return nil, false
	}
	return o.TitleSize, true
}

// HasTitleSize returns a boolean if a field has been set.
func (o *DistributionWidgetDefinition) HasTitleSize() bool {
	if o != nil && o.TitleSize != nil {
		return true
	}

	return false
}

// SetTitleSize gets a reference to the given string and assigns it to the TitleSize field.
func (o *DistributionWidgetDefinition) SetTitleSize(v string) {
	o.TitleSize = &v
}

// GetType returns the Type field value.
func (o *DistributionWidgetDefinition) GetType() DistributionWidgetDefinitionType {
	if o == nil {
		var ret DistributionWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *DistributionWidgetDefinition) GetTypeOk() (*DistributionWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *DistributionWidgetDefinition) SetType(v DistributionWidgetDefinitionType) {
	o.Type = v
}

// GetXaxis returns the Xaxis field value if set, zero value otherwise.
func (o *DistributionWidgetDefinition) GetXaxis() DistributionWidgetXAxis {
	if o == nil || o.Xaxis == nil {
		var ret DistributionWidgetXAxis
		return ret
	}
	return *o.Xaxis
}

// GetXaxisOk returns a tuple with the Xaxis field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DistributionWidgetDefinition) GetXaxisOk() (*DistributionWidgetXAxis, bool) {
	if o == nil || o.Xaxis == nil {
		return nil, false
	}
	return o.Xaxis, true
}

// HasXaxis returns a boolean if a field has been set.
func (o *DistributionWidgetDefinition) HasXaxis() bool {
	if o != nil && o.Xaxis != nil {
		return true
	}

	return false
}

// SetXaxis gets a reference to the given DistributionWidgetXAxis and assigns it to the Xaxis field.
func (o *DistributionWidgetDefinition) SetXaxis(v DistributionWidgetXAxis) {
	o.Xaxis = &v
}

// GetYaxis returns the Yaxis field value if set, zero value otherwise.
func (o *DistributionWidgetDefinition) GetYaxis() DistributionWidgetYAxis {
	if o == nil || o.Yaxis == nil {
		var ret DistributionWidgetYAxis
		return ret
	}
	return *o.Yaxis
}

// GetYaxisOk returns a tuple with the Yaxis field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DistributionWidgetDefinition) GetYaxisOk() (*DistributionWidgetYAxis, bool) {
	if o == nil || o.Yaxis == nil {
		return nil, false
	}
	return o.Yaxis, true
}

// HasYaxis returns a boolean if a field has been set.
func (o *DistributionWidgetDefinition) HasYaxis() bool {
	if o != nil && o.Yaxis != nil {
		return true
	}

	return false
}

// SetYaxis gets a reference to the given DistributionWidgetYAxis and assigns it to the Yaxis field.
func (o *DistributionWidgetDefinition) SetYaxis(v DistributionWidgetYAxis) {
	o.Yaxis = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o DistributionWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.LegendSize != nil {
		toSerialize["legend_size"] = o.LegendSize
	}
	if o.Markers != nil {
		toSerialize["markers"] = o.Markers
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
func (o *DistributionWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Requests *[]DistributionWidgetRequest      `json:"requests"`
		Type     *DistributionWidgetDefinitionType `json:"type"`
	}{}
	all := struct {
		LegendSize *string                          `json:"legend_size,omitempty"`
		Markers    []WidgetMarker                   `json:"markers,omitempty"`
		Requests   []DistributionWidgetRequest      `json:"requests"`
		ShowLegend *bool                            `json:"show_legend,omitempty"`
		Time       *WidgetTime                      `json:"time,omitempty"`
		Title      *string                          `json:"title,omitempty"`
		TitleAlign *WidgetTextAlign                 `json:"title_align,omitempty"`
		TitleSize  *string                          `json:"title_size,omitempty"`
		Type       DistributionWidgetDefinitionType `json:"type"`
		Xaxis      *DistributionWidgetXAxis         `json:"xaxis,omitempty"`
		Yaxis      *DistributionWidgetYAxis         `json:"yaxis,omitempty"`
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
	o.Markers = all.Markers
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
