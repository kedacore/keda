// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogQueryDefinitionGroupBySort Define a sorting method.
type LogQueryDefinitionGroupBySort struct {
	// The aggregation method.
	Aggregation string `json:"aggregation"`
	// Facet name.
	Facet *string `json:"facet,omitempty"`
	// Widget sorting methods.
	Order WidgetSort `json:"order"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogQueryDefinitionGroupBySort instantiates a new LogQueryDefinitionGroupBySort object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogQueryDefinitionGroupBySort(aggregation string, order WidgetSort) *LogQueryDefinitionGroupBySort {
	this := LogQueryDefinitionGroupBySort{}
	this.Aggregation = aggregation
	this.Order = order
	return &this
}

// NewLogQueryDefinitionGroupBySortWithDefaults instantiates a new LogQueryDefinitionGroupBySort object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogQueryDefinitionGroupBySortWithDefaults() *LogQueryDefinitionGroupBySort {
	this := LogQueryDefinitionGroupBySort{}
	return &this
}

// GetAggregation returns the Aggregation field value.
func (o *LogQueryDefinitionGroupBySort) GetAggregation() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Aggregation
}

// GetAggregationOk returns a tuple with the Aggregation field value
// and a boolean to check if the value has been set.
func (o *LogQueryDefinitionGroupBySort) GetAggregationOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Aggregation, true
}

// SetAggregation sets field value.
func (o *LogQueryDefinitionGroupBySort) SetAggregation(v string) {
	o.Aggregation = v
}

// GetFacet returns the Facet field value if set, zero value otherwise.
func (o *LogQueryDefinitionGroupBySort) GetFacet() string {
	if o == nil || o.Facet == nil {
		var ret string
		return ret
	}
	return *o.Facet
}

// GetFacetOk returns a tuple with the Facet field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogQueryDefinitionGroupBySort) GetFacetOk() (*string, bool) {
	if o == nil || o.Facet == nil {
		return nil, false
	}
	return o.Facet, true
}

// HasFacet returns a boolean if a field has been set.
func (o *LogQueryDefinitionGroupBySort) HasFacet() bool {
	if o != nil && o.Facet != nil {
		return true
	}

	return false
}

// SetFacet gets a reference to the given string and assigns it to the Facet field.
func (o *LogQueryDefinitionGroupBySort) SetFacet(v string) {
	o.Facet = &v
}

// GetOrder returns the Order field value.
func (o *LogQueryDefinitionGroupBySort) GetOrder() WidgetSort {
	if o == nil {
		var ret WidgetSort
		return ret
	}
	return o.Order
}

// GetOrderOk returns a tuple with the Order field value
// and a boolean to check if the value has been set.
func (o *LogQueryDefinitionGroupBySort) GetOrderOk() (*WidgetSort, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Order, true
}

// SetOrder sets field value.
func (o *LogQueryDefinitionGroupBySort) SetOrder(v WidgetSort) {
	o.Order = v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogQueryDefinitionGroupBySort) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["aggregation"] = o.Aggregation
	if o.Facet != nil {
		toSerialize["facet"] = o.Facet
	}
	toSerialize["order"] = o.Order

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogQueryDefinitionGroupBySort) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Aggregation *string     `json:"aggregation"`
		Order       *WidgetSort `json:"order"`
	}{}
	all := struct {
		Aggregation string     `json:"aggregation"`
		Facet       *string    `json:"facet,omitempty"`
		Order       WidgetSort `json:"order"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Aggregation == nil {
		return fmt.Errorf("Required field aggregation missing")
	}
	if required.Order == nil {
		return fmt.Errorf("Required field order missing")
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
	if v := all.Order; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Aggregation = all.Aggregation
	o.Facet = all.Facet
	o.Order = all.Order
	return nil
}
