// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsDeviceID The device ID.
type SyntheticsDeviceID string

// List of SyntheticsDeviceID.
const (
	SYNTHETICSDEVICEID_LAPTOP_LARGE         SyntheticsDeviceID = "laptop_large"
	SYNTHETICSDEVICEID_TABLET               SyntheticsDeviceID = "tablet"
	SYNTHETICSDEVICEID_MOBILE_SMALL         SyntheticsDeviceID = "mobile_small"
	SYNTHETICSDEVICEID_CHROME_LAPTOP_LARGE  SyntheticsDeviceID = "chrome.laptop_large"
	SYNTHETICSDEVICEID_CHROME_TABLET        SyntheticsDeviceID = "chrome.tablet"
	SYNTHETICSDEVICEID_CHROME_MOBILE_SMALL  SyntheticsDeviceID = "chrome.mobile_small"
	SYNTHETICSDEVICEID_FIREFOX_LAPTOP_LARGE SyntheticsDeviceID = "firefox.laptop_large"
	SYNTHETICSDEVICEID_FIREFOX_TABLET       SyntheticsDeviceID = "firefox.tablet"
	SYNTHETICSDEVICEID_FIREFOX_MOBILE_SMALL SyntheticsDeviceID = "firefox.mobile_small"
	SYNTHETICSDEVICEID_EDGE_LAPTOP_LARGE    SyntheticsDeviceID = "edge.laptop_large"
	SYNTHETICSDEVICEID_EDGE_TABLET          SyntheticsDeviceID = "edge.tablet"
	SYNTHETICSDEVICEID_EDGE_MOBILE_SMALL    SyntheticsDeviceID = "edge.mobile_small"
)

var allowedSyntheticsDeviceIDEnumValues = []SyntheticsDeviceID{
	SYNTHETICSDEVICEID_LAPTOP_LARGE,
	SYNTHETICSDEVICEID_TABLET,
	SYNTHETICSDEVICEID_MOBILE_SMALL,
	SYNTHETICSDEVICEID_CHROME_LAPTOP_LARGE,
	SYNTHETICSDEVICEID_CHROME_TABLET,
	SYNTHETICSDEVICEID_CHROME_MOBILE_SMALL,
	SYNTHETICSDEVICEID_FIREFOX_LAPTOP_LARGE,
	SYNTHETICSDEVICEID_FIREFOX_TABLET,
	SYNTHETICSDEVICEID_FIREFOX_MOBILE_SMALL,
	SYNTHETICSDEVICEID_EDGE_LAPTOP_LARGE,
	SYNTHETICSDEVICEID_EDGE_TABLET,
	SYNTHETICSDEVICEID_EDGE_MOBILE_SMALL,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsDeviceID) GetAllowedValues() []SyntheticsDeviceID {
	return allowedSyntheticsDeviceIDEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsDeviceID) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsDeviceID(value)
	return nil
}

// NewSyntheticsDeviceIDFromValue returns a pointer to a valid SyntheticsDeviceID
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsDeviceIDFromValue(v string) (*SyntheticsDeviceID, error) {
	ev := SyntheticsDeviceID(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsDeviceID: valid values are %v", v, allowedSyntheticsDeviceIDEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsDeviceID) IsValid() bool {
	for _, existing := range allowedSyntheticsDeviceIDEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsDeviceID value.
func (v SyntheticsDeviceID) Ptr() *SyntheticsDeviceID {
	return &v
}

// NullableSyntheticsDeviceID handles when a null is used for SyntheticsDeviceID.
type NullableSyntheticsDeviceID struct {
	value *SyntheticsDeviceID
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsDeviceID) Get() *SyntheticsDeviceID {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsDeviceID) Set(val *SyntheticsDeviceID) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsDeviceID) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsDeviceID) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsDeviceID initializes the struct as if Set has been called.
func NewNullableSyntheticsDeviceID(val *SyntheticsDeviceID) *NullableSyntheticsDeviceID {
	return &NullableSyntheticsDeviceID{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsDeviceID) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsDeviceID) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
