// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// MonitorGroupSearchResult A single monitor group search result.
type MonitorGroupSearchResult struct {
	// The name of the group.
	Group *string `json:"group,omitempty"`
	// The list of tags of the monitor group.
	GroupTags []string `json:"group_tags,omitempty"`
	// Latest timestamp the monitor group was in NO_DATA state.
	LastNodataTs *int64 `json:"last_nodata_ts,omitempty"`
	// Latest timestamp the monitor group triggered.
	LastTriggeredTs NullableInt64 `json:"last_triggered_ts,omitempty"`
	// The ID of the monitor.
	MonitorId *int64 `json:"monitor_id,omitempty"`
	// The name of the monitor.
	MonitorName *string `json:"monitor_name,omitempty"`
	// The different states your monitor can be in.
	Status *MonitorOverallStates `json:"status,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMonitorGroupSearchResult instantiates a new MonitorGroupSearchResult object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorGroupSearchResult() *MonitorGroupSearchResult {
	this := MonitorGroupSearchResult{}
	return &this
}

// NewMonitorGroupSearchResultWithDefaults instantiates a new MonitorGroupSearchResult object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorGroupSearchResultWithDefaults() *MonitorGroupSearchResult {
	this := MonitorGroupSearchResult{}
	return &this
}

// GetGroup returns the Group field value if set, zero value otherwise.
func (o *MonitorGroupSearchResult) GetGroup() string {
	if o == nil || o.Group == nil {
		var ret string
		return ret
	}
	return *o.Group
}

// GetGroupOk returns a tuple with the Group field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorGroupSearchResult) GetGroupOk() (*string, bool) {
	if o == nil || o.Group == nil {
		return nil, false
	}
	return o.Group, true
}

// HasGroup returns a boolean if a field has been set.
func (o *MonitorGroupSearchResult) HasGroup() bool {
	if o != nil && o.Group != nil {
		return true
	}

	return false
}

// SetGroup gets a reference to the given string and assigns it to the Group field.
func (o *MonitorGroupSearchResult) SetGroup(v string) {
	o.Group = &v
}

// GetGroupTags returns the GroupTags field value if set, zero value otherwise.
func (o *MonitorGroupSearchResult) GetGroupTags() []string {
	if o == nil || o.GroupTags == nil {
		var ret []string
		return ret
	}
	return o.GroupTags
}

// GetGroupTagsOk returns a tuple with the GroupTags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorGroupSearchResult) GetGroupTagsOk() (*[]string, bool) {
	if o == nil || o.GroupTags == nil {
		return nil, false
	}
	return &o.GroupTags, true
}

// HasGroupTags returns a boolean if a field has been set.
func (o *MonitorGroupSearchResult) HasGroupTags() bool {
	if o != nil && o.GroupTags != nil {
		return true
	}

	return false
}

// SetGroupTags gets a reference to the given []string and assigns it to the GroupTags field.
func (o *MonitorGroupSearchResult) SetGroupTags(v []string) {
	o.GroupTags = v
}

// GetLastNodataTs returns the LastNodataTs field value if set, zero value otherwise.
func (o *MonitorGroupSearchResult) GetLastNodataTs() int64 {
	if o == nil || o.LastNodataTs == nil {
		var ret int64
		return ret
	}
	return *o.LastNodataTs
}

// GetLastNodataTsOk returns a tuple with the LastNodataTs field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorGroupSearchResult) GetLastNodataTsOk() (*int64, bool) {
	if o == nil || o.LastNodataTs == nil {
		return nil, false
	}
	return o.LastNodataTs, true
}

// HasLastNodataTs returns a boolean if a field has been set.
func (o *MonitorGroupSearchResult) HasLastNodataTs() bool {
	if o != nil && o.LastNodataTs != nil {
		return true
	}

	return false
}

// SetLastNodataTs gets a reference to the given int64 and assigns it to the LastNodataTs field.
func (o *MonitorGroupSearchResult) SetLastNodataTs(v int64) {
	o.LastNodataTs = &v
}

// GetLastTriggeredTs returns the LastTriggeredTs field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MonitorGroupSearchResult) GetLastTriggeredTs() int64 {
	if o == nil || o.LastTriggeredTs.Get() == nil {
		var ret int64
		return ret
	}
	return *o.LastTriggeredTs.Get()
}

// GetLastTriggeredTsOk returns a tuple with the LastTriggeredTs field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MonitorGroupSearchResult) GetLastTriggeredTsOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.LastTriggeredTs.Get(), o.LastTriggeredTs.IsSet()
}

// HasLastTriggeredTs returns a boolean if a field has been set.
func (o *MonitorGroupSearchResult) HasLastTriggeredTs() bool {
	if o != nil && o.LastTriggeredTs.IsSet() {
		return true
	}

	return false
}

// SetLastTriggeredTs gets a reference to the given NullableInt64 and assigns it to the LastTriggeredTs field.
func (o *MonitorGroupSearchResult) SetLastTriggeredTs(v int64) {
	o.LastTriggeredTs.Set(&v)
}

// SetLastTriggeredTsNil sets the value for LastTriggeredTs to be an explicit nil.
func (o *MonitorGroupSearchResult) SetLastTriggeredTsNil() {
	o.LastTriggeredTs.Set(nil)
}

// UnsetLastTriggeredTs ensures that no value is present for LastTriggeredTs, not even an explicit nil.
func (o *MonitorGroupSearchResult) UnsetLastTriggeredTs() {
	o.LastTriggeredTs.Unset()
}

// GetMonitorId returns the MonitorId field value if set, zero value otherwise.
func (o *MonitorGroupSearchResult) GetMonitorId() int64 {
	if o == nil || o.MonitorId == nil {
		var ret int64
		return ret
	}
	return *o.MonitorId
}

// GetMonitorIdOk returns a tuple with the MonitorId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorGroupSearchResult) GetMonitorIdOk() (*int64, bool) {
	if o == nil || o.MonitorId == nil {
		return nil, false
	}
	return o.MonitorId, true
}

// HasMonitorId returns a boolean if a field has been set.
func (o *MonitorGroupSearchResult) HasMonitorId() bool {
	if o != nil && o.MonitorId != nil {
		return true
	}

	return false
}

// SetMonitorId gets a reference to the given int64 and assigns it to the MonitorId field.
func (o *MonitorGroupSearchResult) SetMonitorId(v int64) {
	o.MonitorId = &v
}

// GetMonitorName returns the MonitorName field value if set, zero value otherwise.
func (o *MonitorGroupSearchResult) GetMonitorName() string {
	if o == nil || o.MonitorName == nil {
		var ret string
		return ret
	}
	return *o.MonitorName
}

// GetMonitorNameOk returns a tuple with the MonitorName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorGroupSearchResult) GetMonitorNameOk() (*string, bool) {
	if o == nil || o.MonitorName == nil {
		return nil, false
	}
	return o.MonitorName, true
}

// HasMonitorName returns a boolean if a field has been set.
func (o *MonitorGroupSearchResult) HasMonitorName() bool {
	if o != nil && o.MonitorName != nil {
		return true
	}

	return false
}

// SetMonitorName gets a reference to the given string and assigns it to the MonitorName field.
func (o *MonitorGroupSearchResult) SetMonitorName(v string) {
	o.MonitorName = &v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *MonitorGroupSearchResult) GetStatus() MonitorOverallStates {
	if o == nil || o.Status == nil {
		var ret MonitorOverallStates
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorGroupSearchResult) GetStatusOk() (*MonitorOverallStates, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *MonitorGroupSearchResult) HasStatus() bool {
	if o != nil && o.Status != nil {
		return true
	}

	return false
}

// SetStatus gets a reference to the given MonitorOverallStates and assigns it to the Status field.
func (o *MonitorGroupSearchResult) SetStatus(v MonitorOverallStates) {
	o.Status = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorGroupSearchResult) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Group != nil {
		toSerialize["group"] = o.Group
	}
	if o.GroupTags != nil {
		toSerialize["group_tags"] = o.GroupTags
	}
	if o.LastNodataTs != nil {
		toSerialize["last_nodata_ts"] = o.LastNodataTs
	}
	if o.LastTriggeredTs.IsSet() {
		toSerialize["last_triggered_ts"] = o.LastTriggeredTs.Get()
	}
	if o.MonitorId != nil {
		toSerialize["monitor_id"] = o.MonitorId
	}
	if o.MonitorName != nil {
		toSerialize["monitor_name"] = o.MonitorName
	}
	if o.Status != nil {
		toSerialize["status"] = o.Status
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorGroupSearchResult) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Group           *string               `json:"group,omitempty"`
		GroupTags       []string              `json:"group_tags,omitempty"`
		LastNodataTs    *int64                `json:"last_nodata_ts,omitempty"`
		LastTriggeredTs NullableInt64         `json:"last_triggered_ts,omitempty"`
		MonitorId       *int64                `json:"monitor_id,omitempty"`
		MonitorName     *string               `json:"monitor_name,omitempty"`
		Status          *MonitorOverallStates `json:"status,omitempty"`
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
	if v := all.Status; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Group = all.Group
	o.GroupTags = all.GroupTags
	o.LastNodataTs = all.LastNodataTs
	o.LastTriggeredTs = all.LastTriggeredTs
	o.MonitorId = all.MonitorId
	o.MonitorName = all.MonitorName
	o.Status = all.Status
	return nil
}
