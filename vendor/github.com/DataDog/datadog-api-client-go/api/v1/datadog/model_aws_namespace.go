// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// AWSNamespace The namespace associated with the tag filter entry.
type AWSNamespace string

// List of AWSNamespace.
const (
	AWSNAMESPACE_ELB             AWSNamespace = "elb"
	AWSNAMESPACE_APPLICATION_ELB AWSNamespace = "application_elb"
	AWSNAMESPACE_SQS             AWSNamespace = "sqs"
	AWSNAMESPACE_RDS             AWSNamespace = "rds"
	AWSNAMESPACE_CUSTOM          AWSNamespace = "custom"
	AWSNAMESPACE_NETWORK_ELB     AWSNamespace = "network_elb"
	AWSNAMESPACE_LAMBDA          AWSNamespace = "lambda"
)

var allowedAWSNamespaceEnumValues = []AWSNamespace{
	AWSNAMESPACE_ELB,
	AWSNAMESPACE_APPLICATION_ELB,
	AWSNAMESPACE_SQS,
	AWSNAMESPACE_RDS,
	AWSNAMESPACE_CUSTOM,
	AWSNAMESPACE_NETWORK_ELB,
	AWSNAMESPACE_LAMBDA,
}

// GetAllowedValues reeturns the list of possible values.
func (v *AWSNamespace) GetAllowedValues() []AWSNamespace {
	return allowedAWSNamespaceEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *AWSNamespace) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = AWSNamespace(value)
	return nil
}

// NewAWSNamespaceFromValue returns a pointer to a valid AWSNamespace
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewAWSNamespaceFromValue(v string) (*AWSNamespace, error) {
	ev := AWSNamespace(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for AWSNamespace: valid values are %v", v, allowedAWSNamespaceEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v AWSNamespace) IsValid() bool {
	for _, existing := range allowedAWSNamespaceEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to AWSNamespace value.
func (v AWSNamespace) Ptr() *AWSNamespace {
	return &v
}

// NullableAWSNamespace handles when a null is used for AWSNamespace.
type NullableAWSNamespace struct {
	value *AWSNamespace
	isSet bool
}

// Get returns the associated value.
func (v NullableAWSNamespace) Get() *AWSNamespace {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableAWSNamespace) Set(val *AWSNamespace) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableAWSNamespace) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableAWSNamespace) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableAWSNamespace initializes the struct as if Set has been called.
func NewNullableAWSNamespace(val *AWSNamespace) *NullableAWSNamespace {
	return &NullableAWSNamespace{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableAWSNamespace) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableAWSNamespace) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
