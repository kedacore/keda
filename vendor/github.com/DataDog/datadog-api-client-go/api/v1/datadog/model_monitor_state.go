// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// MonitorState Wrapper object with the different monitor states.
type MonitorState struct {
	// Dictionary where the keys are groups (comma separated lists of tags) and the values are
	// the list of groups your monitor is broken down on.
	Groups map[string]MonitorStateGroup `json:"groups,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMonitorState instantiates a new MonitorState object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorState() *MonitorState {
	this := MonitorState{}
	return &this
}

// NewMonitorStateWithDefaults instantiates a new MonitorState object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorStateWithDefaults() *MonitorState {
	this := MonitorState{}
	return &this
}

// GetGroups returns the Groups field value if set, zero value otherwise.
func (o *MonitorState) GetGroups() map[string]MonitorStateGroup {
	if o == nil || o.Groups == nil {
		var ret map[string]MonitorStateGroup
		return ret
	}
	return o.Groups
}

// GetGroupsOk returns a tuple with the Groups field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorState) GetGroupsOk() (*map[string]MonitorStateGroup, bool) {
	if o == nil || o.Groups == nil {
		return nil, false
	}
	return &o.Groups, true
}

// HasGroups returns a boolean if a field has been set.
func (o *MonitorState) HasGroups() bool {
	if o != nil && o.Groups != nil {
		return true
	}

	return false
}

// SetGroups gets a reference to the given map[string]MonitorStateGroup and assigns it to the Groups field.
func (o *MonitorState) SetGroups(v map[string]MonitorStateGroup) {
	o.Groups = v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorState) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Groups != nil {
		toSerialize["groups"] = o.Groups
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorState) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Groups map[string]MonitorStateGroup `json:"groups,omitempty"`
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
	o.Groups = all.Groups
	return nil
}
