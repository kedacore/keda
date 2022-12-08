// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SearchSLOResponseDataAttributesFacetsObjectInt Facet
type SearchSLOResponseDataAttributesFacetsObjectInt struct {
	// Count
	Count *int64 `json:"count,omitempty"`
	// Facet
	Name *float64 `json:"name,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSearchSLOResponseDataAttributesFacetsObjectInt instantiates a new SearchSLOResponseDataAttributesFacetsObjectInt object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSearchSLOResponseDataAttributesFacetsObjectInt() *SearchSLOResponseDataAttributesFacetsObjectInt {
	this := SearchSLOResponseDataAttributesFacetsObjectInt{}
	return &this
}

// NewSearchSLOResponseDataAttributesFacetsObjectIntWithDefaults instantiates a new SearchSLOResponseDataAttributesFacetsObjectInt object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSearchSLOResponseDataAttributesFacetsObjectIntWithDefaults() *SearchSLOResponseDataAttributesFacetsObjectInt {
	this := SearchSLOResponseDataAttributesFacetsObjectInt{}
	return &this
}

// GetCount returns the Count field value if set, zero value otherwise.
func (o *SearchSLOResponseDataAttributesFacetsObjectInt) GetCount() int64 {
	if o == nil || o.Count == nil {
		var ret int64
		return ret
	}
	return *o.Count
}

// GetCountOk returns a tuple with the Count field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseDataAttributesFacetsObjectInt) GetCountOk() (*int64, bool) {
	if o == nil || o.Count == nil {
		return nil, false
	}
	return o.Count, true
}

// HasCount returns a boolean if a field has been set.
func (o *SearchSLOResponseDataAttributesFacetsObjectInt) HasCount() bool {
	if o != nil && o.Count != nil {
		return true
	}

	return false
}

// SetCount gets a reference to the given int64 and assigns it to the Count field.
func (o *SearchSLOResponseDataAttributesFacetsObjectInt) SetCount(v int64) {
	o.Count = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *SearchSLOResponseDataAttributesFacetsObjectInt) GetName() float64 {
	if o == nil || o.Name == nil {
		var ret float64
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseDataAttributesFacetsObjectInt) GetNameOk() (*float64, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *SearchSLOResponseDataAttributesFacetsObjectInt) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given float64 and assigns it to the Name field.
func (o *SearchSLOResponseDataAttributesFacetsObjectInt) SetName(v float64) {
	o.Name = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SearchSLOResponseDataAttributesFacetsObjectInt) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Count != nil {
		toSerialize["count"] = o.Count
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SearchSLOResponseDataAttributesFacetsObjectInt) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Count *int64   `json:"count,omitempty"`
		Name  *float64 `json:"name,omitempty"`
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
	o.Count = all.Count
	o.Name = all.Name
	return nil
}
