// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// EventTimelineWidgetDefinition The event timeline is a widget version of the timeline that appears at the top of the Event Stream view. Only available on FREE layout dashboards.
type EventTimelineWidgetDefinition struct {
	// Query to filter the event timeline with.
	Query string `json:"query"`
	// The execution method for multi-value filters. Can be either and or or.
	TagsExecution *string `json:"tags_execution,omitempty"`
	// Time setting for the widget.
	Time *WidgetTime `json:"time,omitempty"`
	// Title of the widget.
	Title *string `json:"title,omitempty"`
	// How to align the text on the widget.
	TitleAlign *WidgetTextAlign `json:"title_align,omitempty"`
	// Size of the title.
	TitleSize *string `json:"title_size,omitempty"`
	// Type of the event timeline widget.
	Type EventTimelineWidgetDefinitionType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewEventTimelineWidgetDefinition instantiates a new EventTimelineWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewEventTimelineWidgetDefinition(query string, typeVar EventTimelineWidgetDefinitionType) *EventTimelineWidgetDefinition {
	this := EventTimelineWidgetDefinition{}
	this.Query = query
	this.Type = typeVar
	return &this
}

// NewEventTimelineWidgetDefinitionWithDefaults instantiates a new EventTimelineWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewEventTimelineWidgetDefinitionWithDefaults() *EventTimelineWidgetDefinition {
	this := EventTimelineWidgetDefinition{}
	var typeVar EventTimelineWidgetDefinitionType = EVENTTIMELINEWIDGETDEFINITIONTYPE_EVENT_TIMELINE
	this.Type = typeVar
	return &this
}

// GetQuery returns the Query field value.
func (o *EventTimelineWidgetDefinition) GetQuery() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Query
}

// GetQueryOk returns a tuple with the Query field value
// and a boolean to check if the value has been set.
func (o *EventTimelineWidgetDefinition) GetQueryOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Query, true
}

// SetQuery sets field value.
func (o *EventTimelineWidgetDefinition) SetQuery(v string) {
	o.Query = v
}

// GetTagsExecution returns the TagsExecution field value if set, zero value otherwise.
func (o *EventTimelineWidgetDefinition) GetTagsExecution() string {
	if o == nil || o.TagsExecution == nil {
		var ret string
		return ret
	}
	return *o.TagsExecution
}

// GetTagsExecutionOk returns a tuple with the TagsExecution field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventTimelineWidgetDefinition) GetTagsExecutionOk() (*string, bool) {
	if o == nil || o.TagsExecution == nil {
		return nil, false
	}
	return o.TagsExecution, true
}

// HasTagsExecution returns a boolean if a field has been set.
func (o *EventTimelineWidgetDefinition) HasTagsExecution() bool {
	if o != nil && o.TagsExecution != nil {
		return true
	}

	return false
}

// SetTagsExecution gets a reference to the given string and assigns it to the TagsExecution field.
func (o *EventTimelineWidgetDefinition) SetTagsExecution(v string) {
	o.TagsExecution = &v
}

// GetTime returns the Time field value if set, zero value otherwise.
func (o *EventTimelineWidgetDefinition) GetTime() WidgetTime {
	if o == nil || o.Time == nil {
		var ret WidgetTime
		return ret
	}
	return *o.Time
}

// GetTimeOk returns a tuple with the Time field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventTimelineWidgetDefinition) GetTimeOk() (*WidgetTime, bool) {
	if o == nil || o.Time == nil {
		return nil, false
	}
	return o.Time, true
}

// HasTime returns a boolean if a field has been set.
func (o *EventTimelineWidgetDefinition) HasTime() bool {
	if o != nil && o.Time != nil {
		return true
	}

	return false
}

// SetTime gets a reference to the given WidgetTime and assigns it to the Time field.
func (o *EventTimelineWidgetDefinition) SetTime(v WidgetTime) {
	o.Time = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *EventTimelineWidgetDefinition) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventTimelineWidgetDefinition) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *EventTimelineWidgetDefinition) HasTitle() bool {
	if o != nil && o.Title != nil {
		return true
	}

	return false
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *EventTimelineWidgetDefinition) SetTitle(v string) {
	o.Title = &v
}

// GetTitleAlign returns the TitleAlign field value if set, zero value otherwise.
func (o *EventTimelineWidgetDefinition) GetTitleAlign() WidgetTextAlign {
	if o == nil || o.TitleAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TitleAlign
}

// GetTitleAlignOk returns a tuple with the TitleAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventTimelineWidgetDefinition) GetTitleAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TitleAlign == nil {
		return nil, false
	}
	return o.TitleAlign, true
}

// HasTitleAlign returns a boolean if a field has been set.
func (o *EventTimelineWidgetDefinition) HasTitleAlign() bool {
	if o != nil && o.TitleAlign != nil {
		return true
	}

	return false
}

// SetTitleAlign gets a reference to the given WidgetTextAlign and assigns it to the TitleAlign field.
func (o *EventTimelineWidgetDefinition) SetTitleAlign(v WidgetTextAlign) {
	o.TitleAlign = &v
}

// GetTitleSize returns the TitleSize field value if set, zero value otherwise.
func (o *EventTimelineWidgetDefinition) GetTitleSize() string {
	if o == nil || o.TitleSize == nil {
		var ret string
		return ret
	}
	return *o.TitleSize
}

// GetTitleSizeOk returns a tuple with the TitleSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventTimelineWidgetDefinition) GetTitleSizeOk() (*string, bool) {
	if o == nil || o.TitleSize == nil {
		return nil, false
	}
	return o.TitleSize, true
}

// HasTitleSize returns a boolean if a field has been set.
func (o *EventTimelineWidgetDefinition) HasTitleSize() bool {
	if o != nil && o.TitleSize != nil {
		return true
	}

	return false
}

// SetTitleSize gets a reference to the given string and assigns it to the TitleSize field.
func (o *EventTimelineWidgetDefinition) SetTitleSize(v string) {
	o.TitleSize = &v
}

// GetType returns the Type field value.
func (o *EventTimelineWidgetDefinition) GetType() EventTimelineWidgetDefinitionType {
	if o == nil {
		var ret EventTimelineWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *EventTimelineWidgetDefinition) GetTypeOk() (*EventTimelineWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *EventTimelineWidgetDefinition) SetType(v EventTimelineWidgetDefinitionType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o EventTimelineWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["query"] = o.Query
	if o.TagsExecution != nil {
		toSerialize["tags_execution"] = o.TagsExecution
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
func (o *EventTimelineWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Query *string                            `json:"query"`
		Type  *EventTimelineWidgetDefinitionType `json:"type"`
	}{}
	all := struct {
		Query         string                            `json:"query"`
		TagsExecution *string                           `json:"tags_execution,omitempty"`
		Time          *WidgetTime                       `json:"time,omitempty"`
		Title         *string                           `json:"title,omitempty"`
		TitleAlign    *WidgetTextAlign                  `json:"title_align,omitempty"`
		TitleSize     *string                           `json:"title_size,omitempty"`
		Type          EventTimelineWidgetDefinitionType `json:"type"`
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
	o.Query = all.Query
	o.TagsExecution = all.TagsExecution
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
