package scalers

import (
	"context"
	"errors"
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
	awsSqsQueueMetricName = "ApproximateNumberOfMessages"
)

type awsSqsQueueScaler struct {
	metadata *awsSqsQueueMetadata
}

type awsSqsQueueMetadata struct {
	targetQueueLength int
	queueURL          string
}

// NewawsSqsQueueScaler creates a new awsSqsQueueScaler
func NewAwsSqsQueueScaler(resolvedEnv, metadata map[string]string) (Scaler, error) {
	meta, err := parseAwsSqsQueueMetadata(metadata, resolvedEnv)
	if err != nil {
		return nil, fmt.Errorf("Error parsing SQS queue metadata: %s", err)
	}

	return &awsSqsQueueScaler{
		metadata: meta,
	}, nil
}

func parseAwsSqsQueueMetadata(metadata, resolvedEnv map[string]string) (*awsSqsQueueMetadata, error) {
	meta := awsSqsQueueMetadata{}
	meta.targetQueueLength = defaultTargetQueueLength

	if val, ok := metadata["queueLength"]; ok {
		queueLength, err := strconv.Atoi(val)
		if err != nil {
			log.Errorf("Error parsing SQS queue metadata %s: %s", "queueLength", err)
		} else {
			meta.targetQueueLength = queueLength
		}
	}

	if val, ok := metadata["queueURL"]; ok {
		queueURL := val

		if len(val) <= 0 {
			log.Errorf("Error parsing SQS queue metadata %s: %s", "queueURL", errors.New("Empty queueURL is not valid"))
		} else {
			meta.queueURL = queueURL
		}
	}

	return &meta, nil
}

// GetScaleDecision is a func
func (s *awsSqsQueueScaler) IsActive(ctx context.Context) (bool, error) {
	length, err := GetAwsSqsQueueLength(ctx, s.metadata.queueURL)

	if err != nil {
		log.Errorf("Error %s", err)
		return false, err
	}

	return length > 0, nil
}

func (s *awsSqsQueueScaler) Close() error {
	return nil
}

func (s *awsSqsQueueScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	targetQueueLengthQty := resource.NewQuantity(int64(s.metadata.targetQueueLength), resource.DecimalSI)
	externalMetric := &v2beta1.ExternalMetricSource{MetricName: awsSqsQueueMetricName, TargetAverageValue: targetQueueLengthQty}
	metricSpec := v2beta1.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta1.MetricSpec{metricSpec}
}

//GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *awsSqsQueueScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	queuelen, err := GetAwsSqsQueueLength(ctx, s.metadata.queueURL)

	if err != nil {
		log.Errorf("Error getting queue length %s", err)
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(queuelen), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
