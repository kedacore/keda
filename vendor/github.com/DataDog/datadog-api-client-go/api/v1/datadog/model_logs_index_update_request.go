// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsIndexUpdateRequest Object for updating a Datadog Log index.
type LogsIndexUpdateRequest struct {
	// The number of log events you can send in this index per day before you are rate-limited.
	DailyLimit *int64 `json:"daily_limit,omitempty"`
	// If true, sets the `daily_limit` value to null and the index is not limited on a daily basis (any
	// specified `daily_limit` value in the request is ignored). If false or omitted, the index's current
	// `daily_limit` is maintained.
	DisableDailyLimit *bool `json:"disable_daily_limit,omitempty"`
	// An array of exclusion objects. The logs are tested against the query of each filter,
	// following the order of the array. Only the first matching active exclusion matters,
	// others (if any) are ignored.
	ExclusionFilters []LogsExclusion `json:"exclusion_filters,omitempty"`
	// Filter for logs.
	Filter LogsFilter `json:"filter"`
	// The number of days before logs are deleted from this index. Available values depend on
	// retention plans specified in your organization's contract/subscriptions.
	//
	// **Note:** Changing the retention for an index adjusts the length of retention for all logs
	// already in this index. It may also affect billing.
	NumRetentionDays *int64 `json:"num_retention_days,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsIndexUpdateRequest instantiates a new LogsIndexUpdateRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsIndexUpdateRequest(filter LogsFilter) *LogsIndexUpdateRequest {
	this := LogsIndexUpdateRequest{}
	this.Filter = filter
	return &this
}

// NewLogsIndexUpdateRequestWithDefaults instantiates a new LogsIndexUpdateRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsIndexUpdateRequestWithDefaults() *LogsIndexUpdateRequest {
	this := LogsIndexUpdateRequest{}
	return &this
}

// GetDailyLimit returns the DailyLimit field value if set, zero value otherwise.
func (o *LogsIndexUpdateRequest) GetDailyLimit() int64 {
	if o == nil || o.DailyLimit == nil {
		var ret int64
		return ret
	}
	return *o.DailyLimit
}

// GetDailyLimitOk returns a tuple with the DailyLimit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsIndexUpdateRequest) GetDailyLimitOk() (*int64, bool) {
	if o == nil || o.DailyLimit == nil {
		return nil, false
	}
	return o.DailyLimit, true
}

// HasDailyLimit returns a boolean if a field has been set.
func (o *LogsIndexUpdateRequest) HasDailyLimit() bool {
	if o != nil && o.DailyLimit != nil {
		return true
	}

	return false
}

// SetDailyLimit gets a reference to the given int64 and assigns it to the DailyLimit field.
func (o *LogsIndexUpdateRequest) SetDailyLimit(v int64) {
	o.DailyLimit = &v
}

// GetDisableDailyLimit returns the DisableDailyLimit field value if set, zero value otherwise.
func (o *LogsIndexUpdateRequest) GetDisableDailyLimit() bool {
	if o == nil || o.DisableDailyLimit == nil {
		var ret bool
		return ret
	}
	return *o.DisableDailyLimit
}

// GetDisableDailyLimitOk returns a tuple with the DisableDailyLimit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsIndexUpdateRequest) GetDisableDailyLimitOk() (*bool, bool) {
	if o == nil || o.DisableDailyLimit == nil {
		return nil, false
	}
	return o.DisableDailyLimit, true
}

// HasDisableDailyLimit returns a boolean if a field has been set.
func (o *LogsIndexUpdateRequest) HasDisableDailyLimit() bool {
	if o != nil && o.DisableDailyLimit != nil {
		return true
	}

	return false
}

// SetDisableDailyLimit gets a reference to the given bool and assigns it to the DisableDailyLimit field.
func (o *LogsIndexUpdateRequest) SetDisableDailyLimit(v bool) {
	o.DisableDailyLimit = &v
}

// GetExclusionFilters returns the ExclusionFilters field value if set, zero value otherwise.
func (o *LogsIndexUpdateRequest) GetExclusionFilters() []LogsExclusion {
	if o == nil || o.ExclusionFilters == nil {
		var ret []LogsExclusion
		return ret
	}
	return o.ExclusionFilters
}

// GetExclusionFiltersOk returns a tuple with the ExclusionFilters field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsIndexUpdateRequest) GetExclusionFiltersOk() (*[]LogsExclusion, bool) {
	if o == nil || o.ExclusionFilters == nil {
		return nil, false
	}
	return &o.ExclusionFilters, true
}

// HasExclusionFilters returns a boolean if a field has been set.
func (o *LogsIndexUpdateRequest) HasExclusionFilters() bool {
	if o != nil && o.ExclusionFilters != nil {
		return true
	}

	return false
}

// SetExclusionFilters gets a reference to the given []LogsExclusion and assigns it to the ExclusionFilters field.
func (o *LogsIndexUpdateRequest) SetExclusionFilters(v []LogsExclusion) {
	o.ExclusionFilters = v
}

// GetFilter returns the Filter field value.
func (o *LogsIndexUpdateRequest) GetFilter() LogsFilter {
	if o == nil {
		var ret LogsFilter
		return ret
	}
	return o.Filter
}

// GetFilterOk returns a tuple with the Filter field value
// and a boolean to check if the value has been set.
func (o *LogsIndexUpdateRequest) GetFilterOk() (*LogsFilter, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Filter, true
}

// SetFilter sets field value.
func (o *LogsIndexUpdateRequest) SetFilter(v LogsFilter) {
	o.Filter = v
}

// GetNumRetentionDays returns the NumRetentionDays field value if set, zero value otherwise.
func (o *LogsIndexUpdateRequest) GetNumRetentionDays() int64 {
	if o == nil || o.NumRetentionDays == nil {
		var ret int64
		return ret
	}
	return *o.NumRetentionDays
}

// GetNumRetentionDaysOk returns a tuple with the NumRetentionDays field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsIndexUpdateRequest) GetNumRetentionDaysOk() (*int64, bool) {
	if o == nil || o.NumRetentionDays == nil {
		return nil, false
	}
	return o.NumRetentionDays, true
}

// HasNumRetentionDays returns a boolean if a field has been set.
func (o *LogsIndexUpdateRequest) HasNumRetentionDays() bool {
	if o != nil && o.NumRetentionDays != nil {
		return true
	}

	return false
}

// SetNumRetentionDays gets a reference to the given int64 and assigns it to the NumRetentionDays field.
func (o *LogsIndexUpdateRequest) SetNumRetentionDays(v int64) {
	o.NumRetentionDays = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsIndexUpdateRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.DailyLimit != nil {
		toSerialize["daily_limit"] = o.DailyLimit
	}
	if o.DisableDailyLimit != nil {
		toSerialize["disable_daily_limit"] = o.DisableDailyLimit
	}
	if o.ExclusionFilters != nil {
		toSerialize["exclusion_filters"] = o.ExclusionFilters
	}
	toSerialize["filter"] = o.Filter
	if o.NumRetentionDays != nil {
		toSerialize["num_retention_days"] = o.NumRetentionDays
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsIndexUpdateRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Filter *LogsFilter `json:"filter"`
	}{}
	all := struct {
		DailyLimit        *int64          `json:"daily_limit,omitempty"`
		DisableDailyLimit *bool           `json:"disable_daily_limit,omitempty"`
		ExclusionFilters  []LogsExclusion `json:"exclusion_filters,omitempty"`
		Filter            LogsFilter      `json:"filter"`
		NumRetentionDays  *int64          `json:"num_retention_days,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Filter == nil {
		return fmt.Errorf("Required field filter missing")
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
	o.DisableDailyLimit = all.DisableDailyLimit
	o.ExclusionFilters = all.ExclusionFilters
	if all.Filter.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Filter = all.Filter
	o.NumRetentionDays = all.NumRetentionDays
	return nil
}
