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
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/admin"
	az "github.com/Azure/go-autorest/autorest/azure"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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
	TargetLength            int64  `keda:"name=messageCount,          order=triggerMetadata, default=5"`
	ActivationTargetLength  int64  `keda:"name=activationMessageCount,          order=triggerMetadata, optional"`
	QueueName               string `keda:"name=queueName,          order=triggerMetadata, optional"`
	TopicName               string `keda:"name=topicName,          order=triggerMetadata, optional"`
	SubscriptionName        string `keda:"name=subscriptionName,          order=triggerMetadata, optional"`
	Connection              string `keda:"name=connection,          order=authParams;resolvedEnv, optional"`
	Namespace               string `keda:"name=namespace,          order=triggerMetadata, optional"`
	EntityType              entityType
	FullyQualifiedNamespace string
	UseRegex                bool `keda:"name=useRegex,          order=triggerMetadata, optional"`
	EntityNameRegex         *regexp.Regexp
	Operation               string `keda:"name=operation,          order=triggerMetadata, enum=sum;max;avg, default=sum"`
	triggerIndex            int
	timeout                 time.Duration
}

func (a *azureServiceBusMetadata) Validate() error {
	a.EntityType = none

	// get queue name OR topic and subscription name & set entity type accordingly
	if a.QueueName != "" {
		a.EntityType = queue

		if a.SubscriptionName != "" {
			return fmt.Errorf("subscription name provided with queue name")
		}

		if a.UseRegex {
			entityNameRegex, err := regexp.Compile(a.QueueName)
			if err != nil {
				return fmt.Errorf("queueName is not a valid regular expression")
			}
			entityNameRegex.Longest()

			a.EntityNameRegex = entityNameRegex
		}
	}

	if a.TopicName != "" {
		if a.EntityType == queue {
			return fmt.Errorf("both topic and queue name metadata provided")
		}
		a.EntityType = subscription

		if a.SubscriptionName == "" {
			return fmt.Errorf("no subscription name provided with topic name")
		}

		if a.UseRegex {
			entityNameRegex, err := regexp.Compile(a.SubscriptionName)
			if err != nil {
				return fmt.Errorf("subscriptionName is not a valid regular expression")
			}
			entityNameRegex.Longest()

			a.EntityNameRegex = entityNameRegex
		}
	}
	if a.EntityType == none {
		return fmt.Errorf("no service bus entity type set")
	}

	return nil
}

// NewAzureServiceBusScaler creates a new AzureServiceBusScaler
func NewAzureServiceBusScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "azure_servicebus_scaler")

	meta, err := parseAzureServiceBusMetadata(config)
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
func parseAzureServiceBusMetadata(config *scalersconfig.ScalerConfig) (*azureServiceBusMetadata, error) {
	meta := &azureServiceBusMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing azure service bus metadata: %w", err)
	}

	meta.triggerIndex = config.TriggerIndex
	meta.timeout = config.GlobalHTTPTimeout

	switch config.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		// get servicebus connection string
		if meta.Connection == "" {
			return meta, fmt.Errorf("no connection setting given")
		}
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		if meta.Namespace != "" {
			envSuffixProvider := func(env az.Environment) (string, error) {
				return env.ServiceBusEndpointSuffix, nil
			}

			endpointSuffix, err := azure.ParseEnvironmentProperty(config.TriggerMetadata, azure.DefaultEndpointSuffixKey, envSuffixProvider)
			if err != nil {
				return nil, err
			}
			meta.FullyQualifiedNamespace = fmt.Sprintf("%s.%s", meta.Namespace, endpointSuffix)
		} else {
			return nil, fmt.Errorf("namespace are required when using pod identity")
		}

	default:
		return nil, fmt.Errorf("azure service bus doesn't support pod identity %s", config.PodIdentity.Provider)
	}

	return meta, nil
}

// Close - nothing to close for SB
func (s *azureServiceBusScaler) Close(context.Context) error {
	return nil
}

// GetMetricSpecForScaling returns the metric spec to be used by the HPA
func (s *azureServiceBusScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := ""

	var entityType string
	if s.metadata.EntityType == queue {
		metricName = s.metadata.QueueName
		entityType = "queue"
	} else {
		metricName = s.metadata.TopicName
		entityType = "topic"
	}

	if s.metadata.UseRegex {
		metricName = fmt.Sprintf("%s-regex", entityType)
	}

	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("azure-servicebus-%s", metricName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.TargetLength),
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

	return []external_metrics.ExternalMetricValue{metric}, queuelen > s.metadata.ActivationTargetLength, nil
}

// Returns the length of the queue or subscription
func (s *azureServiceBusScaler) getAzureServiceBusLength(ctx context.Context) (int64, error) {
	// get adminClient
	adminClient, err := s.getServiceBusAdminClient()
	if err != nil {
		return -1, err
	}
	// switch case for queue vs topic here
	switch s.metadata.EntityType {
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
	opts := &admin.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: kedautil.CreateHTTPClient(s.metadata.timeout, false),
		},
	}

	switch s.podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		client, err = admin.NewClientFromConnectionString(s.metadata.Connection, opts)
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		creds, chainedErr := azure.NewChainedCredential(s.logger, s.podIdentity)
		if chainedErr != nil {
			return nil, chainedErr
		}
		client, err = admin.NewClient(s.metadata.FullyQualifiedNamespace, creds, opts)
	default:
		err = fmt.Errorf("incorrect podIdentity type")
	}

	s.client = client
	return client, err
}

func getQueueLength(ctx context.Context, adminClient *admin.Client, meta *azureServiceBusMetadata) (int64, error) {
	if !meta.UseRegex {
		queueEntity, err := adminClient.GetQueueRuntimeProperties(ctx, meta.QueueName, &admin.GetQueueRuntimePropertiesOptions{})
		if err != nil {
			return -1, err
		}
		if queueEntity == nil {
			return -1, fmt.Errorf("queue %s doesn't exist", meta.QueueName)
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
			if meta.EntityNameRegex.FindString(queue.QueueName) == queue.QueueName {
				messageCounts = append(messageCounts, int64(queue.ActiveMessageCount))
			}
		}
	}

	return performOperation(messageCounts, meta.Operation), nil
}

func getSubscriptionLength(ctx context.Context, adminClient *admin.Client, meta *azureServiceBusMetadata) (int64, error) {
	if !meta.UseRegex {
		subscriptionEntity, err := adminClient.GetSubscriptionRuntimeProperties(ctx, meta.TopicName, meta.SubscriptionName,
			&admin.GetSubscriptionRuntimePropertiesOptions{})
		if err != nil {
			return -1, err
		}
		if subscriptionEntity == nil {
			return -1, fmt.Errorf("subscription %s doesn't exist in topic %s", meta.SubscriptionName, meta.TopicName)
		}

		return int64(subscriptionEntity.ActiveMessageCount), nil
	}

	messageCounts := make([]int64, 0)

	subscriptionPager := adminClient.NewListSubscriptionsRuntimePropertiesPager(meta.TopicName, nil)
	for subscriptionPager.More() {
		page, err := subscriptionPager.NextPage(ctx)
		if err != nil {
			return -1, err
		}

		for _, subscription := range page.SubscriptionRuntimeProperties {
			if meta.EntityNameRegex.FindString(subscription.SubscriptionName) == subscription.SubscriptionName {
				messageCounts = append(messageCounts, int64(subscription.ActiveMessageCount))
			}
		}
	}

	return performOperation(messageCounts, meta.Operation), nil
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
