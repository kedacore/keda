// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsGlobalVariableParseTestOptions Parser options to use for retrieving a Synthetics global variable from a Synthetics Test. Used in conjunction with `parse_test_public_id`.
type SyntheticsGlobalVariableParseTestOptions struct {
	// When type is `http_header`, name of the header to use to extract the value.
	Field *string `json:"field,omitempty"`
	// Details of the parser to use for the global variable.
	Parser SyntheticsVariableParser `json:"parser"`
	// Property of the Synthetics Test Response to use for a Synthetics global variable.
	Type SyntheticsGlobalVariableParseTestOptionsType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsGlobalVariableParseTestOptions instantiates a new SyntheticsGlobalVariableParseTestOptions object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsGlobalVariableParseTestOptions(parser SyntheticsVariableParser, typeVar SyntheticsGlobalVariableParseTestOptionsType) *SyntheticsGlobalVariableParseTestOptions {
	this := SyntheticsGlobalVariableParseTestOptions{}
	this.Parser = parser
	this.Type = typeVar
	return &this
}

// NewSyntheticsGlobalVariableParseTestOptionsWithDefaults instantiates a new SyntheticsGlobalVariableParseTestOptions object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsGlobalVariableParseTestOptionsWithDefaults() *SyntheticsGlobalVariableParseTestOptions {
	this := SyntheticsGlobalVariableParseTestOptions{}
	return &this
}

// GetField returns the Field field value if set, zero value otherwise.
func (o *SyntheticsGlobalVariableParseTestOptions) GetField() string {
	if o == nil || o.Field == nil {
		var ret string
		return ret
	}
	return *o.Field
}

// GetFieldOk returns a tuple with the Field field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsGlobalVariableParseTestOptions) GetFieldOk() (*string, bool) {
	if o == nil || o.Field == nil {
		return nil, false
	}
	return o.Field, true
}

// HasField returns a boolean if a field has been set.
func (o *SyntheticsGlobalVariableParseTestOptions) HasField() bool {
	if o != nil && o.Field != nil {
		return true
	}

	return false
}

// SetField gets a reference to the given string and assigns it to the Field field.
func (o *SyntheticsGlobalVariableParseTestOptions) SetField(v string) {
	o.Field = &v
}

// GetParser returns the Parser field value.
func (o *SyntheticsGlobalVariableParseTestOptions) GetParser() SyntheticsVariableParser {
	if o == nil {
		var ret SyntheticsVariableParser
		return ret
	}
	return o.Parser
}

// GetParserOk returns a tuple with the Parser field value
// and a boolean to check if the value has been set.
func (o *SyntheticsGlobalVariableParseTestOptions) GetParserOk() (*SyntheticsVariableParser, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Parser, true
}

// SetParser sets field value.
func (o *SyntheticsGlobalVariableParseTestOptions) SetParser(v SyntheticsVariableParser) {
	o.Parser = v
}

// GetType returns the Type field value.
func (o *SyntheticsGlobalVariableParseTestOptions) GetType() SyntheticsGlobalVariableParseTestOptionsType {
	if o == nil {
		var ret SyntheticsGlobalVariableParseTestOptionsType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *SyntheticsGlobalVariableParseTestOptions) GetTypeOk() (*SyntheticsGlobalVariableParseTestOptionsType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *SyntheticsGlobalVariableParseTestOptions) SetType(v SyntheticsGlobalVariableParseTestOptionsType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsGlobalVariableParseTestOptions) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Field != nil {
		toSerialize["field"] = o.Field
	}
	toSerialize["parser"] = o.Parser
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsGlobalVariableParseTestOptions) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Parser *SyntheticsVariableParser                     `json:"parser"`
		Type   *SyntheticsGlobalVariableParseTestOptionsType `json:"type"`
	}{}
	all := struct {
		Field  *string                                      `json:"field,omitempty"`
		Parser SyntheticsVariableParser                     `json:"parser"`
		Type   SyntheticsGlobalVariableParseTestOptionsType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Parser == nil {
		return fmt.Errorf("Required field parser missing")
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
	o.Field = all.Field
	if all.Parser.UnparsedObject != nil && o.UnparsedObject == nil {
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
