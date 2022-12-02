// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsAPIStep The steps used in a Synthetics multistep API test.
type SyntheticsAPIStep struct {
	// Determines whether or not to continue with test if this step fails.
	AllowFailure *bool `json:"allowFailure,omitempty"`
	// Array of assertions used for the test.
	Assertions []SyntheticsAssertion `json:"assertions"`
	// Array of values to parse and save as variables from the response.
	ExtractedValues []SyntheticsParsingOptions `json:"extractedValues,omitempty"`
	// Determines whether or not to consider the entire test as failed if this step fails.
	// Can be used only if `allowFailure` is `true`.
	IsCritical *bool `json:"isCritical,omitempty"`
	// The name of the step.
	Name string `json:"name"`
	// Object describing the Synthetic test request.
	Request SyntheticsTestRequest `json:"request"`
	// Object describing the retry strategy to apply to a Synthetic test.
	Retry *SyntheticsTestOptionsRetry `json:"retry,omitempty"`
	// The subtype of the Synthetic multistep API test step, currently only supporting `http`.
	Subtype SyntheticsAPIStepSubtype `json:"subtype"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsAPIStep instantiates a new SyntheticsAPIStep object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsAPIStep(assertions []SyntheticsAssertion, name string, request SyntheticsTestRequest, subtype SyntheticsAPIStepSubtype) *SyntheticsAPIStep {
	this := SyntheticsAPIStep{}
	this.Assertions = assertions
	this.Name = name
	this.Request = request
	this.Subtype = subtype
	return &this
}

// NewSyntheticsAPIStepWithDefaults instantiates a new SyntheticsAPIStep object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsAPIStepWithDefaults() *SyntheticsAPIStep {
	this := SyntheticsAPIStep{}
	return &this
}

// GetAllowFailure returns the AllowFailure field value if set, zero value otherwise.
func (o *SyntheticsAPIStep) GetAllowFailure() bool {
	if o == nil || o.AllowFailure == nil {
		var ret bool
		return ret
	}
	return *o.AllowFailure
}

// GetAllowFailureOk returns a tuple with the AllowFailure field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPIStep) GetAllowFailureOk() (*bool, bool) {
	if o == nil || o.AllowFailure == nil {
		return nil, false
	}
	return o.AllowFailure, true
}

// HasAllowFailure returns a boolean if a field has been set.
func (o *SyntheticsAPIStep) HasAllowFailure() bool {
	if o != nil && o.AllowFailure != nil {
		return true
	}

	return false
}

// SetAllowFailure gets a reference to the given bool and assigns it to the AllowFailure field.
func (o *SyntheticsAPIStep) SetAllowFailure(v bool) {
	o.AllowFailure = &v
}

// GetAssertions returns the Assertions field value.
func (o *SyntheticsAPIStep) GetAssertions() []SyntheticsAssertion {
	if o == nil {
		var ret []SyntheticsAssertion
		return ret
	}
	return o.Assertions
}

// GetAssertionsOk returns a tuple with the Assertions field value
// and a boolean to check if the value has been set.
func (o *SyntheticsAPIStep) GetAssertionsOk() (*[]SyntheticsAssertion, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Assertions, true
}

// SetAssertions sets field value.
func (o *SyntheticsAPIStep) SetAssertions(v []SyntheticsAssertion) {
	o.Assertions = v
}

// GetExtractedValues returns the ExtractedValues field value if set, zero value otherwise.
func (o *SyntheticsAPIStep) GetExtractedValues() []SyntheticsParsingOptions {
	if o == nil || o.ExtractedValues == nil {
		var ret []SyntheticsParsingOptions
		return ret
	}
	return o.ExtractedValues
}

// GetExtractedValuesOk returns a tuple with the ExtractedValues field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPIStep) GetExtractedValuesOk() (*[]SyntheticsParsingOptions, bool) {
	if o == nil || o.ExtractedValues == nil {
		return nil, false
	}
	return &o.ExtractedValues, true
}

// HasExtractedValues returns a boolean if a field has been set.
func (o *SyntheticsAPIStep) HasExtractedValues() bool {
	if o != nil && o.ExtractedValues != nil {
		return true
	}

	return false
}

// SetExtractedValues gets a reference to the given []SyntheticsParsingOptions and assigns it to the ExtractedValues field.
func (o *SyntheticsAPIStep) SetExtractedValues(v []SyntheticsParsingOptions) {
	o.ExtractedValues = v
}

// GetIsCritical returns the IsCritical field value if set, zero value otherwise.
func (o *SyntheticsAPIStep) GetIsCritical() bool {
	if o == nil || o.IsCritical == nil {
		var ret bool
		return ret
	}
	return *o.IsCritical
}

// GetIsCriticalOk returns a tuple with the IsCritical field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPIStep) GetIsCriticalOk() (*bool, bool) {
	if o == nil || o.IsCritical == nil {
		return nil, false
	}
	return o.IsCritical, true
}

// HasIsCritical returns a boolean if a field has been set.
func (o *SyntheticsAPIStep) HasIsCritical() bool {
	if o != nil && o.IsCritical != nil {
		return true
	}

	return false
}

// SetIsCritical gets a reference to the given bool and assigns it to the IsCritical field.
func (o *SyntheticsAPIStep) SetIsCritical(v bool) {
	o.IsCritical = &v
}

// GetName returns the Name field value.
func (o *SyntheticsAPIStep) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *SyntheticsAPIStep) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *SyntheticsAPIStep) SetName(v string) {
	o.Name = v
}

// GetRequest returns the Request field value.
func (o *SyntheticsAPIStep) GetRequest() SyntheticsTestRequest {
	if o == nil {
		var ret SyntheticsTestRequest
		return ret
	}
	return o.Request
}

// GetRequestOk returns a tuple with the Request field value
// and a boolean to check if the value has been set.
func (o *SyntheticsAPIStep) GetRequestOk() (*SyntheticsTestRequest, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Request, true
}

// SetRequest sets field value.
func (o *SyntheticsAPIStep) SetRequest(v SyntheticsTestRequest) {
	o.Request = v
}

// GetRetry returns the Retry field value if set, zero value otherwise.
func (o *SyntheticsAPIStep) GetRetry() SyntheticsTestOptionsRetry {
	if o == nil || o.Retry == nil {
		var ret SyntheticsTestOptionsRetry
		return ret
	}
	return *o.Retry
}

// GetRetryOk returns a tuple with the Retry field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPIStep) GetRetryOk() (*SyntheticsTestOptionsRetry, bool) {
	if o == nil || o.Retry == nil {
		return nil, false
	}
	return o.Retry, true
}

// HasRetry returns a boolean if a field has been set.
func (o *SyntheticsAPIStep) HasRetry() bool {
	if o != nil && o.Retry != nil {
		return true
	}

	return false
}

// SetRetry gets a reference to the given SyntheticsTestOptionsRetry and assigns it to the Retry field.
func (o *SyntheticsAPIStep) SetRetry(v SyntheticsTestOptionsRetry) {
	o.Retry = &v
}

// GetSubtype returns the Subtype field value.
func (o *SyntheticsAPIStep) GetSubtype() SyntheticsAPIStepSubtype {
	if o == nil {
		var ret SyntheticsAPIStepSubtype
		return ret
	}
	return o.Subtype
}

// GetSubtypeOk returns a tuple with the Subtype field value
// and a boolean to check if the value has been set.
func (o *SyntheticsAPIStep) GetSubtypeOk() (*SyntheticsAPIStepSubtype, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Subtype, true
}

// SetSubtype sets field value.
func (o *SyntheticsAPIStep) SetSubtype(v SyntheticsAPIStepSubtype) {
	o.Subtype = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsAPIStep) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AllowFailure != nil {
		toSerialize["allowFailure"] = o.AllowFailure
	}
	toSerialize["assertions"] = o.Assertions
	if o.ExtractedValues != nil {
		toSerialize["extractedValues"] = o.ExtractedValues
	}
	if o.IsCritical != nil {
		toSerialize["isCritical"] = o.IsCritical
	}
	toSerialize["name"] = o.Name
	toSerialize["request"] = o.Request
	if o.Retry != nil {
		toSerialize["retry"] = o.Retry
	}
	toSerialize["subtype"] = o.Subtype

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsAPIStep) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Assertions *[]SyntheticsAssertion    `json:"assertions"`
		Name       *string                   `json:"name"`
		Request    *SyntheticsTestRequest    `json:"request"`
		Subtype    *SyntheticsAPIStepSubtype `json:"subtype"`
	}{}
	all := struct {
		AllowFailure    *bool                       `json:"allowFailure,omitempty"`
		Assertions      []SyntheticsAssertion       `json:"assertions"`
		ExtractedValues []SyntheticsParsingOptions  `json:"extractedValues,omitempty"`
		IsCritical      *bool                       `json:"isCritical,omitempty"`
		Name            string                      `json:"name"`
		Request         SyntheticsTestRequest       `json:"request"`
		Retry           *SyntheticsTestOptionsRetry `json:"retry,omitempty"`
		Subtype         SyntheticsAPIStepSubtype    `json:"subtype"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Assertions == nil {
		return fmt.Errorf("Required field assertions missing")
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
	}
	if required.Request == nil {
		return fmt.Errorf("Required field request missing")
	}
	if required.Subtype == nil {
		return fmt.Errorf("Required field subtype missing")
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
	if v := all.Subtype; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.AllowFailure = all.AllowFailure
	o.Assertions = all.Assertions
	o.ExtractedValues = all.ExtractedValues
	o.IsCritical = all.IsCritical
	o.Name = all.Name
	if all.Request.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Request = all.Request
	if all.Retry != nil && all.Retry.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Retry = all.Retry
	o.Subtype = all.Subtype
	return nil
}
