package scalers

import (
	"context"
	"fmt"
	"strconv"

	"k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type cpuMemoryScaler struct {
	metadata     *cpuMemoryMetadata
	resourceName v1.ResourceName
}

type cpuMemoryMetadata struct {
	Type               v2beta2.MetricTargetType
	Value              *resource.Quantity
	AverageValue       *resource.Quantity
	AverageUtilization *int32
}

// NewCPUMemoryScaler creates a new cpuMemoryScaler
func NewCPUMemoryScaler(resourceName v1.ResourceName, config *ScalerConfig) (Scaler, error) {
	meta, parseErr := parseResourceMetadata(config)
	if parseErr != nil {
		return nil, fmt.Errorf("error parsing %s metadata: %s", resourceName, parseErr)
	}

	return &cpuMemoryScaler{
		metadata:     meta,
		resourceName: resourceName,
	}, nil
}

func parseResourceMetadata(config *ScalerConfig) (*cpuMemoryMetadata, error) {
	meta := &cpuMemoryMetadata{}
	if val, ok := config.TriggerMetadata["type"]; ok && val != "" {
		meta.Type = v2beta2.MetricTargetType(val)
	} else {
		return nil, fmt.Errorf("no type given")
	}

	var value string
	var ok bool
	if value, ok = config.TriggerMetadata["value"]; !ok || value == "" {
		return nil, fmt.Errorf("no value given")
	}
	switch meta.Type {
	case v2beta2.ValueMetricType:
		valueQuantity := resource.MustParse(value)
		meta.Value = &valueQuantity
	case v2beta2.AverageValueMetricType:
		averageValueQuantity := resource.MustParse(value)
		meta.AverageValue = &averageValueQuantity
	case v2beta2.UtilizationMetricType:
		valueNum, err := strconv.Atoi(value)
		if err != nil {
			return nil, err
		}
		utilizationNum := int32(valueNum)
		meta.AverageUtilization = &utilizationNum
	default:
		return nil, fmt.Errorf("unsupport type")
	}
	return meta, nil
}

// IsActive always return true for cpu/memory scaler
func (s *cpuMemoryScaler) IsActive(ctx context.Context) (bool, error) {
	return true, nil
}

// Close no need for cpuMemory scaler
func (s *cpuMemoryScaler) Close() error {
	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *cpuMemoryScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	cpuMemoryMetric := &v2beta2.ResourceMetricSource{
		Name: s.resourceName,
		Target: v2beta2.MetricTarget{
			Type:               s.metadata.Type,
			Value:              s.metadata.Value,
			AverageUtilization: s.metadata.AverageUtilization,
			AverageValue:       s.metadata.AverageValue,
		},
	}
	metricSpec := v2beta2.MetricSpec{Resource: cpuMemoryMetric, Type: v2beta2.ResourceMetricSourceType}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics no need for cpu/memory scaler
func (s *cpuMemoryScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	return nil, nil
}
