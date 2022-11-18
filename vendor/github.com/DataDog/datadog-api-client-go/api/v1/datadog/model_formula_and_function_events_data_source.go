// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// FormulaAndFunctionEventsDataSource Data source for event platform-based queries.
type FormulaAndFunctionEventsDataSource string

// List of FormulaAndFunctionEventsDataSource.
const (
	FORMULAANDFUNCTIONEVENTSDATASOURCE_LOGS             FormulaAndFunctionEventsDataSource = "logs"
	FORMULAANDFUNCTIONEVENTSDATASOURCE_SPANS            FormulaAndFunctionEventsDataSource = "spans"
	FORMULAANDFUNCTIONEVENTSDATASOURCE_NETWORK          FormulaAndFunctionEventsDataSource = "network"
	FORMULAANDFUNCTIONEVENTSDATASOURCE_RUM              FormulaAndFunctionEventsDataSource = "rum"
	FORMULAANDFUNCTIONEVENTSDATASOURCE_SECURITY_SIGNALS FormulaAndFunctionEventsDataSource = "security_signals"
	FORMULAANDFUNCTIONEVENTSDATASOURCE_PROFILES         FormulaAndFunctionEventsDataSource = "profiles"
	FORMULAANDFUNCTIONEVENTSDATASOURCE_AUDIT            FormulaAndFunctionEventsDataSource = "audit"
	FORMULAANDFUNCTIONEVENTSDATASOURCE_EVENTS           FormulaAndFunctionEventsDataSource = "events"
)

var allowedFormulaAndFunctionEventsDataSourceEnumValues = []FormulaAndFunctionEventsDataSource{
	FORMULAANDFUNCTIONEVENTSDATASOURCE_LOGS,
	FORMULAANDFUNCTIONEVENTSDATASOURCE_SPANS,
	FORMULAANDFUNCTIONEVENTSDATASOURCE_NETWORK,
	FORMULAANDFUNCTIONEVENTSDATASOURCE_RUM,
	FORMULAANDFUNCTIONEVENTSDATASOURCE_SECURITY_SIGNALS,
	FORMULAANDFUNCTIONEVENTSDATASOURCE_PROFILES,
	FORMULAANDFUNCTIONEVENTSDATASOURCE_AUDIT,
	FORMULAANDFUNCTIONEVENTSDATASOURCE_EVENTS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *FormulaAndFunctionEventsDataSource) GetAllowedValues() []FormulaAndFunctionEventsDataSource {
	return allowedFormulaAndFunctionEventsDataSourceEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *FormulaAndFunctionEventsDataSource) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = FormulaAndFunctionEventsDataSource(value)
	return nil
}

// NewFormulaAndFunctionEventsDataSourceFromValue returns a pointer to a valid FormulaAndFunctionEventsDataSource
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewFormulaAndFunctionEventsDataSourceFromValue(v string) (*FormulaAndFunctionEventsDataSource, error) {
	ev := FormulaAndFunctionEventsDataSource(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for FormulaAndFunctionEventsDataSource: valid values are %v", v, allowedFormulaAndFunctionEventsDataSourceEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v FormulaAndFunctionEventsDataSource) IsValid() bool {
	for _, existing := range allowedFormulaAndFunctionEventsDataSourceEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to FormulaAndFunctionEventsDataSource value.
func (v FormulaAndFunctionEventsDataSource) Ptr() *FormulaAndFunctionEventsDataSource {
	return &v
}

// NullableFormulaAndFunctionEventsDataSource handles when a null is used for FormulaAndFunctionEventsDataSource.
type NullableFormulaAndFunctionEventsDataSource struct {
	value *FormulaAndFunctionEventsDataSource
	isSet bool
}

// Get returns the associated value.
func (v NullableFormulaAndFunctionEventsDataSource) Get() *FormulaAndFunctionEventsDataSource {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableFormulaAndFunctionEventsDataSource) Set(val *FormulaAndFunctionEventsDataSource) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableFormulaAndFunctionEventsDataSource) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableFormulaAndFunctionEventsDataSource) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableFormulaAndFunctionEventsDataSource initializes the struct as if Set has been called.
func NewNullableFormulaAndFunctionEventsDataSource(val *FormulaAndFunctionEventsDataSource) *NullableFormulaAndFunctionEventsDataSource {
	return &NullableFormulaAndFunctionEventsDataSource{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableFormulaAndFunctionEventsDataSource) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableFormulaAndFunctionEventsDataSource) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
