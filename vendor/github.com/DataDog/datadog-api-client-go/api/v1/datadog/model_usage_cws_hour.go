// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// UsageCWSHour Cloud Workload Security usage for a given organization for a given hour.
type UsageCWSHour struct {
	// The total number of Cloud Workload Security container hours from the start of the given hour’s month until the given hour.
	CwsContainerCount *int64 `json:"cws_container_count,omitempty"`
	// The total number of Cloud Workload Security host hours from the start of the given hour’s month until the given hour.
	CwsHostCount *int64 `json:"cws_host_count,omitempty"`
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

// NewUsageCWSHour instantiates a new UsageCWSHour object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageCWSHour() *UsageCWSHour {
	this := UsageCWSHour{}
	return &this
}

// NewUsageCWSHourWithDefaults instantiates a new UsageCWSHour object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageCWSHourWithDefaults() *UsageCWSHour {
	this := UsageCWSHour{}
	return &this
}

// GetCwsContainerCount returns the CwsContainerCount field value if set, zero value otherwise.
func (o *UsageCWSHour) GetCwsContainerCount() int64 {
	if o == nil || o.CwsContainerCount == nil {
		var ret int64
		return ret
	}
	return *o.CwsContainerCount
}

// GetCwsContainerCountOk returns a tuple with the CwsContainerCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCWSHour) GetCwsContainerCountOk() (*int64, bool) {
	if o == nil || o.CwsContainerCount == nil {
		return nil, false
	}
	return o.CwsContainerCount, true
}

// HasCwsContainerCount returns a boolean if a field has been set.
func (o *UsageCWSHour) HasCwsContainerCount() bool {
	if o != nil && o.CwsContainerCount != nil {
		return true
	}

	return false
}

// SetCwsContainerCount gets a reference to the given int64 and assigns it to the CwsContainerCount field.
func (o *UsageCWSHour) SetCwsContainerCount(v int64) {
	o.CwsContainerCount = &v
}

// GetCwsHostCount returns the CwsHostCount field value if set, zero value otherwise.
func (o *UsageCWSHour) GetCwsHostCount() int64 {
	if o == nil || o.CwsHostCount == nil {
		var ret int64
		return ret
	}
	return *o.CwsHostCount
}

// GetCwsHostCountOk returns a tuple with the CwsHostCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCWSHour) GetCwsHostCountOk() (*int64, bool) {
	if o == nil || o.CwsHostCount == nil {
		return nil, false
	}
	return o.CwsHostCount, true
}

// HasCwsHostCount returns a boolean if a field has been set.
func (o *UsageCWSHour) HasCwsHostCount() bool {
	if o != nil && o.CwsHostCount != nil {
		return true
	}

	return false
}

// SetCwsHostCount gets a reference to the given int64 and assigns it to the CwsHostCount field.
func (o *UsageCWSHour) SetCwsHostCount(v int64) {
	o.CwsHostCount = &v
}

// GetHour returns the Hour field value if set, zero value otherwise.
func (o *UsageCWSHour) GetHour() time.Time {
	if o == nil || o.Hour == nil {
		var ret time.Time
		return ret
	}
	return *o.Hour
}

// GetHourOk returns a tuple with the Hour field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCWSHour) GetHourOk() (*time.Time, bool) {
	if o == nil || o.Hour == nil {
		return nil, false
	}
	return o.Hour, true
}

// HasHour returns a boolean if a field has been set.
func (o *UsageCWSHour) HasHour() bool {
	if o != nil && o.Hour != nil {
		return true
	}

	return false
}

// SetHour gets a reference to the given time.Time and assigns it to the Hour field.
func (o *UsageCWSHour) SetHour(v time.Time) {
	o.Hour = &v
}

// GetOrgName returns the OrgName field value if set, zero value otherwise.
func (o *UsageCWSHour) GetOrgName() string {
	if o == nil || o.OrgName == nil {
		var ret string
		return ret
	}
	return *o.OrgName
}

// GetOrgNameOk returns a tuple with the OrgName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCWSHour) GetOrgNameOk() (*string, bool) {
	if o == nil || o.OrgName == nil {
		return nil, false
	}
	return o.OrgName, true
}

// HasOrgName returns a boolean if a field has been set.
func (o *UsageCWSHour) HasOrgName() bool {
	if o != nil && o.OrgName != nil {
		return true
	}

	return false
}

// SetOrgName gets a reference to the given string and assigns it to the OrgName field.
func (o *UsageCWSHour) SetOrgName(v string) {
	o.OrgName = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *UsageCWSHour) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCWSHour) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *UsageCWSHour) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *UsageCWSHour) SetPublicId(v string) {
	o.PublicId = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageCWSHour) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CwsContainerCount != nil {
		toSerialize["cws_container_count"] = o.CwsContainerCount
	}
	if o.CwsHostCount != nil {
		toSerialize["cws_host_count"] = o.CwsHostCount
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
func (o *UsageCWSHour) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		CwsContainerCount *int64     `json:"cws_container_count,omitempty"`
		CwsHostCount      *int64     `json:"cws_host_count,omitempty"`
		Hour              *time.Time `json:"hour,omitempty"`
		OrgName           *string    `json:"org_name,omitempty"`
		PublicId          *string    `json:"public_id,omitempty"`
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
	o.CwsContainerCount = all.CwsContainerCount
	o.CwsHostCount = all.CwsHostCount
	o.Hour = all.Hour
	o.OrgName = all.OrgName
	o.PublicId = all.PublicId
	return nil
}
