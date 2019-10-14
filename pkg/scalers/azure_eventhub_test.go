package scalers

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/url"
	"os"
	"strings"
	"testing"

	eventhub "github.com/Azure/azure-event-hubs-go"
	"github.com/Azure/azure-storage-blob-go/azblob"
)

const (
	eventHubConsumerGroup     = "testEventHubConsumerGroup"
	eventHubConnectionSetting = "testEventHubConnectionSetting"
	storageConnectionSetting  = "testStorageConnectionSetting"
	testEventHubNamespace     = "kedatesteventhub"
	testEventHubName          = "eventhub1"
	checkpointFormat          = "{\"SequenceNumber\":%d,\"PartitionId\":\"%s\"}"
	testContainerName         = "azure-webjobs-eventhub"
)

type parseEventHubMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type resolvedEnvTestData struct {
	resolvedEnv map[string]string
	isError     bool
}

var sampleEventHubResolvedEnv = map[string]string{eventHubConnectionSetting: "none", storageConnectionSetting: "none"}

var parseEventHubMetadataDataset = []parseEventHubMetadataTestData{
	{map[string]string{}, true},
	// properly formed event hub metadata
	{map[string]string{"storageConnection": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "connection": eventHubConnectionSetting, "unprocessedEventThreshold": "15"}, false},
	// missing event hub connection setting
	{map[string]string{"storageConnection": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "unprocessedEventThreshold": "15"}, true},
	// missing storage connection setting
	{map[string]string{"consumerGroup": eventHubConsumerGroup, "connection": eventHubConnectionSetting, "unprocessedEventThreshold": "15"}, true},
	// missing event hub consumer group - should replace with default
	{map[string]string{"storageConnection": storageConnectionSetting, "connection": eventHubConnectionSetting, "unprocessedEventThreshold": "15"}, false},
	// missing unprocessed event threshold - should replace with default
	{map[string]string{"storageConnection": storageConnectionSetting, "consumerGroup": eventHubConsumerGroup, "connection": eventHubConnectionSetting}, false},
}

var testEventHubScaler = AzureEventHubScaler{
	metadata: &EventHubMetadata{
		eventHubConnection: "none",
		storageConnection:  "none",
	},
}

func TestParseEventHubMetadata(t *testing.T) {
	// Test first with valid resolved environment
	for _, testData := range parseEventHubMetadataDataset {
		_, err := parseAzureEventHubMetadata(testData.metadata, sampleEventHubResolvedEnv)

		if err != nil && !testData.isError {
			t.Errorf("Expected success but got error: %s", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error and got success")
		}
	}
}

func TestGetUnprocessedEventCountInPartition(t *testing.T) {
	t.Log("This test will use the environment variable EVENTHUB_CONNECTION_STRING and STORAGE_CONNECTION_STRING if it is set.")
	t.Log("If set, it will connect to the storage account and event hub to determine how many messages are in the event hub.")
	t.Logf("EventHub has 1 message in partition 0 and 0 messages in partition 1")

	eventHubKey := os.Getenv("AZURE_EVENTHUB_KEY")
	storageConnectionString := os.Getenv("TEST_STORAGE_CONNECTION_STRING")

	if eventHubKey != "" && storageConnectionString != "" {
		eventHubConnectionString := fmt.Sprintf("Endpoint=sb://%s.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=%s;EntityPath=%s", testEventHubNamespace, eventHubKey, testEventHubName)
		storageAccountName := strings.Split(strings.Split(storageConnectionString, ";")[1], "=")[1]

		t.Log("Creating event hub client...")
		hubOption := eventhub.HubWithPartitionedSender("0")
		client, err := eventhub.NewHubFromConnectionString(eventHubConnectionString, hubOption)
		if err != nil {
			t.Errorf("Expected to create event hub client but got error: %s", err)
		}

		_, storageCredentials, err := GetStorageCredentials(storageConnectionString)
		if err != nil {
			t.Errorf("Expected to generate storage credentials but got error: %s", err)
		}

		if eventHubConnectionString == "" {
			t.Fatal("Event hub connection string needed for test")
		}

		if storageConnectionString == "" {
			t.Fatal("Storage connection string needed for test")
		}

		// Can actually test that numbers return
		testEventHubScaler.metadata.eventHubConnection = eventHubConnectionString
		testEventHubScaler.metadata.storageConnection = storageConnectionString
		testEventHubScaler.client = client
		testEventHubScaler.storageCredentials = storageCredentials
		testEventHubScaler.metadata.eventHubConsumerGroup = "$Default"

		// Send 1 message to event hub first
		t.Log("Sending message to event hub")
		err = SendMessageToEventHub(client)
		if err != nil {
			t.Error(err)
		}

		// Create fake checkpoint with path azure-webjobs-eventhub/<eventhub-namespace-name>.servicebus.windows.net/<eventhub-name>/$Default
		t.Log("Creating container..")
		ctx, err := CreateNewCheckpointInStorage(storageAccountName, storageCredentials, client)
		if err != nil {
			t.Errorf("err creating container: %s", err)
		}

		unprocessedEventCountInPartition0, err0 := testEventHubScaler.GetUnprocessedEventCountInPartition(ctx, "0")
		unprocessedEventCountInPartition1, err1 := testEventHubScaler.GetUnprocessedEventCountInPartition(ctx, "1")
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
		err = DeleteContainerInStorage(ctx, storageAccountName, storageCredentials)
		if err != nil {
			t.Error(err)
		}
	}
}

const csharpSdkCheckpoint = `{
		"Epoch": 123456,
		"Offset": "test offset",
		"Owner": "test owner",
		"PartitionId": "test partitionId",
		"SequenceNumber": 12345
	}`

const pythonSdkCheckpoint = `{
		"epoch": 123456,
		"offset": "test offset",
		"owner": "test owner",
		"partition_id": "test partitionId",
		"sequence_number": 12345
	}`

func TestGetCheckpoint(t *testing.T) {
	cckp, err := getCheckpoint([]byte(csharpSdkCheckpoint))
	if err != nil {
		t.Error(err)
	}

	pckp, err := getCheckpoint([]byte(pythonSdkCheckpoint))
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, cckp, pckp)
}

func CreateNewCheckpointInStorage(storageAccountName string, credential *azblob.SharedKeyCredential, client *eventhub.Hub) (context.Context, error) {
	urlPath := fmt.Sprintf("%s.servicebus.windows.net/%s/$Default/", testEventHubNamespace, testEventHubName)

	// Create container
	ctx := context.Background()
	url, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", storageAccountName, testContainerName))
	containerURL := azblob.NewContainerURL(*url, azblob.NewPipeline(credential, azblob.PipelineOptions{}))
	_, err := containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessNone)
	if err != nil {
		return ctx, fmt.Errorf("failed to create container: %s", err)
	}

	// Create directory checkpoints will be in
	err = os.MkdirAll(urlPath, 0777)
	if err != nil {
		return ctx, fmt.Errorf("Unable to create directory: %s", err)
	}
	defer os.RemoveAll(urlPath)

	file, err := os.Create(fmt.Sprintf("%s/file", urlPath))
	if err != nil {
		return ctx, fmt.Errorf("Unable to create folder: %s", err)
	}
	defer file.Close()

	blobFolderURL := containerURL.NewBlockBlobURL(urlPath)

	// Upload file
	_, err = azblob.UploadFileToBlockBlob(ctx, file, blobFolderURL, azblob.UploadToBlockBlobOptions{
		BlockSize:   4 * 1024 * 1024,
		Parallelism: 16})
	if err != nil {
		return ctx, fmt.Errorf("Err uploading file to blob: %s", err)
	}

	// Make checkpoint blob files
	if err := CreatePartitionFile(ctx, urlPath, "0", containerURL, client); err != nil {
		return ctx, fmt.Errorf("failed to create partitionID 0 file: %s", err)
	}
	if err := CreatePartitionFile(ctx, urlPath, "1", containerURL, client); err != nil {
		return ctx, fmt.Errorf("failed to create partitionID 1 file: %s", err)
	}

	return ctx, nil
}

func CreatePartitionFile(ctx context.Context, urlPathToPartition string, partitionID string, containerURL azblob.ContainerURL, client *eventhub.Hub) error {
	// Create folder structure
	filePath := urlPathToPartition + partitionID

	partitionInfo, err := client.GetPartitionInformation(ctx, partitionID)
	if err != nil {
		return fmt.Errorf("unable to get partition info: %s", err)
	}

	f, err := os.Create(partitionID)
	if err != nil {
		return fmt.Errorf("unable to create file: %s", err)
	}

	if partitionID == "0" {
		_, err = f.WriteString(fmt.Sprintf(checkpointFormat, partitionInfo.LastSequenceNumber-1, partitionID))
		if err != nil {
			return fmt.Errorf("unable to write to file: %s", err)
		}
	} else {
		_, err = f.WriteString(fmt.Sprintf(checkpointFormat, partitionInfo.LastSequenceNumber, partitionID))
		if err != nil {
			return fmt.Errorf("unable to write to file: %s", err)
		}
	}

	// Write checkpoints to file
	file, err := os.Open(partitionID)
	if err != nil {
		return fmt.Errorf("Unable to create file: %s", err)
	}
	defer file.Close()

	blobFileURL := containerURL.NewBlockBlobURL(filePath)

	// Upload folder
	_, err = azblob.UploadFileToBlockBlob(ctx, file, blobFileURL, azblob.UploadToBlockBlobOptions{
		BlockSize:   4 * 1024 * 1024,
		Parallelism: 16})
	if err != nil {
		return fmt.Errorf("Err uploading file to blob: %s", err)
	}
	return nil
}

func SendMessageToEventHub(client *eventhub.Hub) error {
	ctx := context.Background()

	err := client.Send(ctx, eventhub.NewEventFromString("1"))
	if err != nil {
		return fmt.Errorf("Error sending msg: %s", err)
	}
	return nil
}

func DeleteContainerInStorage(ctx context.Context, storageAccountName string, credential *azblob.SharedKeyCredential) error {
	url, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", storageAccountName, testContainerName))
	containerURL := azblob.NewContainerURL(*url, azblob.NewPipeline(credential, azblob.PipelineOptions{}))

	_, err := containerURL.Delete(ctx, azblob.ContainerAccessConditions{
		ModifiedAccessConditions: azblob.ModifiedAccessConditions{},
	})
	if err != nil {
		return fmt.Errorf("failed to delete container in blob storage: %s", err)
	}
	return nil
}
