// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// UsageSpecifiedCustomReportsMeta The object containing document metadata.
type UsageSpecifiedCustomReportsMeta struct {
	// The object containing page total count for specified ID.
	Page *UsageSpecifiedCustomReportsPage `json:"page,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageSpecifiedCustomReportsMeta instantiates a new UsageSpecifiedCustomReportsMeta object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageSpecifiedCustomReportsMeta() *UsageSpecifiedCustomReportsMeta {
	this := UsageSpecifiedCustomReportsMeta{}
	return &this
}

// NewUsageSpecifiedCustomReportsMetaWithDefaults instantiates a new UsageSpecifiedCustomReportsMeta object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageSpecifiedCustomReportsMetaWithDefaults() *UsageSpecifiedCustomReportsMeta {
	this := UsageSpecifiedCustomReportsMeta{}
	return &this
}

// GetPage returns the Page field value if set, zero value otherwise.
func (o *UsageSpecifiedCustomReportsMeta) GetPage() UsageSpecifiedCustomReportsPage {
	if o == nil || o.Page == nil {
		var ret UsageSpecifiedCustomReportsPage
		return ret
	}
	return *o.Page
}

// GetPageOk returns a tuple with the Page field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSpecifiedCustomReportsMeta) GetPageOk() (*UsageSpecifiedCustomReportsPage, bool) {
	if o == nil || o.Page == nil {
		return nil, false
	}
	return o.Page, true
}

// HasPage returns a boolean if a field has been set.
func (o *UsageSpecifiedCustomReportsMeta) HasPage() bool {
	if o != nil && o.Page != nil {
		return true
	}

	return false
}

// SetPage gets a reference to the given UsageSpecifiedCustomReportsPage and assigns it to the Page field.
func (o *UsageSpecifiedCustomReportsMeta) SetPage(v UsageSpecifiedCustomReportsPage) {
	o.Page = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageSpecifiedCustomReportsMeta) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Page != nil {
		toSerialize["page"] = o.Page
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageSpecifiedCustomReportsMeta) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Page *UsageSpecifiedCustomReportsPage `json:"page,omitempty"`
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
	if all.Page != nil && all.Page.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Page = all.Page
	return nil
}
