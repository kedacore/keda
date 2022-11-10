// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// UsageTopAvgMetricsMetadata The object containing document metadata.
type UsageTopAvgMetricsMetadata struct {
	// The day value from the user request that contains the returned usage data. (If day was used the request)
	Day *time.Time `json:"day,omitempty"`
	// The month value from the user request that contains the returned usage data. (If month was used the request)
	Month *time.Time `json:"month,omitempty"`
	// The metadata for the current pagination.
	Pagination *UsageTopAvgMetricsPagination `json:"pagination,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageTopAvgMetricsMetadata instantiates a new UsageTopAvgMetricsMetadata object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageTopAvgMetricsMetadata() *UsageTopAvgMetricsMetadata {
	this := UsageTopAvgMetricsMetadata{}
	return &this
}

// NewUsageTopAvgMetricsMetadataWithDefaults instantiates a new UsageTopAvgMetricsMetadata object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageTopAvgMetricsMetadataWithDefaults() *UsageTopAvgMetricsMetadata {
	this := UsageTopAvgMetricsMetadata{}
	return &this
}

// GetDay returns the Day field value if set, zero value otherwise.
func (o *UsageTopAvgMetricsMetadata) GetDay() time.Time {
	if o == nil || o.Day == nil {
		var ret time.Time
		return ret
	}
	return *o.Day
}

// GetDayOk returns a tuple with the Day field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageTopAvgMetricsMetadata) GetDayOk() (*time.Time, bool) {
	if o == nil || o.Day == nil {
		return nil, false
	}
	return o.Day, true
}

// HasDay returns a boolean if a field has been set.
func (o *UsageTopAvgMetricsMetadata) HasDay() bool {
	if o != nil && o.Day != nil {
		return true
	}

	return false
}

// SetDay gets a reference to the given time.Time and assigns it to the Day field.
func (o *UsageTopAvgMetricsMetadata) SetDay(v time.Time) {
	o.Day = &v
}

// GetMonth returns the Month field value if set, zero value otherwise.
func (o *UsageTopAvgMetricsMetadata) GetMonth() time.Time {
	if o == nil || o.Month == nil {
		var ret time.Time
		return ret
	}
	return *o.Month
}

// GetMonthOk returns a tuple with the Month field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageTopAvgMetricsMetadata) GetMonthOk() (*time.Time, bool) {
	if o == nil || o.Month == nil {
		return nil, false
	}
	return o.Month, true
}

// HasMonth returns a boolean if a field has been set.
func (o *UsageTopAvgMetricsMetadata) HasMonth() bool {
	if o != nil && o.Month != nil {
		return true
	}

	return false
}

// SetMonth gets a reference to the given time.Time and assigns it to the Month field.
func (o *UsageTopAvgMetricsMetadata) SetMonth(v time.Time) {
	o.Month = &v
}

// GetPagination returns the Pagination field value if set, zero value otherwise.
func (o *UsageTopAvgMetricsMetadata) GetPagination() UsageTopAvgMetricsPagination {
	if o == nil || o.Pagination == nil {
		var ret UsageTopAvgMetricsPagination
		return ret
	}
	return *o.Pagination
}

// GetPaginationOk returns a tuple with the Pagination field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageTopAvgMetricsMetadata) GetPaginationOk() (*UsageTopAvgMetricsPagination, bool) {
	if o == nil || o.Pagination == nil {
		return nil, false
	}
	return o.Pagination, true
}

// HasPagination returns a boolean if a field has been set.
func (o *UsageTopAvgMetricsMetadata) HasPagination() bool {
	if o != nil && o.Pagination != nil {
		return true
	}

	return false
}

// SetPagination gets a reference to the given UsageTopAvgMetricsPagination and assigns it to the Pagination field.
func (o *UsageTopAvgMetricsMetadata) SetPagination(v UsageTopAvgMetricsPagination) {
	o.Pagination = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageTopAvgMetricsMetadata) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Day != nil {
		if o.Day.Nanosecond() == 0 {
			toSerialize["day"] = o.Day.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["day"] = o.Day.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.Month != nil {
		if o.Month.Nanosecond() == 0 {
			toSerialize["month"] = o.Month.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["month"] = o.Month.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.Pagination != nil {
		toSerialize["pagination"] = o.Pagination
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageTopAvgMetricsMetadata) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Day        *time.Time                    `json:"day,omitempty"`
		Month      *time.Time                    `json:"month,omitempty"`
		Pagination *UsageTopAvgMetricsPagination `json:"pagination,omitempty"`
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
	o.Day = all.Day
	o.Month = all.Month
	if all.Pagination != nil && all.Pagination.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Pagination = all.Pagination
	return nil
}
