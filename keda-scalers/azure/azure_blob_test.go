package azure

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/stretchr/testify/assert"
)

func TestGetBlobLength(t *testing.T) {
	meta := BlobMetadata{Connection: "", BlobContainerName: "blobContainerName", AccountName: "", BlobDelimiter: "", BlobPrefix: nil, EndpointSuffix: ""}
	blobClient, err := azblob.NewClientFromConnectionString("DefaultEndpointsProtocol=https;AccountName=name;AccountKey=key=;EndpointSuffix=core.windows.net", nil)
	assert.NoError(t, err)

	length, err := GetAzureBlobListLength(context.TODO(), blobClient, &meta)
	assert.Equal(t, int64(-1), length)
	assert.Error(t, err)
}
