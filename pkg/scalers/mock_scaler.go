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

package scalers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type mockScaler struct {
	metricType v2.MetricTargetType
	metadata   mockMetadata
	logger     logr.Logger
}

type mockMetadata struct {
	triggerIndex int
	metricValue  float64
	isActive     bool
	shouldFail   bool
	failureType  string
	targetValue  float64
}

// NewMockScaler creates a new mock scaler for testing purposes
func NewMockScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	logger := InitializeLogger(config, "mock_scaler")

	// Parse metadata fields with defaults
	metricValue := 0.0
	if val, ok := config.TriggerMetadata["mockMetricValue"]; ok {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			metricValue = parsed
		}
	}

	isActive := true
	if val, ok := config.TriggerMetadata["mockIsActive"]; ok {
		isActive = val != "false"
	}

	shouldFail := false
	if val, ok := config.TriggerMetadata["mockShouldFail"]; ok {
		shouldFail = val == stringTrue
	}

	failureType := "connection"
	if val, ok := config.TriggerMetadata["mockFailureType"]; ok && val != "" {
		failureType = val
	}

	targetValue := 10.0
	if val, ok := config.TriggerMetadata["mockTargetValue"]; ok {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			targetValue = parsed
		}
	}

	return &mockScaler{
		metricType: v2.AverageValueMetricType,
		metadata: mockMetadata{
			triggerIndex: config.TriggerIndex,
			metricValue:  metricValue,
			isActive:     isActive,
			shouldFail:   shouldFail,
			failureType:  failureType,
			targetValue:  targetValue,
		},
		logger: logger,
	}, nil
}

// GetMetricsAndActivity returns the metric values and activity for the mock scaler
func (s *mockScaler) GetMetricsAndActivity(_ context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	// Simulate failure if configured
	if s.metadata.shouldFail {
		switch s.metadata.failureType {
		case "timeout":
			return nil, false, fmt.Errorf("mock scaler: timeout connecting to service")
		case "invalid":
			return nil, false, fmt.Errorf("mock scaler: invalid metric data received")
		default:
			return nil, false, fmt.Errorf("mock scaler: connection refused")
		}
	}

	// Return configured metric value
	metric := GenerateMetricInMili(metricName, s.metadata.metricValue)
	return []external_metrics.ExternalMetricValue{metric}, s.metadata.isActive, nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *mockScaler) GetMetricSpecForScaling(_ context.Context) []v2.MetricSpec {
	metricName := GenerateMetricNameWithIndex(s.metadata.triggerIndex, "mock")

	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: metricName,
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.targetValue),
	}

	metricSpec := v2.MetricSpec{
		External: externalMetric,
		Type:     externalMetricType,
	}

	return []v2.MetricSpec{metricSpec}
}

// Close cleans up any resources
func (s *mockScaler) Close(_ context.Context) error {
	return nil
}
