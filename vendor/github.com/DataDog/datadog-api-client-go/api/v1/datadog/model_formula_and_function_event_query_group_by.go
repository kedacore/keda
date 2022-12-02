// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// FormulaAndFunctionEventQueryGroupBy List of objects used to group by.
type FormulaAndFunctionEventQueryGroupBy struct {
	// Event facet.
	Facet string `json:"facet"`
	// Number of groups to return.
	Limit *int64 `json:"limit,omitempty"`
	// Options for sorting group by results.
	Sort *FormulaAndFunctionEventQueryGroupBySort `json:"sort,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewFormulaAndFunctionEventQueryGroupBy instantiates a new FormulaAndFunctionEventQueryGroupBy object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewFormulaAndFunctionEventQueryGroupBy(facet string) *FormulaAndFunctionEventQueryGroupBy {
	this := FormulaAndFunctionEventQueryGroupBy{}
	this.Facet = facet
	return &this
}

// NewFormulaAndFunctionEventQueryGroupByWithDefaults instantiates a new FormulaAndFunctionEventQueryGroupBy object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewFormulaAndFunctionEventQueryGroupByWithDefaults() *FormulaAndFunctionEventQueryGroupBy {
	this := FormulaAndFunctionEventQueryGroupBy{}
	return &this
}

// GetFacet returns the Facet field value.
func (o *FormulaAndFunctionEventQueryGroupBy) GetFacet() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Facet
}

// GetFacetOk returns a tuple with the Facet field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionEventQueryGroupBy) GetFacetOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Facet, true
}

// SetFacet sets field value.
func (o *FormulaAndFunctionEventQueryGroupBy) SetFacet(v string) {
	o.Facet = v
}

// GetLimit returns the Limit field value if set, zero value otherwise.
func (o *FormulaAndFunctionEventQueryGroupBy) GetLimit() int64 {
	if o == nil || o.Limit == nil {
		var ret int64
		return ret
	}
	return *o.Limit
}

// GetLimitOk returns a tuple with the Limit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionEventQueryGroupBy) GetLimitOk() (*int64, bool) {
	if o == nil || o.Limit == nil {
		return nil, false
	}
	return o.Limit, true
}

// HasLimit returns a boolean if a field has been set.
func (o *FormulaAndFunctionEventQueryGroupBy) HasLimit() bool {
	if o != nil && o.Limit != nil {
		return true
	}

	return false
}

// SetLimit gets a reference to the given int64 and assigns it to the Limit field.
func (o *FormulaAndFunctionEventQueryGroupBy) SetLimit(v int64) {
	o.Limit = &v
}

// GetSort returns the Sort field value if set, zero value otherwise.
func (o *FormulaAndFunctionEventQueryGroupBy) GetSort() FormulaAndFunctionEventQueryGroupBySort {
	if o == nil || o.Sort == nil {
		var ret FormulaAndFunctionEventQueryGroupBySort
		return ret
	}
	return *o.Sort
}

// GetSortOk returns a tuple with the Sort field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionEventQueryGroupBy) GetSortOk() (*FormulaAndFunctionEventQueryGroupBySort, bool) {
	if o == nil || o.Sort == nil {
		return nil, false
	}
	return o.Sort, true
}

// HasSort returns a boolean if a field has been set.
func (o *FormulaAndFunctionEventQueryGroupBy) HasSort() bool {
	if o != nil && o.Sort != nil {
		return true
	}

	return false
}

// SetSort gets a reference to the given FormulaAndFunctionEventQueryGroupBySort and assigns it to the Sort field.
func (o *FormulaAndFunctionEventQueryGroupBy) SetSort(v FormulaAndFunctionEventQueryGroupBySort) {
	o.Sort = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o FormulaAndFunctionEventQueryGroupBy) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["facet"] = o.Facet
	if o.Limit != nil {
		toSerialize["limit"] = o.Limit
	}
	if o.Sort != nil {
		toSerialize["sort"] = o.Sort
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *FormulaAndFunctionEventQueryGroupBy) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Facet *string `json:"facet"`
	}{}
	all := struct {
		Facet string                                   `json:"facet"`
		Limit *int64                                   `json:"limit,omitempty"`
		Sort  *FormulaAndFunctionEventQueryGroupBySort `json:"sort,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Facet == nil {
		return fmt.Errorf("Required field facet missing")
	}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Facet = all.Facet
	o.Limit = all.Limit
	if all.Sort != nil && all.Sort.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Sort = all.Sort
	return nil
}
