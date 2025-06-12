/*
Copyright 2023 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

// CompareConditions checks if two slices of Conditions are semantically equivalent.
// It compares Type, Status, Reason, and Message of each condition.
// The order of conditions within the slices does not matter.
func CompareConditions(conditions1, conditions2 *kedav1alpha1.Conditions) bool {
	if conditions1 == nil && conditions2 == nil {
		return true
	}
	if conditions1 == nil || conditions2 == nil {
		return false
	}

	// Handle empty vs non-empty comparisons
	len1, len2 := len(*conditions1), len(*conditions2)
	if len1 == 0 || len2 == 0 {
		// Both empty: equal
		if len1 == 0 && len2 == 0 {
			return true
		}
		// One empty, other all unknown: not equal
		// An empty set of conditions is not the same as a set of 'Unknown' conditions
		if len1 == 0 && allConditionsUnknown(conditions2) {
			return false
		}
		if len2 == 0 && allConditionsUnknown(conditions1) {
			return false
		}
		// One empty, other has non-unknown conditions: not equal
		return false
	}

	map1 := conditionsToMap(conditions1)
	map2 := conditionsToMap(conditions2)

	if len(map1) != len(map2) {
		return false
	}

	for condType, cond1 := range map1 {
		cond2, ok := map2[condType]
		if !ok {
			return false // Type present in 1 but not in 2
		}
		if cond1.Status != cond2.Status || cond1.Reason != cond2.Reason || cond1.Message != cond2.Message {
			return false
		}
	}

	return true
}

// conditionsToMap converts a Conditions slice to a map for easier comparison.
func conditionsToMap(conditions *kedav1alpha1.Conditions) map[kedav1alpha1.ConditionType]kedav1alpha1.Condition {
	m := make(map[kedav1alpha1.ConditionType]kedav1alpha1.Condition)
	if conditions == nil {
		return m
	}
	for _, c := range *conditions {
		m[c.Type] = c
	}
	return m
}

// allConditionsUnknown checks if all conditions in the list are of status Unknown.
// Returns true if conditions is nil, empty, or all conditions have Unknown status.
// This is useful because an initialized set of conditions starts with Unknown status.
func allConditionsUnknown(conditions *kedav1alpha1.Conditions) bool {
	if conditions == nil || len(*conditions) == 0 {
		// nil or empty conditions are considered as having no known conditions
		return true
	}
	for _, c := range *conditions {
		if c.Status != metav1.ConditionUnknown {
			return false
		}
	}
	return true
}
