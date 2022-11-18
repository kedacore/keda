// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// TreeMapGroupBy (deprecated) The attribute formerly used to group elements in the widget.
type TreeMapGroupBy string

// List of TreeMapGroupBy.
const (
	TREEMAPGROUPBY_USER    TreeMapGroupBy = "user"
	TREEMAPGROUPBY_FAMILY  TreeMapGroupBy = "family"
	TREEMAPGROUPBY_PROCESS TreeMapGroupBy = "process"
)

var allowedTreeMapGroupByEnumValues = []TreeMapGroupBy{
	TREEMAPGROUPBY_USER,
	TREEMAPGROUPBY_FAMILY,
	TREEMAPGROUPBY_PROCESS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *TreeMapGroupBy) GetAllowedValues() []TreeMapGroupBy {
	return allowedTreeMapGroupByEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *TreeMapGroupBy) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = TreeMapGroupBy(value)
	return nil
}

// NewTreeMapGroupByFromValue returns a pointer to a valid TreeMapGroupBy
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewTreeMapGroupByFromValue(v string) (*TreeMapGroupBy, error) {
	ev := TreeMapGroupBy(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for TreeMapGroupBy: valid values are %v", v, allowedTreeMapGroupByEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v TreeMapGroupBy) IsValid() bool {
	for _, existing := range allowedTreeMapGroupByEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to TreeMapGroupBy value.
func (v TreeMapGroupBy) Ptr() *TreeMapGroupBy {
	return &v
}

// NullableTreeMapGroupBy handles when a null is used for TreeMapGroupBy.
type NullableTreeMapGroupBy struct {
	value *TreeMapGroupBy
	isSet bool
}

// Get returns the associated value.
func (v NullableTreeMapGroupBy) Get() *TreeMapGroupBy {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableTreeMapGroupBy) Set(val *TreeMapGroupBy) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableTreeMapGroupBy) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableTreeMapGroupBy) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableTreeMapGroupBy initializes the struct as if Set has been called.
func NewNullableTreeMapGroupBy(val *TreeMapGroupBy) *NullableTreeMapGroupBy {
	return &NullableTreeMapGroupBy{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableTreeMapGroupBy) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableTreeMapGroupBy) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
