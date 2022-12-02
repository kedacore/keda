// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SLOListResponseMetadata The metadata object containing additional information about the list of SLOs.
type SLOListResponseMetadata struct {
	// The object containing information about the pages of the list of SLOs.
	Page *SLOListResponseMetadataPage `json:"page,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSLOListResponseMetadata instantiates a new SLOListResponseMetadata object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOListResponseMetadata() *SLOListResponseMetadata {
	this := SLOListResponseMetadata{}
	return &this
}

// NewSLOListResponseMetadataWithDefaults instantiates a new SLOListResponseMetadata object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOListResponseMetadataWithDefaults() *SLOListResponseMetadata {
	this := SLOListResponseMetadata{}
	return &this
}

// GetPage returns the Page field value if set, zero value otherwise.
func (o *SLOListResponseMetadata) GetPage() SLOListResponseMetadataPage {
	if o == nil || o.Page == nil {
		var ret SLOListResponseMetadataPage
		return ret
	}
	return *o.Page
}

// GetPageOk returns a tuple with the Page field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOListResponseMetadata) GetPageOk() (*SLOListResponseMetadataPage, bool) {
	if o == nil || o.Page == nil {
		return nil, false
	}
	return o.Page, true
}

// HasPage returns a boolean if a field has been set.
func (o *SLOListResponseMetadata) HasPage() bool {
	if o != nil && o.Page != nil {
		return true
	}

	return false
}

// SetPage gets a reference to the given SLOListResponseMetadataPage and assigns it to the Page field.
func (o *SLOListResponseMetadata) SetPage(v SLOListResponseMetadataPage) {
	o.Page = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOListResponseMetadata) MarshalJSON() ([]byte, error) {
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
func (o *SLOListResponseMetadata) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Page *SLOListResponseMetadataPage `json:"page,omitempty"`
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
