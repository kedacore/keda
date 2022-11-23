// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ListStreamSource Source from which to query items to display in the stream.
type ListStreamSource string

// List of ListStreamSource.
const (
	LISTSTREAMSOURCE_LOGS_STREAM      ListStreamSource = "logs_stream"
	LISTSTREAMSOURCE_AUDIT_STREAM     ListStreamSource = "audit_stream"
	LISTSTREAMSOURCE_RUM_ISSUE_STREAM ListStreamSource = "rum_issue_stream"
	LISTSTREAMSOURCE_APM_ISSUE_STREAM ListStreamSource = "apm_issue_stream"
)

var allowedListStreamSourceEnumValues = []ListStreamSource{
	LISTSTREAMSOURCE_LOGS_STREAM,
	LISTSTREAMSOURCE_AUDIT_STREAM,
	LISTSTREAMSOURCE_RUM_ISSUE_STREAM,
	LISTSTREAMSOURCE_APM_ISSUE_STREAM,
}

// GetAllowedValues reeturns the list of possible values.
func (v *ListStreamSource) GetAllowedValues() []ListStreamSource {
	return allowedListStreamSourceEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *ListStreamSource) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = ListStreamSource(value)
	return nil
}

// NewListStreamSourceFromValue returns a pointer to a valid ListStreamSource
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewListStreamSourceFromValue(v string) (*ListStreamSource, error) {
	ev := ListStreamSource(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for ListStreamSource: valid values are %v", v, allowedListStreamSourceEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v ListStreamSource) IsValid() bool {
	for _, existing := range allowedListStreamSourceEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to ListStreamSource value.
func (v ListStreamSource) Ptr() *ListStreamSource {
	return &v
}

// NullableListStreamSource handles when a null is used for ListStreamSource.
type NullableListStreamSource struct {
	value *ListStreamSource
	isSet bool
}

// Get returns the associated value.
func (v NullableListStreamSource) Get() *ListStreamSource {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableListStreamSource) Set(val *ListStreamSource) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableListStreamSource) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableListStreamSource) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableListStreamSource initializes the struct as if Set has been called.
func NewNullableListStreamSource(val *ListStreamSource) *NullableListStreamSource {
	return &NullableListStreamSource{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableListStreamSource) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableListStreamSource) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
