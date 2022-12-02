// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// UsageDBMHour Database Monitoring usage for a given organization for a given hour.
type UsageDBMHour struct {
	// The total number of Database Monitoring host hours from the start of the given hour’s month until the given hour.
	DbmHostCount *int64 `json:"dbm_host_count,omitempty"`
	// The total number of normalized Database Monitoring queries from the start of the given hour’s month until the given hour.
	DbmQueriesCount *int64 `json:"dbm_queries_count,omitempty"`
	// The hour for the usage.
	Hour *time.Time `json:"hour,omitempty"`
	// The organization name.
	OrgName *string `json:"org_name,omitempty"`
	// The organization public ID.
	PublicId *string `json:"public_id,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageDBMHour instantiates a new UsageDBMHour object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageDBMHour() *UsageDBMHour {
	this := UsageDBMHour{}
	return &this
}

// NewUsageDBMHourWithDefaults instantiates a new UsageDBMHour object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageDBMHourWithDefaults() *UsageDBMHour {
	this := UsageDBMHour{}
	return &this
}

// GetDbmHostCount returns the DbmHostCount field value if set, zero value otherwise.
func (o *UsageDBMHour) GetDbmHostCount() int64 {
	if o == nil || o.DbmHostCount == nil {
		var ret int64
		return ret
	}
	return *o.DbmHostCount
}

// GetDbmHostCountOk returns a tuple with the DbmHostCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageDBMHour) GetDbmHostCountOk() (*int64, bool) {
	if o == nil || o.DbmHostCount == nil {
		return nil, false
	}
	return o.DbmHostCount, true
}

// HasDbmHostCount returns a boolean if a field has been set.
func (o *UsageDBMHour) HasDbmHostCount() bool {
	if o != nil && o.DbmHostCount != nil {
		return true
	}

	return false
}

// SetDbmHostCount gets a reference to the given int64 and assigns it to the DbmHostCount field.
func (o *UsageDBMHour) SetDbmHostCount(v int64) {
	o.DbmHostCount = &v
}

// GetDbmQueriesCount returns the DbmQueriesCount field value if set, zero value otherwise.
func (o *UsageDBMHour) GetDbmQueriesCount() int64 {
	if o == nil || o.DbmQueriesCount == nil {
		var ret int64
		return ret
	}
	return *o.DbmQueriesCount
}

// GetDbmQueriesCountOk returns a tuple with the DbmQueriesCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageDBMHour) GetDbmQueriesCountOk() (*int64, bool) {
	if o == nil || o.DbmQueriesCount == nil {
		return nil, false
	}
	return o.DbmQueriesCount, true
}

// HasDbmQueriesCount returns a boolean if a field has been set.
func (o *UsageDBMHour) HasDbmQueriesCount() bool {
	if o != nil && o.DbmQueriesCount != nil {
		return true
	}

	return false
}

// SetDbmQueriesCount gets a reference to the given int64 and assigns it to the DbmQueriesCount field.
func (o *UsageDBMHour) SetDbmQueriesCount(v int64) {
	o.DbmQueriesCount = &v
}

// GetHour returns the Hour field value if set, zero value otherwise.
func (o *UsageDBMHour) GetHour() time.Time {
	if o == nil || o.Hour == nil {
		var ret time.Time
		return ret
	}
	return *o.Hour
}

// GetHourOk returns a tuple with the Hour field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageDBMHour) GetHourOk() (*time.Time, bool) {
	if o == nil || o.Hour == nil {
		return nil, false
	}
	return o.Hour, true
}

// HasHour returns a boolean if a field has been set.
func (o *UsageDBMHour) HasHour() bool {
	if o != nil && o.Hour != nil {
		return true
	}

	return false
}

// SetHour gets a reference to the given time.Time and assigns it to the Hour field.
func (o *UsageDBMHour) SetHour(v time.Time) {
	o.Hour = &v
}

// GetOrgName returns the OrgName field value if set, zero value otherwise.
func (o *UsageDBMHour) GetOrgName() string {
	if o == nil || o.OrgName == nil {
		var ret string
		return ret
	}
	return *o.OrgName
}

// GetOrgNameOk returns a tuple with the OrgName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageDBMHour) GetOrgNameOk() (*string, bool) {
	if o == nil || o.OrgName == nil {
		return nil, false
	}
	return o.OrgName, true
}

// HasOrgName returns a boolean if a field has been set.
func (o *UsageDBMHour) HasOrgName() bool {
	if o != nil && o.OrgName != nil {
		return true
	}

	return false
}

// SetOrgName gets a reference to the given string and assigns it to the OrgName field.
func (o *UsageDBMHour) SetOrgName(v string) {
	o.OrgName = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *UsageDBMHour) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageDBMHour) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *UsageDBMHour) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *UsageDBMHour) SetPublicId(v string) {
	o.PublicId = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageDBMHour) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.DbmHostCount != nil {
		toSerialize["dbm_host_count"] = o.DbmHostCount
	}
	if o.DbmQueriesCount != nil {
		toSerialize["dbm_queries_count"] = o.DbmQueriesCount
	}
	if o.Hour != nil {
		if o.Hour.Nanosecond() == 0 {
			toSerialize["hour"] = o.Hour.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["hour"] = o.Hour.Format("2006-01-02T15:04:05.000Z07:00")
		}
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
func (o *UsageDBMHour) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		DbmHostCount    *int64     `json:"dbm_host_count,omitempty"`
		DbmQueriesCount *int64     `json:"dbm_queries_count,omitempty"`
		Hour            *time.Time `json:"hour,omitempty"`
		OrgName         *string    `json:"org_name,omitempty"`
		PublicId        *string    `json:"public_id,omitempty"`
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
	o.DbmHostCount = all.DbmHostCount
	o.DbmQueriesCount = all.DbmQueriesCount
	o.Hour = all.Hour
	o.OrgName = all.OrgName
	o.PublicId = all.PublicId
	return nil
}
