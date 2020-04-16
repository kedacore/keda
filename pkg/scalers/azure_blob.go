package scalers

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Azure/azure-storage-blob-go/azblob"
)

// GetAzureBlobListLength returns the count of the blobs in blob container in int
func GetAzureBlobListLength(ctx context.Context, podIdentity string, connectionString, blobContainerName string, accountName string, blobDelimiter string, blobPrefix string) (int, error) {

	var credential azblob.Credential
	var listBlobsSegmentOptions azblob.ListBlobsSegmentOptions
	var err error

	if podIdentity == "" || podIdentity == "none" {

		var accountKey string

		_, accountName, accountKey, _, err = ParseAzureStorageConnectionString(connectionString)

		if err != nil {
			return -1, err
		}

		credential, err = azblob.NewSharedKeyCredential(accountName, accountKey)
		if err != nil {
			return -1, err
		}
	} else if podIdentity == "azure" {
		token, err := getAzureADPodIdentityToken("https://storage.azure.com/")
		if err != nil {
			azureBlobLog.Error(err, "Error fetching token cannot determine blob list count")
			return -1, nil
		}

		credential = azblob.NewTokenCredential(token.AccessToken, nil)
	} else {
		return -1, fmt.Errorf("Azure blobs doesn't support %s pod identity type", podIdentity)

	}

	if blobPrefix != "" {
		listBlobsSegmentOptions.Prefix = blobPrefix
	}

	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	u, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net", accountName))
	serviceURL := azblob.NewServiceURL(*u, p)
	containerURL := serviceURL.NewContainerURL(blobContainerName)

	props, err := containerURL.ListBlobsHierarchySegment(ctx, azblob.Marker{}, blobDelimiter, listBlobsSegmentOptions)
	if err != nil {
		return -1, err
	}

	return len(props.Segment.BlobItems), nil
}
