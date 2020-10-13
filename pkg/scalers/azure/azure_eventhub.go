package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/imdario/mergo"

	"github.com/Azure/azure-amqp-common-go/v3/aad"
	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/azure"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/util"
)

type baseCheckpoint struct {
	Epoch  int64  `json:"Epoch"`
	Offset string `json:"Offset"`
	Owner  string `json:"Owner"`
	Token  string `json:"Token"`
}

// Checkpoint is the object eventhub processor stores in storage
// for checkpointing event processors. This matches the object
// stored by the eventhub C# sdk and Java sdk
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

// EventHubInfo to keep event hub connection and resources
type EventHubInfo struct {
	EventHubConnection    string
	EventHubConsumerGroup string
	StorageConnection     string
	BlobContainer         string
	Namespace             string
	EventHubName          string
}

// GetEventHubClient returns eventhub client
func GetEventHubClient(info EventHubInfo) (*eventhub.Hub, error) {
	// The user wants to use a connectionstring, not a pod identity
	if info.EventHubConnection != "" {
		hub, err := eventhub.NewHubFromConnectionString(info.EventHubConnection)
		if err != nil {
			return nil, fmt.Errorf("failed to create hub client: %s", err)
		}
		return hub, nil
	}

	// Since there is no connectionstring, then user wants to use pod identity
	// Internally, the JWTProvider will use Managed Service Identity to authenticate if no Service Principal info supplied
	provider, aadErr := aad.NewJWTProvider(func(config *aad.TokenProviderConfiguration) error {
		if config.Env == nil {
			config.Env = &azure.PublicCloud
		}
		return nil
	})

	if aadErr == nil {
		return eventhub.NewHub(info.Namespace, info.EventHubName, provider)
	}

	return nil, aadErr
}

// GetCheckpointFromBlobStorage accesses Blob storage and gets checkpoint information of a partition
func GetCheckpointFromBlobStorage(ctx context.Context, httpClient util.HTTPDoer, info EventHubInfo, partitionID string) (Checkpoint, error) {
	blobCreds, storageEndpoint, err := ParseAzureStorageBlobConnection(httpClient, kedav1alpha1.PodIdentityProviderNone, info.StorageConnection, "")
	if err != nil {
		return Checkpoint{}, err
	}

	var eventHubNamespace string
	var eventHubName string
	if info.EventHubConnection != "" {
		eventHubNamespace, eventHubName, err = ParseAzureEventHubConnectionString(info.EventHubConnection)
		if err != nil {
			return Checkpoint{}, err
		}
	} else {
		eventHubNamespace = info.Namespace
		eventHubName = info.EventHubName
	}

	// TODO: add more ways to read from different types of storage and read checkpoints/leases written in different JSON formats
	var baseURL *url.URL
	// Checking blob store for C# and Java applications
	if info.BlobContainer != "" {
		// URL format - <storageEndpoint>/<blobContainer>/<eventHubConsumerGroup>/<partitionID>
		path, _ := url.Parse(fmt.Sprintf("/%s/%s/%s", info.BlobContainer, info.EventHubConsumerGroup, partitionID))
		baseURL = storageEndpoint.ResolveReference(path)
	} else {
		// Checking blob store for Azure functions
		// URL format - <storageEndpoint>/azure-webjobs-eventhub/<eventHubNamespace>/<eventHubName>/<eventHubConsumerGroup>/<partitionID>
		path, _ := url.Parse(fmt.Sprintf("/azure-webjobs-eventhub/%s/%s/%s/%s", eventHubNamespace, eventHubName, info.EventHubConsumerGroup, partitionID))
		baseURL = storageEndpoint.ResolveReference(path)
	}

	// Create a BlockBlobURL object to a blob in the container.
	blobURL := azblob.NewBlockBlobURL(*baseURL, azblob.NewPipeline(blobCreds, azblob.PipelineOptions{}))

	get, err := blobURL.Download(ctx, 0, 0, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return Checkpoint{}, fmt.Errorf("unable to download file from blob storage: %w", err)
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
		return "", "", errors.New("can't parse event hub connection string. Missing eventHubNamespace or eventHubName")
	}

	return eventHubNamespace, eventHubName, nil
}
