// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogStreamWidgetDefinition The Log Stream displays a log flow matching the defined query. Only available on FREE layout dashboards.
type LogStreamWidgetDefinition struct {
	// Which columns to display on the widget.
	Columns []string `json:"columns,omitempty"`
	// An array of index names to query in the stream. Use [] to query all indexes at once.
	Indexes []string `json:"indexes,omitempty"`
	// ID of the log set to use.
	// Deprecated
	Logset *string `json:"logset,omitempty"`
	// Amount of log lines to display
	MessageDisplay *WidgetMessageDisplay `json:"message_display,omitempty"`
	// Query to filter the log stream with.
	Query *string `json:"query,omitempty"`
	// Whether to show the date column or not
	ShowDateColumn *bool `json:"show_date_column,omitempty"`
	// Whether to show the message column or not
	ShowMessageColumn *bool `json:"show_message_column,omitempty"`
	// Which column and order to sort by
	Sort *WidgetFieldSort `json:"sort,omitempty"`
	// Time setting for the widget.
	Time *WidgetTime `json:"time,omitempty"`
	// Title of the widget.
	Title *string `json:"title,omitempty"`
	// How to align the text on the widget.
	TitleAlign *WidgetTextAlign `json:"title_align,omitempty"`
	// Size of the title.
	TitleSize *string `json:"title_size,omitempty"`
	// Type of the log stream widget.
	Type LogStreamWidgetDefinitionType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogStreamWidgetDefinition instantiates a new LogStreamWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogStreamWidgetDefinition(typeVar LogStreamWidgetDefinitionType) *LogStreamWidgetDefinition {
	this := LogStreamWidgetDefinition{}
	this.Type = typeVar
	return &this
}

// NewLogStreamWidgetDefinitionWithDefaults instantiates a new LogStreamWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogStreamWidgetDefinitionWithDefaults() *LogStreamWidgetDefinition {
	this := LogStreamWidgetDefinition{}
	var typeVar LogStreamWidgetDefinitionType = LOGSTREAMWIDGETDEFINITIONTYPE_LOG_STREAM
	this.Type = typeVar
	return &this
}

// GetColumns returns the Columns field value if set, zero value otherwise.
func (o *LogStreamWidgetDefinition) GetColumns() []string {
	if o == nil || o.Columns == nil {
		var ret []string
		return ret
	}
	return o.Columns
}

// GetColumnsOk returns a tuple with the Columns field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogStreamWidgetDefinition) GetColumnsOk() (*[]string, bool) {
	if o == nil || o.Columns == nil {
		return nil, false
	}
	return &o.Columns, true
}

// HasColumns returns a boolean if a field has been set.
func (o *LogStreamWidgetDefinition) HasColumns() bool {
	if o != nil && o.Columns != nil {
		return true
	}

	return false
}

// SetColumns gets a reference to the given []string and assigns it to the Columns field.
func (o *LogStreamWidgetDefinition) SetColumns(v []string) {
	o.Columns = v
}

// GetIndexes returns the Indexes field value if set, zero value otherwise.
func (o *LogStreamWidgetDefinition) GetIndexes() []string {
	if o == nil || o.Indexes == nil {
		var ret []string
		return ret
	}
	return o.Indexes
}

// GetIndexesOk returns a tuple with the Indexes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogStreamWidgetDefinition) GetIndexesOk() (*[]string, bool) {
	if o == nil || o.Indexes == nil {
		return nil, false
	}
	return &o.Indexes, true
}

// HasIndexes returns a boolean if a field has been set.
func (o *LogStreamWidgetDefinition) HasIndexes() bool {
	if o != nil && o.Indexes != nil {
		return true
	}

	return false
}

// SetIndexes gets a reference to the given []string and assigns it to the Indexes field.
func (o *LogStreamWidgetDefinition) SetIndexes(v []string) {
	o.Indexes = v
}

// GetLogset returns the Logset field value if set, zero value otherwise.
// Deprecated
func (o *LogStreamWidgetDefinition) GetLogset() string {
	if o == nil || o.Logset == nil {
		var ret string
		return ret
	}
	return *o.Logset
}

// GetLogsetOk returns a tuple with the Logset field value if set, nil otherwise
// and a boolean to check if the value has been set.
// Deprecated
func (o *LogStreamWidgetDefinition) GetLogsetOk() (*string, bool) {
	if o == nil || o.Logset == nil {
		return nil, false
	}
	return o.Logset, true
}

// HasLogset returns a boolean if a field has been set.
func (o *LogStreamWidgetDefinition) HasLogset() bool {
	if o != nil && o.Logset != nil {
		return true
	}

	return false
}

// SetLogset gets a reference to the given string and assigns it to the Logset field.
// Deprecated
func (o *LogStreamWidgetDefinition) SetLogset(v string) {
	o.Logset = &v
}

// GetMessageDisplay returns the MessageDisplay field value if set, zero value otherwise.
func (o *LogStreamWidgetDefinition) GetMessageDisplay() WidgetMessageDisplay {
	if o == nil || o.MessageDisplay == nil {
		var ret WidgetMessageDisplay
		return ret
	}
	return *o.MessageDisplay
}

// GetMessageDisplayOk returns a tuple with the MessageDisplay field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogStreamWidgetDefinition) GetMessageDisplayOk() (*WidgetMessageDisplay, bool) {
	if o == nil || o.MessageDisplay == nil {
		return nil, false
	}
	return o.MessageDisplay, true
}

// HasMessageDisplay returns a boolean if a field has been set.
func (o *LogStreamWidgetDefinition) HasMessageDisplay() bool {
	if o != nil && o.MessageDisplay != nil {
		return true
	}

	return false
}

// SetMessageDisplay gets a reference to the given WidgetMessageDisplay and assigns it to the MessageDisplay field.
func (o *LogStreamWidgetDefinition) SetMessageDisplay(v WidgetMessageDisplay) {
	o.MessageDisplay = &v
}

// GetQuery returns the Query field value if set, zero value otherwise.
func (o *LogStreamWidgetDefinition) GetQuery() string {
	if o == nil || o.Query == nil {
		var ret string
		return ret
	}
	return *o.Query
}

// GetQueryOk returns a tuple with the Query field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogStreamWidgetDefinition) GetQueryOk() (*string, bool) {
	if o == nil || o.Query == nil {
		return nil, false
	}
	return o.Query, true
}

// HasQuery returns a boolean if a field has been set.
func (o *LogStreamWidgetDefinition) HasQuery() bool {
	if o != nil && o.Query != nil {
		return true
	}

	return false
}

// SetQuery gets a reference to the given string and assigns it to the Query field.
func (o *LogStreamWidgetDefinition) SetQuery(v string) {
	o.Query = &v
}

// GetShowDateColumn returns the ShowDateColumn field value if set, zero value otherwise.
func (o *LogStreamWidgetDefinition) GetShowDateColumn() bool {
	if o == nil || o.ShowDateColumn == nil {
		var ret bool
		return ret
	}
	return *o.ShowDateColumn
}

// GetShowDateColumnOk returns a tuple with the ShowDateColumn field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogStreamWidgetDefinition) GetShowDateColumnOk() (*bool, bool) {
	if o == nil || o.ShowDateColumn == nil {
		return nil, false
	}
	return o.ShowDateColumn, true
}

// HasShowDateColumn returns a boolean if a field has been set.
func (o *LogStreamWidgetDefinition) HasShowDateColumn() bool {
	if o != nil && o.ShowDateColumn != nil {
		return true
	}

	return false
}

// SetShowDateColumn gets a reference to the given bool and assigns it to the ShowDateColumn field.
func (o *LogStreamWidgetDefinition) SetShowDateColumn(v bool) {
	o.ShowDateColumn = &v
}

// GetShowMessageColumn returns the ShowMessageColumn field value if set, zero value otherwise.
func (o *LogStreamWidgetDefinition) GetShowMessageColumn() bool {
	if o == nil || o.ShowMessageColumn == nil {
		var ret bool
		return ret
	}
	return *o.ShowMessageColumn
}

// GetShowMessageColumnOk returns a tuple with the ShowMessageColumn field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogStreamWidgetDefinition) GetShowMessageColumnOk() (*bool, bool) {
	if o == nil || o.ShowMessageColumn == nil {
		return nil, false
	}
	return o.ShowMessageColumn, true
}

// HasShowMessageColumn returns a boolean if a field has been set.
func (o *LogStreamWidgetDefinition) HasShowMessageColumn() bool {
	if o != nil && o.ShowMessageColumn != nil {
		return true
	}

	return false
}

// SetShowMessageColumn gets a reference to the given bool and assigns it to the ShowMessageColumn field.
func (o *LogStreamWidgetDefinition) SetShowMessageColumn(v bool) {
	o.ShowMessageColumn = &v
}

// GetSort returns the Sort field value if set, zero value otherwise.
func (o *LogStreamWidgetDefinition) GetSort() WidgetFieldSort {
	if o == nil || o.Sort == nil {
		var ret WidgetFieldSort
		return ret
	}
	return *o.Sort
}

// GetSortOk returns a tuple with the Sort field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogStreamWidgetDefinition) GetSortOk() (*WidgetFieldSort, bool) {
	if o == nil || o.Sort == nil {
		return nil, false
	}
	return o.Sort, true
}

// HasSort returns a boolean if a field has been set.
func (o *LogStreamWidgetDefinition) HasSort() bool {
	if o != nil && o.Sort != nil {
		return true
	}

	return false
}

// SetSort gets a reference to the given WidgetFieldSort and assigns it to the Sort field.
func (o *LogStreamWidgetDefinition) SetSort(v WidgetFieldSort) {
	o.Sort = &v
}

// GetTime returns the Time field value if set, zero value otherwise.
func (o *LogStreamWidgetDefinition) GetTime() WidgetTime {
	if o == nil || o.Time == nil {
		var ret WidgetTime
		return ret
	}
	return *o.Time
}

// GetTimeOk returns a tuple with the Time field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogStreamWidgetDefinition) GetTimeOk() (*WidgetTime, bool) {
	if o == nil || o.Time == nil {
		return nil, false
	}
	return o.Time, true
}

// HasTime returns a boolean if a field has been set.
func (o *LogStreamWidgetDefinition) HasTime() bool {
	if o != nil && o.Time != nil {
		return true
	}

	return false
}

// SetTime gets a reference to the given WidgetTime and assigns it to the Time field.
func (o *LogStreamWidgetDefinition) SetTime(v WidgetTime) {
	o.Time = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *LogStreamWidgetDefinition) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogStreamWidgetDefinition) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *LogStreamWidgetDefinition) HasTitle() bool {
	if o != nil && o.Title != nil {
		return true
	}

	return false
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *LogStreamWidgetDefinition) SetTitle(v string) {
	o.Title = &v
}

// GetTitleAlign returns the TitleAlign field value if set, zero value otherwise.
func (o *LogStreamWidgetDefinition) GetTitleAlign() WidgetTextAlign {
	if o == nil || o.TitleAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TitleAlign
}

// GetTitleAlignOk returns a tuple with the TitleAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogStreamWidgetDefinition) GetTitleAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TitleAlign == nil {
		return nil, false
	}
	return o.TitleAlign, true
}

// HasTitleAlign returns a boolean if a field has been set.
func (o *LogStreamWidgetDefinition) HasTitleAlign() bool {
	if o != nil && o.TitleAlign != nil {
		return true
	}

	return false
}

// SetTitleAlign gets a reference to the given WidgetTextAlign and assigns it to the TitleAlign field.
func (o *LogStreamWidgetDefinition) SetTitleAlign(v WidgetTextAlign) {
	o.TitleAlign = &v
}

// GetTitleSize returns the TitleSize field value if set, zero value otherwise.
func (o *LogStreamWidgetDefinition) GetTitleSize() string {
	if o == nil || o.TitleSize == nil {
		var ret string
		return ret
	}
	return *o.TitleSize
}

// GetTitleSizeOk returns a tuple with the TitleSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogStreamWidgetDefinition) GetTitleSizeOk() (*string, bool) {
	if o == nil || o.TitleSize == nil {
		return nil, false
	}
	return o.TitleSize, true
}

// HasTitleSize returns a boolean if a field has been set.
func (o *LogStreamWidgetDefinition) HasTitleSize() bool {
	if o != nil && o.TitleSize != nil {
		return true
	}

	return false
}

// SetTitleSize gets a reference to the given string and assigns it to the TitleSize field.
func (o *LogStreamWidgetDefinition) SetTitleSize(v string) {
	o.TitleSize = &v
}

// GetType returns the Type field value.
func (o *LogStreamWidgetDefinition) GetType() LogStreamWidgetDefinitionType {
	if o == nil {
		var ret LogStreamWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *LogStreamWidgetDefinition) GetTypeOk() (*LogStreamWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *LogStreamWidgetDefinition) SetType(v LogStreamWidgetDefinitionType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogStreamWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Columns != nil {
		toSerialize["columns"] = o.Columns
	}
	if o.Indexes != nil {
		toSerialize["indexes"] = o.Indexes
	}
	if o.Logset != nil {
		toSerialize["logset"] = o.Logset
	}
	if o.MessageDisplay != nil {
		toSerialize["message_display"] = o.MessageDisplay
	}
	if o.Query != nil {
		toSerialize["query"] = o.Query
	}
	if o.ShowDateColumn != nil {
		toSerialize["show_date_column"] = o.ShowDateColumn
	}
	if o.ShowMessageColumn != nil {
		toSerialize["show_message_column"] = o.ShowMessageColumn
	}
	if o.Sort != nil {
		toSerialize["sort"] = o.Sort
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
func (o *LogStreamWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Type *LogStreamWidgetDefinitionType `json:"type"`
	}{}
	all := struct {
		Columns           []string                      `json:"columns,omitempty"`
		Indexes           []string                      `json:"indexes,omitempty"`
		Logset            *string                       `json:"logset,omitempty"`
		MessageDisplay    *WidgetMessageDisplay         `json:"message_display,omitempty"`
		Query             *string                       `json:"query,omitempty"`
		ShowDateColumn    *bool                         `json:"show_date_column,omitempty"`
		ShowMessageColumn *bool                         `json:"show_message_column,omitempty"`
		Sort              *WidgetFieldSort              `json:"sort,omitempty"`
		Time              *WidgetTime                   `json:"time,omitempty"`
		Title             *string                       `json:"title,omitempty"`
		TitleAlign        *WidgetTextAlign              `json:"title_align,omitempty"`
		TitleSize         *string                       `json:"title_size,omitempty"`
		Type              LogStreamWidgetDefinitionType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
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
	if v := all.MessageDisplay; v != nil && !v.IsValid() {
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
	o.Columns = all.Columns
	o.Indexes = all.Indexes
	o.Logset = all.Logset
	o.MessageDisplay = all.MessageDisplay
	o.Query = all.Query
	o.ShowDateColumn = all.ShowDateColumn
	o.ShowMessageColumn = all.ShowMessageColumn
	if all.Sort != nil && all.Sort.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Sort = all.Sort
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
