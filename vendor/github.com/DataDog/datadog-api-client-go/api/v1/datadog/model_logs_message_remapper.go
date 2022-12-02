// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsMessageRemapper The message is a key attribute in Datadog.
// It is displayed in the message column of the Log Explorer and you can do full string search on it.
// Use this Processor to define one or more attributes as the official log message.
//
// **Note:** If multiple log message remapper processors can be applied to a given log,
// only the first one (according to the pipeline order) is taken into account.
type LogsMessageRemapper struct {
	// Whether or not the processor is enabled.
	IsEnabled *bool `json:"is_enabled,omitempty"`
	// Name of the processor.
	Name *string `json:"name,omitempty"`
	// Array of source attributes.
	Sources []string `json:"sources"`
	// Type of logs message remapper.
	Type LogsMessageRemapperType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsMessageRemapper instantiates a new LogsMessageRemapper object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsMessageRemapper(sources []string, typeVar LogsMessageRemapperType) *LogsMessageRemapper {
	this := LogsMessageRemapper{}
	var isEnabled bool = false
	this.IsEnabled = &isEnabled
	this.Sources = sources
	this.Type = typeVar
	return &this
}

// NewLogsMessageRemapperWithDefaults instantiates a new LogsMessageRemapper object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsMessageRemapperWithDefaults() *LogsMessageRemapper {
	this := LogsMessageRemapper{}
	var isEnabled bool = false
	this.IsEnabled = &isEnabled
	var typeVar LogsMessageRemapperType = LOGSMESSAGEREMAPPERTYPE_MESSAGE_REMAPPER
	this.Type = typeVar
	return &this
}

// GetIsEnabled returns the IsEnabled field value if set, zero value otherwise.
func (o *LogsMessageRemapper) GetIsEnabled() bool {
	if o == nil || o.IsEnabled == nil {
		var ret bool
		return ret
	}
	return *o.IsEnabled
}

// GetIsEnabledOk returns a tuple with the IsEnabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsMessageRemapper) GetIsEnabledOk() (*bool, bool) {
	if o == nil || o.IsEnabled == nil {
		return nil, false
	}
	return o.IsEnabled, true
}

// HasIsEnabled returns a boolean if a field has been set.
func (o *LogsMessageRemapper) HasIsEnabled() bool {
	if o != nil && o.IsEnabled != nil {
		return true
	}

	return false
}

// SetIsEnabled gets a reference to the given bool and assigns it to the IsEnabled field.
func (o *LogsMessageRemapper) SetIsEnabled(v bool) {
	o.IsEnabled = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *LogsMessageRemapper) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsMessageRemapper) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *LogsMessageRemapper) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *LogsMessageRemapper) SetName(v string) {
	o.Name = &v
}

// GetSources returns the Sources field value.
func (o *LogsMessageRemapper) GetSources() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.Sources
}

// GetSourcesOk returns a tuple with the Sources field value
// and a boolean to check if the value has been set.
func (o *LogsMessageRemapper) GetSourcesOk() (*[]string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Sources, true
}

// SetSources sets field value.
func (o *LogsMessageRemapper) SetSources(v []string) {
	o.Sources = v
}

// GetType returns the Type field value.
func (o *LogsMessageRemapper) GetType() LogsMessageRemapperType {
	if o == nil {
		var ret LogsMessageRemapperType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *LogsMessageRemapper) GetTypeOk() (*LogsMessageRemapperType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *LogsMessageRemapper) SetType(v LogsMessageRemapperType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsMessageRemapper) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.IsEnabled != nil {
		toSerialize["is_enabled"] = o.IsEnabled
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	toSerialize["sources"] = o.Sources
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsMessageRemapper) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Sources *[]string                `json:"sources"`
		Type    *LogsMessageRemapperType `json:"type"`
	}{}
	all := struct {
		IsEnabled *bool                   `json:"is_enabled,omitempty"`
		Name      *string                 `json:"name,omitempty"`
		Sources   []string                `json:"sources"`
		Type      LogsMessageRemapperType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Sources == nil {
		return fmt.Errorf("Required field sources missing")
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
	o.IsEnabled = all.IsEnabled
	o.Name = all.Name
	o.Sources = all.Sources
	o.Type = all.Type
	return nil
}
