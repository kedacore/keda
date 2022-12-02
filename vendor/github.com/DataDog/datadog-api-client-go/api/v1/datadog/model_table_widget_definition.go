// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// TableWidgetDefinition The table visualization is available on timeboards and screenboards. It displays columns of metrics grouped by tag key.
type TableWidgetDefinition struct {
	// List of custom links.
	CustomLinks []WidgetCustomLink `json:"custom_links,omitempty"`
	// Controls the display of the search bar.
	HasSearchBar *TableWidgetHasSearchBar `json:"has_search_bar,omitempty"`
	// Widget definition.
	Requests []TableWidgetRequest `json:"requests"`
	// Time setting for the widget.
	Time *WidgetTime `json:"time,omitempty"`
	// Title of your widget.
	Title *string `json:"title,omitempty"`
	// How to align the text on the widget.
	TitleAlign *WidgetTextAlign `json:"title_align,omitempty"`
	// Size of the title.
	TitleSize *string `json:"title_size,omitempty"`
	// Type of the table widget.
	Type TableWidgetDefinitionType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewTableWidgetDefinition instantiates a new TableWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewTableWidgetDefinition(requests []TableWidgetRequest, typeVar TableWidgetDefinitionType) *TableWidgetDefinition {
	this := TableWidgetDefinition{}
	this.Requests = requests
	this.Type = typeVar
	return &this
}

// NewTableWidgetDefinitionWithDefaults instantiates a new TableWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewTableWidgetDefinitionWithDefaults() *TableWidgetDefinition {
	this := TableWidgetDefinition{}
	var typeVar TableWidgetDefinitionType = TABLEWIDGETDEFINITIONTYPE_QUERY_TABLE
	this.Type = typeVar
	return &this
}

// GetCustomLinks returns the CustomLinks field value if set, zero value otherwise.
func (o *TableWidgetDefinition) GetCustomLinks() []WidgetCustomLink {
	if o == nil || o.CustomLinks == nil {
		var ret []WidgetCustomLink
		return ret
	}
	return o.CustomLinks
}

// GetCustomLinksOk returns a tuple with the CustomLinks field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TableWidgetDefinition) GetCustomLinksOk() (*[]WidgetCustomLink, bool) {
	if o == nil || o.CustomLinks == nil {
		return nil, false
	}
	return &o.CustomLinks, true
}

// HasCustomLinks returns a boolean if a field has been set.
func (o *TableWidgetDefinition) HasCustomLinks() bool {
	if o != nil && o.CustomLinks != nil {
		return true
	}

	return false
}

// SetCustomLinks gets a reference to the given []WidgetCustomLink and assigns it to the CustomLinks field.
func (o *TableWidgetDefinition) SetCustomLinks(v []WidgetCustomLink) {
	o.CustomLinks = v
}

// GetHasSearchBar returns the HasSearchBar field value if set, zero value otherwise.
func (o *TableWidgetDefinition) GetHasSearchBar() TableWidgetHasSearchBar {
	if o == nil || o.HasSearchBar == nil {
		var ret TableWidgetHasSearchBar
		return ret
	}
	return *o.HasSearchBar
}

// GetHasSearchBarOk returns a tuple with the HasSearchBar field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TableWidgetDefinition) GetHasSearchBarOk() (*TableWidgetHasSearchBar, bool) {
	if o == nil || o.HasSearchBar == nil {
		return nil, false
	}
	return o.HasSearchBar, true
}

// HasHasSearchBar returns a boolean if a field has been set.
func (o *TableWidgetDefinition) HasHasSearchBar() bool {
	if o != nil && o.HasSearchBar != nil {
		return true
	}

	return false
}

// SetHasSearchBar gets a reference to the given TableWidgetHasSearchBar and assigns it to the HasSearchBar field.
func (o *TableWidgetDefinition) SetHasSearchBar(v TableWidgetHasSearchBar) {
	o.HasSearchBar = &v
}

// GetRequests returns the Requests field value.
func (o *TableWidgetDefinition) GetRequests() []TableWidgetRequest {
	if o == nil {
		var ret []TableWidgetRequest
		return ret
	}
	return o.Requests
}

// GetRequestsOk returns a tuple with the Requests field value
// and a boolean to check if the value has been set.
func (o *TableWidgetDefinition) GetRequestsOk() (*[]TableWidgetRequest, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Requests, true
}

// SetRequests sets field value.
func (o *TableWidgetDefinition) SetRequests(v []TableWidgetRequest) {
	o.Requests = v
}

// GetTime returns the Time field value if set, zero value otherwise.
func (o *TableWidgetDefinition) GetTime() WidgetTime {
	if o == nil || o.Time == nil {
		var ret WidgetTime
		return ret
	}
	return *o.Time
}

// GetTimeOk returns a tuple with the Time field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TableWidgetDefinition) GetTimeOk() (*WidgetTime, bool) {
	if o == nil || o.Time == nil {
		return nil, false
	}
	return o.Time, true
}

// HasTime returns a boolean if a field has been set.
func (o *TableWidgetDefinition) HasTime() bool {
	if o != nil && o.Time != nil {
		return true
	}

	return false
}

// SetTime gets a reference to the given WidgetTime and assigns it to the Time field.
func (o *TableWidgetDefinition) SetTime(v WidgetTime) {
	o.Time = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *TableWidgetDefinition) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TableWidgetDefinition) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *TableWidgetDefinition) HasTitle() bool {
	if o != nil && o.Title != nil {
		return true
	}

	return false
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *TableWidgetDefinition) SetTitle(v string) {
	o.Title = &v
}

// GetTitleAlign returns the TitleAlign field value if set, zero value otherwise.
func (o *TableWidgetDefinition) GetTitleAlign() WidgetTextAlign {
	if o == nil || o.TitleAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TitleAlign
}

// GetTitleAlignOk returns a tuple with the TitleAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TableWidgetDefinition) GetTitleAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TitleAlign == nil {
		return nil, false
	}
	return o.TitleAlign, true
}

// HasTitleAlign returns a boolean if a field has been set.
func (o *TableWidgetDefinition) HasTitleAlign() bool {
	if o != nil && o.TitleAlign != nil {
		return true
	}

	return false
}

// SetTitleAlign gets a reference to the given WidgetTextAlign and assigns it to the TitleAlign field.
func (o *TableWidgetDefinition) SetTitleAlign(v WidgetTextAlign) {
	o.TitleAlign = &v
}

// GetTitleSize returns the TitleSize field value if set, zero value otherwise.
func (o *TableWidgetDefinition) GetTitleSize() string {
	if o == nil || o.TitleSize == nil {
		var ret string
		return ret
	}
	return *o.TitleSize
}

// GetTitleSizeOk returns a tuple with the TitleSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TableWidgetDefinition) GetTitleSizeOk() (*string, bool) {
	if o == nil || o.TitleSize == nil {
		return nil, false
	}
	return o.TitleSize, true
}

// HasTitleSize returns a boolean if a field has been set.
func (o *TableWidgetDefinition) HasTitleSize() bool {
	if o != nil && o.TitleSize != nil {
		return true
	}

	return false
}

// SetTitleSize gets a reference to the given string and assigns it to the TitleSize field.
func (o *TableWidgetDefinition) SetTitleSize(v string) {
	o.TitleSize = &v
}

// GetType returns the Type field value.
func (o *TableWidgetDefinition) GetType() TableWidgetDefinitionType {
	if o == nil {
		var ret TableWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *TableWidgetDefinition) GetTypeOk() (*TableWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *TableWidgetDefinition) SetType(v TableWidgetDefinitionType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o TableWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CustomLinks != nil {
		toSerialize["custom_links"] = o.CustomLinks
	}
	if o.HasSearchBar != nil {
		toSerialize["has_search_bar"] = o.HasSearchBar
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
func (o *TableWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Requests *[]TableWidgetRequest      `json:"requests"`
		Type     *TableWidgetDefinitionType `json:"type"`
	}{}
	all := struct {
		CustomLinks  []WidgetCustomLink        `json:"custom_links,omitempty"`
		HasSearchBar *TableWidgetHasSearchBar  `json:"has_search_bar,omitempty"`
		Requests     []TableWidgetRequest      `json:"requests"`
		Time         *WidgetTime               `json:"time,omitempty"`
		Title        *string                   `json:"title,omitempty"`
		TitleAlign   *WidgetTextAlign          `json:"title_align,omitempty"`
		TitleSize    *string                   `json:"title_size,omitempty"`
		Type         TableWidgetDefinitionType `json:"type"`
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
	if v := all.HasSearchBar; v != nil && !v.IsValid() {
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
	o.HasSearchBar = all.HasSearchBar
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
