// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// DashboardDeleteResponse Response from the delete dashboard call.
type DashboardDeleteResponse struct {
	// ID of the deleted dashboard.
	DeletedDashboardId *string `json:"deleted_dashboard_id,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewDashboardDeleteResponse instantiates a new DashboardDeleteResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewDashboardDeleteResponse() *DashboardDeleteResponse {
	this := DashboardDeleteResponse{}
	return &this
}

// NewDashboardDeleteResponseWithDefaults instantiates a new DashboardDeleteResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewDashboardDeleteResponseWithDefaults() *DashboardDeleteResponse {
	this := DashboardDeleteResponse{}
	return &this
}

// GetDeletedDashboardId returns the DeletedDashboardId field value if set, zero value otherwise.
func (o *DashboardDeleteResponse) GetDeletedDashboardId() string {
	if o == nil || o.DeletedDashboardId == nil {
		var ret string
		return ret
	}
	return *o.DeletedDashboardId
}

// GetDeletedDashboardIdOk returns a tuple with the DeletedDashboardId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardDeleteResponse) GetDeletedDashboardIdOk() (*string, bool) {
	if o == nil || o.DeletedDashboardId == nil {
		return nil, false
	}
	return o.DeletedDashboardId, true
}

// HasDeletedDashboardId returns a boolean if a field has been set.
func (o *DashboardDeleteResponse) HasDeletedDashboardId() bool {
	if o != nil && o.DeletedDashboardId != nil {
		return true
	}

	return false
}

// SetDeletedDashboardId gets a reference to the given string and assigns it to the DeletedDashboardId field.
func (o *DashboardDeleteResponse) SetDeletedDashboardId(v string) {
	o.DeletedDashboardId = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o DashboardDeleteResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.DeletedDashboardId != nil {
		toSerialize["deleted_dashboard_id"] = o.DeletedDashboardId
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *DashboardDeleteResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		DeletedDashboardId *string `json:"deleted_dashboard_id,omitempty"`
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
	o.DeletedDashboardId = all.DeletedDashboardId
	return nil
}
