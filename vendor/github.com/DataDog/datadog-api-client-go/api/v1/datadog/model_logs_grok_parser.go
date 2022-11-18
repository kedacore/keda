// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsGrokParser Create custom grok rules to parse the full message or [a specific attribute of your raw event](https://docs.datadoghq.com/logs/log_configuration/parsing/#advanced-settings).
// For more information, see the [parsing section](https://docs.datadoghq.com/logs/log_configuration/parsing).
type LogsGrokParser struct {
	// Set of rules for the grok parser.
	Grok LogsGrokParserRules `json:"grok"`
	// Whether or not the processor is enabled.
	IsEnabled *bool `json:"is_enabled,omitempty"`
	// Name of the processor.
	Name *string `json:"name,omitempty"`
	// List of sample logs to test this grok parser.
	Samples []string `json:"samples,omitempty"`
	// Name of the log attribute to parse.
	Source string `json:"source"`
	// Type of logs grok parser.
	Type LogsGrokParserType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsGrokParser instantiates a new LogsGrokParser object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsGrokParser(grok LogsGrokParserRules, source string, typeVar LogsGrokParserType) *LogsGrokParser {
	this := LogsGrokParser{}
	this.Grok = grok
	var isEnabled bool = false
	this.IsEnabled = &isEnabled
	this.Source = source
	this.Type = typeVar
	return &this
}

// NewLogsGrokParserWithDefaults instantiates a new LogsGrokParser object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsGrokParserWithDefaults() *LogsGrokParser {
	this := LogsGrokParser{}
	var isEnabled bool = false
	this.IsEnabled = &isEnabled
	var source string = "message"
	this.Source = source
	var typeVar LogsGrokParserType = LOGSGROKPARSERTYPE_GROK_PARSER
	this.Type = typeVar
	return &this
}

// GetGrok returns the Grok field value.
func (o *LogsGrokParser) GetGrok() LogsGrokParserRules {
	if o == nil {
		var ret LogsGrokParserRules
		return ret
	}
	return o.Grok
}

// GetGrokOk returns a tuple with the Grok field value
// and a boolean to check if the value has been set.
func (o *LogsGrokParser) GetGrokOk() (*LogsGrokParserRules, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Grok, true
}

// SetGrok sets field value.
func (o *LogsGrokParser) SetGrok(v LogsGrokParserRules) {
	o.Grok = v
}

// GetIsEnabled returns the IsEnabled field value if set, zero value otherwise.
func (o *LogsGrokParser) GetIsEnabled() bool {
	if o == nil || o.IsEnabled == nil {
		var ret bool
		return ret
	}
	return *o.IsEnabled
}

// GetIsEnabledOk returns a tuple with the IsEnabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsGrokParser) GetIsEnabledOk() (*bool, bool) {
	if o == nil || o.IsEnabled == nil {
		return nil, false
	}
	return o.IsEnabled, true
}

// HasIsEnabled returns a boolean if a field has been set.
func (o *LogsGrokParser) HasIsEnabled() bool {
	if o != nil && o.IsEnabled != nil {
		return true
	}

	return false
}

// SetIsEnabled gets a reference to the given bool and assigns it to the IsEnabled field.
func (o *LogsGrokParser) SetIsEnabled(v bool) {
	o.IsEnabled = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *LogsGrokParser) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsGrokParser) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *LogsGrokParser) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *LogsGrokParser) SetName(v string) {
	o.Name = &v
}

// GetSamples returns the Samples field value if set, zero value otherwise.
func (o *LogsGrokParser) GetSamples() []string {
	if o == nil || o.Samples == nil {
		var ret []string
		return ret
	}
	return o.Samples
}

// GetSamplesOk returns a tuple with the Samples field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsGrokParser) GetSamplesOk() (*[]string, bool) {
	if o == nil || o.Samples == nil {
		return nil, false
	}
	return &o.Samples, true
}

// HasSamples returns a boolean if a field has been set.
func (o *LogsGrokParser) HasSamples() bool {
	if o != nil && o.Samples != nil {
		return true
	}

	return false
}

// SetSamples gets a reference to the given []string and assigns it to the Samples field.
func (o *LogsGrokParser) SetSamples(v []string) {
	o.Samples = v
}

// GetSource returns the Source field value.
func (o *LogsGrokParser) GetSource() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Source
}

// GetSourceOk returns a tuple with the Source field value
// and a boolean to check if the value has been set.
func (o *LogsGrokParser) GetSourceOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Source, true
}

// SetSource sets field value.
func (o *LogsGrokParser) SetSource(v string) {
	o.Source = v
}

// GetType returns the Type field value.
func (o *LogsGrokParser) GetType() LogsGrokParserType {
	if o == nil {
		var ret LogsGrokParserType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *LogsGrokParser) GetTypeOk() (*LogsGrokParserType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *LogsGrokParser) SetType(v LogsGrokParserType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsGrokParser) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["grok"] = o.Grok
	if o.IsEnabled != nil {
		toSerialize["is_enabled"] = o.IsEnabled
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Samples != nil {
		toSerialize["samples"] = o.Samples
	}
	toSerialize["source"] = o.Source
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsGrokParser) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Grok   *LogsGrokParserRules `json:"grok"`
		Source *string              `json:"source"`
		Type   *LogsGrokParserType  `json:"type"`
	}{}
	all := struct {
		Grok      LogsGrokParserRules `json:"grok"`
		IsEnabled *bool               `json:"is_enabled,omitempty"`
		Name      *string             `json:"name,omitempty"`
		Samples   []string            `json:"samples,omitempty"`
		Source    string              `json:"source"`
		Type      LogsGrokParserType  `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Grok == nil {
		return fmt.Errorf("Required field grok missing")
	}
	if required.Source == nil {
		return fmt.Errorf("Required field source missing")
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
	if all.Grok.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Grok = all.Grok
	o.IsEnabled = all.IsEnabled
	o.Name = all.Name
	o.Samples = all.Samples
	o.Source = all.Source
	o.Type = all.Type
	return nil
}
