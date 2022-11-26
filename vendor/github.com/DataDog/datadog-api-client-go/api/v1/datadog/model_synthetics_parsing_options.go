// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsParsingOptions Parsing options for variables to extract.
type SyntheticsParsingOptions struct {
	// When type is `http_header`, name of the header to use to extract the value.
	Field *string `json:"field,omitempty"`
	// Name of the variable to extract.
	Name *string `json:"name,omitempty"`
	// Details of the parser to use for the global variable.
	Parser *SyntheticsVariableParser `json:"parser,omitempty"`
	// Property of the Synthetics Test Response to use for a Synthetics global variable.
	Type *SyntheticsGlobalVariableParseTestOptionsType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsParsingOptions instantiates a new SyntheticsParsingOptions object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsParsingOptions() *SyntheticsParsingOptions {
	this := SyntheticsParsingOptions{}
	return &this
}

// NewSyntheticsParsingOptionsWithDefaults instantiates a new SyntheticsParsingOptions object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsParsingOptionsWithDefaults() *SyntheticsParsingOptions {
	this := SyntheticsParsingOptions{}
	return &this
}

// GetField returns the Field field value if set, zero value otherwise.
func (o *SyntheticsParsingOptions) GetField() string {
	if o == nil || o.Field == nil {
		var ret string
		return ret
	}
	return *o.Field
}

// GetFieldOk returns a tuple with the Field field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsParsingOptions) GetFieldOk() (*string, bool) {
	if o == nil || o.Field == nil {
		return nil, false
	}
	return o.Field, true
}

// HasField returns a boolean if a field has been set.
func (o *SyntheticsParsingOptions) HasField() bool {
	if o != nil && o.Field != nil {
		return true
	}

	return false
}

// SetField gets a reference to the given string and assigns it to the Field field.
func (o *SyntheticsParsingOptions) SetField(v string) {
	o.Field = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *SyntheticsParsingOptions) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsParsingOptions) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *SyntheticsParsingOptions) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *SyntheticsParsingOptions) SetName(v string) {
	o.Name = &v
}

// GetParser returns the Parser field value if set, zero value otherwise.
func (o *SyntheticsParsingOptions) GetParser() SyntheticsVariableParser {
	if o == nil || o.Parser == nil {
		var ret SyntheticsVariableParser
		return ret
	}
	return *o.Parser
}

// GetParserOk returns a tuple with the Parser field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsParsingOptions) GetParserOk() (*SyntheticsVariableParser, bool) {
	if o == nil || o.Parser == nil {
		return nil, false
	}
	return o.Parser, true
}

// HasParser returns a boolean if a field has been set.
func (o *SyntheticsParsingOptions) HasParser() bool {
	if o != nil && o.Parser != nil {
		return true
	}

	return false
}

// SetParser gets a reference to the given SyntheticsVariableParser and assigns it to the Parser field.
func (o *SyntheticsParsingOptions) SetParser(v SyntheticsVariableParser) {
	o.Parser = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *SyntheticsParsingOptions) GetType() SyntheticsGlobalVariableParseTestOptionsType {
	if o == nil || o.Type == nil {
		var ret SyntheticsGlobalVariableParseTestOptionsType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsParsingOptions) GetTypeOk() (*SyntheticsGlobalVariableParseTestOptionsType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *SyntheticsParsingOptions) HasType() bool {
	if o != nil && o.Type != nil {
		return true
	}

	return false
}

// SetType gets a reference to the given SyntheticsGlobalVariableParseTestOptionsType and assigns it to the Type field.
func (o *SyntheticsParsingOptions) SetType(v SyntheticsGlobalVariableParseTestOptionsType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsParsingOptions) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Field != nil {
		toSerialize["field"] = o.Field
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Parser != nil {
		toSerialize["parser"] = o.Parser
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
func (o *SyntheticsParsingOptions) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Field  *string                                       `json:"field,omitempty"`
		Name   *string                                       `json:"name,omitempty"`
		Parser *SyntheticsVariableParser                     `json:"parser,omitempty"`
		Type   *SyntheticsGlobalVariableParseTestOptionsType `json:"type,omitempty"`
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
	if v := all.Type; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Field = all.Field
	o.Name = all.Name
	if all.Parser != nil && all.Parser.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Parser = all.Parser
	o.Type = all.Type
	return nil
}
