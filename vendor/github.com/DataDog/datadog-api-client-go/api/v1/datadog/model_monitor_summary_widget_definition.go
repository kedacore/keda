// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// MonitorSummaryWidgetDefinition The monitor summary widget displays a summary view of all your Datadog monitors, or a subset based on a query. Only available on FREE layout dashboards.
type MonitorSummaryWidgetDefinition struct {
	// Which color to use on the widget.
	ColorPreference *WidgetColorPreference `json:"color_preference,omitempty"`
	// The number of monitors to display.
	// Deprecated
	Count *int64 `json:"count,omitempty"`
	// What to display on the widget.
	DisplayFormat *WidgetMonitorSummaryDisplayFormat `json:"display_format,omitempty"`
	// Whether to show counts of 0 or not.
	HideZeroCounts *bool `json:"hide_zero_counts,omitempty"`
	// Query to filter the monitors with.
	Query string `json:"query"`
	// Whether to show the time that has elapsed since the monitor/group triggered.
	ShowLastTriggered *bool `json:"show_last_triggered,omitempty"`
	// Widget sorting methods.
	Sort *WidgetMonitorSummarySort `json:"sort,omitempty"`
	// The start of the list. Typically 0.
	// Deprecated
	Start *int64 `json:"start,omitempty"`
	// Which summary type should be used.
	SummaryType *WidgetSummaryType `json:"summary_type,omitempty"`
	// Title of the widget.
	Title *string `json:"title,omitempty"`
	// How to align the text on the widget.
	TitleAlign *WidgetTextAlign `json:"title_align,omitempty"`
	// Size of the title.
	TitleSize *string `json:"title_size,omitempty"`
	// Type of the monitor summary widget.
	Type MonitorSummaryWidgetDefinitionType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMonitorSummaryWidgetDefinition instantiates a new MonitorSummaryWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorSummaryWidgetDefinition(query string, typeVar MonitorSummaryWidgetDefinitionType) *MonitorSummaryWidgetDefinition {
	this := MonitorSummaryWidgetDefinition{}
	this.Query = query
	this.Type = typeVar
	return &this
}

// NewMonitorSummaryWidgetDefinitionWithDefaults instantiates a new MonitorSummaryWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorSummaryWidgetDefinitionWithDefaults() *MonitorSummaryWidgetDefinition {
	this := MonitorSummaryWidgetDefinition{}
	var typeVar MonitorSummaryWidgetDefinitionType = MONITORSUMMARYWIDGETDEFINITIONTYPE_MANAGE_STATUS
	this.Type = typeVar
	return &this
}

// GetColorPreference returns the ColorPreference field value if set, zero value otherwise.
func (o *MonitorSummaryWidgetDefinition) GetColorPreference() WidgetColorPreference {
	if o == nil || o.ColorPreference == nil {
		var ret WidgetColorPreference
		return ret
	}
	return *o.ColorPreference
}

// GetColorPreferenceOk returns a tuple with the ColorPreference field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSummaryWidgetDefinition) GetColorPreferenceOk() (*WidgetColorPreference, bool) {
	if o == nil || o.ColorPreference == nil {
		return nil, false
	}
	return o.ColorPreference, true
}

// HasColorPreference returns a boolean if a field has been set.
func (o *MonitorSummaryWidgetDefinition) HasColorPreference() bool {
	if o != nil && o.ColorPreference != nil {
		return true
	}

	return false
}

// SetColorPreference gets a reference to the given WidgetColorPreference and assigns it to the ColorPreference field.
func (o *MonitorSummaryWidgetDefinition) SetColorPreference(v WidgetColorPreference) {
	o.ColorPreference = &v
}

// GetCount returns the Count field value if set, zero value otherwise.
// Deprecated
func (o *MonitorSummaryWidgetDefinition) GetCount() int64 {
	if o == nil || o.Count == nil {
		var ret int64
		return ret
	}
	return *o.Count
}

// GetCountOk returns a tuple with the Count field value if set, nil otherwise
// and a boolean to check if the value has been set.
// Deprecated
func (o *MonitorSummaryWidgetDefinition) GetCountOk() (*int64, bool) {
	if o == nil || o.Count == nil {
		return nil, false
	}
	return o.Count, true
}

// HasCount returns a boolean if a field has been set.
func (o *MonitorSummaryWidgetDefinition) HasCount() bool {
	if o != nil && o.Count != nil {
		return true
	}

	return false
}

// SetCount gets a reference to the given int64 and assigns it to the Count field.
// Deprecated
func (o *MonitorSummaryWidgetDefinition) SetCount(v int64) {
	o.Count = &v
}

// GetDisplayFormat returns the DisplayFormat field value if set, zero value otherwise.
func (o *MonitorSummaryWidgetDefinition) GetDisplayFormat() WidgetMonitorSummaryDisplayFormat {
	if o == nil || o.DisplayFormat == nil {
		var ret WidgetMonitorSummaryDisplayFormat
		return ret
	}
	return *o.DisplayFormat
}

// GetDisplayFormatOk returns a tuple with the DisplayFormat field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSummaryWidgetDefinition) GetDisplayFormatOk() (*WidgetMonitorSummaryDisplayFormat, bool) {
	if o == nil || o.DisplayFormat == nil {
		return nil, false
	}
	return o.DisplayFormat, true
}

// HasDisplayFormat returns a boolean if a field has been set.
func (o *MonitorSummaryWidgetDefinition) HasDisplayFormat() bool {
	if o != nil && o.DisplayFormat != nil {
		return true
	}

	return false
}

// SetDisplayFormat gets a reference to the given WidgetMonitorSummaryDisplayFormat and assigns it to the DisplayFormat field.
func (o *MonitorSummaryWidgetDefinition) SetDisplayFormat(v WidgetMonitorSummaryDisplayFormat) {
	o.DisplayFormat = &v
}

// GetHideZeroCounts returns the HideZeroCounts field value if set, zero value otherwise.
func (o *MonitorSummaryWidgetDefinition) GetHideZeroCounts() bool {
	if o == nil || o.HideZeroCounts == nil {
		var ret bool
		return ret
	}
	return *o.HideZeroCounts
}

// GetHideZeroCountsOk returns a tuple with the HideZeroCounts field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSummaryWidgetDefinition) GetHideZeroCountsOk() (*bool, bool) {
	if o == nil || o.HideZeroCounts == nil {
		return nil, false
	}
	return o.HideZeroCounts, true
}

// HasHideZeroCounts returns a boolean if a field has been set.
func (o *MonitorSummaryWidgetDefinition) HasHideZeroCounts() bool {
	if o != nil && o.HideZeroCounts != nil {
		return true
	}

	return false
}

// SetHideZeroCounts gets a reference to the given bool and assigns it to the HideZeroCounts field.
func (o *MonitorSummaryWidgetDefinition) SetHideZeroCounts(v bool) {
	o.HideZeroCounts = &v
}

// GetQuery returns the Query field value.
func (o *MonitorSummaryWidgetDefinition) GetQuery() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Query
}

// GetQueryOk returns a tuple with the Query field value
// and a boolean to check if the value has been set.
func (o *MonitorSummaryWidgetDefinition) GetQueryOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Query, true
}

// SetQuery sets field value.
func (o *MonitorSummaryWidgetDefinition) SetQuery(v string) {
	o.Query = v
}

// GetShowLastTriggered returns the ShowLastTriggered field value if set, zero value otherwise.
func (o *MonitorSummaryWidgetDefinition) GetShowLastTriggered() bool {
	if o == nil || o.ShowLastTriggered == nil {
		var ret bool
		return ret
	}
	return *o.ShowLastTriggered
}

// GetShowLastTriggeredOk returns a tuple with the ShowLastTriggered field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSummaryWidgetDefinition) GetShowLastTriggeredOk() (*bool, bool) {
	if o == nil || o.ShowLastTriggered == nil {
		return nil, false
	}
	return o.ShowLastTriggered, true
}

// HasShowLastTriggered returns a boolean if a field has been set.
func (o *MonitorSummaryWidgetDefinition) HasShowLastTriggered() bool {
	if o != nil && o.ShowLastTriggered != nil {
		return true
	}

	return false
}

// SetShowLastTriggered gets a reference to the given bool and assigns it to the ShowLastTriggered field.
func (o *MonitorSummaryWidgetDefinition) SetShowLastTriggered(v bool) {
	o.ShowLastTriggered = &v
}

// GetSort returns the Sort field value if set, zero value otherwise.
func (o *MonitorSummaryWidgetDefinition) GetSort() WidgetMonitorSummarySort {
	if o == nil || o.Sort == nil {
		var ret WidgetMonitorSummarySort
		return ret
	}
	return *o.Sort
}

// GetSortOk returns a tuple with the Sort field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSummaryWidgetDefinition) GetSortOk() (*WidgetMonitorSummarySort, bool) {
	if o == nil || o.Sort == nil {
		return nil, false
	}
	return o.Sort, true
}

// HasSort returns a boolean if a field has been set.
func (o *MonitorSummaryWidgetDefinition) HasSort() bool {
	if o != nil && o.Sort != nil {
		return true
	}

	return false
}

// SetSort gets a reference to the given WidgetMonitorSummarySort and assigns it to the Sort field.
func (o *MonitorSummaryWidgetDefinition) SetSort(v WidgetMonitorSummarySort) {
	o.Sort = &v
}

// GetStart returns the Start field value if set, zero value otherwise.
// Deprecated
func (o *MonitorSummaryWidgetDefinition) GetStart() int64 {
	if o == nil || o.Start == nil {
		var ret int64
		return ret
	}
	return *o.Start
}

// GetStartOk returns a tuple with the Start field value if set, nil otherwise
// and a boolean to check if the value has been set.
// Deprecated
func (o *MonitorSummaryWidgetDefinition) GetStartOk() (*int64, bool) {
	if o == nil || o.Start == nil {
		return nil, false
	}
	return o.Start, true
}

// HasStart returns a boolean if a field has been set.
func (o *MonitorSummaryWidgetDefinition) HasStart() bool {
	if o != nil && o.Start != nil {
		return true
	}

	return false
}

// SetStart gets a reference to the given int64 and assigns it to the Start field.
// Deprecated
func (o *MonitorSummaryWidgetDefinition) SetStart(v int64) {
	o.Start = &v
}

// GetSummaryType returns the SummaryType field value if set, zero value otherwise.
func (o *MonitorSummaryWidgetDefinition) GetSummaryType() WidgetSummaryType {
	if o == nil || o.SummaryType == nil {
		var ret WidgetSummaryType
		return ret
	}
	return *o.SummaryType
}

// GetSummaryTypeOk returns a tuple with the SummaryType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSummaryWidgetDefinition) GetSummaryTypeOk() (*WidgetSummaryType, bool) {
	if o == nil || o.SummaryType == nil {
		return nil, false
	}
	return o.SummaryType, true
}

// HasSummaryType returns a boolean if a field has been set.
func (o *MonitorSummaryWidgetDefinition) HasSummaryType() bool {
	if o != nil && o.SummaryType != nil {
		return true
	}

	return false
}

// SetSummaryType gets a reference to the given WidgetSummaryType and assigns it to the SummaryType field.
func (o *MonitorSummaryWidgetDefinition) SetSummaryType(v WidgetSummaryType) {
	o.SummaryType = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *MonitorSummaryWidgetDefinition) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSummaryWidgetDefinition) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *MonitorSummaryWidgetDefinition) HasTitle() bool {
	if o != nil && o.Title != nil {
		return true
	}

	return false
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *MonitorSummaryWidgetDefinition) SetTitle(v string) {
	o.Title = &v
}

// GetTitleAlign returns the TitleAlign field value if set, zero value otherwise.
func (o *MonitorSummaryWidgetDefinition) GetTitleAlign() WidgetTextAlign {
	if o == nil || o.TitleAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TitleAlign
}

// GetTitleAlignOk returns a tuple with the TitleAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSummaryWidgetDefinition) GetTitleAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TitleAlign == nil {
		return nil, false
	}
	return o.TitleAlign, true
}

// HasTitleAlign returns a boolean if a field has been set.
func (o *MonitorSummaryWidgetDefinition) HasTitleAlign() bool {
	if o != nil && o.TitleAlign != nil {
		return true
	}

	return false
}

// SetTitleAlign gets a reference to the given WidgetTextAlign and assigns it to the TitleAlign field.
func (o *MonitorSummaryWidgetDefinition) SetTitleAlign(v WidgetTextAlign) {
	o.TitleAlign = &v
}

// GetTitleSize returns the TitleSize field value if set, zero value otherwise.
func (o *MonitorSummaryWidgetDefinition) GetTitleSize() string {
	if o == nil || o.TitleSize == nil {
		var ret string
		return ret
	}
	return *o.TitleSize
}

// GetTitleSizeOk returns a tuple with the TitleSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSummaryWidgetDefinition) GetTitleSizeOk() (*string, bool) {
	if o == nil || o.TitleSize == nil {
		return nil, false
	}
	return o.TitleSize, true
}

// HasTitleSize returns a boolean if a field has been set.
func (o *MonitorSummaryWidgetDefinition) HasTitleSize() bool {
	if o != nil && o.TitleSize != nil {
		return true
	}

	return false
}

// SetTitleSize gets a reference to the given string and assigns it to the TitleSize field.
func (o *MonitorSummaryWidgetDefinition) SetTitleSize(v string) {
	o.TitleSize = &v
}

// GetType returns the Type field value.
func (o *MonitorSummaryWidgetDefinition) GetType() MonitorSummaryWidgetDefinitionType {
	if o == nil {
		var ret MonitorSummaryWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *MonitorSummaryWidgetDefinition) GetTypeOk() (*MonitorSummaryWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *MonitorSummaryWidgetDefinition) SetType(v MonitorSummaryWidgetDefinitionType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorSummaryWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ColorPreference != nil {
		toSerialize["color_preference"] = o.ColorPreference
	}
	if o.Count != nil {
		toSerialize["count"] = o.Count
	}
	if o.DisplayFormat != nil {
		toSerialize["display_format"] = o.DisplayFormat
	}
	if o.HideZeroCounts != nil {
		toSerialize["hide_zero_counts"] = o.HideZeroCounts
	}
	toSerialize["query"] = o.Query
	if o.ShowLastTriggered != nil {
		toSerialize["show_last_triggered"] = o.ShowLastTriggered
	}
	if o.Sort != nil {
		toSerialize["sort"] = o.Sort
	}
	if o.Start != nil {
		toSerialize["start"] = o.Start
	}
	if o.SummaryType != nil {
		toSerialize["summary_type"] = o.SummaryType
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
func (o *MonitorSummaryWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Query *string                             `json:"query"`
		Type  *MonitorSummaryWidgetDefinitionType `json:"type"`
	}{}
	all := struct {
		ColorPreference   *WidgetColorPreference             `json:"color_preference,omitempty"`
		Count             *int64                             `json:"count,omitempty"`
		DisplayFormat     *WidgetMonitorSummaryDisplayFormat `json:"display_format,omitempty"`
		HideZeroCounts    *bool                              `json:"hide_zero_counts,omitempty"`
		Query             string                             `json:"query"`
		ShowLastTriggered *bool                              `json:"show_last_triggered,omitempty"`
		Sort              *WidgetMonitorSummarySort          `json:"sort,omitempty"`
		Start             *int64                             `json:"start,omitempty"`
		SummaryType       *WidgetSummaryType                 `json:"summary_type,omitempty"`
		Title             *string                            `json:"title,omitempty"`
		TitleAlign        *WidgetTextAlign                   `json:"title_align,omitempty"`
		TitleSize         *string                            `json:"title_size,omitempty"`
		Type              MonitorSummaryWidgetDefinitionType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Query == nil {
		return fmt.Errorf("Required field query missing")
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
	if v := all.ColorPreference; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.DisplayFormat; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Sort; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.SummaryType; v != nil && !v.IsValid() {
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
	o.ColorPreference = all.ColorPreference
	o.Count = all.Count
	o.DisplayFormat = all.DisplayFormat
	o.HideZeroCounts = all.HideZeroCounts
	o.Query = all.Query
	o.ShowLastTriggered = all.ShowLastTriggered
	o.Sort = all.Sort
	o.Start = all.Start
	o.SummaryType = all.SummaryType
	o.Title = all.Title
	o.TitleAlign = all.TitleAlign
	o.TitleSize = all.TitleSize
	o.Type = all.Type
	return nil
}
