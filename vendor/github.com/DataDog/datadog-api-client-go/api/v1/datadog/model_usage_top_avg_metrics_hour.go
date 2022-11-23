// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// UsageTopAvgMetricsHour Number of hourly recorded custom metrics for a given organization.
type UsageTopAvgMetricsHour struct {
	// Average number of timeseries per hour in which the metric occurs.
	AvgMetricHour *int64 `json:"avg_metric_hour,omitempty"`
	// Maximum number of timeseries per hour in which the metric occurs.
	MaxMetricHour *int64 `json:"max_metric_hour,omitempty"`
	// Contains the metric category.
	MetricCategory *UsageMetricCategory `json:"metric_category,omitempty"`
	// Contains the custom metric name.
	MetricName *string `json:"metric_name,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageTopAvgMetricsHour instantiates a new UsageTopAvgMetricsHour object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageTopAvgMetricsHour() *UsageTopAvgMetricsHour {
	this := UsageTopAvgMetricsHour{}
	return &this
}

// NewUsageTopAvgMetricsHourWithDefaults instantiates a new UsageTopAvgMetricsHour object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageTopAvgMetricsHourWithDefaults() *UsageTopAvgMetricsHour {
	this := UsageTopAvgMetricsHour{}
	return &this
}

// GetAvgMetricHour returns the AvgMetricHour field value if set, zero value otherwise.
func (o *UsageTopAvgMetricsHour) GetAvgMetricHour() int64 {
	if o == nil || o.AvgMetricHour == nil {
		var ret int64
		return ret
	}
	return *o.AvgMetricHour
}

// GetAvgMetricHourOk returns a tuple with the AvgMetricHour field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageTopAvgMetricsHour) GetAvgMetricHourOk() (*int64, bool) {
	if o == nil || o.AvgMetricHour == nil {
		return nil, false
	}
	return o.AvgMetricHour, true
}

// HasAvgMetricHour returns a boolean if a field has been set.
func (o *UsageTopAvgMetricsHour) HasAvgMetricHour() bool {
	if o != nil && o.AvgMetricHour != nil {
		return true
	}

	return false
}

// SetAvgMetricHour gets a reference to the given int64 and assigns it to the AvgMetricHour field.
func (o *UsageTopAvgMetricsHour) SetAvgMetricHour(v int64) {
	o.AvgMetricHour = &v
}

// GetMaxMetricHour returns the MaxMetricHour field value if set, zero value otherwise.
func (o *UsageTopAvgMetricsHour) GetMaxMetricHour() int64 {
	if o == nil || o.MaxMetricHour == nil {
		var ret int64
		return ret
	}
	return *o.MaxMetricHour
}

// GetMaxMetricHourOk returns a tuple with the MaxMetricHour field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageTopAvgMetricsHour) GetMaxMetricHourOk() (*int64, bool) {
	if o == nil || o.MaxMetricHour == nil {
		return nil, false
	}
	return o.MaxMetricHour, true
}

// HasMaxMetricHour returns a boolean if a field has been set.
func (o *UsageTopAvgMetricsHour) HasMaxMetricHour() bool {
	if o != nil && o.MaxMetricHour != nil {
		return true
	}

	return false
}

// SetMaxMetricHour gets a reference to the given int64 and assigns it to the MaxMetricHour field.
func (o *UsageTopAvgMetricsHour) SetMaxMetricHour(v int64) {
	o.MaxMetricHour = &v
}

// GetMetricCategory returns the MetricCategory field value if set, zero value otherwise.
func (o *UsageTopAvgMetricsHour) GetMetricCategory() UsageMetricCategory {
	if o == nil || o.MetricCategory == nil {
		var ret UsageMetricCategory
		return ret
	}
	return *o.MetricCategory
}

// GetMetricCategoryOk returns a tuple with the MetricCategory field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageTopAvgMetricsHour) GetMetricCategoryOk() (*UsageMetricCategory, bool) {
	if o == nil || o.MetricCategory == nil {
		return nil, false
	}
	return o.MetricCategory, true
}

// HasMetricCategory returns a boolean if a field has been set.
func (o *UsageTopAvgMetricsHour) HasMetricCategory() bool {
	if o != nil && o.MetricCategory != nil {
		return true
	}

	return false
}

// SetMetricCategory gets a reference to the given UsageMetricCategory and assigns it to the MetricCategory field.
func (o *UsageTopAvgMetricsHour) SetMetricCategory(v UsageMetricCategory) {
	o.MetricCategory = &v
}

// GetMetricName returns the MetricName field value if set, zero value otherwise.
func (o *UsageTopAvgMetricsHour) GetMetricName() string {
	if o == nil || o.MetricName == nil {
		var ret string
		return ret
	}
	return *o.MetricName
}

// GetMetricNameOk returns a tuple with the MetricName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageTopAvgMetricsHour) GetMetricNameOk() (*string, bool) {
	if o == nil || o.MetricName == nil {
		return nil, false
	}
	return o.MetricName, true
}

// HasMetricName returns a boolean if a field has been set.
func (o *UsageTopAvgMetricsHour) HasMetricName() bool {
	if o != nil && o.MetricName != nil {
		return true
	}

	return false
}

// SetMetricName gets a reference to the given string and assigns it to the MetricName field.
func (o *UsageTopAvgMetricsHour) SetMetricName(v string) {
	o.MetricName = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageTopAvgMetricsHour) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AvgMetricHour != nil {
		toSerialize["avg_metric_hour"] = o.AvgMetricHour
	}
	if o.MaxMetricHour != nil {
		toSerialize["max_metric_hour"] = o.MaxMetricHour
	}
	if o.MetricCategory != nil {
		toSerialize["metric_category"] = o.MetricCategory
	}
	if o.MetricName != nil {
		toSerialize["metric_name"] = o.MetricName
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageTopAvgMetricsHour) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AvgMetricHour  *int64               `json:"avg_metric_hour,omitempty"`
		MaxMetricHour  *int64               `json:"max_metric_hour,omitempty"`
		MetricCategory *UsageMetricCategory `json:"metric_category,omitempty"`
		MetricName     *string              `json:"metric_name,omitempty"`
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
	if v := all.MetricCategory; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.AvgMetricHour = all.AvgMetricHour
	o.MaxMetricHour = all.MaxMetricHour
	o.MetricCategory = all.MetricCategory
	o.MetricName = all.MetricName
	return nil
}
