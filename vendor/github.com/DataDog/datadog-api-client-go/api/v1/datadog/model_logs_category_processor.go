// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsCategoryProcessor Use the Category Processor to add a new attribute (without spaces or special characters in the new attribute name)
// to a log matching a provided search query. Use categories to create groups for an analytical view.
// For example, URL groups, machine groups, environments, and response time buckets.
//
// **Notes**:
//
// - The syntax of the query is the one of Logs Explorer search bar.
//   The query can be done on any log attribute or tag, whether it is a facet or not.
//   Wildcards can also be used inside your query.
// - Once the log has matched one of the Processor queries, it stops.
//   Make sure they are properly ordered in case a log could match several queries.
// - The names of the categories must be unique.
// - Once defined in the Category Processor, you can map categories to log status using the Log Status Remapper.
type LogsCategoryProcessor struct {
	// Array of filters to match or not a log and their
	// corresponding `name` to assign a custom value to the log.
	Categories []LogsCategoryProcessorCategory `json:"categories"`
	// Whether or not the processor is enabled.
	IsEnabled *bool `json:"is_enabled,omitempty"`
	// Name of the processor.
	Name *string `json:"name,omitempty"`
	// Name of the target attribute which value is defined by the matching category.
	Target string `json:"target"`
	// Type of logs category processor.
	Type LogsCategoryProcessorType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsCategoryProcessor instantiates a new LogsCategoryProcessor object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsCategoryProcessor(categories []LogsCategoryProcessorCategory, target string, typeVar LogsCategoryProcessorType) *LogsCategoryProcessor {
	this := LogsCategoryProcessor{}
	this.Categories = categories
	var isEnabled bool = false
	this.IsEnabled = &isEnabled
	this.Target = target
	this.Type = typeVar
	return &this
}

// NewLogsCategoryProcessorWithDefaults instantiates a new LogsCategoryProcessor object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsCategoryProcessorWithDefaults() *LogsCategoryProcessor {
	this := LogsCategoryProcessor{}
	var isEnabled bool = false
	this.IsEnabled = &isEnabled
	var typeVar LogsCategoryProcessorType = LOGSCATEGORYPROCESSORTYPE_CATEGORY_PROCESSOR
	this.Type = typeVar
	return &this
}

// GetCategories returns the Categories field value.
func (o *LogsCategoryProcessor) GetCategories() []LogsCategoryProcessorCategory {
	if o == nil {
		var ret []LogsCategoryProcessorCategory
		return ret
	}
	return o.Categories
}

// GetCategoriesOk returns a tuple with the Categories field value
// and a boolean to check if the value has been set.
func (o *LogsCategoryProcessor) GetCategoriesOk() (*[]LogsCategoryProcessorCategory, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Categories, true
}

// SetCategories sets field value.
func (o *LogsCategoryProcessor) SetCategories(v []LogsCategoryProcessorCategory) {
	o.Categories = v
}

// GetIsEnabled returns the IsEnabled field value if set, zero value otherwise.
func (o *LogsCategoryProcessor) GetIsEnabled() bool {
	if o == nil || o.IsEnabled == nil {
		var ret bool
		return ret
	}
	return *o.IsEnabled
}

// GetIsEnabledOk returns a tuple with the IsEnabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsCategoryProcessor) GetIsEnabledOk() (*bool, bool) {
	if o == nil || o.IsEnabled == nil {
		return nil, false
	}
	return o.IsEnabled, true
}

// HasIsEnabled returns a boolean if a field has been set.
func (o *LogsCategoryProcessor) HasIsEnabled() bool {
	if o != nil && o.IsEnabled != nil {
		return true
	}

	return false
}

// SetIsEnabled gets a reference to the given bool and assigns it to the IsEnabled field.
func (o *LogsCategoryProcessor) SetIsEnabled(v bool) {
	o.IsEnabled = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *LogsCategoryProcessor) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsCategoryProcessor) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *LogsCategoryProcessor) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *LogsCategoryProcessor) SetName(v string) {
	o.Name = &v
}

// GetTarget returns the Target field value.
func (o *LogsCategoryProcessor) GetTarget() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Target
}

// GetTargetOk returns a tuple with the Target field value
// and a boolean to check if the value has been set.
func (o *LogsCategoryProcessor) GetTargetOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Target, true
}

// SetTarget sets field value.
func (o *LogsCategoryProcessor) SetTarget(v string) {
	o.Target = v
}

// GetType returns the Type field value.
func (o *LogsCategoryProcessor) GetType() LogsCategoryProcessorType {
	if o == nil {
		var ret LogsCategoryProcessorType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *LogsCategoryProcessor) GetTypeOk() (*LogsCategoryProcessorType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *LogsCategoryProcessor) SetType(v LogsCategoryProcessorType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsCategoryProcessor) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["categories"] = o.Categories
	if o.IsEnabled != nil {
		toSerialize["is_enabled"] = o.IsEnabled
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	toSerialize["target"] = o.Target
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsCategoryProcessor) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Categories *[]LogsCategoryProcessorCategory `json:"categories"`
		Target     *string                          `json:"target"`
		Type       *LogsCategoryProcessorType       `json:"type"`
	}{}
	all := struct {
		Categories []LogsCategoryProcessorCategory `json:"categories"`
		IsEnabled  *bool                           `json:"is_enabled,omitempty"`
		Name       *string                         `json:"name,omitempty"`
		Target     string                          `json:"target"`
		Type       LogsCategoryProcessorType       `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Categories == nil {
		return fmt.Errorf("Required field categories missing")
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
	if v := all.Type; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Categories = all.Categories
	o.IsEnabled = all.IsEnabled
	o.Name = all.Name
	o.Target = all.Target
	o.Type = all.Type
	return nil
}
