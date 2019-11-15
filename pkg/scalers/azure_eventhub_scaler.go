package scalers

import (
	"context"
	"fmt"
	"math"
	"strconv"

	eventhub "github.com/Azure/azure-event-hubs-go"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	defaultEventHubMessageThreshold  = 64
	eventHubMetricType               = "External"
	thresholdMetricName              = "unprocessedEventThreshold"
	defaultEventHubConsumerGroup     = "$Default"
	defaultEventHubConnectionSetting = "EventHub"
	defaultStorageConnectionSetting  = "AzureWebJobsStorage"
)

var eventhubLog = logf.Log.WithName("azure_eventhub_scaler")

type AzureEventHubScaler struct {
	metadata           *EventHubMetadata
	client             *eventhub.Hub
	storageCredentials *azblob.SharedKeyCredential
}

type EventHubMetadata struct {
	eventHubConnection    string
	eventHubConsumerGroup string
	threshold             int64
	storageConnection     string
}

// NewAzureEventHubScaler creates a new scaler for eventHub
func NewAzureEventHubScaler(resolvedEnv, metadata map[string]string) (Scaler, error) {
	parsedMetadata, err := parseAzureEventHubMetadata(metadata, resolvedEnv)
	if err != nil {
		return nil, fmt.Errorf("unable to get eventhub metadata: %s", err)
	}

	_, cred, err := GetStorageCredentials(parsedMetadata.storageConnection)
	if err != nil {
		return nil, fmt.Errorf("unable to get storage credentials: %s", err)
	}

	hub, err := GetEventHubClient(parsedMetadata.eventHubConnection)
	if err != nil {
		return nil, fmt.Errorf("unable to get eventhub client: %s", err)
	}

	return &AzureEventHubScaler{
		metadata:           parsedMetadata,
		storageCredentials: cred,
		client:             hub,
	}, nil
}

// parseAzureEventHubMetadata parses metadata
func parseAzureEventHubMetadata(metadata, resolvedEnv map[string]string) (*EventHubMetadata, error) {
	meta := EventHubMetadata{}
	meta.threshold = defaultEventHubMessageThreshold

	if val, ok := metadata[thresholdMetricName]; ok {
		threshold, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Error parsing azure eventhub metadata %s: %s", thresholdMetricName, err)
		}

		meta.threshold = threshold
	}

	storageConnectionSetting := defaultStorageConnectionSetting
	if val, ok := metadata["storageConnection"]; ok && val != "" {
		storageConnectionSetting = val
	}

	if val, ok := resolvedEnv[storageConnectionSetting]; ok {
		meta.storageConnection = val
	} else {
		return nil, fmt.Errorf("no storage connection string given")
	}

	eventHubConnectionSetting := defaultEventHubConnectionSetting
	if val, ok := metadata["connection"]; ok && val != "" {
		eventHubConnectionSetting = val
	}

	if val, ok := resolvedEnv[eventHubConnectionSetting]; ok {
		meta.eventHubConnection = val
	} else {
		return nil, fmt.Errorf("no event hub connection string given")
	}

	meta.eventHubConsumerGroup = defaultEventHubConsumerGroup
	if val, ok := metadata["consumerGroup"]; ok {
		meta.eventHubConsumerGroup = val
	}

	return &meta, nil
}

//GetUnprocessedEventCountInPartition gets number of unprocessed events in a given partition
func (scaler *AzureEventHubScaler) GetUnprocessedEventCountInPartition(ctx context.Context, partitionID string) (newEventCount int64, err error) {
	partitionInfo, err := scaler.client.GetPartitionInformation(ctx, partitionID)
	if err != nil {
		return -1, fmt.Errorf("unable to get partition info: %s", err)
	}

	checkpoint, err := GetCheckpointFromBlobStorage(ctx, partitionID, *scaler.metadata)
	if err != nil {
		return -1, fmt.Errorf("unable to get checkpoint from storage: %s", err)
	}

	unprocessedEventCountInPartition := int64(0)

	if checkpoint.SequenceNumber != partitionInfo.LastSequenceNumber {
		if partitionInfo.LastSequenceNumber > checkpoint.SequenceNumber {
			unprocessedEventCountInPartition = partitionInfo.LastSequenceNumber - checkpoint.SequenceNumber

			return unprocessedEventCountInPartition, nil
		}

		unprocessedEventCountInPartition = (math.MaxInt64 - partitionInfo.LastSequenceNumber) + checkpoint.SequenceNumber
	}
	if unprocessedEventCountInPartition < 0 {
		unprocessedEventCountInPartition = 0
	}

	return unprocessedEventCountInPartition, nil
}

// IsActive determines if eventhub is active based on number of unprocessed events
func (scaler *AzureEventHubScaler) IsActive(ctx context.Context) (bool, error) {
	runtimeInfo, err := scaler.client.GetRuntimeInformation(ctx)
	if err != nil {
		eventhubLog.Error(err, "unable to get runtimeInfo for isActive")
		return false, fmt.Errorf("unable to get runtimeInfo for isActive: %s", err)
	}

	partitionIDs := runtimeInfo.PartitionIDs

	for i := 0; i < len(partitionIDs); i++ {
		partitionID := partitionIDs[i]

		unprocessedEventCount, err := scaler.GetUnprocessedEventCountInPartition(ctx, partitionID)

		if err != nil {
			return false, fmt.Errorf("unable to get unprocessedEventCount for isActive: %s", err)
		}

		if unprocessedEventCount > 0 {
			return true, nil
		}
	}

	return false, nil
}

// GetMetricSpecForScaling returns metric spec
func (scaler *AzureEventHubScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	return []v2beta1.MetricSpec{
		{
			External: &v2beta1.ExternalMetricSource{
				MetricName:         thresholdMetricName,
				TargetAverageValue: resource.NewQuantity(scaler.metadata.threshold, resource.DecimalSI),
			},
			Type: eventHubMetricType,
		},
	}
}

// GetMetrics returns metric using total number of unprocessed events in event hub
func (scaler *AzureEventHubScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	totalUnprocessedEventCount := int64(0)
	runtimeInfo, err := scaler.client.GetRuntimeInformation(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("unable to get runtimeInfo for metrics: %s", err)
	}

	partitionIDs := runtimeInfo.PartitionIDs

	for i := 0; i < len(partitionIDs); i++ {
		partitionID := partitionIDs[i]
		partitionRuntimeInfo, err := scaler.client.GetPartitionInformation(ctx, partitionID)
		if err != nil {
			return []external_metrics.ExternalMetricValue{}, fmt.Errorf("unable to get partitionRuntimeInfo for metrics: %s", err)
		}

		checkpoint, err := GetCheckpointFromBlobStorage(ctx, partitionID, *scaler.metadata)
		if err != nil {
			return []external_metrics.ExternalMetricValue{}, fmt.Errorf("unable to get checkpoint from storage: %s", err)
		}

		unprocessedEventCount := int64(0)

		unprocessedEventCount, err = scaler.GetUnprocessedEventCountInPartition(ctx, partitionID)
		if err != nil {
			return []external_metrics.ExternalMetricValue{}, fmt.Errorf("unable to get unprocessedEventCount for metrics: %s", err)
		}

		totalUnprocessedEventCount += unprocessedEventCount

		eventhubLog.V(1).Info(fmt.Sprintf("Partition ID: %s, Last Enqueued Offset: %s, Checkpoint Offset: %s, Total new events in partition: %d",
			partitionRuntimeInfo.PartitionID, partitionRuntimeInfo.LastEnqueuedOffset, checkpoint.Offset, unprocessedEventCount))
	}

	eventhubLog.V(1).Info(fmt.Sprintf("Scaling for %d total unprocessed events in event hub", totalUnprocessedEventCount))

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(totalUnprocessedEventCount, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// Close closes Azure Event Hub Scaler
func (scaler *AzureEventHubScaler) Close() error {
	return nil
}
