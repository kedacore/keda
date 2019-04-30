package scalers

import (
	"context"
	"fmt"
	"strconv"

	servicebus "github.com/Azure/azure-service-bus-go"

	log "github.com/Sirupsen/logrus"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type EntityType int

const (
	None         EntityType = 0
	Queue        EntityType = 1
	Subscription EntityType = 2
)

type azureServiceBusScaler struct {
	metadata *azureServiceBusMetadata
}

type azureServiceBusMetadata struct {
	targetLength     int
	queueName        string
	topicName        string
	subscriptionName string
	connection       string
	entityType       EntityType
}

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

// Creates an azureServiceBusMetadata struct from input metadata/env variables
func parseAzureServiceBusMetadata(metadata, resolvedEnv map[string]string) (*azureServiceBusMetadata, error) {
	meta := azureServiceBusMetadata{}
	meta.entityType = None
	meta.targetLength = defaultTargetQueueLength

	if val, ok := metadata[queueLengthMetricName]; ok {
		queueLength, err := strconv.Atoi(val)
		if err != nil {
			log.Errorf("Error parsing azure queue metadata %s: %s", queueLengthMetricName, err)
		} else {
			meta.targetLength = queueLength
		}
	}

	if val, ok := metadata["queueName"]; ok {
		meta.queueName = val
		meta.entityType = queueEntity
	}

	if val, ok := metadata["topicName"]; ok {
		if meta.entityType == queueEntity {
			return nil, fmt.Errorf("Both topic and queue name metadata provided")
		}
		meta.topicName = val
		meta.entityType = subscriptionEntity

		if val, ok := metadata["subscriptionName"]; ok {
			meta.subscriptionName = val
		} else {
			return nil, fmt.Errorf("No subscription name provided with topic name")
		}
	}

	if meta.entityType == None {
		return nil, fmt.Errorf("No service bus entity type set")
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
	length, err := GetAzureServiceBusLength(ctx)
	if err != nil {
		log.Errorf("error %s", err)
		return false, err
	}

	return length > 0, nil
}

// Close - nothing to close for SB
func (s *AzureServiceBusScaler) Close() error {
	return nil
}

// Returns the metric spec to be used by the HPA
func (s *AzureServiceBusScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	targetLengthQty := resource.NewQuantity(int64(s.metadata.targetLength), resource.DecimalSI)
	externalMetric := &v2beta1.ExternalMetricSource{MetricName: queueLengthMetricName, TargetAverageValue: targetLengthQty}
	metricSpec := v2beta1.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta1.MetricSpec{metricSpec}
}

// Returns the current metrics to be served to the HPA
func (s *AzureServiceBusScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	queuelen, err := s.GetAzureServiceBusLength(ctx)

	if err != nil {
		log.Errorf("error getting service bus entity length %s", err)
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(queuelen), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// Returns the length of the queue or subscription
func (s *AzureServiceBusScaler) GetAzureServiceBusLength(ctx context.Context) (int32, error) {
	// get namespace
	namespace, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(s.metadata.connection))
	if err != nil {
		return -1, err
	}

	// switch case for queue vs topic here
	// return a servicebus entity
	entity := servicebus.Entity{}

	switch s.metadata.entityType {
	case Queue:
	case Subscription:
	}

	// return QueueEntitity.CountDetails.ActiveMessageCount
	return *entity.CountDetails.ActiveMessageCount, nil
}

func GetQueueEntityFromNamespace(ctx context.Context, ns servicebus.Namespace, queueName string) (servicebus.Entity, error) {
	// old code from az_sb_queue.go
	// get queue manager from namespace
	queueManager := namespace.NewQueueManager()

	// queue manager.get(ctx, queueName) -> QueueEntitity
	entity, err := queueManager.Get(ctx, queueName)
	if err != nil {
		return nil, err
	}
}

func GetSubscriptionEntityFromNamespace(ctx context.Context, ns servicebus.Namespace, topicName, subscriptionName string) (servicebus.Entity, error) {
	// TODO
}
