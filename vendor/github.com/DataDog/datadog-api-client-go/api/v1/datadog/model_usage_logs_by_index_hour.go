// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// UsageLogsByIndexHour Number of indexed logs for each hour and index for a given organization.
type UsageLogsByIndexHour struct {
	// The total number of indexed logs for the queried hour.
	EventCount *int64 `json:"event_count,omitempty"`
	// The hour for the usage.
	Hour *time.Time `json:"hour,omitempty"`
	// The index ID for this usage.
	IndexId *string `json:"index_id,omitempty"`
	// The user specified name for this index ID.
	IndexName *string `json:"index_name,omitempty"`
	// The organization name.
	OrgName *string `json:"org_name,omitempty"`
	// The organization public ID.
	PublicId *string `json:"public_id,omitempty"`
	// The retention period (in days) for this index ID.
	Retention *int64 `json:"retention,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageLogsByIndexHour instantiates a new UsageLogsByIndexHour object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageLogsByIndexHour() *UsageLogsByIndexHour {
	this := UsageLogsByIndexHour{}
	return &this
}

// NewUsageLogsByIndexHourWithDefaults instantiates a new UsageLogsByIndexHour object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageLogsByIndexHourWithDefaults() *UsageLogsByIndexHour {
	this := UsageLogsByIndexHour{}
	return &this
}

// GetEventCount returns the EventCount field value if set, zero value otherwise.
func (o *UsageLogsByIndexHour) GetEventCount() int64 {
	if o == nil || o.EventCount == nil {
		var ret int64
		return ret
	}
	return *o.EventCount
}

// GetEventCountOk returns a tuple with the EventCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsByIndexHour) GetEventCountOk() (*int64, bool) {
	if o == nil || o.EventCount == nil {
		return nil, false
	}
	return o.EventCount, true
}

// HasEventCount returns a boolean if a field has been set.
func (o *UsageLogsByIndexHour) HasEventCount() bool {
	if o != nil && o.EventCount != nil {
		return true
	}

	return false
}

// SetEventCount gets a reference to the given int64 and assigns it to the EventCount field.
func (o *UsageLogsByIndexHour) SetEventCount(v int64) {
	o.EventCount = &v
}

// GetHour returns the Hour field value if set, zero value otherwise.
func (o *UsageLogsByIndexHour) GetHour() time.Time {
	if o == nil || o.Hour == nil {
		var ret time.Time
		return ret
	}
	return *o.Hour
}

// GetHourOk returns a tuple with the Hour field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsByIndexHour) GetHourOk() (*time.Time, bool) {
	if o == nil || o.Hour == nil {
		return nil, false
	}
	return o.Hour, true
}

// HasHour returns a boolean if a field has been set.
func (o *UsageLogsByIndexHour) HasHour() bool {
	if o != nil && o.Hour != nil {
		return true
	}

	return false
}

// SetHour gets a reference to the given time.Time and assigns it to the Hour field.
func (o *UsageLogsByIndexHour) SetHour(v time.Time) {
	o.Hour = &v
}

// GetIndexId returns the IndexId field value if set, zero value otherwise.
func (o *UsageLogsByIndexHour) GetIndexId() string {
	if o == nil || o.IndexId == nil {
		var ret string
		return ret
	}
	return *o.IndexId
}

// GetIndexIdOk returns a tuple with the IndexId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsByIndexHour) GetIndexIdOk() (*string, bool) {
	if o == nil || o.IndexId == nil {
		return nil, false
	}
	return o.IndexId, true
}

// HasIndexId returns a boolean if a field has been set.
func (o *UsageLogsByIndexHour) HasIndexId() bool {
	if o != nil && o.IndexId != nil {
		return true
	}

	return false
}

// SetIndexId gets a reference to the given string and assigns it to the IndexId field.
func (o *UsageLogsByIndexHour) SetIndexId(v string) {
	o.IndexId = &v
}

// GetIndexName returns the IndexName field value if set, zero value otherwise.
func (o *UsageLogsByIndexHour) GetIndexName() string {
	if o == nil || o.IndexName == nil {
		var ret string
		return ret
	}
	return *o.IndexName
}

// GetIndexNameOk returns a tuple with the IndexName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsByIndexHour) GetIndexNameOk() (*string, bool) {
	if o == nil || o.IndexName == nil {
		return nil, false
	}
	return o.IndexName, true
}

// HasIndexName returns a boolean if a field has been set.
func (o *UsageLogsByIndexHour) HasIndexName() bool {
	if o != nil && o.IndexName != nil {
		return true
	}

	return false
}

// SetIndexName gets a reference to the given string and assigns it to the IndexName field.
func (o *UsageLogsByIndexHour) SetIndexName(v string) {
	o.IndexName = &v
}

// GetOrgName returns the OrgName field value if set, zero value otherwise.
func (o *UsageLogsByIndexHour) GetOrgName() string {
	if o == nil || o.OrgName == nil {
		var ret string
		return ret
	}
	return *o.OrgName
}

// GetOrgNameOk returns a tuple with the OrgName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsByIndexHour) GetOrgNameOk() (*string, bool) {
	if o == nil || o.OrgName == nil {
		return nil, false
	}
	return o.OrgName, true
}

// HasOrgName returns a boolean if a field has been set.
func (o *UsageLogsByIndexHour) HasOrgName() bool {
	if o != nil && o.OrgName != nil {
		return true
	}

	return false
}

// SetOrgName gets a reference to the given string and assigns it to the OrgName field.
func (o *UsageLogsByIndexHour) SetOrgName(v string) {
	o.OrgName = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *UsageLogsByIndexHour) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsByIndexHour) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *UsageLogsByIndexHour) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *UsageLogsByIndexHour) SetPublicId(v string) {
	o.PublicId = &v
}

// GetRetention returns the Retention field value if set, zero value otherwise.
func (o *UsageLogsByIndexHour) GetRetention() int64 {
	if o == nil || o.Retention == nil {
		var ret int64
		return ret
	}
	return *o.Retention
}

// GetRetentionOk returns a tuple with the Retention field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLogsByIndexHour) GetRetentionOk() (*int64, bool) {
	if o == nil || o.Retention == nil {
		return nil, false
	}
	return o.Retention, true
}

// HasRetention returns a boolean if a field has been set.
func (o *UsageLogsByIndexHour) HasRetention() bool {
	if o != nil && o.Retention != nil {
		return true
	}

	return false
}

// SetRetention gets a reference to the given int64 and assigns it to the Retention field.
func (o *UsageLogsByIndexHour) SetRetention(v int64) {
	o.Retention = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageLogsByIndexHour) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.EventCount != nil {
		toSerialize["event_count"] = o.EventCount
	}
	if o.Hour != nil {
		if o.Hour.Nanosecond() == 0 {
			toSerialize["hour"] = o.Hour.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["hour"] = o.Hour.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.IndexId != nil {
		toSerialize["index_id"] = o.IndexId
	}
	if o.IndexName != nil {
		toSerialize["index_name"] = o.IndexName
	}
	if o.OrgName != nil {
		toSerialize["org_name"] = o.OrgName
	}
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
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
func (o *UsageLogsByIndexHour) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		EventCount *int64     `json:"event_count,omitempty"`
		Hour       *time.Time `json:"hour,omitempty"`
		IndexId    *string    `json:"index_id,omitempty"`
		IndexName  *string    `json:"index_name,omitempty"`
		OrgName    *string    `json:"org_name,omitempty"`
		PublicId   *string    `json:"public_id,omitempty"`
		Retention  *int64     `json:"retention,omitempty"`
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
	o.EventCount = all.EventCount
	o.Hour = all.Hour
	o.IndexId = all.IndexId
	o.IndexName = all.IndexName
	o.OrgName = all.OrgName
	o.PublicId = all.PublicId
	o.Retention = all.Retention
	return nil
}
