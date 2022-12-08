// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// Series A metric to submit to Datadog.
// See [Datadog metrics](https://docs.datadoghq.com/developers/metrics/#custom-metrics-properties).
type Series struct {
	// The name of the host that produced the metric.
	Host *string `json:"host,omitempty"`
	// If the type of the metric is rate or count, define the corresponding interval.
	Interval NullableInt64 `json:"interval,omitempty"`
	// The name of the timeseries.
	Metric string `json:"metric"`
	// Points relating to a metric. All points must be tuples with timestamp and a scalar value (cannot be a string). Timestamps should be in POSIX time in seconds, and cannot be more than ten minutes in the future or more than one hour in the past.
	Points [][]*float64 `json:"points"`
	// A list of tags associated with the metric.
	Tags []string `json:"tags,omitempty"`
	// The type of the metric. Valid types are "",`count`, `gauge`, and `rate`.
	Type *string `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSeries instantiates a new Series object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSeries(metric string, points [][]*float64) *Series {
	this := Series{}
	this.Interval = *NewNullableInt64(nil)
	this.Metric = metric
	this.Points = points
	var typeVar string = ""
	this.Type = &typeVar
	return &this
}

// NewSeriesWithDefaults instantiates a new Series object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSeriesWithDefaults() *Series {
	this := Series{}
	this.Interval = *NewNullableInt64(nil)
	var typeVar string = ""
	this.Type = &typeVar
	return &this
}

// GetHost returns the Host field value if set, zero value otherwise.
func (o *Series) GetHost() string {
	if o == nil || o.Host == nil {
		var ret string
		return ret
	}
	return *o.Host
}

// GetHostOk returns a tuple with the Host field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Series) GetHostOk() (*string, bool) {
	if o == nil || o.Host == nil {
		return nil, false
	}
	return o.Host, true
}

// HasHost returns a boolean if a field has been set.
func (o *Series) HasHost() bool {
	if o != nil && o.Host != nil {
		return true
	}

	return false
}

// SetHost gets a reference to the given string and assigns it to the Host field.
func (o *Series) SetHost(v string) {
	o.Host = &v
}

// GetInterval returns the Interval field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *Series) GetInterval() int64 {
	if o == nil || o.Interval.Get() == nil {
		var ret int64
		return ret
	}
	return *o.Interval.Get()
}

// GetIntervalOk returns a tuple with the Interval field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *Series) GetIntervalOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.Interval.Get(), o.Interval.IsSet()
}

// HasInterval returns a boolean if a field has been set.
func (o *Series) HasInterval() bool {
	if o != nil && o.Interval.IsSet() {
		return true
	}

	return false
}

// SetInterval gets a reference to the given NullableInt64 and assigns it to the Interval field.
func (o *Series) SetInterval(v int64) {
	o.Interval.Set(&v)
}

// SetIntervalNil sets the value for Interval to be an explicit nil.
func (o *Series) SetIntervalNil() {
	o.Interval.Set(nil)
}

// UnsetInterval ensures that no value is present for Interval, not even an explicit nil.
func (o *Series) UnsetInterval() {
	o.Interval.Unset()
}

// GetMetric returns the Metric field value.
func (o *Series) GetMetric() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Metric
}

// GetMetricOk returns a tuple with the Metric field value
// and a boolean to check if the value has been set.
func (o *Series) GetMetricOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Metric, true
}

// SetMetric sets field value.
func (o *Series) SetMetric(v string) {
	o.Metric = v
}

// GetPoints returns the Points field value.
func (o *Series) GetPoints() [][]*float64 {
	if o == nil {
		var ret [][]*float64
		return ret
	}
	return o.Points
}

// GetPointsOk returns a tuple with the Points field value
// and a boolean to check if the value has been set.
func (o *Series) GetPointsOk() (*[][]*float64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Points, true
}

// SetPoints sets field value.
func (o *Series) SetPoints(v [][]*float64) {
	o.Points = v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *Series) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Series) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *Series) HasTags() bool {
	if o != nil && o.Tags != nil {
		return true
	}

	return false
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *Series) SetTags(v []string) {
	o.Tags = v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *Series) GetType() string {
	if o == nil || o.Type == nil {
		var ret string
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Series) GetTypeOk() (*string, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *Series) HasType() bool {
	if o != nil && o.Type != nil {
		return true
	}

	return false
}

// SetType gets a reference to the given string and assigns it to the Type field.
func (o *Series) SetType(v string) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o Series) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Host != nil {
		toSerialize["host"] = o.Host
	}
	if o.Interval.IsSet() {
		toSerialize["interval"] = o.Interval.Get()
	}
	toSerialize["metric"] = o.Metric
	toSerialize["points"] = o.Points
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
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
func (o *Series) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Metric *string       `json:"metric"`
		Points *[][]*float64 `json:"points"`
	}{}
	all := struct {
		Host     *string       `json:"host,omitempty"`
		Interval NullableInt64 `json:"interval,omitempty"`
		Metric   string        `json:"metric"`
		Points   [][]*float64  `json:"points"`
		Tags     []string      `json:"tags,omitempty"`
		Type     *string       `json:"type,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Metric == nil {
		return fmt.Errorf("Required field metric missing")
	}
	if required.Points == nil {
		return fmt.Errorf("Required field points missing")
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
	o.Host = all.Host
	o.Interval = all.Interval
	o.Metric = all.Metric
	o.Points = all.Points
	o.Tags = all.Tags
	o.Type = all.Type
	return nil
}
