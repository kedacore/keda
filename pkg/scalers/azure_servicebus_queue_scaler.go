package scalers

import (
	"context"
	"fmt"

	log "github.com/Sirupsen/logrus"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type azureServiceBusQueueScaler struct {
	metadata *azureQueueMetadata
}

// NewAzureServiceBusQueueScaler creates a new AzureServiceBusQueueScaler
func NewAzureServiceBusQueueScaler(resolvedEnv, metadata map[string]string) (Scaler, error) {
	log.Warnf("sbq metadata %v", metadata)
	log.Warnf("sbq resolvedenv %v", resolvedEnv)

	meta, err := parseAzureQueueMetadata(metadata, resolvedEnv)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure service bus queue metadata: %s", err)
	}

	return &azureServiceBusQueueScaler{
		metadata: meta,
	}, nil
}

// Returns true if the scaler's queue has messages in it, false otherwise
func (s *azureServiceBusQueueScaler) IsActive(ctx context.Context) (bool, error) {
	length, err := GetAzureServiceBusQueueLength(ctx, s.metadata.connection, s.metadata.queueName)
	if err != nil {
		log.Errorf("error %s", err)
		return false, err
	}

	return length > 0, nil
}

// Close - nothing to close for SBQs
func (s *azureServiceBusQueueScaler) Close() error {
	return nil
}

// Returns an external metric spec for scaling. Follow this to see where it goes - it only uses default queue length target
func (s *azureServiceBusQueueScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	targetQueueLengthQty := resource.NewQuantity(int64(s.metadata.targetQueueLength), resource.DecimalSI)
	externalMetric := &v2beta1.ExternalMetricSource{MetricName: queueLengthMetricName, TargetAverageValue: targetQueueLengthQty}
	metricSpec := v2beta1.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta1.MetricSpec{metricSpec}
}

func (s *azureServiceBusQueueScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	queuelen, err := GetAzureServiceBusQueueLength(ctx, s.metadata.connection, s.metadata.queueName)

	if err != nil {
		log.Errorf("error getting queue lenngth %s", err)
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(queuelen), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
