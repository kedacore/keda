// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// LogsCategoryProcessorCategory Object describing the logs filter.
type LogsCategoryProcessorCategory struct {
	// Filter for logs.
	Filter *LogsFilter `json:"filter,omitempty"`
	// Value to assign to the target attribute.
	Name *string `json:"name,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsCategoryProcessorCategory instantiates a new LogsCategoryProcessorCategory object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsCategoryProcessorCategory() *LogsCategoryProcessorCategory {
	this := LogsCategoryProcessorCategory{}
	return &this
}

// NewLogsCategoryProcessorCategoryWithDefaults instantiates a new LogsCategoryProcessorCategory object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsCategoryProcessorCategoryWithDefaults() *LogsCategoryProcessorCategory {
	this := LogsCategoryProcessorCategory{}
	return &this
}

// GetFilter returns the Filter field value if set, zero value otherwise.
func (o *LogsCategoryProcessorCategory) GetFilter() LogsFilter {
	if o == nil || o.Filter == nil {
		var ret LogsFilter
		return ret
	}
	return *o.Filter
}

// GetFilterOk returns a tuple with the Filter field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsCategoryProcessorCategory) GetFilterOk() (*LogsFilter, bool) {
	if o == nil || o.Filter == nil {
		return nil, false
	}
	return o.Filter, true
}

// HasFilter returns a boolean if a field has been set.
func (o *LogsCategoryProcessorCategory) HasFilter() bool {
	if o != nil && o.Filter != nil {
		return true
	}

	return false
}

// SetFilter gets a reference to the given LogsFilter and assigns it to the Filter field.
func (o *LogsCategoryProcessorCategory) SetFilter(v LogsFilter) {
	o.Filter = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *LogsCategoryProcessorCategory) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsCategoryProcessorCategory) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *LogsCategoryProcessorCategory) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *LogsCategoryProcessorCategory) SetName(v string) {
	o.Name = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsCategoryProcessorCategory) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Filter != nil {
		toSerialize["filter"] = o.Filter
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsCategoryProcessorCategory) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Filter *LogsFilter `json:"filter,omitempty"`
		Name   *string     `json:"name,omitempty"`
	}{}
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
	o.Name = all.Name
	return nil
}
