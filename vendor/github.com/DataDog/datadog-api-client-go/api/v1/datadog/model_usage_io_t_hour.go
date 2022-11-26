// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// UsageIoTHour IoT usage for a given organization for a given hour.
type UsageIoTHour struct {
	// The hour for the usage.
	Hour *time.Time `json:"hour,omitempty"`
	// The total number of IoT devices during a given hour.
	IotDeviceCount *int64 `json:"iot_device_count,omitempty"`
	// The organization name.
	OrgName *string `json:"org_name,omitempty"`
	// The organization public ID.
	PublicId *string `json:"public_id,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageIoTHour instantiates a new UsageIoTHour object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageIoTHour() *UsageIoTHour {
	this := UsageIoTHour{}
	return &this
}

// NewUsageIoTHourWithDefaults instantiates a new UsageIoTHour object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageIoTHourWithDefaults() *UsageIoTHour {
	this := UsageIoTHour{}
	return &this
}

// GetHour returns the Hour field value if set, zero value otherwise.
func (o *UsageIoTHour) GetHour() time.Time {
	if o == nil || o.Hour == nil {
		var ret time.Time
		return ret
	}
	return *o.Hour
}

// GetHourOk returns a tuple with the Hour field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageIoTHour) GetHourOk() (*time.Time, bool) {
	if o == nil || o.Hour == nil {
		return nil, false
	}
	return o.Hour, true
}

// HasHour returns a boolean if a field has been set.
func (o *UsageIoTHour) HasHour() bool {
	if o != nil && o.Hour != nil {
		return true
	}

	return false
}

// SetHour gets a reference to the given time.Time and assigns it to the Hour field.
func (o *UsageIoTHour) SetHour(v time.Time) {
	o.Hour = &v
}

// GetIotDeviceCount returns the IotDeviceCount field value if set, zero value otherwise.
func (o *UsageIoTHour) GetIotDeviceCount() int64 {
	if o == nil || o.IotDeviceCount == nil {
		var ret int64
		return ret
	}
	return *o.IotDeviceCount
}

// GetIotDeviceCountOk returns a tuple with the IotDeviceCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageIoTHour) GetIotDeviceCountOk() (*int64, bool) {
	if o == nil || o.IotDeviceCount == nil {
		return nil, false
	}
	return o.IotDeviceCount, true
}

// HasIotDeviceCount returns a boolean if a field has been set.
func (o *UsageIoTHour) HasIotDeviceCount() bool {
	if o != nil && o.IotDeviceCount != nil {
		return true
	}

	return false
}

// SetIotDeviceCount gets a reference to the given int64 and assigns it to the IotDeviceCount field.
func (o *UsageIoTHour) SetIotDeviceCount(v int64) {
	o.IotDeviceCount = &v
}

// GetOrgName returns the OrgName field value if set, zero value otherwise.
func (o *UsageIoTHour) GetOrgName() string {
	if o == nil || o.OrgName == nil {
		var ret string
		return ret
	}
	return *o.OrgName
}

// GetOrgNameOk returns a tuple with the OrgName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageIoTHour) GetOrgNameOk() (*string, bool) {
	if o == nil || o.OrgName == nil {
		return nil, false
	}
	return o.OrgName, true
}

// HasOrgName returns a boolean if a field has been set.
func (o *UsageIoTHour) HasOrgName() bool {
	if o != nil && o.OrgName != nil {
		return true
	}

	return false
}

// SetOrgName gets a reference to the given string and assigns it to the OrgName field.
func (o *UsageIoTHour) SetOrgName(v string) {
	o.OrgName = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *UsageIoTHour) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageIoTHour) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *UsageIoTHour) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *UsageIoTHour) SetPublicId(v string) {
	o.PublicId = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageIoTHour) MarshalJSON() ([]byte, error) {
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
	if o.IotDeviceCount != nil {
		toSerialize["iot_device_count"] = o.IotDeviceCount
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
func (o *UsageIoTHour) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Hour           *time.Time `json:"hour,omitempty"`
		IotDeviceCount *int64     `json:"iot_device_count,omitempty"`
		OrgName        *string    `json:"org_name,omitempty"`
		PublicId       *string    `json:"public_id,omitempty"`
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
	o.IotDeviceCount = all.IotDeviceCount
	o.OrgName = all.OrgName
	o.PublicId = all.PublicId
	return nil
}
