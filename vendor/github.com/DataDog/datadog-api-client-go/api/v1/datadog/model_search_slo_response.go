// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SearchSLOResponse A search SLO response containing results from the search query.
type SearchSLOResponse struct {
	// Data from search SLO response.
	Data *SearchSLOResponseData `json:"data,omitempty"`
	// Pagination links.
	Links *SearchSLOResponseLinks `json:"links,omitempty"`
	// Searches metadata returned by the API.
	Meta *SearchSLOResponseMeta `json:"meta,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSearchSLOResponse instantiates a new SearchSLOResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSearchSLOResponse() *SearchSLOResponse {
	this := SearchSLOResponse{}
	return &this
}

// NewSearchSLOResponseWithDefaults instantiates a new SearchSLOResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSearchSLOResponseWithDefaults() *SearchSLOResponse {
	this := SearchSLOResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *SearchSLOResponse) GetData() SearchSLOResponseData {
	if o == nil || o.Data == nil {
		var ret SearchSLOResponseData
		return ret
	}
	return *o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponse) GetDataOk() (*SearchSLOResponseData, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *SearchSLOResponse) HasData() bool {
	if o != nil && o.Data != nil {
		return true
	}

	return false
}

// SetData gets a reference to the given SearchSLOResponseData and assigns it to the Data field.
func (o *SearchSLOResponse) SetData(v SearchSLOResponseData) {
	o.Data = &v
}

// GetLinks returns the Links field value if set, zero value otherwise.
func (o *SearchSLOResponse) GetLinks() SearchSLOResponseLinks {
	if o == nil || o.Links == nil {
		var ret SearchSLOResponseLinks
		return ret
	}
	return *o.Links
}

// GetLinksOk returns a tuple with the Links field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponse) GetLinksOk() (*SearchSLOResponseLinks, bool) {
	if o == nil || o.Links == nil {
		return nil, false
	}
	return o.Links, true
}

// HasLinks returns a boolean if a field has been set.
func (o *SearchSLOResponse) HasLinks() bool {
	if o != nil && o.Links != nil {
		return true
	}

	return false
}

// SetLinks gets a reference to the given SearchSLOResponseLinks and assigns it to the Links field.
func (o *SearchSLOResponse) SetLinks(v SearchSLOResponseLinks) {
	o.Links = &v
}

// GetMeta returns the Meta field value if set, zero value otherwise.
func (o *SearchSLOResponse) GetMeta() SearchSLOResponseMeta {
	if o == nil || o.Meta == nil {
		var ret SearchSLOResponseMeta
		return ret
	}
	return *o.Meta
}

// GetMetaOk returns a tuple with the Meta field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponse) GetMetaOk() (*SearchSLOResponseMeta, bool) {
	if o == nil || o.Meta == nil {
		return nil, false
	}
	return o.Meta, true
}

// HasMeta returns a boolean if a field has been set.
func (o *SearchSLOResponse) HasMeta() bool {
	if o != nil && o.Meta != nil {
		return true
	}

	return false
}

// SetMeta gets a reference to the given SearchSLOResponseMeta and assigns it to the Meta field.
func (o *SearchSLOResponse) SetMeta(v SearchSLOResponseMeta) {
	o.Meta = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SearchSLOResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Data != nil {
		toSerialize["data"] = o.Data
	}
	if o.Links != nil {
		toSerialize["links"] = o.Links
	}
	if o.Meta != nil {
		toSerialize["meta"] = o.Meta
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SearchSLOResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data  *SearchSLOResponseData  `json:"data,omitempty"`
		Links *SearchSLOResponseLinks `json:"links,omitempty"`
		Meta  *SearchSLOResponseMeta  `json:"meta,omitempty"`
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
	if all.Data != nil && all.Data.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Data = all.Data
	if all.Links != nil && all.Links.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Links = all.Links
	if all.Meta != nil && all.Meta.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Meta = all.Meta
	return nil
}
