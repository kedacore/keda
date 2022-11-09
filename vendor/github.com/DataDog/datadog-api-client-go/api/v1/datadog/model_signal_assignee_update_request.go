// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SignalAssigneeUpdateRequest Attributes describing an assignee update operation over a security signal.
type SignalAssigneeUpdateRequest struct {
	// The UUID of the user being assigned. Use empty string to return signal to unassigned.
	Assignee string `json:"assignee"`
	// Version of the updated signal. If server side version is higher, update will be rejected.
	Version *int64 `json:"version,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSignalAssigneeUpdateRequest instantiates a new SignalAssigneeUpdateRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSignalAssigneeUpdateRequest(assignee string) *SignalAssigneeUpdateRequest {
	this := SignalAssigneeUpdateRequest{}
	this.Assignee = assignee
	return &this
}

// NewSignalAssigneeUpdateRequestWithDefaults instantiates a new SignalAssigneeUpdateRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSignalAssigneeUpdateRequestWithDefaults() *SignalAssigneeUpdateRequest {
	this := SignalAssigneeUpdateRequest{}
	return &this
}

// GetAssignee returns the Assignee field value.
func (o *SignalAssigneeUpdateRequest) GetAssignee() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Assignee
}

// GetAssigneeOk returns a tuple with the Assignee field value
// and a boolean to check if the value has been set.
func (o *SignalAssigneeUpdateRequest) GetAssigneeOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Assignee, true
}

// SetAssignee sets field value.
func (o *SignalAssigneeUpdateRequest) SetAssignee(v string) {
	o.Assignee = v
}

// GetVersion returns the Version field value if set, zero value otherwise.
func (o *SignalAssigneeUpdateRequest) GetVersion() int64 {
	if o == nil || o.Version == nil {
		var ret int64
		return ret
	}
	return *o.Version
}

// GetVersionOk returns a tuple with the Version field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SignalAssigneeUpdateRequest) GetVersionOk() (*int64, bool) {
	if o == nil || o.Version == nil {
		return nil, false
	}
	return o.Version, true
}

// HasVersion returns a boolean if a field has been set.
func (o *SignalAssigneeUpdateRequest) HasVersion() bool {
	if o != nil && o.Version != nil {
		return true
	}

	return false
}

// SetVersion gets a reference to the given int64 and assigns it to the Version field.
func (o *SignalAssigneeUpdateRequest) SetVersion(v int64) {
	o.Version = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SignalAssigneeUpdateRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["assignee"] = o.Assignee
	if o.Version != nil {
		toSerialize["version"] = o.Version
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SignalAssigneeUpdateRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Assignee *string `json:"assignee"`
	}{}
	all := struct {
		Assignee string `json:"assignee"`
		Version  *int64 `json:"version,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Assignee == nil {
		return fmt.Errorf("Required field assignee missing")
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
	o.Assignee = all.Assignee
	o.Version = all.Version
	return nil
}
