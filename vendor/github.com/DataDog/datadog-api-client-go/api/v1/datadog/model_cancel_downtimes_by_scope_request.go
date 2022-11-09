// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// CancelDowntimesByScopeRequest Cancel downtimes according to scope.
type CancelDowntimesByScopeRequest struct {
	// The scope(s) to which the downtime applies. For example, `host:app2`.
	// Provide multiple scopes as a comma-separated list like `env:dev,env:prod`.
	// The resulting downtime applies to sources that matches ALL provided scopes (`env:dev` **AND** `env:prod`).
	Scope string `json:"scope"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewCancelDowntimesByScopeRequest instantiates a new CancelDowntimesByScopeRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCancelDowntimesByScopeRequest(scope string) *CancelDowntimesByScopeRequest {
	this := CancelDowntimesByScopeRequest{}
	this.Scope = scope
	return &this
}

// NewCancelDowntimesByScopeRequestWithDefaults instantiates a new CancelDowntimesByScopeRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCancelDowntimesByScopeRequestWithDefaults() *CancelDowntimesByScopeRequest {
	this := CancelDowntimesByScopeRequest{}
	return &this
}

// GetScope returns the Scope field value.
func (o *CancelDowntimesByScopeRequest) GetScope() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Scope
}

// GetScopeOk returns a tuple with the Scope field value
// and a boolean to check if the value has been set.
func (o *CancelDowntimesByScopeRequest) GetScopeOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Scope, true
}

// SetScope sets field value.
func (o *CancelDowntimesByScopeRequest) SetScope(v string) {
	o.Scope = v
}

// MarshalJSON serializes the struct using spec logic.
func (o CancelDowntimesByScopeRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["scope"] = o.Scope

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CancelDowntimesByScopeRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Scope *string `json:"scope"`
	}{}
	all := struct {
		Scope string `json:"scope"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Scope == nil {
		return fmt.Errorf("Required field scope missing")
	}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Scope = all.Scope
	return nil
}
