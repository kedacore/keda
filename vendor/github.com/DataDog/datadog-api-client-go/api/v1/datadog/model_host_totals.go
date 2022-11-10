// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// HostTotals Total number of host currently monitored by Datadog.
type HostTotals struct {
	// Total number of active host (UP and ???) reporting to Datadog.
	TotalActive *int64 `json:"total_active,omitempty"`
	// Number of host that are UP and reporting to Datadog.
	TotalUp *int64 `json:"total_up,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewHostTotals instantiates a new HostTotals object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewHostTotals() *HostTotals {
	this := HostTotals{}
	return &this
}

// NewHostTotalsWithDefaults instantiates a new HostTotals object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewHostTotalsWithDefaults() *HostTotals {
	this := HostTotals{}
	return &this
}

// GetTotalActive returns the TotalActive field value if set, zero value otherwise.
func (o *HostTotals) GetTotalActive() int64 {
	if o == nil || o.TotalActive == nil {
		var ret int64
		return ret
	}
	return *o.TotalActive
}

// GetTotalActiveOk returns a tuple with the TotalActive field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostTotals) GetTotalActiveOk() (*int64, bool) {
	if o == nil || o.TotalActive == nil {
		return nil, false
	}
	return o.TotalActive, true
}

// HasTotalActive returns a boolean if a field has been set.
func (o *HostTotals) HasTotalActive() bool {
	if o != nil && o.TotalActive != nil {
		return true
	}

	return false
}

// SetTotalActive gets a reference to the given int64 and assigns it to the TotalActive field.
func (o *HostTotals) SetTotalActive(v int64) {
	o.TotalActive = &v
}

// GetTotalUp returns the TotalUp field value if set, zero value otherwise.
func (o *HostTotals) GetTotalUp() int64 {
	if o == nil || o.TotalUp == nil {
		var ret int64
		return ret
	}
	return *o.TotalUp
}

// GetTotalUpOk returns a tuple with the TotalUp field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostTotals) GetTotalUpOk() (*int64, bool) {
	if o == nil || o.TotalUp == nil {
		return nil, false
	}
	return o.TotalUp, true
}

// HasTotalUp returns a boolean if a field has been set.
func (o *HostTotals) HasTotalUp() bool {
	if o != nil && o.TotalUp != nil {
		return true
	}

	return false
}

// SetTotalUp gets a reference to the given int64 and assigns it to the TotalUp field.
func (o *HostTotals) SetTotalUp(v int64) {
	o.TotalUp = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o HostTotals) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.TotalActive != nil {
		toSerialize["total_active"] = o.TotalActive
	}
	if o.TotalUp != nil {
		toSerialize["total_up"] = o.TotalUp
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *HostTotals) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		TotalActive *int64 `json:"total_active,omitempty"`
		TotalUp     *int64 `json:"total_up,omitempty"`
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
	o.TotalActive = all.TotalActive
	o.TotalUp = all.TotalUp
	return nil
}
