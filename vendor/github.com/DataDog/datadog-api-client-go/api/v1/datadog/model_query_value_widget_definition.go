// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// QueryValueWidgetDefinition Query values display the current value of a given metric, APM, or log query.
type QueryValueWidgetDefinition struct {
	// Whether to use auto-scaling or not.
	Autoscale *bool `json:"autoscale,omitempty"`
	// List of custom links.
	CustomLinks []WidgetCustomLink `json:"custom_links,omitempty"`
	// Display a unit of your choice on the widget.
	CustomUnit *string `json:"custom_unit,omitempty"`
	// Number of decimals to show. If not defined, the widget uses the raw value.
	Precision *int64 `json:"precision,omitempty"`
	// Widget definition.
	Requests []QueryValueWidgetRequest `json:"requests"`
	// How to align the text on the widget.
	TextAlign *WidgetTextAlign `json:"text_align,omitempty"`
	// Time setting for the widget.
	Time *WidgetTime `json:"time,omitempty"`
	// Set a timeseries on the widget background.
	TimeseriesBackground *TimeseriesBackground `json:"timeseries_background,omitempty"`
	// Title of your widget.
	Title *string `json:"title,omitempty"`
	// How to align the text on the widget.
	TitleAlign *WidgetTextAlign `json:"title_align,omitempty"`
	// Size of the title.
	TitleSize *string `json:"title_size,omitempty"`
	// Type of the query value widget.
	Type QueryValueWidgetDefinitionType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewQueryValueWidgetDefinition instantiates a new QueryValueWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewQueryValueWidgetDefinition(requests []QueryValueWidgetRequest, typeVar QueryValueWidgetDefinitionType) *QueryValueWidgetDefinition {
	this := QueryValueWidgetDefinition{}
	this.Requests = requests
	this.Type = typeVar
	return &this
}

// NewQueryValueWidgetDefinitionWithDefaults instantiates a new QueryValueWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewQueryValueWidgetDefinitionWithDefaults() *QueryValueWidgetDefinition {
	this := QueryValueWidgetDefinition{}
	var typeVar QueryValueWidgetDefinitionType = QUERYVALUEWIDGETDEFINITIONTYPE_QUERY_VALUE
	this.Type = typeVar
	return &this
}

// GetAutoscale returns the Autoscale field value if set, zero value otherwise.
func (o *QueryValueWidgetDefinition) GetAutoscale() bool {
	if o == nil || o.Autoscale == nil {
		var ret bool
		return ret
	}
	return *o.Autoscale
}

// GetAutoscaleOk returns a tuple with the Autoscale field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *QueryValueWidgetDefinition) GetAutoscaleOk() (*bool, bool) {
	if o == nil || o.Autoscale == nil {
		return nil, false
	}
	return o.Autoscale, true
}

// HasAutoscale returns a boolean if a field has been set.
func (o *QueryValueWidgetDefinition) HasAutoscale() bool {
	if o != nil && o.Autoscale != nil {
		return true
	}

	return false
}

// SetAutoscale gets a reference to the given bool and assigns it to the Autoscale field.
func (o *QueryValueWidgetDefinition) SetAutoscale(v bool) {
	o.Autoscale = &v
}

// GetCustomLinks returns the CustomLinks field value if set, zero value otherwise.
func (o *QueryValueWidgetDefinition) GetCustomLinks() []WidgetCustomLink {
	if o == nil || o.CustomLinks == nil {
		var ret []WidgetCustomLink
		return ret
	}
	return o.CustomLinks
}

// GetCustomLinksOk returns a tuple with the CustomLinks field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *QueryValueWidgetDefinition) GetCustomLinksOk() (*[]WidgetCustomLink, bool) {
	if o == nil || o.CustomLinks == nil {
		return nil, false
	}
	return &o.CustomLinks, true
}

// HasCustomLinks returns a boolean if a field has been set.
func (o *QueryValueWidgetDefinition) HasCustomLinks() bool {
	if o != nil && o.CustomLinks != nil {
		return true
	}

	return false
}

// SetCustomLinks gets a reference to the given []WidgetCustomLink and assigns it to the CustomLinks field.
func (o *QueryValueWidgetDefinition) SetCustomLinks(v []WidgetCustomLink) {
	o.CustomLinks = v
}

// GetCustomUnit returns the CustomUnit field value if set, zero value otherwise.
func (o *QueryValueWidgetDefinition) GetCustomUnit() string {
	if o == nil || o.CustomUnit == nil {
		var ret string
		return ret
	}
	return *o.CustomUnit
}

// GetCustomUnitOk returns a tuple with the CustomUnit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *QueryValueWidgetDefinition) GetCustomUnitOk() (*string, bool) {
	if o == nil || o.CustomUnit == nil {
		return nil, false
	}
	return o.CustomUnit, true
}

// HasCustomUnit returns a boolean if a field has been set.
func (o *QueryValueWidgetDefinition) HasCustomUnit() bool {
	if o != nil && o.CustomUnit != nil {
		return true
	}

	return false
}

// SetCustomUnit gets a reference to the given string and assigns it to the CustomUnit field.
func (o *QueryValueWidgetDefinition) SetCustomUnit(v string) {
	o.CustomUnit = &v
}

// GetPrecision returns the Precision field value if set, zero value otherwise.
func (o *QueryValueWidgetDefinition) GetPrecision() int64 {
	if o == nil || o.Precision == nil {
		var ret int64
		return ret
	}
	return *o.Precision
}

// GetPrecisionOk returns a tuple with the Precision field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *QueryValueWidgetDefinition) GetPrecisionOk() (*int64, bool) {
	if o == nil || o.Precision == nil {
		return nil, false
	}
	return o.Precision, true
}

// HasPrecision returns a boolean if a field has been set.
func (o *QueryValueWidgetDefinition) HasPrecision() bool {
	if o != nil && o.Precision != nil {
		return true
	}

	return false
}

// SetPrecision gets a reference to the given int64 and assigns it to the Precision field.
func (o *QueryValueWidgetDefinition) SetPrecision(v int64) {
	o.Precision = &v
}

// GetRequests returns the Requests field value.
func (o *QueryValueWidgetDefinition) GetRequests() []QueryValueWidgetRequest {
	if o == nil {
		var ret []QueryValueWidgetRequest
		return ret
	}
	return o.Requests
}

// GetRequestsOk returns a tuple with the Requests field value
// and a boolean to check if the value has been set.
func (o *QueryValueWidgetDefinition) GetRequestsOk() (*[]QueryValueWidgetRequest, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Requests, true
}

// SetRequests sets field value.
func (o *QueryValueWidgetDefinition) SetRequests(v []QueryValueWidgetRequest) {
	o.Requests = v
}

// GetTextAlign returns the TextAlign field value if set, zero value otherwise.
func (o *QueryValueWidgetDefinition) GetTextAlign() WidgetTextAlign {
	if o == nil || o.TextAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TextAlign
}

// GetTextAlignOk returns a tuple with the TextAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *QueryValueWidgetDefinition) GetTextAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TextAlign == nil {
		return nil, false
	}
	return o.TextAlign, true
}

// HasTextAlign returns a boolean if a field has been set.
func (o *QueryValueWidgetDefinition) HasTextAlign() bool {
	if o != nil && o.TextAlign != nil {
		return true
	}

	return false
}

// SetTextAlign gets a reference to the given WidgetTextAlign and assigns it to the TextAlign field.
func (o *QueryValueWidgetDefinition) SetTextAlign(v WidgetTextAlign) {
	o.TextAlign = &v
}

// GetTime returns the Time field value if set, zero value otherwise.
func (o *QueryValueWidgetDefinition) GetTime() WidgetTime {
	if o == nil || o.Time == nil {
		var ret WidgetTime
		return ret
	}
	return *o.Time
}

// GetTimeOk returns a tuple with the Time field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *QueryValueWidgetDefinition) GetTimeOk() (*WidgetTime, bool) {
	if o == nil || o.Time == nil {
		return nil, false
	}
	return o.Time, true
}

// HasTime returns a boolean if a field has been set.
func (o *QueryValueWidgetDefinition) HasTime() bool {
	if o != nil && o.Time != nil {
		return true
	}

	return false
}

// SetTime gets a reference to the given WidgetTime and assigns it to the Time field.
func (o *QueryValueWidgetDefinition) SetTime(v WidgetTime) {
	o.Time = &v
}

// GetTimeseriesBackground returns the TimeseriesBackground field value if set, zero value otherwise.
func (o *QueryValueWidgetDefinition) GetTimeseriesBackground() TimeseriesBackground {
	if o == nil || o.TimeseriesBackground == nil {
		var ret TimeseriesBackground
		return ret
	}
	return *o.TimeseriesBackground
}

// GetTimeseriesBackgroundOk returns a tuple with the TimeseriesBackground field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *QueryValueWidgetDefinition) GetTimeseriesBackgroundOk() (*TimeseriesBackground, bool) {
	if o == nil || o.TimeseriesBackground == nil {
		return nil, false
	}
	return o.TimeseriesBackground, true
}

// HasTimeseriesBackground returns a boolean if a field has been set.
func (o *QueryValueWidgetDefinition) HasTimeseriesBackground() bool {
	if o != nil && o.TimeseriesBackground != nil {
		return true
	}

	return false
}

// SetTimeseriesBackground gets a reference to the given TimeseriesBackground and assigns it to the TimeseriesBackground field.
func (o *QueryValueWidgetDefinition) SetTimeseriesBackground(v TimeseriesBackground) {
	o.TimeseriesBackground = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *QueryValueWidgetDefinition) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *QueryValueWidgetDefinition) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *QueryValueWidgetDefinition) HasTitle() bool {
	if o != nil && o.Title != nil {
		return true
	}

	return false
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *QueryValueWidgetDefinition) SetTitle(v string) {
	o.Title = &v
}

// GetTitleAlign returns the TitleAlign field value if set, zero value otherwise.
func (o *QueryValueWidgetDefinition) GetTitleAlign() WidgetTextAlign {
	if o == nil || o.TitleAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TitleAlign
}

// GetTitleAlignOk returns a tuple with the TitleAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *QueryValueWidgetDefinition) GetTitleAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TitleAlign == nil {
		return nil, false
	}
	return o.TitleAlign, true
}

// HasTitleAlign returns a boolean if a field has been set.
func (o *QueryValueWidgetDefinition) HasTitleAlign() bool {
	if o != nil && o.TitleAlign != nil {
		return true
	}

	return false
}

// SetTitleAlign gets a reference to the given WidgetTextAlign and assigns it to the TitleAlign field.
func (o *QueryValueWidgetDefinition) SetTitleAlign(v WidgetTextAlign) {
	o.TitleAlign = &v
}

// GetTitleSize returns the TitleSize field value if set, zero value otherwise.
func (o *QueryValueWidgetDefinition) GetTitleSize() string {
	if o == nil || o.TitleSize == nil {
		var ret string
		return ret
	}
	return *o.TitleSize
}

// GetTitleSizeOk returns a tuple with the TitleSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *QueryValueWidgetDefinition) GetTitleSizeOk() (*string, bool) {
	if o == nil || o.TitleSize == nil {
		return nil, false
	}
	return o.TitleSize, true
}

// HasTitleSize returns a boolean if a field has been set.
func (o *QueryValueWidgetDefinition) HasTitleSize() bool {
	if o != nil && o.TitleSize != nil {
		return true
	}

	return false
}

// SetTitleSize gets a reference to the given string and assigns it to the TitleSize field.
func (o *QueryValueWidgetDefinition) SetTitleSize(v string) {
	o.TitleSize = &v
}

// GetType returns the Type field value.
func (o *QueryValueWidgetDefinition) GetType() QueryValueWidgetDefinitionType {
	if o == nil {
		var ret QueryValueWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *QueryValueWidgetDefinition) GetTypeOk() (*QueryValueWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *QueryValueWidgetDefinition) SetType(v QueryValueWidgetDefinitionType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o QueryValueWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Autoscale != nil {
		toSerialize["autoscale"] = o.Autoscale
	}
	if o.CustomLinks != nil {
		toSerialize["custom_links"] = o.CustomLinks
	}
	if o.CustomUnit != nil {
		toSerialize["custom_unit"] = o.CustomUnit
	}
	if o.Precision != nil {
		toSerialize["precision"] = o.Precision
	}
	toSerialize["requests"] = o.Requests
	if o.TextAlign != nil {
		toSerialize["text_align"] = o.TextAlign
	}
	if o.Time != nil {
		toSerialize["time"] = o.Time
	}
	if o.TimeseriesBackground != nil {
		toSerialize["timeseries_background"] = o.TimeseriesBackground
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
func (o *QueryValueWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Requests *[]QueryValueWidgetRequest      `json:"requests"`
		Type     *QueryValueWidgetDefinitionType `json:"type"`
	}{}
	all := struct {
		Autoscale            *bool                          `json:"autoscale,omitempty"`
		CustomLinks          []WidgetCustomLink             `json:"custom_links,omitempty"`
		CustomUnit           *string                        `json:"custom_unit,omitempty"`
		Precision            *int64                         `json:"precision,omitempty"`
		Requests             []QueryValueWidgetRequest      `json:"requests"`
		TextAlign            *WidgetTextAlign               `json:"text_align,omitempty"`
		Time                 *WidgetTime                    `json:"time,omitempty"`
		TimeseriesBackground *TimeseriesBackground          `json:"timeseries_background,omitempty"`
		Title                *string                        `json:"title,omitempty"`
		TitleAlign           *WidgetTextAlign               `json:"title_align,omitempty"`
		TitleSize            *string                        `json:"title_size,omitempty"`
		Type                 QueryValueWidgetDefinitionType `json:"type"`
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
	if v := all.TextAlign; v != nil && !v.IsValid() {
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
	o.Autoscale = all.Autoscale
	o.CustomLinks = all.CustomLinks
	o.CustomUnit = all.CustomUnit
	o.Precision = all.Precision
	o.Requests = all.Requests
	o.TextAlign = all.TextAlign
	if all.Time != nil && all.Time.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Time = all.Time
	if all.TimeseriesBackground != nil && all.TimeseriesBackground.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.TimeseriesBackground = all.TimeseriesBackground
	o.Title = all.Title
	o.TitleAlign = all.TitleAlign
	o.TitleSize = all.TitleSize
	o.Type = all.Type
	return nil
}
