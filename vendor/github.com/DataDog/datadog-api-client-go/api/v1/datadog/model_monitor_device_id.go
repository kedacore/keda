// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// MonitorDeviceID ID of the device the Synthetics monitor is running on. Same as `SyntheticsDeviceID`.
type MonitorDeviceID string

// List of MonitorDeviceID.
const (
	MONITORDEVICEID_LAPTOP_LARGE         MonitorDeviceID = "laptop_large"
	MONITORDEVICEID_TABLET               MonitorDeviceID = "tablet"
	MONITORDEVICEID_MOBILE_SMALL         MonitorDeviceID = "mobile_small"
	MONITORDEVICEID_CHROME_LAPTOP_LARGE  MonitorDeviceID = "chrome.laptop_large"
	MONITORDEVICEID_CHROME_TABLET        MonitorDeviceID = "chrome.tablet"
	MONITORDEVICEID_CHROME_MOBILE_SMALL  MonitorDeviceID = "chrome.mobile_small"
	MONITORDEVICEID_FIREFOX_LAPTOP_LARGE MonitorDeviceID = "firefox.laptop_large"
	MONITORDEVICEID_FIREFOX_TABLET       MonitorDeviceID = "firefox.tablet"
	MONITORDEVICEID_FIREFOX_MOBILE_SMALL MonitorDeviceID = "firefox.mobile_small"
)

var allowedMonitorDeviceIDEnumValues = []MonitorDeviceID{
	MONITORDEVICEID_LAPTOP_LARGE,
	MONITORDEVICEID_TABLET,
	MONITORDEVICEID_MOBILE_SMALL,
	MONITORDEVICEID_CHROME_LAPTOP_LARGE,
	MONITORDEVICEID_CHROME_TABLET,
	MONITORDEVICEID_CHROME_MOBILE_SMALL,
	MONITORDEVICEID_FIREFOX_LAPTOP_LARGE,
	MONITORDEVICEID_FIREFOX_TABLET,
	MONITORDEVICEID_FIREFOX_MOBILE_SMALL,
}

// GetAllowedValues reeturns the list of possible values.
func (v *MonitorDeviceID) GetAllowedValues() []MonitorDeviceID {
	return allowedMonitorDeviceIDEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *MonitorDeviceID) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = MonitorDeviceID(value)
	return nil
}

// NewMonitorDeviceIDFromValue returns a pointer to a valid MonitorDeviceID
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewMonitorDeviceIDFromValue(v string) (*MonitorDeviceID, error) {
	ev := MonitorDeviceID(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for MonitorDeviceID: valid values are %v", v, allowedMonitorDeviceIDEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v MonitorDeviceID) IsValid() bool {
	for _, existing := range allowedMonitorDeviceIDEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to MonitorDeviceID value.
func (v MonitorDeviceID) Ptr() *MonitorDeviceID {
	return &v
}

// NullableMonitorDeviceID handles when a null is used for MonitorDeviceID.
type NullableMonitorDeviceID struct {
	value *MonitorDeviceID
	isSet bool
}

// Get returns the associated value.
func (v NullableMonitorDeviceID) Get() *MonitorDeviceID {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMonitorDeviceID) Set(val *MonitorDeviceID) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMonitorDeviceID) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableMonitorDeviceID) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMonitorDeviceID initializes the struct as if Set has been called.
func NewNullableMonitorDeviceID(val *MonitorDeviceID) *NullableMonitorDeviceID {
	return &NullableMonitorDeviceID{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMonitorDeviceID) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMonitorDeviceID) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
