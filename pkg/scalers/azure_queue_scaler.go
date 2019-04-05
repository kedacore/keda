package scalers

import (
	"context"

	log "github.com/Sirupsen/logrus"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	queueLengthMetricName = "queue-length"
	targetQueueLength     = 5
	queueLabelName        = "azure-queue-name"
	externalMetricType    = "External"
)

type azureQueueScaler struct {
	resolvedEnv, metadata map[string]string
}

// NewAzureQueueScaler creates a new azureQueueScaler
func NewAzureQueueScaler(resolvedEnv, metadata map[string]string) Scaler {
	return &azureQueueScaler{
		resolvedEnv: resolvedEnv,
		metadata:    metadata,
	}
}

// GetScaleDecision is a func
func (s *azureQueueScaler) IsActive(ctx context.Context) (bool, error) {
	connectionString := s.getConnectionString()
	queueName := s.getQueueName()

	length, err := GetAzureQueueLength(ctx, connectionString, queueName)

	if err != nil {
		log.Errorf("error %s", err)
		return false, err
	}

	return length > 0, nil
}

func (s *azureQueueScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	targetQueueLengthQty := resource.NewQuantity(targetQueueLength, resource.DecimalSI)
	externalMetric := &v2beta1.ExternalMetricSource{MetricName: queueLengthMetricName, TargetAverageValue: targetQueueLengthQty}
	metricSpec := v2beta1.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta1.MetricSpec{metricSpec}
}

//GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureQueueScaler) GetMetrics(ctx context.Context, merticName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	connectionString := s.getConnectionString()
	queueName := s.getQueueName()

	queuelen, err := GetAzureQueueLength(ctx, connectionString, queueName)

	if err != nil {
		log.Errorf("error getting queue lenngth %s", err)
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: merticName,
		Value:      *resource.NewQuantity(int64(queuelen), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (s *azureQueueScaler) getConnectionString() string {
	connectionSettingName := s.metadata["connection"]
	if connectionSettingName == "" {
		connectionSettingName = "AzureWebJobsStorage"
	}

	return s.resolvedEnv[connectionSettingName]
}

func (s *azureQueueScaler) getQueueName() string {
	return s.metadata["queueName"]
}
