// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// HostMetrics Host Metrics collected.
type HostMetrics struct {
	// The percent of CPU used (everything but idle).
	Cpu *float64 `json:"cpu,omitempty"`
	// The percent of CPU spent waiting on the IO (not reported for all platforms).
	Iowait *float64 `json:"iowait,omitempty"`
	// The system load over the last 15 minutes.
	Load *float64 `json:"load,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewHostMetrics instantiates a new HostMetrics object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewHostMetrics() *HostMetrics {
	this := HostMetrics{}
	return &this
}

// NewHostMetricsWithDefaults instantiates a new HostMetrics object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewHostMetricsWithDefaults() *HostMetrics {
	this := HostMetrics{}
	return &this
}

// GetCpu returns the Cpu field value if set, zero value otherwise.
func (o *HostMetrics) GetCpu() float64 {
	if o == nil || o.Cpu == nil {
		var ret float64
		return ret
	}
	return *o.Cpu
}

// GetCpuOk returns a tuple with the Cpu field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMetrics) GetCpuOk() (*float64, bool) {
	if o == nil || o.Cpu == nil {
		return nil, false
	}
	return o.Cpu, true
}

// HasCpu returns a boolean if a field has been set.
func (o *HostMetrics) HasCpu() bool {
	if o != nil && o.Cpu != nil {
		return true
	}

	return false
}

// SetCpu gets a reference to the given float64 and assigns it to the Cpu field.
func (o *HostMetrics) SetCpu(v float64) {
	o.Cpu = &v
}

// GetIowait returns the Iowait field value if set, zero value otherwise.
func (o *HostMetrics) GetIowait() float64 {
	if o == nil || o.Iowait == nil {
		var ret float64
		return ret
	}
	return *o.Iowait
}

// GetIowaitOk returns a tuple with the Iowait field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMetrics) GetIowaitOk() (*float64, bool) {
	if o == nil || o.Iowait == nil {
		return nil, false
	}
	return o.Iowait, true
}

// HasIowait returns a boolean if a field has been set.
func (o *HostMetrics) HasIowait() bool {
	if o != nil && o.Iowait != nil {
		return true
	}

	return false
}

// SetIowait gets a reference to the given float64 and assigns it to the Iowait field.
func (o *HostMetrics) SetIowait(v float64) {
	o.Iowait = &v
}

// GetLoad returns the Load field value if set, zero value otherwise.
func (o *HostMetrics) GetLoad() float64 {
	if o == nil || o.Load == nil {
		var ret float64
		return ret
	}
	return *o.Load
}

// GetLoadOk returns a tuple with the Load field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMetrics) GetLoadOk() (*float64, bool) {
	if o == nil || o.Load == nil {
		return nil, false
	}
	return o.Load, true
}

// HasLoad returns a boolean if a field has been set.
func (o *HostMetrics) HasLoad() bool {
	if o != nil && o.Load != nil {
		return true
	}

	return false
}

// SetLoad gets a reference to the given float64 and assigns it to the Load field.
func (o *HostMetrics) SetLoad(v float64) {
	o.Load = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o HostMetrics) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Cpu != nil {
		toSerialize["cpu"] = o.Cpu
	}
	if o.Iowait != nil {
		toSerialize["iowait"] = o.Iowait
	}
	if o.Load != nil {
		toSerialize["load"] = o.Load
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *HostMetrics) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Cpu    *float64 `json:"cpu,omitempty"`
		Iowait *float64 `json:"iowait,omitempty"`
		Load   *float64 `json:"load,omitempty"`
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
	o.Cpu = all.Cpu
	o.Iowait = all.Iowait
	o.Load = all.Load
	return nil
}
