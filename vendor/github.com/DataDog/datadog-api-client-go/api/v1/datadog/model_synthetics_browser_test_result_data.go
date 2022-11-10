// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsBrowserTestResultData Object containing results for your Synthetic browser test.
type SyntheticsBrowserTestResultData struct {
	// Type of browser device used for the browser test.
	BrowserType *string `json:"browserType,omitempty"`
	// Browser version used for the browser test.
	BrowserVersion *string `json:"browserVersion,omitempty"`
	// Object describing the device used to perform the Synthetic test.
	Device *SyntheticsDevice `json:"device,omitempty"`
	// Global duration in second of the browser test.
	Duration *float64 `json:"duration,omitempty"`
	// Error returned for the browser test.
	Error *string `json:"error,omitempty"`
	// The browser test failure details.
	Failure *SyntheticsBrowserTestResultFailure `json:"failure,omitempty"`
	// Whether or not the browser test was conducted.
	Passed *bool `json:"passed,omitempty"`
	// The amount of email received during the browser test.
	ReceivedEmailCount *int64 `json:"receivedEmailCount,omitempty"`
	// Starting URL for the browser test.
	StartUrl *string `json:"startUrl,omitempty"`
	// Array containing the different browser test steps.
	StepDetails []SyntheticsStepDetail `json:"stepDetails,omitempty"`
	// Whether or not a thumbnail is associated with the browser test.
	ThumbnailsBucketKey *bool `json:"thumbnailsBucketKey,omitempty"`
	// Time in second to wait before the browser test starts after
	// reaching the start URL.
	TimeToInteractive *float64 `json:"timeToInteractive,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsBrowserTestResultData instantiates a new SyntheticsBrowserTestResultData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsBrowserTestResultData() *SyntheticsBrowserTestResultData {
	this := SyntheticsBrowserTestResultData{}
	return &this
}

// NewSyntheticsBrowserTestResultDataWithDefaults instantiates a new SyntheticsBrowserTestResultData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsBrowserTestResultDataWithDefaults() *SyntheticsBrowserTestResultData {
	this := SyntheticsBrowserTestResultData{}
	return &this
}

// GetBrowserType returns the BrowserType field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestResultData) GetBrowserType() string {
	if o == nil || o.BrowserType == nil {
		var ret string
		return ret
	}
	return *o.BrowserType
}

// GetBrowserTypeOk returns a tuple with the BrowserType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestResultData) GetBrowserTypeOk() (*string, bool) {
	if o == nil || o.BrowserType == nil {
		return nil, false
	}
	return o.BrowserType, true
}

// HasBrowserType returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestResultData) HasBrowserType() bool {
	if o != nil && o.BrowserType != nil {
		return true
	}

	return false
}

// SetBrowserType gets a reference to the given string and assigns it to the BrowserType field.
func (o *SyntheticsBrowserTestResultData) SetBrowserType(v string) {
	o.BrowserType = &v
}

// GetBrowserVersion returns the BrowserVersion field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestResultData) GetBrowserVersion() string {
	if o == nil || o.BrowserVersion == nil {
		var ret string
		return ret
	}
	return *o.BrowserVersion
}

// GetBrowserVersionOk returns a tuple with the BrowserVersion field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestResultData) GetBrowserVersionOk() (*string, bool) {
	if o == nil || o.BrowserVersion == nil {
		return nil, false
	}
	return o.BrowserVersion, true
}

// HasBrowserVersion returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestResultData) HasBrowserVersion() bool {
	if o != nil && o.BrowserVersion != nil {
		return true
	}

	return false
}

// SetBrowserVersion gets a reference to the given string and assigns it to the BrowserVersion field.
func (o *SyntheticsBrowserTestResultData) SetBrowserVersion(v string) {
	o.BrowserVersion = &v
}

// GetDevice returns the Device field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestResultData) GetDevice() SyntheticsDevice {
	if o == nil || o.Device == nil {
		var ret SyntheticsDevice
		return ret
	}
	return *o.Device
}

// GetDeviceOk returns a tuple with the Device field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestResultData) GetDeviceOk() (*SyntheticsDevice, bool) {
	if o == nil || o.Device == nil {
		return nil, false
	}
	return o.Device, true
}

// HasDevice returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestResultData) HasDevice() bool {
	if o != nil && o.Device != nil {
		return true
	}

	return false
}

// SetDevice gets a reference to the given SyntheticsDevice and assigns it to the Device field.
func (o *SyntheticsBrowserTestResultData) SetDevice(v SyntheticsDevice) {
	o.Device = &v
}

// GetDuration returns the Duration field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestResultData) GetDuration() float64 {
	if o == nil || o.Duration == nil {
		var ret float64
		return ret
	}
	return *o.Duration
}

// GetDurationOk returns a tuple with the Duration field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestResultData) GetDurationOk() (*float64, bool) {
	if o == nil || o.Duration == nil {
		return nil, false
	}
	return o.Duration, true
}

// HasDuration returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestResultData) HasDuration() bool {
	if o != nil && o.Duration != nil {
		return true
	}

	return false
}

// SetDuration gets a reference to the given float64 and assigns it to the Duration field.
func (o *SyntheticsBrowserTestResultData) SetDuration(v float64) {
	o.Duration = &v
}

// GetError returns the Error field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestResultData) GetError() string {
	if o == nil || o.Error == nil {
		var ret string
		return ret
	}
	return *o.Error
}

// GetErrorOk returns a tuple with the Error field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestResultData) GetErrorOk() (*string, bool) {
	if o == nil || o.Error == nil {
		return nil, false
	}
	return o.Error, true
}

// HasError returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestResultData) HasError() bool {
	if o != nil && o.Error != nil {
		return true
	}

	return false
}

// SetError gets a reference to the given string and assigns it to the Error field.
func (o *SyntheticsBrowserTestResultData) SetError(v string) {
	o.Error = &v
}

// GetFailure returns the Failure field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestResultData) GetFailure() SyntheticsBrowserTestResultFailure {
	if o == nil || o.Failure == nil {
		var ret SyntheticsBrowserTestResultFailure
		return ret
	}
	return *o.Failure
}

// GetFailureOk returns a tuple with the Failure field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestResultData) GetFailureOk() (*SyntheticsBrowserTestResultFailure, bool) {
	if o == nil || o.Failure == nil {
		return nil, false
	}
	return o.Failure, true
}

// HasFailure returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestResultData) HasFailure() bool {
	if o != nil && o.Failure != nil {
		return true
	}

	return false
}

// SetFailure gets a reference to the given SyntheticsBrowserTestResultFailure and assigns it to the Failure field.
func (o *SyntheticsBrowserTestResultData) SetFailure(v SyntheticsBrowserTestResultFailure) {
	o.Failure = &v
}

// GetPassed returns the Passed field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestResultData) GetPassed() bool {
	if o == nil || o.Passed == nil {
		var ret bool
		return ret
	}
	return *o.Passed
}

// GetPassedOk returns a tuple with the Passed field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestResultData) GetPassedOk() (*bool, bool) {
	if o == nil || o.Passed == nil {
		return nil, false
	}
	return o.Passed, true
}

// HasPassed returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestResultData) HasPassed() bool {
	if o != nil && o.Passed != nil {
		return true
	}

	return false
}

// SetPassed gets a reference to the given bool and assigns it to the Passed field.
func (o *SyntheticsBrowserTestResultData) SetPassed(v bool) {
	o.Passed = &v
}

// GetReceivedEmailCount returns the ReceivedEmailCount field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestResultData) GetReceivedEmailCount() int64 {
	if o == nil || o.ReceivedEmailCount == nil {
		var ret int64
		return ret
	}
	return *o.ReceivedEmailCount
}

// GetReceivedEmailCountOk returns a tuple with the ReceivedEmailCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestResultData) GetReceivedEmailCountOk() (*int64, bool) {
	if o == nil || o.ReceivedEmailCount == nil {
		return nil, false
	}
	return o.ReceivedEmailCount, true
}

// HasReceivedEmailCount returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestResultData) HasReceivedEmailCount() bool {
	if o != nil && o.ReceivedEmailCount != nil {
		return true
	}

	return false
}

// SetReceivedEmailCount gets a reference to the given int64 and assigns it to the ReceivedEmailCount field.
func (o *SyntheticsBrowserTestResultData) SetReceivedEmailCount(v int64) {
	o.ReceivedEmailCount = &v
}

// GetStartUrl returns the StartUrl field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestResultData) GetStartUrl() string {
	if o == nil || o.StartUrl == nil {
		var ret string
		return ret
	}
	return *o.StartUrl
}

// GetStartUrlOk returns a tuple with the StartUrl field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestResultData) GetStartUrlOk() (*string, bool) {
	if o == nil || o.StartUrl == nil {
		return nil, false
	}
	return o.StartUrl, true
}

// HasStartUrl returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestResultData) HasStartUrl() bool {
	if o != nil && o.StartUrl != nil {
		return true
	}

	return false
}

// SetStartUrl gets a reference to the given string and assigns it to the StartUrl field.
func (o *SyntheticsBrowserTestResultData) SetStartUrl(v string) {
	o.StartUrl = &v
}

// GetStepDetails returns the StepDetails field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestResultData) GetStepDetails() []SyntheticsStepDetail {
	if o == nil || o.StepDetails == nil {
		var ret []SyntheticsStepDetail
		return ret
	}
	return o.StepDetails
}

// GetStepDetailsOk returns a tuple with the StepDetails field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestResultData) GetStepDetailsOk() (*[]SyntheticsStepDetail, bool) {
	if o == nil || o.StepDetails == nil {
		return nil, false
	}
	return &o.StepDetails, true
}

// HasStepDetails returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestResultData) HasStepDetails() bool {
	if o != nil && o.StepDetails != nil {
		return true
	}

	return false
}

// SetStepDetails gets a reference to the given []SyntheticsStepDetail and assigns it to the StepDetails field.
func (o *SyntheticsBrowserTestResultData) SetStepDetails(v []SyntheticsStepDetail) {
	o.StepDetails = v
}

// GetThumbnailsBucketKey returns the ThumbnailsBucketKey field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestResultData) GetThumbnailsBucketKey() bool {
	if o == nil || o.ThumbnailsBucketKey == nil {
		var ret bool
		return ret
	}
	return *o.ThumbnailsBucketKey
}

// GetThumbnailsBucketKeyOk returns a tuple with the ThumbnailsBucketKey field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestResultData) GetThumbnailsBucketKeyOk() (*bool, bool) {
	if o == nil || o.ThumbnailsBucketKey == nil {
		return nil, false
	}
	return o.ThumbnailsBucketKey, true
}

// HasThumbnailsBucketKey returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestResultData) HasThumbnailsBucketKey() bool {
	if o != nil && o.ThumbnailsBucketKey != nil {
		return true
	}

	return false
}

// SetThumbnailsBucketKey gets a reference to the given bool and assigns it to the ThumbnailsBucketKey field.
func (o *SyntheticsBrowserTestResultData) SetThumbnailsBucketKey(v bool) {
	o.ThumbnailsBucketKey = &v
}

// GetTimeToInteractive returns the TimeToInteractive field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestResultData) GetTimeToInteractive() float64 {
	if o == nil || o.TimeToInteractive == nil {
		var ret float64
		return ret
	}
	return *o.TimeToInteractive
}

// GetTimeToInteractiveOk returns a tuple with the TimeToInteractive field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestResultData) GetTimeToInteractiveOk() (*float64, bool) {
	if o == nil || o.TimeToInteractive == nil {
		return nil, false
	}
	return o.TimeToInteractive, true
}

// HasTimeToInteractive returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestResultData) HasTimeToInteractive() bool {
	if o != nil && o.TimeToInteractive != nil {
		return true
	}

	return false
}

// SetTimeToInteractive gets a reference to the given float64 and assigns it to the TimeToInteractive field.
func (o *SyntheticsBrowserTestResultData) SetTimeToInteractive(v float64) {
	o.TimeToInteractive = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsBrowserTestResultData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.BrowserType != nil {
		toSerialize["browserType"] = o.BrowserType
	}
	if o.BrowserVersion != nil {
		toSerialize["browserVersion"] = o.BrowserVersion
	}
	if o.Device != nil {
		toSerialize["device"] = o.Device
	}
	if o.Duration != nil {
		toSerialize["duration"] = o.Duration
	}
	if o.Error != nil {
		toSerialize["error"] = o.Error
	}
	if o.Failure != nil {
		toSerialize["failure"] = o.Failure
	}
	if o.Passed != nil {
		toSerialize["passed"] = o.Passed
	}
	if o.ReceivedEmailCount != nil {
		toSerialize["receivedEmailCount"] = o.ReceivedEmailCount
	}
	if o.StartUrl != nil {
		toSerialize["startUrl"] = o.StartUrl
	}
	if o.StepDetails != nil {
		toSerialize["stepDetails"] = o.StepDetails
	}
	if o.ThumbnailsBucketKey != nil {
		toSerialize["thumbnailsBucketKey"] = o.ThumbnailsBucketKey
	}
	if o.TimeToInteractive != nil {
		toSerialize["timeToInteractive"] = o.TimeToInteractive
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsBrowserTestResultData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		BrowserType         *string                             `json:"browserType,omitempty"`
		BrowserVersion      *string                             `json:"browserVersion,omitempty"`
		Device              *SyntheticsDevice                   `json:"device,omitempty"`
		Duration            *float64                            `json:"duration,omitempty"`
		Error               *string                             `json:"error,omitempty"`
		Failure             *SyntheticsBrowserTestResultFailure `json:"failure,omitempty"`
		Passed              *bool                               `json:"passed,omitempty"`
		ReceivedEmailCount  *int64                              `json:"receivedEmailCount,omitempty"`
		StartUrl            *string                             `json:"startUrl,omitempty"`
		StepDetails         []SyntheticsStepDetail              `json:"stepDetails,omitempty"`
		ThumbnailsBucketKey *bool                               `json:"thumbnailsBucketKey,omitempty"`
		TimeToInteractive   *float64                            `json:"timeToInteractive,omitempty"`
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
	o.BrowserType = all.BrowserType
	o.BrowserVersion = all.BrowserVersion
	if all.Device != nil && all.Device.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Device = all.Device
	o.Duration = all.Duration
	o.Error = all.Error
	if all.Failure != nil && all.Failure.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Failure = all.Failure
	o.Passed = all.Passed
	o.ReceivedEmailCount = all.ReceivedEmailCount
	o.StartUrl = all.StartUrl
	o.StepDetails = all.StepDetails
	o.ThumbnailsBucketKey = all.ThumbnailsBucketKey
	o.TimeToInteractive = all.TimeToInteractive
	return nil
}
