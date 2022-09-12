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
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/admin"
	az "github.com/Azure/go-autorest/autorest/azure"
	"github.com/go-logr/logr"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/labels"
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
	// Service bus resource id is "https://servicebus.azure.net/" in all cloud environments
	serviceBusResource = "https://servicebus.azure.net/"
)

type azureServiceBusScaler struct {
	ctx         context.Context
	metricType  v2beta2.MetricTargetType
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
	scalerIndex             int
}

// NewAzureServiceBusScaler creates a new AzureServiceBusScaler
func NewAzureServiceBusScaler(ctx context.Context, config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	logger := InitializeLogger(config, "azure_servicebus_scaler")

	meta, err := parseAzureServiceBusMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure service bus metadata: %s", err)
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
		return nil, fmt.Errorf("azure service bus doesn't support pod identity %s", config.PodIdentity)
	}

	meta.scalerIndex = config.ScalerIndex

	return &meta, nil
}

// Returns true if the scaler's queue has messages in it, false otherwise
func (s *azureServiceBusScaler) IsActive(ctx context.Context) (bool, error) {
	length, err := s.getAzureServiceBusLength(ctx)
	if err != nil {
		s.logger.Error(err, "error")
		return false, err
	}

	return length > s.metadata.activationTargetLength, nil
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
		s.logger.Error(err, "error getting service bus entity length")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := GenerateMetricInMili(metricName, float64(queuelen))

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// Returns the length of the queue or subscription
func (s *azureServiceBusScaler) getAzureServiceBusLength(ctx context.Context) (int64, error) {
	// get adminClient
	adminClient, err := s.getServiceBusAdminClient(ctx)
	if err != nil {
		return -1, err
	}
	// switch case for queue vs topic here
	switch s.metadata.entityType {
	case queue:
		return getQueueLength(ctx, adminClient, s.metadata.queueName)
	case subscription:
		return getSubscriptionLength(ctx, adminClient, s.metadata.topicName, s.metadata.subscriptionName)
	default:
		return -1, fmt.Errorf("no entity type")
	}
}

// Returns service bus namespace object
func (s *azureServiceBusScaler) getServiceBusAdminClient(ctx context.Context) (*admin.Client, error) {
	if s.client != nil {
		return s.client, nil
	}

	var adminClient *admin.Client
	var err error

	switch s.podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		adminClient, err = admin.NewClientFromConnectionString(s.metadata.connection, nil)
		if err != nil {
			return nil, err
		}
	case kedav1alpha1.PodIdentityProviderAzure:
		// Once azure-sdk-for-go supports Workload Identity we can use this for Workload identity too
		chain, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, err
		}
		adminClient, err = admin.NewClient(s.metadata.fullyQualifiedNamespace, chain, nil)
		if err != nil {
			return nil, err
		}
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		// Once azure-sdk-for-go supports Workload Identity we can remove this and use default implementation
		// https://github.com/Azure/azure-sdk-for-go/issues/15615
		var creds []azcore.TokenCredential
		options := &azidentity.DefaultAzureCredentialOptions{}

		cliCred, err := azidentity.NewAzureCLICredential(&azidentity.AzureCLICredentialOptions{TenantID: options.TenantID})
		if err == nil {
			creds = append(creds, cliCred)
		}

		wiCred := azure.NewADWorkloadIdentityCredential(ctx, s.podIdentity.IdentityID, serviceBusResource)
		creds = append(creds, wiCred)

		chain, err := azidentity.NewChainedTokenCredential(creds, nil)
		if err != nil {
			return nil, err
		}
		adminClient, err = admin.NewClient(s.metadata.fullyQualifiedNamespace, chain, nil)
		if err != nil {
			return nil, err
		}
	}

	return adminClient, nil
}

func getQueueLength(ctx context.Context, adminClient *admin.Client, queueName string) (int64, error) {
	queueEntity, err := adminClient.GetQueueRuntimeProperties(ctx, queueName, &admin.GetQueueRuntimePropertiesOptions{})
	if err != nil {
		return -1, err
	}
	if queueEntity == nil {
		return -1, fmt.Errorf("queue %s doesn't exist", queueName)
	}

	return int64(queueEntity.ActiveMessageCount), nil
}

func getSubscriptionLength(ctx context.Context, adminClient *admin.Client, topicName, subscriptionName string) (int64, error) {
	subscriptionEntity, err := adminClient.GetSubscriptionRuntimeProperties(ctx, topicName, subscriptionName, &admin.GetSubscriptionRuntimePropertiesOptions{})
	if err != nil {
		return -1, err
	}
	if subscriptionEntity == nil {
		return -1, fmt.Errorf("subscription %s doesn't exist in topic %s", subscriptionName, topicName)
	}

	return int64(subscriptionEntity.ActiveMessageCount), nil
}
