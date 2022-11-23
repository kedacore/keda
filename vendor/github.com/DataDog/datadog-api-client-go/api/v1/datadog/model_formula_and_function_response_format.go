// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// FormulaAndFunctionResponseFormat Timeseries or Scalar response.
type FormulaAndFunctionResponseFormat string

// List of FormulaAndFunctionResponseFormat.
const (
	FORMULAANDFUNCTIONRESPONSEFORMAT_TIMESERIES FormulaAndFunctionResponseFormat = "timeseries"
	FORMULAANDFUNCTIONRESPONSEFORMAT_SCALAR     FormulaAndFunctionResponseFormat = "scalar"
)

var allowedFormulaAndFunctionResponseFormatEnumValues = []FormulaAndFunctionResponseFormat{
	FORMULAANDFUNCTIONRESPONSEFORMAT_TIMESERIES,
	FORMULAANDFUNCTIONRESPONSEFORMAT_SCALAR,
}

// GetAllowedValues reeturns the list of possible values.
func (v *FormulaAndFunctionResponseFormat) GetAllowedValues() []FormulaAndFunctionResponseFormat {
	return allowedFormulaAndFunctionResponseFormatEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *FormulaAndFunctionResponseFormat) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = FormulaAndFunctionResponseFormat(value)
	return nil
}

// NewFormulaAndFunctionResponseFormatFromValue returns a pointer to a valid FormulaAndFunctionResponseFormat
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewFormulaAndFunctionResponseFormatFromValue(v string) (*FormulaAndFunctionResponseFormat, error) {
	ev := FormulaAndFunctionResponseFormat(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for FormulaAndFunctionResponseFormat: valid values are %v", v, allowedFormulaAndFunctionResponseFormatEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v FormulaAndFunctionResponseFormat) IsValid() bool {
	for _, existing := range allowedFormulaAndFunctionResponseFormatEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to FormulaAndFunctionResponseFormat value.
func (v FormulaAndFunctionResponseFormat) Ptr() *FormulaAndFunctionResponseFormat {
	return &v
}

// NullableFormulaAndFunctionResponseFormat handles when a null is used for FormulaAndFunctionResponseFormat.
type NullableFormulaAndFunctionResponseFormat struct {
	value *FormulaAndFunctionResponseFormat
	isSet bool
}

// Get returns the associated value.
func (v NullableFormulaAndFunctionResponseFormat) Get() *FormulaAndFunctionResponseFormat {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableFormulaAndFunctionResponseFormat) Set(val *FormulaAndFunctionResponseFormat) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableFormulaAndFunctionResponseFormat) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableFormulaAndFunctionResponseFormat) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableFormulaAndFunctionResponseFormat initializes the struct as if Set has been called.
func NewNullableFormulaAndFunctionResponseFormat(val *FormulaAndFunctionResponseFormat) *NullableFormulaAndFunctionResponseFormat {
	return &NullableFormulaAndFunctionResponseFormat{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableFormulaAndFunctionResponseFormat) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableFormulaAndFunctionResponseFormat) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
