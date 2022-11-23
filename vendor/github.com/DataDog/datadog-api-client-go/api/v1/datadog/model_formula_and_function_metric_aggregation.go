// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// FormulaAndFunctionMetricAggregation The aggregation methods available for metrics queries.
type FormulaAndFunctionMetricAggregation string

// List of FormulaAndFunctionMetricAggregation.
const (
	FORMULAANDFUNCTIONMETRICAGGREGATION_AVG        FormulaAndFunctionMetricAggregation = "avg"
	FORMULAANDFUNCTIONMETRICAGGREGATION_MIN        FormulaAndFunctionMetricAggregation = "min"
	FORMULAANDFUNCTIONMETRICAGGREGATION_MAX        FormulaAndFunctionMetricAggregation = "max"
	FORMULAANDFUNCTIONMETRICAGGREGATION_SUM        FormulaAndFunctionMetricAggregation = "sum"
	FORMULAANDFUNCTIONMETRICAGGREGATION_LAST       FormulaAndFunctionMetricAggregation = "last"
	FORMULAANDFUNCTIONMETRICAGGREGATION_AREA       FormulaAndFunctionMetricAggregation = "area"
	FORMULAANDFUNCTIONMETRICAGGREGATION_L2NORM     FormulaAndFunctionMetricAggregation = "l2norm"
	FORMULAANDFUNCTIONMETRICAGGREGATION_PERCENTILE FormulaAndFunctionMetricAggregation = "percentile"
)

var allowedFormulaAndFunctionMetricAggregationEnumValues = []FormulaAndFunctionMetricAggregation{
	FORMULAANDFUNCTIONMETRICAGGREGATION_AVG,
	FORMULAANDFUNCTIONMETRICAGGREGATION_MIN,
	FORMULAANDFUNCTIONMETRICAGGREGATION_MAX,
	FORMULAANDFUNCTIONMETRICAGGREGATION_SUM,
	FORMULAANDFUNCTIONMETRICAGGREGATION_LAST,
	FORMULAANDFUNCTIONMETRICAGGREGATION_AREA,
	FORMULAANDFUNCTIONMETRICAGGREGATION_L2NORM,
	FORMULAANDFUNCTIONMETRICAGGREGATION_PERCENTILE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *FormulaAndFunctionMetricAggregation) GetAllowedValues() []FormulaAndFunctionMetricAggregation {
	return allowedFormulaAndFunctionMetricAggregationEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *FormulaAndFunctionMetricAggregation) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = FormulaAndFunctionMetricAggregation(value)
	return nil
}

// NewFormulaAndFunctionMetricAggregationFromValue returns a pointer to a valid FormulaAndFunctionMetricAggregation
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewFormulaAndFunctionMetricAggregationFromValue(v string) (*FormulaAndFunctionMetricAggregation, error) {
	ev := FormulaAndFunctionMetricAggregation(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for FormulaAndFunctionMetricAggregation: valid values are %v", v, allowedFormulaAndFunctionMetricAggregationEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v FormulaAndFunctionMetricAggregation) IsValid() bool {
	for _, existing := range allowedFormulaAndFunctionMetricAggregationEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to FormulaAndFunctionMetricAggregation value.
func (v FormulaAndFunctionMetricAggregation) Ptr() *FormulaAndFunctionMetricAggregation {
	return &v
}

// NullableFormulaAndFunctionMetricAggregation handles when a null is used for FormulaAndFunctionMetricAggregation.
type NullableFormulaAndFunctionMetricAggregation struct {
	value *FormulaAndFunctionMetricAggregation
	isSet bool
}

// Get returns the associated value.
func (v NullableFormulaAndFunctionMetricAggregation) Get() *FormulaAndFunctionMetricAggregation {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableFormulaAndFunctionMetricAggregation) Set(val *FormulaAndFunctionMetricAggregation) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableFormulaAndFunctionMetricAggregation) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableFormulaAndFunctionMetricAggregation) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableFormulaAndFunctionMetricAggregation initializes the struct as if Set has been called.
func NewNullableFormulaAndFunctionMetricAggregation(val *FormulaAndFunctionMetricAggregation) *NullableFormulaAndFunctionMetricAggregation {
	return &NullableFormulaAndFunctionMetricAggregation{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableFormulaAndFunctionMetricAggregation) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableFormulaAndFunctionMetricAggregation) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
