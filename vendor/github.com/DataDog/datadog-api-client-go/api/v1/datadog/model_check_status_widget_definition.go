// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// CheckStatusWidgetDefinition Check status shows the current status or number of results for any check performed.
type CheckStatusWidgetDefinition struct {
	// Name of the check to use in the widget.
	Check string `json:"check"`
	// Group reporting a single check.
	Group *string `json:"group,omitempty"`
	// List of tag prefixes to group by in the case of a cluster check.
	GroupBy []string `json:"group_by,omitempty"`
	// The kind of grouping to use.
	Grouping WidgetGrouping `json:"grouping"`
	// List of tags used to filter the groups reporting a cluster check.
	Tags []string `json:"tags,omitempty"`
	// Time setting for the widget.
	Time *WidgetTime `json:"time,omitempty"`
	// Title of the widget.
	Title *string `json:"title,omitempty"`
	// How to align the text on the widget.
	TitleAlign *WidgetTextAlign `json:"title_align,omitempty"`
	// Size of the title.
	TitleSize *string `json:"title_size,omitempty"`
	// Type of the check status widget.
	Type CheckStatusWidgetDefinitionType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewCheckStatusWidgetDefinition instantiates a new CheckStatusWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCheckStatusWidgetDefinition(check string, grouping WidgetGrouping, typeVar CheckStatusWidgetDefinitionType) *CheckStatusWidgetDefinition {
	this := CheckStatusWidgetDefinition{}
	this.Check = check
	this.Grouping = grouping
	this.Type = typeVar
	return &this
}

// NewCheckStatusWidgetDefinitionWithDefaults instantiates a new CheckStatusWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCheckStatusWidgetDefinitionWithDefaults() *CheckStatusWidgetDefinition {
	this := CheckStatusWidgetDefinition{}
	var typeVar CheckStatusWidgetDefinitionType = CHECKSTATUSWIDGETDEFINITIONTYPE_CHECK_STATUS
	this.Type = typeVar
	return &this
}

// GetCheck returns the Check field value.
func (o *CheckStatusWidgetDefinition) GetCheck() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Check
}

// GetCheckOk returns a tuple with the Check field value
// and a boolean to check if the value has been set.
func (o *CheckStatusWidgetDefinition) GetCheckOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Check, true
}

// SetCheck sets field value.
func (o *CheckStatusWidgetDefinition) SetCheck(v string) {
	o.Check = v
}

// GetGroup returns the Group field value if set, zero value otherwise.
func (o *CheckStatusWidgetDefinition) GetGroup() string {
	if o == nil || o.Group == nil {
		var ret string
		return ret
	}
	return *o.Group
}

// GetGroupOk returns a tuple with the Group field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CheckStatusWidgetDefinition) GetGroupOk() (*string, bool) {
	if o == nil || o.Group == nil {
		return nil, false
	}
	return o.Group, true
}

// HasGroup returns a boolean if a field has been set.
func (o *CheckStatusWidgetDefinition) HasGroup() bool {
	if o != nil && o.Group != nil {
		return true
	}

	return false
}

// SetGroup gets a reference to the given string and assigns it to the Group field.
func (o *CheckStatusWidgetDefinition) SetGroup(v string) {
	o.Group = &v
}

// GetGroupBy returns the GroupBy field value if set, zero value otherwise.
func (o *CheckStatusWidgetDefinition) GetGroupBy() []string {
	if o == nil || o.GroupBy == nil {
		var ret []string
		return ret
	}
	return o.GroupBy
}

// GetGroupByOk returns a tuple with the GroupBy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CheckStatusWidgetDefinition) GetGroupByOk() (*[]string, bool) {
	if o == nil || o.GroupBy == nil {
		return nil, false
	}
	return &o.GroupBy, true
}

// HasGroupBy returns a boolean if a field has been set.
func (o *CheckStatusWidgetDefinition) HasGroupBy() bool {
	if o != nil && o.GroupBy != nil {
		return true
	}

	return false
}

// SetGroupBy gets a reference to the given []string and assigns it to the GroupBy field.
func (o *CheckStatusWidgetDefinition) SetGroupBy(v []string) {
	o.GroupBy = v
}

// GetGrouping returns the Grouping field value.
func (o *CheckStatusWidgetDefinition) GetGrouping() WidgetGrouping {
	if o == nil {
		var ret WidgetGrouping
		return ret
	}
	return o.Grouping
}

// GetGroupingOk returns a tuple with the Grouping field value
// and a boolean to check if the value has been set.
func (o *CheckStatusWidgetDefinition) GetGroupingOk() (*WidgetGrouping, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Grouping, true
}

// SetGrouping sets field value.
func (o *CheckStatusWidgetDefinition) SetGrouping(v WidgetGrouping) {
	o.Grouping = v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *CheckStatusWidgetDefinition) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CheckStatusWidgetDefinition) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *CheckStatusWidgetDefinition) HasTags() bool {
	if o != nil && o.Tags != nil {
		return true
	}

	return false
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *CheckStatusWidgetDefinition) SetTags(v []string) {
	o.Tags = v
}

// GetTime returns the Time field value if set, zero value otherwise.
func (o *CheckStatusWidgetDefinition) GetTime() WidgetTime {
	if o == nil || o.Time == nil {
		var ret WidgetTime
		return ret
	}
	return *o.Time
}

// GetTimeOk returns a tuple with the Time field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CheckStatusWidgetDefinition) GetTimeOk() (*WidgetTime, bool) {
	if o == nil || o.Time == nil {
		return nil, false
	}
	return o.Time, true
}

// HasTime returns a boolean if a field has been set.
func (o *CheckStatusWidgetDefinition) HasTime() bool {
	if o != nil && o.Time != nil {
		return true
	}

	return false
}

// SetTime gets a reference to the given WidgetTime and assigns it to the Time field.
func (o *CheckStatusWidgetDefinition) SetTime(v WidgetTime) {
	o.Time = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *CheckStatusWidgetDefinition) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CheckStatusWidgetDefinition) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *CheckStatusWidgetDefinition) HasTitle() bool {
	if o != nil && o.Title != nil {
		return true
	}

	return false
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *CheckStatusWidgetDefinition) SetTitle(v string) {
	o.Title = &v
}

// GetTitleAlign returns the TitleAlign field value if set, zero value otherwise.
func (o *CheckStatusWidgetDefinition) GetTitleAlign() WidgetTextAlign {
	if o == nil || o.TitleAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TitleAlign
}

// GetTitleAlignOk returns a tuple with the TitleAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CheckStatusWidgetDefinition) GetTitleAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TitleAlign == nil {
		return nil, false
	}
	return o.TitleAlign, true
}

// HasTitleAlign returns a boolean if a field has been set.
func (o *CheckStatusWidgetDefinition) HasTitleAlign() bool {
	if o != nil && o.TitleAlign != nil {
		return true
	}

	return false
}

// SetTitleAlign gets a reference to the given WidgetTextAlign and assigns it to the TitleAlign field.
func (o *CheckStatusWidgetDefinition) SetTitleAlign(v WidgetTextAlign) {
	o.TitleAlign = &v
}

// GetTitleSize returns the TitleSize field value if set, zero value otherwise.
func (o *CheckStatusWidgetDefinition) GetTitleSize() string {
	if o == nil || o.TitleSize == nil {
		var ret string
		return ret
	}
	return *o.TitleSize
}

// GetTitleSizeOk returns a tuple with the TitleSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CheckStatusWidgetDefinition) GetTitleSizeOk() (*string, bool) {
	if o == nil || o.TitleSize == nil {
		return nil, false
	}
	return o.TitleSize, true
}

// HasTitleSize returns a boolean if a field has been set.
func (o *CheckStatusWidgetDefinition) HasTitleSize() bool {
	if o != nil && o.TitleSize != nil {
		return true
	}

	return false
}

// SetTitleSize gets a reference to the given string and assigns it to the TitleSize field.
func (o *CheckStatusWidgetDefinition) SetTitleSize(v string) {
	o.TitleSize = &v
}

// GetType returns the Type field value.
func (o *CheckStatusWidgetDefinition) GetType() CheckStatusWidgetDefinitionType {
	if o == nil {
		var ret CheckStatusWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *CheckStatusWidgetDefinition) GetTypeOk() (*CheckStatusWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *CheckStatusWidgetDefinition) SetType(v CheckStatusWidgetDefinitionType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o CheckStatusWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["check"] = o.Check
	if o.Group != nil {
		toSerialize["group"] = o.Group
	}
	if o.GroupBy != nil {
		toSerialize["group_by"] = o.GroupBy
	}
	toSerialize["grouping"] = o.Grouping
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
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
func (o *CheckStatusWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Check    *string                          `json:"check"`
		Grouping *WidgetGrouping                  `json:"grouping"`
		Type     *CheckStatusWidgetDefinitionType `json:"type"`
	}{}
	all := struct {
		Check      string                          `json:"check"`
		Group      *string                         `json:"group,omitempty"`
		GroupBy    []string                        `json:"group_by,omitempty"`
		Grouping   WidgetGrouping                  `json:"grouping"`
		Tags       []string                        `json:"tags,omitempty"`
		Time       *WidgetTime                     `json:"time,omitempty"`
		Title      *string                         `json:"title,omitempty"`
		TitleAlign *WidgetTextAlign                `json:"title_align,omitempty"`
		TitleSize  *string                         `json:"title_size,omitempty"`
		Type       CheckStatusWidgetDefinitionType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Check == nil {
		return fmt.Errorf("Required field check missing")
	}
	if required.Grouping == nil {
		return fmt.Errorf("Required field grouping missing")
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
	if v := all.Grouping; !v.IsValid() {
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
	o.Check = all.Check
	o.Group = all.Group
	o.GroupBy = all.GroupBy
	o.Grouping = all.Grouping
	o.Tags = all.Tags
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
