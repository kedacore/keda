// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// TimeseriesWidgetLegendColumn Legend column.
type TimeseriesWidgetLegendColumn string

// List of TimeseriesWidgetLegendColumn.
const (
	TIMESERIESWIDGETLEGENDCOLUMN_VALUE TimeseriesWidgetLegendColumn = "value"
	TIMESERIESWIDGETLEGENDCOLUMN_AVG   TimeseriesWidgetLegendColumn = "avg"
	TIMESERIESWIDGETLEGENDCOLUMN_SUM   TimeseriesWidgetLegendColumn = "sum"
	TIMESERIESWIDGETLEGENDCOLUMN_MIN   TimeseriesWidgetLegendColumn = "min"
	TIMESERIESWIDGETLEGENDCOLUMN_MAX   TimeseriesWidgetLegendColumn = "max"
)

var allowedTimeseriesWidgetLegendColumnEnumValues = []TimeseriesWidgetLegendColumn{
	TIMESERIESWIDGETLEGENDCOLUMN_VALUE,
	TIMESERIESWIDGETLEGENDCOLUMN_AVG,
	TIMESERIESWIDGETLEGENDCOLUMN_SUM,
	TIMESERIESWIDGETLEGENDCOLUMN_MIN,
	TIMESERIESWIDGETLEGENDCOLUMN_MAX,
}

// GetAllowedValues reeturns the list of possible values.
func (v *TimeseriesWidgetLegendColumn) GetAllowedValues() []TimeseriesWidgetLegendColumn {
	return allowedTimeseriesWidgetLegendColumnEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *TimeseriesWidgetLegendColumn) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = TimeseriesWidgetLegendColumn(value)
	return nil
}

// NewTimeseriesWidgetLegendColumnFromValue returns a pointer to a valid TimeseriesWidgetLegendColumn
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewTimeseriesWidgetLegendColumnFromValue(v string) (*TimeseriesWidgetLegendColumn, error) {
	ev := TimeseriesWidgetLegendColumn(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for TimeseriesWidgetLegendColumn: valid values are %v", v, allowedTimeseriesWidgetLegendColumnEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v TimeseriesWidgetLegendColumn) IsValid() bool {
	for _, existing := range allowedTimeseriesWidgetLegendColumnEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to TimeseriesWidgetLegendColumn value.
func (v TimeseriesWidgetLegendColumn) Ptr() *TimeseriesWidgetLegendColumn {
	return &v
}

// NullableTimeseriesWidgetLegendColumn handles when a null is used for TimeseriesWidgetLegendColumn.
type NullableTimeseriesWidgetLegendColumn struct {
	value *TimeseriesWidgetLegendColumn
	isSet bool
}

// Get returns the associated value.
func (v NullableTimeseriesWidgetLegendColumn) Get() *TimeseriesWidgetLegendColumn {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableTimeseriesWidgetLegendColumn) Set(val *TimeseriesWidgetLegendColumn) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableTimeseriesWidgetLegendColumn) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableTimeseriesWidgetLegendColumn) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableTimeseriesWidgetLegendColumn initializes the struct as if Set has been called.
func NewNullableTimeseriesWidgetLegendColumn(val *TimeseriesWidgetLegendColumn) *NullableTimeseriesWidgetLegendColumn {
	return &NullableTimeseriesWidgetLegendColumn{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableTimeseriesWidgetLegendColumn) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableTimeseriesWidgetLegendColumn) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
