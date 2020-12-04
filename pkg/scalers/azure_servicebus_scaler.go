package scalers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Azure/azure-amqp-common-go/v3/auth"
	servicebus "github.com/Azure/azure-service-bus-go"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type entityType int

const (
	none                      entityType = 0
	queue                     entityType = 1
	subscription              entityType = 2
	messageCountMetricName               = "messageCount"
	defaultTargetMessageCount            = 5
)

var azureServiceBusLog = logf.Log.WithName("azure_servicebus_scaler")

type azureServiceBusScaler struct {
	metadata    *azureServiceBusMetadata
	podIdentity kedav1alpha1.PodIdentityProvider
	httpClient  *http.Client
}

type azureServiceBusMetadata struct {
	targetLength     int
	queueName        string
	topicName        string
	subscriptionName string
	connection       string
	entityType       entityType
	namespace        string
}

// NewAzureServiceBusScaler creates a new AzureServiceBusScaler
func NewAzureServiceBusScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parseAzureServiceBusMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure service bus metadata: %s", err)
	}

	return &azureServiceBusScaler{
		metadata:    meta,
		podIdentity: config.PodIdentity,
		httpClient:  kedautil.CreateHTTPClient(config.GlobalHTTPTimeout),
	}, nil
}

// Creates an azureServiceBusMetadata struct from input metadata/env variables
func parseAzureServiceBusMetadata(config *ScalerConfig) (*azureServiceBusMetadata, error) {
	meta := azureServiceBusMetadata{}
	meta.entityType = none
	meta.targetLength = defaultTargetMessageCount

	// get target metric value
	if val, ok := config.TriggerMetadata[messageCountMetricName]; ok {
		messageCount, err := strconv.Atoi(val)
		if err != nil {
			azureServiceBusLog.Error(err, "Error parsing azure queue metadata", "messageCount", messageCountMetricName)
		} else {
			meta.targetLength = messageCount
		}
	}

	// get queue name OR topic and subscription name & set entity type accordingly
	if val, ok := config.TriggerMetadata["queueName"]; ok {
		meta.queueName = val
		meta.entityType = queue

		if _, ok := config.TriggerMetadata["subscriptionName"]; ok {
			return nil, fmt.Errorf("subscription name provided with queue name")
		}
	}

	if val, ok := config.TriggerMetadata["topicName"]; ok {
		if meta.entityType == queue {
			return nil, fmt.Errorf("both topic and queue name metadata provided")
		}
		meta.topicName = val
		meta.entityType = subscription

		if val, ok := config.TriggerMetadata["subscriptionName"]; ok {
			meta.subscriptionName = val
		} else {
			return nil, fmt.Errorf("no subscription name provided with topic name")
		}
	}

	if meta.entityType == none {
		return nil, fmt.Errorf("no service bus entity type set")
	}
	switch config.PodIdentity {
	case "", kedav1alpha1.PodIdentityProviderNone:
		// get servicebus connection string
		if config.AuthParams["connection"] != "" {
			meta.connection = config.AuthParams["connection"]
		} else if config.TriggerMetadata["connectionFromEnv"] != "" {
			meta.connection = config.ResolvedEnv[config.TriggerMetadata["connectionFromEnv"]]
		}

		if len(meta.connection) == 0 {
			return nil, fmt.Errorf("no connection setting given")
		}
	case kedav1alpha1.PodIdentityProviderAzure:
		if val, ok := config.TriggerMetadata["namespace"]; ok {
			meta.namespace = val
		} else {
			return nil, fmt.Errorf("namespace is required when using pod identity")
		}
	default:
		return nil, fmt.Errorf("azure service bus doesn't support pod identity %s", config.PodIdentity)
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
func (s *azureServiceBusScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetLengthQty := resource.NewQuantity(int64(s.metadata.targetLength), resource.DecimalSI)
	metricName := "azure-servicebus"
	if s.metadata.entityType == queue {
		metricName = kedautil.NormalizeString(fmt.Sprintf("%s-%s", metricName, s.metadata.queueName))
	} else {
		metricName = kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s", metricName, s.metadata.topicName, s.metadata.subscriptionName))
	}
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: metricName,
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetLengthQty,
		},
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta2.MetricSpec{metricSpec}
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

type azureTokenProvider struct {
	httpClient *http.Client
}

// GetToken implements TokenProvider interface for azureTokenProvider
func (a azureTokenProvider) GetToken(uri string) (*auth.Token, error) {
	token, err := azure.GetAzureADPodIdentityToken(a.httpClient, "https://servicebus.azure.net")
	if err != nil {
		return nil, err
	}

	return &auth.Token{
		TokenType: auth.CBSTokenTypeJWT,
		Token:     token.AccessToken,
		Expiry:    token.ExpiresOn,
	}, nil
}

// Returns the length of the queue or subscription
func (s *azureServiceBusScaler) GetAzureServiceBusLength(ctx context.Context) (int32, error) {
	// get namespace
	var namespace *servicebus.Namespace
	var err error
	if s.podIdentity == "" || s.podIdentity == kedav1alpha1.PodIdentityProviderNone {
		namespace, err = servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(s.metadata.connection))
		if err != nil {
			return -1, err
		}
	} else if s.podIdentity == kedav1alpha1.PodIdentityProviderAzure {
		namespace, err = servicebus.NewNamespace()
		if err != nil {
			return -1, err
		}
		namespace.TokenProvider = azureTokenProvider{
			httpClient: s.httpClient,
		}
		namespace.Name = s.metadata.namespace
	}

	// switch case for queue vs topic here
	switch s.metadata.entityType {
	case queue:
		return getQueueEntityFromNamespace(ctx, namespace, s.metadata.queueName)
	case subscription:
		return getSubscriptionEntityFromNamespace(ctx, namespace, s.metadata.topicName, s.metadata.subscriptionName)
	default:
		return -1, fmt.Errorf("no entity type")
	}
}

func getQueueEntityFromNamespace(ctx context.Context, ns *servicebus.Namespace, queueName string) (int32, error) {
	// get queue manager from namespace
	queueManager := ns.NewQueueManager()

	// queue manager.get(ctx, queueName) -> QueueEntitity
	queueEntity, err := queueManager.Get(ctx, queueName)
	if err != nil {
		return -1, err
	}

	return *queueEntity.CountDetails.ActiveMessageCount, nil
}

func getSubscriptionEntityFromNamespace(ctx context.Context, ns *servicebus.Namespace, topicName, subscriptionName string) (int32, error) {
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
