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

package modifiers

import (
	"testing"

	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scaling/cache"
)

func TestArrayContainsElement(t *testing.T) {
	tests := []struct {
		name     string
		element  string
		array    []string
		expected bool
	}{
		{
			name:     "element found exact case",
			element:  "trigger1",
			array:    []string{"trigger1", "trigger2"},
			expected: true,
		},
		{
			name:     "element found case insensitive",
			element:  "Trigger1",
			array:    []string{"trigger1", "trigger2"},
			expected: true,
		},
		{
			name:     "element not found",
			element:  "trigger3",
			array:    []string{"trigger1", "trigger2"},
			expected: false,
		},
		{
			name:     "empty array",
			element:  "trigger1",
			array:    []string{},
			expected: false,
		},
		{
			name:     "empty element in non-empty array",
			element:  "",
			array:    []string{"trigger1", "trigger2"},
			expected: false,
		},
		{
			name:     "empty element in array with empty string",
			element:  "",
			array:    []string{"", "trigger1"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ArrayContainsElement(tt.element, tt.array)
			if result != tt.expected {
				t.Errorf("ArrayContainsElement(%q, %v) = %v, want %v", tt.element, tt.array, result, tt.expected)
			}
		})
	}
}

func TestShouldTriggerBeNil(t *testing.T) {
	// Note: metricName in these tests uses simple names like "trigger-a" for clarity,
	// but in actual usage it would be the HPA-generated metric name (e.g., "s0-trigger-a")
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

func TestGetPairTriggerAndMetric(t *testing.T) {
	tests := []struct {
		name          string
		scaledObject  *kedav1alpha1.ScaledObject
		metric        string
		trigger       string
		expectedPair  map[string]string
		expectedError bool
	}{
		{
			name: "no formula defined returns empty map",
			scaledObject: &kedav1alpha1.ScaledObject{
				Spec: kedav1alpha1.ScaledObjectSpec{},
			},
			metric:        "metric1",
			trigger:       "trigger1",
			expectedPair:  map[string]string{},
			expectedError: false,
		},
		{
			name: "formula defined with valid trigger returns pair",
			scaledObject: &kedav1alpha1.ScaledObject{
				Spec: kedav1alpha1.ScaledObjectSpec{
					Advanced: &kedav1alpha1.AdvancedConfig{
						ScalingModifiers: kedav1alpha1.ScalingModifiers{
							Formula: "trigger1 + trigger2",
						},
					},
				},
			},
			metric:        "s0-metric1",
			trigger:       "trigger1",
			expectedPair:  map[string]string{"s0-metric1": "trigger1"},
			expectedError: false,
		},
		{
			name: "formula defined with empty trigger returns error",
			scaledObject: &kedav1alpha1.ScaledObject{
				Spec: kedav1alpha1.ScaledObjectSpec{
					Advanced: &kedav1alpha1.AdvancedConfig{
						ScalingModifiers: kedav1alpha1.ScalingModifiers{
							Formula: "trigger1 + trigger2",
						},
					},
				},
			},
			metric:        "s0-metric1",
			trigger:       "",
			expectedPair:  map[string]string{},
			expectedError: true,
		},
		{
			name: "nil advanced returns empty map",
			scaledObject: &kedav1alpha1.ScaledObject{
				Spec: kedav1alpha1.ScaledObjectSpec{
					Advanced: nil,
				},
			},
			metric:        "metric1",
			trigger:       "trigger1",
			expectedPair:  map[string]string{},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetPairTriggerAndMetric(tt.scaledObject, tt.metric, tt.trigger)
			if (err != nil) != tt.expectedError {
				t.Errorf("GetPairTriggerAndMetric() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if len(result) != len(tt.expectedPair) {
				t.Errorf("GetPairTriggerAndMetric() returned %d pairs, want %d", len(result), len(tt.expectedPair))
				return
			}
			for k, v := range tt.expectedPair {
				if result[k] != v {
					t.Errorf("GetPairTriggerAndMetric()[%q] = %q, want %q", k, result[k], v)
				}
			}
		})
	}
}

func TestCalculateScalingModifiersFormula(t *testing.T) {
	formula := "trigger1 + trigger2"
	compiledFormula, err := expr.Compile(formula)
	if err != nil {
		t.Fatalf("failed to compile formula: %v", err)
	}

	metric1 := external_metrics.ExternalMetricValue{
		MetricName: "s0-metric1",
		Value:      *resource.NewMilliQuantity(2000, resource.DecimalSI),
		Timestamp:  v1.Now(),
	}
	metric2 := external_metrics.ExternalMetricValue{
		MetricName: "s1-metric2",
		Value:      *resource.NewMilliQuantity(5000, resource.DecimalSI),
		Timestamp:  v1.Now(),
	}

	pairList := map[string]string{
		"s0-metric1": "trigger1",
		"s1-metric2": "trigger2",
	}

	cacheObj := &cache.ScalersCache{
		CompiledFormula: compiledFormula,
	}

	scaledObject := &kedav1alpha1.ScaledObject{}

	result, err := calculateScalingModifiersFormula(
		scaledObject,
		[]external_metrics.ExternalMetricValue{metric1, metric2},
		cacheObj,
		pairList,
	)
	if err != nil {
		t.Fatalf("calculateScalingModifiersFormula() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("calculateScalingModifiersFormula() returned %d metrics, want 1", len(result))
	}
	if result[0].MetricName != kedav1alpha1.CompositeMetricName {
		t.Errorf("metric name = %v, want %v", result[0].MetricName, kedav1alpha1.CompositeMetricName)
	}

	// trigger1=2 + trigger2=5 = 7, stored as 7000m
	expectedMilli := int64(7000)
	if result[0].Value.MilliValue() != expectedMilli {
		t.Errorf("metric value = %v milli, want %v milli", result[0].Value.MilliValue(), expectedMilli)
	}
}

func TestCalculateScalingModifiersFormulaNilCompiled(t *testing.T) {
	cacheObj := &cache.ScalersCache{
		CompiledFormula: nil,
	}

	scaledObject := &kedav1alpha1.ScaledObject{}

	_, err := calculateScalingModifiersFormula(
		scaledObject,
		[]external_metrics.ExternalMetricValue{},
		cacheObj,
		map[string]string{},
	)
	if err == nil {
		t.Error("calculateScalingModifiersFormula() with nil compiled formula should return error")
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}
