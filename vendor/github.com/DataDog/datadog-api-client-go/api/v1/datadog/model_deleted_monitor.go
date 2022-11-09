// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// DeletedMonitor Response from the delete monitor call.
type DeletedMonitor struct {
	// ID of the deleted monitor.
	DeletedMonitorId *int64 `json:"deleted_monitor_id,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewDeletedMonitor instantiates a new DeletedMonitor object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewDeletedMonitor() *DeletedMonitor {
	this := DeletedMonitor{}
	return &this
}

// NewDeletedMonitorWithDefaults instantiates a new DeletedMonitor object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewDeletedMonitorWithDefaults() *DeletedMonitor {
	this := DeletedMonitor{}
	return &this
}

// GetDeletedMonitorId returns the DeletedMonitorId field value if set, zero value otherwise.
func (o *DeletedMonitor) GetDeletedMonitorId() int64 {
	if o == nil || o.DeletedMonitorId == nil {
		var ret int64
		return ret
	}
	return *o.DeletedMonitorId
}

// GetDeletedMonitorIdOk returns a tuple with the DeletedMonitorId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DeletedMonitor) GetDeletedMonitorIdOk() (*int64, bool) {
	if o == nil || o.DeletedMonitorId == nil {
		return nil, false
	}
	return o.DeletedMonitorId, true
}

// HasDeletedMonitorId returns a boolean if a field has been set.
func (o *DeletedMonitor) HasDeletedMonitorId() bool {
	if o != nil && o.DeletedMonitorId != nil {
		return true
	}

	return false
}

// SetDeletedMonitorId gets a reference to the given int64 and assigns it to the DeletedMonitorId field.
func (o *DeletedMonitor) SetDeletedMonitorId(v int64) {
	o.DeletedMonitorId = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o DeletedMonitor) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.DeletedMonitorId != nil {
		toSerialize["deleted_monitor_id"] = o.DeletedMonitorId
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *DeletedMonitor) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		DeletedMonitorId *int64 `json:"deleted_monitor_id,omitempty"`
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
	o.DeletedMonitorId = all.DeletedMonitorId
	return nil
}
