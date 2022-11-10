// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// AddSignalToIncidentRequest Attributes describing which incident to add the signal to.
type AddSignalToIncidentRequest struct {
	// Whether to post the signal on the incident timeline.
	AddToSignalTimeline *bool `json:"add_to_signal_timeline,omitempty"`
	// Public ID attribute of the incident to which the signal will be added.
	IncidentId int64 `json:"incident_id"`
	// Version of the updated signal. If server side version is higher, update will be rejected.
	Version *int64 `json:"version,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewAddSignalToIncidentRequest instantiates a new AddSignalToIncidentRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAddSignalToIncidentRequest(incidentId int64) *AddSignalToIncidentRequest {
	this := AddSignalToIncidentRequest{}
	this.IncidentId = incidentId
	return &this
}

// NewAddSignalToIncidentRequestWithDefaults instantiates a new AddSignalToIncidentRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAddSignalToIncidentRequestWithDefaults() *AddSignalToIncidentRequest {
	this := AddSignalToIncidentRequest{}
	return &this
}

// GetAddToSignalTimeline returns the AddToSignalTimeline field value if set, zero value otherwise.
func (o *AddSignalToIncidentRequest) GetAddToSignalTimeline() bool {
	if o == nil || o.AddToSignalTimeline == nil {
		var ret bool
		return ret
	}
	return *o.AddToSignalTimeline
}

// GetAddToSignalTimelineOk returns a tuple with the AddToSignalTimeline field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AddSignalToIncidentRequest) GetAddToSignalTimelineOk() (*bool, bool) {
	if o == nil || o.AddToSignalTimeline == nil {
		return nil, false
	}
	return o.AddToSignalTimeline, true
}

// HasAddToSignalTimeline returns a boolean if a field has been set.
func (o *AddSignalToIncidentRequest) HasAddToSignalTimeline() bool {
	if o != nil && o.AddToSignalTimeline != nil {
		return true
	}

	return false
}

// SetAddToSignalTimeline gets a reference to the given bool and assigns it to the AddToSignalTimeline field.
func (o *AddSignalToIncidentRequest) SetAddToSignalTimeline(v bool) {
	o.AddToSignalTimeline = &v
}

// GetIncidentId returns the IncidentId field value.
func (o *AddSignalToIncidentRequest) GetIncidentId() int64 {
	if o == nil {
		var ret int64
		return ret
	}
	return o.IncidentId
}

// GetIncidentIdOk returns a tuple with the IncidentId field value
// and a boolean to check if the value has been set.
func (o *AddSignalToIncidentRequest) GetIncidentIdOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.IncidentId, true
}

// SetIncidentId sets field value.
func (o *AddSignalToIncidentRequest) SetIncidentId(v int64) {
	o.IncidentId = v
}

// GetVersion returns the Version field value if set, zero value otherwise.
func (o *AddSignalToIncidentRequest) GetVersion() int64 {
	if o == nil || o.Version == nil {
		var ret int64
		return ret
	}
	return *o.Version
}

// GetVersionOk returns a tuple with the Version field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AddSignalToIncidentRequest) GetVersionOk() (*int64, bool) {
	if o == nil || o.Version == nil {
		return nil, false
	}
	return o.Version, true
}

// HasVersion returns a boolean if a field has been set.
func (o *AddSignalToIncidentRequest) HasVersion() bool {
	if o != nil && o.Version != nil {
		return true
	}

	return false
}

// SetVersion gets a reference to the given int64 and assigns it to the Version field.
func (o *AddSignalToIncidentRequest) SetVersion(v int64) {
	o.Version = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o AddSignalToIncidentRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AddToSignalTimeline != nil {
		toSerialize["add_to_signal_timeline"] = o.AddToSignalTimeline
	}
	toSerialize["incident_id"] = o.IncidentId
	if o.Version != nil {
		toSerialize["version"] = o.Version
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *AddSignalToIncidentRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		IncidentId *int64 `json:"incident_id"`
	}{}
	all := struct {
		AddToSignalTimeline *bool  `json:"add_to_signal_timeline,omitempty"`
		IncidentId          int64  `json:"incident_id"`
		Version             *int64 `json:"version,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.IncidentId == nil {
		return fmt.Errorf("Required field incident_id missing")
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
	o.AddToSignalTimeline = all.AddToSignalTimeline
	o.IncidentId = all.IncidentId
	o.Version = all.Version
	return nil
}
