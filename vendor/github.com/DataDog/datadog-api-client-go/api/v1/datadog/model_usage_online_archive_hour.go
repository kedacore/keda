// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// UsageOnlineArchiveHour Online Archive usage in a given hour.
type UsageOnlineArchiveHour struct {
	// The hour for the usage.
	Hour *time.Time `json:"hour,omitempty"`
	// Total count of online archived events within the hour.
	OnlineArchiveEventsCount *int32 `json:"online_archive_events_count,omitempty"`
	// The organization name.
	OrgName *string `json:"org_name,omitempty"`
	// The organization public ID.
	PublicId *string `json:"public_id,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageOnlineArchiveHour instantiates a new UsageOnlineArchiveHour object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageOnlineArchiveHour() *UsageOnlineArchiveHour {
	this := UsageOnlineArchiveHour{}
	return &this
}

// NewUsageOnlineArchiveHourWithDefaults instantiates a new UsageOnlineArchiveHour object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageOnlineArchiveHourWithDefaults() *UsageOnlineArchiveHour {
	this := UsageOnlineArchiveHour{}
	return &this
}

// GetHour returns the Hour field value if set, zero value otherwise.
func (o *UsageOnlineArchiveHour) GetHour() time.Time {
	if o == nil || o.Hour == nil {
		var ret time.Time
		return ret
	}
	return *o.Hour
}

// GetHourOk returns a tuple with the Hour field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageOnlineArchiveHour) GetHourOk() (*time.Time, bool) {
	if o == nil || o.Hour == nil {
		return nil, false
	}
	return o.Hour, true
}

// HasHour returns a boolean if a field has been set.
func (o *UsageOnlineArchiveHour) HasHour() bool {
	if o != nil && o.Hour != nil {
		return true
	}

	return false
}

// SetHour gets a reference to the given time.Time and assigns it to the Hour field.
func (o *UsageOnlineArchiveHour) SetHour(v time.Time) {
	o.Hour = &v
}

// GetOnlineArchiveEventsCount returns the OnlineArchiveEventsCount field value if set, zero value otherwise.
func (o *UsageOnlineArchiveHour) GetOnlineArchiveEventsCount() int32 {
	if o == nil || o.OnlineArchiveEventsCount == nil {
		var ret int32
		return ret
	}
	return *o.OnlineArchiveEventsCount
}

// GetOnlineArchiveEventsCountOk returns a tuple with the OnlineArchiveEventsCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageOnlineArchiveHour) GetOnlineArchiveEventsCountOk() (*int32, bool) {
	if o == nil || o.OnlineArchiveEventsCount == nil {
		return nil, false
	}
	return o.OnlineArchiveEventsCount, true
}

// HasOnlineArchiveEventsCount returns a boolean if a field has been set.
func (o *UsageOnlineArchiveHour) HasOnlineArchiveEventsCount() bool {
	if o != nil && o.OnlineArchiveEventsCount != nil {
		return true
	}

	return false
}

// SetOnlineArchiveEventsCount gets a reference to the given int32 and assigns it to the OnlineArchiveEventsCount field.
func (o *UsageOnlineArchiveHour) SetOnlineArchiveEventsCount(v int32) {
	o.OnlineArchiveEventsCount = &v
}

// GetOrgName returns the OrgName field value if set, zero value otherwise.
func (o *UsageOnlineArchiveHour) GetOrgName() string {
	if o == nil || o.OrgName == nil {
		var ret string
		return ret
	}
	return *o.OrgName
}

// GetOrgNameOk returns a tuple with the OrgName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageOnlineArchiveHour) GetOrgNameOk() (*string, bool) {
	if o == nil || o.OrgName == nil {
		return nil, false
	}
	return o.OrgName, true
}

// HasOrgName returns a boolean if a field has been set.
func (o *UsageOnlineArchiveHour) HasOrgName() bool {
	if o != nil && o.OrgName != nil {
		return true
	}

	return false
}

// SetOrgName gets a reference to the given string and assigns it to the OrgName field.
func (o *UsageOnlineArchiveHour) SetOrgName(v string) {
	o.OrgName = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *UsageOnlineArchiveHour) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageOnlineArchiveHour) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *UsageOnlineArchiveHour) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *UsageOnlineArchiveHour) SetPublicId(v string) {
	o.PublicId = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageOnlineArchiveHour) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Hour != nil {
		if o.Hour.Nanosecond() == 0 {
			toSerialize["hour"] = o.Hour.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["hour"] = o.Hour.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.OnlineArchiveEventsCount != nil {
		toSerialize["online_archive_events_count"] = o.OnlineArchiveEventsCount
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
func (o *UsageOnlineArchiveHour) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Hour                     *time.Time `json:"hour,omitempty"`
		OnlineArchiveEventsCount *int32     `json:"online_archive_events_count,omitempty"`
		OrgName                  *string    `json:"org_name,omitempty"`
		PublicId                 *string    `json:"public_id,omitempty"`
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
	o.Hour = all.Hour
	o.OnlineArchiveEventsCount = all.OnlineArchiveEventsCount
	o.OrgName = all.OrgName
	o.PublicId = all.PublicId
	return nil
}
