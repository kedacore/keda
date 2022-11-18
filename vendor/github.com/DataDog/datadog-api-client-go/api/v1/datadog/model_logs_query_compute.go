// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsQueryCompute Define computation for a log query.
type LogsQueryCompute struct {
	// The aggregation method.
	Aggregation string `json:"aggregation"`
	// Facet name.
	Facet *string `json:"facet,omitempty"`
	// Define a time interval in seconds.
	Interval *int64 `json:"interval,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsQueryCompute instantiates a new LogsQueryCompute object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsQueryCompute(aggregation string) *LogsQueryCompute {
	this := LogsQueryCompute{}
	this.Aggregation = aggregation
	return &this
}

// NewLogsQueryComputeWithDefaults instantiates a new LogsQueryCompute object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsQueryComputeWithDefaults() *LogsQueryCompute {
	this := LogsQueryCompute{}
	return &this
}

// GetAggregation returns the Aggregation field value.
func (o *LogsQueryCompute) GetAggregation() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Aggregation
}

// GetAggregationOk returns a tuple with the Aggregation field value
// and a boolean to check if the value has been set.
func (o *LogsQueryCompute) GetAggregationOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Aggregation, true
}

// SetAggregation sets field value.
func (o *LogsQueryCompute) SetAggregation(v string) {
	o.Aggregation = v
}

// GetFacet returns the Facet field value if set, zero value otherwise.
func (o *LogsQueryCompute) GetFacet() string {
	if o == nil || o.Facet == nil {
		var ret string
		return ret
	}
	return *o.Facet
}

// GetFacetOk returns a tuple with the Facet field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsQueryCompute) GetFacetOk() (*string, bool) {
	if o == nil || o.Facet == nil {
		return nil, false
	}
	return o.Facet, true
}

// HasFacet returns a boolean if a field has been set.
func (o *LogsQueryCompute) HasFacet() bool {
	if o != nil && o.Facet != nil {
		return true
	}

	return false
}

// SetFacet gets a reference to the given string and assigns it to the Facet field.
func (o *LogsQueryCompute) SetFacet(v string) {
	o.Facet = &v
}

// GetInterval returns the Interval field value if set, zero value otherwise.
func (o *LogsQueryCompute) GetInterval() int64 {
	if o == nil || o.Interval == nil {
		var ret int64
		return ret
	}
	return *o.Interval
}

// GetIntervalOk returns a tuple with the Interval field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsQueryCompute) GetIntervalOk() (*int64, bool) {
	if o == nil || o.Interval == nil {
		return nil, false
	}
	return o.Interval, true
}

// HasInterval returns a boolean if a field has been set.
func (o *LogsQueryCompute) HasInterval() bool {
	if o != nil && o.Interval != nil {
		return true
	}

	return false
}

// SetInterval gets a reference to the given int64 and assigns it to the Interval field.
func (o *LogsQueryCompute) SetInterval(v int64) {
	o.Interval = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsQueryCompute) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["aggregation"] = o.Aggregation
	if o.Facet != nil {
		toSerialize["facet"] = o.Facet
	}
	if o.Interval != nil {
		toSerialize["interval"] = o.Interval
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsQueryCompute) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Aggregation *string `json:"aggregation"`
	}{}
	all := struct {
		Aggregation string  `json:"aggregation"`
		Facet       *string `json:"facet,omitempty"`
		Interval    *int64  `json:"interval,omitempty"`
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
	o.Aggregation = all.Aggregation
	o.Facet = all.Facet
	o.Interval = all.Interval
	return nil
}
