// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// UsageRumSessionsHour Number of RUM Sessions recorded for each hour for a given organization.
type UsageRumSessionsHour struct {
	// The hour for the usage.
	Hour *time.Time `json:"hour,omitempty"`
	// The organization name.
	OrgName *string `json:"org_name,omitempty"`
	// The organization public ID.
	PublicId *string `json:"public_id,omitempty"`
	// Contains the number of RUM Replay Sessions (data available beginning November 1, 2021).
	ReplaySessionCount *int64 `json:"replay_session_count,omitempty"`
	// Contains the number of browser RUM Lite Sessions.
	SessionCount NullableInt64 `json:"session_count,omitempty"`
	// Contains the number of mobile RUM Sessions on Android (data available beginning December 1, 2020).
	SessionCountAndroid NullableInt64 `json:"session_count_android,omitempty"`
	// Contains the number of mobile RUM Sessions on iOS (data available beginning December 1, 2020).
	SessionCountIos NullableInt64 `json:"session_count_ios,omitempty"`
	// Contains the number of mobile RUM Sessions on React Native (data available beginning May 1, 2022).
	SessionCountReactnative NullableInt64 `json:"session_count_reactnative,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageRumSessionsHour instantiates a new UsageRumSessionsHour object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageRumSessionsHour() *UsageRumSessionsHour {
	this := UsageRumSessionsHour{}
	return &this
}

// NewUsageRumSessionsHourWithDefaults instantiates a new UsageRumSessionsHour object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageRumSessionsHourWithDefaults() *UsageRumSessionsHour {
	this := UsageRumSessionsHour{}
	return &this
}

// GetHour returns the Hour field value if set, zero value otherwise.
func (o *UsageRumSessionsHour) GetHour() time.Time {
	if o == nil || o.Hour == nil {
		var ret time.Time
		return ret
	}
	return *o.Hour
}

// GetHourOk returns a tuple with the Hour field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageRumSessionsHour) GetHourOk() (*time.Time, bool) {
	if o == nil || o.Hour == nil {
		return nil, false
	}
	return o.Hour, true
}

// HasHour returns a boolean if a field has been set.
func (o *UsageRumSessionsHour) HasHour() bool {
	if o != nil && o.Hour != nil {
		return true
	}

	return false
}

// SetHour gets a reference to the given time.Time and assigns it to the Hour field.
func (o *UsageRumSessionsHour) SetHour(v time.Time) {
	o.Hour = &v
}

// GetOrgName returns the OrgName field value if set, zero value otherwise.
func (o *UsageRumSessionsHour) GetOrgName() string {
	if o == nil || o.OrgName == nil {
		var ret string
		return ret
	}
	return *o.OrgName
}

// GetOrgNameOk returns a tuple with the OrgName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageRumSessionsHour) GetOrgNameOk() (*string, bool) {
	if o == nil || o.OrgName == nil {
		return nil, false
	}
	return o.OrgName, true
}

// HasOrgName returns a boolean if a field has been set.
func (o *UsageRumSessionsHour) HasOrgName() bool {
	if o != nil && o.OrgName != nil {
		return true
	}

	return false
}

// SetOrgName gets a reference to the given string and assigns it to the OrgName field.
func (o *UsageRumSessionsHour) SetOrgName(v string) {
	o.OrgName = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *UsageRumSessionsHour) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageRumSessionsHour) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *UsageRumSessionsHour) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *UsageRumSessionsHour) SetPublicId(v string) {
	o.PublicId = &v
}

// GetReplaySessionCount returns the ReplaySessionCount field value if set, zero value otherwise.
func (o *UsageRumSessionsHour) GetReplaySessionCount() int64 {
	if o == nil || o.ReplaySessionCount == nil {
		var ret int64
		return ret
	}
	return *o.ReplaySessionCount
}

// GetReplaySessionCountOk returns a tuple with the ReplaySessionCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageRumSessionsHour) GetReplaySessionCountOk() (*int64, bool) {
	if o == nil || o.ReplaySessionCount == nil {
		return nil, false
	}
	return o.ReplaySessionCount, true
}

// HasReplaySessionCount returns a boolean if a field has been set.
func (o *UsageRumSessionsHour) HasReplaySessionCount() bool {
	if o != nil && o.ReplaySessionCount != nil {
		return true
	}

	return false
}

// SetReplaySessionCount gets a reference to the given int64 and assigns it to the ReplaySessionCount field.
func (o *UsageRumSessionsHour) SetReplaySessionCount(v int64) {
	o.ReplaySessionCount = &v
}

// GetSessionCount returns the SessionCount field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *UsageRumSessionsHour) GetSessionCount() int64 {
	if o == nil || o.SessionCount.Get() == nil {
		var ret int64
		return ret
	}
	return *o.SessionCount.Get()
}

// GetSessionCountOk returns a tuple with the SessionCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *UsageRumSessionsHour) GetSessionCountOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.SessionCount.Get(), o.SessionCount.IsSet()
}

// HasSessionCount returns a boolean if a field has been set.
func (o *UsageRumSessionsHour) HasSessionCount() bool {
	if o != nil && o.SessionCount.IsSet() {
		return true
	}

	return false
}

// SetSessionCount gets a reference to the given NullableInt64 and assigns it to the SessionCount field.
func (o *UsageRumSessionsHour) SetSessionCount(v int64) {
	o.SessionCount.Set(&v)
}

// SetSessionCountNil sets the value for SessionCount to be an explicit nil.
func (o *UsageRumSessionsHour) SetSessionCountNil() {
	o.SessionCount.Set(nil)
}

// UnsetSessionCount ensures that no value is present for SessionCount, not even an explicit nil.
func (o *UsageRumSessionsHour) UnsetSessionCount() {
	o.SessionCount.Unset()
}

// GetSessionCountAndroid returns the SessionCountAndroid field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *UsageRumSessionsHour) GetSessionCountAndroid() int64 {
	if o == nil || o.SessionCountAndroid.Get() == nil {
		var ret int64
		return ret
	}
	return *o.SessionCountAndroid.Get()
}

// GetSessionCountAndroidOk returns a tuple with the SessionCountAndroid field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *UsageRumSessionsHour) GetSessionCountAndroidOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.SessionCountAndroid.Get(), o.SessionCountAndroid.IsSet()
}

// HasSessionCountAndroid returns a boolean if a field has been set.
func (o *UsageRumSessionsHour) HasSessionCountAndroid() bool {
	if o != nil && o.SessionCountAndroid.IsSet() {
		return true
	}

	return false
}

// SetSessionCountAndroid gets a reference to the given NullableInt64 and assigns it to the SessionCountAndroid field.
func (o *UsageRumSessionsHour) SetSessionCountAndroid(v int64) {
	o.SessionCountAndroid.Set(&v)
}

// SetSessionCountAndroidNil sets the value for SessionCountAndroid to be an explicit nil.
func (o *UsageRumSessionsHour) SetSessionCountAndroidNil() {
	o.SessionCountAndroid.Set(nil)
}

// UnsetSessionCountAndroid ensures that no value is present for SessionCountAndroid, not even an explicit nil.
func (o *UsageRumSessionsHour) UnsetSessionCountAndroid() {
	o.SessionCountAndroid.Unset()
}

// GetSessionCountIos returns the SessionCountIos field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *UsageRumSessionsHour) GetSessionCountIos() int64 {
	if o == nil || o.SessionCountIos.Get() == nil {
		var ret int64
		return ret
	}
	return *o.SessionCountIos.Get()
}

// GetSessionCountIosOk returns a tuple with the SessionCountIos field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *UsageRumSessionsHour) GetSessionCountIosOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.SessionCountIos.Get(), o.SessionCountIos.IsSet()
}

// HasSessionCountIos returns a boolean if a field has been set.
func (o *UsageRumSessionsHour) HasSessionCountIos() bool {
	if o != nil && o.SessionCountIos.IsSet() {
		return true
	}

	return false
}

// SetSessionCountIos gets a reference to the given NullableInt64 and assigns it to the SessionCountIos field.
func (o *UsageRumSessionsHour) SetSessionCountIos(v int64) {
	o.SessionCountIos.Set(&v)
}

// SetSessionCountIosNil sets the value for SessionCountIos to be an explicit nil.
func (o *UsageRumSessionsHour) SetSessionCountIosNil() {
	o.SessionCountIos.Set(nil)
}

// UnsetSessionCountIos ensures that no value is present for SessionCountIos, not even an explicit nil.
func (o *UsageRumSessionsHour) UnsetSessionCountIos() {
	o.SessionCountIos.Unset()
}

// GetSessionCountReactnative returns the SessionCountReactnative field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *UsageRumSessionsHour) GetSessionCountReactnative() int64 {
	if o == nil || o.SessionCountReactnative.Get() == nil {
		var ret int64
		return ret
	}
	return *o.SessionCountReactnative.Get()
}

// GetSessionCountReactnativeOk returns a tuple with the SessionCountReactnative field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *UsageRumSessionsHour) GetSessionCountReactnativeOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.SessionCountReactnative.Get(), o.SessionCountReactnative.IsSet()
}

// HasSessionCountReactnative returns a boolean if a field has been set.
func (o *UsageRumSessionsHour) HasSessionCountReactnative() bool {
	if o != nil && o.SessionCountReactnative.IsSet() {
		return true
	}

	return false
}

// SetSessionCountReactnative gets a reference to the given NullableInt64 and assigns it to the SessionCountReactnative field.
func (o *UsageRumSessionsHour) SetSessionCountReactnative(v int64) {
	o.SessionCountReactnative.Set(&v)
}

// SetSessionCountReactnativeNil sets the value for SessionCountReactnative to be an explicit nil.
func (o *UsageRumSessionsHour) SetSessionCountReactnativeNil() {
	o.SessionCountReactnative.Set(nil)
}

// UnsetSessionCountReactnative ensures that no value is present for SessionCountReactnative, not even an explicit nil.
func (o *UsageRumSessionsHour) UnsetSessionCountReactnative() {
	o.SessionCountReactnative.Unset()
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageRumSessionsHour) MarshalJSON() ([]byte, error) {
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
	if o.OrgName != nil {
		toSerialize["org_name"] = o.OrgName
	}
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
	}
	if o.ReplaySessionCount != nil {
		toSerialize["replay_session_count"] = o.ReplaySessionCount
	}
	if o.SessionCount.IsSet() {
		toSerialize["session_count"] = o.SessionCount.Get()
	}
	if o.SessionCountAndroid.IsSet() {
		toSerialize["session_count_android"] = o.SessionCountAndroid.Get()
	}
	if o.SessionCountIos.IsSet() {
		toSerialize["session_count_ios"] = o.SessionCountIos.Get()
	}
	if o.SessionCountReactnative.IsSet() {
		toSerialize["session_count_reactnative"] = o.SessionCountReactnative.Get()
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageRumSessionsHour) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Hour                    *time.Time    `json:"hour,omitempty"`
		OrgName                 *string       `json:"org_name,omitempty"`
		PublicId                *string       `json:"public_id,omitempty"`
		ReplaySessionCount      *int64        `json:"replay_session_count,omitempty"`
		SessionCount            NullableInt64 `json:"session_count,omitempty"`
		SessionCountAndroid     NullableInt64 `json:"session_count_android,omitempty"`
		SessionCountIos         NullableInt64 `json:"session_count_ios,omitempty"`
		SessionCountReactnative NullableInt64 `json:"session_count_reactnative,omitempty"`
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
	o.OrgName = all.OrgName
	o.PublicId = all.PublicId
	o.ReplaySessionCount = all.ReplaySessionCount
	o.SessionCount = all.SessionCount
	o.SessionCountAndroid = all.SessionCountAndroid
	o.SessionCountIos = all.SessionCountIos
	o.SessionCountReactnative = all.SessionCountReactnative
	return nil
}
