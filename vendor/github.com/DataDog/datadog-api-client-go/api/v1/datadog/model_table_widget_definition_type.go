// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// TableWidgetDefinitionType Type of the table widget.
type TableWidgetDefinitionType string

// List of TableWidgetDefinitionType.
const (
	TABLEWIDGETDEFINITIONTYPE_QUERY_TABLE TableWidgetDefinitionType = "query_table"
)

var allowedTableWidgetDefinitionTypeEnumValues = []TableWidgetDefinitionType{
	TABLEWIDGETDEFINITIONTYPE_QUERY_TABLE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *TableWidgetDefinitionType) GetAllowedValues() []TableWidgetDefinitionType {
	return allowedTableWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *TableWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = TableWidgetDefinitionType(value)
	return nil
}

// NewTableWidgetDefinitionTypeFromValue returns a pointer to a valid TableWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewTableWidgetDefinitionTypeFromValue(v string) (*TableWidgetDefinitionType, error) {
	ev := TableWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for TableWidgetDefinitionType: valid values are %v", v, allowedTableWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v TableWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedTableWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to TableWidgetDefinitionType value.
func (v TableWidgetDefinitionType) Ptr() *TableWidgetDefinitionType {
	return &v
}

// NullableTableWidgetDefinitionType handles when a null is used for TableWidgetDefinitionType.
type NullableTableWidgetDefinitionType struct {
	value *TableWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableTableWidgetDefinitionType) Get() *TableWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableTableWidgetDefinitionType) Set(val *TableWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableTableWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableTableWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableTableWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableTableWidgetDefinitionType(val *TableWidgetDefinitionType) *NullableTableWidgetDefinitionType {
	return &NullableTableWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableTableWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableTableWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
