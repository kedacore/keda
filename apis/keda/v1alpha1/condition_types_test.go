/*
Copyright 2026 The KEDA Authors

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

package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetInitializedConditions(t *testing.T) {
	conditions := GetInitializedConditions()

	if conditions == nil {
		t.Fatal("expected non-nil Conditions")
	}
	if len(*conditions) != 4 {
		t.Fatalf("expected 4 conditions, got %d", len(*conditions))
	}

	tests := []struct {
		conditionType  ConditionType
		expectedStatus metav1.ConditionStatus
	}{
		{ConditionReady, metav1.ConditionUnknown},
		{ConditionActive, metav1.ConditionUnknown},
		{ConditionFallback, metav1.ConditionUnknown},
		{ConditionPaused, metav1.ConditionFalse},
	}

	for _, tt := range tests {
		found := false
		for _, c := range *conditions {
			if c.Type == tt.conditionType {
				found = true
				if c.Status != tt.expectedStatus {
					t.Errorf("condition %s: expected status %s, got %s", tt.conditionType, tt.expectedStatus, c.Status)
				}
				break
			}
		}
		if !found {
			t.Errorf("condition %s not found", tt.conditionType)
		}
	}
}

func TestConditions_AreInitialized(t *testing.T) {
	tests := []struct {
		name       string
		conditions Conditions
		want       bool
	}{
		{
			name:       "nil conditions",
			conditions: nil,
			want:       false,
		},
		{
			name:       "empty conditions",
			conditions: Conditions{},
			want:       false,
		},
		{
			name: "partial conditions - missing Paused",
			conditions: Conditions{
				{Type: ConditionReady, Status: metav1.ConditionUnknown},
				{Type: ConditionActive, Status: metav1.ConditionUnknown},
				{Type: ConditionFallback, Status: metav1.ConditionUnknown},
			},
			want: false,
		},
		{
			name:       "fully initialized conditions",
			conditions: *GetInitializedConditions(),
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.conditions.AreInitialized()
			if got != tt.want {
				t.Errorf("AreInitialized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCondition_IsTrue(t *testing.T) {
	tests := []struct {
		name      string
		condition *Condition
		want      bool
	}{
		{
			name:      "nil condition",
			condition: nil,
			want:      false,
		},
		{
			name:      "condition is True",
			condition: &Condition{Status: metav1.ConditionTrue},
			want:      true,
		},
		{
			name:      "condition is False",
			condition: &Condition{Status: metav1.ConditionFalse},
			want:      false,
		},
		{
			name:      "condition is Unknown",
			condition: &Condition{Status: metav1.ConditionUnknown},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.condition.IsTrue(); got != tt.want {
				t.Errorf("IsTrue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCondition_IsFalse(t *testing.T) {
	tests := []struct {
		name      string
		condition *Condition
		want      bool
	}{
		{
			name:      "nil condition",
			condition: nil,
			want:      false,
		},
		{
			name:      "condition is True",
			condition: &Condition{Status: metav1.ConditionTrue},
			want:      false,
		},
		{
			name:      "condition is False",
			condition: &Condition{Status: metav1.ConditionFalse},
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.condition.IsFalse(); got != tt.want {
				t.Errorf("IsFalse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCondition_IsUnknown(t *testing.T) {
	tests := []struct {
		name      string
		condition *Condition
		want      bool
	}{
		{
			name:      "nil condition returns true",
			condition: nil,
			want:      true,
		},
		{
			name:      "condition is Unknown",
			condition: &Condition{Status: metav1.ConditionUnknown},
			want:      true,
		},
		{
			name:      "condition is True",
			condition: &Condition{Status: metav1.ConditionTrue},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.condition.IsUnknown(); got != tt.want {
				t.Errorf("IsUnknown() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConditions_SetAndGetConditions(t *testing.T) {
	conditions := GetInitializedConditions()

	tests := []struct {
		name          string
		setFunc       func(metav1.ConditionStatus, string, string)
		getFunc       func() Condition
		conditionType ConditionType
	}{
		{
			name:          "Ready",
			setFunc:       conditions.SetReadyCondition,
			getFunc:       conditions.GetReadyCondition,
			conditionType: ConditionReady,
		},
		{
			name:          "Active",
			setFunc:       conditions.SetActiveCondition,
			getFunc:       conditions.GetActiveCondition,
			conditionType: ConditionActive,
		},
		{
			name:          "Fallback",
			setFunc:       conditions.SetFallbackCondition,
			getFunc:       conditions.GetFallbackCondition,
			conditionType: ConditionFallback,
		},
		{
			name:          "Paused",
			setFunc:       conditions.SetPausedCondition,
			getFunc:       conditions.GetPausedCondition,
			conditionType: ConditionPaused,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setFunc(metav1.ConditionTrue, "TestReason", "test message")
			got := tt.getFunc()

			if got.Type != tt.conditionType {
				t.Errorf("expected type %s, got %s", tt.conditionType, got.Type)
			}
			if got.Status != metav1.ConditionTrue {
				t.Errorf("expected status True, got %s", got.Status)
			}
			if got.Reason != "TestReason" {
				t.Errorf("expected reason TestReason, got %s", got.Reason)
			}
			if got.Message != "test message" {
				t.Errorf("expected message 'test message', got %s", got.Message)
			}
		})
	}
}

func TestConditions_SetCondition_NilInitializes(t *testing.T) {
	var conditions Conditions

	conditions.SetReadyCondition(metav1.ConditionTrue, "Ready", "ready message")

	if conditions == nil {
		t.Fatal("expected SetReadyCondition to initialize nil Conditions")
	}
	if !conditions.AreInitialized() {
		t.Error("expected all condition types to be present after initialization")
	}

	got := conditions.GetReadyCondition()
	if got.Status != metav1.ConditionTrue {
		t.Errorf("expected Ready status True, got %s", got.Status)
	}
	if got.Reason != "Ready" {
		t.Errorf("expected reason 'Ready', got %s", got.Reason)
	}
}

func TestConditions_GetCondition_NilInitializes(t *testing.T) {
	var conditions Conditions

	got := conditions.GetActiveCondition()

	if conditions == nil {
		t.Fatal("expected GetActiveCondition to initialize nil Conditions")
	}
	if got.Type != ConditionActive {
		t.Errorf("expected Active condition, got %+v", got)
	}
	if got.Status != metav1.ConditionUnknown {
		t.Errorf("expected Unknown status for unset Active condition, got %s", got.Status)
	}
}

func TestConditions_getCondition_NotFound(t *testing.T) {
	conditions := Conditions{
		{Type: ConditionReady, Status: metav1.ConditionTrue},
	}

	got := conditions.getCondition(ConditionActive)
	if got.Type != "" {
		t.Errorf("expected empty condition for missing type, got %+v", got)
	}
}
