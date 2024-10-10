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

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type cpuMemoryScaler struct {
	metadata     cpuMemoryMetadata
	resourceName v1.ResourceName
	logger       logr.Logger
}

type cpuMemoryMetadata struct {
	MetricType        string `keda:"name=type,order=triggerMetadata,enum=Utilization;AverageValue"`
	Value             string `keda:"name=value,order=triggerMetadata"`
	ContainerName     string `keda:"name=containerName,order=triggerMetadata,optional"`
	AverageValue      *resource.Quantity
	AverageUtilization *int32
}

func (m *cpuMemoryMetadata) Validate() error {
	return nil
}

// NewCPUMemoryScaler creates a new cpuMemoryScaler
func NewCPUMemoryScaler(resourceName v1.ResourceName, config *scalersconfig.ScalerConfig) (Scaler, error) {
	logger := InitializeLogger(config, "cpu_memory_scaler")

	meta, err := parseResourceMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s metadata: %w", resourceName, err)
	}

	return &cpuMemoryScaler{
		metadata:     meta,
		resourceName: resourceName,
		logger:       logger,
	}, nil
}

func parseResourceMetadata(config *scalersconfig.ScalerConfig) (cpuMemoryMetadata, error) {
    meta := cpuMemoryMetadata{}
    err := config.TypedConfig(&meta)
    if err != nil {
        return meta, err
    }

    if config.MetricType != "" {
        if meta.MetricType != "" {
            return meta, fmt.Errorf("only one of trigger.metadata.type or trigger.metricType should be defined")
        }
        meta.MetricType = string(config.MetricType)
    }

    switch v2.MetricTargetType(meta.MetricType) {
    case v2.AverageValueMetricType:
        averageValueQuantity := resource.MustParse(meta.Value)
        meta.AverageValue = &averageValueQuantity
    case v2.UtilizationMetricType:
        utilizationNum, err := parseUtilization(meta.Value)
        if err != nil {
            return meta, err
        }
        meta.AverageUtilization = utilizationNum
    }

    return meta, nil
}

func parseUtilization(value string) (*int32, error) {
	valueNum, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return nil, err
	}
	utilizationNum := int32(valueNum)
	return &utilizationNum, nil
}

// Close no need for cpuMemory scaler
func (s *cpuMemoryScaler) Close(context.Context) error {
	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *cpuMemoryScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricType := v2.MetricTargetType(s.metadata.MetricType)
	
	var metricSpec v2.MetricSpec
	if s.metadata.ContainerName != "" {
		containerCPUMemoryMetric := &v2.ContainerResourceMetricSource{
			Name: s.resourceName,
			Target: v2.MetricTarget{
				Type:               metricType,
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
				Type:               metricType,
				AverageUtilization: s.metadata.AverageUtilization,
				AverageValue:       s.metadata.AverageValue,
			},
		}
		metricSpec = v2.MetricSpec{Resource: cpuMemoryMetric, Type: v2.ResourceMetricSourceType}
	}

	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity no need for cpu/memory scaler and always active for cpu/memory scaler
func (s *cpuMemoryScaler) GetMetricsAndActivity(_ context.Context, _ string) ([]external_metrics.ExternalMetricValue, bool, error) {
	return nil, true, nil
}
