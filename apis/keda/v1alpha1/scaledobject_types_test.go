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

package v1alpha1

import (
	"strings"
	"testing"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
)

func TestCheckFallbackValid(t *testing.T) {
	tests := []struct {
		name          string
		scaledObject  *ScaledObject
		expectedError bool
		errorContains string
	}{
		{
			name: "No fallback configured",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					Fallback: nil,
					Triggers: []ScaleTriggers{
						{
							Type: "couchdb",
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Explicit AverageValue metricType - valid",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					Fallback: &Fallback{
						FailureThreshold: 3,
						Replicas:         1,
					},
					Triggers: []ScaleTriggers{
						{
							Type:       "couchdb",
							MetricType: autoscalingv2.AverageValueMetricType,
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Implicit AverageValue metricType (empty string) - valid",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					Fallback: &Fallback{
						FailureThreshold: 3,
						Replicas:         1,
					},
					Triggers: []ScaleTriggers{
						{
							Type:       "couchdb",
							MetricType: "", // Empty string should default to AverageValue
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Value metricType - invalid",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					Fallback: &Fallback{
						FailureThreshold: 3,
						Replicas:         1,
					},
					Triggers: []ScaleTriggers{
						{
							Type:       "couchdb",
							MetricType: autoscalingv2.ValueMetricType,
						},
					},
				},
			},
			expectedError: true,
			errorContains: "type for the fallback to be enabled",
		},
		{
			name: "Multiple triggers with one valid AverageValue - valid",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					Fallback: &Fallback{
						FailureThreshold: 3,
						Replicas:         1,
					},
					Triggers: []ScaleTriggers{
						{
							Type:       "prometheus",
							MetricType: autoscalingv2.ValueMetricType,
						},
						{
							Type:       "couchdb",
							MetricType: autoscalingv2.AverageValueMetricType,
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "CPU trigger - invalid",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					Fallback: &Fallback{
						FailureThreshold: 3,
						Replicas:         1,
					},
					Triggers: []ScaleTriggers{
						{
							Type:       "cpu",
							MetricType: autoscalingv2.AverageValueMetricType,
						},
					},
				},
			},
			expectedError: true,
			errorContains: "type for the fallback to be enabled",
		},
		{
			name: "Memory trigger - invalid",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					Fallback: &Fallback{
						FailureThreshold: 3,
						Replicas:         1,
					},
					Triggers: []ScaleTriggers{
						{
							Type:       "memory",
							MetricType: autoscalingv2.AverageValueMetricType,
						},
					},
				},
			},
			expectedError: true,
			errorContains: "type for the fallback to be enabled",
		},
		{
			name: "Multiple triggers with one CPU and one valid - valid",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					Fallback: &Fallback{
						FailureThreshold: 3,
						Replicas:         1,
					},
					Triggers: []ScaleTriggers{
						{
							Type:       "cpu",
							MetricType: autoscalingv2.UtilizationMetricType,
						},
						{
							Type:       "couchdb",
							MetricType: autoscalingv2.AverageValueMetricType,
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Negative FailureThreshold - invalid",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					Fallback: &Fallback{
						FailureThreshold: -1,
						Replicas:         1,
					},
					Triggers: []ScaleTriggers{
						{
							Type:       "couchdb",
							MetricType: autoscalingv2.AverageValueMetricType,
						},
					},
				},
			},
			expectedError: true,
			errorContains: "must both be greater than or equal to 0",
		},
		{
			name: "Negative Replicas - invalid",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					Fallback: &Fallback{
						FailureThreshold: 3,
						Replicas:         -1,
					},
					Triggers: []ScaleTriggers{
						{
							Type:       "couchdb",
							MetricType: autoscalingv2.AverageValueMetricType,
						},
					},
				},
			},
			expectedError: true,
			errorContains: "must both be greater than or equal to 0",
		},
		{
			name: "Using ScalingModifiers with AverageValue MetricType - valid",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					Fallback: &Fallback{
						FailureThreshold: 3,
						Replicas:         1,
					},
					Advanced: &AdvancedConfig{
						ScalingModifiers: ScalingModifiers{
							MetricType: autoscalingv2.AverageValueMetricType,
							Formula:    "x * 2",
						},
					},
					Triggers: []ScaleTriggers{
						{
							Type: "couchdb",
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Using ScalingModifiers with Value MetricType - invalid",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					Fallback: &Fallback{
						FailureThreshold: 3,
						Replicas:         1,
					},
					Advanced: &AdvancedConfig{
						ScalingModifiers: ScalingModifiers{
							MetricType: autoscalingv2.ValueMetricType,
							Formula:    "x * 2",
						},
					},
					Triggers: []ScaleTriggers{
						{
							Type: "couchdb",
						},
					},
				},
			},
			expectedError: true,
			errorContains: "ScaledObject.Spec.Advanced.ScalingModifiers.MetricType must be AverageValue",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := CheckFallbackValid(test.scaledObject)
			
			if test.expectedError && err == nil {
				t.Error("Expected error but got nil")
			}
			
			if !test.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			
			if test.expectedError && err != nil && test.errorContains != "" {
				if !contains(err.Error(), test.errorContains) {
					t.Errorf("Error message does not contain expected text.\nExpected to contain: %s\nActual: %s", 
						test.errorContains, err.Error())
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
