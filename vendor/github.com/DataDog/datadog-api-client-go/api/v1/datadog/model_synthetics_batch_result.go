// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsBatchResult Object with the results of a Synthetics batch.
type SyntheticsBatchResult struct {
	// The device ID.
	Device *SyntheticsDeviceID `json:"device,omitempty"`
	// Total duration in millisecond of the test.
	Duration *float64 `json:"duration,omitempty"`
	// Execution rule for a Synthetics test.
	ExecutionRule *SyntheticsTestExecutionRule `json:"execution_rule,omitempty"`
	// Name of the location.
	Location *string `json:"location,omitempty"`
	// The ID of the result to get.
	ResultId *string `json:"result_id,omitempty"`
	// Number of times this result has been retried.
	Retries *float64 `json:"retries,omitempty"`
	// Determines whether or not the batch has passed, failed, or is in progress.
	Status *SyntheticsStatus `json:"status,omitempty"`
	// Name of the test.
	TestName *string `json:"test_name,omitempty"`
	// The public ID of the Synthetic test.
	TestPublicId *string `json:"test_public_id,omitempty"`
	// Type of the Synthetic test, either `api` or `browser`.
	TestType *SyntheticsTestDetailsType `json:"test_type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsBatchResult instantiates a new SyntheticsBatchResult object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsBatchResult() *SyntheticsBatchResult {
	this := SyntheticsBatchResult{}
	return &this
}

// NewSyntheticsBatchResultWithDefaults instantiates a new SyntheticsBatchResult object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsBatchResultWithDefaults() *SyntheticsBatchResult {
	this := SyntheticsBatchResult{}
	return &this
}

// GetDevice returns the Device field value if set, zero value otherwise.
func (o *SyntheticsBatchResult) GetDevice() SyntheticsDeviceID {
	if o == nil || o.Device == nil {
		var ret SyntheticsDeviceID
		return ret
	}
	return *o.Device
}

// GetDeviceOk returns a tuple with the Device field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBatchResult) GetDeviceOk() (*SyntheticsDeviceID, bool) {
	if o == nil || o.Device == nil {
		return nil, false
	}
	return o.Device, true
}

// HasDevice returns a boolean if a field has been set.
func (o *SyntheticsBatchResult) HasDevice() bool {
	if o != nil && o.Device != nil {
		return true
	}

	return false
}

// SetDevice gets a reference to the given SyntheticsDeviceID and assigns it to the Device field.
func (o *SyntheticsBatchResult) SetDevice(v SyntheticsDeviceID) {
	o.Device = &v
}

// GetDuration returns the Duration field value if set, zero value otherwise.
func (o *SyntheticsBatchResult) GetDuration() float64 {
	if o == nil || o.Duration == nil {
		var ret float64
		return ret
	}
	return *o.Duration
}

// GetDurationOk returns a tuple with the Duration field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBatchResult) GetDurationOk() (*float64, bool) {
	if o == nil || o.Duration == nil {
		return nil, false
	}
	return o.Duration, true
}

// HasDuration returns a boolean if a field has been set.
func (o *SyntheticsBatchResult) HasDuration() bool {
	if o != nil && o.Duration != nil {
		return true
	}

	return false
}

// SetDuration gets a reference to the given float64 and assigns it to the Duration field.
func (o *SyntheticsBatchResult) SetDuration(v float64) {
	o.Duration = &v
}

// GetExecutionRule returns the ExecutionRule field value if set, zero value otherwise.
func (o *SyntheticsBatchResult) GetExecutionRule() SyntheticsTestExecutionRule {
	if o == nil || o.ExecutionRule == nil {
		var ret SyntheticsTestExecutionRule
		return ret
	}
	return *o.ExecutionRule
}

// GetExecutionRuleOk returns a tuple with the ExecutionRule field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBatchResult) GetExecutionRuleOk() (*SyntheticsTestExecutionRule, bool) {
	if o == nil || o.ExecutionRule == nil {
		return nil, false
	}
	return o.ExecutionRule, true
}

// HasExecutionRule returns a boolean if a field has been set.
func (o *SyntheticsBatchResult) HasExecutionRule() bool {
	if o != nil && o.ExecutionRule != nil {
		return true
	}

	return false
}

// SetExecutionRule gets a reference to the given SyntheticsTestExecutionRule and assigns it to the ExecutionRule field.
func (o *SyntheticsBatchResult) SetExecutionRule(v SyntheticsTestExecutionRule) {
	o.ExecutionRule = &v
}

// GetLocation returns the Location field value if set, zero value otherwise.
func (o *SyntheticsBatchResult) GetLocation() string {
	if o == nil || o.Location == nil {
		var ret string
		return ret
	}
	return *o.Location
}

// GetLocationOk returns a tuple with the Location field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBatchResult) GetLocationOk() (*string, bool) {
	if o == nil || o.Location == nil {
		return nil, false
	}
	return o.Location, true
}

// HasLocation returns a boolean if a field has been set.
func (o *SyntheticsBatchResult) HasLocation() bool {
	if o != nil && o.Location != nil {
		return true
	}

	return false
}

// SetLocation gets a reference to the given string and assigns it to the Location field.
func (o *SyntheticsBatchResult) SetLocation(v string) {
	o.Location = &v
}

// GetResultId returns the ResultId field value if set, zero value otherwise.
func (o *SyntheticsBatchResult) GetResultId() string {
	if o == nil || o.ResultId == nil {
		var ret string
		return ret
	}
	return *o.ResultId
}

// GetResultIdOk returns a tuple with the ResultId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBatchResult) GetResultIdOk() (*string, bool) {
	if o == nil || o.ResultId == nil {
		return nil, false
	}
	return o.ResultId, true
}

// HasResultId returns a boolean if a field has been set.
func (o *SyntheticsBatchResult) HasResultId() bool {
	if o != nil && o.ResultId != nil {
		return true
	}

	return false
}

// SetResultId gets a reference to the given string and assigns it to the ResultId field.
func (o *SyntheticsBatchResult) SetResultId(v string) {
	o.ResultId = &v
}

// GetRetries returns the Retries field value if set, zero value otherwise.
func (o *SyntheticsBatchResult) GetRetries() float64 {
	if o == nil || o.Retries == nil {
		var ret float64
		return ret
	}
	return *o.Retries
}

// GetRetriesOk returns a tuple with the Retries field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBatchResult) GetRetriesOk() (*float64, bool) {
	if o == nil || o.Retries == nil {
		return nil, false
	}
	return o.Retries, true
}

// HasRetries returns a boolean if a field has been set.
func (o *SyntheticsBatchResult) HasRetries() bool {
	if o != nil && o.Retries != nil {
		return true
	}

	return false
}

// SetRetries gets a reference to the given float64 and assigns it to the Retries field.
func (o *SyntheticsBatchResult) SetRetries(v float64) {
	o.Retries = &v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *SyntheticsBatchResult) GetStatus() SyntheticsStatus {
	if o == nil || o.Status == nil {
		var ret SyntheticsStatus
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBatchResult) GetStatusOk() (*SyntheticsStatus, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *SyntheticsBatchResult) HasStatus() bool {
	if o != nil && o.Status != nil {
		return true
	}

	return false
}

// SetStatus gets a reference to the given SyntheticsStatus and assigns it to the Status field.
func (o *SyntheticsBatchResult) SetStatus(v SyntheticsStatus) {
	o.Status = &v
}

// GetTestName returns the TestName field value if set, zero value otherwise.
func (o *SyntheticsBatchResult) GetTestName() string {
	if o == nil || o.TestName == nil {
		var ret string
		return ret
	}
	return *o.TestName
}

// GetTestNameOk returns a tuple with the TestName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBatchResult) GetTestNameOk() (*string, bool) {
	if o == nil || o.TestName == nil {
		return nil, false
	}
	return o.TestName, true
}

// HasTestName returns a boolean if a field has been set.
func (o *SyntheticsBatchResult) HasTestName() bool {
	if o != nil && o.TestName != nil {
		return true
	}

	return false
}

// SetTestName gets a reference to the given string and assigns it to the TestName field.
func (o *SyntheticsBatchResult) SetTestName(v string) {
	o.TestName = &v
}

// GetTestPublicId returns the TestPublicId field value if set, zero value otherwise.
func (o *SyntheticsBatchResult) GetTestPublicId() string {
	if o == nil || o.TestPublicId == nil {
		var ret string
		return ret
	}
	return *o.TestPublicId
}

// GetTestPublicIdOk returns a tuple with the TestPublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBatchResult) GetTestPublicIdOk() (*string, bool) {
	if o == nil || o.TestPublicId == nil {
		return nil, false
	}
	return o.TestPublicId, true
}

// HasTestPublicId returns a boolean if a field has been set.
func (o *SyntheticsBatchResult) HasTestPublicId() bool {
	if o != nil && o.TestPublicId != nil {
		return true
	}

	return false
}

// SetTestPublicId gets a reference to the given string and assigns it to the TestPublicId field.
func (o *SyntheticsBatchResult) SetTestPublicId(v string) {
	o.TestPublicId = &v
}

// GetTestType returns the TestType field value if set, zero value otherwise.
func (o *SyntheticsBatchResult) GetTestType() SyntheticsTestDetailsType {
	if o == nil || o.TestType == nil {
		var ret SyntheticsTestDetailsType
		return ret
	}
	return *o.TestType
}

// GetTestTypeOk returns a tuple with the TestType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBatchResult) GetTestTypeOk() (*SyntheticsTestDetailsType, bool) {
	if o == nil || o.TestType == nil {
		return nil, false
	}
	return o.TestType, true
}

// HasTestType returns a boolean if a field has been set.
func (o *SyntheticsBatchResult) HasTestType() bool {
	if o != nil && o.TestType != nil {
		return true
	}

	return false
}

// SetTestType gets a reference to the given SyntheticsTestDetailsType and assigns it to the TestType field.
func (o *SyntheticsBatchResult) SetTestType(v SyntheticsTestDetailsType) {
	o.TestType = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsBatchResult) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Device != nil {
		toSerialize["device"] = o.Device
	}
	if o.Duration != nil {
		toSerialize["duration"] = o.Duration
	}
	if o.ExecutionRule != nil {
		toSerialize["execution_rule"] = o.ExecutionRule
	}
	if o.Location != nil {
		toSerialize["location"] = o.Location
	}
	if o.ResultId != nil {
		toSerialize["result_id"] = o.ResultId
	}
	if o.Retries != nil {
		toSerialize["retries"] = o.Retries
	}
	if o.Status != nil {
		toSerialize["status"] = o.Status
	}
	if o.TestName != nil {
		toSerialize["test_name"] = o.TestName
	}
	if o.TestPublicId != nil {
		toSerialize["test_public_id"] = o.TestPublicId
	}
	if o.TestType != nil {
		toSerialize["test_type"] = o.TestType
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsBatchResult) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Device        *SyntheticsDeviceID          `json:"device,omitempty"`
		Duration      *float64                     `json:"duration,omitempty"`
		ExecutionRule *SyntheticsTestExecutionRule `json:"execution_rule,omitempty"`
		Location      *string                      `json:"location,omitempty"`
		ResultId      *string                      `json:"result_id,omitempty"`
		Retries       *float64                     `json:"retries,omitempty"`
		Status        *SyntheticsStatus            `json:"status,omitempty"`
		TestName      *string                      `json:"test_name,omitempty"`
		TestPublicId  *string                      `json:"test_public_id,omitempty"`
		TestType      *SyntheticsTestDetailsType   `json:"test_type,omitempty"`
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
	if v := all.Device; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.ExecutionRule; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Status; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.TestType; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Device = all.Device
	o.Duration = all.Duration
	o.ExecutionRule = all.ExecutionRule
	o.Location = all.Location
	o.ResultId = all.ResultId
	o.Retries = all.Retries
	o.Status = all.Status
	o.TestName = all.TestName
	o.TestPublicId = all.TestPublicId
	o.TestType = all.TestType
	return nil
}
