// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// AWSTagFilterListResponse An array of tag filter rules by `namespace` and tag filter string.
type AWSTagFilterListResponse struct {
	// An array of tag filters.
	Filters []AWSTagFilter `json:"filters,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewAWSTagFilterListResponse instantiates a new AWSTagFilterListResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAWSTagFilterListResponse() *AWSTagFilterListResponse {
	this := AWSTagFilterListResponse{}
	return &this
}

// NewAWSTagFilterListResponseWithDefaults instantiates a new AWSTagFilterListResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAWSTagFilterListResponseWithDefaults() *AWSTagFilterListResponse {
	this := AWSTagFilterListResponse{}
	return &this
}

// GetFilters returns the Filters field value if set, zero value otherwise.
func (o *AWSTagFilterListResponse) GetFilters() []AWSTagFilter {
	if o == nil || o.Filters == nil {
		var ret []AWSTagFilter
		return ret
	}
	return o.Filters
}

// GetFiltersOk returns a tuple with the Filters field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSTagFilterListResponse) GetFiltersOk() (*[]AWSTagFilter, bool) {
	if o == nil || o.Filters == nil {
		return nil, false
	}
	return &o.Filters, true
}

// HasFilters returns a boolean if a field has been set.
func (o *AWSTagFilterListResponse) HasFilters() bool {
	if o != nil && o.Filters != nil {
		return true
	}

	return false
}

// SetFilters gets a reference to the given []AWSTagFilter and assigns it to the Filters field.
func (o *AWSTagFilterListResponse) SetFilters(v []AWSTagFilter) {
	o.Filters = v
}

// MarshalJSON serializes the struct using spec logic.
func (o AWSTagFilterListResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Filters != nil {
		toSerialize["filters"] = o.Filters
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *AWSTagFilterListResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Filters []AWSTagFilter `json:"filters,omitempty"`
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
	o.Filters = all.Filters
	return nil
}
