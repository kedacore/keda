// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// MetricsQueryMetadata Object containing all metric names returned and their associated metadata.
type MetricsQueryMetadata struct {
	// Aggregation type.
	Aggr NullableString `json:"aggr,omitempty"`
	// Display name of the metric.
	DisplayName *string `json:"display_name,omitempty"`
	// End of the time window, milliseconds since Unix epoch.
	End *int64 `json:"end,omitempty"`
	// Metric expression.
	Expression *string `json:"expression,omitempty"`
	// Number of seconds between data samples.
	Interval *int64 `json:"interval,omitempty"`
	// Number of data samples.
	Length *int64 `json:"length,omitempty"`
	// Metric name.
	Metric *string `json:"metric,omitempty"`
	// List of points of the time series.
	Pointlist [][]*float64 `json:"pointlist,omitempty"`
	// The index of the series' query within the request.
	QueryIndex *int64 `json:"query_index,omitempty"`
	// Metric scope, comma separated list of tags.
	Scope *string `json:"scope,omitempty"`
	// Start of the time window, milliseconds since Unix epoch.
	Start *int64 `json:"start,omitempty"`
	// Unique tags identifying this series.
	TagSet []string `json:"tag_set,omitempty"`
	// Detailed information about the metric unit.
	// First element describes the "primary unit" (for example, `bytes` in `bytes per second`),
	// second describes the "per unit" (for example, `second` in `bytes per second`).
	Unit []MetricsQueryUnit `json:"unit,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMetricsQueryMetadata instantiates a new MetricsQueryMetadata object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricsQueryMetadata() *MetricsQueryMetadata {
	this := MetricsQueryMetadata{}
	return &this
}

// NewMetricsQueryMetadataWithDefaults instantiates a new MetricsQueryMetadata object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricsQueryMetadataWithDefaults() *MetricsQueryMetadata {
	this := MetricsQueryMetadata{}
	return &this
}

// GetAggr returns the Aggr field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MetricsQueryMetadata) GetAggr() string {
	if o == nil || o.Aggr.Get() == nil {
		var ret string
		return ret
	}
	return *o.Aggr.Get()
}

// GetAggrOk returns a tuple with the Aggr field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MetricsQueryMetadata) GetAggrOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Aggr.Get(), o.Aggr.IsSet()
}

// HasAggr returns a boolean if a field has been set.
func (o *MetricsQueryMetadata) HasAggr() bool {
	if o != nil && o.Aggr.IsSet() {
		return true
	}

	return false
}

// SetAggr gets a reference to the given NullableString and assigns it to the Aggr field.
func (o *MetricsQueryMetadata) SetAggr(v string) {
	o.Aggr.Set(&v)
}

// SetAggrNil sets the value for Aggr to be an explicit nil.
func (o *MetricsQueryMetadata) SetAggrNil() {
	o.Aggr.Set(nil)
}

// UnsetAggr ensures that no value is present for Aggr, not even an explicit nil.
func (o *MetricsQueryMetadata) UnsetAggr() {
	o.Aggr.Unset()
}

// GetDisplayName returns the DisplayName field value if set, zero value otherwise.
func (o *MetricsQueryMetadata) GetDisplayName() string {
	if o == nil || o.DisplayName == nil {
		var ret string
		return ret
	}
	return *o.DisplayName
}

// GetDisplayNameOk returns a tuple with the DisplayName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryMetadata) GetDisplayNameOk() (*string, bool) {
	if o == nil || o.DisplayName == nil {
		return nil, false
	}
	return o.DisplayName, true
}

// HasDisplayName returns a boolean if a field has been set.
func (o *MetricsQueryMetadata) HasDisplayName() bool {
	if o != nil && o.DisplayName != nil {
		return true
	}

	return false
}

// SetDisplayName gets a reference to the given string and assigns it to the DisplayName field.
func (o *MetricsQueryMetadata) SetDisplayName(v string) {
	o.DisplayName = &v
}

// GetEnd returns the End field value if set, zero value otherwise.
func (o *MetricsQueryMetadata) GetEnd() int64 {
	if o == nil || o.End == nil {
		var ret int64
		return ret
	}
	return *o.End
}

// GetEndOk returns a tuple with the End field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryMetadata) GetEndOk() (*int64, bool) {
	if o == nil || o.End == nil {
		return nil, false
	}
	return o.End, true
}

// HasEnd returns a boolean if a field has been set.
func (o *MetricsQueryMetadata) HasEnd() bool {
	if o != nil && o.End != nil {
		return true
	}

	return false
}

// SetEnd gets a reference to the given int64 and assigns it to the End field.
func (o *MetricsQueryMetadata) SetEnd(v int64) {
	o.End = &v
}

// GetExpression returns the Expression field value if set, zero value otherwise.
func (o *MetricsQueryMetadata) GetExpression() string {
	if o == nil || o.Expression == nil {
		var ret string
		return ret
	}
	return *o.Expression
}

// GetExpressionOk returns a tuple with the Expression field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryMetadata) GetExpressionOk() (*string, bool) {
	if o == nil || o.Expression == nil {
		return nil, false
	}
	return o.Expression, true
}

// HasExpression returns a boolean if a field has been set.
func (o *MetricsQueryMetadata) HasExpression() bool {
	if o != nil && o.Expression != nil {
		return true
	}

	return false
}

// SetExpression gets a reference to the given string and assigns it to the Expression field.
func (o *MetricsQueryMetadata) SetExpression(v string) {
	o.Expression = &v
}

// GetInterval returns the Interval field value if set, zero value otherwise.
func (o *MetricsQueryMetadata) GetInterval() int64 {
	if o == nil || o.Interval == nil {
		var ret int64
		return ret
	}
	return *o.Interval
}

// GetIntervalOk returns a tuple with the Interval field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryMetadata) GetIntervalOk() (*int64, bool) {
	if o == nil || o.Interval == nil {
		return nil, false
	}
	return o.Interval, true
}

// HasInterval returns a boolean if a field has been set.
func (o *MetricsQueryMetadata) HasInterval() bool {
	if o != nil && o.Interval != nil {
		return true
	}

	return false
}

// SetInterval gets a reference to the given int64 and assigns it to the Interval field.
func (o *MetricsQueryMetadata) SetInterval(v int64) {
	o.Interval = &v
}

// GetLength returns the Length field value if set, zero value otherwise.
func (o *MetricsQueryMetadata) GetLength() int64 {
	if o == nil || o.Length == nil {
		var ret int64
		return ret
	}
	return *o.Length
}

// GetLengthOk returns a tuple with the Length field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryMetadata) GetLengthOk() (*int64, bool) {
	if o == nil || o.Length == nil {
		return nil, false
	}
	return o.Length, true
}

// HasLength returns a boolean if a field has been set.
func (o *MetricsQueryMetadata) HasLength() bool {
	if o != nil && o.Length != nil {
		return true
	}

	return false
}

// SetLength gets a reference to the given int64 and assigns it to the Length field.
func (o *MetricsQueryMetadata) SetLength(v int64) {
	o.Length = &v
}

// GetMetric returns the Metric field value if set, zero value otherwise.
func (o *MetricsQueryMetadata) GetMetric() string {
	if o == nil || o.Metric == nil {
		var ret string
		return ret
	}
	return *o.Metric
}

// GetMetricOk returns a tuple with the Metric field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryMetadata) GetMetricOk() (*string, bool) {
	if o == nil || o.Metric == nil {
		return nil, false
	}
	return o.Metric, true
}

// HasMetric returns a boolean if a field has been set.
func (o *MetricsQueryMetadata) HasMetric() bool {
	if o != nil && o.Metric != nil {
		return true
	}

	return false
}

// SetMetric gets a reference to the given string and assigns it to the Metric field.
func (o *MetricsQueryMetadata) SetMetric(v string) {
	o.Metric = &v
}

// GetPointlist returns the Pointlist field value if set, zero value otherwise.
func (o *MetricsQueryMetadata) GetPointlist() [][]*float64 {
	if o == nil || o.Pointlist == nil {
		var ret [][]*float64
		return ret
	}
	return o.Pointlist
}

// GetPointlistOk returns a tuple with the Pointlist field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryMetadata) GetPointlistOk() (*[][]*float64, bool) {
	if o == nil || o.Pointlist == nil {
		return nil, false
	}
	return &o.Pointlist, true
}

// HasPointlist returns a boolean if a field has been set.
func (o *MetricsQueryMetadata) HasPointlist() bool {
	if o != nil && o.Pointlist != nil {
		return true
	}

	return false
}

// SetPointlist gets a reference to the given [][]*float64 and assigns it to the Pointlist field.
func (o *MetricsQueryMetadata) SetPointlist(v [][]*float64) {
	o.Pointlist = v
}

// GetQueryIndex returns the QueryIndex field value if set, zero value otherwise.
func (o *MetricsQueryMetadata) GetQueryIndex() int64 {
	if o == nil || o.QueryIndex == nil {
		var ret int64
		return ret
	}
	return *o.QueryIndex
}

// GetQueryIndexOk returns a tuple with the QueryIndex field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryMetadata) GetQueryIndexOk() (*int64, bool) {
	if o == nil || o.QueryIndex == nil {
		return nil, false
	}
	return o.QueryIndex, true
}

// HasQueryIndex returns a boolean if a field has been set.
func (o *MetricsQueryMetadata) HasQueryIndex() bool {
	if o != nil && o.QueryIndex != nil {
		return true
	}

	return false
}

// SetQueryIndex gets a reference to the given int64 and assigns it to the QueryIndex field.
func (o *MetricsQueryMetadata) SetQueryIndex(v int64) {
	o.QueryIndex = &v
}

// GetScope returns the Scope field value if set, zero value otherwise.
func (o *MetricsQueryMetadata) GetScope() string {
	if o == nil || o.Scope == nil {
		var ret string
		return ret
	}
	return *o.Scope
}

// GetScopeOk returns a tuple with the Scope field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryMetadata) GetScopeOk() (*string, bool) {
	if o == nil || o.Scope == nil {
		return nil, false
	}
	return o.Scope, true
}

// HasScope returns a boolean if a field has been set.
func (o *MetricsQueryMetadata) HasScope() bool {
	if o != nil && o.Scope != nil {
		return true
	}

	return false
}

// SetScope gets a reference to the given string and assigns it to the Scope field.
func (o *MetricsQueryMetadata) SetScope(v string) {
	o.Scope = &v
}

// GetStart returns the Start field value if set, zero value otherwise.
func (o *MetricsQueryMetadata) GetStart() int64 {
	if o == nil || o.Start == nil {
		var ret int64
		return ret
	}
	return *o.Start
}

// GetStartOk returns a tuple with the Start field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryMetadata) GetStartOk() (*int64, bool) {
	if o == nil || o.Start == nil {
		return nil, false
	}
	return o.Start, true
}

// HasStart returns a boolean if a field has been set.
func (o *MetricsQueryMetadata) HasStart() bool {
	if o != nil && o.Start != nil {
		return true
	}

	return false
}

// SetStart gets a reference to the given int64 and assigns it to the Start field.
func (o *MetricsQueryMetadata) SetStart(v int64) {
	o.Start = &v
}

// GetTagSet returns the TagSet field value if set, zero value otherwise.
func (o *MetricsQueryMetadata) GetTagSet() []string {
	if o == nil || o.TagSet == nil {
		var ret []string
		return ret
	}
	return o.TagSet
}

// GetTagSetOk returns a tuple with the TagSet field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryMetadata) GetTagSetOk() (*[]string, bool) {
	if o == nil || o.TagSet == nil {
		return nil, false
	}
	return &o.TagSet, true
}

// HasTagSet returns a boolean if a field has been set.
func (o *MetricsQueryMetadata) HasTagSet() bool {
	if o != nil && o.TagSet != nil {
		return true
	}

	return false
}

// SetTagSet gets a reference to the given []string and assigns it to the TagSet field.
func (o *MetricsQueryMetadata) SetTagSet(v []string) {
	o.TagSet = v
}

// GetUnit returns the Unit field value if set, zero value otherwise.
func (o *MetricsQueryMetadata) GetUnit() []MetricsQueryUnit {
	if o == nil || o.Unit == nil {
		var ret []MetricsQueryUnit
		return ret
	}
	return o.Unit
}

// GetUnitOk returns a tuple with the Unit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryMetadata) GetUnitOk() (*[]MetricsQueryUnit, bool) {
	if o == nil || o.Unit == nil {
		return nil, false
	}
	return &o.Unit, true
}

// HasUnit returns a boolean if a field has been set.
func (o *MetricsQueryMetadata) HasUnit() bool {
	if o != nil && o.Unit != nil {
		return true
	}

	return false
}

// SetUnit gets a reference to the given []MetricsQueryUnit and assigns it to the Unit field.
func (o *MetricsQueryMetadata) SetUnit(v []MetricsQueryUnit) {
	o.Unit = v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricsQueryMetadata) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Aggr.IsSet() {
		toSerialize["aggr"] = o.Aggr.Get()
	}
	if o.DisplayName != nil {
		toSerialize["display_name"] = o.DisplayName
	}
	if o.End != nil {
		toSerialize["end"] = o.End
	}
	if o.Expression != nil {
		toSerialize["expression"] = o.Expression
	}
	if o.Interval != nil {
		toSerialize["interval"] = o.Interval
	}
	if o.Length != nil {
		toSerialize["length"] = o.Length
	}
	if o.Metric != nil {
		toSerialize["metric"] = o.Metric
	}
	if o.Pointlist != nil {
		toSerialize["pointlist"] = o.Pointlist
	}
	if o.QueryIndex != nil {
		toSerialize["query_index"] = o.QueryIndex
	}
	if o.Scope != nil {
		toSerialize["scope"] = o.Scope
	}
	if o.Start != nil {
		toSerialize["start"] = o.Start
	}
	if o.TagSet != nil {
		toSerialize["tag_set"] = o.TagSet
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
func (o *MetricsQueryMetadata) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Aggr        NullableString     `json:"aggr,omitempty"`
		DisplayName *string            `json:"display_name,omitempty"`
		End         *int64             `json:"end,omitempty"`
		Expression  *string            `json:"expression,omitempty"`
		Interval    *int64             `json:"interval,omitempty"`
		Length      *int64             `json:"length,omitempty"`
		Metric      *string            `json:"metric,omitempty"`
		Pointlist   [][]*float64       `json:"pointlist,omitempty"`
		QueryIndex  *int64             `json:"query_index,omitempty"`
		Scope       *string            `json:"scope,omitempty"`
		Start       *int64             `json:"start,omitempty"`
		TagSet      []string           `json:"tag_set,omitempty"`
		Unit        []MetricsQueryUnit `json:"unit,omitempty"`
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
	o.DisplayName = all.DisplayName
	o.End = all.End
	o.Expression = all.Expression
	o.Interval = all.Interval
	o.Length = all.Length
	o.Metric = all.Metric
	o.Pointlist = all.Pointlist
	o.QueryIndex = all.QueryIndex
	o.Scope = all.Scope
	o.Start = all.Start
	o.TagSet = all.TagSet
	o.Unit = all.Unit
	return nil
}
