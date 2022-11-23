// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsPipeline Pipelines and processors operate on incoming logs,
// parsing and transforming them into structured attributes for easier querying.
//
// **Note**: These endpoints are only available for admin users.
// Make sure to use an application key created by an admin.
type LogsPipeline struct {
	// Filter for logs.
	Filter *LogsFilter `json:"filter,omitempty"`
	// ID of the pipeline.
	Id *string `json:"id,omitempty"`
	// Whether or not the pipeline is enabled.
	IsEnabled *bool `json:"is_enabled,omitempty"`
	// Whether or not the pipeline can be edited.
	IsReadOnly *bool `json:"is_read_only,omitempty"`
	// Name of the pipeline.
	Name string `json:"name"`
	// Ordered list of processors in this pipeline.
	Processors []LogsProcessor `json:"processors,omitempty"`
	// Type of pipeline.
	Type *string `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsPipeline instantiates a new LogsPipeline object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsPipeline(name string) *LogsPipeline {
	this := LogsPipeline{}
	this.Name = name
	return &this
}

// NewLogsPipelineWithDefaults instantiates a new LogsPipeline object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsPipelineWithDefaults() *LogsPipeline {
	this := LogsPipeline{}
	return &this
}

// GetFilter returns the Filter field value if set, zero value otherwise.
func (o *LogsPipeline) GetFilter() LogsFilter {
	if o == nil || o.Filter == nil {
		var ret LogsFilter
		return ret
	}
	return *o.Filter
}

// GetFilterOk returns a tuple with the Filter field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsPipeline) GetFilterOk() (*LogsFilter, bool) {
	if o == nil || o.Filter == nil {
		return nil, false
	}
	return o.Filter, true
}

// HasFilter returns a boolean if a field has been set.
func (o *LogsPipeline) HasFilter() bool {
	if o != nil && o.Filter != nil {
		return true
	}

	return false
}

// SetFilter gets a reference to the given LogsFilter and assigns it to the Filter field.
func (o *LogsPipeline) SetFilter(v LogsFilter) {
	o.Filter = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *LogsPipeline) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsPipeline) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *LogsPipeline) HasId() bool {
	if o != nil && o.Id != nil {
		return true
	}

	return false
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *LogsPipeline) SetId(v string) {
	o.Id = &v
}

// GetIsEnabled returns the IsEnabled field value if set, zero value otherwise.
func (o *LogsPipeline) GetIsEnabled() bool {
	if o == nil || o.IsEnabled == nil {
		var ret bool
		return ret
	}
	return *o.IsEnabled
}

// GetIsEnabledOk returns a tuple with the IsEnabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsPipeline) GetIsEnabledOk() (*bool, bool) {
	if o == nil || o.IsEnabled == nil {
		return nil, false
	}
	return o.IsEnabled, true
}

// HasIsEnabled returns a boolean if a field has been set.
func (o *LogsPipeline) HasIsEnabled() bool {
	if o != nil && o.IsEnabled != nil {
		return true
	}

	return false
}

// SetIsEnabled gets a reference to the given bool and assigns it to the IsEnabled field.
func (o *LogsPipeline) SetIsEnabled(v bool) {
	o.IsEnabled = &v
}

// GetIsReadOnly returns the IsReadOnly field value if set, zero value otherwise.
func (o *LogsPipeline) GetIsReadOnly() bool {
	if o == nil || o.IsReadOnly == nil {
		var ret bool
		return ret
	}
	return *o.IsReadOnly
}

// GetIsReadOnlyOk returns a tuple with the IsReadOnly field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsPipeline) GetIsReadOnlyOk() (*bool, bool) {
	if o == nil || o.IsReadOnly == nil {
		return nil, false
	}
	return o.IsReadOnly, true
}

// HasIsReadOnly returns a boolean if a field has been set.
func (o *LogsPipeline) HasIsReadOnly() bool {
	if o != nil && o.IsReadOnly != nil {
		return true
	}

	return false
}

// SetIsReadOnly gets a reference to the given bool and assigns it to the IsReadOnly field.
func (o *LogsPipeline) SetIsReadOnly(v bool) {
	o.IsReadOnly = &v
}

// GetName returns the Name field value.
func (o *LogsPipeline) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *LogsPipeline) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *LogsPipeline) SetName(v string) {
	o.Name = v
}

// GetProcessors returns the Processors field value if set, zero value otherwise.
func (o *LogsPipeline) GetProcessors() []LogsProcessor {
	if o == nil || o.Processors == nil {
		var ret []LogsProcessor
		return ret
	}
	return o.Processors
}

// GetProcessorsOk returns a tuple with the Processors field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsPipeline) GetProcessorsOk() (*[]LogsProcessor, bool) {
	if o == nil || o.Processors == nil {
		return nil, false
	}
	return &o.Processors, true
}

// HasProcessors returns a boolean if a field has been set.
func (o *LogsPipeline) HasProcessors() bool {
	if o != nil && o.Processors != nil {
		return true
	}

	return false
}

// SetProcessors gets a reference to the given []LogsProcessor and assigns it to the Processors field.
func (o *LogsPipeline) SetProcessors(v []LogsProcessor) {
	o.Processors = v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *LogsPipeline) GetType() string {
	if o == nil || o.Type == nil {
		var ret string
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsPipeline) GetTypeOk() (*string, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *LogsPipeline) HasType() bool {
	if o != nil && o.Type != nil {
		return true
	}

	return false
}

// SetType gets a reference to the given string and assigns it to the Type field.
func (o *LogsPipeline) SetType(v string) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsPipeline) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Filter != nil {
		toSerialize["filter"] = o.Filter
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
	if o.IsEnabled != nil {
		toSerialize["is_enabled"] = o.IsEnabled
	}
	if o.IsReadOnly != nil {
		toSerialize["is_read_only"] = o.IsReadOnly
	}
	toSerialize["name"] = o.Name
	if o.Processors != nil {
		toSerialize["processors"] = o.Processors
	}
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsPipeline) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Name *string `json:"name"`
	}{}
	all := struct {
		Filter     *LogsFilter     `json:"filter,omitempty"`
		Id         *string         `json:"id,omitempty"`
		IsEnabled  *bool           `json:"is_enabled,omitempty"`
		IsReadOnly *bool           `json:"is_read_only,omitempty"`
		Name       string          `json:"name"`
		Processors []LogsProcessor `json:"processors,omitempty"`
		Type       *string         `json:"type,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
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
	if all.Filter != nil && all.Filter.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Filter = all.Filter
	o.Id = all.Id
	o.IsEnabled = all.IsEnabled
	o.IsReadOnly = all.IsReadOnly
	o.Name = all.Name
	o.Processors = all.Processors
	o.Type = all.Type
	return nil
}
