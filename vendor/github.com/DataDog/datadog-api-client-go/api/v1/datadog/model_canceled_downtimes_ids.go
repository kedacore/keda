// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// CanceledDowntimesIds Object containing array of IDs of canceled downtimes.
type CanceledDowntimesIds struct {
	// ID of downtimes that were canceled.
	CancelledIds []int64 `json:"cancelled_ids,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewCanceledDowntimesIds instantiates a new CanceledDowntimesIds object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCanceledDowntimesIds() *CanceledDowntimesIds {
	this := CanceledDowntimesIds{}
	return &this
}

// NewCanceledDowntimesIdsWithDefaults instantiates a new CanceledDowntimesIds object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCanceledDowntimesIdsWithDefaults() *CanceledDowntimesIds {
	this := CanceledDowntimesIds{}
	return &this
}

// GetCancelledIds returns the CancelledIds field value if set, zero value otherwise.
func (o *CanceledDowntimesIds) GetCancelledIds() []int64 {
	if o == nil || o.CancelledIds == nil {
		var ret []int64
		return ret
	}
	return o.CancelledIds
}

// GetCancelledIdsOk returns a tuple with the CancelledIds field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CanceledDowntimesIds) GetCancelledIdsOk() (*[]int64, bool) {
	if o == nil || o.CancelledIds == nil {
		return nil, false
	}
	return &o.CancelledIds, true
}

// HasCancelledIds returns a boolean if a field has been set.
func (o *CanceledDowntimesIds) HasCancelledIds() bool {
	if o != nil && o.CancelledIds != nil {
		return true
	}

	return false
}

// SetCancelledIds gets a reference to the given []int64 and assigns it to the CancelledIds field.
func (o *CanceledDowntimesIds) SetCancelledIds(v []int64) {
	o.CancelledIds = v
}

// MarshalJSON serializes the struct using spec logic.
func (o CanceledDowntimesIds) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CancelledIds != nil {
		toSerialize["cancelled_ids"] = o.CancelledIds
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CanceledDowntimesIds) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		CancelledIds []int64 `json:"cancelled_ids,omitempty"`
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
	o.CancelledIds = all.CancelledIds
	return nil
}
