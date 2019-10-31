package scalers

import (
	"context"
	"fmt"
	"strconv"

	servicebus "github.com/Azure/azure-service-bus-go"

	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type EntityType int

const (
	None         EntityType = 0
	Queue        EntityType = 1
	Subscription EntityType = 2
)

var azureServiceBusLog = logf.Log.WithName("azure_servicebus_scaler")

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
	meta, err := parseAzureServiceBusMetadata(resolvedEnv, metadata)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure service bus metadata: %s", err)
	}

	return &azureServiceBusScaler{
		metadata: meta,
	}, nil
}

// Creates an azureServiceBusMetadata struct from input metadata/env variables
func parseAzureServiceBusMetadata(resolvedEnv, metadata map[string]string) (*azureServiceBusMetadata, error) {
	meta := azureServiceBusMetadata{}
	meta.entityType = None
	meta.targetLength = defaultTargetQueueLength

	// get target metric value
	if val, ok := metadata[queueLengthMetricName]; ok {
		queueLength, err := strconv.Atoi(val)
		if err != nil {
			azureServiceBusLog.Error(err, "Error parsing azure queue metadata", "queueLengthMetricName", queueLengthMetricName)
		} else {
			meta.targetLength = queueLength
		}
	}

	// get queue name OR topic and subscription name & set entity type accordingly
	if val, ok := metadata["queueName"]; ok {
		meta.queueName = val
		meta.entityType = Queue

		if _, ok := metadata["subscriptionName"]; ok {
			return nil, fmt.Errorf("No subscription name provided with topic name")
		}
	}

	if val, ok := metadata["topicName"]; ok {
		if meta.entityType == Queue {
			return nil, fmt.Errorf("Both topic and queue name metadata provided")
		}
		meta.topicName = val
		meta.entityType = Subscription

		if val, ok := metadata["subscriptionName"]; ok {
			meta.subscriptionName = val
		} else {
			return nil, fmt.Errorf("No subscription name provided with topic name")
		}
	}

	if meta.entityType == None {
		return nil, fmt.Errorf("No service bus entity type set")
	}

	// get servicebus connection string
	if val, ok := metadata["connection"]; ok {
		connectionSetting := val

		if val, ok := resolvedEnv[connectionSetting]; ok {
			meta.connection = val
		}
	}

	if meta.connection == "" {
		return nil, fmt.Errorf("no connection setting given")
	}

	return &meta, nil
}

// Returns true if the scaler's queue has messages in it, false otherwise
func (s *azureServiceBusScaler) IsActive(ctx context.Context) (bool, error) {
	length, err := s.GetAzureServiceBusLength(ctx)
	if err != nil {
		azureServiceBusLog.Error(err, "error")
		return false, err
	}

	return length > 0, nil
}

// Close - nothing to close for SB
func (s *azureServiceBusScaler) Close() error {
	return nil
}

// Returns the metric spec to be used by the HPA
func (s *azureServiceBusScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	targetLengthQty := resource.NewQuantity(int64(s.metadata.targetLength), resource.DecimalSI)
	externalMetric := &v2beta1.ExternalMetricSource{MetricName: queueLengthMetricName, TargetAverageValue: targetLengthQty}
	metricSpec := v2beta1.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta1.MetricSpec{metricSpec}
}

// Returns the current metrics to be served to the HPA
func (s *azureServiceBusScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	queuelen, err := s.GetAzureServiceBusLength(ctx)

	if err != nil {
		azureServiceBusLog.Error(err, "error getting service bus entity length")
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
func (s *azureServiceBusScaler) GetAzureServiceBusLength(ctx context.Context) (int32, error) {
	// get namespace
	namespace, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(s.metadata.connection))
	if err != nil {
		return -1, err
	}

	// switch case for queue vs topic here
	switch s.metadata.entityType {
	case Queue:
		return GetQueueEntityFromNamespace(ctx, namespace, s.metadata.queueName)
	case Subscription:
		return GetSubscriptionEntityFromNamespace(ctx, namespace, s.metadata.topicName, s.metadata.subscriptionName)
	default:
		return -1, fmt.Errorf("No entity type")
	}
}

func GetQueueEntityFromNamespace(ctx context.Context, ns *servicebus.Namespace, queueName string) (int32, error) {
	// get queue manager from namespace
	queueManager := ns.NewQueueManager()

	// queue manager.get(ctx, queueName) -> QueueEntitity
	queueEntity, err := queueManager.Get(ctx, queueName)
	if err != nil {
		return -1, err
	}

	return *queueEntity.CountDetails.ActiveMessageCount, nil
}

func GetSubscriptionEntityFromNamespace(ctx context.Context, ns *servicebus.Namespace, topicName, subscriptionName string) (int32, error) {
	// get subscription manager from namespace
	subscriptionManager, err := ns.NewSubscriptionManager(topicName)
	if err != nil {
		return -1, err
	}

	// subscription manager.get(ctx, subName) -> SubscriptionEntity
	subscriptionEntity, err := subscriptionManager.Get(ctx, subscriptionName)
	if err != nil {
		return -1, err
	}

	return *subscriptionEntity.CountDetails.ActiveMessageCount, nil
}
