// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBrowserVariable Object defining a variable that can be used in your browser test.
// Learn more in the [Browser test Actions documentation](https://docs.datadoghq.com/synthetics/browser_tests/actions#variable).
type SyntheticsBrowserVariable struct {
	// Example for the variable.
	Example *string `json:"example,omitempty"`
	// ID for the variable. Global variables require an ID.
	Id *string `json:"id,omitempty"`
	// Name of the variable.
	Name string `json:"name"`
	// Pattern of the variable.
	Pattern *string `json:"pattern,omitempty"`
	// Type of browser test variable.
	Type SyntheticsBrowserVariableType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsBrowserVariable instantiates a new SyntheticsBrowserVariable object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsBrowserVariable(name string, typeVar SyntheticsBrowserVariableType) *SyntheticsBrowserVariable {
	this := SyntheticsBrowserVariable{}
	this.Name = name
	this.Type = typeVar
	return &this
}

// NewSyntheticsBrowserVariableWithDefaults instantiates a new SyntheticsBrowserVariable object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsBrowserVariableWithDefaults() *SyntheticsBrowserVariable {
	this := SyntheticsBrowserVariable{}
	return &this
}

// GetExample returns the Example field value if set, zero value otherwise.
func (o *SyntheticsBrowserVariable) GetExample() string {
	if o == nil || o.Example == nil {
		var ret string
		return ret
	}
	return *o.Example
}

// GetExampleOk returns a tuple with the Example field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserVariable) GetExampleOk() (*string, bool) {
	if o == nil || o.Example == nil {
		return nil, false
	}
	return o.Example, true
}

// HasExample returns a boolean if a field has been set.
func (o *SyntheticsBrowserVariable) HasExample() bool {
	if o != nil && o.Example != nil {
		return true
	}

	return false
}

// SetExample gets a reference to the given string and assigns it to the Example field.
func (o *SyntheticsBrowserVariable) SetExample(v string) {
	o.Example = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *SyntheticsBrowserVariable) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserVariable) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *SyntheticsBrowserVariable) HasId() bool {
	if o != nil && o.Id != nil {
		return true
	}

	return false
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *SyntheticsBrowserVariable) SetId(v string) {
	o.Id = &v
}

// GetName returns the Name field value.
func (o *SyntheticsBrowserVariable) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserVariable) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *SyntheticsBrowserVariable) SetName(v string) {
	o.Name = v
}

// GetPattern returns the Pattern field value if set, zero value otherwise.
func (o *SyntheticsBrowserVariable) GetPattern() string {
	if o == nil || o.Pattern == nil {
		var ret string
		return ret
	}
	return *o.Pattern
}

// GetPatternOk returns a tuple with the Pattern field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserVariable) GetPatternOk() (*string, bool) {
	if o == nil || o.Pattern == nil {
		return nil, false
	}
	return o.Pattern, true
}

// HasPattern returns a boolean if a field has been set.
func (o *SyntheticsBrowserVariable) HasPattern() bool {
	if o != nil && o.Pattern != nil {
		return true
	}

	return false
}

// SetPattern gets a reference to the given string and assigns it to the Pattern field.
func (o *SyntheticsBrowserVariable) SetPattern(v string) {
	o.Pattern = &v
}

// GetType returns the Type field value.
func (o *SyntheticsBrowserVariable) GetType() SyntheticsBrowserVariableType {
	if o == nil {
		var ret SyntheticsBrowserVariableType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserVariable) GetTypeOk() (*SyntheticsBrowserVariableType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *SyntheticsBrowserVariable) SetType(v SyntheticsBrowserVariableType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsBrowserVariable) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Example != nil {
		toSerialize["example"] = o.Example
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
	toSerialize["name"] = o.Name
	if o.Pattern != nil {
		toSerialize["pattern"] = o.Pattern
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsBrowserVariable) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Name *string                        `json:"name"`
		Type *SyntheticsBrowserVariableType `json:"type"`
	}{}
	all := struct {
		Example *string                       `json:"example,omitempty"`
		Id      *string                       `json:"id,omitempty"`
		Name    string                        `json:"name"`
		Pattern *string                       `json:"pattern,omitempty"`
		Type    SyntheticsBrowserVariableType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
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
	o.Example = all.Example
	o.Id = all.Id
	o.Name = all.Name
	o.Pattern = all.Pattern
	o.Type = all.Type
	return nil
}
