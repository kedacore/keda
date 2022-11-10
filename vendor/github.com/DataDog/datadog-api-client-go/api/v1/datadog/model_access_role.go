// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// AccessRole The access role of the user. Options are **st** (standard user), **adm** (admin user), or **ro** (read-only user).
type AccessRole string

// List of AccessRole.
const (
	ACCESSROLE_STANDARD  AccessRole = "st"
	ACCESSROLE_ADMIN     AccessRole = "adm"
	ACCESSROLE_READ_ONLY AccessRole = "ro"
	ACCESSROLE_ERROR     AccessRole = "ERROR"
)

var allowedAccessRoleEnumValues = []AccessRole{
	ACCESSROLE_STANDARD,
	ACCESSROLE_ADMIN,
	ACCESSROLE_READ_ONLY,
	ACCESSROLE_ERROR,
}

// GetAllowedValues reeturns the list of possible values.
func (v *AccessRole) GetAllowedValues() []AccessRole {
	return allowedAccessRoleEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *AccessRole) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = AccessRole(value)
	return nil
}

// NewAccessRoleFromValue returns a pointer to a valid AccessRole
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewAccessRoleFromValue(v string) (*AccessRole, error) {
	ev := AccessRole(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for AccessRole: valid values are %v", v, allowedAccessRoleEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v AccessRole) IsValid() bool {
	for _, existing := range allowedAccessRoleEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to AccessRole value.
func (v AccessRole) Ptr() *AccessRole {
	return &v
}

// NullableAccessRole handles when a null is used for AccessRole.
type NullableAccessRole struct {
	value *AccessRole
	isSet bool
}

// Get returns the associated value.
func (v NullableAccessRole) Get() *AccessRole {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableAccessRole) Set(val *AccessRole) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableAccessRole) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableAccessRole) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableAccessRole initializes the struct as if Set has been called.
func NewNullableAccessRole(val *AccessRole) *NullableAccessRole {
	return &NullableAccessRole{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableAccessRole) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableAccessRole) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
