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
	"math"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	eventHubMetricType                 = "External"
	defaultEventHubConsumerGroup       = "$Default"
	defaultBlobContainer               = ""
	defaultCheckpointStrategy          = ""
	defaultStalePartitionInfoThreshold = 10000
)

type azureEventHubScaler struct {
	metricType        v2.MetricTargetType
	metadata          *eventHubMetadata
	eventHubClient    *azeventhubs.ProducerClient
	blobStorageClient *azblob.Client
	logger            logr.Logger
}

type eventHubMetadata struct {
	Threshold                   int64              `keda:"name=unprocessedEventThreshold,          order=triggerMetadata, default=64"`
	ActivationThreshold         int64              `keda:"name=activationUnprocessedEventThreshold,          order=triggerMetadata, default=0"`
	StalePartitionInfoThreshold int64              `keda:"name=stalePartitionInfoThreshold,          order=triggerMetadata, default=10000"`
	EventHubInfo                azure.EventHubInfo `keda:"optional"`
	triggerIndex                int
}

// NewAzureEventHubScaler creates a new scaler for eventHub
func NewAzureEventHubScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "azure_eventhub_scaler")

	parsedMetadata, err := parseAzureEventHubMetadata(logger, config)
	if err != nil {
		return nil, fmt.Errorf("unable to get eventhub metadata: %w", err)
	}

	eventHubClient, err := azure.GetEventHubClient(parsedMetadata.EventHubInfo, logger)
	if err != nil {
		return nil, fmt.Errorf("unable to get eventhub client: %w", err)
	}

	blobStorageClient, err := azure.GetStorageBlobClient(logger, config.PodIdentity, parsedMetadata.EventHubInfo.StorageConnection, parsedMetadata.EventHubInfo.StorageAccountName, parsedMetadata.EventHubInfo.BlobStorageEndpoint, config.GlobalHTTPTimeout)
	if err != nil {
		return nil, fmt.Errorf("unable to get eventhub client: %w", err)
	}

	return &azureEventHubScaler{
		metricType:        metricType,
		metadata:          parsedMetadata,
		eventHubClient:    eventHubClient,
		blobStorageClient: blobStorageClient,
		logger:            logger,
	}, nil
}

// parseAzureEventHubMetadata parses metadata
func parseAzureEventHubMetadata(logger logr.Logger, config *scalersconfig.ScalerConfig) (*eventHubMetadata, error) {
	meta := eventHubMetadata{
		EventHubInfo: azure.EventHubInfo{},
	}

	if err := config.TypedConfig(&meta); err != nil {
		return nil, fmt.Errorf("error parsing azure eventhub metadata: %w", err)
	}

	err := parseCommonAzureEventHubMetadata(config, &meta)
	if err != nil {
		return nil, err
	}

	err = parseAzureEventHubAuthenticationMetadata(logger, config, &meta)
	if err != nil {
		return nil, err
	}

	return &meta, nil
}

func parseCommonAzureEventHubMetadata(config *scalersconfig.ScalerConfig, meta *eventHubMetadata) error {
	serviceBusEndpointSuffixProvider := func(env azure.AzEnvironment) (string, error) {
		return env.ServiceBusEndpointSuffix, nil
	}
	serviceBusEndpointSuffix, err := azure.ParseEnvironmentProperty(config.TriggerMetadata, azure.DefaultEndpointSuffixKey, serviceBusEndpointSuffixProvider)
	if err != nil {
		return err
	}
	meta.EventHubInfo.ServiceBusEndpointSuffix = serviceBusEndpointSuffix

	meta.triggerIndex = config.TriggerIndex

	return nil
}

func parseAzureEventHubAuthenticationMetadata(logger logr.Logger, config *scalersconfig.ScalerConfig, meta *eventHubMetadata) error {
	meta.EventHubInfo.PodIdentity = config.PodIdentity

	switch config.PodIdentity.Provider {
	case "", v1alpha1.PodIdentityProviderNone:
		if len(meta.EventHubInfo.StorageConnection) == 0 {
			return fmt.Errorf("no storage connection string given")
		}

		connection := meta.EventHubInfo.EventHubConnection
		if len(connection) == 0 {
			return fmt.Errorf("no event hub connection string given")
		}

		if !strings.Contains(connection, "EntityPath") {
			eventHubName := meta.EventHubInfo.EventHubName

			if eventHubName == "" {
				return fmt.Errorf("connection string does not contain event hub name, and parameter eventHubName not provided")
			}

			connection = fmt.Sprintf("%s;EntityPath=%s", connection, eventHubName)
		}

		meta.EventHubInfo.EventHubConnection = connection
	case v1alpha1.PodIdentityProviderAzureWorkload:
		if meta.EventHubInfo.StorageAccountName == "" {
			logger.Info("no 'storageAccountName' provided to enable identity based authentication to Blob Storage. Attempting to use connection string instead")
		}

		if len(meta.EventHubInfo.StorageAccountName) != 0 {
			storageEndpointSuffixProvider := func(env azure.AzEnvironment) (string, error) {
				return env.StorageEndpointSuffix, nil
			}
			storageEndpointSuffix, err := azure.ParseEnvironmentProperty(config.TriggerMetadata, azure.DefaultStorageSuffixKey, storageEndpointSuffixProvider)
			if err != nil {
				return err
			}
			meta.EventHubInfo.BlobStorageEndpoint = "blob." + storageEndpointSuffix
		}

		if len(meta.EventHubInfo.StorageConnection) == 0 && len(meta.EventHubInfo.StorageAccountName) == 0 {
			return fmt.Errorf("no storage connection string or storage account name for pod identity based authentication given")
		}

		if len(meta.EventHubInfo.Namespace) == 0 {
			return fmt.Errorf("no event hub namespace string given")
		}

		if len(meta.EventHubInfo.EventHubName) == 0 {
			return fmt.Errorf("no event hub name string given")
		}
	}

	return nil
}

// GetUnprocessedEventCountInPartition gets number of unprocessed events in a given partition
func (s *azureEventHubScaler) GetUnprocessedEventCountInPartition(ctx context.Context, partitionInfo azeventhubs.PartitionProperties) (newEventCount int64, checkpoint azure.Checkpoint, err error) {
	// if partitionInfo.LastEnqueuedSequenceNumber = -1, that means event hub partition is empty
	if partitionInfo.LastEnqueuedSequenceNumber == -1 {
		return 0, azure.Checkpoint{}, nil
	}

	checkpoint, err = azure.GetCheckpointFromBlobStorage(ctx, s.blobStorageClient, s.metadata.EventHubInfo, partitionInfo.PartitionID)
	if err != nil {
		// if blob not found return the total partition event count
		if bloberror.HasCode(err, bloberror.BlobNotFound, bloberror.ContainerNotFound) {
			s.logger.V(1).Error(err, fmt.Sprintf("Blob container : %s not found to use checkpoint strategy, getting unprocessed event count without checkpoint", s.metadata.EventHubInfo.BlobContainer))
			return GetUnprocessedEventCountWithoutCheckpoint(partitionInfo), azure.Checkpoint{}, nil
		}
		return -1, azure.Checkpoint{}, fmt.Errorf("unable to get checkpoint from storage: %w", err)
	}

	unprocessedEventCountInPartition := calculateUnprocessedEvents(partitionInfo, checkpoint, s.metadata.StalePartitionInfoThreshold)

	return unprocessedEventCountInPartition, checkpoint, nil
}

func calculateUnprocessedEvents(partitionInfo azeventhubs.PartitionProperties, checkpoint azure.Checkpoint, stalePartitionInfoThreshold int64) int64 {
	unprocessedEventCount := int64(0)

	if partitionInfo.LastEnqueuedSequenceNumber >= checkpoint.SequenceNumber {
		unprocessedEventCount = partitionInfo.LastEnqueuedSequenceNumber - checkpoint.SequenceNumber
	} else {
		// Partition is a circular buffer, so it is possible that
		// partitionInfo.LastSequenceNumber < blob checkpoint's SequenceNumber

		// Checkpointing may or may not be always behind partition's LastSequenceNumber.
		// The partition information read could be stale compared to checkpoint,
		// especially when load is very small and checkpointing is happening often.
		// This also results in partitionInfo.LastSequenceNumber < blob checkpoint's SequenceNumber
		// e.g., (9223372036854775807 - 15) + 10 = 9223372036854775802

		// Calculate the unprocessed events
		unprocessedEventCount = (math.MaxInt64 - checkpoint.SequenceNumber) + partitionInfo.LastEnqueuedSequenceNumber
	}

	// If the result is greater than the buffer size - stale partition threshold
	// we assume the partition info is stale.
	if unprocessedEventCount > (math.MaxInt64 - stalePartitionInfoThreshold) {
		return 0
	}

	return unprocessedEventCount
}

// GetUnprocessedEventCountWithoutCheckpoint returns the number of messages on the without a checkoutpoint info
func GetUnprocessedEventCountWithoutCheckpoint(partitionInfo azeventhubs.PartitionProperties) int64 {
	// if both values are 0 then there is exactly one message inside the hub. First message after init
	if (partitionInfo.BeginningSequenceNumber == 0 && partitionInfo.LastEnqueuedSequenceNumber == 0) || (partitionInfo.BeginningSequenceNumber != partitionInfo.LastEnqueuedSequenceNumber) {
		return (partitionInfo.LastEnqueuedSequenceNumber - partitionInfo.BeginningSequenceNumber) + 1
	}

	return 0
}

// GetMetricSpecForScaling returns metric spec
func (s *azureEventHubScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("azure-eventhub-%s", s.metadata.EventHubInfo.EventHubConsumerGroup))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.Threshold),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: eventHubMetricType}
	return []v2.MetricSpec{metricSpec}
}

func getTotalLagRelatedToPartitionAmount(unprocessedEventsCount int64, partitionCount int64, threshold int64) int64 {
	if (unprocessedEventsCount / threshold) > partitionCount {
		return partitionCount * threshold
	}

	return unprocessedEventsCount
}

// Close closes Azure Event Hub Scaler
func (s *azureEventHubScaler) Close(ctx context.Context) error {
	if s.eventHubClient != nil {
		err := s.eventHubClient.Close(ctx)
		if err != nil {
			s.logger.Error(err, "error closing azure event hub client")
			return err
		}
	}

	return nil
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureEventHubScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	totalUnprocessedEventCount := int64(0)
	runtimeInfo, err := s.eventHubClient.GetEventHubProperties(ctx, nil)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("unable to get runtimeInfo for metrics: %w", err)
	}

	partitionIDs := runtimeInfo.PartitionIDs

	for i := 0; i < len(partitionIDs); i++ {
		partitionID := partitionIDs[i]
		partitionRuntimeInfo, err := s.eventHubClient.GetPartitionProperties(ctx, partitionID, nil)
		if err != nil {
			return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("unable to get partitionRuntimeInfo for metrics: %w", err)
		}

		unprocessedEventCount := int64(0)

		unprocessedEventCount, checkpoint, err := s.GetUnprocessedEventCountInPartition(ctx, partitionRuntimeInfo)
		if err != nil {
			return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("unable to get unprocessedEventCount for metrics: %w", err)
		}

		totalUnprocessedEventCount += unprocessedEventCount

		s.logger.V(1).Info(fmt.Sprintf("Partition ID: %s, Last SequenceNumber: %d, Checkpoint SequenceNumber: %d, Total new events in partition: %d",
			partitionRuntimeInfo.PartitionID, partitionRuntimeInfo.LastEnqueuedSequenceNumber, checkpoint.SequenceNumber, unprocessedEventCount))
	}

	// set count to max if the sum is negative (Int64 overflow) to prevent negative metric values
	// e.g., 9223372036854775797 (Partition 1) + 20 (Partition 2) = -9223372036854775799
	if totalUnprocessedEventCount < 0 {
		totalUnprocessedEventCount = math.MaxInt64
	}

	// don't scale out beyond the number of partitions
	lagRelatedToPartitionCount := getTotalLagRelatedToPartitionAmount(totalUnprocessedEventCount, int64(len(partitionIDs)), s.metadata.Threshold)

	s.logger.V(1).Info(fmt.Sprintf("Unprocessed events in event hub total: %d, scaling for a lag of %d related to %d partitions", totalUnprocessedEventCount, lagRelatedToPartitionCount, len(partitionIDs)))

	metric := GenerateMetricInMili(metricName, float64(lagRelatedToPartitionCount))

	return []external_metrics.ExternalMetricValue{metric}, totalUnprocessedEventCount > s.metadata.ActivationThreshold, nil
}
