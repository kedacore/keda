// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// UsageLogsHour Hour usage for logs.
type UsageLogsHour struct {
	// Contains the number of billable log bytes ingested.
	BillableIngestedBytes *int64 `json:"billable_ingested_bytes,omitempty"`
	// The hour for the usage.
	Hour *time.Time `json:"hour,omitempty"`
	// Contains the number of log events indexed.
	IndexedEventsCount *int64 `json:"indexed_events_count,omitempty"`
	// Contains the number of log bytes ingested.
	IngestedEventsBytes *int64 `json:"ingested_events_bytes,omitempty"`
	// Contains the number of live log events indexed (data available as of December 1, 2020).
	LogsLiveIndexedCount *int64 `json:"logs_live_indexed_count,omitempty"`
	// Contains the number of live log bytes ingested (data available as of December 1, 2020).
	LogsLiveIngestedBytes *int64 `json:"logs_live_ingested_bytes,omitempty"`
	// Contains the number of rehydrated log events indexed (data available as of December 1, 2020).
	LogsRehydratedIndexedCount *int64 `json:"logs_rehydrated_indexed_count,omitempty"`
	// Contains the number of rehydrated log bytes ingested (data available as of December 1, 2020).
	LogsRehydratedIngestedBytes *int64 `json:"logs_rehydrated_ingested_bytes,omitempty"`
	// The organization name.
	OrgName *string `json:"org_name,omitempty"`
	// The organization public ID.
	PublicId *string `json:"public_id,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageLogsHour instantiates a new UsageLogsHour object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageLogsHour() *UsageLogsHour {
	this := UsageLogsHour{}
	return &this
}

// NewUsageLogsHourWithDefaults instantiates a new UsageLogsHour object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageLogsHourWithDefaults() *UsageLogsHour {
	this := UsageLogsHour{}
	return &this
}

// GetBillableIngestedBytes returns the BillableIngestedBytes field value if set, zero value otherwise.
func (o *UsageLogsHour) GetBillableIngestedBytes() int64 {
	if o == nil || o.BillableIngestedBytes == nil {
		var ret int64
		return ret
	}
	return *o.BillableIngestedBytes
}

// GetBillableIngestedBytesOk returns a tuple with the BillableIngestedBytes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsHour) GetBillableIngestedBytesOk() (*int64, bool) {
	if o == nil || o.BillableIngestedBytes == nil {
		return nil, false
	}
	return o.BillableIngestedBytes, true
}

// HasBillableIngestedBytes returns a boolean if a field has been set.
func (o *UsageLogsHour) HasBillableIngestedBytes() bool {
	if o != nil && o.BillableIngestedBytes != nil {
		return true
	}

	return false
}

// SetBillableIngestedBytes gets a reference to the given int64 and assigns it to the BillableIngestedBytes field.
func (o *UsageLogsHour) SetBillableIngestedBytes(v int64) {
	o.BillableIngestedBytes = &v
}

// GetHour returns the Hour field value if set, zero value otherwise.
func (o *UsageLogsHour) GetHour() time.Time {
	if o == nil || o.Hour == nil {
		var ret time.Time
		return ret
	}
	return *o.Hour
}

// GetHourOk returns a tuple with the Hour field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsHour) GetHourOk() (*time.Time, bool) {
	if o == nil || o.Hour == nil {
		return nil, false
	}
	return o.Hour, true
}

// HasHour returns a boolean if a field has been set.
func (o *UsageLogsHour) HasHour() bool {
	if o != nil && o.Hour != nil {
		return true
	}

	return false
}

// SetHour gets a reference to the given time.Time and assigns it to the Hour field.
func (o *UsageLogsHour) SetHour(v time.Time) {
	o.Hour = &v
}

// GetIndexedEventsCount returns the IndexedEventsCount field value if set, zero value otherwise.
func (o *UsageLogsHour) GetIndexedEventsCount() int64 {
	if o == nil || o.IndexedEventsCount == nil {
		var ret int64
		return ret
	}
	return *o.IndexedEventsCount
}

// GetIndexedEventsCountOk returns a tuple with the IndexedEventsCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsHour) GetIndexedEventsCountOk() (*int64, bool) {
	if o == nil || o.IndexedEventsCount == nil {
		return nil, false
	}
	return o.IndexedEventsCount, true
}

// HasIndexedEventsCount returns a boolean if a field has been set.
func (o *UsageLogsHour) HasIndexedEventsCount() bool {
	if o != nil && o.IndexedEventsCount != nil {
		return true
	}

	return false
}

// SetIndexedEventsCount gets a reference to the given int64 and assigns it to the IndexedEventsCount field.
func (o *UsageLogsHour) SetIndexedEventsCount(v int64) {
	o.IndexedEventsCount = &v
}

// GetIngestedEventsBytes returns the IngestedEventsBytes field value if set, zero value otherwise.
func (o *UsageLogsHour) GetIngestedEventsBytes() int64 {
	if o == nil || o.IngestedEventsBytes == nil {
		var ret int64
		return ret
	}
	return *o.IngestedEventsBytes
}

// GetIngestedEventsBytesOk returns a tuple with the IngestedEventsBytes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsHour) GetIngestedEventsBytesOk() (*int64, bool) {
	if o == nil || o.IngestedEventsBytes == nil {
		return nil, false
	}
	return o.IngestedEventsBytes, true
}

// HasIngestedEventsBytes returns a boolean if a field has been set.
func (o *UsageLogsHour) HasIngestedEventsBytes() bool {
	if o != nil && o.IngestedEventsBytes != nil {
		return true
	}

	return false
}

// SetIngestedEventsBytes gets a reference to the given int64 and assigns it to the IngestedEventsBytes field.
func (o *UsageLogsHour) SetIngestedEventsBytes(v int64) {
	o.IngestedEventsBytes = &v
}

// GetLogsLiveIndexedCount returns the LogsLiveIndexedCount field value if set, zero value otherwise.
func (o *UsageLogsHour) GetLogsLiveIndexedCount() int64 {
	if o == nil || o.LogsLiveIndexedCount == nil {
		var ret int64
		return ret
	}
	return *o.LogsLiveIndexedCount
}

// GetLogsLiveIndexedCountOk returns a tuple with the LogsLiveIndexedCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsHour) GetLogsLiveIndexedCountOk() (*int64, bool) {
	if o == nil || o.LogsLiveIndexedCount == nil {
		return nil, false
	}
	return o.LogsLiveIndexedCount, true
}

// HasLogsLiveIndexedCount returns a boolean if a field has been set.
func (o *UsageLogsHour) HasLogsLiveIndexedCount() bool {
	if o != nil && o.LogsLiveIndexedCount != nil {
		return true
	}

	return false
}

// SetLogsLiveIndexedCount gets a reference to the given int64 and assigns it to the LogsLiveIndexedCount field.
func (o *UsageLogsHour) SetLogsLiveIndexedCount(v int64) {
	o.LogsLiveIndexedCount = &v
}

// GetLogsLiveIngestedBytes returns the LogsLiveIngestedBytes field value if set, zero value otherwise.
func (o *UsageLogsHour) GetLogsLiveIngestedBytes() int64 {
	if o == nil || o.LogsLiveIngestedBytes == nil {
		var ret int64
		return ret
	}
	return *o.LogsLiveIngestedBytes
}

// GetLogsLiveIngestedBytesOk returns a tuple with the LogsLiveIngestedBytes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsHour) GetLogsLiveIngestedBytesOk() (*int64, bool) {
	if o == nil || o.LogsLiveIngestedBytes == nil {
		return nil, false
	}
	return o.LogsLiveIngestedBytes, true
}

// HasLogsLiveIngestedBytes returns a boolean if a field has been set.
func (o *UsageLogsHour) HasLogsLiveIngestedBytes() bool {
	if o != nil && o.LogsLiveIngestedBytes != nil {
		return true
	}

	return false
}

// SetLogsLiveIngestedBytes gets a reference to the given int64 and assigns it to the LogsLiveIngestedBytes field.
func (o *UsageLogsHour) SetLogsLiveIngestedBytes(v int64) {
	o.LogsLiveIngestedBytes = &v
}

// GetLogsRehydratedIndexedCount returns the LogsRehydratedIndexedCount field value if set, zero value otherwise.
func (o *UsageLogsHour) GetLogsRehydratedIndexedCount() int64 {
	if o == nil || o.LogsRehydratedIndexedCount == nil {
		var ret int64
		return ret
	}
	return *o.LogsRehydratedIndexedCount
}

// GetLogsRehydratedIndexedCountOk returns a tuple with the LogsRehydratedIndexedCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsHour) GetLogsRehydratedIndexedCountOk() (*int64, bool) {
	if o == nil || o.LogsRehydratedIndexedCount == nil {
		return nil, false
	}
	return o.LogsRehydratedIndexedCount, true
}

// HasLogsRehydratedIndexedCount returns a boolean if a field has been set.
func (o *UsageLogsHour) HasLogsRehydratedIndexedCount() bool {
	if o != nil && o.LogsRehydratedIndexedCount != nil {
		return true
	}

	return false
}

// SetLogsRehydratedIndexedCount gets a reference to the given int64 and assigns it to the LogsRehydratedIndexedCount field.
func (o *UsageLogsHour) SetLogsRehydratedIndexedCount(v int64) {
	o.LogsRehydratedIndexedCount = &v
}

// GetLogsRehydratedIngestedBytes returns the LogsRehydratedIngestedBytes field value if set, zero value otherwise.
func (o *UsageLogsHour) GetLogsRehydratedIngestedBytes() int64 {
	if o == nil || o.LogsRehydratedIngestedBytes == nil {
		var ret int64
		return ret
	}
	return *o.LogsRehydratedIngestedBytes
}

// GetLogsRehydratedIngestedBytesOk returns a tuple with the LogsRehydratedIngestedBytes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsHour) GetLogsRehydratedIngestedBytesOk() (*int64, bool) {
	if o == nil || o.LogsRehydratedIngestedBytes == nil {
		return nil, false
	}
	return o.LogsRehydratedIngestedBytes, true
}

// HasLogsRehydratedIngestedBytes returns a boolean if a field has been set.
func (o *UsageLogsHour) HasLogsRehydratedIngestedBytes() bool {
	if o != nil && o.LogsRehydratedIngestedBytes != nil {
		return true
	}

	return false
}

// SetLogsRehydratedIngestedBytes gets a reference to the given int64 and assigns it to the LogsRehydratedIngestedBytes field.
func (o *UsageLogsHour) SetLogsRehydratedIngestedBytes(v int64) {
	o.LogsRehydratedIngestedBytes = &v
}

// GetOrgName returns the OrgName field value if set, zero value otherwise.
func (o *UsageLogsHour) GetOrgName() string {
	if o == nil || o.OrgName == nil {
		var ret string
		return ret
	}
	return *o.OrgName
}

// GetOrgNameOk returns a tuple with the OrgName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsHour) GetOrgNameOk() (*string, bool) {
	if o == nil || o.OrgName == nil {
		return nil, false
	}
	return o.OrgName, true
}

// HasOrgName returns a boolean if a field has been set.
func (o *UsageLogsHour) HasOrgName() bool {
	if o != nil && o.OrgName != nil {
		return true
	}

	return false
}

// SetOrgName gets a reference to the given string and assigns it to the OrgName field.
func (o *UsageLogsHour) SetOrgName(v string) {
	o.OrgName = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *UsageLogsHour) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsHour) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *UsageLogsHour) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *UsageLogsHour) SetPublicId(v string) {
	o.PublicId = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageLogsHour) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.BillableIngestedBytes != nil {
		toSerialize["billable_ingested_bytes"] = o.BillableIngestedBytes
	}
	if o.Hour != nil {
		if o.Hour.Nanosecond() == 0 {
			toSerialize["hour"] = o.Hour.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["hour"] = o.Hour.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.IndexedEventsCount != nil {
		toSerialize["indexed_events_count"] = o.IndexedEventsCount
	}
	if o.IngestedEventsBytes != nil {
		toSerialize["ingested_events_bytes"] = o.IngestedEventsBytes
	}
	if o.LogsLiveIndexedCount != nil {
		toSerialize["logs_live_indexed_count"] = o.LogsLiveIndexedCount
	}
	if o.LogsLiveIngestedBytes != nil {
		toSerialize["logs_live_ingested_bytes"] = o.LogsLiveIngestedBytes
	}
	if o.LogsRehydratedIndexedCount != nil {
		toSerialize["logs_rehydrated_indexed_count"] = o.LogsRehydratedIndexedCount
	}
	if o.LogsRehydratedIngestedBytes != nil {
		toSerialize["logs_rehydrated_ingested_bytes"] = o.LogsRehydratedIngestedBytes
	}
	if o.OrgName != nil {
		toSerialize["org_name"] = o.OrgName
	}
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageLogsHour) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		BillableIngestedBytes       *int64     `json:"billable_ingested_bytes,omitempty"`
		Hour                        *time.Time `json:"hour,omitempty"`
		IndexedEventsCount          *int64     `json:"indexed_events_count,omitempty"`
		IngestedEventsBytes         *int64     `json:"ingested_events_bytes,omitempty"`
		LogsLiveIndexedCount        *int64     `json:"logs_live_indexed_count,omitempty"`
		LogsLiveIngestedBytes       *int64     `json:"logs_live_ingested_bytes,omitempty"`
		LogsRehydratedIndexedCount  *int64     `json:"logs_rehydrated_indexed_count,omitempty"`
		LogsRehydratedIngestedBytes *int64     `json:"logs_rehydrated_ingested_bytes,omitempty"`
		OrgName                     *string    `json:"org_name,omitempty"`
		PublicId                    *string    `json:"public_id,omitempty"`
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
	o.BillableIngestedBytes = all.BillableIngestedBytes
	o.Hour = all.Hour
	o.IndexedEventsCount = all.IndexedEventsCount
	o.IngestedEventsBytes = all.IngestedEventsBytes
	o.LogsLiveIndexedCount = all.LogsLiveIndexedCount
	o.LogsLiveIngestedBytes = all.LogsLiveIngestedBytes
	o.LogsRehydratedIndexedCount = all.LogsRehydratedIndexedCount
	o.LogsRehydratedIngestedBytes = all.LogsRehydratedIngestedBytes
	o.OrgName = all.OrgName
	o.PublicId = all.PublicId
	return nil
}
