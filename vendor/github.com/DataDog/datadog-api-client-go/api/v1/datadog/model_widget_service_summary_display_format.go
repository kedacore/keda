// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetServiceSummaryDisplayFormat Number of columns to display.
type WidgetServiceSummaryDisplayFormat string

// List of WidgetServiceSummaryDisplayFormat.
const (
	WIDGETSERVICESUMMARYDISPLAYFORMAT_ONE_COLUMN   WidgetServiceSummaryDisplayFormat = "one_column"
	WIDGETSERVICESUMMARYDISPLAYFORMAT_TWO_COLUMN   WidgetServiceSummaryDisplayFormat = "two_column"
	WIDGETSERVICESUMMARYDISPLAYFORMAT_THREE_COLUMN WidgetServiceSummaryDisplayFormat = "three_column"
)

var allowedWidgetServiceSummaryDisplayFormatEnumValues = []WidgetServiceSummaryDisplayFormat{
	WIDGETSERVICESUMMARYDISPLAYFORMAT_ONE_COLUMN,
	WIDGETSERVICESUMMARYDISPLAYFORMAT_TWO_COLUMN,
	WIDGETSERVICESUMMARYDISPLAYFORMAT_THREE_COLUMN,
}

// GetAllowedValues reeturns the list of possible values.
func (v *WidgetServiceSummaryDisplayFormat) GetAllowedValues() []WidgetServiceSummaryDisplayFormat {
	return allowedWidgetServiceSummaryDisplayFormatEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *WidgetServiceSummaryDisplayFormat) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = WidgetServiceSummaryDisplayFormat(value)
	return nil
}

// NewWidgetServiceSummaryDisplayFormatFromValue returns a pointer to a valid WidgetServiceSummaryDisplayFormat
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewWidgetServiceSummaryDisplayFormatFromValue(v string) (*WidgetServiceSummaryDisplayFormat, error) {
	ev := WidgetServiceSummaryDisplayFormat(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for WidgetServiceSummaryDisplayFormat: valid values are %v", v, allowedWidgetServiceSummaryDisplayFormatEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v WidgetServiceSummaryDisplayFormat) IsValid() bool {
	for _, existing := range allowedWidgetServiceSummaryDisplayFormatEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to WidgetServiceSummaryDisplayFormat value.
func (v WidgetServiceSummaryDisplayFormat) Ptr() *WidgetServiceSummaryDisplayFormat {
	return &v
}

// NullableWidgetServiceSummaryDisplayFormat handles when a null is used for WidgetServiceSummaryDisplayFormat.
type NullableWidgetServiceSummaryDisplayFormat struct {
	value *WidgetServiceSummaryDisplayFormat
	isSet bool
}

// Get returns the associated value.
func (v NullableWidgetServiceSummaryDisplayFormat) Get() *WidgetServiceSummaryDisplayFormat {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableWidgetServiceSummaryDisplayFormat) Set(val *WidgetServiceSummaryDisplayFormat) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableWidgetServiceSummaryDisplayFormat) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableWidgetServiceSummaryDisplayFormat) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableWidgetServiceSummaryDisplayFormat initializes the struct as if Set has been called.
func NewNullableWidgetServiceSummaryDisplayFormat(val *WidgetServiceSummaryDisplayFormat) *NullableWidgetServiceSummaryDisplayFormat {
	return &NullableWidgetServiceSummaryDisplayFormat{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableWidgetServiceSummaryDisplayFormat) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableWidgetServiceSummaryDisplayFormat) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
