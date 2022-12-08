// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// HTTPMethod The HTTP method.
type HTTPMethod string

// List of HTTPMethod.
const (
	HTTPMETHOD_GET     HTTPMethod = "GET"
	HTTPMETHOD_POST    HTTPMethod = "POST"
	HTTPMETHOD_PATCH   HTTPMethod = "PATCH"
	HTTPMETHOD_PUT     HTTPMethod = "PUT"
	HTTPMETHOD_DELETE  HTTPMethod = "DELETE"
	HTTPMETHOD_HEAD    HTTPMethod = "HEAD"
	HTTPMETHOD_OPTIONS HTTPMethod = "OPTIONS"
)

var allowedHTTPMethodEnumValues = []HTTPMethod{
	HTTPMETHOD_GET,
	HTTPMETHOD_POST,
	HTTPMETHOD_PATCH,
	HTTPMETHOD_PUT,
	HTTPMETHOD_DELETE,
	HTTPMETHOD_HEAD,
	HTTPMETHOD_OPTIONS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *HTTPMethod) GetAllowedValues() []HTTPMethod {
	return allowedHTTPMethodEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *HTTPMethod) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = HTTPMethod(value)
	return nil
}

// NewHTTPMethodFromValue returns a pointer to a valid HTTPMethod
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewHTTPMethodFromValue(v string) (*HTTPMethod, error) {
	ev := HTTPMethod(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for HTTPMethod: valid values are %v", v, allowedHTTPMethodEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v HTTPMethod) IsValid() bool {
	for _, existing := range allowedHTTPMethodEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to HTTPMethod value.
func (v HTTPMethod) Ptr() *HTTPMethod {
	return &v
}

// NullableHTTPMethod handles when a null is used for HTTPMethod.
type NullableHTTPMethod struct {
	value *HTTPMethod
	isSet bool
}

// Get returns the associated value.
func (v NullableHTTPMethod) Get() *HTTPMethod {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableHTTPMethod) Set(val *HTTPMethod) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableHTTPMethod) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableHTTPMethod) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableHTTPMethod initializes the struct as if Set has been called.
func NewNullableHTTPMethod(val *HTTPMethod) *NullableHTTPMethod {
	return &NullableHTTPMethod{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableHTTPMethod) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableHTTPMethod) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
