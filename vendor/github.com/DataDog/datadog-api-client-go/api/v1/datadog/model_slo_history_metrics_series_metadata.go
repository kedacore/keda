// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SLOHistoryMetricsSeriesMetadata Query metadata.
type SLOHistoryMetricsSeriesMetadata struct {
	// Query aggregator function.
	Aggr *string `json:"aggr,omitempty"`
	// Query expression.
	Expression *string `json:"expression,omitempty"`
	// Query metric used.
	Metric *string `json:"metric,omitempty"`
	// Query index from original combined query.
	QueryIndex *int64 `json:"query_index,omitempty"`
	// Query scope.
	Scope *string `json:"scope,omitempty"`
	// An array of metric units that contains up to two unit objects.
	// For example, bytes represents one unit object and bytes per second represents two unit objects.
	// If a metric query only has one unit object, the second array element is null.
	Unit []SLOHistoryMetricsSeriesMetadataUnit `json:"unit,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSLOHistoryMetricsSeriesMetadata instantiates a new SLOHistoryMetricsSeriesMetadata object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOHistoryMetricsSeriesMetadata() *SLOHistoryMetricsSeriesMetadata {
	this := SLOHistoryMetricsSeriesMetadata{}
	return &this
}

// NewSLOHistoryMetricsSeriesMetadataWithDefaults instantiates a new SLOHistoryMetricsSeriesMetadata object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOHistoryMetricsSeriesMetadataWithDefaults() *SLOHistoryMetricsSeriesMetadata {
	this := SLOHistoryMetricsSeriesMetadata{}
	return &this
}

// GetAggr returns the Aggr field value if set, zero value otherwise.
func (o *SLOHistoryMetricsSeriesMetadata) GetAggr() string {
	if o == nil || o.Aggr == nil {
		var ret string
		return ret
	}
	return *o.Aggr
}

// GetAggrOk returns a tuple with the Aggr field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetricsSeriesMetadata) GetAggrOk() (*string, bool) {
	if o == nil || o.Aggr == nil {
		return nil, false
	}
	return o.Aggr, true
}

// HasAggr returns a boolean if a field has been set.
func (o *SLOHistoryMetricsSeriesMetadata) HasAggr() bool {
	if o != nil && o.Aggr != nil {
		return true
	}

	return false
}

// SetAggr gets a reference to the given string and assigns it to the Aggr field.
func (o *SLOHistoryMetricsSeriesMetadata) SetAggr(v string) {
	o.Aggr = &v
}

// GetExpression returns the Expression field value if set, zero value otherwise.
func (o *SLOHistoryMetricsSeriesMetadata) GetExpression() string {
	if o == nil || o.Expression == nil {
		var ret string
		return ret
	}
	return *o.Expression
}

// GetExpressionOk returns a tuple with the Expression field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetricsSeriesMetadata) GetExpressionOk() (*string, bool) {
	if o == nil || o.Expression == nil {
		return nil, false
	}
	return o.Expression, true
}

// HasExpression returns a boolean if a field has been set.
func (o *SLOHistoryMetricsSeriesMetadata) HasExpression() bool {
	if o != nil && o.Expression != nil {
		return true
	}

	return false
}

// SetExpression gets a reference to the given string and assigns it to the Expression field.
func (o *SLOHistoryMetricsSeriesMetadata) SetExpression(v string) {
	o.Expression = &v
}

// GetMetric returns the Metric field value if set, zero value otherwise.
func (o *SLOHistoryMetricsSeriesMetadata) GetMetric() string {
	if o == nil || o.Metric == nil {
		var ret string
		return ret
	}
	return *o.Metric
}

// GetMetricOk returns a tuple with the Metric field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetricsSeriesMetadata) GetMetricOk() (*string, bool) {
	if o == nil || o.Metric == nil {
		return nil, false
	}
	return o.Metric, true
}

// HasMetric returns a boolean if a field has been set.
func (o *SLOHistoryMetricsSeriesMetadata) HasMetric() bool {
	if o != nil && o.Metric != nil {
		return true
	}

	return false
}

// SetMetric gets a reference to the given string and assigns it to the Metric field.
func (o *SLOHistoryMetricsSeriesMetadata) SetMetric(v string) {
	o.Metric = &v
}

// GetQueryIndex returns the QueryIndex field value if set, zero value otherwise.
func (o *SLOHistoryMetricsSeriesMetadata) GetQueryIndex() int64 {
	if o == nil || o.QueryIndex == nil {
		var ret int64
		return ret
	}
	return *o.QueryIndex
}

// GetQueryIndexOk returns a tuple with the QueryIndex field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetricsSeriesMetadata) GetQueryIndexOk() (*int64, bool) {
	if o == nil || o.QueryIndex == nil {
		return nil, false
	}
	return o.QueryIndex, true
}

// HasQueryIndex returns a boolean if a field has been set.
func (o *SLOHistoryMetricsSeriesMetadata) HasQueryIndex() bool {
	if o != nil && o.QueryIndex != nil {
		return true
	}

	return false
}

// SetQueryIndex gets a reference to the given int64 and assigns it to the QueryIndex field.
func (o *SLOHistoryMetricsSeriesMetadata) SetQueryIndex(v int64) {
	o.QueryIndex = &v
}

// GetScope returns the Scope field value if set, zero value otherwise.
func (o *SLOHistoryMetricsSeriesMetadata) GetScope() string {
	if o == nil || o.Scope == nil {
		var ret string
		return ret
	}
	return *o.Scope
}

// GetScopeOk returns a tuple with the Scope field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetricsSeriesMetadata) GetScopeOk() (*string, bool) {
	if o == nil || o.Scope == nil {
		return nil, false
	}
	return o.Scope, true
}

// HasScope returns a boolean if a field has been set.
func (o *SLOHistoryMetricsSeriesMetadata) HasScope() bool {
	if o != nil && o.Scope != nil {
		return true
	}

	return false
}

// SetScope gets a reference to the given string and assigns it to the Scope field.
func (o *SLOHistoryMetricsSeriesMetadata) SetScope(v string) {
	o.Scope = &v
}

// GetUnit returns the Unit field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SLOHistoryMetricsSeriesMetadata) GetUnit() []SLOHistoryMetricsSeriesMetadataUnit {
	if o == nil {
		var ret []SLOHistoryMetricsSeriesMetadataUnit
		return ret
	}
	return o.Unit
}

// GetUnitOk returns a tuple with the Unit field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SLOHistoryMetricsSeriesMetadata) GetUnitOk() (*[]SLOHistoryMetricsSeriesMetadataUnit, bool) {
	if o == nil || o.Unit == nil {
		return nil, false
	}
	return &o.Unit, true
}

// HasUnit returns a boolean if a field has been set.
func (o *SLOHistoryMetricsSeriesMetadata) HasUnit() bool {
	if o != nil && o.Unit != nil {
		return true
	}

	return false
}

// SetUnit gets a reference to the given []SLOHistoryMetricsSeriesMetadataUnit and assigns it to the Unit field.
func (o *SLOHistoryMetricsSeriesMetadata) SetUnit(v []SLOHistoryMetricsSeriesMetadataUnit) {
	o.Unit = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOHistoryMetricsSeriesMetadata) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Aggr != nil {
		toSerialize["aggr"] = o.Aggr
	}
	if o.Expression != nil {
		toSerialize["expression"] = o.Expression
	}
	if o.Metric != nil {
		toSerialize["metric"] = o.Metric
	}
	if o.QueryIndex != nil {
		toSerialize["query_index"] = o.QueryIndex
	}
	if o.Scope != nil {
		toSerialize["scope"] = o.Scope
	}
	if o.Unit != nil {
		toSerialize["unit"] = o.Unit
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOHistoryMetricsSeriesMetadata) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Aggr       *string                               `json:"aggr,omitempty"`
		Expression *string                               `json:"expression,omitempty"`
		Metric     *string                               `json:"metric,omitempty"`
		QueryIndex *int64                                `json:"query_index,omitempty"`
		Scope      *string                               `json:"scope,omitempty"`
		Unit       []SLOHistoryMetricsSeriesMetadataUnit `json:"unit,omitempty"`
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
	o.Aggr = all.Aggr
	o.Expression = all.Expression
	o.Metric = all.Metric
	o.QueryIndex = all.QueryIndex
	o.Scope = all.Scope
	o.Unit = all.Unit
	return nil
}
