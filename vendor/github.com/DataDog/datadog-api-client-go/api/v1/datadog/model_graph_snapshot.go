// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// GraphSnapshot Object representing a graph snapshot.
type GraphSnapshot struct {
	// A JSON document defining the graph. `graph_def` can be used instead of `metric_query`.
	// The JSON document uses the [grammar defined here](https://docs.datadoghq.com/graphing/graphing_json/#grammar)
	// and should be formatted to a single line then URL encoded.
	GraphDef *string `json:"graph_def,omitempty"`
	// The metric query. One of `metric_query` or `graph_def` is required.
	MetricQuery *string `json:"metric_query,omitempty"`
	// URL of your [graph snapshot](https://docs.datadoghq.com/metrics/explorer/#snapshot).
	SnapshotUrl *string `json:"snapshot_url,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewGraphSnapshot instantiates a new GraphSnapshot object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewGraphSnapshot() *GraphSnapshot {
	this := GraphSnapshot{}
	return &this
}

// NewGraphSnapshotWithDefaults instantiates a new GraphSnapshot object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewGraphSnapshotWithDefaults() *GraphSnapshot {
	this := GraphSnapshot{}
	return &this
}

// GetGraphDef returns the GraphDef field value if set, zero value otherwise.
func (o *GraphSnapshot) GetGraphDef() string {
	if o == nil || o.GraphDef == nil {
		var ret string
		return ret
	}
	return *o.GraphDef
}

// GetGraphDefOk returns a tuple with the GraphDef field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GraphSnapshot) GetGraphDefOk() (*string, bool) {
	if o == nil || o.GraphDef == nil {
		return nil, false
	}
	return o.GraphDef, true
}

// HasGraphDef returns a boolean if a field has been set.
func (o *GraphSnapshot) HasGraphDef() bool {
	if o != nil && o.GraphDef != nil {
		return true
	}

	return false
}

// SetGraphDef gets a reference to the given string and assigns it to the GraphDef field.
func (o *GraphSnapshot) SetGraphDef(v string) {
	o.GraphDef = &v
}

// GetMetricQuery returns the MetricQuery field value if set, zero value otherwise.
func (o *GraphSnapshot) GetMetricQuery() string {
	if o == nil || o.MetricQuery == nil {
		var ret string
		return ret
	}
	return *o.MetricQuery
}

// GetMetricQueryOk returns a tuple with the MetricQuery field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GraphSnapshot) GetMetricQueryOk() (*string, bool) {
	if o == nil || o.MetricQuery == nil {
		return nil, false
	}
	return o.MetricQuery, true
}

// HasMetricQuery returns a boolean if a field has been set.
func (o *GraphSnapshot) HasMetricQuery() bool {
	if o != nil && o.MetricQuery != nil {
		return true
	}

	return false
}

// SetMetricQuery gets a reference to the given string and assigns it to the MetricQuery field.
func (o *GraphSnapshot) SetMetricQuery(v string) {
	o.MetricQuery = &v
}

// GetSnapshotUrl returns the SnapshotUrl field value if set, zero value otherwise.
func (o *GraphSnapshot) GetSnapshotUrl() string {
	if o == nil || o.SnapshotUrl == nil {
		var ret string
		return ret
	}
	return *o.SnapshotUrl
}

// GetSnapshotUrlOk returns a tuple with the SnapshotUrl field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GraphSnapshot) GetSnapshotUrlOk() (*string, bool) {
	if o == nil || o.SnapshotUrl == nil {
		return nil, false
	}
	return o.SnapshotUrl, true
}

// HasSnapshotUrl returns a boolean if a field has been set.
func (o *GraphSnapshot) HasSnapshotUrl() bool {
	if o != nil && o.SnapshotUrl != nil {
		return true
	}

	return false
}

// SetSnapshotUrl gets a reference to the given string and assigns it to the SnapshotUrl field.
func (o *GraphSnapshot) SetSnapshotUrl(v string) {
	o.SnapshotUrl = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o GraphSnapshot) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.GraphDef != nil {
		toSerialize["graph_def"] = o.GraphDef
	}
	if o.MetricQuery != nil {
		toSerialize["metric_query"] = o.MetricQuery
	}
	if o.SnapshotUrl != nil {
		toSerialize["snapshot_url"] = o.SnapshotUrl
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *GraphSnapshot) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		GraphDef    *string `json:"graph_def,omitempty"`
		MetricQuery *string `json:"metric_query,omitempty"`
		SnapshotUrl *string `json:"snapshot_url,omitempty"`
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
	o.GraphDef = all.GraphDef
	o.MetricQuery = all.MetricQuery
	o.SnapshotUrl = all.SnapshotUrl
	return nil
}
