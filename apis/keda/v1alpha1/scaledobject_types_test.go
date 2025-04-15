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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
					t.Errorf("Error message does not contain expected text.\nExpected to contain: %s\nActual: %s", test.errorContains, err.Error())
				}
			}
		})
	}
}

func TestHasPausedReplicaAnnotation(t *testing.T) {
	tests := []struct {
		name         string
		annotations  map[string]string
		expectResult bool
	}{
		{
			name:         "No annotations",
			annotations:  nil,
			expectResult: false,
		},
		{
			name:         "Has PausedReplicasAnnotation",
			annotations:  map[string]string{PausedReplicasAnnotation: "5"},
			expectResult: true,
		},
		{
			name:         "Has other annotations but not PausedReplicasAnnotation",
			annotations:  map[string]string{"some-other-annotation": "value"},
			expectResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			so := &ScaledObject{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: test.annotations,
				},
			}
			result := so.HasPausedReplicaAnnotation()
			if result != test.expectResult {
				t.Errorf("Expected HasPausedReplicaAnnotation to return %v, got %v", test.expectResult, result)
			}
		})
	}
}

func TestHasPausedAnnotation(t *testing.T) {
	tests := []struct {
		name         string
		annotations  map[string]string
		expectResult bool
	}{
		{
			name:         "No annotations",
			annotations:  nil,
			expectResult: false,
		},
		{
			name:         "Has PausedAnnotation only",
			annotations:  map[string]string{PausedAnnotation: "true"},
			expectResult: true,
		},
		{
			name:         "Has PausedReplicasAnnotation only",
			annotations:  map[string]string{PausedReplicasAnnotation: "5"},
			expectResult: true,
		},
		{
			name:         "Has both annotations",
			annotations:  map[string]string{PausedAnnotation: "true", PausedReplicasAnnotation: "5"},
			expectResult: true,
		},
		{
			name:         "Has other annotations but not paused ones",
			annotations:  map[string]string{"some-other-annotation": "value"},
			expectResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			so := &ScaledObject{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: test.annotations,
				},
			}
			result := so.HasPausedAnnotation()
			if result != test.expectResult {
				t.Errorf("Expected HasPausedAnnotation to return %v, got %v", test.expectResult, result)
			}
		})
	}
}

func TestNeedToBePausedByAnnotation(t *testing.T) {
	pausedReplicaCount := int32(5)

	tests := []struct {
		name               string
		annotations        map[string]string
		pausedReplicaCount *int32
		expectResult       bool
	}{
		{
			name:               "No annotations",
			annotations:        nil,
			pausedReplicaCount: nil,
			expectResult:       false,
		},
		{
			name:               "PausedAnnotation with true value",
			annotations:        map[string]string{PausedAnnotation: "true"},
			pausedReplicaCount: nil,
			expectResult:       true,
		},
		{
			name:               "PausedAnnotation with false value",
			annotations:        map[string]string{PausedAnnotation: "false"},
			pausedReplicaCount: nil,
			expectResult:       false,
		},
		{
			name:               "PausedAnnotation with invalid value",
			annotations:        map[string]string{PausedAnnotation: "invalid"},
			pausedReplicaCount: nil,
			expectResult:       true, // Non-boolean values should default to true
		},
		{
			name:               "PausedReplicasAnnotation with value and status set",
			annotations:        map[string]string{PausedReplicasAnnotation: "5"},
			pausedReplicaCount: &pausedReplicaCount,
			expectResult:       true,
		},
		{
			name:               "PausedReplicasAnnotation with value but no status set",
			annotations:        map[string]string{PausedReplicasAnnotation: "5"},
			pausedReplicaCount: nil,
			expectResult:       false,
		},
		{
			name:               "Both annotations set",
			annotations:        map[string]string{PausedAnnotation: "true", PausedReplicasAnnotation: "5"},
			pausedReplicaCount: &pausedReplicaCount,
			expectResult:       true, // PausedReplicasAnnotation has precedence
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			so := &ScaledObject{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: test.annotations,
				},
				Status: ScaledObjectStatus{
					PausedReplicaCount: test.pausedReplicaCount,
				},
			}
			result := so.NeedToBePausedByAnnotation()
			if result != test.expectResult {
				t.Errorf("Expected NeedToBePausedByAnnotation to return %v, got %v", test.expectResult, result)
			}
		})
	}
}

func TestIsUsingModifiers(t *testing.T) {
	tests := []struct {
		name         string
		scaledObject *ScaledObject
		expectResult bool
	}{
		{
			name: "No Advanced config",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					Advanced: nil,
				},
			},
			expectResult: false,
		},
		{
			name: "Empty ScalingModifiers",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					Advanced: &AdvancedConfig{
						ScalingModifiers: ScalingModifiers{},
					},
				},
			},
			expectResult: false,
		},
		{
			name: "Has ScalingModifiers formula",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					Advanced: &AdvancedConfig{
						ScalingModifiers: ScalingModifiers{
							Formula: "x * 2",
						},
					},
				},
			},
			expectResult: true,
		},
		{
			name: "Has ScalingModifiers target",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					Advanced: &AdvancedConfig{
						ScalingModifiers: ScalingModifiers{
							Target: "100",
						},
					},
				},
			},
			expectResult: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.scaledObject.IsUsingModifiers()
			if result != test.expectResult {
				t.Errorf("Expected IsUsingModifiers to return %v, got %v", test.expectResult, result)
			}
		})
	}
}

func TestCheckReplicaCountBoundsAreValid(t *testing.T) {
	min1 := int32(1)
	min2 := int32(2)
	max5 := int32(5)
	idle0 := int32(0)
	idle1 := int32(1)
	idle2 := int32(2)

	tests := []struct {
		name          string
		scaledObject  *ScaledObject
		expectedError bool
		errorContains string
	}{
		{
			name: "Valid: min 1, max 5, no idle",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					MinReplicaCount:  &min1,
					MaxReplicaCount:  &max5,
					IdleReplicaCount: nil,
				},
			},
			expectedError: false,
		},
		{
			name: "Valid: min 1, max 5, idle 0",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					MinReplicaCount:  &min1,
					MaxReplicaCount:  &max5,
					IdleReplicaCount: &idle0,
				},
			},
			expectedError: false,
		},
		{
			name: "Invalid: min 2 > max 1",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					MinReplicaCount: &min2,
					MaxReplicaCount: &min1,
				},
			},
			expectedError: true,
			errorContains: "MinReplicaCount=2 must be less than MaxReplicaCount=1",
		},
		{
			name: "Invalid: idle 1 == min 1",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					MinReplicaCount:  &min1,
					MaxReplicaCount:  &max5,
					IdleReplicaCount: &idle1,
				},
			},
			expectedError: true,
			errorContains: "IdleReplicaCount=1 must be less than MinReplicaCount=1",
		},
		{
			name: "Invalid: idle 2 > min 1",
			scaledObject: &ScaledObject{
				Spec: ScaledObjectSpec{
					MinReplicaCount:  &min1,
					MaxReplicaCount:  &max5,
					IdleReplicaCount: &idle2,
				},
			},
			expectedError: true,
			errorContains: "IdleReplicaCount=2 must be less than MinReplicaCount=1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := CheckReplicaCountBoundsAreValid(test.scaledObject)

			if test.expectedError && err == nil {
				t.Error("Expected error but got nil")
			}

			if !test.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if test.expectedError && err != nil && test.errorContains != "" {
				if !strings.Contains(err.Error(), test.errorContains) {
					t.Errorf("Error message does not contain expected text.\nExpected to contain: %s\nActual: %s",
						test.errorContains, err.Error())
				}
			}
		})
	}
}

func TestGetHPAReplicas(t *testing.T) {
	min0 := int32(0)
	min5 := int32(5)
	max10 := int32(10)

	tests := []struct {
		name            string
		minReplicaCount *int32
		maxReplicaCount *int32
		expectedMin     int32
		expectedMax     int32
	}{
		{
			name:            "Default min and max",
			minReplicaCount: nil,
			maxReplicaCount: nil,
			expectedMin:     1,   // default minimum
			expectedMax:     100, // default maximum
		},
		{
			name:            "Custom min, default max",
			minReplicaCount: &min5,
			maxReplicaCount: nil,
			expectedMin:     5,
			expectedMax:     100,
		},
		{
			name:            "Default min, custom max",
			minReplicaCount: nil,
			maxReplicaCount: &max10,
			expectedMin:     1,
			expectedMax:     10,
		},
		{
			name:            "Custom min and max",
			minReplicaCount: &min5,
			maxReplicaCount: &max10,
			expectedMin:     5,
			expectedMax:     10,
		},
		{
			name:            "Zero min, default max",
			minReplicaCount: &min0,
			maxReplicaCount: nil,
			expectedMin:     1, // should use default for 0 value
			expectedMax:     100,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			so := &ScaledObject{
				Spec: ScaledObjectSpec{
					MinReplicaCount: test.minReplicaCount,
					MaxReplicaCount: test.maxReplicaCount,
				},
			}

			minReplicas := so.GetHPAMinReplicas()
			if *minReplicas != test.expectedMin {
				t.Errorf("Expected GetHPAMinReplicas to return %d, got %d", test.expectedMin, *minReplicas)
			}

			maxReplicas := so.GetHPAMaxReplicas()
			if maxReplicas != test.expectedMax {
				t.Errorf("Expected GetHPAMaxReplicas to return %d, got %d", test.expectedMax, maxReplicas)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
