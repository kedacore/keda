// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetTickEdge Define how you want to align the text on the widget.
type WidgetTickEdge string

// List of WidgetTickEdge.
const (
	WIDGETTICKEDGE_BOTTOM WidgetTickEdge = "bottom"
	WIDGETTICKEDGE_LEFT   WidgetTickEdge = "left"
	WIDGETTICKEDGE_RIGHT  WidgetTickEdge = "right"
	WIDGETTICKEDGE_TOP    WidgetTickEdge = "top"
)

var allowedWidgetTickEdgeEnumValues = []WidgetTickEdge{
	WIDGETTICKEDGE_BOTTOM,
	WIDGETTICKEDGE_LEFT,
	WIDGETTICKEDGE_RIGHT,
	WIDGETTICKEDGE_TOP,
}

// GetAllowedValues reeturns the list of possible values.
func (v *WidgetTickEdge) GetAllowedValues() []WidgetTickEdge {
	return allowedWidgetTickEdgeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *WidgetTickEdge) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = WidgetTickEdge(value)
	return nil
}

// NewWidgetTickEdgeFromValue returns a pointer to a valid WidgetTickEdge
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewWidgetTickEdgeFromValue(v string) (*WidgetTickEdge, error) {
	ev := WidgetTickEdge(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for WidgetTickEdge: valid values are %v", v, allowedWidgetTickEdgeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v WidgetTickEdge) IsValid() bool {
	for _, existing := range allowedWidgetTickEdgeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to WidgetTickEdge value.
func (v WidgetTickEdge) Ptr() *WidgetTickEdge {
	return &v
}

// NullableWidgetTickEdge handles when a null is used for WidgetTickEdge.
type NullableWidgetTickEdge struct {
	value *WidgetTickEdge
	isSet bool
}

// Get returns the associated value.
func (v NullableWidgetTickEdge) Get() *WidgetTickEdge {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableWidgetTickEdge) Set(val *WidgetTickEdge) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableWidgetTickEdge) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableWidgetTickEdge) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableWidgetTickEdge initializes the struct as if Set has been called.
func NewNullableWidgetTickEdge(val *WidgetTickEdge) *NullableWidgetTickEdge {
	return &NullableWidgetTickEdge{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableWidgetTickEdge) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableWidgetTickEdge) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
