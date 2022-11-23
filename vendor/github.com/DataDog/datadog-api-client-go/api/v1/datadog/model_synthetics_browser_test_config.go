// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBrowserTestConfig Configuration object for a Synthetic browser test.
type SyntheticsBrowserTestConfig struct {
	// Array of assertions used for the test.
	Assertions []SyntheticsAssertion `json:"assertions"`
	// Array of variables used for the test.
	ConfigVariables []SyntheticsConfigVariable `json:"configVariables,omitempty"`
	// Object describing the Synthetic test request.
	Request SyntheticsTestRequest `json:"request"`
	// Cookies to be used for the request, using the [Set-Cookie](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie) syntax.
	SetCookie *string `json:"setCookie,omitempty"`
	// Array of variables used for the test steps.
	Variables []SyntheticsBrowserVariable `json:"variables,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsBrowserTestConfig instantiates a new SyntheticsBrowserTestConfig object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsBrowserTestConfig(assertions []SyntheticsAssertion, request SyntheticsTestRequest) *SyntheticsBrowserTestConfig {
	this := SyntheticsBrowserTestConfig{}
	this.Assertions = assertions
	this.Request = request
	return &this
}

// NewSyntheticsBrowserTestConfigWithDefaults instantiates a new SyntheticsBrowserTestConfig object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsBrowserTestConfigWithDefaults() *SyntheticsBrowserTestConfig {
	this := SyntheticsBrowserTestConfig{}
	return &this
}

// GetAssertions returns the Assertions field value.
func (o *SyntheticsBrowserTestConfig) GetAssertions() []SyntheticsAssertion {
	if o == nil {
		var ret []SyntheticsAssertion
		return ret
	}
	return o.Assertions
}

// GetAssertionsOk returns a tuple with the Assertions field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestConfig) GetAssertionsOk() (*[]SyntheticsAssertion, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Assertions, true
}

// SetAssertions sets field value.
func (o *SyntheticsBrowserTestConfig) SetAssertions(v []SyntheticsAssertion) {
	o.Assertions = v
}

// GetConfigVariables returns the ConfigVariables field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestConfig) GetConfigVariables() []SyntheticsConfigVariable {
	if o == nil || o.ConfigVariables == nil {
		var ret []SyntheticsConfigVariable
		return ret
	}
	return o.ConfigVariables
}

// GetConfigVariablesOk returns a tuple with the ConfigVariables field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestConfig) GetConfigVariablesOk() (*[]SyntheticsConfigVariable, bool) {
	if o == nil || o.ConfigVariables == nil {
		return nil, false
	}
	return &o.ConfigVariables, true
}

// HasConfigVariables returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestConfig) HasConfigVariables() bool {
	if o != nil && o.ConfigVariables != nil {
		return true
	}

	return false
}

// SetConfigVariables gets a reference to the given []SyntheticsConfigVariable and assigns it to the ConfigVariables field.
func (o *SyntheticsBrowserTestConfig) SetConfigVariables(v []SyntheticsConfigVariable) {
	o.ConfigVariables = v
}

// GetRequest returns the Request field value.
func (o *SyntheticsBrowserTestConfig) GetRequest() SyntheticsTestRequest {
	if o == nil {
		var ret SyntheticsTestRequest
		return ret
	}
	return o.Request
}

// GetRequestOk returns a tuple with the Request field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestConfig) GetRequestOk() (*SyntheticsTestRequest, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Request, true
}

// SetRequest sets field value.
func (o *SyntheticsBrowserTestConfig) SetRequest(v SyntheticsTestRequest) {
	o.Request = v
}

// GetSetCookie returns the SetCookie field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestConfig) GetSetCookie() string {
	if o == nil || o.SetCookie == nil {
		var ret string
		return ret
	}
	return *o.SetCookie
}

// GetSetCookieOk returns a tuple with the SetCookie field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestConfig) GetSetCookieOk() (*string, bool) {
	if o == nil || o.SetCookie == nil {
		return nil, false
	}
	return o.SetCookie, true
}

// HasSetCookie returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestConfig) HasSetCookie() bool {
	if o != nil && o.SetCookie != nil {
		return true
	}

	return false
}

// SetSetCookie gets a reference to the given string and assigns it to the SetCookie field.
func (o *SyntheticsBrowserTestConfig) SetSetCookie(v string) {
	o.SetCookie = &v
}

// GetVariables returns the Variables field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestConfig) GetVariables() []SyntheticsBrowserVariable {
	if o == nil || o.Variables == nil {
		var ret []SyntheticsBrowserVariable
		return ret
	}
	return o.Variables
}

// GetVariablesOk returns a tuple with the Variables field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestConfig) GetVariablesOk() (*[]SyntheticsBrowserVariable, bool) {
	if o == nil || o.Variables == nil {
		return nil, false
	}
	return &o.Variables, true
}

// HasVariables returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestConfig) HasVariables() bool {
	if o != nil && o.Variables != nil {
		return true
	}

	return false
}

// SetVariables gets a reference to the given []SyntheticsBrowserVariable and assigns it to the Variables field.
func (o *SyntheticsBrowserTestConfig) SetVariables(v []SyntheticsBrowserVariable) {
	o.Variables = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsBrowserTestConfig) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["assertions"] = o.Assertions
	if o.ConfigVariables != nil {
		toSerialize["configVariables"] = o.ConfigVariables
	}
	toSerialize["request"] = o.Request
	if o.SetCookie != nil {
		toSerialize["setCookie"] = o.SetCookie
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
func (o *SyntheticsBrowserTestConfig) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Assertions *[]SyntheticsAssertion `json:"assertions"`
		Request    *SyntheticsTestRequest `json:"request"`
	}{}
	all := struct {
		Assertions      []SyntheticsAssertion       `json:"assertions"`
		ConfigVariables []SyntheticsConfigVariable  `json:"configVariables,omitempty"`
		Request         SyntheticsTestRequest       `json:"request"`
		SetCookie       *string                     `json:"setCookie,omitempty"`
		Variables       []SyntheticsBrowserVariable `json:"variables,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Assertions == nil {
		return fmt.Errorf("Required field assertions missing")
	}
	if required.Request == nil {
		return fmt.Errorf("Required field request missing")
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
	o.Assertions = all.Assertions
	o.ConfigVariables = all.ConfigVariables
	if all.Request.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Request = all.Request
	o.SetCookie = all.SetCookie
	o.Variables = all.Variables
	return nil
}
