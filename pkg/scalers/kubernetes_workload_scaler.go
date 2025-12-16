package scalers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type kubernetesWorkloadScaler struct {
	metricType v2.MetricTargetType
	metadata   kubernetesWorkloadMetadata
	kubeClient client.Client
	logger     logr.Logger
}

const (
	kubernetesWorkloadMetricType = "External"
	allNamespaces                = "*"
	podListPageSize              = 200
)

var phasesCountedAsTerminated = []corev1.PodPhase{
	corev1.PodSucceeded,
	corev1.PodFailed,
}

type kubernetesWorkloadMetadata struct {
	PodSelector     string  `keda:"name=podSelector,     order=triggerMetadata"`
	Value           float64 `keda:"name=value,           order=triggerMetadata, default=0"`
	ActivationValue float64 `keda:"name=activationValue, order=triggerMetadata, default=0"`
	Namespace       string  `keda:"name=namespace,       order=triggerMetadata, optional"`

	namespace      string
	triggerIndex   int
	podSelector    labels.Selector
	asMetricSource bool
}

func (m *kubernetesWorkloadMetadata) Validate() error {
	if m.Value <= 0 && !m.asMetricSource {
		return fmt.Errorf("value must be a float greater than 0")
	}

	return nil
}

// NewKubernetesWorkloadScaler creates a new kubernetesWorkloadScaler
func NewKubernetesWorkloadScaler(kubeClient client.Client, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseKubernetesWorkloadMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing kubernetes workload metadata: %w", err)
	}

	return &kubernetesWorkloadScaler{
		metricType: metricType,
		metadata:   meta,
		kubeClient: kubeClient,
		logger:     InitializeLogger(config, "kubernetes_workload_scaler"),
	}, nil
}

func parseKubernetesWorkloadMetadata(config *scalersconfig.ScalerConfig) (kubernetesWorkloadMetadata, error) {
	meta := kubernetesWorkloadMetadata{}
	meta.namespace = config.ScalableObjectNamespace
	meta.triggerIndex = config.TriggerIndex
	meta.asMetricSource = config.AsMetricSource

	err := config.TypedConfig(&meta)
	if err != nil {
		return meta, fmt.Errorf("error parsing kubernetes workload metadata: %w", err)
	}

	selector, err := labels.Parse(meta.PodSelector)
	if err != nil {
		return meta, fmt.Errorf("error parsing pod selector: %w", err)
	}
	meta.podSelector = selector

	if meta.Namespace == allNamespaces {
		meta.namespace = ""
	} else if meta.Namespace != "" {
		meta.namespace = meta.Namespace
	}

	return meta, nil
}

func (s *kubernetesWorkloadScaler) Close(context.Context) error {
	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *kubernetesWorkloadScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("workload-%s", s.metadata.namespace))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Value),
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

	return []external_metrics.ExternalMetricValue{metric}, float64(pods) > s.metadata.ActivationValue, nil
}

func (s *kubernetesWorkloadScaler) getMetricValue(ctx context.Context) (int64, error) {

	var allPods []corev1.Pod
	continueToken := ""

	for {
		podList := corev1.PodList{}

		listOptions := client.ListOptions{
			LabelSelector: s.metadata.podSelector,
			Namespace:     s.metadata.namespace,
			Limit:         int64(podListPageSize),
			Continue:      continueToken,
		}

		err := s.kubeClient.List(ctx, &podList, &listOptions)
		if err != nil {
			return 0, err
		}

		allPods = append(allPods, podList.Items...)
		continueToken = podList.Continue

		if continueToken == "" {
			break
		}
	}

	var count int64
	for _, pod := range allPods {
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
