package scalers

import (
	"context"
	"fmt"
	"strconv"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
	"go.etcd.io/etcd/client"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type kubernetesWorkloadScaler struct {
	metadata   *kubernetesWorkloadMetadata
	kubeClient client.Client
}

const (
	podSelectorKey       = "PodSelector"
	namespaceSelectorKey = "NamespaceSelector"
	valueKey             = "Value"
)

type kubernetesWorkloadMetadata struct {
	podSelector labels.Selector
	value       int64
}

// NewKubernetesWorkloadScaler creates a new kubernetesWorkloadScaler
func NewKubernetesWorkloadScaler(kubeClient client.Client, config *ScalerConfig) (Scaler, error) {
	meta, parseErr := parseWorkloadMetadata(config)
	if parseErr != nil {
		return nil, fmt.Errorf("error parsing kubernetes workload metadata: %s", parseErr)
	}

	return &kubernetesWorkloadScaler{
		metadata:   meta,
		kubeClient: kubeClient,
	}, nil
}

func parseWorkloadMetadata(config *ScalerConfig) (*kubernetesWorkloadMetadata, error) {
	meta := &kubernetesWorkloadMetadata{}
	var err error
	meta.podSelector, err = labels.Parse(config.TriggerMetadata[podSelectorKey])
	if err != nil {
		return nil, fmt.Errorf("invalid pod selector")
	}
	meta.value, err = strconv.ParseInt(config.TriggerMetadata[valueKey], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("value must be an integer")
	}
	return meta, nil
}

// IsActive determines if we need to scale from zero
func (s *kubernetesWorkloadScaler) IsActive(ctx context.Context) (bool, error) {
	//TODO
	return true, nil
}

// Close no need for kubernetes workload scaler
func (s *kubernetesWorkloadScaler) Close() error {
	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *kubernetesWorkloadScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetMetricValue := resource.NewQuantity(s.metadata.value, resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s", "workload", s.metadata.podSelector.String())),
		},
		Target: v2beta2.MetricTarget{
			Type:  v2beta2.ValueMetricType,
			Value: targetMetricValue,
		},
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: v2beta2.ResourceMetricSourceType}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric
func (s *kubernetesWorkloadScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	return nil, nil
	// TODO
}
