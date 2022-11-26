// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// HourlyUsageAttributionMetadata The object containing document metadata.
type HourlyUsageAttributionMetadata struct {
	// The metadata for the current pagination.
	Pagination *HourlyUsageAttributionPagination `json:"pagination,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewHourlyUsageAttributionMetadata instantiates a new HourlyUsageAttributionMetadata object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewHourlyUsageAttributionMetadata() *HourlyUsageAttributionMetadata {
	this := HourlyUsageAttributionMetadata{}
	return &this
}

// NewHourlyUsageAttributionMetadataWithDefaults instantiates a new HourlyUsageAttributionMetadata object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewHourlyUsageAttributionMetadataWithDefaults() *HourlyUsageAttributionMetadata {
	this := HourlyUsageAttributionMetadata{}
	return &this
}

// GetPagination returns the Pagination field value if set, zero value otherwise.
func (o *HourlyUsageAttributionMetadata) GetPagination() HourlyUsageAttributionPagination {
	if o == nil || o.Pagination == nil {
		var ret HourlyUsageAttributionPagination
		return ret
	}
	return *o.Pagination
}

// GetPaginationOk returns a tuple with the Pagination field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageAttributionMetadata) GetPaginationOk() (*HourlyUsageAttributionPagination, bool) {
	if o == nil || o.Pagination == nil {
		return nil, false
	}
	return o.Pagination, true
}

// HasPagination returns a boolean if a field has been set.
func (o *HourlyUsageAttributionMetadata) HasPagination() bool {
	if o != nil && o.Pagination != nil {
		return true
	}

	return false
}

// SetPagination gets a reference to the given HourlyUsageAttributionPagination and assigns it to the Pagination field.
func (o *HourlyUsageAttributionMetadata) SetPagination(v HourlyUsageAttributionPagination) {
	o.Pagination = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o HourlyUsageAttributionMetadata) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Pagination != nil {
		toSerialize["pagination"] = o.Pagination
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *HourlyUsageAttributionMetadata) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Pagination *HourlyUsageAttributionPagination `json:"pagination,omitempty"`
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
	if all.Pagination != nil && all.Pagination.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Pagination = all.Pagination
	return nil
}
