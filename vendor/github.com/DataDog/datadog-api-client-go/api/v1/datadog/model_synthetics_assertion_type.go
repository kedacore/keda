// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsAssertionType Type of the assertion.
type SyntheticsAssertionType string

// List of SyntheticsAssertionType.
const (
	SYNTHETICSASSERTIONTYPE_BODY                    SyntheticsAssertionType = "body"
	SYNTHETICSASSERTIONTYPE_HEADER                  SyntheticsAssertionType = "header"
	SYNTHETICSASSERTIONTYPE_STATUS_CODE             SyntheticsAssertionType = "statusCode"
	SYNTHETICSASSERTIONTYPE_CERTIFICATE             SyntheticsAssertionType = "certificate"
	SYNTHETICSASSERTIONTYPE_RESPONSE_TIME           SyntheticsAssertionType = "responseTime"
	SYNTHETICSASSERTIONTYPE_PROPERTY                SyntheticsAssertionType = "property"
	SYNTHETICSASSERTIONTYPE_RECORD_EVERY            SyntheticsAssertionType = "recordEvery"
	SYNTHETICSASSERTIONTYPE_RECORD_SOME             SyntheticsAssertionType = "recordSome"
	SYNTHETICSASSERTIONTYPE_TLS_VERSION             SyntheticsAssertionType = "tlsVersion"
	SYNTHETICSASSERTIONTYPE_MIN_TLS_VERSION         SyntheticsAssertionType = "minTlsVersion"
	SYNTHETICSASSERTIONTYPE_LATENCY                 SyntheticsAssertionType = "latency"
	SYNTHETICSASSERTIONTYPE_PACKET_LOSS_PERCENTAGE  SyntheticsAssertionType = "packetLossPercentage"
	SYNTHETICSASSERTIONTYPE_PACKETS_RECEIVED        SyntheticsAssertionType = "packetsReceived"
	SYNTHETICSASSERTIONTYPE_NETWORK_HOP             SyntheticsAssertionType = "networkHop"
	SYNTHETICSASSERTIONTYPE_RECEIVED_MESSAGE        SyntheticsAssertionType = "receivedMessage"
	SYNTHETICSASSERTIONTYPE_GRPC_HEALTHCHECK_STATUS SyntheticsAssertionType = "grpcHealthcheckStatus"
	SYNTHETICSASSERTIONTYPE_CONNECTION              SyntheticsAssertionType = "connection"
)

var allowedSyntheticsAssertionTypeEnumValues = []SyntheticsAssertionType{
	SYNTHETICSASSERTIONTYPE_BODY,
	SYNTHETICSASSERTIONTYPE_HEADER,
	SYNTHETICSASSERTIONTYPE_STATUS_CODE,
	SYNTHETICSASSERTIONTYPE_CERTIFICATE,
	SYNTHETICSASSERTIONTYPE_RESPONSE_TIME,
	SYNTHETICSASSERTIONTYPE_PROPERTY,
	SYNTHETICSASSERTIONTYPE_RECORD_EVERY,
	SYNTHETICSASSERTIONTYPE_RECORD_SOME,
	SYNTHETICSASSERTIONTYPE_TLS_VERSION,
	SYNTHETICSASSERTIONTYPE_MIN_TLS_VERSION,
	SYNTHETICSASSERTIONTYPE_LATENCY,
	SYNTHETICSASSERTIONTYPE_PACKET_LOSS_PERCENTAGE,
	SYNTHETICSASSERTIONTYPE_PACKETS_RECEIVED,
	SYNTHETICSASSERTIONTYPE_NETWORK_HOP,
	SYNTHETICSASSERTIONTYPE_RECEIVED_MESSAGE,
	SYNTHETICSASSERTIONTYPE_GRPC_HEALTHCHECK_STATUS,
	SYNTHETICSASSERTIONTYPE_CONNECTION,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsAssertionType) GetAllowedValues() []SyntheticsAssertionType {
	return allowedSyntheticsAssertionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsAssertionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsAssertionType(value)
	return nil
}

// NewSyntheticsAssertionTypeFromValue returns a pointer to a valid SyntheticsAssertionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsAssertionTypeFromValue(v string) (*SyntheticsAssertionType, error) {
	ev := SyntheticsAssertionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsAssertionType: valid values are %v", v, allowedSyntheticsAssertionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsAssertionType) IsValid() bool {
	for _, existing := range allowedSyntheticsAssertionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsAssertionType value.
func (v SyntheticsAssertionType) Ptr() *SyntheticsAssertionType {
	return &v
}

// NullableSyntheticsAssertionType handles when a null is used for SyntheticsAssertionType.
type NullableSyntheticsAssertionType struct {
	value *SyntheticsAssertionType
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsAssertionType) Get() *SyntheticsAssertionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsAssertionType) Set(val *SyntheticsAssertionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsAssertionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsAssertionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsAssertionType initializes the struct as if Set has been called.
func NewNullableSyntheticsAssertionType(val *SyntheticsAssertionType) *NullableSyntheticsAssertionType {
	return &NullableSyntheticsAssertionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsAssertionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsAssertionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
