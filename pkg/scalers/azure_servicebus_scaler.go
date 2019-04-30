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
	servicebus "github.com/Azure/azure-service-bus-go"
)

type azureServiceBusMetadata struct {
	targetDepth 	  int
	queueName         string
	topicName 		  string
	subscriptionName  string
	connection        string
	entityType        string // Topic or Queue
}

type azureServiceBusScaler struct {
	metadata *azureServiceBusMetadata
}

const (
	queueEntity = "QUEUE"
	subscriptionEntity = "SUBSCRIPTION"
)

// NewAzureServiceBusScaler creates a new AzureServiceBusScaler
func NewAzureServiceBusScaler(resolvedEnv, metadata map[string]string) (Scaler, error) {
	meta, err := parseAzureServiceBusMetadata(metadata, resolvedEnv)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure service bus metadata: %s", err)
	}

	return &AzureServiceBusScaler{
		metadata: meta,
	}, nil
}

func parseAzureServiceBusMetadata(metadata, resolvedEnv map[string]string) (*azureServiceBusMetadata, error) {
	meta := azureServiceBusMetadata{}
	meta.targetDepth = defaultTargetQueueLength

	if val, ok := metadata[queueLengthMetricName]; ok {
		queueLength, err := strconv.Atoi(val)
		if err != nil {
			log.Errorf("Error parsing azure queue metadata %s: %s", queueLengthMetricName, err)
		} else {
			meta.targetDepth = queueLength
		}
	}

	if val, ok := metadata["queueName"]; ok {
		meta.queueName = val
		meta.entityType = queueEntity
	}

	if val, ok := metadata["topicName"]; ok {
		if meta.entityType == queueEntity {
			return nil, fmt.Errorf("Both topic and queue name metadata provided: %s", err)
		}
		meta.topicName = val
		meta.entityType = subscriptionEntity

		if val, ok := metadata["subscriptionName"]; ok {
			meta.subscriptionName = val
		} else {
			return nil, fmt.Errorf("No subscription name provided: %s", err)
		}
	}

	if meta.entityType == "" {
		return nil, fmt.Errorf("No type set %s", err)
	}

	connectionSetting := defaultConnectionSetting
	if val, ok := metadata["connection"]; ok {
		connectionSetting = val
	}

	if val, ok := resolvedEnv[connectionSetting]; ok {
		meta.connection = val
	} else {
		return nil, fmt.Errorf("no connection setting given")
	}

	return &meta, nil
}

// Returns true if the scaler's queue has messages in it, false otherwise
func (s *AzureServiceBusScaler) IsActive(ctx context.Context) (bool, error) {
	length, err := GetAzureServiceBusQueueLength(ctx, s.metadata.connection, s.metadata.queueName)
	if err != nil {
		log.Errorf("error %s", err)
		return false, err
	}

	return length > 0, nil
}

// Close - nothing to close for SB Topics
func (s *AzureServiceBusScaler) Close() error {
	return nil
}

// Returns an external metric spec for scaling. Follow this to see where it goes - it only uses default queue length target
func (s *AzureServiceBusScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	targetQueueLengthQty := resource.NewQuantity(int64(s.metadata.targetQueueLength), resource.DecimalSI)
	externalMetric := &v2beta1.ExternalMetricSource{MetricName: queueLengthMetricName, TargetAverageValue: targetQueueLengthQty}
	metricSpec := v2beta1.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta1.MetricSpec{metricSpec}
}

func (s *AzureServiceBusScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	queuelen, err := GetAzureServiceBusSubLength(ctx, s.metadata.connection, s.metadata.queueName)

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

func GetAzureServiceBusSubLength(ctx context.Context, connectionString, topicName, subscriptionName string) (int32, error) {
}
