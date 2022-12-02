// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ContentEncoding HTTP header used to compress the media-type.
type ContentEncoding string

// List of ContentEncoding.
const (
	CONTENTENCODING_GZIP    ContentEncoding = "gzip"
	CONTENTENCODING_DEFLATE ContentEncoding = "deflate"
)

var allowedContentEncodingEnumValues = []ContentEncoding{
	CONTENTENCODING_GZIP,
	CONTENTENCODING_DEFLATE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *ContentEncoding) GetAllowedValues() []ContentEncoding {
	return allowedContentEncodingEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *ContentEncoding) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = ContentEncoding(value)
	return nil
}

// NewContentEncodingFromValue returns a pointer to a valid ContentEncoding
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewContentEncodingFromValue(v string) (*ContentEncoding, error) {
	ev := ContentEncoding(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for ContentEncoding: valid values are %v", v, allowedContentEncodingEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v ContentEncoding) IsValid() bool {
	for _, existing := range allowedContentEncodingEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to ContentEncoding value.
func (v ContentEncoding) Ptr() *ContentEncoding {
	return &v
}

// NullableContentEncoding handles when a null is used for ContentEncoding.
type NullableContentEncoding struct {
	value *ContentEncoding
	isSet bool
}

// Get returns the associated value.
func (v NullableContentEncoding) Get() *ContentEncoding {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableContentEncoding) Set(val *ContentEncoding) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableContentEncoding) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableContentEncoding) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableContentEncoding initializes the struct as if Set has been called.
func NewNullableContentEncoding(val *ContentEncoding) *NullableContentEncoding {
	return &NullableContentEncoding{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableContentEncoding) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableContentEncoding) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
