// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetAggregator Aggregator used for the request.
type WidgetAggregator string

// List of WidgetAggregator.
const (
	WIDGETAGGREGATOR_AVERAGE    WidgetAggregator = "avg"
	WIDGETAGGREGATOR_LAST       WidgetAggregator = "last"
	WIDGETAGGREGATOR_MAXIMUM    WidgetAggregator = "max"
	WIDGETAGGREGATOR_MINIMUM    WidgetAggregator = "min"
	WIDGETAGGREGATOR_SUM        WidgetAggregator = "sum"
	WIDGETAGGREGATOR_PERCENTILE WidgetAggregator = "percentile"
)

var allowedWidgetAggregatorEnumValues = []WidgetAggregator{
	WIDGETAGGREGATOR_AVERAGE,
	WIDGETAGGREGATOR_LAST,
	WIDGETAGGREGATOR_MAXIMUM,
	WIDGETAGGREGATOR_MINIMUM,
	WIDGETAGGREGATOR_SUM,
	WIDGETAGGREGATOR_PERCENTILE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *WidgetAggregator) GetAllowedValues() []WidgetAggregator {
	return allowedWidgetAggregatorEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *WidgetAggregator) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = WidgetAggregator(value)
	return nil
}

// NewWidgetAggregatorFromValue returns a pointer to a valid WidgetAggregator
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewWidgetAggregatorFromValue(v string) (*WidgetAggregator, error) {
	ev := WidgetAggregator(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for WidgetAggregator: valid values are %v", v, allowedWidgetAggregatorEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v WidgetAggregator) IsValid() bool {
	for _, existing := range allowedWidgetAggregatorEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to WidgetAggregator value.
func (v WidgetAggregator) Ptr() *WidgetAggregator {
	return &v
}

// NullableWidgetAggregator handles when a null is used for WidgetAggregator.
type NullableWidgetAggregator struct {
	value *WidgetAggregator
	isSet bool
}

// Get returns the associated value.
func (v NullableWidgetAggregator) Get() *WidgetAggregator {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableWidgetAggregator) Set(val *WidgetAggregator) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableWidgetAggregator) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableWidgetAggregator) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableWidgetAggregator initializes the struct as if Set has been called.
func NewNullableWidgetAggregator(val *WidgetAggregator) *NullableWidgetAggregator {
	return &NullableWidgetAggregator{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableWidgetAggregator) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableWidgetAggregator) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
