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
	"regexp"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/admin"
	az "github.com/Azure/go-autorest/autorest/azure"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type entityType int

const (
	none                             entityType = 0
	queue                            entityType = 1
	subscription                     entityType = 2
	messageCountMetricName                      = "messageCount"
	activationMessageCountMetricName            = "activationMessageCount"
	defaultTargetMessageCount                   = 5
)

type azureServiceBusScaler struct {
	ctx         context.Context
	metricType  v2.MetricTargetType
	metadata    *azureServiceBusMetadata
	podIdentity kedav1alpha1.AuthPodIdentity
	client      *admin.Client
	logger      logr.Logger
}

type azureServiceBusMetadata struct {
	targetLength            int64
	activationTargetLength  int64
	queueName               string
	topicName               string
	subscriptionName        string
	connection              string
	entityType              entityType
	fullyQualifiedNamespace string
	useRegex                bool
	entityNameRegex         *regexp.Regexp
	operation               string
	scalerIndex             int
}

// NewAzureServiceBusScaler creates a new AzureServiceBusScaler
func NewAzureServiceBusScaler(ctx context.Context, config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "azure_servicebus_scaler")

	meta, err := parseAzureServiceBusMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure service bus metadata: %w", err)
	}

	return &azureServiceBusScaler{
		ctx:         ctx,
		metricType:  metricType,
		metadata:    meta,
		podIdentity: config.PodIdentity,
		logger:      logger,
	}, nil
}

// Creates an azureServiceBusMetadata struct from input metadata/env variables
func parseAzureServiceBusMetadata(config *ScalerConfig, logger logr.Logger) (*azureServiceBusMetadata, error) {
	meta := azureServiceBusMetadata{}
	meta.entityType = none
	meta.targetLength = defaultTargetMessageCount

	// get target metric value
	if val, ok := config.TriggerMetadata[messageCountMetricName]; ok {
		messageCount, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			logger.Error(err, "Error parsing azure queue metadata", "messageCount", messageCountMetricName)
		} else {
			meta.targetLength = messageCount
		}
	}

	meta.activationTargetLength = 0
	if val, ok := config.TriggerMetadata[activationMessageCountMetricName]; ok {
		activationMessageCount, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			logger.Error(err, "Error parsing azure queue metadata", activationMessageCountMetricName, activationMessageCountMetricName)
			return nil, fmt.Errorf("error parsing azure queue metadata %s", activationMessageCountMetricName)
		}
		meta.activationTargetLength = activationMessageCount
	}

	meta.useRegex = false
	if val, ok := config.TriggerMetadata["useRegex"]; ok {
		useRegex, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("useRegex has invalid value")
		}
		meta.useRegex = useRegex
	}

	meta.operation = sumOperation
	if meta.useRegex {
		if val, ok := config.TriggerMetadata["operation"]; ok {
			meta.operation = val
		}

		switch meta.operation {
		case avgOperation, maxOperation, sumOperation:
		default:
			return nil, fmt.Errorf("operation must be one of avg, max, or sum")
		}
	}

	// get queue name OR topic and subscription name & set entity type accordingly
	if val, ok := config.TriggerMetadata["queueName"]; ok {
		meta.queueName = val
		meta.entityType = queue

		if _, ok := config.TriggerMetadata["subscriptionName"]; ok {
			return nil, fmt.Errorf("subscription name provided with queue name")
		}

		if meta.useRegex {
			entityNameRegex, err := regexp.Compile(meta.queueName)
			if err != nil {
				return nil, fmt.Errorf("queueName is not a valid regular expression")
			}
			entityNameRegex.Longest()

			meta.entityNameRegex = entityNameRegex
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

		if meta.useRegex {
			entityNameRegex, err := regexp.Compile(meta.subscriptionName)
			if err != nil {
				return nil, fmt.Errorf("subscriptionName is not a valid regular expression")
			}
			entityNameRegex.Longest()

			meta.entityNameRegex = entityNameRegex
		}
	}
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
	case kedav1alpha1.PodIdentityProviderAzure, kedav1alpha1.PodIdentityProviderAzureWorkload:
		if val, ok := config.TriggerMetadata["namespace"]; ok {
			envSuffixProvider := func(env az.Environment) (string, error) {
				return env.ServiceBusEndpointSuffix, nil
			}

			endpointSuffix, err := azure.ParseEnvironmentProperty(config.TriggerMetadata, azure.DefaultEndpointSuffixKey, envSuffixProvider)
			if err != nil {
				return nil, err
			}
			meta.fullyQualifiedNamespace = fmt.Sprintf("%s.%s", val, endpointSuffix)
		} else {
			return nil, fmt.Errorf("namespace are required when using pod identity")
		}

	default:
		return nil, fmt.Errorf("azure service bus doesn't support pod identity %s", config.PodIdentity.Provider)
	}

	meta.scalerIndex = config.ScalerIndex

	return &meta, nil
}

// Close - nothing to close for SB
func (s *azureServiceBusScaler) Close(context.Context) error {
	return nil
}

// GetMetricSpecForScaling returns the metric spec to be used by the HPA
func (s *azureServiceBusScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := ""

	var entityType string
	if s.metadata.entityType == queue {
		metricName = s.metadata.queueName
		entityType = "queue"
	} else {
		metricName = s.metadata.topicName
		entityType = "topic"
	}

	if s.metadata.useRegex {
		metricName = fmt.Sprintf("%s-regex", entityType)
	}

	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("azure-servicebus-%s", metricName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetLength),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns the current metrics to be served to the HPA
func (s *azureServiceBusScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	queuelen, err := s.getAzureServiceBusLength(ctx)

	if err != nil {
		s.logger.Error(err, "error getting service bus entity length")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(queuelen))

	return []external_metrics.ExternalMetricValue{metric}, queuelen > s.metadata.activationTargetLength, nil
}

// Returns the length of the queue or subscription
func (s *azureServiceBusScaler) getAzureServiceBusLength(ctx context.Context) (int64, error) {
	// get adminClient
	adminClient, err := s.getServiceBusAdminClient()
	if err != nil {
		return -1, err
	}
	// switch case for queue vs topic here
	switch s.metadata.entityType {
	case queue:
		return getQueueLength(ctx, adminClient, s.metadata)
	case subscription:
		return getSubscriptionLength(ctx, adminClient, s.metadata)
	default:
		return -1, fmt.Errorf("no entity type")
	}
}

// Returns service bus namespace object
func (s *azureServiceBusScaler) getServiceBusAdminClient() (*admin.Client, error) {
	if s.client != nil {
		return s.client, nil
	}
	var err error
	var client *admin.Client
	switch s.podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		client, err = admin.NewClientFromConnectionString(s.metadata.connection, nil)
	case kedav1alpha1.PodIdentityProviderAzure, kedav1alpha1.PodIdentityProviderAzureWorkload:
		creds, chainedErr := azure.NewChainedCredential(s.logger, s.podIdentity.GetIdentityID(), s.podIdentity.Provider)
		if chainedErr != nil {
			return nil, chainedErr
		}
		client, err = admin.NewClient(s.metadata.fullyQualifiedNamespace, creds, nil)
	default:
		err = fmt.Errorf("incorrect podIdentity type")
	}

	s.client = client
	return client, err
}

func getQueueLength(ctx context.Context, adminClient *admin.Client, meta *azureServiceBusMetadata) (int64, error) {
	if !meta.useRegex {
		queueEntity, err := adminClient.GetQueueRuntimeProperties(ctx, meta.queueName, &admin.GetQueueRuntimePropertiesOptions{})
		if err != nil {
			return -1, err
		}
		if queueEntity == nil {
			return -1, fmt.Errorf("queue %s doesn't exist", meta.queueName)
		}

		return int64(queueEntity.ActiveMessageCount), nil
	}

	messageCounts := make([]int64, 0)

	queuePager := adminClient.NewListQueuesRuntimePropertiesPager(nil)
	for queuePager.More() {
		page, err := queuePager.NextPage(ctx)
		if err != nil {
			return -1, err
		}

		for _, queue := range page.QueueRuntimeProperties {
			if meta.entityNameRegex.FindString(queue.QueueName) == queue.QueueName {
				messageCounts = append(messageCounts, int64(queue.ActiveMessageCount))
			}
		}
	}

	return performOperation(messageCounts, meta.operation), nil
}

func getSubscriptionLength(ctx context.Context, adminClient *admin.Client, meta *azureServiceBusMetadata) (int64, error) {
	if !meta.useRegex {
		subscriptionEntity, err := adminClient.GetSubscriptionRuntimeProperties(ctx, meta.topicName, meta.subscriptionName,
			&admin.GetSubscriptionRuntimePropertiesOptions{})
		if err != nil {
			return -1, err
		}
		if subscriptionEntity == nil {
			return -1, fmt.Errorf("subscription %s doesn't exist in topic %s", meta.subscriptionName, meta.topicName)
		}

		return int64(subscriptionEntity.ActiveMessageCount), nil
	}

	messageCounts := make([]int64, 0)

	subscriptionPager := adminClient.NewListSubscriptionsRuntimePropertiesPager(meta.topicName, nil)
	for subscriptionPager.More() {
		page, err := subscriptionPager.NextPage(ctx)
		if err != nil {
			return -1, err
		}

		for _, subscription := range page.SubscriptionRuntimeProperties {
			if meta.entityNameRegex.FindString(subscription.SubscriptionName) == subscription.SubscriptionName {
				messageCounts = append(messageCounts, int64(subscription.ActiveMessageCount))
			}
		}
	}

	return performOperation(messageCounts, meta.operation), nil
}

func performOperation(messageCounts []int64, operation string) int64 {
	var result int64
	for _, val := range messageCounts {
		switch operation {
		case avgOperation, sumOperation:
			result += val
		case maxOperation:
			if val > result {
				result = val
			}
		}
	}

	total := int64(len(messageCounts))
	if operation == "avg" && total != 0 {
		return result / total
	}
	return result
}
