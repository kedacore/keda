// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SearchSLOResponseDataAttributes Attributes
type SearchSLOResponseDataAttributes struct {
	// Facets
	Facets *SearchSLOResponseDataAttributesFacets `json:"facets,omitempty"`
	// SLOs
	Slo []ServiceLevelObjective `json:"slo,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSearchSLOResponseDataAttributes instantiates a new SearchSLOResponseDataAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSearchSLOResponseDataAttributes() *SearchSLOResponseDataAttributes {
	this := SearchSLOResponseDataAttributes{}
	return &this
}

// NewSearchSLOResponseDataAttributesWithDefaults instantiates a new SearchSLOResponseDataAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSearchSLOResponseDataAttributesWithDefaults() *SearchSLOResponseDataAttributes {
	this := SearchSLOResponseDataAttributes{}
	return &this
}

// GetFacets returns the Facets field value if set, zero value otherwise.
func (o *SearchSLOResponseDataAttributes) GetFacets() SearchSLOResponseDataAttributesFacets {
	if o == nil || o.Facets == nil {
		var ret SearchSLOResponseDataAttributesFacets
		return ret
	}
	return *o.Facets
}

// GetFacetsOk returns a tuple with the Facets field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseDataAttributes) GetFacetsOk() (*SearchSLOResponseDataAttributesFacets, bool) {
	if o == nil || o.Facets == nil {
		return nil, false
	}
	return o.Facets, true
}

// HasFacets returns a boolean if a field has been set.
func (o *SearchSLOResponseDataAttributes) HasFacets() bool {
	if o != nil && o.Facets != nil {
		return true
	}

	return false
}

// SetFacets gets a reference to the given SearchSLOResponseDataAttributesFacets and assigns it to the Facets field.
func (o *SearchSLOResponseDataAttributes) SetFacets(v SearchSLOResponseDataAttributesFacets) {
	o.Facets = &v
}

// GetSlo returns the Slo field value if set, zero value otherwise.
func (o *SearchSLOResponseDataAttributes) GetSlo() []ServiceLevelObjective {
	if o == nil || o.Slo == nil {
		var ret []ServiceLevelObjective
		return ret
	}
	return o.Slo
}

// GetSloOk returns a tuple with the Slo field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseDataAttributes) GetSloOk() (*[]ServiceLevelObjective, bool) {
	if o == nil || o.Slo == nil {
		return nil, false
	}
	return &o.Slo, true
}

// HasSlo returns a boolean if a field has been set.
func (o *SearchSLOResponseDataAttributes) HasSlo() bool {
	if o != nil && o.Slo != nil {
		return true
	}

	return false
}

// SetSlo gets a reference to the given []ServiceLevelObjective and assigns it to the Slo field.
func (o *SearchSLOResponseDataAttributes) SetSlo(v []ServiceLevelObjective) {
	o.Slo = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SearchSLOResponseDataAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Facets != nil {
		toSerialize["facets"] = o.Facets
	}
	if o.Slo != nil {
		toSerialize["slo"] = o.Slo
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SearchSLOResponseDataAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Facets *SearchSLOResponseDataAttributesFacets `json:"facets,omitempty"`
		Slo    []ServiceLevelObjective                `json:"slo,omitempty"`
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
	if all.Facets != nil && all.Facets.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Facets = all.Facets
	o.Slo = all.Slo
	return nil
}
