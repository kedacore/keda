// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsAssertionJSONPathTargetTarget Composed target for `validatesJSONPath` operator.
type SyntheticsAssertionJSONPathTargetTarget struct {
	// The JSON path to assert.
	JsonPath *string `json:"jsonPath,omitempty"`
	// The specific operator to use on the path.
	Operator *string `json:"operator,omitempty"`
	// The path target value to compare to.
	TargetValue interface{} `json:"targetValue,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsAssertionJSONPathTargetTarget instantiates a new SyntheticsAssertionJSONPathTargetTarget object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsAssertionJSONPathTargetTarget() *SyntheticsAssertionJSONPathTargetTarget {
	this := SyntheticsAssertionJSONPathTargetTarget{}
	return &this
}

// NewSyntheticsAssertionJSONPathTargetTargetWithDefaults instantiates a new SyntheticsAssertionJSONPathTargetTarget object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsAssertionJSONPathTargetTargetWithDefaults() *SyntheticsAssertionJSONPathTargetTarget {
	this := SyntheticsAssertionJSONPathTargetTarget{}
	return &this
}

// GetJsonPath returns the JsonPath field value if set, zero value otherwise.
func (o *SyntheticsAssertionJSONPathTargetTarget) GetJsonPath() string {
	if o == nil || o.JsonPath == nil {
		var ret string
		return ret
	}
	return *o.JsonPath
}

// GetJsonPathOk returns a tuple with the JsonPath field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAssertionJSONPathTargetTarget) GetJsonPathOk() (*string, bool) {
	if o == nil || o.JsonPath == nil {
		return nil, false
	}
	return o.JsonPath, true
}

// HasJsonPath returns a boolean if a field has been set.
func (o *SyntheticsAssertionJSONPathTargetTarget) HasJsonPath() bool {
	if o != nil && o.JsonPath != nil {
		return true
	}

	return false
}

// SetJsonPath gets a reference to the given string and assigns it to the JsonPath field.
func (o *SyntheticsAssertionJSONPathTargetTarget) SetJsonPath(v string) {
	o.JsonPath = &v
}

// GetOperator returns the Operator field value if set, zero value otherwise.
func (o *SyntheticsAssertionJSONPathTargetTarget) GetOperator() string {
	if o == nil || o.Operator == nil {
		var ret string
		return ret
	}
	return *o.Operator
}

// GetOperatorOk returns a tuple with the Operator field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAssertionJSONPathTargetTarget) GetOperatorOk() (*string, bool) {
	if o == nil || o.Operator == nil {
		return nil, false
	}
	return o.Operator, true
}

// HasOperator returns a boolean if a field has been set.
func (o *SyntheticsAssertionJSONPathTargetTarget) HasOperator() bool {
	if o != nil && o.Operator != nil {
		return true
	}

	return false
}

// SetOperator gets a reference to the given string and assigns it to the Operator field.
func (o *SyntheticsAssertionJSONPathTargetTarget) SetOperator(v string) {
	o.Operator = &v
}

// GetTargetValue returns the TargetValue field value if set, zero value otherwise.
func (o *SyntheticsAssertionJSONPathTargetTarget) GetTargetValue() interface{} {
	if o == nil || o.TargetValue == nil {
		var ret interface{}
		return ret
	}
	return o.TargetValue
}

// GetTargetValueOk returns a tuple with the TargetValue field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAssertionJSONPathTargetTarget) GetTargetValueOk() (*interface{}, bool) {
	if o == nil || o.TargetValue == nil {
		return nil, false
	}
	return &o.TargetValue, true
}

// HasTargetValue returns a boolean if a field has been set.
func (o *SyntheticsAssertionJSONPathTargetTarget) HasTargetValue() bool {
	if o != nil && o.TargetValue != nil {
		return true
	}

	return false
}

// SetTargetValue gets a reference to the given interface{} and assigns it to the TargetValue field.
func (o *SyntheticsAssertionJSONPathTargetTarget) SetTargetValue(v interface{}) {
	o.TargetValue = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsAssertionJSONPathTargetTarget) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.JsonPath != nil {
		toSerialize["jsonPath"] = o.JsonPath
	}
	if o.Operator != nil {
		toSerialize["operator"] = o.Operator
	}
	if o.TargetValue != nil {
		toSerialize["targetValue"] = o.TargetValue
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsAssertionJSONPathTargetTarget) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		JsonPath    *string     `json:"jsonPath,omitempty"`
		Operator    *string     `json:"operator,omitempty"`
		TargetValue interface{} `json:"targetValue,omitempty"`
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
	o.JsonPath = all.JsonPath
	o.Operator = all.Operator
	o.TargetValue = all.TargetValue
	return nil
}
