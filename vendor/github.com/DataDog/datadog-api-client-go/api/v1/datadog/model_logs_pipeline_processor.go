// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsPipelineProcessor Nested Pipelines are pipelines within a pipeline. Use Nested Pipelines to split the processing into two steps.
// For example, first use a high-level filtering such as team and then a second level of filtering based on the
// integration, service, or any other tag or attribute.
//
// A pipeline can contain Nested Pipelines and Processors whereas a Nested Pipeline can only contain Processors.
type LogsPipelineProcessor struct {
	// Filter for logs.
	Filter *LogsFilter `json:"filter,omitempty"`
	// Whether or not the processor is enabled.
	IsEnabled *bool `json:"is_enabled,omitempty"`
	// Name of the processor.
	Name *string `json:"name,omitempty"`
	// Ordered list of processors in this pipeline.
	Processors []LogsProcessor `json:"processors,omitempty"`
	// Type of logs pipeline processor.
	Type LogsPipelineProcessorType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsPipelineProcessor instantiates a new LogsPipelineProcessor object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsPipelineProcessor(typeVar LogsPipelineProcessorType) *LogsPipelineProcessor {
	this := LogsPipelineProcessor{}
	var isEnabled bool = false
	this.IsEnabled = &isEnabled
	this.Type = typeVar
	return &this
}

// NewLogsPipelineProcessorWithDefaults instantiates a new LogsPipelineProcessor object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsPipelineProcessorWithDefaults() *LogsPipelineProcessor {
	this := LogsPipelineProcessor{}
	var isEnabled bool = false
	this.IsEnabled = &isEnabled
	var typeVar LogsPipelineProcessorType = LOGSPIPELINEPROCESSORTYPE_PIPELINE
	this.Type = typeVar
	return &this
}

// GetFilter returns the Filter field value if set, zero value otherwise.
func (o *LogsPipelineProcessor) GetFilter() LogsFilter {
	if o == nil || o.Filter == nil {
		var ret LogsFilter
		return ret
	}
	return *o.Filter
}

// GetFilterOk returns a tuple with the Filter field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsPipelineProcessor) GetFilterOk() (*LogsFilter, bool) {
	if o == nil || o.Filter == nil {
		return nil, false
	}
	return o.Filter, true
}

// HasFilter returns a boolean if a field has been set.
func (o *LogsPipelineProcessor) HasFilter() bool {
	if o != nil && o.Filter != nil {
		return true
	}

	return false
}

// SetFilter gets a reference to the given LogsFilter and assigns it to the Filter field.
func (o *LogsPipelineProcessor) SetFilter(v LogsFilter) {
	o.Filter = &v
}

// GetIsEnabled returns the IsEnabled field value if set, zero value otherwise.
func (o *LogsPipelineProcessor) GetIsEnabled() bool {
	if o == nil || o.IsEnabled == nil {
		var ret bool
		return ret
	}
	return *o.IsEnabled
}

// GetIsEnabledOk returns a tuple with the IsEnabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsPipelineProcessor) GetIsEnabledOk() (*bool, bool) {
	if o == nil || o.IsEnabled == nil {
		return nil, false
	}
	return o.IsEnabled, true
}

// HasIsEnabled returns a boolean if a field has been set.
func (o *LogsPipelineProcessor) HasIsEnabled() bool {
	if o != nil && o.IsEnabled != nil {
		return true
	}

	return false
}

// SetIsEnabled gets a reference to the given bool and assigns it to the IsEnabled field.
func (o *LogsPipelineProcessor) SetIsEnabled(v bool) {
	o.IsEnabled = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *LogsPipelineProcessor) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsPipelineProcessor) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *LogsPipelineProcessor) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *LogsPipelineProcessor) SetName(v string) {
	o.Name = &v
}

// GetProcessors returns the Processors field value if set, zero value otherwise.
func (o *LogsPipelineProcessor) GetProcessors() []LogsProcessor {
	if o == nil || o.Processors == nil {
		var ret []LogsProcessor
		return ret
	}
	return o.Processors
}

// GetProcessorsOk returns a tuple with the Processors field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsPipelineProcessor) GetProcessorsOk() (*[]LogsProcessor, bool) {
	if o == nil || o.Processors == nil {
		return nil, false
	}
	return &o.Processors, true
}

// HasProcessors returns a boolean if a field has been set.
func (o *LogsPipelineProcessor) HasProcessors() bool {
	if o != nil && o.Processors != nil {
		return true
	}

	return false
}

// SetProcessors gets a reference to the given []LogsProcessor and assigns it to the Processors field.
func (o *LogsPipelineProcessor) SetProcessors(v []LogsProcessor) {
	o.Processors = v
}

// GetType returns the Type field value.
func (o *LogsPipelineProcessor) GetType() LogsPipelineProcessorType {
	if o == nil {
		var ret LogsPipelineProcessorType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *LogsPipelineProcessor) GetTypeOk() (*LogsPipelineProcessorType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *LogsPipelineProcessor) SetType(v LogsPipelineProcessorType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsPipelineProcessor) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Filter != nil {
		toSerialize["filter"] = o.Filter
	}
	if o.IsEnabled != nil {
		toSerialize["is_enabled"] = o.IsEnabled
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Processors != nil {
		toSerialize["processors"] = o.Processors
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsPipelineProcessor) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Type *LogsPipelineProcessorType `json:"type"`
	}{}
	all := struct {
		Filter     *LogsFilter               `json:"filter,omitempty"`
		IsEnabled  *bool                     `json:"is_enabled,omitempty"`
		Name       *string                   `json:"name,omitempty"`
		Processors []LogsProcessor           `json:"processors,omitempty"`
		Type       LogsPipelineProcessorType `json:"type"`
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
	if v := all.Type; !v.IsValid() {
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
	o.IsEnabled = all.IsEnabled
	o.Name = all.Name
	o.Processors = all.Processors
	o.Type = all.Type
	return nil
}
