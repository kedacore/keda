// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// HostListResponse Response with Host information from Datadog.
type HostListResponse struct {
	// Array of hosts.
	HostList []Host `json:"host_list,omitempty"`
	// Number of host matching the query.
	TotalMatching *int64 `json:"total_matching,omitempty"`
	// Number of host returned.
	TotalReturned *int64 `json:"total_returned,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewHostListResponse instantiates a new HostListResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewHostListResponse() *HostListResponse {
	this := HostListResponse{}
	return &this
}

// NewHostListResponseWithDefaults instantiates a new HostListResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewHostListResponseWithDefaults() *HostListResponse {
	this := HostListResponse{}
	return &this
}

// GetHostList returns the HostList field value if set, zero value otherwise.
func (o *HostListResponse) GetHostList() []Host {
	if o == nil || o.HostList == nil {
		var ret []Host
		return ret
	}
	return o.HostList
}

// GetHostListOk returns a tuple with the HostList field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostListResponse) GetHostListOk() (*[]Host, bool) {
	if o == nil || o.HostList == nil {
		return nil, false
	}
	return &o.HostList, true
}

// HasHostList returns a boolean if a field has been set.
func (o *HostListResponse) HasHostList() bool {
	if o != nil && o.HostList != nil {
		return true
	}

	return false
}

// SetHostList gets a reference to the given []Host and assigns it to the HostList field.
func (o *HostListResponse) SetHostList(v []Host) {
	o.HostList = v
}

// GetTotalMatching returns the TotalMatching field value if set, zero value otherwise.
func (o *HostListResponse) GetTotalMatching() int64 {
	if o == nil || o.TotalMatching == nil {
		var ret int64
		return ret
	}
	return *o.TotalMatching
}

// GetTotalMatchingOk returns a tuple with the TotalMatching field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostListResponse) GetTotalMatchingOk() (*int64, bool) {
	if o == nil || o.TotalMatching == nil {
		return nil, false
	}
	return o.TotalMatching, true
}

// HasTotalMatching returns a boolean if a field has been set.
func (o *HostListResponse) HasTotalMatching() bool {
	if o != nil && o.TotalMatching != nil {
		return true
	}

	return false
}

// SetTotalMatching gets a reference to the given int64 and assigns it to the TotalMatching field.
func (o *HostListResponse) SetTotalMatching(v int64) {
	o.TotalMatching = &v
}

// GetTotalReturned returns the TotalReturned field value if set, zero value otherwise.
func (o *HostListResponse) GetTotalReturned() int64 {
	if o == nil || o.TotalReturned == nil {
		var ret int64
		return ret
	}
	return *o.TotalReturned
}

// GetTotalReturnedOk returns a tuple with the TotalReturned field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostListResponse) GetTotalReturnedOk() (*int64, bool) {
	if o == nil || o.TotalReturned == nil {
		return nil, false
	}
	return o.TotalReturned, true
}

// HasTotalReturned returns a boolean if a field has been set.
func (o *HostListResponse) HasTotalReturned() bool {
	if o != nil && o.TotalReturned != nil {
		return true
	}

	return false
}

// SetTotalReturned gets a reference to the given int64 and assigns it to the TotalReturned field.
func (o *HostListResponse) SetTotalReturned(v int64) {
	o.TotalReturned = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o HostListResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.HostList != nil {
		toSerialize["host_list"] = o.HostList
	}
	if o.TotalMatching != nil {
		toSerialize["total_matching"] = o.TotalMatching
	}
	if o.TotalReturned != nil {
		toSerialize["total_returned"] = o.TotalReturned
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *HostListResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		HostList      []Host `json:"host_list,omitempty"`
		TotalMatching *int64 `json:"total_matching,omitempty"`
		TotalReturned *int64 `json:"total_returned,omitempty"`
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
	o.HostList = all.HostList
	o.TotalMatching = all.TotalMatching
	o.TotalReturned = all.TotalReturned
	return nil
}
