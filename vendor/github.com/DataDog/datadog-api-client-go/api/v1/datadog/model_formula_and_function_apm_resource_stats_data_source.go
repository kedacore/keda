// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// FormulaAndFunctionApmResourceStatsDataSource Data source for APM resource stats queries.
type FormulaAndFunctionApmResourceStatsDataSource string

// List of FormulaAndFunctionApmResourceStatsDataSource.
const (
	FORMULAANDFUNCTIONAPMRESOURCESTATSDATASOURCE_APM_RESOURCE_STATS FormulaAndFunctionApmResourceStatsDataSource = "apm_resource_stats"
)

var allowedFormulaAndFunctionApmResourceStatsDataSourceEnumValues = []FormulaAndFunctionApmResourceStatsDataSource{
	FORMULAANDFUNCTIONAPMRESOURCESTATSDATASOURCE_APM_RESOURCE_STATS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *FormulaAndFunctionApmResourceStatsDataSource) GetAllowedValues() []FormulaAndFunctionApmResourceStatsDataSource {
	return allowedFormulaAndFunctionApmResourceStatsDataSourceEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *FormulaAndFunctionApmResourceStatsDataSource) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = FormulaAndFunctionApmResourceStatsDataSource(value)
	return nil
}

// NewFormulaAndFunctionApmResourceStatsDataSourceFromValue returns a pointer to a valid FormulaAndFunctionApmResourceStatsDataSource
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewFormulaAndFunctionApmResourceStatsDataSourceFromValue(v string) (*FormulaAndFunctionApmResourceStatsDataSource, error) {
	ev := FormulaAndFunctionApmResourceStatsDataSource(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for FormulaAndFunctionApmResourceStatsDataSource: valid values are %v", v, allowedFormulaAndFunctionApmResourceStatsDataSourceEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v FormulaAndFunctionApmResourceStatsDataSource) IsValid() bool {
	for _, existing := range allowedFormulaAndFunctionApmResourceStatsDataSourceEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to FormulaAndFunctionApmResourceStatsDataSource value.
func (v FormulaAndFunctionApmResourceStatsDataSource) Ptr() *FormulaAndFunctionApmResourceStatsDataSource {
	return &v
}

// NullableFormulaAndFunctionApmResourceStatsDataSource handles when a null is used for FormulaAndFunctionApmResourceStatsDataSource.
type NullableFormulaAndFunctionApmResourceStatsDataSource struct {
	value *FormulaAndFunctionApmResourceStatsDataSource
	isSet bool
}

// Get returns the associated value.
func (v NullableFormulaAndFunctionApmResourceStatsDataSource) Get() *FormulaAndFunctionApmResourceStatsDataSource {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableFormulaAndFunctionApmResourceStatsDataSource) Set(val *FormulaAndFunctionApmResourceStatsDataSource) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableFormulaAndFunctionApmResourceStatsDataSource) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableFormulaAndFunctionApmResourceStatsDataSource) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableFormulaAndFunctionApmResourceStatsDataSource initializes the struct as if Set has been called.
func NewNullableFormulaAndFunctionApmResourceStatsDataSource(val *FormulaAndFunctionApmResourceStatsDataSource) *NullableFormulaAndFunctionApmResourceStatsDataSource {
	return &NullableFormulaAndFunctionApmResourceStatsDataSource{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableFormulaAndFunctionApmResourceStatsDataSource) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableFormulaAndFunctionApmResourceStatsDataSource) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
