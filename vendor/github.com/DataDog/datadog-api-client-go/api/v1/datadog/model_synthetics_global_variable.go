// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsGlobalVariable Synthetics global variable.
type SyntheticsGlobalVariable struct {
	// Attributes of the global variable.
	Attributes *SyntheticsGlobalVariableAttributes `json:"attributes,omitempty"`
	// Description of the global variable.
	Description string `json:"description"`
	// Unique identifier of the global variable.
	Id *string `json:"id,omitempty"`
	// Name of the global variable. Unique across Synthetics global variables.
	Name string `json:"name"`
	// Parser options to use for retrieving a Synthetics global variable from a Synthetics Test. Used in conjunction with `parse_test_public_id`.
	ParseTestOptions *SyntheticsGlobalVariableParseTestOptions `json:"parse_test_options,omitempty"`
	// A Synthetic test ID to use as a test to generate the variable value.
	ParseTestPublicId *string `json:"parse_test_public_id,omitempty"`
	// Tags of the global variable.
	Tags []string `json:"tags"`
	// Value of the global variable.
	Value SyntheticsGlobalVariableValue `json:"value"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsGlobalVariable instantiates a new SyntheticsGlobalVariable object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsGlobalVariable(description string, name string, tags []string, value SyntheticsGlobalVariableValue) *SyntheticsGlobalVariable {
	this := SyntheticsGlobalVariable{}
	this.Description = description
	this.Name = name
	this.Tags = tags
	this.Value = value
	return &this
}

// NewSyntheticsGlobalVariableWithDefaults instantiates a new SyntheticsGlobalVariable object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsGlobalVariableWithDefaults() *SyntheticsGlobalVariable {
	this := SyntheticsGlobalVariable{}
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *SyntheticsGlobalVariable) GetAttributes() SyntheticsGlobalVariableAttributes {
	if o == nil || o.Attributes == nil {
		var ret SyntheticsGlobalVariableAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsGlobalVariable) GetAttributesOk() (*SyntheticsGlobalVariableAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *SyntheticsGlobalVariable) HasAttributes() bool {
	if o != nil && o.Attributes != nil {
		return true
	}

	return false
}

// SetAttributes gets a reference to the given SyntheticsGlobalVariableAttributes and assigns it to the Attributes field.
func (o *SyntheticsGlobalVariable) SetAttributes(v SyntheticsGlobalVariableAttributes) {
	o.Attributes = &v
}

// GetDescription returns the Description field value.
func (o *SyntheticsGlobalVariable) GetDescription() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Description
}

// GetDescriptionOk returns a tuple with the Description field value
// and a boolean to check if the value has been set.
func (o *SyntheticsGlobalVariable) GetDescriptionOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Description, true
}

// SetDescription sets field value.
func (o *SyntheticsGlobalVariable) SetDescription(v string) {
	o.Description = v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *SyntheticsGlobalVariable) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsGlobalVariable) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *SyntheticsGlobalVariable) HasId() bool {
	if o != nil && o.Id != nil {
		return true
	}

	return false
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *SyntheticsGlobalVariable) SetId(v string) {
	o.Id = &v
}

// GetName returns the Name field value.
func (o *SyntheticsGlobalVariable) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *SyntheticsGlobalVariable) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *SyntheticsGlobalVariable) SetName(v string) {
	o.Name = v
}

// GetParseTestOptions returns the ParseTestOptions field value if set, zero value otherwise.
func (o *SyntheticsGlobalVariable) GetParseTestOptions() SyntheticsGlobalVariableParseTestOptions {
	if o == nil || o.ParseTestOptions == nil {
		var ret SyntheticsGlobalVariableParseTestOptions
		return ret
	}
	return *o.ParseTestOptions
}

// GetParseTestOptionsOk returns a tuple with the ParseTestOptions field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsGlobalVariable) GetParseTestOptionsOk() (*SyntheticsGlobalVariableParseTestOptions, bool) {
	if o == nil || o.ParseTestOptions == nil {
		return nil, false
	}
	return o.ParseTestOptions, true
}

// HasParseTestOptions returns a boolean if a field has been set.
func (o *SyntheticsGlobalVariable) HasParseTestOptions() bool {
	if o != nil && o.ParseTestOptions != nil {
		return true
	}

	return false
}

// SetParseTestOptions gets a reference to the given SyntheticsGlobalVariableParseTestOptions and assigns it to the ParseTestOptions field.
func (o *SyntheticsGlobalVariable) SetParseTestOptions(v SyntheticsGlobalVariableParseTestOptions) {
	o.ParseTestOptions = &v
}

// GetParseTestPublicId returns the ParseTestPublicId field value if set, zero value otherwise.
func (o *SyntheticsGlobalVariable) GetParseTestPublicId() string {
	if o == nil || o.ParseTestPublicId == nil {
		var ret string
		return ret
	}
	return *o.ParseTestPublicId
}

// GetParseTestPublicIdOk returns a tuple with the ParseTestPublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsGlobalVariable) GetParseTestPublicIdOk() (*string, bool) {
	if o == nil || o.ParseTestPublicId == nil {
		return nil, false
	}
	return o.ParseTestPublicId, true
}

// HasParseTestPublicId returns a boolean if a field has been set.
func (o *SyntheticsGlobalVariable) HasParseTestPublicId() bool {
	if o != nil && o.ParseTestPublicId != nil {
		return true
	}

	return false
}

// SetParseTestPublicId gets a reference to the given string and assigns it to the ParseTestPublicId field.
func (o *SyntheticsGlobalVariable) SetParseTestPublicId(v string) {
	o.ParseTestPublicId = &v
}

// GetTags returns the Tags field value.
func (o *SyntheticsGlobalVariable) GetTags() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value
// and a boolean to check if the value has been set.
func (o *SyntheticsGlobalVariable) GetTagsOk() (*[]string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Tags, true
}

// SetTags sets field value.
func (o *SyntheticsGlobalVariable) SetTags(v []string) {
	o.Tags = v
}

// GetValue returns the Value field value.
func (o *SyntheticsGlobalVariable) GetValue() SyntheticsGlobalVariableValue {
	if o == nil {
		var ret SyntheticsGlobalVariableValue
		return ret
	}
	return o.Value
}

// GetValueOk returns a tuple with the Value field value
// and a boolean to check if the value has been set.
func (o *SyntheticsGlobalVariable) GetValueOk() (*SyntheticsGlobalVariableValue, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Value, true
}

// SetValue sets field value.
func (o *SyntheticsGlobalVariable) SetValue(v SyntheticsGlobalVariableValue) {
	o.Value = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsGlobalVariable) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
	}
	toSerialize["description"] = o.Description
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
	toSerialize["name"] = o.Name
	if o.ParseTestOptions != nil {
		toSerialize["parse_test_options"] = o.ParseTestOptions
	}
	if o.ParseTestPublicId != nil {
		toSerialize["parse_test_public_id"] = o.ParseTestPublicId
	}
	toSerialize["tags"] = o.Tags
	toSerialize["value"] = o.Value

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsGlobalVariable) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Description *string                        `json:"description"`
		Name        *string                        `json:"name"`
		Tags        *[]string                      `json:"tags"`
		Value       *SyntheticsGlobalVariableValue `json:"value"`
	}{}
	all := struct {
		Attributes        *SyntheticsGlobalVariableAttributes       `json:"attributes,omitempty"`
		Description       string                                    `json:"description"`
		Id                *string                                   `json:"id,omitempty"`
		Name              string                                    `json:"name"`
		ParseTestOptions  *SyntheticsGlobalVariableParseTestOptions `json:"parse_test_options,omitempty"`
		ParseTestPublicId *string                                   `json:"parse_test_public_id,omitempty"`
		Tags              []string                                  `json:"tags"`
		Value             SyntheticsGlobalVariableValue             `json:"value"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Description == nil {
		return fmt.Errorf("Required field description missing")
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
	}
	if required.Tags == nil {
		return fmt.Errorf("Required field tags missing")
	}
	if required.Value == nil {
		return fmt.Errorf("Required field value missing")
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
	if all.Attributes != nil && all.Attributes.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Attributes = all.Attributes
	o.Description = all.Description
	o.Id = all.Id
	o.Name = all.Name
	if all.ParseTestOptions != nil && all.ParseTestOptions.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ParseTestOptions = all.ParseTestOptions
	o.ParseTestPublicId = all.ParseTestPublicId
	o.Tags = all.Tags
	if all.Value.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Value = all.Value
	return nil
}
