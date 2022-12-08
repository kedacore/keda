// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// MetricSearchResponseResults Search result.
type MetricSearchResponseResults struct {
	// List of metrics that match the search query.
	Metrics []string `json:"metrics,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMetricSearchResponseResults instantiates a new MetricSearchResponseResults object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricSearchResponseResults() *MetricSearchResponseResults {
	this := MetricSearchResponseResults{}
	return &this
}

// NewMetricSearchResponseResultsWithDefaults instantiates a new MetricSearchResponseResults object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricSearchResponseResultsWithDefaults() *MetricSearchResponseResults {
	this := MetricSearchResponseResults{}
	return &this
}

// GetMetrics returns the Metrics field value if set, zero value otherwise.
func (o *MetricSearchResponseResults) GetMetrics() []string {
	if o == nil || o.Metrics == nil {
		var ret []string
		return ret
	}
	return o.Metrics
}

// GetMetricsOk returns a tuple with the Metrics field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricSearchResponseResults) GetMetricsOk() (*[]string, bool) {
	if o == nil || o.Metrics == nil {
		return nil, false
	}
	return &o.Metrics, true
}

// HasMetrics returns a boolean if a field has been set.
func (o *MetricSearchResponseResults) HasMetrics() bool {
	if o != nil && o.Metrics != nil {
		return true
	}

	return false
}

// SetMetrics gets a reference to the given []string and assigns it to the Metrics field.
func (o *MetricSearchResponseResults) SetMetrics(v []string) {
	o.Metrics = v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricSearchResponseResults) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Metrics != nil {
		toSerialize["metrics"] = o.Metrics
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MetricSearchResponseResults) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Metrics []string `json:"metrics,omitempty"`
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
	o.Metrics = all.Metrics
	return nil
}
