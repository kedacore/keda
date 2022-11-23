// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// FormulaAndFunctionMetricDataSource Data source for metrics queries.
type FormulaAndFunctionMetricDataSource string

// List of FormulaAndFunctionMetricDataSource.
const (
	FORMULAANDFUNCTIONMETRICDATASOURCE_METRICS FormulaAndFunctionMetricDataSource = "metrics"
)

var allowedFormulaAndFunctionMetricDataSourceEnumValues = []FormulaAndFunctionMetricDataSource{
	FORMULAANDFUNCTIONMETRICDATASOURCE_METRICS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *FormulaAndFunctionMetricDataSource) GetAllowedValues() []FormulaAndFunctionMetricDataSource {
	return allowedFormulaAndFunctionMetricDataSourceEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *FormulaAndFunctionMetricDataSource) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = FormulaAndFunctionMetricDataSource(value)
	return nil
}

// NewFormulaAndFunctionMetricDataSourceFromValue returns a pointer to a valid FormulaAndFunctionMetricDataSource
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewFormulaAndFunctionMetricDataSourceFromValue(v string) (*FormulaAndFunctionMetricDataSource, error) {
	ev := FormulaAndFunctionMetricDataSource(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for FormulaAndFunctionMetricDataSource: valid values are %v", v, allowedFormulaAndFunctionMetricDataSourceEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v FormulaAndFunctionMetricDataSource) IsValid() bool {
	for _, existing := range allowedFormulaAndFunctionMetricDataSourceEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to FormulaAndFunctionMetricDataSource value.
func (v FormulaAndFunctionMetricDataSource) Ptr() *FormulaAndFunctionMetricDataSource {
	return &v
}

// NullableFormulaAndFunctionMetricDataSource handles when a null is used for FormulaAndFunctionMetricDataSource.
type NullableFormulaAndFunctionMetricDataSource struct {
	value *FormulaAndFunctionMetricDataSource
	isSet bool
}

// Get returns the associated value.
func (v NullableFormulaAndFunctionMetricDataSource) Get() *FormulaAndFunctionMetricDataSource {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableFormulaAndFunctionMetricDataSource) Set(val *FormulaAndFunctionMetricDataSource) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableFormulaAndFunctionMetricDataSource) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableFormulaAndFunctionMetricDataSource) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableFormulaAndFunctionMetricDataSource initializes the struct as if Set has been called.
func NewNullableFormulaAndFunctionMetricDataSource(val *FormulaAndFunctionMetricDataSource) *NullableFormulaAndFunctionMetricDataSource {
	return &NullableFormulaAndFunctionMetricDataSource{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableFormulaAndFunctionMetricDataSource) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableFormulaAndFunctionMetricDataSource) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
