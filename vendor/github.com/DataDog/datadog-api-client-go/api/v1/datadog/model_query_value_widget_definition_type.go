// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// QueryValueWidgetDefinitionType Type of the query value widget.
type QueryValueWidgetDefinitionType string

// List of QueryValueWidgetDefinitionType.
const (
	QUERYVALUEWIDGETDEFINITIONTYPE_QUERY_VALUE QueryValueWidgetDefinitionType = "query_value"
)

var allowedQueryValueWidgetDefinitionTypeEnumValues = []QueryValueWidgetDefinitionType{
	QUERYVALUEWIDGETDEFINITIONTYPE_QUERY_VALUE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *QueryValueWidgetDefinitionType) GetAllowedValues() []QueryValueWidgetDefinitionType {
	return allowedQueryValueWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *QueryValueWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = QueryValueWidgetDefinitionType(value)
	return nil
}

// NewQueryValueWidgetDefinitionTypeFromValue returns a pointer to a valid QueryValueWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewQueryValueWidgetDefinitionTypeFromValue(v string) (*QueryValueWidgetDefinitionType, error) {
	ev := QueryValueWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for QueryValueWidgetDefinitionType: valid values are %v", v, allowedQueryValueWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v QueryValueWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedQueryValueWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to QueryValueWidgetDefinitionType value.
func (v QueryValueWidgetDefinitionType) Ptr() *QueryValueWidgetDefinitionType {
	return &v
}

// NullableQueryValueWidgetDefinitionType handles when a null is used for QueryValueWidgetDefinitionType.
type NullableQueryValueWidgetDefinitionType struct {
	value *QueryValueWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableQueryValueWidgetDefinitionType) Get() *QueryValueWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableQueryValueWidgetDefinitionType) Set(val *QueryValueWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableQueryValueWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableQueryValueWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableQueryValueWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableQueryValueWidgetDefinitionType(val *QueryValueWidgetDefinitionType) *NullableQueryValueWidgetDefinitionType {
	return &NullableQueryValueWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableQueryValueWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableQueryValueWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
