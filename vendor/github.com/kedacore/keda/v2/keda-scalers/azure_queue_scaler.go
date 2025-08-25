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

package scalers

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/keda-scalers/azure"
	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	externalMetricType             = "External"
	queueLengthStrategyVisibleOnly = "visibleonly"
)

var maxPeekMessages int32 = 32

type azureQueueScaler struct {
	metricType  v2.MetricTargetType
	metadata    azureQueueMetadata
	queueClient *azqueue.QueueClient
	logger      logr.Logger
}

type azureQueueMetadata struct {
	ActivationQueueLength int64  `keda:"name=activationQueueLength, order=triggerMetadata, default=0"`
	QueueName             string `keda:"name=queueName,             order=triggerMetadata"`
	QueueLength           int64  `keda:"name=queueLength,           order=triggerMetadata, default=5"`
	Connection            string `keda:"name=connection,            order=authParams;triggerMetadata;resolvedEnv, optional"`
	AccountName           string `keda:"name=accountName,           order=triggerMetadata, optional"`
	EndpointSuffix        string `keda:"name=endpointSuffix,        order=triggerMetadata, optional"`
	QueueLengthStrategy   string `keda:"name=queueLengthStrategy,   order=triggerMetadata, enum=all;visibleonly, default=all"`
	TriggerIndex          int
}

func NewAzureQueueScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "azure_queue_scaler")

	meta, podIdentity, err := parseAzureQueueMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure queue metadata: %w", err)
	}

	queueClient, err := azure.GetStorageQueueClient(logger, podIdentity, meta.Connection, meta.AccountName, meta.EndpointSuffix, meta.QueueName, config.GlobalHTTPTimeout)
	if err != nil {
		return nil, fmt.Errorf("error creating azure queue client: %w", err)
	}

	return &azureQueueScaler{
		metricType:  metricType,
		metadata:    meta,
		queueClient: queueClient,
		logger:      logger,
	}, nil
}

func parseAzureQueueMetadata(config *scalersconfig.ScalerConfig) (azureQueueMetadata, kedav1alpha1.AuthPodIdentity, error) {
	meta := azureQueueMetadata{}
	err := config.TypedConfig(&meta)
	if err != nil {
		return meta, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("error parsing azure queue metadata: %w", err)
	}

	endpointSuffix, err := azure.ParseAzureStorageEndpointSuffix(config.TriggerMetadata, azure.QueueEndpoint)
	if err != nil {
		return meta, kedav1alpha1.AuthPodIdentity{}, err
	}
	meta.EndpointSuffix = endpointSuffix

	// If the Use AAD Pod Identity is not present, or set to "none"
	// then check for connection string
	switch config.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		// Azure Queue Scaler expects a "connection" parameter in the metadata
		// of the scaler or in a TriggerAuthentication object
		if meta.Connection == "" {
			return meta, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("no connection setting given")
		}
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		// If the Use AAD Pod Identity is present then check account name
		if meta.AccountName == "" {
			return meta, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("no accountName given")
		}
	default:
		return meta, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("pod identity %s not supported for azure storage queues", config.PodIdentity.Provider)
	}

	meta.TriggerIndex = config.TriggerIndex
	return meta, config.PodIdentity, nil
}

func (s *azureQueueScaler) Close(context.Context) error {
	return nil
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureQueueScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("azure-queue-%s", s.metadata.QueueName))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.QueueLength),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *azureQueueScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	queuelen, err := s.getMessageCount(ctx)
	if err != nil {
		s.logger.Error(err, "error getting queue length")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(queuelen))
	return []external_metrics.ExternalMetricValue{metric}, queuelen > s.metadata.ActivationQueueLength, nil
}

func (s *azureQueueScaler) getMessageCount(ctx context.Context) (int64, error) {
	if strings.ToLower(s.metadata.QueueLengthStrategy) == queueLengthStrategyVisibleOnly {
		queue, err := s.queueClient.PeekMessages(ctx, &azqueue.PeekMessagesOptions{NumberOfMessages: &maxPeekMessages})
		if err != nil {
			return 0, err
		}
		visibleMessageCount := len(queue.Messages)

		// Queue has less messages than we allowed to peek for,
		// so no need to fall back to the 'all' strategy
		if visibleMessageCount < int(maxPeekMessages) {
			return int64(visibleMessageCount), nil
		}
	}

	props, err := s.queueClient.GetProperties(ctx, nil)
	if err != nil {
		return 0, err
	}
	return int64(*props.ApproximateMessagesCount), nil
}
