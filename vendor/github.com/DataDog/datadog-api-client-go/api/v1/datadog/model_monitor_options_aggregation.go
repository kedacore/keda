// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// MonitorOptionsAggregation Type of aggregation performed in the monitor query.
type MonitorOptionsAggregation struct {
	// Group to break down the monitor on.
	GroupBy *string `json:"group_by,omitempty"`
	// Metric name used in the monitor.
	Metric *string `json:"metric,omitempty"`
	// Metric type used in the monitor.
	Type *string `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMonitorOptionsAggregation instantiates a new MonitorOptionsAggregation object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorOptionsAggregation() *MonitorOptionsAggregation {
	this := MonitorOptionsAggregation{}
	return &this
}

// NewMonitorOptionsAggregationWithDefaults instantiates a new MonitorOptionsAggregation object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorOptionsAggregationWithDefaults() *MonitorOptionsAggregation {
	this := MonitorOptionsAggregation{}
	return &this
}

// GetGroupBy returns the GroupBy field value if set, zero value otherwise.
func (o *MonitorOptionsAggregation) GetGroupBy() string {
	if o == nil || o.GroupBy == nil {
		var ret string
		return ret
	}
	return *o.GroupBy
}

// GetGroupByOk returns a tuple with the GroupBy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptionsAggregation) GetGroupByOk() (*string, bool) {
	if o == nil || o.GroupBy == nil {
		return nil, false
	}
	return o.GroupBy, true
}

// HasGroupBy returns a boolean if a field has been set.
func (o *MonitorOptionsAggregation) HasGroupBy() bool {
	if o != nil && o.GroupBy != nil {
		return true
	}

	return false
}

// SetGroupBy gets a reference to the given string and assigns it to the GroupBy field.
func (o *MonitorOptionsAggregation) SetGroupBy(v string) {
	o.GroupBy = &v
}

// GetMetric returns the Metric field value if set, zero value otherwise.
func (o *MonitorOptionsAggregation) GetMetric() string {
	if o == nil || o.Metric == nil {
		var ret string
		return ret
	}
	return *o.Metric
}

// GetMetricOk returns a tuple with the Metric field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptionsAggregation) GetMetricOk() (*string, bool) {
	if o == nil || o.Metric == nil {
		return nil, false
	}
	return o.Metric, true
}

// HasMetric returns a boolean if a field has been set.
func (o *MonitorOptionsAggregation) HasMetric() bool {
	if o != nil && o.Metric != nil {
		return true
	}

	return false
}

// SetMetric gets a reference to the given string and assigns it to the Metric field.
func (o *MonitorOptionsAggregation) SetMetric(v string) {
	o.Metric = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *MonitorOptionsAggregation) GetType() string {
	if o == nil || o.Type == nil {
		var ret string
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptionsAggregation) GetTypeOk() (*string, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *MonitorOptionsAggregation) HasType() bool {
	if o != nil && o.Type != nil {
		return true
	}

	return false
}

// SetType gets a reference to the given string and assigns it to the Type field.
func (o *MonitorOptionsAggregation) SetType(v string) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorOptionsAggregation) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.GroupBy != nil {
		toSerialize["group_by"] = o.GroupBy
	}
	if o.Metric != nil {
		toSerialize["metric"] = o.Metric
	}
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorOptionsAggregation) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		GroupBy *string `json:"group_by,omitempty"`
		Metric  *string `json:"metric,omitempty"`
		Type    *string `json:"type,omitempty"`
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
	o.GroupBy = all.GroupBy
	o.Metric = all.Metric
	o.Type = all.Type
	return nil
}
