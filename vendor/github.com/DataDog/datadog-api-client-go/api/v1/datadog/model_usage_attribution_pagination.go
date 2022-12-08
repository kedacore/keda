// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// UsageAttributionPagination The metadata for the current pagination.
type UsageAttributionPagination struct {
	// Maximum amount of records to be returned.
	Limit *int64 `json:"limit,omitempty"`
	// Records to be skipped before beginning to return.
	Offset *int64 `json:"offset,omitempty"`
	// Direction to sort by.
	SortDirection *string `json:"sort_direction,omitempty"`
	// Field to sort by.
	SortName *string `json:"sort_name,omitempty"`
	// Total number of records.
	TotalNumberOfRecords *int64 `json:"total_number_of_records,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageAttributionPagination instantiates a new UsageAttributionPagination object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageAttributionPagination() *UsageAttributionPagination {
	this := UsageAttributionPagination{}
	return &this
}

// NewUsageAttributionPaginationWithDefaults instantiates a new UsageAttributionPagination object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageAttributionPaginationWithDefaults() *UsageAttributionPagination {
	this := UsageAttributionPagination{}
	return &this
}

// GetLimit returns the Limit field value if set, zero value otherwise.
func (o *UsageAttributionPagination) GetLimit() int64 {
	if o == nil || o.Limit == nil {
		var ret int64
		return ret
	}
	return *o.Limit
}

// GetLimitOk returns a tuple with the Limit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionPagination) GetLimitOk() (*int64, bool) {
	if o == nil || o.Limit == nil {
		return nil, false
	}
	return o.Limit, true
}

// HasLimit returns a boolean if a field has been set.
func (o *UsageAttributionPagination) HasLimit() bool {
	if o != nil && o.Limit != nil {
		return true
	}

	return false
}

// SetLimit gets a reference to the given int64 and assigns it to the Limit field.
func (o *UsageAttributionPagination) SetLimit(v int64) {
	o.Limit = &v
}

// GetOffset returns the Offset field value if set, zero value otherwise.
func (o *UsageAttributionPagination) GetOffset() int64 {
	if o == nil || o.Offset == nil {
		var ret int64
		return ret
	}
	return *o.Offset
}

// GetOffsetOk returns a tuple with the Offset field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionPagination) GetOffsetOk() (*int64, bool) {
	if o == nil || o.Offset == nil {
		return nil, false
	}
	return o.Offset, true
}

// HasOffset returns a boolean if a field has been set.
func (o *UsageAttributionPagination) HasOffset() bool {
	if o != nil && o.Offset != nil {
		return true
	}

	return false
}

// SetOffset gets a reference to the given int64 and assigns it to the Offset field.
func (o *UsageAttributionPagination) SetOffset(v int64) {
	o.Offset = &v
}

// GetSortDirection returns the SortDirection field value if set, zero value otherwise.
func (o *UsageAttributionPagination) GetSortDirection() string {
	if o == nil || o.SortDirection == nil {
		var ret string
		return ret
	}
	return *o.SortDirection
}

// GetSortDirectionOk returns a tuple with the SortDirection field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionPagination) GetSortDirectionOk() (*string, bool) {
	if o == nil || o.SortDirection == nil {
		return nil, false
	}
	return o.SortDirection, true
}

// HasSortDirection returns a boolean if a field has been set.
func (o *UsageAttributionPagination) HasSortDirection() bool {
	if o != nil && o.SortDirection != nil {
		return true
	}

	return false
}

// SetSortDirection gets a reference to the given string and assigns it to the SortDirection field.
func (o *UsageAttributionPagination) SetSortDirection(v string) {
	o.SortDirection = &v
}

// GetSortName returns the SortName field value if set, zero value otherwise.
func (o *UsageAttributionPagination) GetSortName() string {
	if o == nil || o.SortName == nil {
		var ret string
		return ret
	}
	return *o.SortName
}

// GetSortNameOk returns a tuple with the SortName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionPagination) GetSortNameOk() (*string, bool) {
	if o == nil || o.SortName == nil {
		return nil, false
	}
	return o.SortName, true
}

// HasSortName returns a boolean if a field has been set.
func (o *UsageAttributionPagination) HasSortName() bool {
	if o != nil && o.SortName != nil {
		return true
	}

	return false
}

// SetSortName gets a reference to the given string and assigns it to the SortName field.
func (o *UsageAttributionPagination) SetSortName(v string) {
	o.SortName = &v
}

// GetTotalNumberOfRecords returns the TotalNumberOfRecords field value if set, zero value otherwise.
func (o *UsageAttributionPagination) GetTotalNumberOfRecords() int64 {
	if o == nil || o.TotalNumberOfRecords == nil {
		var ret int64
		return ret
	}
	return *o.TotalNumberOfRecords
}

// GetTotalNumberOfRecordsOk returns a tuple with the TotalNumberOfRecords field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionPagination) GetTotalNumberOfRecordsOk() (*int64, bool) {
	if o == nil || o.TotalNumberOfRecords == nil {
		return nil, false
	}
	return o.TotalNumberOfRecords, true
}

// HasTotalNumberOfRecords returns a boolean if a field has been set.
func (o *UsageAttributionPagination) HasTotalNumberOfRecords() bool {
	if o != nil && o.TotalNumberOfRecords != nil {
		return true
	}

	return false
}

// SetTotalNumberOfRecords gets a reference to the given int64 and assigns it to the TotalNumberOfRecords field.
func (o *UsageAttributionPagination) SetTotalNumberOfRecords(v int64) {
	o.TotalNumberOfRecords = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageAttributionPagination) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Limit != nil {
		toSerialize["limit"] = o.Limit
	}
	if o.Offset != nil {
		toSerialize["offset"] = o.Offset
	}
	if o.SortDirection != nil {
		toSerialize["sort_direction"] = o.SortDirection
	}
	if o.SortName != nil {
		toSerialize["sort_name"] = o.SortName
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
func (o *UsageAttributionPagination) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Limit                *int64  `json:"limit,omitempty"`
		Offset               *int64  `json:"offset,omitempty"`
		SortDirection        *string `json:"sort_direction,omitempty"`
		SortName             *string `json:"sort_name,omitempty"`
		TotalNumberOfRecords *int64  `json:"total_number_of_records,omitempty"`
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
	o.Offset = all.Offset
	o.SortDirection = all.SortDirection
	o.SortName = all.SortName
	o.TotalNumberOfRecords = all.TotalNumberOfRecords
	return nil
}
