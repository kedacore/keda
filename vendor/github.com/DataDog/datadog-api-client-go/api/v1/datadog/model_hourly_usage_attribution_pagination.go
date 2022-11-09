// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// HourlyUsageAttributionPagination The metadata for the current pagination.
type HourlyUsageAttributionPagination struct {
	// The cursor to get the next results (if any). To make the next request, use the same parameters and add `next_record_id`.
	NextRecordId NullableString `json:"next_record_id,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewHourlyUsageAttributionPagination instantiates a new HourlyUsageAttributionPagination object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewHourlyUsageAttributionPagination() *HourlyUsageAttributionPagination {
	this := HourlyUsageAttributionPagination{}
	return &this
}

// NewHourlyUsageAttributionPaginationWithDefaults instantiates a new HourlyUsageAttributionPagination object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewHourlyUsageAttributionPaginationWithDefaults() *HourlyUsageAttributionPagination {
	this := HourlyUsageAttributionPagination{}
	return &this
}

// GetNextRecordId returns the NextRecordId field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *HourlyUsageAttributionPagination) GetNextRecordId() string {
	if o == nil || o.NextRecordId.Get() == nil {
		var ret string
		return ret
	}
	return *o.NextRecordId.Get()
}

// GetNextRecordIdOk returns a tuple with the NextRecordId field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *HourlyUsageAttributionPagination) GetNextRecordIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.NextRecordId.Get(), o.NextRecordId.IsSet()
}

// HasNextRecordId returns a boolean if a field has been set.
func (o *HourlyUsageAttributionPagination) HasNextRecordId() bool {
	if o != nil && o.NextRecordId.IsSet() {
		return true
	}

	return false
}

// SetNextRecordId gets a reference to the given NullableString and assigns it to the NextRecordId field.
func (o *HourlyUsageAttributionPagination) SetNextRecordId(v string) {
	o.NextRecordId.Set(&v)
}

// SetNextRecordIdNil sets the value for NextRecordId to be an explicit nil.
func (o *HourlyUsageAttributionPagination) SetNextRecordIdNil() {
	o.NextRecordId.Set(nil)
}

// UnsetNextRecordId ensures that no value is present for NextRecordId, not even an explicit nil.
func (o *HourlyUsageAttributionPagination) UnsetNextRecordId() {
	o.NextRecordId.Unset()
}

// MarshalJSON serializes the struct using spec logic.
func (o HourlyUsageAttributionPagination) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.NextRecordId.IsSet() {
		toSerialize["next_record_id"] = o.NextRecordId.Get()
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *HourlyUsageAttributionPagination) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		NextRecordId NullableString `json:"next_record_id,omitempty"`
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
	o.NextRecordId = all.NextRecordId
	return nil
}
