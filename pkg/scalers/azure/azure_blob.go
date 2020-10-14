package azure

import (
	"context"

	"github.com/Azure/azure-storage-blob-go/azblob"

	kedav1alpha1 "github.com/kedacore/keda/api/v1alpha1"
)

// GetAzureBlobListLength returns the count of the blobs in blob container in int
func GetAzureBlobListLength(ctx context.Context, podIdentity kedav1alpha1.PodIdentityProvider, connectionString, blobContainerName string, accountName string, blobDelimiter string, blobPrefix string) (int, error) {
	credential, endpoint, err := ParseAzureStorageBlobConnection(podIdentity, connectionString, accountName)
	if err != nil {
		return -1, err
	}

	listBlobsSegmentOptions := azblob.ListBlobsSegmentOptions{
		Prefix: blobPrefix,
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	serviceURL := azblob.NewServiceURL(*endpoint, p)
	containerURL := serviceURL.NewContainerURL(blobContainerName)

	props, err := containerURL.ListBlobsHierarchySegment(ctx, azblob.Marker{}, blobDelimiter, listBlobsSegmentOptions)
	if err != nil {
		return -1, err
	}

	return len(props.Segment.BlobItems), nil
}
