package scalers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
	"k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type kubernetesWorkloadScaler struct {
	metadata   *kubernetesWorkloadMetadata
	kubeClient client.Client
}

const (
	kubernetesWorkloadMetricType = "External"
	podSelectorKey               = "podSelector"
	namespaceSelectorKey         = "namespaceSelector"
	valueKey                     = "value"
)

type kubernetesWorkloadMetadata struct {
	podSelector labels.Selector
	name        string
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
	if err != nil || meta.podSelector.String() == "" {
		return nil, fmt.Errorf("invalid pod selector")
	}
	meta.value, err = strconv.ParseInt(config.TriggerMetadata[valueKey], 10, 64)
	if err != nil || meta.value == 0 {
		return nil, fmt.Errorf("value must be an integer greater than 0")
	}
	meta.name = config.Name

	return meta, nil
}

// IsActive determines if we need to scale from zero
func (s *kubernetesWorkloadScaler) IsActive(ctx context.Context) (bool, error) {
	pods, err := s.getMetricValue(ctx)

	if err != nil {
		return false, err
	}

	if pods == 0 {
		return false, fmt.Errorf("monitored pod count is 0")
	}

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
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s", "workload", normalizeSelectorString(s.metadata.podSelector))),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetMetricValue,
		},
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: kubernetesWorkloadMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric
func (s *kubernetesWorkloadScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	pods, err := s.getMetricValue(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("error inspecting kubernetes workload: %s", err)
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(pods), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (s *kubernetesWorkloadScaler) getMetricValue(ctx context.Context) (int, error) {
	podList := &corev1.PodList{}
	listOptions := client.ListOptions{}
	listOptions.LabelSelector = s.metadata.podSelector
	opts := []client.ListOption{
		&listOptions,
	}

	err := s.kubeClient.List(ctx, podList, opts...)
	if err != nil {
		return 0, err
	}

	return len(podList.Items), nil
}

func normalizeSelectorString(selector labels.Selector) string {
	s := selector.String()
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "(", "-")
	s = strings.ReplaceAll(s, ")", "-")
	s = strings.ReplaceAll(s, ",", "-")
	s = strings.ReplaceAll(s, "!", "-")
	return s
}
