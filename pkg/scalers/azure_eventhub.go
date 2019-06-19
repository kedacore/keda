package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	eventhub "github.com/Azure/azure-event-hubs-go"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/azure"
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

// GetStorageCredentials returns azure env and storage credentials
func GetStorageCredentials(storageConnection string) (azure.Environment, *azblob.SharedKeyCredential, error) {
	storageAccountName, storageAccountKey, err := ParseAzureStorageConnectionString(storageConnection)
	if err != nil {
		return azure.Environment{}, &azblob.SharedKeyCredential{}, fmt.Errorf("unable to parse connection string: %s\n", storageConnection)
	}

	azureEnv, err := azure.EnvironmentFromName("AzurePublicCloud")
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

// GetLeaseFromBlobStorage accesses Blob storage and gets lease information of a partition
func GetLeaseFromBlobStorage(ctx context.Context, partitionID string, storageConnection string, storageContainerName string) (Lease, error) {
	storageAccountName, _, err := ParseAzureStorageConnectionString(storageConnection)
	if err != nil {
		return Lease{}, fmt.Errorf("unable to parse storage connection string: %s", err)
	}

	u, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", storageAccountName, storageContainerName, partitionID))

	_, cred, err := GetStorageCredentials(storageConnection)
	if err != nil {
		return Lease{}, fmt.Errorf("unable to get storage credentials: %s", err)
	}

	// Create a BlockBlobURL object to a blob in the container.
	blobURL := azblob.NewBlockBlobURL(*u, azblob.NewPipeline(cred, azblob.PipelineOptions{}))

	get, err := blobURL.Download(ctx, 0, 0, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return Lease{}, fmt.Errorf("unable to download file from blob storage: %s", err)
	}

	blobData := &bytes.Buffer{}
	reader := get.Body(azblob.RetryReaderOptions{})
	blobData.ReadFrom(reader)
	defer reader.Close() // The client must close the response body when finished with it

	var dat Lease

	if err := json.Unmarshal(blobData.Bytes(), &dat); err != nil {
		return Lease{}, fmt.Errorf("failed to decode blob data: %s", err)
	}

	return dat, nil
}
