// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsUpdateTestPauseStatusPayload Object to start or pause an existing Synthetic test.
type SyntheticsUpdateTestPauseStatusPayload struct {
	// Define whether you want to start (`live`) or pause (`paused`) a
	// Synthetic test.
	NewStatus *SyntheticsTestPauseStatus `json:"new_status,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsUpdateTestPauseStatusPayload instantiates a new SyntheticsUpdateTestPauseStatusPayload object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsUpdateTestPauseStatusPayload() *SyntheticsUpdateTestPauseStatusPayload {
	this := SyntheticsUpdateTestPauseStatusPayload{}
	return &this
}

// NewSyntheticsUpdateTestPauseStatusPayloadWithDefaults instantiates a new SyntheticsUpdateTestPauseStatusPayload object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsUpdateTestPauseStatusPayloadWithDefaults() *SyntheticsUpdateTestPauseStatusPayload {
	this := SyntheticsUpdateTestPauseStatusPayload{}
	return &this
}

// GetNewStatus returns the NewStatus field value if set, zero value otherwise.
func (o *SyntheticsUpdateTestPauseStatusPayload) GetNewStatus() SyntheticsTestPauseStatus {
	if o == nil || o.NewStatus == nil {
		var ret SyntheticsTestPauseStatus
		return ret
	}
	return *o.NewStatus
}

// GetNewStatusOk returns a tuple with the NewStatus field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsUpdateTestPauseStatusPayload) GetNewStatusOk() (*SyntheticsTestPauseStatus, bool) {
	if o == nil || o.NewStatus == nil {
		return nil, false
	}
	return o.NewStatus, true
}

// HasNewStatus returns a boolean if a field has been set.
func (o *SyntheticsUpdateTestPauseStatusPayload) HasNewStatus() bool {
	if o != nil && o.NewStatus != nil {
		return true
	}

	return false
}

// SetNewStatus gets a reference to the given SyntheticsTestPauseStatus and assigns it to the NewStatus field.
func (o *SyntheticsUpdateTestPauseStatusPayload) SetNewStatus(v SyntheticsTestPauseStatus) {
	o.NewStatus = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsUpdateTestPauseStatusPayload) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.NewStatus != nil {
		toSerialize["new_status"] = o.NewStatus
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsUpdateTestPauseStatusPayload) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		NewStatus *SyntheticsTestPauseStatus `json:"new_status,omitempty"`
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
	if v := all.NewStatus; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.NewStatus = all.NewStatus
	return nil
}
