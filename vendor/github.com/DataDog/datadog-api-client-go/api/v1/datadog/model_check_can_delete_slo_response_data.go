// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// CheckCanDeleteSLOResponseData An array of service level objective objects.
type CheckCanDeleteSLOResponseData struct {
	// An array of of SLO IDs that can be safely deleted.
	Ok []string `json:"ok,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewCheckCanDeleteSLOResponseData instantiates a new CheckCanDeleteSLOResponseData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCheckCanDeleteSLOResponseData() *CheckCanDeleteSLOResponseData {
	this := CheckCanDeleteSLOResponseData{}
	return &this
}

// NewCheckCanDeleteSLOResponseDataWithDefaults instantiates a new CheckCanDeleteSLOResponseData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCheckCanDeleteSLOResponseDataWithDefaults() *CheckCanDeleteSLOResponseData {
	this := CheckCanDeleteSLOResponseData{}
	return &this
}

// GetOk returns the Ok field value if set, zero value otherwise.
func (o *CheckCanDeleteSLOResponseData) GetOk() []string {
	if o == nil || o.Ok == nil {
		var ret []string
		return ret
	}
	return o.Ok
}

// GetOkOk returns a tuple with the Ok field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CheckCanDeleteSLOResponseData) GetOkOk() (*[]string, bool) {
	if o == nil || o.Ok == nil {
		return nil, false
	}
	return &o.Ok, true
}

// HasOk returns a boolean if a field has been set.
func (o *CheckCanDeleteSLOResponseData) HasOk() bool {
	if o != nil && o.Ok != nil {
		return true
	}

	return false
}

// SetOk gets a reference to the given []string and assigns it to the Ok field.
func (o *CheckCanDeleteSLOResponseData) SetOk(v []string) {
	o.Ok = v
}

// MarshalJSON serializes the struct using spec logic.
func (o CheckCanDeleteSLOResponseData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Ok != nil {
		toSerialize["ok"] = o.Ok
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CheckCanDeleteSLOResponseData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Ok []string `json:"ok,omitempty"`
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
	o.Ok = all.Ok
	return nil
}
