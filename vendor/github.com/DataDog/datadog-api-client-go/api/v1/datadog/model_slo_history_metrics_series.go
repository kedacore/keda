// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SLOHistoryMetricsSeries A representation of `metric` based SLO time series for the provided queries.
// This is the same response type from `batch_query` endpoint.
type SLOHistoryMetricsSeries struct {
	// Count of submitted metrics.
	Count int64 `json:"count"`
	// Query metadata.
	Metadata *SLOHistoryMetricsSeriesMetadata `json:"metadata,omitempty"`
	// Total sum of the query.
	Sum float64 `json:"sum"`
	// The query values for each metric.
	Values []float64 `json:"values"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSLOHistoryMetricsSeries instantiates a new SLOHistoryMetricsSeries object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOHistoryMetricsSeries(count int64, sum float64, values []float64) *SLOHistoryMetricsSeries {
	this := SLOHistoryMetricsSeries{}
	this.Count = count
	this.Sum = sum
	this.Values = values
	return &this
}

// NewSLOHistoryMetricsSeriesWithDefaults instantiates a new SLOHistoryMetricsSeries object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOHistoryMetricsSeriesWithDefaults() *SLOHistoryMetricsSeries {
	this := SLOHistoryMetricsSeries{}
	return &this
}

// GetCount returns the Count field value.
func (o *SLOHistoryMetricsSeries) GetCount() int64 {
	if o == nil {
		var ret int64
		return ret
	}
	return o.Count
}

// GetCountOk returns a tuple with the Count field value
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetricsSeries) GetCountOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Count, true
}

// SetCount sets field value.
func (o *SLOHistoryMetricsSeries) SetCount(v int64) {
	o.Count = v
}

// GetMetadata returns the Metadata field value if set, zero value otherwise.
func (o *SLOHistoryMetricsSeries) GetMetadata() SLOHistoryMetricsSeriesMetadata {
	if o == nil || o.Metadata == nil {
		var ret SLOHistoryMetricsSeriesMetadata
		return ret
	}
	return *o.Metadata
}

// GetMetadataOk returns a tuple with the Metadata field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetricsSeries) GetMetadataOk() (*SLOHistoryMetricsSeriesMetadata, bool) {
	if o == nil || o.Metadata == nil {
		return nil, false
	}
	return o.Metadata, true
}

// HasMetadata returns a boolean if a field has been set.
func (o *SLOHistoryMetricsSeries) HasMetadata() bool {
	if o != nil && o.Metadata != nil {
		return true
	}

	return false
}

// SetMetadata gets a reference to the given SLOHistoryMetricsSeriesMetadata and assigns it to the Metadata field.
func (o *SLOHistoryMetricsSeries) SetMetadata(v SLOHistoryMetricsSeriesMetadata) {
	o.Metadata = &v
}

// GetSum returns the Sum field value.
func (o *SLOHistoryMetricsSeries) GetSum() float64 {
	if o == nil {
		var ret float64
		return ret
	}
	return o.Sum
}

// GetSumOk returns a tuple with the Sum field value
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetricsSeries) GetSumOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Sum, true
}

// SetSum sets field value.
func (o *SLOHistoryMetricsSeries) SetSum(v float64) {
	o.Sum = v
}

// GetValues returns the Values field value.
func (o *SLOHistoryMetricsSeries) GetValues() []float64 {
	if o == nil {
		var ret []float64
		return ret
	}
	return o.Values
}

// GetValuesOk returns a tuple with the Values field value
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetricsSeries) GetValuesOk() (*[]float64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Values, true
}

// SetValues sets field value.
func (o *SLOHistoryMetricsSeries) SetValues(v []float64) {
	o.Values = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOHistoryMetricsSeries) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["count"] = o.Count
	if o.Metadata != nil {
		toSerialize["metadata"] = o.Metadata
	}
	toSerialize["sum"] = o.Sum
	toSerialize["values"] = o.Values

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOHistoryMetricsSeries) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Count  *int64     `json:"count"`
		Sum    *float64   `json:"sum"`
		Values *[]float64 `json:"values"`
	}{}
	all := struct {
		Count    int64                            `json:"count"`
		Metadata *SLOHistoryMetricsSeriesMetadata `json:"metadata,omitempty"`
		Sum      float64                          `json:"sum"`
		Values   []float64                        `json:"values"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Count == nil {
		return fmt.Errorf("Required field count missing")
	}
	if required.Sum == nil {
		return fmt.Errorf("Required field sum missing")
	}
	if required.Values == nil {
		return fmt.Errorf("Required field values missing")
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
	o.Count = all.Count
	if all.Metadata != nil && all.Metadata.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Metadata = all.Metadata
	o.Sum = all.Sum
	o.Values = all.Values
	return nil
}
