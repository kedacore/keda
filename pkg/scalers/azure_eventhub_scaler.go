package scalers

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	eventhub "github.com/Azure/azure-event-hubs-go"
	storageLeaser "github.com/Azure/azure-event-hubs-go/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	log "github.com/Sirupsen/logrus"
	"k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	defaultEventHubMessageThreshold = 10
	eventHubMetricType              = "External"
	thresholdMetricName             = "unprocessedEventThreshold"
	defaultEventHubConnectionString = "EventHub"
	defaultStorageConnectionString  = "AzureWebJobsStorage"
)

type Lease struct {
	PartitionID string `json:"partitionID"`
	Epoch       int    `json:"epoch"`
	Owner       string `json:"owner"`
	Checkpoint  struct {
		Offset         string    `json:"offset"`
		SequenceNumber int64     `json:"sequenceNumber"`
		EnqueueTime    time.Time `json:"enqueueTime"`
	} `json:"checkpoint"`
	State string `json:"state"`
	Token string `json:"token"`
}

type AzureEventHubScaler struct {
	metadata           *EventHubMetadata
	client             *eventhub.Hub
	storageCredentials *azblob.SharedKeyCredential
	leaserCheckpointer *storageLeaser.LeaserCheckpointer
}

type EventHubMetadata struct {
	eventHubConnection   string
	threshold            int64
	eventHubName         string
	storageConnection    string
	storageContainerName string
}

// NewAzureEventHubScaler creates a new scaler for eventHub
func NewAzureEventHubScaler(resolvedEnv, metadata map[string]string) (Scaler, error) {
	eventHubScaler := AzureEventHubScaler{}

	parsedMetadata, err := ParseAzureEventHubMetadata(metadata, resolvedEnv)
	if err != nil {
		return &AzureEventHubScaler{}, fmt.Errorf("unable to get eventhub metadata: %s", err)
	}
	eventHubScaler.metadata = parsedMetadata

	_, cred, err := GetStorageCredentials(parsedMetadata.storageConnection)
	if err != nil {
		return &AzureEventHubScaler{}, fmt.Errorf("unable to get storage credentials: %s", err)
	}
	eventHubScaler.storageCredentials = cred

	hub, err := GetEventHubClient(parsedMetadata.eventHubConnection)
	if err != nil {
		return &AzureEventHubScaler{}, fmt.Errorf("unable to get eventhub client: %s", err)
	}
	eventHubScaler.client = hub

	leaserCheckpointer, err := GetLeaserCheckpointer(parsedMetadata.storageConnection, parsedMetadata.storageContainerName)
	if err != nil {
		return &AzureEventHubScaler{}, fmt.Errorf("unable to get leaser/checkpointer: %s", err)
	}
	eventHubScaler.leaserCheckpointer = leaserCheckpointer

	return &eventHubScaler, nil
}

// ParseAzureEventHubMetadata parses metadata
func ParseAzureEventHubMetadata(metadata, resolvedEnv map[string]string) (*EventHubMetadata, error) {
	meta := EventHubMetadata{}
	meta.threshold = defaultEventHubMessageThreshold

	if val, ok := metadata[thresholdMetricName]; ok {
		threshold, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Error parsing azure eventhub metadata %s: %s", thresholdMetricName, err)
		} else {
			meta.threshold = threshold
		}
	}

	storageConnectionSetting := defaultStorageConnectionString
	if val, ok := metadata["storageConnection"]; ok && val != "" {
		storageConnectionSetting = val
	} else {
		return nil, fmt.Errorf("no storage connection setting given")
	}

	if val, ok := resolvedEnv[storageConnectionSetting]; ok {
		meta.storageConnection = val
	} else {
		return nil, fmt.Errorf("no storage connection setting given")
	}

	eventHubConnectionSetting := defaultEventHubConnectionString
	if val, ok := metadata["eventHubConnection"]; ok && val != "" {
		eventHubConnectionSetting = val
	} else {
		return nil, fmt.Errorf("no event hub connection setting given")
	}

	if val, ok := resolvedEnv[eventHubConnectionSetting]; ok {
		meta.eventHubConnection = val
	} else {
		return nil, fmt.Errorf("no event hub connection setting given")
	}

	if val, ok := metadata["eventHubName"]; ok {
		meta.eventHubName = val
	} else {
		return nil, fmt.Errorf("no eventHubName given")
	}

	if val, ok := metadata["storageContainerName"]; ok {
		meta.storageContainerName = val
	} else {
		return nil, fmt.Errorf("no storageContainerName given")
	}

	return &meta, nil
}

//GetUnprocessedEventCountInPartition gets number of unprocessed events in a given partition
func (scaler *AzureEventHubScaler) GetUnprocessedEventCountInPartition(ctx context.Context, partitionID string) (newEventCount int64, err error) {
	partitionInfo, err := scaler.client.GetPartitionInformation(ctx, partitionID)
	if err != nil {
		return -1, fmt.Errorf("unable to get partition info: %s", err)
	}

	lease, err := GetLeaseFromBlobStorage(ctx, partitionID, scaler.metadata.storageConnection, scaler.metadata.storageContainerName)
	if err != nil {
		return -1, fmt.Errorf("unable to get lease from storage: %s", err)
	}

	checkpoint := lease.Checkpoint

	unprocessedEventCountInPartition := int64(0)

	if checkpoint.SequenceNumber != partitionInfo.LastSequenceNumber {
		if partitionInfo.LastSequenceNumber > checkpoint.SequenceNumber {
			unprocessedEventCountInPartition = partitionInfo.LastSequenceNumber - checkpoint.SequenceNumber
			return unprocessedEventCountInPartition, nil
		} else {
			unprocessedEventCountInPartition = (math.MaxInt64 - partitionInfo.LastSequenceNumber) + checkpoint.SequenceNumber
		}
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
		log.Errorf("unable to get runtimeInfo for isActive: %s", err)
	}

	partitionIDs := runtimeInfo.PartitionIDs

	for i := 0; i < len(partitionIDs); i++ {
		partitionID := partitionIDs[i]

		unprocessedEventCount, err := scaler.GetUnprocessedEventCountInPartition(ctx, partitionID)

		if err != nil {
			log.Errorf("unable to get unprocessedEventCount for isActive: %s", err)
		}

		if unprocessedEventCount > 0 {
			return true, nil
		}
	}

	return false, nil
}

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

		lease, err := GetLeaseFromBlobStorage(ctx, partitionID, scaler.metadata.storageConnection, scaler.metadata.storageContainerName)
		if err != nil {
			return []external_metrics.ExternalMetricValue{}, fmt.Errorf("unable to get lease from storage: %s", err)
		}

		unprocessedEventCount := int64(0)

		unprocessedEventCount, err = scaler.GetUnprocessedEventCountInPartition(ctx, partitionID)
		if err != nil {
			return []external_metrics.ExternalMetricValue{}, fmt.Errorf("unable to get unprocessedEventCount for metrics: %s", err)
		}

		totalUnprocessedEventCount += unprocessedEventCount

		log.Debugf("Partition ID: %s, Last Enqueued Offset: %s, Checkpoint Offset: %s, Total new events in partition: %s",
			partitionRuntimeInfo.PartitionID, partitionRuntimeInfo.LastEnqueuedOffset, lease.Checkpoint.Offset, strconv.FormatInt(unprocessedEventCount, 10))
	}

	log.Debugf("Scaling for %s total unprocessed events in event hub %s", strconv.FormatInt(totalUnprocessedEventCount, 10), scaler.metadata.eventHubName)

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(totalUnprocessedEventCount, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// Close closes eventhub client and leasercheckpointer
func (scaler *AzureEventHubScaler) Close() error {
	return nil
}
