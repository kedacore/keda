// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsAttributeRemapper The remapper processor remaps any source attribute(s) or tag to another target attribute or tag.
// Constraints on the tag/attribute name are explained in the [Tag Best Practice documentation](https://docs.datadoghq.com/logs/guide/log-parsing-best-practice).
// Some additional constraints are applied as `:` or `,` are not allowed in the target tag/attribute name.
type LogsAttributeRemapper struct {
	// Whether or not the processor is enabled.
	IsEnabled *bool `json:"is_enabled,omitempty"`
	// Name of the processor.
	Name *string `json:"name,omitempty"`
	// Override or not the target element if already set,
	OverrideOnConflict *bool `json:"override_on_conflict,omitempty"`
	// Remove or preserve the remapped source element.
	PreserveSource *bool `json:"preserve_source,omitempty"`
	// Defines if the sources are from log `attribute` or `tag`.
	SourceType *string `json:"source_type,omitempty"`
	// Array of source attributes.
	Sources []string `json:"sources"`
	// Final attribute or tag name to remap the sources to.
	Target string `json:"target"`
	// If the `target_type` of the remapper is `attribute`, try to cast the value to a new specific type.
	// If the cast is not possible, the original type is kept. `string`, `integer`, or `double` are the possible types.
	// If the `target_type` is `tag`, this parameter may not be specified.
	TargetFormat *TargetFormatType `json:"target_format,omitempty"`
	// Defines if the final attribute or tag name is from log `attribute` or `tag`.
	TargetType *string `json:"target_type,omitempty"`
	// Type of logs attribute remapper.
	Type LogsAttributeRemapperType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsAttributeRemapper instantiates a new LogsAttributeRemapper object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsAttributeRemapper(sources []string, target string, typeVar LogsAttributeRemapperType) *LogsAttributeRemapper {
	this := LogsAttributeRemapper{}
	var isEnabled bool = false
	this.IsEnabled = &isEnabled
	var overrideOnConflict bool = false
	this.OverrideOnConflict = &overrideOnConflict
	var preserveSource bool = false
	this.PreserveSource = &preserveSource
	var sourceType string = "attribute"
	this.SourceType = &sourceType
	this.Sources = sources
	this.Target = target
	var targetType string = "attribute"
	this.TargetType = &targetType
	this.Type = typeVar
	return &this
}

// NewLogsAttributeRemapperWithDefaults instantiates a new LogsAttributeRemapper object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsAttributeRemapperWithDefaults() *LogsAttributeRemapper {
	this := LogsAttributeRemapper{}
	var isEnabled bool = false
	this.IsEnabled = &isEnabled
	var overrideOnConflict bool = false
	this.OverrideOnConflict = &overrideOnConflict
	var preserveSource bool = false
	this.PreserveSource = &preserveSource
	var sourceType string = "attribute"
	this.SourceType = &sourceType
	var targetType string = "attribute"
	this.TargetType = &targetType
	var typeVar LogsAttributeRemapperType = LOGSATTRIBUTEREMAPPERTYPE_ATTRIBUTE_REMAPPER
	this.Type = typeVar
	return &this
}

// GetIsEnabled returns the IsEnabled field value if set, zero value otherwise.
func (o *LogsAttributeRemapper) GetIsEnabled() bool {
	if o == nil || o.IsEnabled == nil {
		var ret bool
		return ret
	}
	return *o.IsEnabled
}

// GetIsEnabledOk returns a tuple with the IsEnabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsAttributeRemapper) GetIsEnabledOk() (*bool, bool) {
	if o == nil || o.IsEnabled == nil {
		return nil, false
	}
	return o.IsEnabled, true
}

// HasIsEnabled returns a boolean if a field has been set.
func (o *LogsAttributeRemapper) HasIsEnabled() bool {
	if o != nil && o.IsEnabled != nil {
		return true
	}

	return false
}

// SetIsEnabled gets a reference to the given bool and assigns it to the IsEnabled field.
func (o *LogsAttributeRemapper) SetIsEnabled(v bool) {
	o.IsEnabled = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *LogsAttributeRemapper) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsAttributeRemapper) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *LogsAttributeRemapper) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *LogsAttributeRemapper) SetName(v string) {
	o.Name = &v
}

// GetOverrideOnConflict returns the OverrideOnConflict field value if set, zero value otherwise.
func (o *LogsAttributeRemapper) GetOverrideOnConflict() bool {
	if o == nil || o.OverrideOnConflict == nil {
		var ret bool
		return ret
	}
	return *o.OverrideOnConflict
}

// GetOverrideOnConflictOk returns a tuple with the OverrideOnConflict field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsAttributeRemapper) GetOverrideOnConflictOk() (*bool, bool) {
	if o == nil || o.OverrideOnConflict == nil {
		return nil, false
	}
	return o.OverrideOnConflict, true
}

// HasOverrideOnConflict returns a boolean if a field has been set.
func (o *LogsAttributeRemapper) HasOverrideOnConflict() bool {
	if o != nil && o.OverrideOnConflict != nil {
		return true
	}

	return false
}

// SetOverrideOnConflict gets a reference to the given bool and assigns it to the OverrideOnConflict field.
func (o *LogsAttributeRemapper) SetOverrideOnConflict(v bool) {
	o.OverrideOnConflict = &v
}

// GetPreserveSource returns the PreserveSource field value if set, zero value otherwise.
func (o *LogsAttributeRemapper) GetPreserveSource() bool {
	if o == nil || o.PreserveSource == nil {
		var ret bool
		return ret
	}
	return *o.PreserveSource
}

// GetPreserveSourceOk returns a tuple with the PreserveSource field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsAttributeRemapper) GetPreserveSourceOk() (*bool, bool) {
	if o == nil || o.PreserveSource == nil {
		return nil, false
	}
	return o.PreserveSource, true
}

// HasPreserveSource returns a boolean if a field has been set.
func (o *LogsAttributeRemapper) HasPreserveSource() bool {
	if o != nil && o.PreserveSource != nil {
		return true
	}

	return false
}

// SetPreserveSource gets a reference to the given bool and assigns it to the PreserveSource field.
func (o *LogsAttributeRemapper) SetPreserveSource(v bool) {
	o.PreserveSource = &v
}

// GetSourceType returns the SourceType field value if set, zero value otherwise.
func (o *LogsAttributeRemapper) GetSourceType() string {
	if o == nil || o.SourceType == nil {
		var ret string
		return ret
	}
	return *o.SourceType
}

// GetSourceTypeOk returns a tuple with the SourceType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsAttributeRemapper) GetSourceTypeOk() (*string, bool) {
	if o == nil || o.SourceType == nil {
		return nil, false
	}
	return o.SourceType, true
}

// HasSourceType returns a boolean if a field has been set.
func (o *LogsAttributeRemapper) HasSourceType() bool {
	if o != nil && o.SourceType != nil {
		return true
	}

	return false
}

// SetSourceType gets a reference to the given string and assigns it to the SourceType field.
func (o *LogsAttributeRemapper) SetSourceType(v string) {
	o.SourceType = &v
}

// GetSources returns the Sources field value.
func (o *LogsAttributeRemapper) GetSources() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.Sources
}

// GetSourcesOk returns a tuple with the Sources field value
// and a boolean to check if the value has been set.
func (o *LogsAttributeRemapper) GetSourcesOk() (*[]string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Sources, true
}

// SetSources sets field value.
func (o *LogsAttributeRemapper) SetSources(v []string) {
	o.Sources = v
}

// GetTarget returns the Target field value.
func (o *LogsAttributeRemapper) GetTarget() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Target
}

// GetTargetOk returns a tuple with the Target field value
// and a boolean to check if the value has been set.
func (o *LogsAttributeRemapper) GetTargetOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Target, true
}

// SetTarget sets field value.
func (o *LogsAttributeRemapper) SetTarget(v string) {
	o.Target = v
}

// GetTargetFormat returns the TargetFormat field value if set, zero value otherwise.
func (o *LogsAttributeRemapper) GetTargetFormat() TargetFormatType {
	if o == nil || o.TargetFormat == nil {
		var ret TargetFormatType
		return ret
	}
	return *o.TargetFormat
}

// GetTargetFormatOk returns a tuple with the TargetFormat field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsAttributeRemapper) GetTargetFormatOk() (*TargetFormatType, bool) {
	if o == nil || o.TargetFormat == nil {
		return nil, false
	}
	return o.TargetFormat, true
}

// HasTargetFormat returns a boolean if a field has been set.
func (o *LogsAttributeRemapper) HasTargetFormat() bool {
	if o != nil && o.TargetFormat != nil {
		return true
	}

	return false
}

// SetTargetFormat gets a reference to the given TargetFormatType and assigns it to the TargetFormat field.
func (o *LogsAttributeRemapper) SetTargetFormat(v TargetFormatType) {
	o.TargetFormat = &v
}

// GetTargetType returns the TargetType field value if set, zero value otherwise.
func (o *LogsAttributeRemapper) GetTargetType() string {
	if o == nil || o.TargetType == nil {
		var ret string
		return ret
	}
	return *o.TargetType
}

// GetTargetTypeOk returns a tuple with the TargetType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsAttributeRemapper) GetTargetTypeOk() (*string, bool) {
	if o == nil || o.TargetType == nil {
		return nil, false
	}
	return o.TargetType, true
}

// HasTargetType returns a boolean if a field has been set.
func (o *LogsAttributeRemapper) HasTargetType() bool {
	if o != nil && o.TargetType != nil {
		return true
	}

	return false
}

// SetTargetType gets a reference to the given string and assigns it to the TargetType field.
func (o *LogsAttributeRemapper) SetTargetType(v string) {
	o.TargetType = &v
}

// GetType returns the Type field value.
func (o *LogsAttributeRemapper) GetType() LogsAttributeRemapperType {
	if o == nil {
		var ret LogsAttributeRemapperType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *LogsAttributeRemapper) GetTypeOk() (*LogsAttributeRemapperType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *LogsAttributeRemapper) SetType(v LogsAttributeRemapperType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsAttributeRemapper) MarshalJSON() ([]byte, error) {
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
	if o.OverrideOnConflict != nil {
		toSerialize["override_on_conflict"] = o.OverrideOnConflict
	}
	if o.PreserveSource != nil {
		toSerialize["preserve_source"] = o.PreserveSource
	}
	if o.SourceType != nil {
		toSerialize["source_type"] = o.SourceType
	}
	toSerialize["sources"] = o.Sources
	toSerialize["target"] = o.Target
	if o.TargetFormat != nil {
		toSerialize["target_format"] = o.TargetFormat
	}
	if o.TargetType != nil {
		toSerialize["target_type"] = o.TargetType
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsAttributeRemapper) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Sources *[]string                  `json:"sources"`
		Target  *string                    `json:"target"`
		Type    *LogsAttributeRemapperType `json:"type"`
	}{}
	all := struct {
		IsEnabled          *bool                     `json:"is_enabled,omitempty"`
		Name               *string                   `json:"name,omitempty"`
		OverrideOnConflict *bool                     `json:"override_on_conflict,omitempty"`
		PreserveSource     *bool                     `json:"preserve_source,omitempty"`
		SourceType         *string                   `json:"source_type,omitempty"`
		Sources            []string                  `json:"sources"`
		Target             string                    `json:"target"`
		TargetFormat       *TargetFormatType         `json:"target_format,omitempty"`
		TargetType         *string                   `json:"target_type,omitempty"`
		Type               LogsAttributeRemapperType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Sources == nil {
		return fmt.Errorf("Required field sources missing")
	}
	if required.Target == nil {
		return fmt.Errorf("Required field target missing")
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
	if v := all.TargetFormat; v != nil && !v.IsValid() {
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
	o.OverrideOnConflict = all.OverrideOnConflict
	o.PreserveSource = all.PreserveSource
	o.SourceType = all.SourceType
	o.Sources = all.Sources
	o.Target = all.Target
	o.TargetFormat = all.TargetFormat
	o.TargetType = all.TargetType
	o.Type = all.Type
	return nil
}
