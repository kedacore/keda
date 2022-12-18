package scalers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type cpuMemoryScaler struct {
	metadata     *cpuMemoryMetadata
	resourceName v1.ResourceName
	logger       logr.Logger
}

type cpuMemoryMetadata struct {
	Type               v2.MetricTargetType
	AverageValue       *resource.Quantity
	AverageUtilization *int32
	ContainerName      string
}

// NewCPUMemoryScaler creates a new cpuMemoryScaler
func NewCPUMemoryScaler(resourceName v1.ResourceName, config *ScalerConfig) (Scaler, error) {
	logger := InitializeLogger(config, "cpu_memory_scaler")

	meta, parseErr := parseResourceMetadata(config, logger)
	if parseErr != nil {
		return nil, fmt.Errorf("error parsing %s metadata: %s", resourceName, parseErr)
	}

	return &cpuMemoryScaler{
		metadata:     meta,
		resourceName: resourceName,
		logger:       logger,
	}, nil
}

func parseResourceMetadata(config *ScalerConfig, logger logr.Logger) (*cpuMemoryMetadata, error) {
	meta := &cpuMemoryMetadata{}
	var value string
	var ok bool
	value, ok = config.TriggerMetadata["type"]
	switch {
	case ok && value != "" && config.MetricType != "":
		return nil, fmt.Errorf("only one of trigger.metadata.type or trigger.metricType should be defined")
	case ok && value != "":
		logger.V(0).Info("trigger.metadata.type is deprecated in favor of trigger.metricType")
		meta.Type = v2.MetricTargetType(value)
	case config.MetricType != "":
		meta.Type = config.MetricType
	default:
		return nil, fmt.Errorf("no type given in neither trigger.metadata.type or trigger.metricType")
	}

	if value, ok = config.TriggerMetadata["value"]; !ok || value == "" {
		return nil, fmt.Errorf("no value given")
	}
	switch meta.Type {
	case v2.AverageValueMetricType:
		averageValueQuantity := resource.MustParse(value)
		meta.AverageValue = &averageValueQuantity
	case v2.UtilizationMetricType:
		valueNum, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return nil, err
		}
		utilizationNum := int32(valueNum)
		meta.AverageUtilization = &utilizationNum
	default:
		return nil, fmt.Errorf("unsupported metric type, allowed values are 'Utilization' or 'AverageValue'")
	}

	if value, ok = config.TriggerMetadata["containerName"]; ok && value != "" {
		meta.ContainerName = value
	}

	return meta, nil
}

// Close no need for cpuMemory scaler
func (s *cpuMemoryScaler) Close(context.Context) error {
	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *cpuMemoryScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	var metricSpec v2.MetricSpec

	if s.metadata.ContainerName != "" {
		containerCPUMemoryMetric := &v2.ContainerResourceMetricSource{
			Name: s.resourceName,
			Target: v2.MetricTarget{
				Type:               s.metadata.Type,
				AverageUtilization: s.metadata.AverageUtilization,
				AverageValue:       s.metadata.AverageValue,
			},
			Container: s.metadata.ContainerName,
		}
		metricSpec = v2.MetricSpec{ContainerResource: containerCPUMemoryMetric, Type: v2.ContainerResourceMetricSourceType}
	} else {
		cpuMemoryMetric := &v2.ResourceMetricSource{
			Name: s.resourceName,
			Target: v2.MetricTarget{
				Type:               s.metadata.Type,
				AverageUtilization: s.metadata.AverageUtilization,
				AverageValue:       s.metadata.AverageValue,
			},
		}
		metricSpec = v2.MetricSpec{Resource: cpuMemoryMetric, Type: v2.ResourceMetricSourceType}
	}

	return []v2.MetricSpec{metricSpec}
}

// GetMetrics no need for cpu/memory scaler and always active for cpu/memory scaler
func (s *cpuMemoryScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	return nil, true, nil
}
