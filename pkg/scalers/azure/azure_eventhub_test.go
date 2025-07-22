package azure

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

// Add a valid Storage account connection string here
const StorageConnectionString = ""

func TestCheckpointFromBlobStorageAzureFunction(t *testing.T) {
	if StorageConnectionString == "" {
		return
	}

	ctx := context.Background()
	partitionID := "0"
	consumerGroup := "$Default1"

	sequencenumber := int64(1)

	containerName := "azure-webjobs-eventhub"
	checkpointFormat := "{\"SequenceNumber\":%d,\"PartitionId\":\"%s\",\"Owner\":\"\",\"Token\":\"\",\"Epoch\":0}"
	checkpoint := fmt.Sprintf(checkpointFormat, sequencenumber, partitionID)
	urlPath := fmt.Sprintf("eventhubnamespace.servicebus.windows.net/hub/%s/%s", consumerGroup, partitionID)
	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub",
		StorageConnection:     StorageConnectionString,
		EventHubConsumerGroup: consumerGroup,
		EventHubName:          "hub",
	}

	client, err := GetStorageBlobClient(logr.Discard(), eventHubInfo.PodIdentity, eventHubInfo.StorageConnection, eventHubInfo.StorageAccountName, eventHubInfo.BlobStorageEndpoint, 3*time.Second)
	assert.NoError(t, err, "error creating the blob client")

	err = createNewCheckpointInStorage(ctx, client, containerName, urlPath, checkpoint, nil)
	assert.NoError(t, err, "error creating checkpoint")

	expectedCheckpoint := Checkpoint{
		PartitionID:    partitionID,
		SequenceNumber: sequencenumber,
	}

	check, _ := GetCheckpointFromBlobStorage(ctx, client, eventHubInfo, "0")
	assert.Equal(t, check, expectedCheckpoint)
}

func TestCheckpointFromBlobStorageDefault(t *testing.T) {
	if StorageConnectionString == "" {
		return
	}

	ctx := context.Background()
	partitionID := "1"
	consumerGroup := "$Default2"

	sequencenumber := int64(1)

	containerName := "defaultcontainer"
	checkpointFormat := "{\"SequenceNumber\":%d,\"PartitionId\":\"%s\",\"Owner\":\"\",\"Token\":\"\",\"Epoch\":0}"
	checkpoint := fmt.Sprintf(checkpointFormat, sequencenumber, partitionID)
	urlPath := fmt.Sprintf("%s/%s", consumerGroup, partitionID)

	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub",
		StorageConnection:     StorageConnectionString,
		EventHubConsumerGroup: consumerGroup,
		EventHubName:          "hub",
		BlobContainer:         containerName,
	}
	client, err := GetStorageBlobClient(logr.Discard(), eventHubInfo.PodIdentity, eventHubInfo.StorageConnection, eventHubInfo.StorageAccountName, eventHubInfo.BlobStorageEndpoint, 3*time.Second)
	assert.NoError(t, err, "error creating the blob client")

	err = createNewCheckpointInStorage(ctx, client, containerName, urlPath, checkpoint, nil)
	assert.NoError(t, err, "error creating checkpoint")

	expectedCheckpoint := Checkpoint{
		PartitionID:    partitionID,
		SequenceNumber: sequencenumber,
	}

	check, _ := GetCheckpointFromBlobStorage(ctx, client, eventHubInfo, partitionID)
	assert.Equal(t, check, expectedCheckpoint)
}

func TestCheckpointFromBlobStorageDefaultDeprecatedPythonCheckpoint(t *testing.T) {
	if StorageConnectionString == "" {
		return
	}

	ctx := context.Background()
	partitionID := "2"
	consumerGroup := "$Default3"

	sequencenumber := int64(1)

	containerName := "defaultcontainerpython"
	checkpointFormat := "{\"sequence_number\":%d,\"partition_id\":\"%s\",\"Owner\":\"\",\"Token\":\"\",\"Epoch\":0}"
	checkpoint := fmt.Sprintf(checkpointFormat, sequencenumber, partitionID)
	urlPath := fmt.Sprintf("%s/%s", consumerGroup, partitionID)

	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub",
		StorageConnection:     StorageConnectionString,
		EventHubConsumerGroup: consumerGroup,
		EventHubName:          "hub",
		BlobContainer:         containerName,
	}

	client, err := GetStorageBlobClient(logr.Discard(), eventHubInfo.PodIdentity, eventHubInfo.StorageConnection, eventHubInfo.StorageAccountName, eventHubInfo.BlobStorageEndpoint, 3*time.Second)
	assert.NoError(t, err, "error creating the blob client")

	err = createNewCheckpointInStorage(ctx, client, containerName, urlPath, checkpoint, nil)
	assert.NoError(t, err, "error creating checkpoint")

	expectedCheckpoint := Checkpoint{
		PartitionID:    partitionID,
		SequenceNumber: sequencenumber,
	}

	check, _ := GetCheckpointFromBlobStorage(ctx, client, eventHubInfo, partitionID)
	assert.Equal(t, check, expectedCheckpoint)
}

func TestCheckpointFromBlobStorageWithBlobMetadata(t *testing.T) {
	if StorageConnectionString == "" {
		return
	}

	ctx := context.Background()
	partitionID := "4"
	consumerGroup := "$default"

	sequencenumber := int64(1)
	sequencenumberString := strconv.FormatInt(sequencenumber, 10)
	metadata := map[string]*string{
		"sequencenumber": &sequencenumberString,
	}

	containerName := "blobmetadatacontainer"
	urlPath := fmt.Sprintf("eventhubnamespace.servicebus.windows.net/hub/%s/checkpoint/%s", consumerGroup, partitionID)

	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub",
		StorageConnection:     StorageConnectionString,
		EventHubConsumerGroup: consumerGroup,
		EventHubName:          "hub",
		BlobContainer:         containerName,
		CheckpointStrategy:    "blobMetadata",
	}

	client, err := GetStorageBlobClient(logr.Discard(), eventHubInfo.PodIdentity, eventHubInfo.StorageConnection, eventHubInfo.StorageAccountName, eventHubInfo.BlobStorageEndpoint, 3*time.Second)
	assert.NoError(t, err, "error creating the blob client")

	err = createNewCheckpointInStorage(ctx, client, containerName, urlPath, "", metadata)
	assert.NoError(t, err, "error creating checkpoint")

	expectedCheckpoint := Checkpoint{
		PartitionID:    partitionID,
		SequenceNumber: sequencenumber,
	}

	check, _ := GetCheckpointFromBlobStorage(ctx, client, eventHubInfo, partitionID)
	assert.Equal(t, check, expectedCheckpoint)
}

func TestCheckpointFromBlobStorageGoSdk(t *testing.T) {
	if StorageConnectionString == "" {
		return
	}

	ctx := context.Background()
	partitionID := "0"

	sequencenumber := int64(1)

	containerName := "gosdkcontainer"
	checkpointFormat := "{\"partitionID\":\"%s\",\"epoch\":0,\"owner\":\"\",\"checkpoint\":{\"sequenceNumber\":%d,\"enqueueTime\":\"\"},\"state\":\"\",\"token\":\"\"}"
	checkpoint := fmt.Sprintf(checkpointFormat, partitionID, sequencenumber)

	urlPath := partitionID

	eventHubInfo := EventHubInfo{
		EventHubConnection: "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub",
		StorageConnection:  StorageConnectionString,
		EventHubName:       "hub",
		BlobContainer:      containerName,
		CheckpointStrategy: "goSdk",
	}

	client, err := GetStorageBlobClient(logr.Discard(), eventHubInfo.PodIdentity, eventHubInfo.StorageConnection, eventHubInfo.StorageAccountName, eventHubInfo.BlobStorageEndpoint, 3*time.Second)
	assert.NoError(t, err, "error creating the blob client")

	err = createNewCheckpointInStorage(ctx, client, containerName, urlPath, checkpoint, nil)
	assert.NoError(t, err, "error creating checkpoint")

	expectedCheckpoint := Checkpoint{
		PartitionID:    partitionID,
		SequenceNumber: sequencenumber,
	}

	check, _ := GetCheckpointFromBlobStorage(ctx, client, eventHubInfo, partitionID)
	assert.Equal(t, check, expectedCheckpoint)
}

func TestCheckpointFromBlobStorageDapr(t *testing.T) {
	if StorageConnectionString == "" {
		return
	}

	ctx := context.Background()
	partitionID := "0"
	consumerGroup := "$default"
	eventhubName := "hub"

	sequencenumber := int64(1)

	containerName := "dapr-container"
	checkpointFormat := "{\"partitionID\":\"%s\",\"epoch\":0,\"owner\":\"\",\"checkpoint\":{\"sequenceNumber\":%d,\"enqueueTime\":\"\"},\"state\":\"\",\"token\":\"\"}"
	checkpoint := fmt.Sprintf(checkpointFormat, partitionID, sequencenumber)

	urlPath := fmt.Sprintf("dapr-%s-%s-%s", eventhubName, consumerGroup, partitionID)

	eventHubInfo := EventHubInfo{
		EventHubConnection:    fmt.Sprintf("Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=%s", eventhubName),
		StorageConnection:     StorageConnectionString,
		EventHubName:          eventhubName,
		BlobContainer:         containerName,
		EventHubConsumerGroup: consumerGroup,
		CheckpointStrategy:    "dapr",
	}

	client, err := GetStorageBlobClient(logr.Discard(), eventHubInfo.PodIdentity, eventHubInfo.StorageConnection, eventHubInfo.StorageAccountName, eventHubInfo.BlobStorageEndpoint, 3*time.Second)
	assert.NoError(t, err, "error creating the blob client")

	err = createNewCheckpointInStorage(ctx, client, containerName, urlPath, checkpoint, nil)
	assert.NoError(t, err, "error creating checkpoint")

	expectedCheckpoint := Checkpoint{
		PartitionID:    partitionID,
		SequenceNumber: sequencenumber,
	}

	check, _ := GetCheckpointFromBlobStorage(ctx, client, eventHubInfo, partitionID)
	assert.Equal(t, check, expectedCheckpoint)
}

func TestShouldParseCheckpointForFunction(t *testing.T) {
	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub-test",
		EventHubConsumerGroup: "$Default",
	}

	cp := newCheckpointer(eventHubInfo, "0")
	container, path, _ := cp.resolvePath(eventHubInfo)

	assert.Equal(t, container, "azure-webjobs-eventhub")
	assert.Equal(t, path, "eventhubnamespace.servicebus.windows.net/hub-test/$Default/0")
}

func TestShouldParseCheckpointForFunctionWithCheckpointStrategy(t *testing.T) {
	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub-test",
		EventHubConsumerGroup: "$Default",
		CheckpointStrategy:    "azureFunction",
	}

	cp := newCheckpointer(eventHubInfo, "0")
	container, path, _ := cp.resolvePath(eventHubInfo)

	assert.Equal(t, container, "azure-webjobs-eventhub")
	assert.Equal(t, path, "eventhubnamespace.servicebus.windows.net/hub-test/$Default/0")
}

func TestShouldParseCheckpointForFunctionWithPodIdentity(t *testing.T) {
	eventHubInfo := EventHubInfo{
		Namespace:                "eventhubnamespace",
		EventHubName:             "hub-test",
		EventHubConsumerGroup:    "$Default",
		ServiceBusEndpointSuffix: "servicebus.windows.net",
		PodIdentity:              kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload},
	}

	cp := newCheckpointer(eventHubInfo, "0")
	container, path, _ := cp.resolvePath(eventHubInfo)

	assert.Equal(t, container, "azure-webjobs-eventhub")
	assert.Equal(t, path, "eventhubnamespace.servicebus.windows.net/hub-test/$Default/0")

	eventHubInfo.PodIdentity = kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload}
	cp = newCheckpointer(eventHubInfo, "0")
	container, path, _ = cp.resolvePath(eventHubInfo)

	assert.Equal(t, container, "azure-webjobs-eventhub")
	assert.Equal(t, path, "eventhubnamespace.servicebus.windows.net/hub-test/$Default/0")
}

func TestShouldParseCheckpointForFunctionWithCheckpointStrategyAndPodIdentity(t *testing.T) {
	eventHubInfo := EventHubInfo{
		Namespace:                "eventhubnamespace",
		EventHubName:             "hub-test",
		EventHubConsumerGroup:    "$Default",
		ServiceBusEndpointSuffix: "servicebus.windows.net",
		CheckpointStrategy:       "azureFunction",
		PodIdentity:              kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload},
	}

	cp := newCheckpointer(eventHubInfo, "0")
	container, path, _ := cp.resolvePath(eventHubInfo)

	assert.Equal(t, container, "azure-webjobs-eventhub")
	assert.Equal(t, path, "eventhubnamespace.servicebus.windows.net/hub-test/$Default/0")

	eventHubInfo.PodIdentity = kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload}
	cp = newCheckpointer(eventHubInfo, "0")
	container, path, _ = cp.resolvePath(eventHubInfo)

	assert.Equal(t, container, "azure-webjobs-eventhub")
	assert.Equal(t, path, "eventhubnamespace.servicebus.windows.net/hub-test/$Default/0")
}

func TestShouldParseCheckpointForDefault(t *testing.T) {
	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub-test",
		EventHubConsumerGroup: "$Default",
		BlobContainer:         "DefaultContainer",
	}

	cp := newCheckpointer(eventHubInfo, "0")
	container, path, _ := cp.resolvePath(eventHubInfo)

	assert.Equal(t, container, eventHubInfo.BlobContainer)
	assert.Equal(t, path, "$Default/0")
}

func TestShouldParseCheckpointForBlobMetadata(t *testing.T) {
	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub-test;",
		EventHubConsumerGroup: "$Default",
		BlobContainer:         "containername",
		CheckpointStrategy:    "blobMetadata",
	}

	cp := newCheckpointer(eventHubInfo, "0")
	container, path, _ := cp.resolvePath(eventHubInfo)

	assert.Equal(t, container, eventHubInfo.BlobContainer)
	assert.Equal(t, path, "eventhubnamespace.servicebus.windows.net/hub-test/$default/checkpoint/0")
}

func TestShouldParseCheckpointForBlobMetadataWithError(t *testing.T) {
	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub-test\n",
		EventHubConsumerGroup: "$Default",
		BlobContainer:         "containername",
		CheckpointStrategy:    "blobMetadata",
	}

	cp := newCheckpointer(eventHubInfo, "0")
	_, _, err := cp.resolvePath(eventHubInfo)
	assert.Error(t, err, "Should have return an err on invalid url characters")
}

func TestShouldParseCheckpointForBlobMetadataWithPodIdentity(t *testing.T) {
	eventHubInfo := EventHubInfo{
		Namespace:                "eventhubnamespace",
		EventHubName:             "hub-test",
		EventHubConsumerGroup:    "$Default",
		ServiceBusEndpointSuffix: "servicebus.windows.net",
		BlobContainer:            "containername",
		CheckpointStrategy:       "blobMetadata",
	}

	cp := newCheckpointer(eventHubInfo, "0")
	container, path, _ := cp.resolvePath(eventHubInfo)

	assert.Equal(t, container, eventHubInfo.BlobContainer)
	assert.Equal(t, path, "eventhubnamespace.servicebus.windows.net/hub-test/$default/checkpoint/0")
}

func TestShouldParseCheckpointForGoSdk(t *testing.T) {
	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub-test",
		EventHubConsumerGroup: "$Default",
		BlobContainer:         "containername",
		CheckpointStrategy:    "goSdk",
	}

	cp := newCheckpointer(eventHubInfo, "0")
	container, path, _ := cp.resolvePath(eventHubInfo)

	assert.Equal(t, container, eventHubInfo.BlobContainer)
	assert.Equal(t, path, "0")
}

func TestShouldParseCheckpointForDapr(t *testing.T) {
	eventHubInfo := EventHubInfo{
		EventHubConnection:    "Endpoint=sb://eventhubnamespace.servicebus.windows.net/;EntityPath=hub-test",
		EventHubConsumerGroup: "$default",
		BlobContainer:         "containername",
		CheckpointStrategy:    "dapr",
	}

	cp := newCheckpointer(eventHubInfo, "0")
	container, path, _ := cp.resolvePath(eventHubInfo)

	assert.Equal(t, container, eventHubInfo.BlobContainer)
	assert.Equal(t, path, "dapr-hub-test-$default-0")
}

func TestShouldParseCheckpointForDaprWithPodIdentity(t *testing.T) {
	eventHubInfo := EventHubInfo{
		Namespace:                "eventhubnamespace",
		EventHubName:             "hub-test",
		EventHubConsumerGroup:    "$default",
		ServiceBusEndpointSuffix: "servicebus.windows.net",
		BlobContainer:            "containername",
		CheckpointStrategy:       "dapr",
	}

	cp := newCheckpointer(eventHubInfo, "0")
	container, path, _ := cp.resolvePath(eventHubInfo)

	assert.Equal(t, container, eventHubInfo.BlobContainer)
	assert.Equal(t, path, "dapr-hub-test-$default-0")
}

func createNewCheckpointInStorage(ctx context.Context, client *azblob.Client, containerName string, path string, checkpoint string, metadata map[string]*string) error {
	// Create container
	_, err := client.CreateContainer(ctx, containerName, nil)
	if err != nil && !bloberror.HasCode(err, bloberror.ContainerAlreadyExists) {
		return fmt.Errorf("failed to create container: %w", err)
	}
	var b bytes.Buffer
	b.WriteString(checkpoint)

	// Upload file
	_, err = client.UploadBuffer(ctx, containerName, path, b.Bytes(), &blockblob.UploadBufferOptions{
		BlockSize: 4 * 1024 * 1024,
		Metadata:  metadata,
	})
	return err
}
