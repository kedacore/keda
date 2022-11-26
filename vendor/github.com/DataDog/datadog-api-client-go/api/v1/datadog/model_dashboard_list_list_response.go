// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// DashboardListListResponse Information on your dashboard lists.
type DashboardListListResponse struct {
	// List of all your dashboard lists.
	DashboardLists []DashboardList `json:"dashboard_lists,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewDashboardListListResponse instantiates a new DashboardListListResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewDashboardListListResponse() *DashboardListListResponse {
	this := DashboardListListResponse{}
	return &this
}

// NewDashboardListListResponseWithDefaults instantiates a new DashboardListListResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewDashboardListListResponseWithDefaults() *DashboardListListResponse {
	this := DashboardListListResponse{}
	return &this
}

// GetDashboardLists returns the DashboardLists field value if set, zero value otherwise.
func (o *DashboardListListResponse) GetDashboardLists() []DashboardList {
	if o == nil || o.DashboardLists == nil {
		var ret []DashboardList
		return ret
	}
	return o.DashboardLists
}

// GetDashboardListsOk returns a tuple with the DashboardLists field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardListListResponse) GetDashboardListsOk() (*[]DashboardList, bool) {
	if o == nil || o.DashboardLists == nil {
		return nil, false
	}
	return &o.DashboardLists, true
}

// HasDashboardLists returns a boolean if a field has been set.
func (o *DashboardListListResponse) HasDashboardLists() bool {
	if o != nil && o.DashboardLists != nil {
		return true
	}

	return false
}

// SetDashboardLists gets a reference to the given []DashboardList and assigns it to the DashboardLists field.
func (o *DashboardListListResponse) SetDashboardLists(v []DashboardList) {
	o.DashboardLists = v
}

// MarshalJSON serializes the struct using spec logic.
func (o DashboardListListResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.DashboardLists != nil {
		toSerialize["dashboard_lists"] = o.DashboardLists
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *DashboardListListResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		DashboardLists []DashboardList `json:"dashboard_lists,omitempty"`
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
	o.DashboardLists = all.DashboardLists
	return nil
}
