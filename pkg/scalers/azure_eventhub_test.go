package scalers

import (
	"context"
	"os"
	"testing"
)

const (
	eventHubName              = "testEventHubName"
	storageContainerName      = "testStorageContainerName"
	eventHubConnectionSetting = "testEventHubConnectionSetting"
	storageConnectionSetting  = "testStorageConnectionSetting"
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
	{map[string]string{"storageConnection": storageConnectionSetting, "storageContainerName": storageContainerName, "eventHubConnection": eventHubConnectionSetting, "eventHubName": eventHubName, "unprocessedEventThreshold": "15"}, false},
	// missing event hub connection setting
	{map[string]string{"storageConnection": storageConnectionSetting, "storageContainerName": storageContainerName, "eventHubName": eventHubName, "unprocessedEventThreshold": "15"}, true},
	// missing event hub name
	{map[string]string{"storageConnection": storageConnectionSetting, "storageContainerName": storageContainerName, "eventHubConnection": eventHubConnectionSetting, "unprocessedEventThreshold": "15"}, true},
	// missing storage connection setting
	{map[string]string{"storageContainerName": storageContainerName, "eventHubConnection": eventHubConnectionSetting, "eventHubName": eventHubName, "unprocessedEventThreshold": "15"}, true},
	// missing storage container name
	{map[string]string{"storageConnection": storageConnectionSetting, "eventHubConnection": eventHubConnectionSetting, "eventHubName": eventHubName, "unprocessedEventThreshold": "15"}, true},
	// missing unprocessed event threshold - should replace with default
	{map[string]string{"storageConnection": storageConnectionSetting, "storageContainerName": storageContainerName, "eventHubConnection": eventHubConnectionSetting, "eventHubName": eventHubName}, false},
}

var testEventHubScaler = AzureEventHubScaler{
	metadata: &EventHubMetadata{
		eventHubConnection:   "none",
		storageConnection:    "none",
		eventHubName:         eventHubName,
		storageContainerName: storageContainerName,
	},
}

func TestParseEventHubMetadata(t *testing.T) {
	// Test first with valid resolved environment
	for _, testData := range parseEventHubMetadataDataset {
		_, err := ParseAzureEventHubMetadata(testData.metadata, sampleEventHubResolvedEnv)

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
	t.Logf("\tEventHub '%s' with storage container %s has 1 message in partition 0 and 0 messages in partition 1\n", eventHubName, storageContainerName)

	eventHubConnectionString := os.Getenv("EVENTHUB_CONNECTION_STRING")
	storageConnectionString := os.Getenv("STORAGE_CONNECTION_STRING")

	if eventHubConnectionString == "" {
		t.Fatal("Event hub connection string needed for test")
	}

	if storageConnectionString == "" {
		t.Fatal("Storage connection string needed for test")
	}

	client, err := GetEventHubClient(eventHubConnectionString)
	if err != nil {
		t.Errorf("Expected to create event hub client but got error: %s", err)
	}
	_, storageCredentials, err := GetStorageCredentials(storageConnectionString)
	if err != nil {
		t.Errorf("Expected to generate storage credentials but got error: %s", err)
	}

	// Can actually test that numbers return
	testEventHubScaler.metadata.eventHubConnection = eventHubConnectionString
	testEventHubScaler.metadata.storageConnection = storageConnectionString
	testEventHubScaler.client = client
	testEventHubScaler.storageCredentials = storageCredentials

	unprocessedEventCountInPartition0, err0 := testEventHubScaler.GetUnprocessedEventCountInPartition(context.TODO(), "0")
	unprocessedEventCountInPartition1, err1 := testEventHubScaler.GetUnprocessedEventCountInPartition(context.TODO(), "1")
	if err0 != nil {
		t.Errorf("Expected success but got error: %s", err0)
	}
	if err1 != nil {
		t.Errorf("Expected success but got error: %s", err1)
	}

	if unprocessedEventCountInPartition0 != 1 {
		t.Errorf("Expected 1 message in partition 0, got %d", unprocessedEventCountInPartition0)
	}

	if unprocessedEventCountInPartition1 != 1 {
		t.Errorf("Expected 0 messages in partition 1, got %d", unprocessedEventCountInPartition1)
	}
}
