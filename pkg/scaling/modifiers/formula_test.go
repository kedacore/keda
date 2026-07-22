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
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"k8s.io/utils/ptr"

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

func TestScalingModifierTriggerValue(t *testing.T) {
	// Metric names in actual usage are HPA-generated (e.g. "s0-trigger-a");
	// "trigger-a" keeps the table readable.
	const metricName = "trigger-a"
	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewMilliQuantity(7000, resource.DecimalSI),
		Timestamp:  v1.Now(),
	}
	const metricValue = 7.0

	fallback := &kedav1alpha1.Fallback{
		FailureThreshold: 3,
		Replicas:         5,
		Behavior:         kedav1alpha1.FallbackBehaviorScalingModifiers,
	}
	withHealth := func(h *kedav1alpha1.HealthStatus) *kedav1alpha1.ScaledObject {
		so := &kedav1alpha1.ScaledObject{
			Spec:   kedav1alpha1.ScaledObjectSpec{Fallback: fallback},
			Status: kedav1alpha1.ScaledObjectStatus{Health: map[string]kedav1alpha1.HealthStatus{}},
		}
		if h != nil {
			so.Status.Health[metricName] = *h
		}
		return so
	}
	failing := func(n int32) *kedav1alpha1.HealthStatus {
		return &kedav1alpha1.HealthStatus{NumberOfFailures: ptr.To(n), Status: kedav1alpha1.HealthStatusFailing}
	}

	tests := []struct {
		name    string
		so      *kedav1alpha1.ScaledObject
		wantNil bool
	}{
		{name: "no fallback configured returns the metric value",
			so: &kedav1alpha1.ScaledObject{}},
		{name: "no health status for trigger returns the metric value",
			so: withHealth(nil)},
		{name: "failures below threshold returns the metric value",
			so: withHealth(failing(2))},
		{name: "failures at threshold returns the metric value (gate is strict >)",
			so: withHealth(failing(3))},
		{name: "failures one over threshold returns nil",
			so: withHealth(failing(4)), wantNil: true},
		{name: "failures exceed threshold returns nil",
			so: withHealth(failing(5)), wantNil: true},
		{name: "healthy (failures reset to 0) returns the metric value",
			so: withHealth(&kedav1alpha1.HealthStatus{NumberOfFailures: ptr.To(int32(0)), Status: kedav1alpha1.HealthStatusHappy})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scalingModifierTriggerValue(tt.so, metric)
			if tt.wantNil {
				assert.Nil(t, result)
				return
			}
			assert.Equal(t, metricValue, result)
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

func TestCalculateScalingModifiersFormulaFallbackReplicas(t *testing.T) {
	// "trigger1 ?? trigger2" with no constant fallback: when both triggers
	// resolve to nil, expr returns nil and the evaluator substitutes
	// Fallback.Replicas * target as the composite metric value.
	compiled, err := expr.Compile("trigger1 ?? trigger2")
	require.NoError(t, err)

	failing := func(name string) external_metrics.ExternalMetricValue {
		return external_metrics.ExternalMetricValue{
			MetricName: name,
			Value:      *resource.NewQuantity(-1, resource.DecimalSI), // placeholder, discarded by health check
			Timestamp:  v1.Now(),
		}
	}
	failures := ptr.To(int32(5))
	so := &kedav1alpha1.ScaledObject{
		Spec: kedav1alpha1.ScaledObjectSpec{
			Advanced: &kedav1alpha1.AdvancedConfig{
				ScalingModifiers: kedav1alpha1.ScalingModifiers{Target: "2"},
			},
			Fallback: &kedav1alpha1.Fallback{
				FailureThreshold: 3,
				Replicas:         7,
				Behavior:         kedav1alpha1.FallbackBehaviorScalingModifiers,
			},
		},
		Status: kedav1alpha1.ScaledObjectStatus{
			Health: map[string]kedav1alpha1.HealthStatus{
				"s0-trigger1": {NumberOfFailures: failures, Status: kedav1alpha1.HealthStatusFailing},
				"s1-trigger2": {NumberOfFailures: failures, Status: kedav1alpha1.HealthStatusFailing},
			},
		},
	}

	result, err := calculateScalingModifiersFormula(
		so,
		[]external_metrics.ExternalMetricValue{failing("s0-trigger1"), failing("s1-trigger2")},
		&cache.ScalersCache{CompiledFormula: compiled},
		map[string]string{"s0-trigger1": "trigger1", "s1-trigger2": "trigger2"},
	)
	require.NoError(t, err)
	require.Len(t, result, 1)
	// Replicas (7) * target (2) = 14 → HPA divides by target → 7 desired.
	assert.Equal(t, int64(14000), result[0].Value.MilliValue())
}

// TestHandleScalingModifiersFallbackDoesNotMultiplyByTriggerCount is a regression test for
// https://github.com/kedacore/keda/issues/5371: with 2+ triggers and a scalingModifiers
// formula (e.g. "max(trigger_1, trigger_2)"), when fallback is active every trigger
// independently reports its own fallback-derived metric value. HandleScalingModifiers must
// collapse these into a single composite metric using only the first entry - if it instead
// returned all of them (or summed them), the HPA would compute replicas as
// triggerCount * fallback.replicas instead of fallback.replicas.
func TestHandleScalingModifiersFallbackDoesNotMultiplyByTriggerCount(t *testing.T) {
	so := &kedav1alpha1.ScaledObject{
		Spec: kedav1alpha1.ScaledObjectSpec{
			Advanced: &kedav1alpha1.AdvancedConfig{
				ScalingModifiers: kedav1alpha1.ScalingModifiers{
					Formula:    "max(trigger_1, trigger_2)",
					MetricType: "AverageValue",
					Target:     "1",
				},
			},
			Fallback: &kedav1alpha1.Fallback{
				FailureThreshold: 4,
				Replicas:         12,
			},
		},
	}

	// Each trigger independently computes its own fallback metric (target * fallback.replicas = 1 * 12).
	trigger1Fallback := external_metrics.ExternalMetricValue{MetricName: "s0-trigger_1", Value: *resource.NewMilliQuantity(12000, resource.DecimalSI), Timestamp: v1.Now()}
	trigger2Fallback := external_metrics.ExternalMetricValue{MetricName: "s1-trigger_2", Value: *resource.NewMilliQuantity(12000, resource.DecimalSI), Timestamp: v1.Now()}

	result := HandleScalingModifiers(
		so,
		nil,
		map[string]string{"s0-trigger_1": "trigger_1", "s1-trigger_2": "trigger_2"},
		true,
		[]external_metrics.ExternalMetricValue{trigger1Fallback, trigger2Fallback},
		&cache.ScalersCache{},
		logr.Discard(),
	)

	require.Len(t, result, 1, "fallback with multiple triggers must collapse to a single composite metric")
	assert.Equal(t, kedav1alpha1.CompositeMetricName, result[0].MetricName)
	// 12 (target 1 * fallback.replicas 12), NOT 24 (2 triggers * 12).
	assert.Equal(t, int64(12000), result[0].Value.MilliValue())
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
