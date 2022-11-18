// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// IFrameWidgetDefinitionType Type of the iframe widget.
type IFrameWidgetDefinitionType string

// List of IFrameWidgetDefinitionType.
const (
	IFRAMEWIDGETDEFINITIONTYPE_IFRAME IFrameWidgetDefinitionType = "iframe"
)

var allowedIFrameWidgetDefinitionTypeEnumValues = []IFrameWidgetDefinitionType{
	IFRAMEWIDGETDEFINITIONTYPE_IFRAME,
}

// GetAllowedValues reeturns the list of possible values.
func (v *IFrameWidgetDefinitionType) GetAllowedValues() []IFrameWidgetDefinitionType {
	return allowedIFrameWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *IFrameWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = IFrameWidgetDefinitionType(value)
	return nil
}

// NewIFrameWidgetDefinitionTypeFromValue returns a pointer to a valid IFrameWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewIFrameWidgetDefinitionTypeFromValue(v string) (*IFrameWidgetDefinitionType, error) {
	ev := IFrameWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for IFrameWidgetDefinitionType: valid values are %v", v, allowedIFrameWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v IFrameWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedIFrameWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to IFrameWidgetDefinitionType value.
func (v IFrameWidgetDefinitionType) Ptr() *IFrameWidgetDefinitionType {
	return &v
}

// NullableIFrameWidgetDefinitionType handles when a null is used for IFrameWidgetDefinitionType.
type NullableIFrameWidgetDefinitionType struct {
	value *IFrameWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableIFrameWidgetDefinitionType) Get() *IFrameWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableIFrameWidgetDefinitionType) Set(val *IFrameWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableIFrameWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableIFrameWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableIFrameWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableIFrameWidgetDefinitionType(val *IFrameWidgetDefinitionType) *NullableIFrameWidgetDefinitionType {
	return &NullableIFrameWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableIFrameWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableIFrameWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
