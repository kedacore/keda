// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsIndex Object describing a Datadog Log index.
type LogsIndex struct {
	// The number of log events you can send in this index per day before you are rate-limited.
	DailyLimit *int64 `json:"daily_limit,omitempty"`
	// An array of exclusion objects. The logs are tested against the query of each filter,
	// following the order of the array. Only the first matching active exclusion matters,
	// others (if any) are ignored.
	ExclusionFilters []LogsExclusion `json:"exclusion_filters,omitempty"`
	// Filter for logs.
	Filter LogsFilter `json:"filter"`
	// A boolean stating if the index is rate limited, meaning more logs than the daily limit have been sent.
	// Rate limit is reset every-day at 2pm UTC.
	IsRateLimited *bool `json:"is_rate_limited,omitempty"`
	// The name of the index.
	Name string `json:"name"`
	// The number of days before logs are deleted from this index. Available values depend on
	// retention plans specified in your organization's contract/subscriptions.
	NumRetentionDays *int64 `json:"num_retention_days,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsIndex instantiates a new LogsIndex object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsIndex(filter LogsFilter, name string) *LogsIndex {
	this := LogsIndex{}
	this.Filter = filter
	this.Name = name
	return &this
}

// NewLogsIndexWithDefaults instantiates a new LogsIndex object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsIndexWithDefaults() *LogsIndex {
	this := LogsIndex{}
	return &this
}

// GetDailyLimit returns the DailyLimit field value if set, zero value otherwise.
func (o *LogsIndex) GetDailyLimit() int64 {
	if o == nil || o.DailyLimit == nil {
		var ret int64
		return ret
	}
	return *o.DailyLimit
}

// GetDailyLimitOk returns a tuple with the DailyLimit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsIndex) GetDailyLimitOk() (*int64, bool) {
	if o == nil || o.DailyLimit == nil {
		return nil, false
	}
	return o.DailyLimit, true
}

// HasDailyLimit returns a boolean if a field has been set.
func (o *LogsIndex) HasDailyLimit() bool {
	if o != nil && o.DailyLimit != nil {
		return true
	}

	return false
}

// SetDailyLimit gets a reference to the given int64 and assigns it to the DailyLimit field.
func (o *LogsIndex) SetDailyLimit(v int64) {
	o.DailyLimit = &v
}

// GetExclusionFilters returns the ExclusionFilters field value if set, zero value otherwise.
func (o *LogsIndex) GetExclusionFilters() []LogsExclusion {
	if o == nil || o.ExclusionFilters == nil {
		var ret []LogsExclusion
		return ret
	}
	return o.ExclusionFilters
}

// GetExclusionFiltersOk returns a tuple with the ExclusionFilters field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsIndex) GetExclusionFiltersOk() (*[]LogsExclusion, bool) {
	if o == nil || o.ExclusionFilters == nil {
		return nil, false
	}
	return &o.ExclusionFilters, true
}

// HasExclusionFilters returns a boolean if a field has been set.
func (o *LogsIndex) HasExclusionFilters() bool {
	if o != nil && o.ExclusionFilters != nil {
		return true
	}

	return false
}

// SetExclusionFilters gets a reference to the given []LogsExclusion and assigns it to the ExclusionFilters field.
func (o *LogsIndex) SetExclusionFilters(v []LogsExclusion) {
	o.ExclusionFilters = v
}

// GetFilter returns the Filter field value.
func (o *LogsIndex) GetFilter() LogsFilter {
	if o == nil {
		var ret LogsFilter
		return ret
	}
	return o.Filter
}

// GetFilterOk returns a tuple with the Filter field value
// and a boolean to check if the value has been set.
func (o *LogsIndex) GetFilterOk() (*LogsFilter, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Filter, true
}

// SetFilter sets field value.
func (o *LogsIndex) SetFilter(v LogsFilter) {
	o.Filter = v
}

// GetIsRateLimited returns the IsRateLimited field value if set, zero value otherwise.
func (o *LogsIndex) GetIsRateLimited() bool {
	if o == nil || o.IsRateLimited == nil {
		var ret bool
		return ret
	}
	return *o.IsRateLimited
}

// GetIsRateLimitedOk returns a tuple with the IsRateLimited field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsIndex) GetIsRateLimitedOk() (*bool, bool) {
	if o == nil || o.IsRateLimited == nil {
		return nil, false
	}
	return o.IsRateLimited, true
}

// HasIsRateLimited returns a boolean if a field has been set.
func (o *LogsIndex) HasIsRateLimited() bool {
	if o != nil && o.IsRateLimited != nil {
		return true
	}

	return false
}

// SetIsRateLimited gets a reference to the given bool and assigns it to the IsRateLimited field.
func (o *LogsIndex) SetIsRateLimited(v bool) {
	o.IsRateLimited = &v
}

// GetName returns the Name field value.
func (o *LogsIndex) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *LogsIndex) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *LogsIndex) SetName(v string) {
	o.Name = v
}

// GetNumRetentionDays returns the NumRetentionDays field value if set, zero value otherwise.
func (o *LogsIndex) GetNumRetentionDays() int64 {
	if o == nil || o.NumRetentionDays == nil {
		var ret int64
		return ret
	}
	return *o.NumRetentionDays
}

// GetNumRetentionDaysOk returns a tuple with the NumRetentionDays field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsIndex) GetNumRetentionDaysOk() (*int64, bool) {
	if o == nil || o.NumRetentionDays == nil {
		return nil, false
	}
	return o.NumRetentionDays, true
}

// HasNumRetentionDays returns a boolean if a field has been set.
func (o *LogsIndex) HasNumRetentionDays() bool {
	if o != nil && o.NumRetentionDays != nil {
		return true
	}

	return false
}

// SetNumRetentionDays gets a reference to the given int64 and assigns it to the NumRetentionDays field.
func (o *LogsIndex) SetNumRetentionDays(v int64) {
	o.NumRetentionDays = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsIndex) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.DailyLimit != nil {
		toSerialize["daily_limit"] = o.DailyLimit
	}
	if o.ExclusionFilters != nil {
		toSerialize["exclusion_filters"] = o.ExclusionFilters
	}
	toSerialize["filter"] = o.Filter
	if o.IsRateLimited != nil {
		toSerialize["is_rate_limited"] = o.IsRateLimited
	}
	toSerialize["name"] = o.Name
	if o.NumRetentionDays != nil {
		toSerialize["num_retention_days"] = o.NumRetentionDays
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsIndex) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Filter *LogsFilter `json:"filter"`
		Name   *string     `json:"name"`
	}{}
	all := struct {
		DailyLimit       *int64          `json:"daily_limit,omitempty"`
		ExclusionFilters []LogsExclusion `json:"exclusion_filters,omitempty"`
		Filter           LogsFilter      `json:"filter"`
		IsRateLimited    *bool           `json:"is_rate_limited,omitempty"`
		Name             string          `json:"name"`
		NumRetentionDays *int64          `json:"num_retention_days,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Filter == nil {
		return fmt.Errorf("Required field filter missing")
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
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
	o.DailyLimit = all.DailyLimit
	o.ExclusionFilters = all.ExclusionFilters
	if all.Filter.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Filter = all.Filter
	o.IsRateLimited = all.IsRateLimited
	o.Name = all.Name
	o.NumRetentionDays = all.NumRetentionDays
	return nil
}
