// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SignalStateUpdateRequest Attributes describing the change of state for a given state.
type SignalStateUpdateRequest struct {
	// Optional comment to explain why a signal is being archived.
	ArchiveComment *string `json:"archiveComment,omitempty"`
	// Reason why a signal has been archived.
	ArchiveReason *SignalArchiveReason `json:"archiveReason,omitempty"`
	// The new triage state of the signal.
	State SignalTriageState `json:"state"`
	// Version of the updated signal. If server side version is higher, update will be rejected.
	Version *int64 `json:"version,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSignalStateUpdateRequest instantiates a new SignalStateUpdateRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSignalStateUpdateRequest(state SignalTriageState) *SignalStateUpdateRequest {
	this := SignalStateUpdateRequest{}
	this.State = state
	return &this
}

// NewSignalStateUpdateRequestWithDefaults instantiates a new SignalStateUpdateRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSignalStateUpdateRequestWithDefaults() *SignalStateUpdateRequest {
	this := SignalStateUpdateRequest{}
	return &this
}

// GetArchiveComment returns the ArchiveComment field value if set, zero value otherwise.
func (o *SignalStateUpdateRequest) GetArchiveComment() string {
	if o == nil || o.ArchiveComment == nil {
		var ret string
		return ret
	}
	return *o.ArchiveComment
}

// GetArchiveCommentOk returns a tuple with the ArchiveComment field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SignalStateUpdateRequest) GetArchiveCommentOk() (*string, bool) {
	if o == nil || o.ArchiveComment == nil {
		return nil, false
	}
	return o.ArchiveComment, true
}

// HasArchiveComment returns a boolean if a field has been set.
func (o *SignalStateUpdateRequest) HasArchiveComment() bool {
	if o != nil && o.ArchiveComment != nil {
		return true
	}

	return false
}

// SetArchiveComment gets a reference to the given string and assigns it to the ArchiveComment field.
func (o *SignalStateUpdateRequest) SetArchiveComment(v string) {
	o.ArchiveComment = &v
}

// GetArchiveReason returns the ArchiveReason field value if set, zero value otherwise.
func (o *SignalStateUpdateRequest) GetArchiveReason() SignalArchiveReason {
	if o == nil || o.ArchiveReason == nil {
		var ret SignalArchiveReason
		return ret
	}
	return *o.ArchiveReason
}

// GetArchiveReasonOk returns a tuple with the ArchiveReason field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SignalStateUpdateRequest) GetArchiveReasonOk() (*SignalArchiveReason, bool) {
	if o == nil || o.ArchiveReason == nil {
		return nil, false
	}
	return o.ArchiveReason, true
}

// HasArchiveReason returns a boolean if a field has been set.
func (o *SignalStateUpdateRequest) HasArchiveReason() bool {
	if o != nil && o.ArchiveReason != nil {
		return true
	}

	return false
}

// SetArchiveReason gets a reference to the given SignalArchiveReason and assigns it to the ArchiveReason field.
func (o *SignalStateUpdateRequest) SetArchiveReason(v SignalArchiveReason) {
	o.ArchiveReason = &v
}

// GetState returns the State field value.
func (o *SignalStateUpdateRequest) GetState() SignalTriageState {
	if o == nil {
		var ret SignalTriageState
		return ret
	}
	return o.State
}

// GetStateOk returns a tuple with the State field value
// and a boolean to check if the value has been set.
func (o *SignalStateUpdateRequest) GetStateOk() (*SignalTriageState, bool) {
	if o == nil {
		return nil, false
	}
	return &o.State, true
}

// SetState sets field value.
func (o *SignalStateUpdateRequest) SetState(v SignalTriageState) {
	o.State = v
}

// GetVersion returns the Version field value if set, zero value otherwise.
func (o *SignalStateUpdateRequest) GetVersion() int64 {
	if o == nil || o.Version == nil {
		var ret int64
		return ret
	}
	return *o.Version
}

// GetVersionOk returns a tuple with the Version field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SignalStateUpdateRequest) GetVersionOk() (*int64, bool) {
	if o == nil || o.Version == nil {
		return nil, false
	}
	return o.Version, true
}

// HasVersion returns a boolean if a field has been set.
func (o *SignalStateUpdateRequest) HasVersion() bool {
	if o != nil && o.Version != nil {
		return true
	}

	return false
}

// SetVersion gets a reference to the given int64 and assigns it to the Version field.
func (o *SignalStateUpdateRequest) SetVersion(v int64) {
	o.Version = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SignalStateUpdateRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ArchiveComment != nil {
		toSerialize["archiveComment"] = o.ArchiveComment
	}
	if o.ArchiveReason != nil {
		toSerialize["archiveReason"] = o.ArchiveReason
	}
	toSerialize["state"] = o.State
	if o.Version != nil {
		toSerialize["version"] = o.Version
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SignalStateUpdateRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		State *SignalTriageState `json:"state"`
	}{}
	all := struct {
		ArchiveComment *string              `json:"archiveComment,omitempty"`
		ArchiveReason  *SignalArchiveReason `json:"archiveReason,omitempty"`
		State          SignalTriageState    `json:"state"`
		Version        *int64               `json:"version,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.State == nil {
		return fmt.Errorf("Required field state missing")
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
	if v := all.ArchiveReason; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.State; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.ArchiveComment = all.ArchiveComment
	o.ArchiveReason = all.ArchiveReason
	o.State = all.State
	o.Version = all.Version
	return nil
}
