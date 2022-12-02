// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// PagerDutyServiceKey PagerDuty service object key.
type PagerDutyServiceKey struct {
	// Your service key in PagerDuty.
	ServiceKey string `json:"service_key"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewPagerDutyServiceKey instantiates a new PagerDutyServiceKey object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewPagerDutyServiceKey(serviceKey string) *PagerDutyServiceKey {
	this := PagerDutyServiceKey{}
	this.ServiceKey = serviceKey
	return &this
}

// NewPagerDutyServiceKeyWithDefaults instantiates a new PagerDutyServiceKey object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewPagerDutyServiceKeyWithDefaults() *PagerDutyServiceKey {
	this := PagerDutyServiceKey{}
	return &this
}

// GetServiceKey returns the ServiceKey field value.
func (o *PagerDutyServiceKey) GetServiceKey() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.ServiceKey
}

// GetServiceKeyOk returns a tuple with the ServiceKey field value
// and a boolean to check if the value has been set.
func (o *PagerDutyServiceKey) GetServiceKeyOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ServiceKey, true
}

// SetServiceKey sets field value.
func (o *PagerDutyServiceKey) SetServiceKey(v string) {
	o.ServiceKey = v
}

// MarshalJSON serializes the struct using spec logic.
func (o PagerDutyServiceKey) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["service_key"] = o.ServiceKey

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *PagerDutyServiceKey) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		ServiceKey *string `json:"service_key"`
	}{}
	all := struct {
		ServiceKey string `json:"service_key"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.ServiceKey == nil {
		return fmt.Errorf("Required field service_key missing")
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
	o.ServiceKey = all.ServiceKey
	return nil
}
