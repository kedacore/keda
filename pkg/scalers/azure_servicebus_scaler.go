package scalers

/*
Copyright 2021 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Azure/azure-amqp-common-go/v3/auth"
	servicebus "github.com/Azure/azure-service-bus-go"
	az "github.com/Azure/go-autorest/autorest/azure"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
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
	ctx         context.Context
	metricType  v2beta2.MetricTargetType
	metadata    *azureServiceBusMetadata
	podIdentity kedav1alpha1.AuthPodIdentity
	httpClient  *http.Client
}

type azureServiceBusMetadata struct {
	targetLength     int64
	queueName        string
	topicName        string
	subscriptionName string
	connection       string
	entityType       entityType
	namespace        string
	endpointSuffix   string
	scalerIndex      int
}

// NewAzureServiceBusScaler creates a new AzureServiceBusScaler
func NewAzureServiceBusScaler(ctx context.Context, config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	meta, err := parseAzureServiceBusMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure service bus metadata: %s", err)
	}

	return &azureServiceBusScaler{
		ctx:         ctx,
		metricType:  metricType,
		metadata:    meta,
		podIdentity: config.PodIdentity,
		httpClient:  kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false),
	}, nil
}

// Creates an azureServiceBusMetadata struct from input metadata/env variables
func parseAzureServiceBusMetadata(config *ScalerConfig) (*azureServiceBusMetadata, error) {
	meta := azureServiceBusMetadata{}
	meta.entityType = none
	meta.targetLength = defaultTargetMessageCount

	// get target metric value
	if val, ok := config.TriggerMetadata[messageCountMetricName]; ok {
		messageCount, err := strconv.ParseInt(val, 10, 64)
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

	envSuffixProvider := func(env az.Environment) (string, error) {
		return env.ServiceBusEndpointSuffix, nil
	}

	endpointSuffix, err := azure.ParseEnvironmentProperty(config.TriggerMetadata, azure.DefaultEndpointSuffixKey, envSuffixProvider)
	if err != nil {
		return nil, err
	}
	meta.endpointSuffix = endpointSuffix

	if meta.entityType == none {
		return nil, fmt.Errorf("no service bus entity type set")
	}
	switch config.PodIdentity.Provider {
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

	meta.scalerIndex = config.ScalerIndex

	return &meta, nil
}

// Returns true if the scaler's queue has messages in it, false otherwise
func (s *azureServiceBusScaler) IsActive(ctx context.Context) (bool, error) {
	length, err := s.getAzureServiceBusLength(ctx)
	if err != nil {
		azureServiceBusLog.Error(err, "error")
		return false, err
	}

	return length > 0, nil
}

// Close - nothing to close for SB
func (s *azureServiceBusScaler) Close(context.Context) error {
	return nil
}

// Returns the metric spec to be used by the HPA
func (s *azureServiceBusScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	metricName := ""
	if s.metadata.entityType == queue {
		metricName = s.metadata.queueName
	} else {
		metricName = s.metadata.topicName
	}

	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("azure-servicebus-%s", metricName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetLength),
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

// Returns the current metrics to be served to the HPA
func (s *azureServiceBusScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	queuelen, err := s.getAzureServiceBusLength(ctx)

	if err != nil {
		azureServiceBusLog.Error(err, "error getting service bus entity length")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(queuelen, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

type azureTokenProvider struct {
	httpClient *http.Client
	ctx        context.Context
}

// GetToken implements TokenProvider interface for azureTokenProvider
func (a azureTokenProvider) GetToken(uri string) (*auth.Token, error) {
	ctx := a.ctx

	// Service bus resource id is "https://servicebus.azure.net/" in all cloud environments
	token, err := azure.GetAzureServiceBusADPodIdentityToken(ctx, a.httpClient, "https://servicebus.azure.net/")
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
func (s *azureServiceBusScaler) getAzureServiceBusLength(ctx context.Context) (int64, error) {
	// get namespace
	namespace, err := s.getServiceBusNamespace(ctx)
	if err != nil {
		return -1, err
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

// Returns service bus namespace object
func (s *azureServiceBusScaler) getServiceBusNamespace(ctx context.Context) (*servicebus.Namespace, error) {
	var namespace *servicebus.Namespace
	var err error

	if s.podIdentity.Provider == "" || s.podIdentity.Provider == kedav1alpha1.PodIdentityProviderNone {
		namespace, err = servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(s.metadata.connection))
		if err != nil {
			return namespace, err
		}
	} else if s.podIdentity.Provider == kedav1alpha1.PodIdentityProviderAzure {
		namespace, err = servicebus.NewNamespace()
		if err != nil {
			return namespace, err
		}
		namespace.TokenProvider = azureTokenProvider{
			ctx:        ctx,
			httpClient: s.httpClient,
		}
		namespace.Name = s.metadata.namespace
	}

	namespace.Suffix = s.metadata.endpointSuffix
	return namespace, nil
}

func getQueueEntityFromNamespace(ctx context.Context, ns *servicebus.Namespace, queueName string) (int64, error) {
	// get queue manager from namespace
	queueManager := ns.NewQueueManager()

	// queue manager.get(ctx, queueName) -> QueueEntitity
	queueEntity, err := queueManager.Get(ctx, queueName)
	if err != nil {
		return -1, err
	}

	return int64(*queueEntity.CountDetails.ActiveMessageCount), nil
}

func getSubscriptionEntityFromNamespace(ctx context.Context, ns *servicebus.Namespace, topicName, subscriptionName string) (int64, error) {
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

	return int64(*subscriptionEntity.CountDetails.ActiveMessageCount), nil
}
