// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// LogsRetentionAggSumUsage Object containing indexed logs usage aggregated across organizations and months for a retention period.
type LogsRetentionAggSumUsage struct {
	// Total indexed logs for this retention period.
	LogsIndexedLogsUsageAggSum *int64 `json:"logs_indexed_logs_usage_agg_sum,omitempty"`
	// Live indexed logs for this retention period.
	LogsLiveIndexedLogsUsageAggSum *int64 `json:"logs_live_indexed_logs_usage_agg_sum,omitempty"`
	// Rehydrated indexed logs for this retention period.
	LogsRehydratedIndexedLogsUsageAggSum *int64 `json:"logs_rehydrated_indexed_logs_usage_agg_sum,omitempty"`
	// The retention period in days or "custom" for all custom retention periods.
	Retention *string `json:"retention,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsRetentionAggSumUsage instantiates a new LogsRetentionAggSumUsage object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsRetentionAggSumUsage() *LogsRetentionAggSumUsage {
	this := LogsRetentionAggSumUsage{}
	return &this
}

// NewLogsRetentionAggSumUsageWithDefaults instantiates a new LogsRetentionAggSumUsage object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsRetentionAggSumUsageWithDefaults() *LogsRetentionAggSumUsage {
	this := LogsRetentionAggSumUsage{}
	return &this
}

// GetLogsIndexedLogsUsageAggSum returns the LogsIndexedLogsUsageAggSum field value if set, zero value otherwise.
func (o *LogsRetentionAggSumUsage) GetLogsIndexedLogsUsageAggSum() int64 {
	if o == nil || o.LogsIndexedLogsUsageAggSum == nil {
		var ret int64
		return ret
	}
	return *o.LogsIndexedLogsUsageAggSum
}

// GetLogsIndexedLogsUsageAggSumOk returns a tuple with the LogsIndexedLogsUsageAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsRetentionAggSumUsage) GetLogsIndexedLogsUsageAggSumOk() (*int64, bool) {
	if o == nil || o.LogsIndexedLogsUsageAggSum == nil {
		return nil, false
	}
	return o.LogsIndexedLogsUsageAggSum, true
}

// HasLogsIndexedLogsUsageAggSum returns a boolean if a field has been set.
func (o *LogsRetentionAggSumUsage) HasLogsIndexedLogsUsageAggSum() bool {
	if o != nil && o.LogsIndexedLogsUsageAggSum != nil {
		return true
	}

	return false
}

// SetLogsIndexedLogsUsageAggSum gets a reference to the given int64 and assigns it to the LogsIndexedLogsUsageAggSum field.
func (o *LogsRetentionAggSumUsage) SetLogsIndexedLogsUsageAggSum(v int64) {
	o.LogsIndexedLogsUsageAggSum = &v
}

// GetLogsLiveIndexedLogsUsageAggSum returns the LogsLiveIndexedLogsUsageAggSum field value if set, zero value otherwise.
func (o *LogsRetentionAggSumUsage) GetLogsLiveIndexedLogsUsageAggSum() int64 {
	if o == nil || o.LogsLiveIndexedLogsUsageAggSum == nil {
		var ret int64
		return ret
	}
	return *o.LogsLiveIndexedLogsUsageAggSum
}

// GetLogsLiveIndexedLogsUsageAggSumOk returns a tuple with the LogsLiveIndexedLogsUsageAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsRetentionAggSumUsage) GetLogsLiveIndexedLogsUsageAggSumOk() (*int64, bool) {
	if o == nil || o.LogsLiveIndexedLogsUsageAggSum == nil {
		return nil, false
	}
	return o.LogsLiveIndexedLogsUsageAggSum, true
}

// HasLogsLiveIndexedLogsUsageAggSum returns a boolean if a field has been set.
func (o *LogsRetentionAggSumUsage) HasLogsLiveIndexedLogsUsageAggSum() bool {
	if o != nil && o.LogsLiveIndexedLogsUsageAggSum != nil {
		return true
	}

	return false
}

// SetLogsLiveIndexedLogsUsageAggSum gets a reference to the given int64 and assigns it to the LogsLiveIndexedLogsUsageAggSum field.
func (o *LogsRetentionAggSumUsage) SetLogsLiveIndexedLogsUsageAggSum(v int64) {
	o.LogsLiveIndexedLogsUsageAggSum = &v
}

// GetLogsRehydratedIndexedLogsUsageAggSum returns the LogsRehydratedIndexedLogsUsageAggSum field value if set, zero value otherwise.
func (o *LogsRetentionAggSumUsage) GetLogsRehydratedIndexedLogsUsageAggSum() int64 {
	if o == nil || o.LogsRehydratedIndexedLogsUsageAggSum == nil {
		var ret int64
		return ret
	}
	return *o.LogsRehydratedIndexedLogsUsageAggSum
}

// GetLogsRehydratedIndexedLogsUsageAggSumOk returns a tuple with the LogsRehydratedIndexedLogsUsageAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsRetentionAggSumUsage) GetLogsRehydratedIndexedLogsUsageAggSumOk() (*int64, bool) {
	if o == nil || o.LogsRehydratedIndexedLogsUsageAggSum == nil {
		return nil, false
	}
	return o.LogsRehydratedIndexedLogsUsageAggSum, true
}

// HasLogsRehydratedIndexedLogsUsageAggSum returns a boolean if a field has been set.
func (o *LogsRetentionAggSumUsage) HasLogsRehydratedIndexedLogsUsageAggSum() bool {
	if o != nil && o.LogsRehydratedIndexedLogsUsageAggSum != nil {
		return true
	}

	return false
}

// SetLogsRehydratedIndexedLogsUsageAggSum gets a reference to the given int64 and assigns it to the LogsRehydratedIndexedLogsUsageAggSum field.
func (o *LogsRetentionAggSumUsage) SetLogsRehydratedIndexedLogsUsageAggSum(v int64) {
	o.LogsRehydratedIndexedLogsUsageAggSum = &v
}

// GetRetention returns the Retention field value if set, zero value otherwise.
func (o *LogsRetentionAggSumUsage) GetRetention() string {
	if o == nil || o.Retention == nil {
		var ret string
		return ret
	}
	return *o.Retention
}

// GetRetentionOk returns a tuple with the Retention field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsRetentionAggSumUsage) GetRetentionOk() (*string, bool) {
	if o == nil || o.Retention == nil {
		return nil, false
	}
	return o.Retention, true
}

// HasRetention returns a boolean if a field has been set.
func (o *LogsRetentionAggSumUsage) HasRetention() bool {
	if o != nil && o.Retention != nil {
		return true
	}

	return false
}

// SetRetention gets a reference to the given string and assigns it to the Retention field.
func (o *LogsRetentionAggSumUsage) SetRetention(v string) {
	o.Retention = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsRetentionAggSumUsage) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.LogsIndexedLogsUsageAggSum != nil {
		toSerialize["logs_indexed_logs_usage_agg_sum"] = o.LogsIndexedLogsUsageAggSum
	}
	if o.LogsLiveIndexedLogsUsageAggSum != nil {
		toSerialize["logs_live_indexed_logs_usage_agg_sum"] = o.LogsLiveIndexedLogsUsageAggSum
	}
	if o.LogsRehydratedIndexedLogsUsageAggSum != nil {
		toSerialize["logs_rehydrated_indexed_logs_usage_agg_sum"] = o.LogsRehydratedIndexedLogsUsageAggSum
	}
	if o.Retention != nil {
		toSerialize["retention"] = o.Retention
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsRetentionAggSumUsage) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		LogsIndexedLogsUsageAggSum           *int64  `json:"logs_indexed_logs_usage_agg_sum,omitempty"`
		LogsLiveIndexedLogsUsageAggSum       *int64  `json:"logs_live_indexed_logs_usage_agg_sum,omitempty"`
		LogsRehydratedIndexedLogsUsageAggSum *int64  `json:"logs_rehydrated_indexed_logs_usage_agg_sum,omitempty"`
		Retention                            *string `json:"retention,omitempty"`
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
	o.LogsIndexedLogsUsageAggSum = all.LogsIndexedLogsUsageAggSum
	o.LogsLiveIndexedLogsUsageAggSum = all.LogsLiveIndexedLogsUsageAggSum
	o.LogsRehydratedIndexedLogsUsageAggSum = all.LogsRehydratedIndexedLogsUsageAggSum
	o.Retention = all.Retention
	return nil
}
