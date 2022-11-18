// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// MonitorFormulaAndFunctionEventQueryGroupBySort Options for sorting group by results.
type MonitorFormulaAndFunctionEventQueryGroupBySort struct {
	// Aggregation methods for event platform queries.
	Aggregation MonitorFormulaAndFunctionEventAggregation `json:"aggregation"`
	// Metric used for sorting group by results.
	Metric *string `json:"metric,omitempty"`
	// Direction of sort.
	Order *QuerySortOrder `json:"order,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMonitorFormulaAndFunctionEventQueryGroupBySort instantiates a new MonitorFormulaAndFunctionEventQueryGroupBySort object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorFormulaAndFunctionEventQueryGroupBySort(aggregation MonitorFormulaAndFunctionEventAggregation) *MonitorFormulaAndFunctionEventQueryGroupBySort {
	this := MonitorFormulaAndFunctionEventQueryGroupBySort{}
	this.Aggregation = aggregation
	var order QuerySortOrder = QUERYSORTORDER_DESC
	this.Order = &order
	return &this
}

// NewMonitorFormulaAndFunctionEventQueryGroupBySortWithDefaults instantiates a new MonitorFormulaAndFunctionEventQueryGroupBySort object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorFormulaAndFunctionEventQueryGroupBySortWithDefaults() *MonitorFormulaAndFunctionEventQueryGroupBySort {
	this := MonitorFormulaAndFunctionEventQueryGroupBySort{}
	var order QuerySortOrder = QUERYSORTORDER_DESC
	this.Order = &order
	return &this
}

// GetAggregation returns the Aggregation field value.
func (o *MonitorFormulaAndFunctionEventQueryGroupBySort) GetAggregation() MonitorFormulaAndFunctionEventAggregation {
	if o == nil {
		var ret MonitorFormulaAndFunctionEventAggregation
		return ret
	}
	return o.Aggregation
}

// GetAggregationOk returns a tuple with the Aggregation field value
// and a boolean to check if the value has been set.
func (o *MonitorFormulaAndFunctionEventQueryGroupBySort) GetAggregationOk() (*MonitorFormulaAndFunctionEventAggregation, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Aggregation, true
}

// SetAggregation sets field value.
func (o *MonitorFormulaAndFunctionEventQueryGroupBySort) SetAggregation(v MonitorFormulaAndFunctionEventAggregation) {
	o.Aggregation = v
}

// GetMetric returns the Metric field value if set, zero value otherwise.
func (o *MonitorFormulaAndFunctionEventQueryGroupBySort) GetMetric() string {
	if o == nil || o.Metric == nil {
		var ret string
		return ret
	}
	return *o.Metric
}

// GetMetricOk returns a tuple with the Metric field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorFormulaAndFunctionEventQueryGroupBySort) GetMetricOk() (*string, bool) {
	if o == nil || o.Metric == nil {
		return nil, false
	}
	return o.Metric, true
}

// HasMetric returns a boolean if a field has been set.
func (o *MonitorFormulaAndFunctionEventQueryGroupBySort) HasMetric() bool {
	if o != nil && o.Metric != nil {
		return true
	}

	return false
}

// SetMetric gets a reference to the given string and assigns it to the Metric field.
func (o *MonitorFormulaAndFunctionEventQueryGroupBySort) SetMetric(v string) {
	o.Metric = &v
}

// GetOrder returns the Order field value if set, zero value otherwise.
func (o *MonitorFormulaAndFunctionEventQueryGroupBySort) GetOrder() QuerySortOrder {
	if o == nil || o.Order == nil {
		var ret QuerySortOrder
		return ret
	}
	return *o.Order
}

// GetOrderOk returns a tuple with the Order field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorFormulaAndFunctionEventQueryGroupBySort) GetOrderOk() (*QuerySortOrder, bool) {
	if o == nil || o.Order == nil {
		return nil, false
	}
	return o.Order, true
}

// HasOrder returns a boolean if a field has been set.
func (o *MonitorFormulaAndFunctionEventQueryGroupBySort) HasOrder() bool {
	if o != nil && o.Order != nil {
		return true
	}

	return false
}

// SetOrder gets a reference to the given QuerySortOrder and assigns it to the Order field.
func (o *MonitorFormulaAndFunctionEventQueryGroupBySort) SetOrder(v QuerySortOrder) {
	o.Order = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorFormulaAndFunctionEventQueryGroupBySort) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["aggregation"] = o.Aggregation
	if o.Metric != nil {
		toSerialize["metric"] = o.Metric
	}
	if o.Order != nil {
		toSerialize["order"] = o.Order
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorFormulaAndFunctionEventQueryGroupBySort) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Aggregation *MonitorFormulaAndFunctionEventAggregation `json:"aggregation"`
	}{}
	all := struct {
		Aggregation MonitorFormulaAndFunctionEventAggregation `json:"aggregation"`
		Metric      *string                                   `json:"metric,omitempty"`
		Order       *QuerySortOrder                           `json:"order,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Aggregation == nil {
		return fmt.Errorf("Required field aggregation missing")
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
	if v := all.Aggregation; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Order; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Aggregation = all.Aggregation
	o.Metric = all.Metric
	o.Order = all.Order
	return nil
}
