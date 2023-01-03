package scalers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/go-logr/logr"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
)

const (
	eventHubConsumerGroup     = "testEventHubConsumerGroup"
	eventHubConnectionSetting = "testEventHubConnectionSetting"
	storageConnectionSetting  = "testStorageConnectionSetting"
	eventHubsConnection       = "Endpoint=sb://testEventHubNamespace.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=testKey;EntityPath=testEventHub"
	serviceBusEndpointSuffix  = "serviceBusEndpointSuffix"
	storageEndpointSuffix     = "storageEndpointSuffix"
	activeDirectoryEndpoint   = "activeDirectoryEndpoint"
	eventHubResourceURL       = "eventHubResourceURL"
	testEventHubNamespace     = "kedatesteventhub"
	testEventHubName          = "eventhub1"
	checkpointFormat          = "{\"SequenceNumber\":%d,\"PartitionId\":\"%s\"}"
	testContainerName         = "azure-webjobs-eventhub"
)

type parseEventHubMetadataTestData struct {
	metadata    map[string]string
	resolvedEnv map[string]string
	isError     bool
}

type eventHubMetricIdentifier struct {
	metadataTestData *parseEventHubMetadataTestData
	scalerIndex      int
	name             string
}

var sampleEventHubResolvedEnv = map[string]string{eventHubConnectionSetting: eventHubsConnection, storageConnectionSetting: "none"}

var parseEventHubMetadataDataset = []parseEventHubMetadataTestData{
	{
		metadata: map[string]string{},
		isError:  true,
	},
	// properly formed event hub metadata
	{
		metadata:    map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "connectionFromEnv": eventHubConnectionSetting, "unprocessedEventThreshold": "15"},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     false,
	},
	// missing event hub connection setting
	{
		metadata:    map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "unprocessedEventThreshold": "15"},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     true,
	},
	// missing storage connection setting
	{
		metadata:    map[string]string{"consumerGroup": eventHubConsumerGroup, "connectionFromEnv": eventHubConnectionSetting, "unprocessedEventThreshold": "15"},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     true,
	},
	// missing event hub consumer group - should replace with default
	{
		metadata: map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "connectionFromEnv": eventHubConnectionSetting, "unprocessedEventThreshold": "15"}, resolvedEnv: sampleEventHubResolvedEnv,
		isError: false,
	},
	// missing unprocessed event threshold - should replace with default
	{
		metadata:    map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "connectionFromEnv": eventHubConnectionSetting},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     false,
	},
	// invalid activation unprocessed event threshold
	{
		metadata:    map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "connectionFromEnv": eventHubConnectionSetting, "activationUnprocessedEventThreshold": "AA"},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     true,
	},
	// added blob container details
	{
		metadata:    map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "connectionFromEnv": eventHubConnectionSetting, "blobContainer": testContainerName, "checkpointStrategy": "azureFunction"},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     false,
	},
	// connection string without EntityPath, no event hub name provided
	{
		metadata:    map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "connectionFromEnv": eventHubConnectionSetting, "unprocessedEventThreshold": "15"},
		resolvedEnv: map[string]string{eventHubConnectionSetting: "Endpoint=sb://testEventHubNamespace.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=testKey;", storageConnectionSetting: "none"},
		isError:     true,
	},
	// connection string without EntityPath, event hub name provided
	{
		metadata:    map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "connectionFromEnv": eventHubConnectionSetting, "unprocessedEventThreshold": "15", "eventHubName": testEventHubName},
		resolvedEnv: map[string]string{eventHubConnectionSetting: "Endpoint=sb://testEventHubNamespace.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=testKey;", storageConnectionSetting: "none"},
		isError:     false,
	},
}

var parseEventHubMetadataDatasetWithPodIdentity = []parseEventHubMetadataTestData{
	{
		metadata:    map[string]string{},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     true,
	},
	// Even though connection string is provided, this should fail because the eventhub Namespace is not provided explicitly when using Pod Identity
	{
		metadata: map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "connectionFromEnv": eventHubConnectionSetting, "unprocessedEventThreshold": "15"},
		isError:  true},
	// properly formed event hub metadata with Pod Identity
	{
		metadata:    map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "unprocessedEventThreshold": "15", "eventHubName": testEventHubName, "eventHubNamespace": testEventHubNamespace},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     false,
	},
	// missing eventHubname
	{
		metadata:    map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "unprocessedEventThreshold": "15", "eventHubNamespace": testEventHubNamespace},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     true,
	},
	// missing eventHubNamespace
	{
		metadata:    map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "unprocessedEventThreshold": "15", "eventHubName": testEventHubName},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     true},
	// metadata with cloud specified
	{
		metadata:    map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "unprocessedEventThreshold": "15", "eventHubName": testEventHubName, "eventHubNamespace": testEventHubNamespace, "cloud": "azurePublicCloud"},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     false,
	},
	// metadata with private cloud missing service bus endpoint suffix and active directory endpoint and eventHubResourceURL
	{
		metadata:    map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "unprocessedEventThreshold": "15", "eventHubName": testEventHubName, "eventHubNamespace": testEventHubNamespace, "cloud": "private"},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     true,
	},
	// metadata with private cloud missing active directory endpoint and resourceURL
	{
		metadata:    map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "unprocessedEventThreshold": "15", "eventHubName": testEventHubName, "eventHubNamespace": testEventHubNamespace, "cloud": "private", "endpointSuffix": serviceBusEndpointSuffix},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     true},
	// metadata with private cloud missing service bus endpoint suffix and resource URL
	{
		metadata:    map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "unprocessedEventThreshold": "15", "eventHubName": testEventHubName, "eventHubNamespace": testEventHubNamespace, "cloud": "private", "activeDirectoryEndpoint": activeDirectoryEndpoint},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     true},
	// metadata with private cloud missing service bus endpoint suffix and active directory endpoint
	{
		metadata:    map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "unprocessedEventThreshold": "15", "eventHubName": testEventHubName, "eventHubNamespace": testEventHubNamespace, "cloud": "private", "eventHubResourceURL": eventHubResourceURL},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     true,
	},
	// properly formed metadata with private cloud
	{
		metadata:    map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "unprocessedEventThreshold": "15", "eventHubName": testEventHubName, "eventHubNamespace": testEventHubNamespace, "cloud": "private", "endpointSuffix": serviceBusEndpointSuffix, "activeDirectoryEndpoint": activeDirectoryEndpoint, "eventHubResourceURL": eventHubResourceURL},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     false,
	},
	// properly formed event hub metadata with Pod Identity and no storage connection string
	{
		metadata:    map[string]string{"storageAccountName": "blobstorage", "consumerGroup": eventHubConsumerGroup, "unprocessedEventThreshold": "15", "eventHubName": testEventHubName, "eventHubNamespace": testEventHubNamespace},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     false,
	},
	// event hub metadata with Pod Identity, no storage connection string, no storageAccountName - should fail
	{
		metadata:    map[string]string{"consumerGroup": eventHubConsumerGroup, "unprocessedEventThreshold": "15", "eventHubName": testEventHubName, "eventHubNamespace": testEventHubNamespace},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     true,
	},
	// event hub metadata with Pod Identity, no storage connection string, empty storageAccountName - should fail
	{
		metadata:    map[string]string{"storageAccount": "", "consumerGroup": eventHubConsumerGroup, "unprocessedEventThreshold": "15", "eventHubName": testEventHubName, "eventHubNamespace": testEventHubNamespace},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     true},
	// event hub metadata with Pod Identity, storage connection string, empty storageAccountName - should ignore pod identity for blob storage and succeed
	{
		metadata:    map[string]string{"storageConnectionFromEnv": storageConnectionSetting, "storageAccountName": "", "consumerGroup": eventHubConsumerGroup, "unprocessedEventThreshold": "15", "eventHubName": testEventHubName, "eventHubNamespace": testEventHubNamespace},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     false,
	},
	// event hub metadata with Pod Identity and no storage connection string, private cloud and no storageEndpointSuffix - should fail
	{
		metadata:    map[string]string{"cloud": "private", "storageAccountName": "blobstorage", "consumerGroup": eventHubConsumerGroup, "unprocessedEventThreshold": "15", "eventHubName": testEventHubName, "eventHubNamespace": testEventHubNamespace},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     true},
	// properly formed event hub metadata with Pod Identity and no storage connection string, private cloud and storageEndpointSuffix
	{
		metadata:    map[string]string{"cloud": "private", "endpointSuffix": serviceBusEndpointSuffix, "activeDirectoryEndpoint": activeDirectoryEndpoint, "eventHubResourceURL": eventHubResourceURL, "storageAccountName": "aStorageAccount", "storageEndpointSuffix": storageEndpointSuffix, "consumerGroup": eventHubConsumerGroup, "unprocessedEventThreshold": "15", "eventHubName": testEventHubName, "eventHubNamespace": testEventHubNamespace},
		resolvedEnv: sampleEventHubResolvedEnv,
		isError:     false,
	},
}

var eventHubMetricIdentifiers = []eventHubMetricIdentifier{
	{&parseEventHubMetadataDataset[1], 0, "s0-azure-eventhub-testEventHubConsumerGroup"},
	{&parseEventHubMetadataDataset[1], 1, "s1-azure-eventhub-testEventHubConsumerGroup"},
}

var testEventHubScaler = azureEventHubScaler{
	metadata: &eventHubMetadata{
		eventHubInfo: azure.EventHubInfo{
			EventHubConnection: "none",
			StorageConnection:  "none",
		},
	},
}

func TestParseEventHubMetadata(t *testing.T) {
	// Test first with valid resolved environment
	for _, testData := range parseEventHubMetadataDataset {
		_, err := parseAzureEventHubMetadata(logr.Discard(), &ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testData.resolvedEnv, AuthParams: map[string]string{}})

		if err != nil && !testData.isError {
			t.Errorf("Expected success but got error: %s", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error and got success")
		}
	}

	for _, testData := range parseEventHubMetadataDatasetWithPodIdentity {
		_, err := parseAzureEventHubMetadata(logr.Discard(), &ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: sampleEventHubResolvedEnv,
			AuthParams: map[string]string{}, PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzure}})

		if err != nil && !testData.isError {
			t.Errorf("Expected success but got error: %s", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error and got success")
		}
	}

	for _, testData := range parseEventHubMetadataDatasetWithPodIdentity {
		_, err := parseAzureEventHubMetadata(logr.Discard(), &ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: sampleEventHubResolvedEnv,
			AuthParams: map[string]string{}, PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload}})

		if err != nil && !testData.isError {
			t.Errorf("Expected success but got error: %s", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error and got success")
		}
	}
}

func TestGetUnprocessedEventCountInPartition(t *testing.T) {
	ctx := context.Background()
	t.Log("This test will use the environment variable EVENTHUB_CONNECTION_STRING and STORAGE_CONNECTION_STRING if it is set.")
	t.Log("If set, it will connect to the storage account and event hub to determine how many messages are in the event hub.")
	t.Logf("EventHub has 1 message in partition 0 and 0 messages in partition 1")

	eventHubKey := os.Getenv("AZURE_EVENTHUB_KEY")
	storageConnectionString := os.Getenv("AZURE_STORAGE_CONNECTION_STRING")

	if eventHubKey != "" && storageConnectionString != "" {
		eventHubConnectionString := fmt.Sprintf("Endpoint=sb://%s.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=%s;EntityPath=%s", testEventHubNamespace, eventHubKey, testEventHubName)
		storageCredentials, endpoint, err := azure.ParseAzureStorageBlobConnection(ctx, http.DefaultClient,
			kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone}, storageConnectionString, "", "")
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		t.Log("Creating event hub client...")
		hubOption := eventhub.HubWithPartitionedSender("0")
		client, err := eventhub.NewHubFromConnectionString(eventHubConnectionString, hubOption)
		if err != nil {
			t.Fatalf("Expected to create event hub client but got error: %s", err)
		}

		if eventHubConnectionString == "" {
			t.Fatal("Event hub connection string needed for test")
		}

		if storageConnectionString == "" {
			t.Fatal("Storage connection string needed for test")
		}

		// Can actually test that numbers return
		testEventHubScaler.metadata.eventHubInfo.EventHubConnection = eventHubConnectionString
		testEventHubScaler.metadata.eventHubInfo.StorageConnection = storageConnectionString
		testEventHubScaler.client = client
		testEventHubScaler.metadata.eventHubInfo.EventHubConsumerGroup = "$Default"

		// Send 1 message to event hub first
		t.Log("Sending message to event hub")
		err = SendMessageToEventHub(client)
		if err != nil {
			t.Error(err)
		}

		// Create fake checkpoint with path azure-webjobs-eventhub/<eventhub-namespace-name>.servicebus.windows.net/<eventhub-name>/$Default
		t.Log("Creating container..")
		ctx, err := CreateNewCheckpointInStorage(endpoint, storageCredentials, client)
		if err != nil {
			t.Errorf("err creating container: %s", err)
		}

		partitionInfo0, err := testEventHubScaler.client.GetPartitionInformation(ctx, "0")
		if err != nil {
			t.Errorf("unable to get partitionRuntimeInfo for partition 0: %s", err)
		}

		partitionInfo1, err := testEventHubScaler.client.GetPartitionInformation(ctx, "0")
		if err != nil {
			t.Errorf("unable to get partitionRuntimeInfo for partition 1: %s", err)
		}

		unprocessedEventCountInPartition0, _, err0 := testEventHubScaler.GetUnprocessedEventCountInPartition(ctx, partitionInfo0)
		unprocessedEventCountInPartition1, _, err1 := testEventHubScaler.GetUnprocessedEventCountInPartition(ctx, partitionInfo1)
		if err0 != nil {
			t.Errorf("Expected success but got error: %s", err0)
		}
		if err1 != nil {
			t.Errorf("Expected success but got error: %s", err1)
		}

		if unprocessedEventCountInPartition0 != 1 {
			t.Errorf("Expected 1 message in partition 0, got %d", unprocessedEventCountInPartition0)
		}

		if unprocessedEventCountInPartition1 != 0 {
			t.Errorf("Expected 0 messages in partition 1, got %d", unprocessedEventCountInPartition1)
		}

		// Delete container - this will also delete checkpoint
		t.Log("Deleting container...")
		err = DeleteContainerInStorage(ctx, endpoint, storageCredentials)
		if err != nil {
			t.Error(err)
		}
	}
}
func TestGetUnprocessedEventCountIfNoCheckpointExists(t *testing.T) {
	t.Log("This test will use the environment variable EVENTHUB_CONNECTION_STRING and STORAGE_CONNECTION_STRING if it is set.")
	t.Log("If set, it will connect to the storage account and event hub to determine how many messages are in the event hub.")
	t.Logf("EventHub has 1 message in partition 0 and 0 messages in partition 1")

	eventHubKey := os.Getenv("AZURE_EVENTHUB_KEY")
	storageConnectionString := os.Getenv("AZURE_STORAGE_CONNECTION_STRING")

	if eventHubKey != "" && storageConnectionString != "" {
		eventHubConnectionString := fmt.Sprintf("Endpoint=sb://%s.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=%s;EntityPath=%s", testEventHubNamespace, eventHubKey, testEventHubName)
		t.Log("Creating event hub client...")
		hubOption := eventhub.HubWithPartitionedSender("0")
		client, err := eventhub.NewHubFromConnectionString(eventHubConnectionString, hubOption)
		if err != nil {
			t.Errorf("Expected to create event hub client but got error: %s", err)
		}

		if eventHubConnectionString == "" {
			t.Fatal("Event hub connection string needed for test")
		}

		if storageConnectionString == "" {
			t.Fatal("Storage connection string needed for test")
		}

		// Can actually test that numbers return
		testEventHubScaler.metadata.eventHubInfo.EventHubConnection = eventHubConnectionString
		testEventHubScaler.metadata.eventHubInfo.StorageConnection = storageConnectionString
		testEventHubScaler.client = client
		testEventHubScaler.metadata.eventHubInfo.EventHubConsumerGroup = "$Default"

		// Send 1 message to event hub first
		t.Log("Sending message to event hub")
		err = SendMessageToEventHub(client)
		if err != nil {
			t.Error(err)
		}

		ctx := context.Background()

		partitionInfo0, err := testEventHubScaler.client.GetPartitionInformation(ctx, "0")
		if err != nil {
			t.Errorf("unable to get partitionRuntimeInfo for partition 0: %s", err)
		}

		partitionInfo1, err := testEventHubScaler.client.GetPartitionInformation(ctx, "1")
		if err != nil {
			t.Errorf("unable to get partitionRuntimeInfo for partition 1: %s", err)
		}

		unprocessedEventCountInPartition0, _, err0 := testEventHubScaler.GetUnprocessedEventCountInPartition(ctx, partitionInfo0)
		unprocessedEventCountInPartition1, _, err1 := testEventHubScaler.GetUnprocessedEventCountInPartition(ctx, partitionInfo1)
		if err0 != nil {
			t.Errorf("Expected success but got error: %s", err0)
		}
		if err1 != nil {
			t.Errorf("Expected success but got error: %s", err1)
		}

		if unprocessedEventCountInPartition0 != 1 {
			t.Errorf("Expected 1 message in partition 0, got %d", unprocessedEventCountInPartition0)
		}

		if unprocessedEventCountInPartition1 != 0 {
			t.Errorf("Expected 0 messages in partition 1, got %d", unprocessedEventCountInPartition1)
		}
	}
}

func TestGetUnprocessedEventCountWithoutCheckpointReturning1Message(t *testing.T) {
	// After the first message the lastsequencenumber init to 0
	partitionInfo := eventhub.HubPartitionRuntimeInformation{
		PartitionID:             "0",
		LastSequenceNumber:      0,
		BeginningSequenceNumber: 0,
	}

	unprocessedEventCountInPartition0 := GetUnprocessedEventCountWithoutCheckpoint(&partitionInfo)

	if unprocessedEventCountInPartition0 != 1 {
		t.Errorf("Expected 1 messages in partition 0, got %d", unprocessedEventCountInPartition0)
	}
}

func TestGetUnprocessedEventCountWithoutCheckpointReturning0Message(t *testing.T) {
	// An empty partition starts with an equal value on last-/beginning-sequencenumber other than 0
	partitionInfo := eventhub.HubPartitionRuntimeInformation{
		PartitionID:             "0",
		LastSequenceNumber:      255,
		BeginningSequenceNumber: 255,
	}

	unprocessedEventCountInPartition0 := GetUnprocessedEventCountWithoutCheckpoint(&partitionInfo)

	if unprocessedEventCountInPartition0 != 0 {
		t.Errorf("Expected 0 messages in partition 0, got %d", unprocessedEventCountInPartition0)
	}
}

func TestGetUnprocessedEventCountWithoutCheckpointReturning2Messages(t *testing.T) {
	partitionInfo := eventhub.HubPartitionRuntimeInformation{
		PartitionID:             "0",
		LastSequenceNumber:      1,
		BeginningSequenceNumber: 0,
	}

	unprocessedEventCountInPartition0 := GetUnprocessedEventCountWithoutCheckpoint(&partitionInfo)

	if unprocessedEventCountInPartition0 != 2 {
		t.Errorf("Expected 0 messages in partition 0, got %d", unprocessedEventCountInPartition0)
	}
}

func TestGetATotalLagOf20For2PartitionsOn100UnprocessedEvents(t *testing.T) {
	lag := getTotalLagRelatedToPartitionAmount(100, 2, 10)

	if lag != 20 {
		t.Errorf("Expected a lag of 20 for 2 partitions, got %d", lag)
	}
}

func TestGetATotalLagOf100For20PartitionsOn100UnprocessedEvents(t *testing.T) {
	lag := getTotalLagRelatedToPartitionAmount(100, 20, 10)

	if lag != 100 {
		t.Errorf("Expected a lag of 100 for 20 partitions, got %d", lag)
	}
}

func CreateNewCheckpointInStorage(endpoint *url.URL, credential azblob.Credential, client *eventhub.Hub) (context.Context, error) {
	urlPath := fmt.Sprintf("%s.servicebus.windows.net/%s/$Default/", testEventHubNamespace, testEventHubName)

	// Create container
	ctx := context.Background()
	path, _ := url.Parse(testContainerName)
	url := endpoint.ResolveReference(path)
	containerURL := azblob.NewContainerURL(*url, azblob.NewPipeline(credential, azblob.PipelineOptions{}))
	_, err := containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessNone)
	if err != nil {
		return ctx, fmt.Errorf("failed to create container: %w", err)
	}

	// Create directory checkpoints will be in
	err = os.MkdirAll(urlPath, 0777)
	if err != nil {
		return ctx, fmt.Errorf("Unable to create directory: %w", err)
	}
	defer os.RemoveAll(urlPath)

	file, err := os.Create(fmt.Sprintf("%s/file", urlPath))
	if err != nil {
		return ctx, fmt.Errorf("Unable to create folder: %w", err)
	}
	defer file.Close()

	blobFolderURL := containerURL.NewBlockBlobURL(urlPath)

	// Upload file
	_, err = azblob.UploadFileToBlockBlob(ctx, file, blobFolderURL, azblob.UploadToBlockBlobOptions{
		BlockSize:   4 * 1024 * 1024,
		Parallelism: 16})
	if err != nil {
		return ctx, fmt.Errorf("Err uploading file to blob: %w", err)
	}

	// Make checkpoint blob files
	if err := CreatePartitionFile(ctx, urlPath, "0", containerURL, client); err != nil {
		return ctx, fmt.Errorf("failed to create partitionID 0 file: %w", err)
	}
	if err := CreatePartitionFile(ctx, urlPath, "1", containerURL, client); err != nil {
		return ctx, fmt.Errorf("failed to create partitionID 1 file: %w", err)
	}

	return ctx, nil
}

func CreatePartitionFile(ctx context.Context, urlPathToPartition string, partitionID string, containerURL azblob.ContainerURL, client *eventhub.Hub) error {
	// Create folder structure
	filePath := urlPathToPartition + partitionID

	partitionInfo, err := client.GetPartitionInformation(ctx, partitionID)
	if err != nil {
		return fmt.Errorf("unable to get partition info: %w", err)
	}

	f, err := os.Create(partitionID)
	if err != nil {
		return fmt.Errorf("unable to create file: %w", err)
	}

	if partitionID == "0" {
		_, err = f.WriteString(fmt.Sprintf(checkpointFormat, partitionInfo.LastSequenceNumber-1, partitionID))
		if err != nil {
			return fmt.Errorf("unable to write to file: %w", err)
		}
	} else {
		_, err = f.WriteString(fmt.Sprintf(checkpointFormat, partitionInfo.LastSequenceNumber, partitionID))
		if err != nil {
			return fmt.Errorf("unable to write to file: %w", err)
		}
	}

	// Write checkpoints to file
	file, err := os.Open(partitionID)
	if err != nil {
		return fmt.Errorf("Unable to create file: %w", err)
	}
	defer file.Close()

	blobFileURL := containerURL.NewBlockBlobURL(filePath)

	// Upload folder
	_, err = azblob.UploadFileToBlockBlob(ctx, file, blobFileURL, azblob.UploadToBlockBlobOptions{
		BlockSize:   4 * 1024 * 1024,
		Parallelism: 16})
	if err != nil {
		return fmt.Errorf("Err uploading file to blob: %w", err)
	}
	return nil
}

func SendMessageToEventHub(client *eventhub.Hub) error {
	ctx := context.Background()

	err := client.Send(ctx, eventhub.NewEventFromString("1"))
	if err != nil {
		return fmt.Errorf("Error sending msg: %w", err)
	}
	return nil
}

func DeleteContainerInStorage(ctx context.Context, endpoint *url.URL, credential azblob.Credential) error {
	path, _ := url.Parse(testContainerName)
	url := endpoint.ResolveReference(path)
	containerURL := azblob.NewContainerURL(*url, azblob.NewPipeline(credential, azblob.PipelineOptions{}))

	_, err := containerURL.Delete(ctx, azblob.ContainerAccessConditions{
		ModifiedAccessConditions: azblob.ModifiedAccessConditions{},
	})
	if err != nil {
		return fmt.Errorf("failed to delete container in blob storage: %w", err)
	}
	return nil
}

func TestEventHubGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range eventHubMetricIdentifiers {
		meta, err := parseAzureEventHubMetadata(logr.Discard(), &ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: sampleEventHubResolvedEnv, AuthParams: map[string]string{}, ScalerIndex: testData.scalerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockEventHubScaler := azureEventHubScaler{
			metadata:   meta,
			client:     nil,
			httpClient: http.DefaultClient,
		}

		metricSpec := mockEventHubScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
