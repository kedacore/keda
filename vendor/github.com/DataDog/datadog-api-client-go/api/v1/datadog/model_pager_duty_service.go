// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// PagerDutyService The PagerDuty service that is available for integration with Datadog.
type PagerDutyService struct {
	// Your service key in PagerDuty.
	ServiceKey string `json:"service_key"`
	// Your service name associated with a service key in PagerDuty.
	ServiceName string `json:"service_name"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewPagerDutyService instantiates a new PagerDutyService object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewPagerDutyService(serviceKey string, serviceName string) *PagerDutyService {
	this := PagerDutyService{}
	this.ServiceKey = serviceKey
	this.ServiceName = serviceName
	return &this
}

// NewPagerDutyServiceWithDefaults instantiates a new PagerDutyService object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewPagerDutyServiceWithDefaults() *PagerDutyService {
	this := PagerDutyService{}
	return &this
}

// GetServiceKey returns the ServiceKey field value.
func (o *PagerDutyService) GetServiceKey() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.ServiceKey
}

// GetServiceKeyOk returns a tuple with the ServiceKey field value
// and a boolean to check if the value has been set.
func (o *PagerDutyService) GetServiceKeyOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ServiceKey, true
}

// SetServiceKey sets field value.
func (o *PagerDutyService) SetServiceKey(v string) {
	o.ServiceKey = v
}

// GetServiceName returns the ServiceName field value.
func (o *PagerDutyService) GetServiceName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.ServiceName
}

// GetServiceNameOk returns a tuple with the ServiceName field value
// and a boolean to check if the value has been set.
func (o *PagerDutyService) GetServiceNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ServiceName, true
}

// SetServiceName sets field value.
func (o *PagerDutyService) SetServiceName(v string) {
	o.ServiceName = v
}

// MarshalJSON serializes the struct using spec logic.
func (o PagerDutyService) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["service_key"] = o.ServiceKey
	toSerialize["service_name"] = o.ServiceName

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *PagerDutyService) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		ServiceKey  *string `json:"service_key"`
		ServiceName *string `json:"service_name"`
	}{}
	all := struct {
		ServiceKey  string `json:"service_key"`
		ServiceName string `json:"service_name"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.ServiceKey == nil {
		return fmt.Errorf("Required field service_key missing")
	}
	if required.ServiceName == nil {
		return fmt.Errorf("Required field service_name missing")
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
	o.ServiceName = all.ServiceName
	return nil
}
