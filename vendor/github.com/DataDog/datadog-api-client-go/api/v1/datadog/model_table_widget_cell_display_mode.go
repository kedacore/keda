// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// TableWidgetCellDisplayMode Define a display mode for the table cell.
type TableWidgetCellDisplayMode string

// List of TableWidgetCellDisplayMode.
const (
	TABLEWIDGETCELLDISPLAYMODE_NUMBER TableWidgetCellDisplayMode = "number"
	TABLEWIDGETCELLDISPLAYMODE_BAR    TableWidgetCellDisplayMode = "bar"
)

var allowedTableWidgetCellDisplayModeEnumValues = []TableWidgetCellDisplayMode{
	TABLEWIDGETCELLDISPLAYMODE_NUMBER,
	TABLEWIDGETCELLDISPLAYMODE_BAR,
}

// GetAllowedValues reeturns the list of possible values.
func (v *TableWidgetCellDisplayMode) GetAllowedValues() []TableWidgetCellDisplayMode {
	return allowedTableWidgetCellDisplayModeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *TableWidgetCellDisplayMode) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = TableWidgetCellDisplayMode(value)
	return nil
}

// NewTableWidgetCellDisplayModeFromValue returns a pointer to a valid TableWidgetCellDisplayMode
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewTableWidgetCellDisplayModeFromValue(v string) (*TableWidgetCellDisplayMode, error) {
	ev := TableWidgetCellDisplayMode(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for TableWidgetCellDisplayMode: valid values are %v", v, allowedTableWidgetCellDisplayModeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v TableWidgetCellDisplayMode) IsValid() bool {
	for _, existing := range allowedTableWidgetCellDisplayModeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to TableWidgetCellDisplayMode value.
func (v TableWidgetCellDisplayMode) Ptr() *TableWidgetCellDisplayMode {
	return &v
}

// NullableTableWidgetCellDisplayMode handles when a null is used for TableWidgetCellDisplayMode.
type NullableTableWidgetCellDisplayMode struct {
	value *TableWidgetCellDisplayMode
	isSet bool
}

// Get returns the associated value.
func (v NullableTableWidgetCellDisplayMode) Get() *TableWidgetCellDisplayMode {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableTableWidgetCellDisplayMode) Set(val *TableWidgetCellDisplayMode) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableTableWidgetCellDisplayMode) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableTableWidgetCellDisplayMode) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableTableWidgetCellDisplayMode initializes the struct as if Set has been called.
func NewNullableTableWidgetCellDisplayMode(val *TableWidgetCellDisplayMode) *NullableTableWidgetCellDisplayMode {
	return &NullableTableWidgetCellDisplayMode{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableTableWidgetCellDisplayMode) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableTableWidgetCellDisplayMode) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
