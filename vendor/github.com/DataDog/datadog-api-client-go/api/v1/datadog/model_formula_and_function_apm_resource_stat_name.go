// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// FormulaAndFunctionApmResourceStatName APM resource stat name.
type FormulaAndFunctionApmResourceStatName string

// List of FormulaAndFunctionApmResourceStatName.
const (
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_ERRORS               FormulaAndFunctionApmResourceStatName = "errors"
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_ERROR_RATE           FormulaAndFunctionApmResourceStatName = "error_rate"
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_HITS                 FormulaAndFunctionApmResourceStatName = "hits"
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_LATENCY_AVG          FormulaAndFunctionApmResourceStatName = "latency_avg"
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_LATENCY_DISTRIBUTION FormulaAndFunctionApmResourceStatName = "latency_distribution"
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_LATENCY_MAX          FormulaAndFunctionApmResourceStatName = "latency_max"
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_LATENCY_P50          FormulaAndFunctionApmResourceStatName = "latency_p50"
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_LATENCY_P75          FormulaAndFunctionApmResourceStatName = "latency_p75"
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_LATENCY_P90          FormulaAndFunctionApmResourceStatName = "latency_p90"
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_LATENCY_P95          FormulaAndFunctionApmResourceStatName = "latency_p95"
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_LATENCY_P99          FormulaAndFunctionApmResourceStatName = "latency_p99"
)

var allowedFormulaAndFunctionApmResourceStatNameEnumValues = []FormulaAndFunctionApmResourceStatName{
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_ERRORS,
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_ERROR_RATE,
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_HITS,
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_LATENCY_AVG,
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_LATENCY_DISTRIBUTION,
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_LATENCY_MAX,
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_LATENCY_P50,
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_LATENCY_P75,
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_LATENCY_P90,
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_LATENCY_P95,
	FORMULAANDFUNCTIONAPMRESOURCESTATNAME_LATENCY_P99,
}

// GetAllowedValues reeturns the list of possible values.
func (v *FormulaAndFunctionApmResourceStatName) GetAllowedValues() []FormulaAndFunctionApmResourceStatName {
	return allowedFormulaAndFunctionApmResourceStatNameEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *FormulaAndFunctionApmResourceStatName) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = FormulaAndFunctionApmResourceStatName(value)
	return nil
}

// NewFormulaAndFunctionApmResourceStatNameFromValue returns a pointer to a valid FormulaAndFunctionApmResourceStatName
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewFormulaAndFunctionApmResourceStatNameFromValue(v string) (*FormulaAndFunctionApmResourceStatName, error) {
	ev := FormulaAndFunctionApmResourceStatName(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for FormulaAndFunctionApmResourceStatName: valid values are %v", v, allowedFormulaAndFunctionApmResourceStatNameEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v FormulaAndFunctionApmResourceStatName) IsValid() bool {
	for _, existing := range allowedFormulaAndFunctionApmResourceStatNameEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to FormulaAndFunctionApmResourceStatName value.
func (v FormulaAndFunctionApmResourceStatName) Ptr() *FormulaAndFunctionApmResourceStatName {
	return &v
}

// NullableFormulaAndFunctionApmResourceStatName handles when a null is used for FormulaAndFunctionApmResourceStatName.
type NullableFormulaAndFunctionApmResourceStatName struct {
	value *FormulaAndFunctionApmResourceStatName
	isSet bool
}

// Get returns the associated value.
func (v NullableFormulaAndFunctionApmResourceStatName) Get() *FormulaAndFunctionApmResourceStatName {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableFormulaAndFunctionApmResourceStatName) Set(val *FormulaAndFunctionApmResourceStatName) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableFormulaAndFunctionApmResourceStatName) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableFormulaAndFunctionApmResourceStatName) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableFormulaAndFunctionApmResourceStatName initializes the struct as if Set has been called.
func NewNullableFormulaAndFunctionApmResourceStatName(val *FormulaAndFunctionApmResourceStatName) *NullableFormulaAndFunctionApmResourceStatName {
	return &NullableFormulaAndFunctionApmResourceStatName{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableFormulaAndFunctionApmResourceStatName) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableFormulaAndFunctionApmResourceStatName) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
