package scalers

import (
	"context"
	"fmt"
	"strconv"

	log "github.com/Sirupsen/logrus"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	queueLengthMetricName    = "queueLength"
	defaultTargetQueueLength = 5
	externalMetricType       = "External"
	defaultConnectionSetting = "AzureWebJobsStorage"
)

type azureQueueScaler struct {
	metadata *azureQueueMetadata
}

type azureQueueMetadata struct {
	targetQueueLength int
	queueName         string
	connection        string
}

// NewAzureQueueScaler creates a new azureQueueScaler
func NewAzureQueueScaler(resolvedEnv, metadata map[string]string) (Scaler, error) {
	meta, err := parseAzureQueueMetadata(metadata, resolvedEnv)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure queue metadata: %s", err)
	}

	return &azureQueueScaler{
		metadata: meta,
	}, nil
}

func parseAzureQueueMetadata(metadata, resolvedEnv map[string]string) (*azureQueueMetadata, error) {
	meta := azureQueueMetadata{}
	meta.targetQueueLength = defaultTargetQueueLength

	if val, ok := metadata[queueLengthMetricName]; ok {
		queueLength, err := strconv.Atoi(val)
		if err != nil {
			log.Errorf("Error parsing azure queue metadata %s: %s", queueLengthMetricName, err)
		} else {
			meta.targetQueueLength = queueLength
		}
	}

	if val, ok := metadata["queueName"]; ok {
		meta.queueName = val
	} else {
		return nil, fmt.Errorf("no queueName given")
	}

	connectionSetting := defaultConnectionSetting
	if val, ok := metadata["connection"]; ok && val != "" {
		connectionSetting = val
	}

	if val, ok := resolvedEnv[connectionSetting]; ok {
		meta.connection = val
	} else {
		return nil, fmt.Errorf("no connection setting given")
	}

	return &meta, nil
}

// GetScaleDecision is a func
func (s *azureQueueScaler) IsActive(ctx context.Context) (bool, error) {
	length, err := GetAzureQueueLength(ctx, s.metadata.connection, s.metadata.queueName)

	if err != nil {
		log.Errorf("error %s", err)
		return false, err
	}

	return length > 0, nil
}

func (s *azureQueueScaler) Close() error {
	return nil
}

func (s *azureQueueScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	targetQueueLengthQty := resource.NewQuantity(int64(s.metadata.targetQueueLength), resource.DecimalSI)
	externalMetric := &v2beta1.ExternalMetricSource{MetricName: queueLengthMetricName, TargetAverageValue: targetQueueLengthQty}
	metricSpec := v2beta1.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta1.MetricSpec{metricSpec}
}

//GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureQueueScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	queuelen, err := GetAzureQueueLength(ctx, s.metadata.connection, s.metadata.queueName)

	if err != nil {
		log.Errorf("error getting queue length %s", err)
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(queuelen), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
