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

package util

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestIsPodReady(t *testing.T) {
	withConditions := func(conditions ...corev1.PodCondition) *corev1.Pod {
		return &corev1.Pod{Status: corev1.PodStatus{Conditions: conditions}}
	}

	tests := []struct {
		name string
		pod  *corev1.Pod
		want bool
	}{
		{"nil pod", nil, false},
		{"no conditions", withConditions(), false},
		{"ready true", withConditions(corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionTrue}), true},
		{"ready false", withConditions(corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionFalse}), false},
		{"ready unknown", withConditions(corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionUnknown}), false},
		{"only other conditions", withConditions(corev1.PodCondition{Type: corev1.ContainersReady, Status: corev1.ConditionTrue}), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPodReady(tt.pod); got != tt.want {
				t.Errorf("IsPodReady() = %v, want %v", got, tt.want)
			}
		})
	}
}
