// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// MonitorThresholdWindowOptions Alerting time window options.
type MonitorThresholdWindowOptions struct {
	// Describes how long an anomalous metric must be normal before the alert recovers.
	RecoveryWindow NullableString `json:"recovery_window,omitempty"`
	// Describes how long a metric must be anomalous before an alert triggers.
	TriggerWindow NullableString `json:"trigger_window,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMonitorThresholdWindowOptions instantiates a new MonitorThresholdWindowOptions object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorThresholdWindowOptions() *MonitorThresholdWindowOptions {
	this := MonitorThresholdWindowOptions{}
	return &this
}

// NewMonitorThresholdWindowOptionsWithDefaults instantiates a new MonitorThresholdWindowOptions object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorThresholdWindowOptionsWithDefaults() *MonitorThresholdWindowOptions {
	this := MonitorThresholdWindowOptions{}
	return &this
}

// GetRecoveryWindow returns the RecoveryWindow field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MonitorThresholdWindowOptions) GetRecoveryWindow() string {
	if o == nil || o.RecoveryWindow.Get() == nil {
		var ret string
		return ret
	}
	return *o.RecoveryWindow.Get()
}

// GetRecoveryWindowOk returns a tuple with the RecoveryWindow field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MonitorThresholdWindowOptions) GetRecoveryWindowOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.RecoveryWindow.Get(), o.RecoveryWindow.IsSet()
}

// HasRecoveryWindow returns a boolean if a field has been set.
func (o *MonitorThresholdWindowOptions) HasRecoveryWindow() bool {
	if o != nil && o.RecoveryWindow.IsSet() {
		return true
	}

	return false
}

// SetRecoveryWindow gets a reference to the given NullableString and assigns it to the RecoveryWindow field.
func (o *MonitorThresholdWindowOptions) SetRecoveryWindow(v string) {
	o.RecoveryWindow.Set(&v)
}

// SetRecoveryWindowNil sets the value for RecoveryWindow to be an explicit nil.
func (o *MonitorThresholdWindowOptions) SetRecoveryWindowNil() {
	o.RecoveryWindow.Set(nil)
}

// UnsetRecoveryWindow ensures that no value is present for RecoveryWindow, not even an explicit nil.
func (o *MonitorThresholdWindowOptions) UnsetRecoveryWindow() {
	o.RecoveryWindow.Unset()
}

// GetTriggerWindow returns the TriggerWindow field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MonitorThresholdWindowOptions) GetTriggerWindow() string {
	if o == nil || o.TriggerWindow.Get() == nil {
		var ret string
		return ret
	}
	return *o.TriggerWindow.Get()
}

// GetTriggerWindowOk returns a tuple with the TriggerWindow field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MonitorThresholdWindowOptions) GetTriggerWindowOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.TriggerWindow.Get(), o.TriggerWindow.IsSet()
}

// HasTriggerWindow returns a boolean if a field has been set.
func (o *MonitorThresholdWindowOptions) HasTriggerWindow() bool {
	if o != nil && o.TriggerWindow.IsSet() {
		return true
	}

	return false
}

// SetTriggerWindow gets a reference to the given NullableString and assigns it to the TriggerWindow field.
func (o *MonitorThresholdWindowOptions) SetTriggerWindow(v string) {
	o.TriggerWindow.Set(&v)
}

// SetTriggerWindowNil sets the value for TriggerWindow to be an explicit nil.
func (o *MonitorThresholdWindowOptions) SetTriggerWindowNil() {
	o.TriggerWindow.Set(nil)
}

// UnsetTriggerWindow ensures that no value is present for TriggerWindow, not even an explicit nil.
func (o *MonitorThresholdWindowOptions) UnsetTriggerWindow() {
	o.TriggerWindow.Unset()
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorThresholdWindowOptions) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.RecoveryWindow.IsSet() {
		toSerialize["recovery_window"] = o.RecoveryWindow.Get()
	}
	if o.TriggerWindow.IsSet() {
		toSerialize["trigger_window"] = o.TriggerWindow.Get()
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorThresholdWindowOptions) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		RecoveryWindow NullableString `json:"recovery_window,omitempty"`
		TriggerWindow  NullableString `json:"trigger_window,omitempty"`
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
	o.RecoveryWindow = all.RecoveryWindow
	o.TriggerWindow = all.TriggerWindow
	return nil
}
