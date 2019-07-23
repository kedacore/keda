package scalers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	kedakube "github.com/kedacore/keda/pkg/kubernetes"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	eventsMetricName              = "KubernetesEventsMetric"
	defaultScaleDownPeriodSeconds = 60
	defaultNumberOfEvents         = 5
)

type k8sEventsScaler struct {
	metadata *kubernetesEventsMetadata
}

type kubernetesEventsMetadata struct {
	fieldSelector          string
	scaleDownPeriodSeconds int
	numberOfEvents         int
}

// NewKubernetesEventsScaler creates a new Kubernetes Events Scaler
func NewKubernetesEventsScaler(resolvedEnv, metadata map[string]string) (Scaler, error) {
	meta, err := parseKubernetesEventsMetadata(metadata)
	if err != nil {
		return nil, fmt.Errorf("error parsing kubernetes events metadata: %s", err)
	}

	return &k8sEventsScaler{
		metadata: meta,
	}, nil
}

func parseKubernetesEventsMetadata(metadata map[string]string) (*kubernetesEventsMetadata, error) {
	meta := kubernetesEventsMetadata{
		scaleDownPeriodSeconds: defaultScaleDownPeriodSeconds,
		numberOfEvents:         defaultNumberOfEvents,
		fieldSelector:          "",
	}

	if val, ok := metadata["scaleDownPeriodSeconds"]; ok {
		scaleDownPeriodSeconds, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("Error parsing scaleDownPeriodSeconds %s", err.Error())
		}

		meta.scaleDownPeriodSeconds = scaleDownPeriodSeconds
	}

	if val, ok := metadata["numberOfEvents"]; ok {
		numberOfEvents, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("Error parsing numberOfEvents %s", err.Error())
		}

		meta.numberOfEvents = numberOfEvents
	}

	if val, ok := metadata["fieldSelector"]; ok {
		meta.fieldSelector = val
	}

	return &meta, nil
}

// IsActive checks if there have been any events in the scaleDownPeriodSeconds
func (s *k8sEventsScaler) IsActive(ctx context.Context) (bool, error) {

	// Get Events from the kubernetes API
	count, err := s.getNumberOfEvents(ctx)
	if err != nil {
		log.Errorf("error %s", err)
		return false, err
	}

	return count > 0, nil
}

func (s *k8sEventsScaler) Close() error {
	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
// This is constructed using the numberOfEvents quantity which is set
// in the ScaledObject metadata.
func (s *k8sEventsScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {

	targetQty := resource.NewQuantity(
		int64(s.metadata.numberOfEvents), resource.DecimalSI)

	metricSpec := []v2beta1.MetricSpec{v2beta1.MetricSpec{
		External: &v2beta1.ExternalMetricSource{
			MetricName:         eventsMetricName,
			TargetAverageValue: targetQty,
		},
		Type: externalMetricType,
	}}

	return metricSpec
}

// GetMetrics checks if there have been any events in the scaleDownPeriodSeconds
// and returns the count
func (s *k8sEventsScaler) GetMetrics(ctx context.Context,
	metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {

	count, err := s.getNumberOfEvents(ctx)

	if err != nil {
		log.Errorf("error getting event count %s", err)
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(count, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// getNumberOfEvents Fetches all the events from the kubernetes cluster.
// It then finds the total number by adding all the events after a certain
// period
func (s *k8sEventsScaler) getNumberOfEvents(ctx context.Context) (int64, error) {
	_, kubeClient, err := kedakube.GetClients()
	if err != nil {
		return -1, err
	}

	events, err := kubeClient.CoreV1().Events("").List(metav1.ListOptions{FieldSelector: s.metadata.fieldSelector})

	// Subtract scaleDownPeriodSeconds from current
	timeScaleDown := time.Now().Add(
		time.Second * time.Duration(-s.metadata.scaleDownPeriodSeconds))
	var count int64

	for _, evt := range events.Items {
		if evt.LastTimestamp.Time.After(timeScaleDown) {
			count = count + 1
		}
	}

	return count, nil
}
