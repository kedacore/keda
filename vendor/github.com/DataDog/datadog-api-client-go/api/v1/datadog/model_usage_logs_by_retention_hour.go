// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// UsageLogsByRetentionHour The number of indexed logs for each hour for a given organization broken down by retention period.
type UsageLogsByRetentionHour struct {
	// Total logs indexed with this retention period during a given hour.
	IndexedEventsCount *int64 `json:"indexed_events_count,omitempty"`
	// Live logs indexed with this retention period during a given hour.
	LiveIndexedEventsCount *int64 `json:"live_indexed_events_count,omitempty"`
	// The organization name.
	OrgName *string `json:"org_name,omitempty"`
	// The organization public ID.
	PublicId *string `json:"public_id,omitempty"`
	// Rehydrated logs indexed with this retention period during a given hour.
	RehydratedIndexedEventsCount *int64 `json:"rehydrated_indexed_events_count,omitempty"`
	// The retention period in days or "custom" for all custom retention usage.
	Retention *string `json:"retention,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageLogsByRetentionHour instantiates a new UsageLogsByRetentionHour object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageLogsByRetentionHour() *UsageLogsByRetentionHour {
	this := UsageLogsByRetentionHour{}
	return &this
}

// NewUsageLogsByRetentionHourWithDefaults instantiates a new UsageLogsByRetentionHour object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageLogsByRetentionHourWithDefaults() *UsageLogsByRetentionHour {
	this := UsageLogsByRetentionHour{}
	return &this
}

// GetIndexedEventsCount returns the IndexedEventsCount field value if set, zero value otherwise.
func (o *UsageLogsByRetentionHour) GetIndexedEventsCount() int64 {
	if o == nil || o.IndexedEventsCount == nil {
		var ret int64
		return ret
	}
	return *o.IndexedEventsCount
}

// GetIndexedEventsCountOk returns a tuple with the IndexedEventsCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsByRetentionHour) GetIndexedEventsCountOk() (*int64, bool) {
	if o == nil || o.IndexedEventsCount == nil {
		return nil, false
	}
	return o.IndexedEventsCount, true
}

// HasIndexedEventsCount returns a boolean if a field has been set.
func (o *UsageLogsByRetentionHour) HasIndexedEventsCount() bool {
	if o != nil && o.IndexedEventsCount != nil {
		return true
	}

	return false
}

// SetIndexedEventsCount gets a reference to the given int64 and assigns it to the IndexedEventsCount field.
func (o *UsageLogsByRetentionHour) SetIndexedEventsCount(v int64) {
	o.IndexedEventsCount = &v
}

// GetLiveIndexedEventsCount returns the LiveIndexedEventsCount field value if set, zero value otherwise.
func (o *UsageLogsByRetentionHour) GetLiveIndexedEventsCount() int64 {
	if o == nil || o.LiveIndexedEventsCount == nil {
		var ret int64
		return ret
	}
	return *o.LiveIndexedEventsCount
}

// GetLiveIndexedEventsCountOk returns a tuple with the LiveIndexedEventsCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsByRetentionHour) GetLiveIndexedEventsCountOk() (*int64, bool) {
	if o == nil || o.LiveIndexedEventsCount == nil {
		return nil, false
	}
	return o.LiveIndexedEventsCount, true
}

// HasLiveIndexedEventsCount returns a boolean if a field has been set.
func (o *UsageLogsByRetentionHour) HasLiveIndexedEventsCount() bool {
	if o != nil && o.LiveIndexedEventsCount != nil {
		return true
	}

	return false
}

// SetLiveIndexedEventsCount gets a reference to the given int64 and assigns it to the LiveIndexedEventsCount field.
func (o *UsageLogsByRetentionHour) SetLiveIndexedEventsCount(v int64) {
	o.LiveIndexedEventsCount = &v
}

// GetOrgName returns the OrgName field value if set, zero value otherwise.
func (o *UsageLogsByRetentionHour) GetOrgName() string {
	if o == nil || o.OrgName == nil {
		var ret string
		return ret
	}
	return *o.OrgName
}

// GetOrgNameOk returns a tuple with the OrgName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsByRetentionHour) GetOrgNameOk() (*string, bool) {
	if o == nil || o.OrgName == nil {
		return nil, false
	}
	return o.OrgName, true
}

// HasOrgName returns a boolean if a field has been set.
func (o *UsageLogsByRetentionHour) HasOrgName() bool {
	if o != nil && o.OrgName != nil {
		return true
	}

	return false
}

// SetOrgName gets a reference to the given string and assigns it to the OrgName field.
func (o *UsageLogsByRetentionHour) SetOrgName(v string) {
	o.OrgName = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *UsageLogsByRetentionHour) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsByRetentionHour) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *UsageLogsByRetentionHour) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *UsageLogsByRetentionHour) SetPublicId(v string) {
	o.PublicId = &v
}

// GetRehydratedIndexedEventsCount returns the RehydratedIndexedEventsCount field value if set, zero value otherwise.
func (o *UsageLogsByRetentionHour) GetRehydratedIndexedEventsCount() int64 {
	if o == nil || o.RehydratedIndexedEventsCount == nil {
		var ret int64
		return ret
	}
	return *o.RehydratedIndexedEventsCount
}

// GetRehydratedIndexedEventsCountOk returns a tuple with the RehydratedIndexedEventsCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsByRetentionHour) GetRehydratedIndexedEventsCountOk() (*int64, bool) {
	if o == nil || o.RehydratedIndexedEventsCount == nil {
		return nil, false
	}
	return o.RehydratedIndexedEventsCount, true
}

// HasRehydratedIndexedEventsCount returns a boolean if a field has been set.
func (o *UsageLogsByRetentionHour) HasRehydratedIndexedEventsCount() bool {
	if o != nil && o.RehydratedIndexedEventsCount != nil {
		return true
	}

	return false
}

// SetRehydratedIndexedEventsCount gets a reference to the given int64 and assigns it to the RehydratedIndexedEventsCount field.
func (o *UsageLogsByRetentionHour) SetRehydratedIndexedEventsCount(v int64) {
	o.RehydratedIndexedEventsCount = &v
}

// GetRetention returns the Retention field value if set, zero value otherwise.
func (o *UsageLogsByRetentionHour) GetRetention() string {
	if o == nil || o.Retention == nil {
		var ret string
		return ret
	}
	return *o.Retention
}

// GetRetentionOk returns a tuple with the Retention field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsByRetentionHour) GetRetentionOk() (*string, bool) {
	if o == nil || o.Retention == nil {
		return nil, false
	}
	return o.Retention, true
}

// HasRetention returns a boolean if a field has been set.
func (o *UsageLogsByRetentionHour) HasRetention() bool {
	if o != nil && o.Retention != nil {
		return true
	}

	return false
}

// SetRetention gets a reference to the given string and assigns it to the Retention field.
func (o *UsageLogsByRetentionHour) SetRetention(v string) {
	o.Retention = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageLogsByRetentionHour) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.IndexedEventsCount != nil {
		toSerialize["indexed_events_count"] = o.IndexedEventsCount
	}
	if o.LiveIndexedEventsCount != nil {
		toSerialize["live_indexed_events_count"] = o.LiveIndexedEventsCount
	}
	if o.OrgName != nil {
		toSerialize["org_name"] = o.OrgName
	}
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
	}
	if o.RehydratedIndexedEventsCount != nil {
		toSerialize["rehydrated_indexed_events_count"] = o.RehydratedIndexedEventsCount
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
func (o *UsageLogsByRetentionHour) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		IndexedEventsCount           *int64  `json:"indexed_events_count,omitempty"`
		LiveIndexedEventsCount       *int64  `json:"live_indexed_events_count,omitempty"`
		OrgName                      *string `json:"org_name,omitempty"`
		PublicId                     *string `json:"public_id,omitempty"`
		RehydratedIndexedEventsCount *int64  `json:"rehydrated_indexed_events_count,omitempty"`
		Retention                    *string `json:"retention,omitempty"`
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
	o.IndexedEventsCount = all.IndexedEventsCount
	o.LiveIndexedEventsCount = all.LiveIndexedEventsCount
	o.OrgName = all.OrgName
	o.PublicId = all.PublicId
	o.RehydratedIndexedEventsCount = all.RehydratedIndexedEventsCount
	o.Retention = all.Retention
	return nil
}
