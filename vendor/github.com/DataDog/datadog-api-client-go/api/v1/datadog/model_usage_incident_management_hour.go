// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// UsageIncidentManagementHour Incident management usage for a given organization for a given hour.
type UsageIncidentManagementHour struct {
	// The hour for the usage.
	Hour *time.Time `json:"hour,omitempty"`
	// Contains the total number monthly active users from the start of the given hour's month until the given hour.
	MonthlyActiveUsers *int64 `json:"monthly_active_users,omitempty"`
	// The organization name.
	OrgName *string `json:"org_name,omitempty"`
	// The organization public ID.
	PublicId *string `json:"public_id,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageIncidentManagementHour instantiates a new UsageIncidentManagementHour object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageIncidentManagementHour() *UsageIncidentManagementHour {
	this := UsageIncidentManagementHour{}
	return &this
}

// NewUsageIncidentManagementHourWithDefaults instantiates a new UsageIncidentManagementHour object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageIncidentManagementHourWithDefaults() *UsageIncidentManagementHour {
	this := UsageIncidentManagementHour{}
	return &this
}

// GetHour returns the Hour field value if set, zero value otherwise.
func (o *UsageIncidentManagementHour) GetHour() time.Time {
	if o == nil || o.Hour == nil {
		var ret time.Time
		return ret
	}
	return *o.Hour
}

// GetHourOk returns a tuple with the Hour field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageIncidentManagementHour) GetHourOk() (*time.Time, bool) {
	if o == nil || o.Hour == nil {
		return nil, false
	}
	return o.Hour, true
}

// HasHour returns a boolean if a field has been set.
func (o *UsageIncidentManagementHour) HasHour() bool {
	if o != nil && o.Hour != nil {
		return true
	}

	return false
}

// SetHour gets a reference to the given time.Time and assigns it to the Hour field.
func (o *UsageIncidentManagementHour) SetHour(v time.Time) {
	o.Hour = &v
}

// GetMonthlyActiveUsers returns the MonthlyActiveUsers field value if set, zero value otherwise.
func (o *UsageIncidentManagementHour) GetMonthlyActiveUsers() int64 {
	if o == nil || o.MonthlyActiveUsers == nil {
		var ret int64
		return ret
	}
	return *o.MonthlyActiveUsers
}

// GetMonthlyActiveUsersOk returns a tuple with the MonthlyActiveUsers field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageIncidentManagementHour) GetMonthlyActiveUsersOk() (*int64, bool) {
	if o == nil || o.MonthlyActiveUsers == nil {
		return nil, false
	}
	return o.MonthlyActiveUsers, true
}

// HasMonthlyActiveUsers returns a boolean if a field has been set.
func (o *UsageIncidentManagementHour) HasMonthlyActiveUsers() bool {
	if o != nil && o.MonthlyActiveUsers != nil {
		return true
	}

	return false
}

// SetMonthlyActiveUsers gets a reference to the given int64 and assigns it to the MonthlyActiveUsers field.
func (o *UsageIncidentManagementHour) SetMonthlyActiveUsers(v int64) {
	o.MonthlyActiveUsers = &v
}

// GetOrgName returns the OrgName field value if set, zero value otherwise.
func (o *UsageIncidentManagementHour) GetOrgName() string {
	if o == nil || o.OrgName == nil {
		var ret string
		return ret
	}
	return *o.OrgName
}

// GetOrgNameOk returns a tuple with the OrgName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageIncidentManagementHour) GetOrgNameOk() (*string, bool) {
	if o == nil || o.OrgName == nil {
		return nil, false
	}
	return o.OrgName, true
}

// HasOrgName returns a boolean if a field has been set.
func (o *UsageIncidentManagementHour) HasOrgName() bool {
	if o != nil && o.OrgName != nil {
		return true
	}

	return false
}

// SetOrgName gets a reference to the given string and assigns it to the OrgName field.
func (o *UsageIncidentManagementHour) SetOrgName(v string) {
	o.OrgName = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *UsageIncidentManagementHour) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageIncidentManagementHour) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *UsageIncidentManagementHour) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *UsageIncidentManagementHour) SetPublicId(v string) {
	o.PublicId = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageIncidentManagementHour) MarshalJSON() ([]byte, error) {
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
	if o.MonthlyActiveUsers != nil {
		toSerialize["monthly_active_users"] = o.MonthlyActiveUsers
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
func (o *UsageIncidentManagementHour) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Hour               *time.Time `json:"hour,omitempty"`
		MonthlyActiveUsers *int64     `json:"monthly_active_users,omitempty"`
		OrgName            *string    `json:"org_name,omitempty"`
		PublicId           *string    `json:"public_id,omitempty"`
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
	o.MonthlyActiveUsers = all.MonthlyActiveUsers
	o.OrgName = all.OrgName
	o.PublicId = all.PublicId
	return nil
}
