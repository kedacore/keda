// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsBrowserTestResultShortResult Object with the result of the last browser test run.
type SyntheticsBrowserTestResultShortResult struct {
	// Object describing the device used to perform the Synthetic test.
	Device *SyntheticsDevice `json:"device,omitempty"`
	// Length in milliseconds of the browser test run.
	Duration *float64 `json:"duration,omitempty"`
	// Amount of errors collected for a single browser test run.
	ErrorCount *int64 `json:"errorCount,omitempty"`
	// Amount of browser test steps completed before failing.
	StepCountCompleted *int64 `json:"stepCountCompleted,omitempty"`
	// Total amount of browser test steps.
	StepCountTotal *int64 `json:"stepCountTotal,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsBrowserTestResultShortResult instantiates a new SyntheticsBrowserTestResultShortResult object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsBrowserTestResultShortResult() *SyntheticsBrowserTestResultShortResult {
	this := SyntheticsBrowserTestResultShortResult{}
	return &this
}

// NewSyntheticsBrowserTestResultShortResultWithDefaults instantiates a new SyntheticsBrowserTestResultShortResult object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsBrowserTestResultShortResultWithDefaults() *SyntheticsBrowserTestResultShortResult {
	this := SyntheticsBrowserTestResultShortResult{}
	return &this
}

// GetDevice returns the Device field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestResultShortResult) GetDevice() SyntheticsDevice {
	if o == nil || o.Device == nil {
		var ret SyntheticsDevice
		return ret
	}
	return *o.Device
}

// GetDeviceOk returns a tuple with the Device field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestResultShortResult) GetDeviceOk() (*SyntheticsDevice, bool) {
	if o == nil || o.Device == nil {
		return nil, false
	}
	return o.Device, true
}

// HasDevice returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestResultShortResult) HasDevice() bool {
	if o != nil && o.Device != nil {
		return true
	}

	return false
}

// SetDevice gets a reference to the given SyntheticsDevice and assigns it to the Device field.
func (o *SyntheticsBrowserTestResultShortResult) SetDevice(v SyntheticsDevice) {
	o.Device = &v
}

// GetDuration returns the Duration field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestResultShortResult) GetDuration() float64 {
	if o == nil || o.Duration == nil {
		var ret float64
		return ret
	}
	return *o.Duration
}

// GetDurationOk returns a tuple with the Duration field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestResultShortResult) GetDurationOk() (*float64, bool) {
	if o == nil || o.Duration == nil {
		return nil, false
	}
	return o.Duration, true
}

// HasDuration returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestResultShortResult) HasDuration() bool {
	if o != nil && o.Duration != nil {
		return true
	}

	return false
}

// SetDuration gets a reference to the given float64 and assigns it to the Duration field.
func (o *SyntheticsBrowserTestResultShortResult) SetDuration(v float64) {
	o.Duration = &v
}

// GetErrorCount returns the ErrorCount field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestResultShortResult) GetErrorCount() int64 {
	if o == nil || o.ErrorCount == nil {
		var ret int64
		return ret
	}
	return *o.ErrorCount
}

// GetErrorCountOk returns a tuple with the ErrorCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestResultShortResult) GetErrorCountOk() (*int64, bool) {
	if o == nil || o.ErrorCount == nil {
		return nil, false
	}
	return o.ErrorCount, true
}

// HasErrorCount returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestResultShortResult) HasErrorCount() bool {
	if o != nil && o.ErrorCount != nil {
		return true
	}

	return false
}

// SetErrorCount gets a reference to the given int64 and assigns it to the ErrorCount field.
func (o *SyntheticsBrowserTestResultShortResult) SetErrorCount(v int64) {
	o.ErrorCount = &v
}

// GetStepCountCompleted returns the StepCountCompleted field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestResultShortResult) GetStepCountCompleted() int64 {
	if o == nil || o.StepCountCompleted == nil {
		var ret int64
		return ret
	}
	return *o.StepCountCompleted
}

// GetStepCountCompletedOk returns a tuple with the StepCountCompleted field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestResultShortResult) GetStepCountCompletedOk() (*int64, bool) {
	if o == nil || o.StepCountCompleted == nil {
		return nil, false
	}
	return o.StepCountCompleted, true
}

// HasStepCountCompleted returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestResultShortResult) HasStepCountCompleted() bool {
	if o != nil && o.StepCountCompleted != nil {
		return true
	}

	return false
}

// SetStepCountCompleted gets a reference to the given int64 and assigns it to the StepCountCompleted field.
func (o *SyntheticsBrowserTestResultShortResult) SetStepCountCompleted(v int64) {
	o.StepCountCompleted = &v
}

// GetStepCountTotal returns the StepCountTotal field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestResultShortResult) GetStepCountTotal() int64 {
	if o == nil || o.StepCountTotal == nil {
		var ret int64
		return ret
	}
	return *o.StepCountTotal
}

// GetStepCountTotalOk returns a tuple with the StepCountTotal field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestResultShortResult) GetStepCountTotalOk() (*int64, bool) {
	if o == nil || o.StepCountTotal == nil {
		return nil, false
	}
	return o.StepCountTotal, true
}

// HasStepCountTotal returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestResultShortResult) HasStepCountTotal() bool {
	if o != nil && o.StepCountTotal != nil {
		return true
	}

	return false
}

// SetStepCountTotal gets a reference to the given int64 and assigns it to the StepCountTotal field.
func (o *SyntheticsBrowserTestResultShortResult) SetStepCountTotal(v int64) {
	o.StepCountTotal = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsBrowserTestResultShortResult) MarshalJSON() ([]byte, error) {
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
	if o.ErrorCount != nil {
		toSerialize["errorCount"] = o.ErrorCount
	}
	if o.StepCountCompleted != nil {
		toSerialize["stepCountCompleted"] = o.StepCountCompleted
	}
	if o.StepCountTotal != nil {
		toSerialize["stepCountTotal"] = o.StepCountTotal
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsBrowserTestResultShortResult) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Device             *SyntheticsDevice `json:"device,omitempty"`
		Duration           *float64          `json:"duration,omitempty"`
		ErrorCount         *int64            `json:"errorCount,omitempty"`
		StepCountCompleted *int64            `json:"stepCountCompleted,omitempty"`
		StepCountTotal     *int64            `json:"stepCountTotal,omitempty"`
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
	if all.Device != nil && all.Device.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Device = all.Device
	o.Duration = all.Duration
	o.ErrorCount = all.ErrorCount
	o.StepCountCompleted = all.StepCountCompleted
	o.StepCountTotal = all.StepCountTotal
	return nil
}
