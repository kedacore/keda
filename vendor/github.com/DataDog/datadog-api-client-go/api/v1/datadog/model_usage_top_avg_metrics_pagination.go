// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// UsageTopAvgMetricsPagination The metadata for the current pagination.
type UsageTopAvgMetricsPagination struct {
	// Maximum amount of records to be returned.
	Limit *int64 `json:"limit,omitempty"`
	// The cursor to get the next results (if any). To make the next request, use the same parameters and add `next_record_id`.
	NextRecordId NullableString `json:"next_record_id,omitempty"`
	// Total number of records.
	TotalNumberOfRecords *int64 `json:"total_number_of_records,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageTopAvgMetricsPagination instantiates a new UsageTopAvgMetricsPagination object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageTopAvgMetricsPagination() *UsageTopAvgMetricsPagination {
	this := UsageTopAvgMetricsPagination{}
	return &this
}

// NewUsageTopAvgMetricsPaginationWithDefaults instantiates a new UsageTopAvgMetricsPagination object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageTopAvgMetricsPaginationWithDefaults() *UsageTopAvgMetricsPagination {
	this := UsageTopAvgMetricsPagination{}
	return &this
}

// GetLimit returns the Limit field value if set, zero value otherwise.
func (o *UsageTopAvgMetricsPagination) GetLimit() int64 {
	if o == nil || o.Limit == nil {
		var ret int64
		return ret
	}
	return *o.Limit
}

// GetLimitOk returns a tuple with the Limit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageTopAvgMetricsPagination) GetLimitOk() (*int64, bool) {
	if o == nil || o.Limit == nil {
		return nil, false
	}
	return o.Limit, true
}

// HasLimit returns a boolean if a field has been set.
func (o *UsageTopAvgMetricsPagination) HasLimit() bool {
	if o != nil && o.Limit != nil {
		return true
	}

	return false
}

// SetLimit gets a reference to the given int64 and assigns it to the Limit field.
func (o *UsageTopAvgMetricsPagination) SetLimit(v int64) {
	o.Limit = &v
}

// GetNextRecordId returns the NextRecordId field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *UsageTopAvgMetricsPagination) GetNextRecordId() string {
	if o == nil || o.NextRecordId.Get() == nil {
		var ret string
		return ret
	}
	return *o.NextRecordId.Get()
}

// GetNextRecordIdOk returns a tuple with the NextRecordId field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *UsageTopAvgMetricsPagination) GetNextRecordIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.NextRecordId.Get(), o.NextRecordId.IsSet()
}

// HasNextRecordId returns a boolean if a field has been set.
func (o *UsageTopAvgMetricsPagination) HasNextRecordId() bool {
	if o != nil && o.NextRecordId.IsSet() {
		return true
	}

	return false
}

// SetNextRecordId gets a reference to the given NullableString and assigns it to the NextRecordId field.
func (o *UsageTopAvgMetricsPagination) SetNextRecordId(v string) {
	o.NextRecordId.Set(&v)
}

// SetNextRecordIdNil sets the value for NextRecordId to be an explicit nil.
func (o *UsageTopAvgMetricsPagination) SetNextRecordIdNil() {
	o.NextRecordId.Set(nil)
}

// UnsetNextRecordId ensures that no value is present for NextRecordId, not even an explicit nil.
func (o *UsageTopAvgMetricsPagination) UnsetNextRecordId() {
	o.NextRecordId.Unset()
}

// GetTotalNumberOfRecords returns the TotalNumberOfRecords field value if set, zero value otherwise.
func (o *UsageTopAvgMetricsPagination) GetTotalNumberOfRecords() int64 {
	if o == nil || o.TotalNumberOfRecords == nil {
		var ret int64
		return ret
	}
	return *o.TotalNumberOfRecords
}

// GetTotalNumberOfRecordsOk returns a tuple with the TotalNumberOfRecords field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageTopAvgMetricsPagination) GetTotalNumberOfRecordsOk() (*int64, bool) {
	if o == nil || o.TotalNumberOfRecords == nil {
		return nil, false
	}
	return o.TotalNumberOfRecords, true
}

// HasTotalNumberOfRecords returns a boolean if a field has been set.
func (o *UsageTopAvgMetricsPagination) HasTotalNumberOfRecords() bool {
	if o != nil && o.TotalNumberOfRecords != nil {
		return true
	}

	return false
}

// SetTotalNumberOfRecords gets a reference to the given int64 and assigns it to the TotalNumberOfRecords field.
func (o *UsageTopAvgMetricsPagination) SetTotalNumberOfRecords(v int64) {
	o.TotalNumberOfRecords = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageTopAvgMetricsPagination) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Limit != nil {
		toSerialize["limit"] = o.Limit
	}
	if o.NextRecordId.IsSet() {
		toSerialize["next_record_id"] = o.NextRecordId.Get()
	}
	if o.TotalNumberOfRecords != nil {
		toSerialize["total_number_of_records"] = o.TotalNumberOfRecords
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageTopAvgMetricsPagination) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Limit                *int64         `json:"limit,omitempty"`
		NextRecordId         NullableString `json:"next_record_id,omitempty"`
		TotalNumberOfRecords *int64         `json:"total_number_of_records,omitempty"`
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
	o.Limit = all.Limit
	o.NextRecordId = all.NextRecordId
	o.TotalNumberOfRecords = all.TotalNumberOfRecords
	return nil
}
