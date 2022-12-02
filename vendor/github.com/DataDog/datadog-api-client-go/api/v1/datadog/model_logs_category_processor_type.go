// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsCategoryProcessorType Type of logs category processor.
type LogsCategoryProcessorType string

// List of LogsCategoryProcessorType.
const (
	LOGSCATEGORYPROCESSORTYPE_CATEGORY_PROCESSOR LogsCategoryProcessorType = "category-processor"
)

var allowedLogsCategoryProcessorTypeEnumValues = []LogsCategoryProcessorType{
	LOGSCATEGORYPROCESSORTYPE_CATEGORY_PROCESSOR,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsCategoryProcessorType) GetAllowedValues() []LogsCategoryProcessorType {
	return allowedLogsCategoryProcessorTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsCategoryProcessorType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsCategoryProcessorType(value)
	return nil
}

// NewLogsCategoryProcessorTypeFromValue returns a pointer to a valid LogsCategoryProcessorType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsCategoryProcessorTypeFromValue(v string) (*LogsCategoryProcessorType, error) {
	ev := LogsCategoryProcessorType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsCategoryProcessorType: valid values are %v", v, allowedLogsCategoryProcessorTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsCategoryProcessorType) IsValid() bool {
	for _, existing := range allowedLogsCategoryProcessorTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsCategoryProcessorType value.
func (v LogsCategoryProcessorType) Ptr() *LogsCategoryProcessorType {
	return &v
}

// NullableLogsCategoryProcessorType handles when a null is used for LogsCategoryProcessorType.
type NullableLogsCategoryProcessorType struct {
	value *LogsCategoryProcessorType
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsCategoryProcessorType) Get() *LogsCategoryProcessorType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsCategoryProcessorType) Set(val *LogsCategoryProcessorType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsCategoryProcessorType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsCategoryProcessorType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsCategoryProcessorType initializes the struct as if Set has been called.
func NewNullableLogsCategoryProcessorType(val *LogsCategoryProcessorType) *NullableLogsCategoryProcessorType {
	return &NullableLogsCategoryProcessorType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsCategoryProcessorType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsCategoryProcessorType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
