// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// MonitorSearchResponseMetadata Metadata about the response.
type MonitorSearchResponseMetadata struct {
	// The page to start paginating from.
	Page *int64 `json:"page,omitempty"`
	// The number of pages.
	PageCount *int64 `json:"page_count,omitempty"`
	// The number of monitors to return per page.
	PerPage *int64 `json:"per_page,omitempty"`
	// The total number of monitors.
	TotalCount *int64 `json:"total_count,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMonitorSearchResponseMetadata instantiates a new MonitorSearchResponseMetadata object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorSearchResponseMetadata() *MonitorSearchResponseMetadata {
	this := MonitorSearchResponseMetadata{}
	return &this
}

// NewMonitorSearchResponseMetadataWithDefaults instantiates a new MonitorSearchResponseMetadata object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorSearchResponseMetadataWithDefaults() *MonitorSearchResponseMetadata {
	this := MonitorSearchResponseMetadata{}
	return &this
}

// GetPage returns the Page field value if set, zero value otherwise.
func (o *MonitorSearchResponseMetadata) GetPage() int64 {
	if o == nil || o.Page == nil {
		var ret int64
		return ret
	}
	return *o.Page
}

// GetPageOk returns a tuple with the Page field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSearchResponseMetadata) GetPageOk() (*int64, bool) {
	if o == nil || o.Page == nil {
		return nil, false
	}
	return o.Page, true
}

// HasPage returns a boolean if a field has been set.
func (o *MonitorSearchResponseMetadata) HasPage() bool {
	if o != nil && o.Page != nil {
		return true
	}

	return false
}

// SetPage gets a reference to the given int64 and assigns it to the Page field.
func (o *MonitorSearchResponseMetadata) SetPage(v int64) {
	o.Page = &v
}

// GetPageCount returns the PageCount field value if set, zero value otherwise.
func (o *MonitorSearchResponseMetadata) GetPageCount() int64 {
	if o == nil || o.PageCount == nil {
		var ret int64
		return ret
	}
	return *o.PageCount
}

// GetPageCountOk returns a tuple with the PageCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSearchResponseMetadata) GetPageCountOk() (*int64, bool) {
	if o == nil || o.PageCount == nil {
		return nil, false
	}
	return o.PageCount, true
}

// HasPageCount returns a boolean if a field has been set.
func (o *MonitorSearchResponseMetadata) HasPageCount() bool {
	if o != nil && o.PageCount != nil {
		return true
	}

	return false
}

// SetPageCount gets a reference to the given int64 and assigns it to the PageCount field.
func (o *MonitorSearchResponseMetadata) SetPageCount(v int64) {
	o.PageCount = &v
}

// GetPerPage returns the PerPage field value if set, zero value otherwise.
func (o *MonitorSearchResponseMetadata) GetPerPage() int64 {
	if o == nil || o.PerPage == nil {
		var ret int64
		return ret
	}
	return *o.PerPage
}

// GetPerPageOk returns a tuple with the PerPage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSearchResponseMetadata) GetPerPageOk() (*int64, bool) {
	if o == nil || o.PerPage == nil {
		return nil, false
	}
	return o.PerPage, true
}

// HasPerPage returns a boolean if a field has been set.
func (o *MonitorSearchResponseMetadata) HasPerPage() bool {
	if o != nil && o.PerPage != nil {
		return true
	}

	return false
}

// SetPerPage gets a reference to the given int64 and assigns it to the PerPage field.
func (o *MonitorSearchResponseMetadata) SetPerPage(v int64) {
	o.PerPage = &v
}

// GetTotalCount returns the TotalCount field value if set, zero value otherwise.
func (o *MonitorSearchResponseMetadata) GetTotalCount() int64 {
	if o == nil || o.TotalCount == nil {
		var ret int64
		return ret
	}
	return *o.TotalCount
}

// GetTotalCountOk returns a tuple with the TotalCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSearchResponseMetadata) GetTotalCountOk() (*int64, bool) {
	if o == nil || o.TotalCount == nil {
		return nil, false
	}
	return o.TotalCount, true
}

// HasTotalCount returns a boolean if a field has been set.
func (o *MonitorSearchResponseMetadata) HasTotalCount() bool {
	if o != nil && o.TotalCount != nil {
		return true
	}

	return false
}

// SetTotalCount gets a reference to the given int64 and assigns it to the TotalCount field.
func (o *MonitorSearchResponseMetadata) SetTotalCount(v int64) {
	o.TotalCount = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorSearchResponseMetadata) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Page != nil {
		toSerialize["page"] = o.Page
	}
	if o.PageCount != nil {
		toSerialize["page_count"] = o.PageCount
	}
	if o.PerPage != nil {
		toSerialize["per_page"] = o.PerPage
	}
	if o.TotalCount != nil {
		toSerialize["total_count"] = o.TotalCount
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorSearchResponseMetadata) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Page       *int64 `json:"page,omitempty"`
		PageCount  *int64 `json:"page_count,omitempty"`
		PerPage    *int64 `json:"per_page,omitempty"`
		TotalCount *int64 `json:"total_count,omitempty"`
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
	o.Page = all.Page
	o.PageCount = all.PageCount
	o.PerPage = all.PerPage
	o.TotalCount = all.TotalCount
	return nil
}
