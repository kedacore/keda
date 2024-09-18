package scalers

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type cpuMemoryScaler struct {
	metadata     *cpuMemoryMetadata
	resourceName v1.ResourceName
	logger       logr.Logger
	kubeClient   client.Client
}

type cpuMemoryMetadata struct {
	Type                         v2.MetricTargetType
	AverageValue                 *resource.Quantity
	AverageUtilization           *int32
	ContainerName                string
	ActivationAverageValue       *resource.Quantity
	ActivationAverageUtilization *int32
	ScalableObjectName           string
	ScalableObjectType           string
	ScalableObjectNamespace      string
}

// NewCPUMemoryScaler creates a new cpuMemoryScaler
func NewCPUMemoryScaler(resourceName v1.ResourceName, config *scalersconfig.ScalerConfig, kubeClient client.Client) (Scaler, error) {
	logger := InitializeLogger(config, "cpu_memory_scaler")

	meta, parseErr := parseResourceMetadata(config, logger)
	if parseErr != nil {
		return nil, fmt.Errorf("error parsing %s metadata: %w", resourceName, parseErr)
	}

	return &cpuMemoryScaler{
		metadata:     meta,
		resourceName: resourceName,
		logger:       logger,
		kubeClient:   kubeClient,
	}, nil
}

func parseResourceMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*cpuMemoryMetadata, error) {
	meta := &cpuMemoryMetadata{}
	var value, activationValue string
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
	if activationValue, ok = config.TriggerMetadata["activationValue"]; !ok || activationValue == "" {
		activationValue = "0"
	}

	switch meta.Type {
	case v2.AverageValueMetricType:
		averageValueQuantity := resource.MustParse(value)
		meta.AverageValue = &averageValueQuantity

		activationValueQuantity := resource.MustParse(activationValue)
		meta.ActivationAverageValue = &activationValueQuantity
	case v2.UtilizationMetricType:
		valueNum, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return nil, err
		}
		utilizationNum := int32(valueNum)
		meta.AverageUtilization = &utilizationNum

		valueNum, err = strconv.ParseInt(activationValue, 10, 32)
		if err != nil {
			return nil, err
		}
		activationAverageUtilization := int32(valueNum)
		meta.ActivationAverageUtilization = &activationAverageUtilization
	default:
		return nil, fmt.Errorf("unsupported metric type, allowed values are 'Utilization' or 'AverageValue'")
	}

	if value, ok = config.TriggerMetadata["containerName"]; ok && value != "" {
		meta.ContainerName = value
	}

	meta.ScalableObjectName = config.ScalableObjectName
	meta.ScalableObjectNamespace = config.ScalableObjectNamespace
	meta.ScalableObjectType = config.ScalableObjectType

	return meta, nil
}

// Close no need for cpuMemory scaler
func (s *cpuMemoryScaler) Close(context.Context) error {
	return nil
}

func (s *cpuMemoryScaler) getHPA(ctx context.Context) (*v2.HorizontalPodAutoscaler, error) {
	if s.metadata.ScalableObjectType == "ScaledObject" {
		scaledObject := &kedav1alpha1.ScaledObject{}
		err := s.kubeClient.Get(ctx, types.NamespacedName{
			Name:      s.metadata.ScalableObjectName,
			Namespace: s.metadata.ScalableObjectNamespace,
		}, scaledObject)

		if err != nil {
			return nil, err
		}

		hpa := &v2.HorizontalPodAutoscaler{}
		err = s.kubeClient.Get(ctx, types.NamespacedName{
			Name:      scaledObject.Status.HpaName,
			Namespace: s.metadata.ScalableObjectNamespace,
		}, hpa)

		if err != nil {
			return nil, err
		}

		return hpa, nil
	} else if s.metadata.ScalableObjectType == "ScaledJob" {
		scaledJob := &kedav1alpha1.ScaledJob{}
		err := s.kubeClient.Get(ctx, types.NamespacedName{
			Name:      s.metadata.ScalableObjectName,
			Namespace: s.metadata.ScalableObjectNamespace,
		}, scaledJob)

		return nil, err
	}

	return nil, fmt.Errorf("invalid scalable object type: %s", s.metadata.ScalableObjectType)
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

// GetMetricsAndActivity only returns the activity of the cpu/memory scaler
func (s *cpuMemoryScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	hpa, err := s.getHPA(ctx)
	if err != nil {
		return nil, false, err
	}

	if hpa == nil {
		return nil, false, fmt.Errorf("HPA not found")
	}

	for _, metric := range hpa.Status.CurrentMetrics {
		if metric.Resource == nil {
			continue
		}

		if string(metric.Resource.Name) != metricName {
			continue
		}

		if s.metadata.Type == v2.AverageValueMetricType {
			averageValue := metric.Resource.Current.AverageValue
			if averageValue == nil {
				return nil, false, fmt.Errorf("HPA has no average value")
			}

			return nil, averageValue.Cmp(*s.metadata.ActivationAverageValue) == 1, nil
		} else if s.metadata.Type == v2.UtilizationMetricType {
			averageUtilization := metric.Resource.Current.AverageUtilization
			if averageUtilization == nil {
				return nil, false, fmt.Errorf("HPA has no average utilization")
			}

			return nil, *averageUtilization > *s.metadata.ActivationAverageUtilization, nil
		}
	}

	return nil, false, fmt.Errorf("no matching resource metric found for %s", s.resourceName)
}
