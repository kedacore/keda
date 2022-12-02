// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsTestDetailsSubType The subtype of the Synthetic API test, `http`, `ssl`, `tcp`,
// `dns`, `icmp`, `udp`, `websocket`, `grpc` or `multi`.
type SyntheticsTestDetailsSubType string

// List of SyntheticsTestDetailsSubType.
const (
	SYNTHETICSTESTDETAILSSUBTYPE_HTTP      SyntheticsTestDetailsSubType = "http"
	SYNTHETICSTESTDETAILSSUBTYPE_SSL       SyntheticsTestDetailsSubType = "ssl"
	SYNTHETICSTESTDETAILSSUBTYPE_TCP       SyntheticsTestDetailsSubType = "tcp"
	SYNTHETICSTESTDETAILSSUBTYPE_DNS       SyntheticsTestDetailsSubType = "dns"
	SYNTHETICSTESTDETAILSSUBTYPE_MULTI     SyntheticsTestDetailsSubType = "multi"
	SYNTHETICSTESTDETAILSSUBTYPE_ICMP      SyntheticsTestDetailsSubType = "icmp"
	SYNTHETICSTESTDETAILSSUBTYPE_UDP       SyntheticsTestDetailsSubType = "udp"
	SYNTHETICSTESTDETAILSSUBTYPE_WEBSOCKET SyntheticsTestDetailsSubType = "websocket"
	SYNTHETICSTESTDETAILSSUBTYPE_GRPC      SyntheticsTestDetailsSubType = "grpc"
)

var allowedSyntheticsTestDetailsSubTypeEnumValues = []SyntheticsTestDetailsSubType{
	SYNTHETICSTESTDETAILSSUBTYPE_HTTP,
	SYNTHETICSTESTDETAILSSUBTYPE_SSL,
	SYNTHETICSTESTDETAILSSUBTYPE_TCP,
	SYNTHETICSTESTDETAILSSUBTYPE_DNS,
	SYNTHETICSTESTDETAILSSUBTYPE_MULTI,
	SYNTHETICSTESTDETAILSSUBTYPE_ICMP,
	SYNTHETICSTESTDETAILSSUBTYPE_UDP,
	SYNTHETICSTESTDETAILSSUBTYPE_WEBSOCKET,
	SYNTHETICSTESTDETAILSSUBTYPE_GRPC,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsTestDetailsSubType) GetAllowedValues() []SyntheticsTestDetailsSubType {
	return allowedSyntheticsTestDetailsSubTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsTestDetailsSubType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsTestDetailsSubType(value)
	return nil
}

// NewSyntheticsTestDetailsSubTypeFromValue returns a pointer to a valid SyntheticsTestDetailsSubType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsTestDetailsSubTypeFromValue(v string) (*SyntheticsTestDetailsSubType, error) {
	ev := SyntheticsTestDetailsSubType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsTestDetailsSubType: valid values are %v", v, allowedSyntheticsTestDetailsSubTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsTestDetailsSubType) IsValid() bool {
	for _, existing := range allowedSyntheticsTestDetailsSubTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsTestDetailsSubType value.
func (v SyntheticsTestDetailsSubType) Ptr() *SyntheticsTestDetailsSubType {
	return &v
}

// NullableSyntheticsTestDetailsSubType handles when a null is used for SyntheticsTestDetailsSubType.
type NullableSyntheticsTestDetailsSubType struct {
	value *SyntheticsTestDetailsSubType
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsTestDetailsSubType) Get() *SyntheticsTestDetailsSubType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsTestDetailsSubType) Set(val *SyntheticsTestDetailsSubType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsTestDetailsSubType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsTestDetailsSubType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsTestDetailsSubType initializes the struct as if Set has been called.
func NewNullableSyntheticsTestDetailsSubType(val *SyntheticsTestDetailsSubType) *NullableSyntheticsTestDetailsSubType {
	return &NullableSyntheticsTestDetailsSubType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsTestDetailsSubType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsTestDetailsSubType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
