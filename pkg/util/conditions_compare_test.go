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
	"testing"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCompareConditions(t *testing.T) {
	tests := []struct {
		name       string
		conditions1 *kedav1alpha1.Conditions
		conditions2 *kedav1alpha1.Conditions
		expected   bool
	}{
		{
			name:        "both nil conditions should be equal",
			conditions1: nil,
			conditions2: nil,
			expected:    true,
		},
		{
			name:        "nil vs non-nil conditions should not be equal",
			conditions1: nil,
			conditions2: &kedav1alpha1.Conditions{},
			expected:    false,
		},
		{
			name:        "non-nil vs nil conditions should not be equal",
			conditions1: &kedav1alpha1.Conditions{},
			conditions2: nil,
			expected:    false,
		},
		{
			name:        "both empty conditions should be equal",
			conditions1: &kedav1alpha1.Conditions{},
			conditions2: &kedav1alpha1.Conditions{},
			expected:    true,
		},
		{
			name:        "empty vs non-empty conditions should not be equal",
			conditions1: &kedav1alpha1.Conditions{},
			conditions2: &kedav1alpha1.Conditions{
				{
					Type:   kedav1alpha1.ConditionReady,
					Status: metav1.ConditionTrue,
				},
			},
			expected: false,
		},
		{
			name: "empty vs all unknown conditions should not be equal",
			conditions1: &kedav1alpha1.Conditions{},
			conditions2: &kedav1alpha1.Conditions{
				{
					Type:   kedav1alpha1.ConditionReady,
					Status: metav1.ConditionUnknown,
				},
				{
					Type:   kedav1alpha1.ConditionActive,
					Status: metav1.ConditionUnknown,
				},
			},
			expected: false,
		},
		{
			name: "identical conditions should be equal",
			conditions1: &kedav1alpha1.Conditions{
				{
					Type:    kedav1alpha1.ConditionReady,
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "ScaledObject is ready",
				},
				{
					Type:    kedav1alpha1.ConditionActive,
					Status:  metav1.ConditionTrue,
					Reason:  "Active",
					Message: "ScaledObject is active",
				},
			},
			conditions2: &kedav1alpha1.Conditions{
				{
					Type:    kedav1alpha1.ConditionReady,
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "ScaledObject is ready",
				},
				{
					Type:    kedav1alpha1.ConditionActive,
					Status:  metav1.ConditionTrue,
					Reason:  "Active",
					Message: "ScaledObject is active",
				},
			},
			expected: true,
		},
		{
			name: "identical conditions in different order should be equal",
			conditions1: &kedav1alpha1.Conditions{
				{
					Type:    kedav1alpha1.ConditionReady,
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "ScaledObject is ready",
				},
				{
					Type:    kedav1alpha1.ConditionActive,
					Status:  metav1.ConditionTrue,
					Reason:  "Active",
					Message: "ScaledObject is active",
				},
			},
			conditions2: &kedav1alpha1.Conditions{
				{
					Type:    kedav1alpha1.ConditionActive,
					Status:  metav1.ConditionTrue,
					Reason:  "Active",
					Message: "ScaledObject is active",
				},
				{
					Type:    kedav1alpha1.ConditionReady,
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "ScaledObject is ready",
				},
			},
			expected: true,
		},
		{
			name: "conditions with different status should not be equal",
			conditions1: &kedav1alpha1.Conditions{
				{
					Type:    kedav1alpha1.ConditionReady,
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "ScaledObject is ready",
				},
			},
			conditions2: &kedav1alpha1.Conditions{
				{
					Type:    kedav1alpha1.ConditionReady,
					Status:  metav1.ConditionFalse,
					Reason:  "Ready",
					Message: "ScaledObject is ready",
				},
			},
			expected: false,
		},
		{
			name: "conditions with different reason should not be equal",
			conditions1: &kedav1alpha1.Conditions{
				{
					Type:    kedav1alpha1.ConditionReady,
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "ScaledObject is ready",
				},
			},
			conditions2: &kedav1alpha1.Conditions{
				{
					Type:    kedav1alpha1.ConditionReady,
					Status:  metav1.ConditionTrue,
					Reason:  "NotReady",
					Message: "ScaledObject is ready",
				},
			},
			expected: false,
		},
		{
			name: "conditions with different message should not be equal",
			conditions1: &kedav1alpha1.Conditions{
				{
					Type:    kedav1alpha1.ConditionReady,
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "ScaledObject is ready",
				},
			},
			conditions2: &kedav1alpha1.Conditions{
				{
					Type:    kedav1alpha1.ConditionReady,
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "ScaledObject is not ready",
				},
			},
			expected: false,
		},
		{
			name: "conditions with different types should not be equal",
			conditions1: &kedav1alpha1.Conditions{
				{
					Type:    kedav1alpha1.ConditionReady,
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "ScaledObject is ready",
				},
			},
			conditions2: &kedav1alpha1.Conditions{
				{
					Type:    kedav1alpha1.ConditionActive,
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "ScaledObject is ready",
				},
			},
			expected: false,
		},
		{
			name: "conditions with different number of conditions should not be equal",
			conditions1: &kedav1alpha1.Conditions{
				{
					Type:    kedav1alpha1.ConditionReady,
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "ScaledObject is ready",
				},
			},
			conditions2: &kedav1alpha1.Conditions{
				{
					Type:    kedav1alpha1.ConditionReady,
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "ScaledObject is ready",
				},
				{
					Type:    kedav1alpha1.ConditionActive,
					Status:  metav1.ConditionTrue,
					Reason:  "Active",
					Message: "ScaledObject is active",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareConditions(tt.conditions1, tt.conditions2)
			if result != tt.expected {
				t.Errorf("CompareConditions() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestAllConditionsUnknown(t *testing.T) {
	tests := []struct {
		name       string
		conditions *kedav1alpha1.Conditions
		expected   bool
	}{
		{
			name:       "nil conditions should return true",
			conditions: nil,
			expected:   true,
		},
		{
			name:       "empty conditions should return true",
			conditions: &kedav1alpha1.Conditions{},
			expected:   true,
		},
		{
			name: "all unknown conditions should return true",
			conditions: &kedav1alpha1.Conditions{
				{
					Type:   kedav1alpha1.ConditionReady,
					Status: metav1.ConditionUnknown,
				},
				{
					Type:   kedav1alpha1.ConditionActive,
					Status: metav1.ConditionUnknown,
				},
			},
			expected: true,
		},
		{
			name: "mixed conditions with unknown should return false",
			conditions: &kedav1alpha1.Conditions{
				{
					Type:   kedav1alpha1.ConditionReady,
					Status: metav1.ConditionTrue,
				},
				{
					Type:   kedav1alpha1.ConditionActive,
					Status: metav1.ConditionUnknown,
				},
			},
			expected: false,
		},
		{
			name: "all non-unknown conditions should return false",
			conditions: &kedav1alpha1.Conditions{
				{
					Type:   kedav1alpha1.ConditionReady,
					Status: metav1.ConditionTrue,
				},
				{
					Type:   kedav1alpha1.ConditionActive,
					Status: metav1.ConditionFalse,
				},
			},
			expected: false,
		},
		{
			name: "single unknown condition should return true",
			conditions: &kedav1alpha1.Conditions{
				{
					Type:   kedav1alpha1.ConditionReady,
					Status: metav1.ConditionUnknown,
				},
			},
			expected: true,
		},
		{
			name: "single non-unknown condition should return false",
			conditions: &kedav1alpha1.Conditions{
				{
					Type:   kedav1alpha1.ConditionReady,
					Status: metav1.ConditionTrue,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := allConditionsUnknown(tt.conditions)
			if result != tt.expected {
				t.Errorf("allConditionsUnknown() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestConditionsToMap(t *testing.T) {
	tests := []struct {
		name       string
		conditions *kedav1alpha1.Conditions
		expected   map[kedav1alpha1.ConditionType]kedav1alpha1.Condition
	}{
		{
			name:       "nil conditions should return empty map",
			conditions: nil,
			expected:   map[kedav1alpha1.ConditionType]kedav1alpha1.Condition{},
		},
		{
			name:       "empty conditions should return empty map",
			conditions: &kedav1alpha1.Conditions{},
			expected:   map[kedav1alpha1.ConditionType]kedav1alpha1.Condition{},
		},
		{
			name: "conditions should be mapped correctly",
			conditions: &kedav1alpha1.Conditions{
				{
					Type:    kedav1alpha1.ConditionReady,
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "ScaledObject is ready",
				},
				{
					Type:    kedav1alpha1.ConditionActive,
					Status:  metav1.ConditionFalse,
					Reason:  "NotActive",
					Message: "ScaledObject is not active",
				},
			},
			expected: map[kedav1alpha1.ConditionType]kedav1alpha1.Condition{
				kedav1alpha1.ConditionReady: {
					Type:    kedav1alpha1.ConditionReady,
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "ScaledObject is ready",
				},
				kedav1alpha1.ConditionActive: {
					Type:    kedav1alpha1.ConditionActive,
					Status:  metav1.ConditionFalse,
					Reason:  "NotActive",
					Message: "ScaledObject is not active",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := conditionsToMap(tt.conditions)
			if len(result) != len(tt.expected) {
				t.Errorf("conditionsToMap() returned map with %d elements, expected %d", len(result), len(tt.expected))
				return
			}
			for condType, expectedCond := range tt.expected {
				if actualCond, exists := result[condType]; !exists {
					t.Errorf("conditionsToMap() missing condition type %s", condType)
				} else if actualCond.Type != expectedCond.Type ||
					actualCond.Status != expectedCond.Status ||
					actualCond.Reason != expectedCond.Reason ||
					actualCond.Message != expectedCond.Message {
					t.Errorf("conditionsToMap() condition mismatch for type %s: got %+v, expected %+v", condType, actualCond, expectedCond)
				}
			}
		})
	}
} 