package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/imdario/mergo"

	eventhub "github.com/Azure/azure-event-hubs-go"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/azure"
)

const (
	environmentName = "AzurePublicCloud"
)

type baseCheckpoint struct {
	Epoch  int64  `json:"Epoch"`
	Offset string `json:"Offset"`
	Owner  string `json:"Owner"`
	Token  string `json:"Token"`
}

// Checkpoint is the object eventhub processor stores in storage
// for checkpointing event processors. This matches the object
// stored by the eventhub C# sdk
type Checkpoint struct {
	baseCheckpoint
	PartitionID    string `json:"PartitionId"`
	SequenceNumber int64  `json:"SequenceNumber"`
}

// Eventhub python sdk stores the checkpoint differently
type pythonCheckpoint struct {
	baseCheckpoint
	PartitionID    string `json:"partition_id"`
	SequenceNumber int64  `json:"sequence_number"`
}

// GetStorageCredentials returns azure env and storage credentials
func GetStorageCredentials(storageConnection string) (azure.Environment, *azblob.SharedKeyCredential, error) {
	_, storageAccountName, storageAccountKey, _, err := ParseAzureStorageConnectionString(storageConnection)
	if err != nil {
		return azure.Environment{}, &azblob.SharedKeyCredential{}, fmt.Errorf("unable to parse connection string: %s", storageConnection)
	}

	azureEnv, err := azure.EnvironmentFromName(environmentName)
	if err != nil {
		return azureEnv, nil, fmt.Errorf("could not get azure.Environment struct: %s", err)
	}

	cred, err := azblob.NewSharedKeyCredential(storageAccountName, storageAccountKey)
	if err != nil {
		return azureEnv, nil, fmt.Errorf("could not prepare a blob storage credential: %s", err)
	}

	return azureEnv, cred, nil
}

// GetEventHubClient returns eventhub client
func GetEventHubClient(connectionString string) (*eventhub.Hub, error) {
	hub, err := eventhub.NewHubFromConnectionString(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to create hub client: %s", err)
	}

	return hub, nil
}

// GetCheckpointFromBlobStorage accesses Blob storage and gets checkpoint information of a partition
func GetCheckpointFromBlobStorage(ctx context.Context, partitionID string, eventHubMetadata EventHubMetadata) (Checkpoint, error) {
	endpointProtocol, storageAccountName, _, endpointSuffix, err := ParseAzureStorageConnectionString(eventHubMetadata.storageConnection)
	if err != nil {
		return Checkpoint{}, fmt.Errorf("unable to parse storage connection string: %s", err)
	}

	eventHubNamespace, eventHubName, err := ParseAzureEventHubConnectionString(eventHubMetadata.eventHubConnection)
	if err != nil {
		return Checkpoint{}, fmt.Errorf("unable to parse event hub connection string: %s", err)
	}

	// TODO: add more ways to read from different types of storage and read checkpoints/leases written in different JSON formats
	u, _ := url.Parse(fmt.Sprintf("%s://%s.blob.%s/azure-webjobs-eventhub/%s/%s/%s/%s", endpointProtocol, storageAccountName, endpointSuffix, eventHubNamespace, eventHubName, eventHubMetadata.eventHubConsumerGroup, partitionID))

	_, cred, err := GetStorageCredentials(eventHubMetadata.storageConnection)
	if err != nil {
		return Checkpoint{}, fmt.Errorf("unable to get storage credentials: %s", err)
	}

	// Create a BlockBlobURL object to a blob in the container.
	blobURL := azblob.NewBlockBlobURL(*u, azblob.NewPipeline(cred, azblob.PipelineOptions{}))

	get, err := blobURL.Download(ctx, 0, 0, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return Checkpoint{}, fmt.Errorf("unable to download file from blob storage: %s", err)
	}

	blobData := &bytes.Buffer{}
	reader := get.Body(azblob.RetryReaderOptions{})
	if _, err := blobData.ReadFrom(reader); err != nil {
		return Checkpoint{}, fmt.Errorf("failed to read blob data: %s", err)
	}
	defer reader.Close() // The client must close the response body when finished with it

	return getCheckpoint(blobData.Bytes())
}

func getCheckpoint(bytes []byte) (Checkpoint, error) {
	var checkpoint Checkpoint
	var pyCheckpoint pythonCheckpoint

	if err := json.Unmarshal(bytes, &checkpoint); err != nil {
		return Checkpoint{}, fmt.Errorf("failed to decode blob data: %s", err)
	}

	if err := json.Unmarshal(bytes, &pyCheckpoint); err != nil {
		return Checkpoint{}, fmt.Errorf("failed to decode blob data: %s", err)
	}

	err := mergo.Merge(&checkpoint, Checkpoint(pyCheckpoint))

	return checkpoint, err
}

// ParseAzureEventHubConnectionString parses Event Hub connection string into (namespace, name)
// Connection string should be in following format:
// Endpoint=sb://eventhub-namespace.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=secretKey123;EntityPath=eventhub-name
func ParseAzureEventHubConnectionString(connectionString string) (string, string, error) {
	parts := strings.Split(connectionString, ";")

	var eventHubNamespace, eventHubName string
	for _, v := range parts {
		if strings.HasPrefix(v, "Endpoint") {
			endpointParts := strings.SplitN(v, "=", 2)
			if len(endpointParts) == 2 {
				endpointParts[1] = strings.TrimPrefix(endpointParts[1], "sb://")
				endpointParts[1] = strings.TrimSuffix(endpointParts[1], "/")
				eventHubNamespace = endpointParts[1]
			}
		} else if strings.HasPrefix(v, "EntityPath") {
			entityPathParts := strings.SplitN(v, "=", 2)
			if len(entityPathParts) == 2 {
				eventHubName = entityPathParts[1]
			}
		}
	}

	if eventHubNamespace == "" || eventHubName == "" {
		return "", "", errors.New("Can't parse event hub connection string")
	}

	return eventHubNamespace, eventHubName, nil
}
