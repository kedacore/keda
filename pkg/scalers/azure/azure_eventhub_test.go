package azure

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/go-playground/assert/v2"
)

// Add a valid Storage account connection string here
const StorageConnectionString = ""

func TestCheckpointFromBlobStorageAzureFunction(t *testing.T) {
	if StorageConnectionString == "" {
		return
	}

	partitionID := "0"
	offset := "1001"
	consumerGroup := "$Default1"

	sequencenumber := int64(1)

	containerName := "azure-webjobs-eventhub"
	checkpointFormat := "{\"Offset\":\"%s\",\"SequenceNumber\":%d,\"PartitionId\":\"%s\",\"Owner\":\"\",\"Token\":\"\",\"Epoch\":0}"
	checkpoint := fmt.Sprintf(checkpointFormat, offset, sequencenumber, partitionID)
	urlPath := fmt.Sprintf("eventhubnamespace.servicebus.windows.net/hub/%s/", consumerGroup)

	ctx, err := createNewCheckpointInStorage(urlPath, containerName, partitionID, checkpoint, nil)
	assert.Equal(t, err, nil)

	expectedCheckpoint := Checkpoint{
		baseCheckpoint: baseCheckpoint{
			Offset: offset,
		},
		PartitionID:    partitionID,
		SequenceNumber: sequencenumber,
	}

	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub",
		StorageConnection:     StorageConnectionString,
		EventHubConsumerGroup: consumerGroup,
		EventHubName:          "hub",
	}

	check, _ := GetCheckpointFromBlobStorage(ctx, http.DefaultClient, eventHubInfo, "0")
	_ = check.Offset
	_ = expectedCheckpoint.Offset
	assert.Equal(t, check, expectedCheckpoint)
}

func TestCheckpointFromBlobStorageDefault(t *testing.T) {
	if StorageConnectionString == "" {
		return
	}

	partitionID := "1"
	offset := "1005"
	consumerGroup := "$Default2"

	sequencenumber := int64(1)

	containerName := "defaultcontainer"
	checkpointFormat := "{\"Offset\":\"%s\",\"SequenceNumber\":%d,\"PartitionId\":\"%s\",\"Owner\":\"\",\"Token\":\"\",\"Epoch\":0}"
	checkpoint := fmt.Sprintf(checkpointFormat, offset, sequencenumber, partitionID)
	urlPath := fmt.Sprintf("%s/", consumerGroup)

	ctx, err := createNewCheckpointInStorage(urlPath, containerName, partitionID, checkpoint, nil)
	assert.Equal(t, err, nil)

	expectedCheckpoint := Checkpoint{
		baseCheckpoint: baseCheckpoint{
			Offset: offset,
		},
		PartitionID:    partitionID,
		SequenceNumber: sequencenumber,
	}

	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub",
		StorageConnection:     StorageConnectionString,
		EventHubConsumerGroup: consumerGroup,
		EventHubName:          "hub",
		BlobContainer:         containerName,
	}

	check, _ := GetCheckpointFromBlobStorage(ctx, http.DefaultClient, eventHubInfo, partitionID)
	_ = check.Offset
	_ = expectedCheckpoint.Offset
	assert.Equal(t, check, expectedCheckpoint)
}

func TestCheckpointFromBlobStorageDefaultDeprecatedPythonCheckpoint(t *testing.T) {
	if StorageConnectionString == "" {
		return
	}

	partitionID := "2"
	offset := "1006"
	consumerGroup := "$Default3"

	sequencenumber := int64(1)

	containerName := "defaultcontainerpython"
	checkpointFormat := "{\"Offset\":\"%s\",\"sequence_number\":%d,\"partition_id\":\"%s\",\"Owner\":\"\",\"Token\":\"\",\"Epoch\":0}"
	checkpoint := fmt.Sprintf(checkpointFormat, offset, sequencenumber, partitionID)
	urlPath := fmt.Sprintf("%s/", consumerGroup)

	ctx, err := createNewCheckpointInStorage(urlPath, containerName, partitionID, checkpoint, nil)
	assert.Equal(t, err, nil)

	expectedCheckpoint := Checkpoint{
		baseCheckpoint: baseCheckpoint{
			Offset: offset,
		},
		PartitionID:    partitionID,
		SequenceNumber: sequencenumber,
	}

	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub",
		StorageConnection:     StorageConnectionString,
		EventHubConsumerGroup: consumerGroup,
		EventHubName:          "hub",
		BlobContainer:         containerName,
	}

	check, _ := GetCheckpointFromBlobStorage(ctx, http.DefaultClient, eventHubInfo, partitionID)
	_ = check.Offset
	_ = expectedCheckpoint.Offset
	assert.Equal(t, check, expectedCheckpoint)
}

func TestCheckpointFromBlobStorageWithBlobMetadata(t *testing.T) {
	if StorageConnectionString == "" {
		return
	}

	partitionID := "4"
	offset := "1002"
	consumerGroup := "$default"

	sequencenumber := int64(1)

	metadata := map[string]string{
		"offset":         offset,
		"sequencenumber": strconv.FormatInt(sequencenumber, 10),
	}

	containerName := "blobmetadatacontainer"
	urlPath := fmt.Sprintf("eventhubnamespace.servicebus.windows.net/hub/%s/checkpoint/", consumerGroup)

	ctx, err := createNewCheckpointInStorage(urlPath, containerName, partitionID, "", metadata)
	assert.Equal(t, err, nil)

	expectedCheckpoint := Checkpoint{
		baseCheckpoint: baseCheckpoint{
			Offset: offset,
		},
		PartitionID:    partitionID,
		SequenceNumber: sequencenumber,
	}

	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub",
		StorageConnection:     StorageConnectionString,
		EventHubConsumerGroup: consumerGroup,
		EventHubName:          "hub",
		BlobContainer:         containerName,
		CheckpointStrategy:    "blobMetadata",
	}

	check, _ := GetCheckpointFromBlobStorage(ctx, http.DefaultClient, eventHubInfo, partitionID)
	_ = check.Offset
	_ = expectedCheckpoint.Offset
	assert.Equal(t, check, expectedCheckpoint)
}

func TestCheckpointFromBlobStorageGoSdk(t *testing.T) {
	if StorageConnectionString == "" {
		return
	}

	partitionID := "0"
	offset := "1003"

	sequencenumber := int64(1)

	containerName := "gosdkcontainer"
	checkpointFormat := "{\"partitionID\":\"%s\",\"epoch\":0,\"owner\":\"\",\"checkpoint\":{\"offset\":\"%s\",\"sequenceNumber\":%d,\"enqueueTime\":\"\"},\"state\":\"\",\"token\":\"\"}"
	checkpoint := fmt.Sprintf(checkpointFormat, partitionID, offset, sequencenumber)

	urlPath := ""

	ctx, err := createNewCheckpointInStorage(urlPath, containerName, partitionID, checkpoint, nil)
	assert.Equal(t, err, nil)

	expectedCheckpoint := Checkpoint{
		baseCheckpoint: baseCheckpoint{
			Offset: offset,
		},
		PartitionID:    partitionID,
		SequenceNumber: sequencenumber,
	}

	eventHubInfo := EventHubInfo{
		EventHubConnection: "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub",
		StorageConnection:  StorageConnectionString,
		EventHubName:       "hub",
		BlobContainer:      containerName,
		CheckpointStrategy: "goSdk",
	}

	check, _ := GetCheckpointFromBlobStorage(ctx, http.DefaultClient, eventHubInfo, partitionID)
	_ = check.Offset
	_ = expectedCheckpoint.Offset
	assert.Equal(t, check, expectedCheckpoint)
}

func TestShouldParseCheckpointForFunction(t *testing.T) {
	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub-test",
		EventHubConsumerGroup: "$Default",
	}

	cp := newCheckpointer(eventHubInfo, "0")
	url, _ := cp.resolvePath(eventHubInfo)

	assert.Equal(t, url.Path, "/azure-webjobs-eventhub/eventhubnamespace.servicebus.windows.net/hub-test/$Default/0")
}

func TestShouldParseCheckpointForFunctionWithCheckpointStrategy(t *testing.T) {
	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub-test",
		EventHubConsumerGroup: "$Default",
		CheckpointStrategy:    "azureFunction",
	}

	cp := newCheckpointer(eventHubInfo, "0")
	url, _ := cp.resolvePath(eventHubInfo)

	assert.Equal(t, url.Path, "/azure-webjobs-eventhub/eventhubnamespace.servicebus.windows.net/hub-test/$Default/0")
}

func TestShouldParseCheckpointForDefault(t *testing.T) {
	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub-test",
		EventHubConsumerGroup: "$Default",
		BlobContainer:         "DefaultContainer",
	}

	cp := newCheckpointer(eventHubInfo, "0")
	url, _ := cp.resolvePath(eventHubInfo)

	assert.Equal(t, url.Path, "/DefaultContainer/$Default/0")
}

func TestShouldParseCheckpointForBlobMetadata(t *testing.T) {
	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub-test",
		EventHubConsumerGroup: "$Default",
		BlobContainer:         "containername",
		CheckpointStrategy:    "blobMetadata",
	}

	cp := newCheckpointer(eventHubInfo, "0")
	url, _ := cp.resolvePath(eventHubInfo)

	assert.Equal(t, url.Path, "/containername/eventhubnamespace.servicebus.windows.net/hub-test/$default/checkpoint/0")
}

func TestShouldParseCheckpointForGoSdk(t *testing.T) {
	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub-test",
		EventHubConsumerGroup: "$Default",
		BlobContainer:         "containername",
		CheckpointStrategy:    "goSdk",
	}

	cp := newCheckpointer(eventHubInfo, "0")
	url, _ := cp.resolvePath(eventHubInfo)

	assert.Equal(t, url.Path, "/containername/0")
}

func createNewCheckpointInStorage(urlPath string, containerName string, partitionID string, checkpoint string, metadata map[string]string) (context.Context, error) {
	credential, endpoint, _ := ParseAzureStorageBlobConnection(http.DefaultClient, "none", StorageConnectionString, "", "")

	// Create container
	ctx := context.Background()
	path, _ := url.Parse(containerName)
	url := endpoint.ResolveReference(path)
	containerURL := azblob.NewContainerURL(*url, azblob.NewPipeline(credential, azblob.PipelineOptions{}))
	_, err := containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessNone)

	err = errors.Unwrap(err)
	if err != nil {
		if stErr, ok := err.(azblob.StorageError); ok {
			if stErr.ServiceCode() == azblob.ServiceCodeContainerAlreadyExists {
				return ctx, fmt.Errorf("failed to create container: %s", err)
			}
		}
	}

	blobFolderURL := containerURL.NewBlockBlobURL(urlPath + partitionID)

	var b bytes.Buffer
	b.WriteString(checkpoint)

	// Upload file
	_, err = azblob.UploadBufferToBlockBlob(ctx, b.Bytes(), blobFolderURL, azblob.UploadToBlockBlobOptions{
		BlockSize:   4 * 1024 * 1024,
		Metadata:    metadata,
		Parallelism: 16})
	if err != nil {
		return ctx, fmt.Errorf("Err uploading file to blob: %s", err)
	}
	return ctx, nil
}
