/*
Copyright 2025 The KEDA Authors

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

package modifiers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

func TestShouldTriggerBeNil(t *testing.T) {
	tests := []struct {
		name           string
		scaledObject   *kedav1alpha1.ScaledObject
		metricName     string
		expectedResult bool
		description    string
	}{
		{
			name: "no fallback configured",
			scaledObject: &kedav1alpha1.ScaledObject{
				Spec: kedav1alpha1.ScaledObjectSpec{
					Fallback: nil,
				},
				Status: kedav1alpha1.ScaledObjectStatus{},
			},
			metricName:     "trigger-a",
			expectedResult: false,
			description:    "should return false when no fallback is configured",
		},
		{
			name: "no health status for trigger",
			scaledObject: &kedav1alpha1.ScaledObject{
				Spec: kedav1alpha1.ScaledObjectSpec{
					Fallback: &kedav1alpha1.Fallback{
						FailureThreshold: 3,
						Replicas:         5,
						Behavior:         kedav1alpha1.FallbackBehaviorTriggerScoped,
					},
				},
				Status: kedav1alpha1.ScaledObjectStatus{
					Health: map[string]kedav1alpha1.HealthStatus{},
				},
			},
			metricName:     "trigger-a",
			expectedResult: false,
			description:    "should return false when trigger has no health status",
		},
		{
			name: "trigger failures below threshold",
			scaledObject: &kedav1alpha1.ScaledObject{
				Spec: kedav1alpha1.ScaledObjectSpec{
					Fallback: &kedav1alpha1.Fallback{
						FailureThreshold: 3,
						Replicas:         5,
						Behavior:         kedav1alpha1.FallbackBehaviorTriggerScoped,
					},
				},
				Status: kedav1alpha1.ScaledObjectStatus{
					Health: map[string]kedav1alpha1.HealthStatus{
						"trigger-a": {
							NumberOfFailures: int32Ptr(2),
							Status:           kedav1alpha1.HealthStatusFailing,
						},
					},
				},
			},
			metricName:     "trigger-a",
			expectedResult: false,
			description:    "should return false when failures are below threshold",
		},
		{
			name: "trigger failures at threshold",
			scaledObject: &kedav1alpha1.ScaledObject{
				Spec: kedav1alpha1.ScaledObjectSpec{
					Fallback: &kedav1alpha1.Fallback{
						FailureThreshold: 3,
						Replicas:         5,
						Behavior:         kedav1alpha1.FallbackBehaviorTriggerScoped,
					},
				},
				Status: kedav1alpha1.ScaledObjectStatus{
					Health: map[string]kedav1alpha1.HealthStatus{
						"trigger-a": {
							NumberOfFailures: int32Ptr(3),
							Status:           kedav1alpha1.HealthStatusFailing,
						},
					},
				},
			},
			metricName:     "trigger-a",
			expectedResult: true,
			description:    "should return true when failures equal threshold",
		},
		{
			name: "trigger failures exceed threshold",
			scaledObject: &kedav1alpha1.ScaledObject{
				Spec: kedav1alpha1.ScaledObjectSpec{
					Fallback: &kedav1alpha1.Fallback{
						FailureThreshold: 3,
						Replicas:         5,
						Behavior:         kedav1alpha1.FallbackBehaviorTriggerScoped,
					},
				},
				Status: kedav1alpha1.ScaledObjectStatus{
					Health: map[string]kedav1alpha1.HealthStatus{
						"trigger-a": {
							NumberOfFailures: int32Ptr(5),
							Status:           kedav1alpha1.HealthStatusFailing,
						},
					},
				},
			},
			metricName:     "trigger-a",
			expectedResult: true,
			description:    "should return true when failures exceed threshold",
		},
		{
			name: "trigger healthy after failures",
			scaledObject: &kedav1alpha1.ScaledObject{
				Spec: kedav1alpha1.ScaledObjectSpec{
					Fallback: &kedav1alpha1.Fallback{
						FailureThreshold: 3,
						Replicas:         5,
						Behavior:         kedav1alpha1.FallbackBehaviorTriggerScoped,
					},
				},
				Status: kedav1alpha1.ScaledObjectStatus{
					Health: map[string]kedav1alpha1.HealthStatus{
						"trigger-a": {
							NumberOfFailures: int32Ptr(0),
							Status:           kedav1alpha1.HealthStatusHappy,
						},
					},
				},
			},
			metricName:     "trigger-a",
			expectedResult: false,
			description:    "should return false when trigger is healthy (failures reset to 0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldTriggerBeNil(tt.scaledObject, tt.metricName)
			assert.Equal(t, tt.expectedResult, result, tt.description)
		})
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}
