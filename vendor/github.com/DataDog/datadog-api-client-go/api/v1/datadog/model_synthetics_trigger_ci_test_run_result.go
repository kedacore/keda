// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsTriggerCITestRunResult Information about a single test run.
type SyntheticsTriggerCITestRunResult struct {
	// The device ID.
	Device *SyntheticsDeviceID `json:"device,omitempty"`
	// The location ID of the test run.
	Location *int64 `json:"location,omitempty"`
	// The public ID of the Synthetics test.
	PublicId *string `json:"public_id,omitempty"`
	// ID of the result.
	ResultId *string `json:"result_id,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsTriggerCITestRunResult instantiates a new SyntheticsTriggerCITestRunResult object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsTriggerCITestRunResult() *SyntheticsTriggerCITestRunResult {
	this := SyntheticsTriggerCITestRunResult{}
	return &this
}

// NewSyntheticsTriggerCITestRunResultWithDefaults instantiates a new SyntheticsTriggerCITestRunResult object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsTriggerCITestRunResultWithDefaults() *SyntheticsTriggerCITestRunResult {
	this := SyntheticsTriggerCITestRunResult{}
	return &this
}

// GetDevice returns the Device field value if set, zero value otherwise.
func (o *SyntheticsTriggerCITestRunResult) GetDevice() SyntheticsDeviceID {
	if o == nil || o.Device == nil {
		var ret SyntheticsDeviceID
		return ret
	}
	return *o.Device
}

// GetDeviceOk returns a tuple with the Device field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTriggerCITestRunResult) GetDeviceOk() (*SyntheticsDeviceID, bool) {
	if o == nil || o.Device == nil {
		return nil, false
	}
	return o.Device, true
}

// HasDevice returns a boolean if a field has been set.
func (o *SyntheticsTriggerCITestRunResult) HasDevice() bool {
	if o != nil && o.Device != nil {
		return true
	}

	return false
}

// SetDevice gets a reference to the given SyntheticsDeviceID and assigns it to the Device field.
func (o *SyntheticsTriggerCITestRunResult) SetDevice(v SyntheticsDeviceID) {
	o.Device = &v
}

// GetLocation returns the Location field value if set, zero value otherwise.
func (o *SyntheticsTriggerCITestRunResult) GetLocation() int64 {
	if o == nil || o.Location == nil {
		var ret int64
		return ret
	}
	return *o.Location
}

// GetLocationOk returns a tuple with the Location field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTriggerCITestRunResult) GetLocationOk() (*int64, bool) {
	if o == nil || o.Location == nil {
		return nil, false
	}
	return o.Location, true
}

// HasLocation returns a boolean if a field has been set.
func (o *SyntheticsTriggerCITestRunResult) HasLocation() bool {
	if o != nil && o.Location != nil {
		return true
	}

	return false
}

// SetLocation gets a reference to the given int64 and assigns it to the Location field.
func (o *SyntheticsTriggerCITestRunResult) SetLocation(v int64) {
	o.Location = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *SyntheticsTriggerCITestRunResult) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTriggerCITestRunResult) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *SyntheticsTriggerCITestRunResult) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *SyntheticsTriggerCITestRunResult) SetPublicId(v string) {
	o.PublicId = &v
}

// GetResultId returns the ResultId field value if set, zero value otherwise.
func (o *SyntheticsTriggerCITestRunResult) GetResultId() string {
	if o == nil || o.ResultId == nil {
		var ret string
		return ret
	}
	return *o.ResultId
}

// GetResultIdOk returns a tuple with the ResultId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTriggerCITestRunResult) GetResultIdOk() (*string, bool) {
	if o == nil || o.ResultId == nil {
		return nil, false
	}
	return o.ResultId, true
}

// HasResultId returns a boolean if a field has been set.
func (o *SyntheticsTriggerCITestRunResult) HasResultId() bool {
	if o != nil && o.ResultId != nil {
		return true
	}

	return false
}

// SetResultId gets a reference to the given string and assigns it to the ResultId field.
func (o *SyntheticsTriggerCITestRunResult) SetResultId(v string) {
	o.ResultId = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsTriggerCITestRunResult) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Device != nil {
		toSerialize["device"] = o.Device
	}
	if o.Location != nil {
		toSerialize["location"] = o.Location
	}
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
	}
	if o.ResultId != nil {
		toSerialize["result_id"] = o.ResultId
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsTriggerCITestRunResult) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Device   *SyntheticsDeviceID `json:"device,omitempty"`
		Location *int64              `json:"location,omitempty"`
		PublicId *string             `json:"public_id,omitempty"`
		ResultId *string             `json:"result_id,omitempty"`
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
	o.Device = all.Device
	o.Location = all.Location
	o.PublicId = all.PublicId
	o.ResultId = all.ResultId
	return nil
}
