// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// MetricContentEncoding HTTP header used to compress the media-type.
type MetricContentEncoding string

// List of MetricContentEncoding.
const (
	METRICCONTENTENCODING_DEFLATE MetricContentEncoding = "deflate"
)

var allowedMetricContentEncodingEnumValues = []MetricContentEncoding{
	METRICCONTENTENCODING_DEFLATE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *MetricContentEncoding) GetAllowedValues() []MetricContentEncoding {
	return allowedMetricContentEncodingEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *MetricContentEncoding) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = MetricContentEncoding(value)
	return nil
}

// NewMetricContentEncodingFromValue returns a pointer to a valid MetricContentEncoding
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewMetricContentEncodingFromValue(v string) (*MetricContentEncoding, error) {
	ev := MetricContentEncoding(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for MetricContentEncoding: valid values are %v", v, allowedMetricContentEncodingEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v MetricContentEncoding) IsValid() bool {
	for _, existing := range allowedMetricContentEncodingEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to MetricContentEncoding value.
func (v MetricContentEncoding) Ptr() *MetricContentEncoding {
	return &v
}

// NullableMetricContentEncoding handles when a null is used for MetricContentEncoding.
type NullableMetricContentEncoding struct {
	value *MetricContentEncoding
	isSet bool
}

// Get returns the associated value.
func (v NullableMetricContentEncoding) Get() *MetricContentEncoding {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMetricContentEncoding) Set(val *MetricContentEncoding) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMetricContentEncoding) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableMetricContentEncoding) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMetricContentEncoding initializes the struct as if Set has been called.
func NewNullableMetricContentEncoding(val *MetricContentEncoding) *NullableMetricContentEncoding {
	return &NullableMetricContentEncoding{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMetricContentEncoding) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMetricContentEncoding) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
