// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// MonitorThresholds List of the different monitor threshold available.
type MonitorThresholds struct {
	// The monitor `CRITICAL` threshold.
	Critical *float64 `json:"critical,omitempty"`
	// The monitor `CRITICAL` recovery threshold.
	CriticalRecovery NullableFloat64 `json:"critical_recovery,omitempty"`
	// The monitor `OK` threshold.
	Ok NullableFloat64 `json:"ok,omitempty"`
	// The monitor UNKNOWN threshold.
	Unknown NullableFloat64 `json:"unknown,omitempty"`
	// The monitor `WARNING` threshold.
	Warning NullableFloat64 `json:"warning,omitempty"`
	// The monitor `WARNING` recovery threshold.
	WarningRecovery NullableFloat64 `json:"warning_recovery,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMonitorThresholds instantiates a new MonitorThresholds object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorThresholds() *MonitorThresholds {
	this := MonitorThresholds{}
	return &this
}

// NewMonitorThresholdsWithDefaults instantiates a new MonitorThresholds object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorThresholdsWithDefaults() *MonitorThresholds {
	this := MonitorThresholds{}
	return &this
}

// GetCritical returns the Critical field value if set, zero value otherwise.
func (o *MonitorThresholds) GetCritical() float64 {
	if o == nil || o.Critical == nil {
		var ret float64
		return ret
	}
	return *o.Critical
}

// GetCriticalOk returns a tuple with the Critical field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorThresholds) GetCriticalOk() (*float64, bool) {
	if o == nil || o.Critical == nil {
		return nil, false
	}
	return o.Critical, true
}

// HasCritical returns a boolean if a field has been set.
func (o *MonitorThresholds) HasCritical() bool {
	if o != nil && o.Critical != nil {
		return true
	}

	return false
}

// SetCritical gets a reference to the given float64 and assigns it to the Critical field.
func (o *MonitorThresholds) SetCritical(v float64) {
	o.Critical = &v
}

// GetCriticalRecovery returns the CriticalRecovery field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MonitorThresholds) GetCriticalRecovery() float64 {
	if o == nil || o.CriticalRecovery.Get() == nil {
		var ret float64
		return ret
	}
	return *o.CriticalRecovery.Get()
}

// GetCriticalRecoveryOk returns a tuple with the CriticalRecovery field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MonitorThresholds) GetCriticalRecoveryOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return o.CriticalRecovery.Get(), o.CriticalRecovery.IsSet()
}

// HasCriticalRecovery returns a boolean if a field has been set.
func (o *MonitorThresholds) HasCriticalRecovery() bool {
	if o != nil && o.CriticalRecovery.IsSet() {
		return true
	}

	return false
}

// SetCriticalRecovery gets a reference to the given NullableFloat64 and assigns it to the CriticalRecovery field.
func (o *MonitorThresholds) SetCriticalRecovery(v float64) {
	o.CriticalRecovery.Set(&v)
}

// SetCriticalRecoveryNil sets the value for CriticalRecovery to be an explicit nil.
func (o *MonitorThresholds) SetCriticalRecoveryNil() {
	o.CriticalRecovery.Set(nil)
}

// UnsetCriticalRecovery ensures that no value is present for CriticalRecovery, not even an explicit nil.
func (o *MonitorThresholds) UnsetCriticalRecovery() {
	o.CriticalRecovery.Unset()
}

// GetOk returns the Ok field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MonitorThresholds) GetOk() float64 {
	if o == nil || o.Ok.Get() == nil {
		var ret float64
		return ret
	}
	return *o.Ok.Get()
}

// GetOkOk returns a tuple with the Ok field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MonitorThresholds) GetOkOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return o.Ok.Get(), o.Ok.IsSet()
}

// HasOk returns a boolean if a field has been set.
func (o *MonitorThresholds) HasOk() bool {
	if o != nil && o.Ok.IsSet() {
		return true
	}

	return false
}

// SetOk gets a reference to the given NullableFloat64 and assigns it to the Ok field.
func (o *MonitorThresholds) SetOk(v float64) {
	o.Ok.Set(&v)
}

// SetOkNil sets the value for Ok to be an explicit nil.
func (o *MonitorThresholds) SetOkNil() {
	o.Ok.Set(nil)
}

// UnsetOk ensures that no value is present for Ok, not even an explicit nil.
func (o *MonitorThresholds) UnsetOk() {
	o.Ok.Unset()
}

// GetUnknown returns the Unknown field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MonitorThresholds) GetUnknown() float64 {
	if o == nil || o.Unknown.Get() == nil {
		var ret float64
		return ret
	}
	return *o.Unknown.Get()
}

// GetUnknownOk returns a tuple with the Unknown field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MonitorThresholds) GetUnknownOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return o.Unknown.Get(), o.Unknown.IsSet()
}

// HasUnknown returns a boolean if a field has been set.
func (o *MonitorThresholds) HasUnknown() bool {
	if o != nil && o.Unknown.IsSet() {
		return true
	}

	return false
}

// SetUnknown gets a reference to the given NullableFloat64 and assigns it to the Unknown field.
func (o *MonitorThresholds) SetUnknown(v float64) {
	o.Unknown.Set(&v)
}

// SetUnknownNil sets the value for Unknown to be an explicit nil.
func (o *MonitorThresholds) SetUnknownNil() {
	o.Unknown.Set(nil)
}

// UnsetUnknown ensures that no value is present for Unknown, not even an explicit nil.
func (o *MonitorThresholds) UnsetUnknown() {
	o.Unknown.Unset()
}

// GetWarning returns the Warning field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MonitorThresholds) GetWarning() float64 {
	if o == nil || o.Warning.Get() == nil {
		var ret float64
		return ret
	}
	return *o.Warning.Get()
}

// GetWarningOk returns a tuple with the Warning field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MonitorThresholds) GetWarningOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return o.Warning.Get(), o.Warning.IsSet()
}

// HasWarning returns a boolean if a field has been set.
func (o *MonitorThresholds) HasWarning() bool {
	if o != nil && o.Warning.IsSet() {
		return true
	}

	return false
}

// SetWarning gets a reference to the given NullableFloat64 and assigns it to the Warning field.
func (o *MonitorThresholds) SetWarning(v float64) {
	o.Warning.Set(&v)
}

// SetWarningNil sets the value for Warning to be an explicit nil.
func (o *MonitorThresholds) SetWarningNil() {
	o.Warning.Set(nil)
}

// UnsetWarning ensures that no value is present for Warning, not even an explicit nil.
func (o *MonitorThresholds) UnsetWarning() {
	o.Warning.Unset()
}

// GetWarningRecovery returns the WarningRecovery field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MonitorThresholds) GetWarningRecovery() float64 {
	if o == nil || o.WarningRecovery.Get() == nil {
		var ret float64
		return ret
	}
	return *o.WarningRecovery.Get()
}

// GetWarningRecoveryOk returns a tuple with the WarningRecovery field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MonitorThresholds) GetWarningRecoveryOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return o.WarningRecovery.Get(), o.WarningRecovery.IsSet()
}

// HasWarningRecovery returns a boolean if a field has been set.
func (o *MonitorThresholds) HasWarningRecovery() bool {
	if o != nil && o.WarningRecovery.IsSet() {
		return true
	}

	return false
}

// SetWarningRecovery gets a reference to the given NullableFloat64 and assigns it to the WarningRecovery field.
func (o *MonitorThresholds) SetWarningRecovery(v float64) {
	o.WarningRecovery.Set(&v)
}

// SetWarningRecoveryNil sets the value for WarningRecovery to be an explicit nil.
func (o *MonitorThresholds) SetWarningRecoveryNil() {
	o.WarningRecovery.Set(nil)
}

// UnsetWarningRecovery ensures that no value is present for WarningRecovery, not even an explicit nil.
func (o *MonitorThresholds) UnsetWarningRecovery() {
	o.WarningRecovery.Unset()
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorThresholds) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Critical != nil {
		toSerialize["critical"] = o.Critical
	}
	if o.CriticalRecovery.IsSet() {
		toSerialize["critical_recovery"] = o.CriticalRecovery.Get()
	}
	if o.Ok.IsSet() {
		toSerialize["ok"] = o.Ok.Get()
	}
	if o.Unknown.IsSet() {
		toSerialize["unknown"] = o.Unknown.Get()
	}
	if o.Warning.IsSet() {
		toSerialize["warning"] = o.Warning.Get()
	}
	if o.WarningRecovery.IsSet() {
		toSerialize["warning_recovery"] = o.WarningRecovery.Get()
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorThresholds) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Critical         *float64        `json:"critical,omitempty"`
		CriticalRecovery NullableFloat64 `json:"critical_recovery,omitempty"`
		Ok               NullableFloat64 `json:"ok,omitempty"`
		Unknown          NullableFloat64 `json:"unknown,omitempty"`
		Warning          NullableFloat64 `json:"warning,omitempty"`
		WarningRecovery  NullableFloat64 `json:"warning_recovery,omitempty"`
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
	o.Critical = all.Critical
	o.CriticalRecovery = all.CriticalRecovery
	o.Ok = all.Ok
	o.Unknown = all.Unknown
	o.Warning = all.Warning
	o.WarningRecovery = all.WarningRecovery
	return nil
}
