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

type AzureQueueScaler struct {
	ResolvedSecrets, Metadata map[string]string
}

// GetScaleDecision is a func
func (s *AzureQueueScaler) GetScaleDecision(ctx context.Context) (int32, error) {
	connectionString := getConnectionString(s)
	queueName := getQueueName(s)

	length, err := getQueueLength(ctx, connectionString, queueName)

	if err != nil {
		log.Errorf("error %s", err)
		return -1, err
	}

	if length > 0 {
		return 1, nil
	}

	return 0, nil
}

func (s *AzureQueueScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	targetQueueLengthQty := resource.NewQuantity(targetQueueLength, resource.DecimalSI)
	externalMetric := &v2beta1.ExternalMetricSource{MetricName: queueLengthMetricName, TargetAverageValue: targetQueueLengthQty}
	metricSpec := v2beta1.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta1.MetricSpec{metricSpec}
}

// returns value for a supported metric and an error if there is a problem getting the metric
func (s *AzureQueueScaler) GetMetrics(ctx context.Context, merticName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	connectionString := getConnectionString(s)
	queueName := getQueueName(s)

	queuelen, err := getQueueLength(ctx, connectionString, queueName)

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

func getConnectionString(s *AzureQueueScaler) string {
	connectionSettingName := s.Metadata["connection"]
	if connectionSettingName == "" {
		connectionSettingName = "AzureWebJobsStorage"
	}

	return s.ResolvedSecrets[connectionSettingName]
}

func getQueueName(s *AzureQueueScaler) string {
	return s.Metadata["queueName"]
}
