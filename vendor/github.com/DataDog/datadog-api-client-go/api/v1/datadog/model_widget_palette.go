// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetPalette Color palette to apply.
type WidgetPalette string

// List of WidgetPalette.
const (
	WIDGETPALETTE_BLUE                  WidgetPalette = "blue"
	WIDGETPALETTE_CUSTOM_BACKGROUND     WidgetPalette = "custom_bg"
	WIDGETPALETTE_CUSTOM_IMAGE          WidgetPalette = "custom_image"
	WIDGETPALETTE_CUSTOM_TEXT           WidgetPalette = "custom_text"
	WIDGETPALETTE_GRAY_ON_WHITE         WidgetPalette = "gray_on_white"
	WIDGETPALETTE_GREY                  WidgetPalette = "grey"
	WIDGETPALETTE_GREEN                 WidgetPalette = "green"
	WIDGETPALETTE_ORANGE                WidgetPalette = "orange"
	WIDGETPALETTE_RED                   WidgetPalette = "red"
	WIDGETPALETTE_RED_ON_WHITE          WidgetPalette = "red_on_white"
	WIDGETPALETTE_WHITE_ON_GRAY         WidgetPalette = "white_on_gray"
	WIDGETPALETTE_WHITE_ON_GREEN        WidgetPalette = "white_on_green"
	WIDGETPALETTE_GREEN_ON_WHITE        WidgetPalette = "green_on_white"
	WIDGETPALETTE_WHITE_ON_RED          WidgetPalette = "white_on_red"
	WIDGETPALETTE_WHITE_ON_YELLOW       WidgetPalette = "white_on_yellow"
	WIDGETPALETTE_YELLOW_ON_WHITE       WidgetPalette = "yellow_on_white"
	WIDGETPALETTE_BLACK_ON_LIGHT_YELLOW WidgetPalette = "black_on_light_yellow"
	WIDGETPALETTE_BLACK_ON_LIGHT_GREEN  WidgetPalette = "black_on_light_green"
	WIDGETPALETTE_BLACK_ON_LIGHT_RED    WidgetPalette = "black_on_light_red"
)

var allowedWidgetPaletteEnumValues = []WidgetPalette{
	WIDGETPALETTE_BLUE,
	WIDGETPALETTE_CUSTOM_BACKGROUND,
	WIDGETPALETTE_CUSTOM_IMAGE,
	WIDGETPALETTE_CUSTOM_TEXT,
	WIDGETPALETTE_GRAY_ON_WHITE,
	WIDGETPALETTE_GREY,
	WIDGETPALETTE_GREEN,
	WIDGETPALETTE_ORANGE,
	WIDGETPALETTE_RED,
	WIDGETPALETTE_RED_ON_WHITE,
	WIDGETPALETTE_WHITE_ON_GRAY,
	WIDGETPALETTE_WHITE_ON_GREEN,
	WIDGETPALETTE_GREEN_ON_WHITE,
	WIDGETPALETTE_WHITE_ON_RED,
	WIDGETPALETTE_WHITE_ON_YELLOW,
	WIDGETPALETTE_YELLOW_ON_WHITE,
	WIDGETPALETTE_BLACK_ON_LIGHT_YELLOW,
	WIDGETPALETTE_BLACK_ON_LIGHT_GREEN,
	WIDGETPALETTE_BLACK_ON_LIGHT_RED,
}

// GetAllowedValues reeturns the list of possible values.
func (v *WidgetPalette) GetAllowedValues() []WidgetPalette {
	return allowedWidgetPaletteEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *WidgetPalette) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = WidgetPalette(value)
	return nil
}

// NewWidgetPaletteFromValue returns a pointer to a valid WidgetPalette
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewWidgetPaletteFromValue(v string) (*WidgetPalette, error) {
	ev := WidgetPalette(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for WidgetPalette: valid values are %v", v, allowedWidgetPaletteEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v WidgetPalette) IsValid() bool {
	for _, existing := range allowedWidgetPaletteEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to WidgetPalette value.
func (v WidgetPalette) Ptr() *WidgetPalette {
	return &v
}

// NullableWidgetPalette handles when a null is used for WidgetPalette.
type NullableWidgetPalette struct {
	value *WidgetPalette
	isSet bool
}

// Get returns the associated value.
func (v NullableWidgetPalette) Get() *WidgetPalette {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableWidgetPalette) Set(val *WidgetPalette) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableWidgetPalette) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableWidgetPalette) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableWidgetPalette initializes the struct as if Set has been called.
func NewNullableWidgetPalette(val *WidgetPalette) *NullableWidgetPalette {
	return &NullableWidgetPalette{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableWidgetPalette) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableWidgetPalette) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
