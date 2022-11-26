// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// MonitorFormulaAndFunctionEventQueryDefinitionCompute Compute options.
type MonitorFormulaAndFunctionEventQueryDefinitionCompute struct {
	// Aggregation methods for event platform queries.
	Aggregation MonitorFormulaAndFunctionEventAggregation `json:"aggregation"`
	// A time interval in milliseconds.
	Interval *int64 `json:"interval,omitempty"`
	// Measurable attribute to compute.
	Metric *string `json:"metric,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMonitorFormulaAndFunctionEventQueryDefinitionCompute instantiates a new MonitorFormulaAndFunctionEventQueryDefinitionCompute object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorFormulaAndFunctionEventQueryDefinitionCompute(aggregation MonitorFormulaAndFunctionEventAggregation) *MonitorFormulaAndFunctionEventQueryDefinitionCompute {
	this := MonitorFormulaAndFunctionEventQueryDefinitionCompute{}
	this.Aggregation = aggregation
	return &this
}

// NewMonitorFormulaAndFunctionEventQueryDefinitionComputeWithDefaults instantiates a new MonitorFormulaAndFunctionEventQueryDefinitionCompute object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorFormulaAndFunctionEventQueryDefinitionComputeWithDefaults() *MonitorFormulaAndFunctionEventQueryDefinitionCompute {
	this := MonitorFormulaAndFunctionEventQueryDefinitionCompute{}
	return &this
}

// GetAggregation returns the Aggregation field value.
func (o *MonitorFormulaAndFunctionEventQueryDefinitionCompute) GetAggregation() MonitorFormulaAndFunctionEventAggregation {
	if o == nil {
		var ret MonitorFormulaAndFunctionEventAggregation
		return ret
	}
	return o.Aggregation
}

// GetAggregationOk returns a tuple with the Aggregation field value
// and a boolean to check if the value has been set.
func (o *MonitorFormulaAndFunctionEventQueryDefinitionCompute) GetAggregationOk() (*MonitorFormulaAndFunctionEventAggregation, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Aggregation, true
}

// SetAggregation sets field value.
func (o *MonitorFormulaAndFunctionEventQueryDefinitionCompute) SetAggregation(v MonitorFormulaAndFunctionEventAggregation) {
	o.Aggregation = v
}

// GetInterval returns the Interval field value if set, zero value otherwise.
func (o *MonitorFormulaAndFunctionEventQueryDefinitionCompute) GetInterval() int64 {
	if o == nil || o.Interval == nil {
		var ret int64
		return ret
	}
	return *o.Interval
}

// GetIntervalOk returns a tuple with the Interval field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorFormulaAndFunctionEventQueryDefinitionCompute) GetIntervalOk() (*int64, bool) {
	if o == nil || o.Interval == nil {
		return nil, false
	}
	return o.Interval, true
}

// HasInterval returns a boolean if a field has been set.
func (o *MonitorFormulaAndFunctionEventQueryDefinitionCompute) HasInterval() bool {
	if o != nil && o.Interval != nil {
		return true
	}

	return false
}

// SetInterval gets a reference to the given int64 and assigns it to the Interval field.
func (o *MonitorFormulaAndFunctionEventQueryDefinitionCompute) SetInterval(v int64) {
	o.Interval = &v
}

// GetMetric returns the Metric field value if set, zero value otherwise.
func (o *MonitorFormulaAndFunctionEventQueryDefinitionCompute) GetMetric() string {
	if o == nil || o.Metric == nil {
		var ret string
		return ret
	}
	return *o.Metric
}

// GetMetricOk returns a tuple with the Metric field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorFormulaAndFunctionEventQueryDefinitionCompute) GetMetricOk() (*string, bool) {
	if o == nil || o.Metric == nil {
		return nil, false
	}
	return o.Metric, true
}

// HasMetric returns a boolean if a field has been set.
func (o *MonitorFormulaAndFunctionEventQueryDefinitionCompute) HasMetric() bool {
	if o != nil && o.Metric != nil {
		return true
	}

	return false
}

// SetMetric gets a reference to the given string and assigns it to the Metric field.
func (o *MonitorFormulaAndFunctionEventQueryDefinitionCompute) SetMetric(v string) {
	o.Metric = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorFormulaAndFunctionEventQueryDefinitionCompute) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["aggregation"] = o.Aggregation
	if o.Interval != nil {
		toSerialize["interval"] = o.Interval
	}
	if o.Metric != nil {
		toSerialize["metric"] = o.Metric
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorFormulaAndFunctionEventQueryDefinitionCompute) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Aggregation *MonitorFormulaAndFunctionEventAggregation `json:"aggregation"`
	}{}
	all := struct {
		Aggregation MonitorFormulaAndFunctionEventAggregation `json:"aggregation"`
		Interval    *int64                                    `json:"interval,omitempty"`
		Metric      *string                                   `json:"metric,omitempty"`
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
	o.Aggregation = all.Aggregation
	o.Interval = all.Interval
	o.Metric = all.Metric
	return nil
}
