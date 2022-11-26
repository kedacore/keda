// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// LogsRetentionSumUsage Object containing indexed logs usage grouped by retention period and summed.
type LogsRetentionSumUsage struct {
	// Total indexed logs for this retention period.
	LogsIndexedLogsUsageSum *int64 `json:"logs_indexed_logs_usage_sum,omitempty"`
	// Live indexed logs for this retention period.
	LogsLiveIndexedLogsUsageSum *int64 `json:"logs_live_indexed_logs_usage_sum,omitempty"`
	// Rehydrated indexed logs for this retention period.
	LogsRehydratedIndexedLogsUsageSum *int64 `json:"logs_rehydrated_indexed_logs_usage_sum,omitempty"`
	// The retention period in days or "custom" for all custom retention periods.
	Retention *string `json:"retention,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsRetentionSumUsage instantiates a new LogsRetentionSumUsage object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsRetentionSumUsage() *LogsRetentionSumUsage {
	this := LogsRetentionSumUsage{}
	return &this
}

// NewLogsRetentionSumUsageWithDefaults instantiates a new LogsRetentionSumUsage object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsRetentionSumUsageWithDefaults() *LogsRetentionSumUsage {
	this := LogsRetentionSumUsage{}
	return &this
}

// GetLogsIndexedLogsUsageSum returns the LogsIndexedLogsUsageSum field value if set, zero value otherwise.
func (o *LogsRetentionSumUsage) GetLogsIndexedLogsUsageSum() int64 {
	if o == nil || o.LogsIndexedLogsUsageSum == nil {
		var ret int64
		return ret
	}
	return *o.LogsIndexedLogsUsageSum
}

// GetLogsIndexedLogsUsageSumOk returns a tuple with the LogsIndexedLogsUsageSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsRetentionSumUsage) GetLogsIndexedLogsUsageSumOk() (*int64, bool) {
	if o == nil || o.LogsIndexedLogsUsageSum == nil {
		return nil, false
	}
	return o.LogsIndexedLogsUsageSum, true
}

// HasLogsIndexedLogsUsageSum returns a boolean if a field has been set.
func (o *LogsRetentionSumUsage) HasLogsIndexedLogsUsageSum() bool {
	if o != nil && o.LogsIndexedLogsUsageSum != nil {
		return true
	}

	return false
}

// SetLogsIndexedLogsUsageSum gets a reference to the given int64 and assigns it to the LogsIndexedLogsUsageSum field.
func (o *LogsRetentionSumUsage) SetLogsIndexedLogsUsageSum(v int64) {
	o.LogsIndexedLogsUsageSum = &v
}

// GetLogsLiveIndexedLogsUsageSum returns the LogsLiveIndexedLogsUsageSum field value if set, zero value otherwise.
func (o *LogsRetentionSumUsage) GetLogsLiveIndexedLogsUsageSum() int64 {
	if o == nil || o.LogsLiveIndexedLogsUsageSum == nil {
		var ret int64
		return ret
	}
	return *o.LogsLiveIndexedLogsUsageSum
}

// GetLogsLiveIndexedLogsUsageSumOk returns a tuple with the LogsLiveIndexedLogsUsageSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsRetentionSumUsage) GetLogsLiveIndexedLogsUsageSumOk() (*int64, bool) {
	if o == nil || o.LogsLiveIndexedLogsUsageSum == nil {
		return nil, false
	}
	return o.LogsLiveIndexedLogsUsageSum, true
}

// HasLogsLiveIndexedLogsUsageSum returns a boolean if a field has been set.
func (o *LogsRetentionSumUsage) HasLogsLiveIndexedLogsUsageSum() bool {
	if o != nil && o.LogsLiveIndexedLogsUsageSum != nil {
		return true
	}

	return false
}

// SetLogsLiveIndexedLogsUsageSum gets a reference to the given int64 and assigns it to the LogsLiveIndexedLogsUsageSum field.
func (o *LogsRetentionSumUsage) SetLogsLiveIndexedLogsUsageSum(v int64) {
	o.LogsLiveIndexedLogsUsageSum = &v
}

// GetLogsRehydratedIndexedLogsUsageSum returns the LogsRehydratedIndexedLogsUsageSum field value if set, zero value otherwise.
func (o *LogsRetentionSumUsage) GetLogsRehydratedIndexedLogsUsageSum() int64 {
	if o == nil || o.LogsRehydratedIndexedLogsUsageSum == nil {
		var ret int64
		return ret
	}
	return *o.LogsRehydratedIndexedLogsUsageSum
}

// GetLogsRehydratedIndexedLogsUsageSumOk returns a tuple with the LogsRehydratedIndexedLogsUsageSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsRetentionSumUsage) GetLogsRehydratedIndexedLogsUsageSumOk() (*int64, bool) {
	if o == nil || o.LogsRehydratedIndexedLogsUsageSum == nil {
		return nil, false
	}
	return o.LogsRehydratedIndexedLogsUsageSum, true
}

// HasLogsRehydratedIndexedLogsUsageSum returns a boolean if a field has been set.
func (o *LogsRetentionSumUsage) HasLogsRehydratedIndexedLogsUsageSum() bool {
	if o != nil && o.LogsRehydratedIndexedLogsUsageSum != nil {
		return true
	}

	return false
}

// SetLogsRehydratedIndexedLogsUsageSum gets a reference to the given int64 and assigns it to the LogsRehydratedIndexedLogsUsageSum field.
func (o *LogsRetentionSumUsage) SetLogsRehydratedIndexedLogsUsageSum(v int64) {
	o.LogsRehydratedIndexedLogsUsageSum = &v
}

// GetRetention returns the Retention field value if set, zero value otherwise.
func (o *LogsRetentionSumUsage) GetRetention() string {
	if o == nil || o.Retention == nil {
		var ret string
		return ret
	}
	return *o.Retention
}

// GetRetentionOk returns a tuple with the Retention field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsRetentionSumUsage) GetRetentionOk() (*string, bool) {
	if o == nil || o.Retention == nil {
		return nil, false
	}
	return o.Retention, true
}

// HasRetention returns a boolean if a field has been set.
func (o *LogsRetentionSumUsage) HasRetention() bool {
	if o != nil && o.Retention != nil {
		return true
	}

	return false
}

// SetRetention gets a reference to the given string and assigns it to the Retention field.
func (o *LogsRetentionSumUsage) SetRetention(v string) {
	o.Retention = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsRetentionSumUsage) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.LogsIndexedLogsUsageSum != nil {
		toSerialize["logs_indexed_logs_usage_sum"] = o.LogsIndexedLogsUsageSum
	}
	if o.LogsLiveIndexedLogsUsageSum != nil {
		toSerialize["logs_live_indexed_logs_usage_sum"] = o.LogsLiveIndexedLogsUsageSum
	}
	if o.LogsRehydratedIndexedLogsUsageSum != nil {
		toSerialize["logs_rehydrated_indexed_logs_usage_sum"] = o.LogsRehydratedIndexedLogsUsageSum
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
func (o *LogsRetentionSumUsage) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		LogsIndexedLogsUsageSum           *int64  `json:"logs_indexed_logs_usage_sum,omitempty"`
		LogsLiveIndexedLogsUsageSum       *int64  `json:"logs_live_indexed_logs_usage_sum,omitempty"`
		LogsRehydratedIndexedLogsUsageSum *int64  `json:"logs_rehydrated_indexed_logs_usage_sum,omitempty"`
		Retention                         *string `json:"retention,omitempty"`
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
	o.LogsIndexedLogsUsageSum = all.LogsIndexedLogsUsageSum
	o.LogsLiveIndexedLogsUsageSum = all.LogsLiveIndexedLogsUsageSum
	o.LogsRehydratedIndexedLogsUsageSum = all.LogsRehydratedIndexedLogsUsageSum
	o.Retention = all.Retention
	return nil
}
