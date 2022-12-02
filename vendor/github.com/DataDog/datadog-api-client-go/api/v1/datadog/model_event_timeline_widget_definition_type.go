// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// EventTimelineWidgetDefinitionType Type of the event timeline widget.
type EventTimelineWidgetDefinitionType string

// List of EventTimelineWidgetDefinitionType.
const (
	EVENTTIMELINEWIDGETDEFINITIONTYPE_EVENT_TIMELINE EventTimelineWidgetDefinitionType = "event_timeline"
)

var allowedEventTimelineWidgetDefinitionTypeEnumValues = []EventTimelineWidgetDefinitionType{
	EVENTTIMELINEWIDGETDEFINITIONTYPE_EVENT_TIMELINE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *EventTimelineWidgetDefinitionType) GetAllowedValues() []EventTimelineWidgetDefinitionType {
	return allowedEventTimelineWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *EventTimelineWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = EventTimelineWidgetDefinitionType(value)
	return nil
}

// NewEventTimelineWidgetDefinitionTypeFromValue returns a pointer to a valid EventTimelineWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewEventTimelineWidgetDefinitionTypeFromValue(v string) (*EventTimelineWidgetDefinitionType, error) {
	ev := EventTimelineWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for EventTimelineWidgetDefinitionType: valid values are %v", v, allowedEventTimelineWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v EventTimelineWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedEventTimelineWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to EventTimelineWidgetDefinitionType value.
func (v EventTimelineWidgetDefinitionType) Ptr() *EventTimelineWidgetDefinitionType {
	return &v
}

// NullableEventTimelineWidgetDefinitionType handles when a null is used for EventTimelineWidgetDefinitionType.
type NullableEventTimelineWidgetDefinitionType struct {
	value *EventTimelineWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableEventTimelineWidgetDefinitionType) Get() *EventTimelineWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableEventTimelineWidgetDefinitionType) Set(val *EventTimelineWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableEventTimelineWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableEventTimelineWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableEventTimelineWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableEventTimelineWidgetDefinitionType(val *EventTimelineWidgetDefinitionType) *NullableEventTimelineWidgetDefinitionType {
	return &NullableEventTimelineWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableEventTimelineWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableEventTimelineWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
