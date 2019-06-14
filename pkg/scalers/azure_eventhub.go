package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	eventhub "github.com/Azure/azure-event-hubs-go"
	storageLeaser "github.com/Azure/azure-event-hubs-go/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/azure"
	log "github.com/Sirupsen/logrus"
)

// GetStorageCredentials returns azure env and storage credentials
func GetStorageCredentials(storageConnection string) (azure.Environment, *azblob.SharedKeyCredential, error) {
	storageAccountName, storageAccountKey, err := parseAzureStorageConnectionString(storageConnection)
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

// GetLeaserCheckpointer gets the leaser/checkpointer using storage credentials
func GetLeaserCheckpointer(storageConnection string, storageContainerName string) (*storageLeaser.LeaserCheckpointer, error) {
	storageAccountName, _, err := ParseAzureStorageConnectionString(storageConnection)
	if err != nil {
		return &storageLeaser.LeaserCheckpointer{}, fmt.Errorf("unable to parse storage connection string: %s", err)
	}

	env, cred, err := GetStorageCredentials(storageConnection)
	if err != nil {
		return &storageLeaser.LeaserCheckpointer{}, fmt.Errorf("unable to get storage credentials: %s", err)
	}

	leaserCheckpointer, err := storageLeaser.NewStorageLeaserCheckpointer(
		cred,
		storageAccountName,
		storageContainerName,
		env)
	if err != nil {
		return nil, fmt.Errorf("could not prepare a storage leaserCheckpointer: %s", err)
	}

	return leaserCheckpointer, nil
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
	storageAccountName, _, err := parseAzureStorageConnectionString(storageConnection)
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
		log.Fatal(err)
	}
	blobData := &bytes.Buffer{}
	reader := get.Body(azblob.RetryReaderOptions{})
	blobData.ReadFrom(reader)
	reader.Close() // The client must close the response body when finished with it

	var dat Lease

	if err := json.Unmarshal(blobData.Bytes(), &dat); err != nil {
		log.Fatalf("failed to decode blob data: %s\n", err)
	}

	return dat, nil
}

// ParseAzureStorageConnectionString parses a storage account connection string into (accountName, key)
func parseAzureStorageConnectionString(connectionString string) (string, string, error) {
	parts := strings.Split(connectionString, ";")

	var name, key string
	for _, v := range parts {
		if strings.HasPrefix(v, "AccountName") {
			accountParts := strings.SplitN(v, "=", 2)
			if len(accountParts) == 2 {
				name = accountParts[1]
			}
		} else if strings.HasPrefix(v, "AccountKey") {
			keyParts := strings.SplitN(v, "=", 2)
			if len(keyParts) == 2 {
				key = keyParts[1]
			}
		}
	}
	if name == "" || key == "" {
		return "", "", errors.New("Can't parse connection string")
	}

	return name, key, nil
}
