package scalers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type kubernetesWorkloadScaler struct {
	metricType v2.MetricTargetType
	metadata   *kubernetesWorkloadMetadata
	kubeClient client.Client
	logger     logr.Logger
}

const (
	kubernetesWorkloadMetricType = "External"
	podSelectorKey               = "podSelector"
	valueKey                     = "value"
	activationValueKey           = "activationValue"
)

var phasesCountedAsTerminated = []corev1.PodPhase{
	corev1.PodSucceeded,
	corev1.PodFailed,
}

type kubernetesWorkloadMetadata struct {
	podSelector     labels.Selector
	namespace       string
	value           float64
	activationValue float64
	scalerIndex     int
}

// NewKubernetesWorkloadScaler creates a new kubernetesWorkloadScaler
func NewKubernetesWorkloadScaler(kubeClient client.Client, config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, parseErr := parseWorkloadMetadata(config)
	if parseErr != nil {
		return nil, fmt.Errorf("error parsing kubernetes workload metadata: %w", parseErr)
	}

	return &kubernetesWorkloadScaler{
		metricType: metricType,
		metadata:   meta,
		kubeClient: kubeClient,
		logger:     InitializeLogger(config, "kubernetes_workload_scaler"),
	}, nil
}

func parseWorkloadMetadata(config *ScalerConfig) (*kubernetesWorkloadMetadata, error) {
	meta := &kubernetesWorkloadMetadata{}
	var err error
	meta.namespace = config.ScalableObjectNamespace
	podSelector, err := labels.Parse(config.TriggerMetadata[podSelectorKey])
	if err != nil || podSelector.String() == "" {
		return nil, fmt.Errorf("invalid pod selector")
	}
	meta.podSelector = podSelector
	value, err := strconv.ParseFloat(config.TriggerMetadata[valueKey], 64)
	if err != nil || value == 0 {
		if config.AsMetricSource {
			value = 0
		} else {
			return nil, fmt.Errorf("value must be a float greater than 0")
		}
	}
	meta.value = value

	meta.activationValue = 0
	if val, ok := config.TriggerMetadata[activationValueKey]; ok {
		activationValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("value must be a float")
		}
		meta.activationValue = activationValue
	}

	meta.scalerIndex = config.ScalerIndex
	return meta, nil
}

// Close no need for kubernetes workload scaler
func (s *kubernetesWorkloadScaler) Close(context.Context) error {
	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *kubernetesWorkloadScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("workload-%s", s.metadata.namespace))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.value),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: kubernetesWorkloadMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric
func (s *kubernetesWorkloadScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	pods, err := s.getMetricValue(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error inspecting kubernetes workload: %w", err)
	}

	metric := GenerateMetricInMili(metricName, float64(pods))

	return []external_metrics.ExternalMetricValue{metric}, float64(pods) > s.metadata.activationValue, nil
}

func (s *kubernetesWorkloadScaler) getMetricValue(ctx context.Context) (int64, error) {
	podList := &corev1.PodList{}
	listOptions := client.ListOptions{}
	listOptions.LabelSelector = s.metadata.podSelector
	listOptions.Namespace = s.metadata.namespace
	opts := []client.ListOption{
		&listOptions,
	}

	err := s.kubeClient.List(ctx, podList, opts...)
	if err != nil {
		return 0, err
	}

	var count int64
	for _, pod := range podList.Items {
		count += getCountValue(pod)
	}

	return count, nil
}

func getCountValue(pod corev1.Pod) int64 {
	for _, ignore := range phasesCountedAsTerminated {
		if pod.Status.Phase == ignore {
			return 0
		}
	}
	return 1
}
