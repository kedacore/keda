// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsTestConfig Configuration object for a Synthetic test.
type SyntheticsTestConfig struct {
	// Array of assertions used for the test. Required for single API tests.
	Assertions []SyntheticsAssertion `json:"assertions,omitempty"`
	// Array of variables used for the test.
	ConfigVariables []SyntheticsConfigVariable `json:"configVariables,omitempty"`
	// Object describing the Synthetic test request.
	Request *SyntheticsTestRequest `json:"request,omitempty"`
	// Browser tests only - array of variables used for the test steps.
	Variables []SyntheticsBrowserVariable `json:"variables,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsTestConfig instantiates a new SyntheticsTestConfig object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsTestConfig() *SyntheticsTestConfig {
	this := SyntheticsTestConfig{}
	return &this
}

// NewSyntheticsTestConfigWithDefaults instantiates a new SyntheticsTestConfig object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsTestConfigWithDefaults() *SyntheticsTestConfig {
	this := SyntheticsTestConfig{}
	return &this
}

// GetAssertions returns the Assertions field value if set, zero value otherwise.
func (o *SyntheticsTestConfig) GetAssertions() []SyntheticsAssertion {
	if o == nil || o.Assertions == nil {
		var ret []SyntheticsAssertion
		return ret
	}
	return o.Assertions
}

// GetAssertionsOk returns a tuple with the Assertions field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestConfig) GetAssertionsOk() (*[]SyntheticsAssertion, bool) {
	if o == nil || o.Assertions == nil {
		return nil, false
	}
	return &o.Assertions, true
}

// HasAssertions returns a boolean if a field has been set.
func (o *SyntheticsTestConfig) HasAssertions() bool {
	if o != nil && o.Assertions != nil {
		return true
	}

	return false
}

// SetAssertions gets a reference to the given []SyntheticsAssertion and assigns it to the Assertions field.
func (o *SyntheticsTestConfig) SetAssertions(v []SyntheticsAssertion) {
	o.Assertions = v
}

// GetConfigVariables returns the ConfigVariables field value if set, zero value otherwise.
func (o *SyntheticsTestConfig) GetConfigVariables() []SyntheticsConfigVariable {
	if o == nil || o.ConfigVariables == nil {
		var ret []SyntheticsConfigVariable
		return ret
	}
	return o.ConfigVariables
}

// GetConfigVariablesOk returns a tuple with the ConfigVariables field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestConfig) GetConfigVariablesOk() (*[]SyntheticsConfigVariable, bool) {
	if o == nil || o.ConfigVariables == nil {
		return nil, false
	}
	return &o.ConfigVariables, true
}

// HasConfigVariables returns a boolean if a field has been set.
func (o *SyntheticsTestConfig) HasConfigVariables() bool {
	if o != nil && o.ConfigVariables != nil {
		return true
	}

	return false
}

// SetConfigVariables gets a reference to the given []SyntheticsConfigVariable and assigns it to the ConfigVariables field.
func (o *SyntheticsTestConfig) SetConfigVariables(v []SyntheticsConfigVariable) {
	o.ConfigVariables = v
}

// GetRequest returns the Request field value if set, zero value otherwise.
func (o *SyntheticsTestConfig) GetRequest() SyntheticsTestRequest {
	if o == nil || o.Request == nil {
		var ret SyntheticsTestRequest
		return ret
	}
	return *o.Request
}

// GetRequestOk returns a tuple with the Request field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestConfig) GetRequestOk() (*SyntheticsTestRequest, bool) {
	if o == nil || o.Request == nil {
		return nil, false
	}
	return o.Request, true
}

// HasRequest returns a boolean if a field has been set.
func (o *SyntheticsTestConfig) HasRequest() bool {
	if o != nil && o.Request != nil {
		return true
	}

	return false
}

// SetRequest gets a reference to the given SyntheticsTestRequest and assigns it to the Request field.
func (o *SyntheticsTestConfig) SetRequest(v SyntheticsTestRequest) {
	o.Request = &v
}

// GetVariables returns the Variables field value if set, zero value otherwise.
func (o *SyntheticsTestConfig) GetVariables() []SyntheticsBrowserVariable {
	if o == nil || o.Variables == nil {
		var ret []SyntheticsBrowserVariable
		return ret
	}
	return o.Variables
}

// GetVariablesOk returns a tuple with the Variables field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestConfig) GetVariablesOk() (*[]SyntheticsBrowserVariable, bool) {
	if o == nil || o.Variables == nil {
		return nil, false
	}
	return &o.Variables, true
}

// HasVariables returns a boolean if a field has been set.
func (o *SyntheticsTestConfig) HasVariables() bool {
	if o != nil && o.Variables != nil {
		return true
	}

	return false
}

// SetVariables gets a reference to the given []SyntheticsBrowserVariable and assigns it to the Variables field.
func (o *SyntheticsTestConfig) SetVariables(v []SyntheticsBrowserVariable) {
	o.Variables = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsTestConfig) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Assertions != nil {
		toSerialize["assertions"] = o.Assertions
	}
	if o.ConfigVariables != nil {
		toSerialize["configVariables"] = o.ConfigVariables
	}
	if o.Request != nil {
		toSerialize["request"] = o.Request
	}
	if o.Variables != nil {
		toSerialize["variables"] = o.Variables
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsTestConfig) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Assertions      []SyntheticsAssertion       `json:"assertions,omitempty"`
		ConfigVariables []SyntheticsConfigVariable  `json:"configVariables,omitempty"`
		Request         *SyntheticsTestRequest      `json:"request,omitempty"`
		Variables       []SyntheticsBrowserVariable `json:"variables,omitempty"`
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
	o.Assertions = all.Assertions
	o.ConfigVariables = all.ConfigVariables
	if all.Request != nil && all.Request.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Request = all.Request
	o.Variables = all.Variables
	return nil
}
