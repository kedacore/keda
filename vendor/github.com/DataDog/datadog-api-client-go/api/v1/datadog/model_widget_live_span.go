// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetLiveSpan The available timeframes depend on the widget you are using.
type WidgetLiveSpan string

// List of WidgetLiveSpan.
const (
	WIDGETLIVESPAN_PAST_ONE_MINUTE      WidgetLiveSpan = "1m"
	WIDGETLIVESPAN_PAST_FIVE_MINUTES    WidgetLiveSpan = "5m"
	WIDGETLIVESPAN_PAST_TEN_MINUTES     WidgetLiveSpan = "10m"
	WIDGETLIVESPAN_PAST_FIFTEEN_MINUTES WidgetLiveSpan = "15m"
	WIDGETLIVESPAN_PAST_THIRTY_MINUTES  WidgetLiveSpan = "30m"
	WIDGETLIVESPAN_PAST_ONE_HOUR        WidgetLiveSpan = "1h"
	WIDGETLIVESPAN_PAST_FOUR_HOURS      WidgetLiveSpan = "4h"
	WIDGETLIVESPAN_PAST_ONE_DAY         WidgetLiveSpan = "1d"
	WIDGETLIVESPAN_PAST_TWO_DAYS        WidgetLiveSpan = "2d"
	WIDGETLIVESPAN_PAST_ONE_WEEK        WidgetLiveSpan = "1w"
	WIDGETLIVESPAN_PAST_ONE_MONTH       WidgetLiveSpan = "1mo"
	WIDGETLIVESPAN_PAST_THREE_MONTHS    WidgetLiveSpan = "3mo"
	WIDGETLIVESPAN_PAST_SIX_MONTHS      WidgetLiveSpan = "6mo"
	WIDGETLIVESPAN_PAST_ONE_YEAR        WidgetLiveSpan = "1y"
	WIDGETLIVESPAN_ALERT                WidgetLiveSpan = "alert"
)

var allowedWidgetLiveSpanEnumValues = []WidgetLiveSpan{
	WIDGETLIVESPAN_PAST_ONE_MINUTE,
	WIDGETLIVESPAN_PAST_FIVE_MINUTES,
	WIDGETLIVESPAN_PAST_TEN_MINUTES,
	WIDGETLIVESPAN_PAST_FIFTEEN_MINUTES,
	WIDGETLIVESPAN_PAST_THIRTY_MINUTES,
	WIDGETLIVESPAN_PAST_ONE_HOUR,
	WIDGETLIVESPAN_PAST_FOUR_HOURS,
	WIDGETLIVESPAN_PAST_ONE_DAY,
	WIDGETLIVESPAN_PAST_TWO_DAYS,
	WIDGETLIVESPAN_PAST_ONE_WEEK,
	WIDGETLIVESPAN_PAST_ONE_MONTH,
	WIDGETLIVESPAN_PAST_THREE_MONTHS,
	WIDGETLIVESPAN_PAST_SIX_MONTHS,
	WIDGETLIVESPAN_PAST_ONE_YEAR,
	WIDGETLIVESPAN_ALERT,
}

// GetAllowedValues reeturns the list of possible values.
func (v *WidgetLiveSpan) GetAllowedValues() []WidgetLiveSpan {
	return allowedWidgetLiveSpanEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *WidgetLiveSpan) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = WidgetLiveSpan(value)
	return nil
}

// NewWidgetLiveSpanFromValue returns a pointer to a valid WidgetLiveSpan
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewWidgetLiveSpanFromValue(v string) (*WidgetLiveSpan, error) {
	ev := WidgetLiveSpan(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for WidgetLiveSpan: valid values are %v", v, allowedWidgetLiveSpanEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v WidgetLiveSpan) IsValid() bool {
	for _, existing := range allowedWidgetLiveSpanEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to WidgetLiveSpan value.
func (v WidgetLiveSpan) Ptr() *WidgetLiveSpan {
	return &v
}

// NullableWidgetLiveSpan handles when a null is used for WidgetLiveSpan.
type NullableWidgetLiveSpan struct {
	value *WidgetLiveSpan
	isSet bool
}

// Get returns the associated value.
func (v NullableWidgetLiveSpan) Get() *WidgetLiveSpan {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableWidgetLiveSpan) Set(val *WidgetLiveSpan) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableWidgetLiveSpan) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableWidgetLiveSpan) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableWidgetLiveSpan initializes the struct as if Set has been called.
func NewNullableWidgetLiveSpan(val *WidgetLiveSpan) *NullableWidgetLiveSpan {
	return &NullableWidgetLiveSpan{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableWidgetLiveSpan) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableWidgetLiveSpan) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
