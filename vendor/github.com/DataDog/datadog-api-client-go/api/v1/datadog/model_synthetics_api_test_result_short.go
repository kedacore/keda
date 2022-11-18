// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsAPITestResultShort Object with the results of a single Synthetic API test.
type SyntheticsAPITestResultShort struct {
	// Last time the API test was performed.
	CheckTime *float64 `json:"check_time,omitempty"`
	// Location from which the API test was performed.
	ProbeDc *string `json:"probe_dc,omitempty"`
	// Result of the last API test run.
	Result *SyntheticsAPITestResultShortResult `json:"result,omitempty"`
	// ID of the API test result.
	ResultId *string `json:"result_id,omitempty"`
	// The status of your Synthetic monitor.
	// * `O` for not triggered
	// * `1` for triggered
	// * `2` for no data
	Status *SyntheticsTestMonitorStatus `json:"status,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsAPITestResultShort instantiates a new SyntheticsAPITestResultShort object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsAPITestResultShort() *SyntheticsAPITestResultShort {
	this := SyntheticsAPITestResultShort{}
	return &this
}

// NewSyntheticsAPITestResultShortWithDefaults instantiates a new SyntheticsAPITestResultShort object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsAPITestResultShortWithDefaults() *SyntheticsAPITestResultShort {
	this := SyntheticsAPITestResultShort{}
	return &this
}

// GetCheckTime returns the CheckTime field value if set, zero value otherwise.
func (o *SyntheticsAPITestResultShort) GetCheckTime() float64 {
	if o == nil || o.CheckTime == nil {
		var ret float64
		return ret
	}
	return *o.CheckTime
}

// GetCheckTimeOk returns a tuple with the CheckTime field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITestResultShort) GetCheckTimeOk() (*float64, bool) {
	if o == nil || o.CheckTime == nil {
		return nil, false
	}
	return o.CheckTime, true
}

// HasCheckTime returns a boolean if a field has been set.
func (o *SyntheticsAPITestResultShort) HasCheckTime() bool {
	if o != nil && o.CheckTime != nil {
		return true
	}

	return false
}

// SetCheckTime gets a reference to the given float64 and assigns it to the CheckTime field.
func (o *SyntheticsAPITestResultShort) SetCheckTime(v float64) {
	o.CheckTime = &v
}

// GetProbeDc returns the ProbeDc field value if set, zero value otherwise.
func (o *SyntheticsAPITestResultShort) GetProbeDc() string {
	if o == nil || o.ProbeDc == nil {
		var ret string
		return ret
	}
	return *o.ProbeDc
}

// GetProbeDcOk returns a tuple with the ProbeDc field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITestResultShort) GetProbeDcOk() (*string, bool) {
	if o == nil || o.ProbeDc == nil {
		return nil, false
	}
	return o.ProbeDc, true
}

// HasProbeDc returns a boolean if a field has been set.
func (o *SyntheticsAPITestResultShort) HasProbeDc() bool {
	if o != nil && o.ProbeDc != nil {
		return true
	}

	return false
}

// SetProbeDc gets a reference to the given string and assigns it to the ProbeDc field.
func (o *SyntheticsAPITestResultShort) SetProbeDc(v string) {
	o.ProbeDc = &v
}

// GetResult returns the Result field value if set, zero value otherwise.
func (o *SyntheticsAPITestResultShort) GetResult() SyntheticsAPITestResultShortResult {
	if o == nil || o.Result == nil {
		var ret SyntheticsAPITestResultShortResult
		return ret
	}
	return *o.Result
}

// GetResultOk returns a tuple with the Result field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITestResultShort) GetResultOk() (*SyntheticsAPITestResultShortResult, bool) {
	if o == nil || o.Result == nil {
		return nil, false
	}
	return o.Result, true
}

// HasResult returns a boolean if a field has been set.
func (o *SyntheticsAPITestResultShort) HasResult() bool {
	if o != nil && o.Result != nil {
		return true
	}

	return false
}

// SetResult gets a reference to the given SyntheticsAPITestResultShortResult and assigns it to the Result field.
func (o *SyntheticsAPITestResultShort) SetResult(v SyntheticsAPITestResultShortResult) {
	o.Result = &v
}

// GetResultId returns the ResultId field value if set, zero value otherwise.
func (o *SyntheticsAPITestResultShort) GetResultId() string {
	if o == nil || o.ResultId == nil {
		var ret string
		return ret
	}
	return *o.ResultId
}

// GetResultIdOk returns a tuple with the ResultId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITestResultShort) GetResultIdOk() (*string, bool) {
	if o == nil || o.ResultId == nil {
		return nil, false
	}
	return o.ResultId, true
}

// HasResultId returns a boolean if a field has been set.
func (o *SyntheticsAPITestResultShort) HasResultId() bool {
	if o != nil && o.ResultId != nil {
		return true
	}

	return false
}

// SetResultId gets a reference to the given string and assigns it to the ResultId field.
func (o *SyntheticsAPITestResultShort) SetResultId(v string) {
	o.ResultId = &v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *SyntheticsAPITestResultShort) GetStatus() SyntheticsTestMonitorStatus {
	if o == nil || o.Status == nil {
		var ret SyntheticsTestMonitorStatus
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITestResultShort) GetStatusOk() (*SyntheticsTestMonitorStatus, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *SyntheticsAPITestResultShort) HasStatus() bool {
	if o != nil && o.Status != nil {
		return true
	}

	return false
}

// SetStatus gets a reference to the given SyntheticsTestMonitorStatus and assigns it to the Status field.
func (o *SyntheticsAPITestResultShort) SetStatus(v SyntheticsTestMonitorStatus) {
	o.Status = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsAPITestResultShort) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CheckTime != nil {
		toSerialize["check_time"] = o.CheckTime
	}
	if o.ProbeDc != nil {
		toSerialize["probe_dc"] = o.ProbeDc
	}
	if o.Result != nil {
		toSerialize["result"] = o.Result
	}
	if o.ResultId != nil {
		toSerialize["result_id"] = o.ResultId
	}
	if o.Status != nil {
		toSerialize["status"] = o.Status
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsAPITestResultShort) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		CheckTime *float64                            `json:"check_time,omitempty"`
		ProbeDc   *string                             `json:"probe_dc,omitempty"`
		Result    *SyntheticsAPITestResultShortResult `json:"result,omitempty"`
		ResultId  *string                             `json:"result_id,omitempty"`
		Status    *SyntheticsTestMonitorStatus        `json:"status,omitempty"`
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
	if v := all.Status; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.CheckTime = all.CheckTime
	o.ProbeDc = all.ProbeDc
	if all.Result != nil && all.Result.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Result = all.Result
	o.ResultId = all.ResultId
	o.Status = all.Status
	return nil
}
