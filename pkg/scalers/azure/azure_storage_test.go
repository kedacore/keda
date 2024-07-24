package azure

import "testing"

type parseAzureStorageEndpointSuffixTestData struct {
	metadata       map[string]string
	endpointSuffix string
	endpointType   StorageEndpointType
	isError        bool
}

var parseAzureStorageEndpointSuffixTestDataset = []parseAzureStorageEndpointSuffixTestData{
	{map[string]string{}, "queue.core.windows.net", QueueEndpoint, false},
	{map[string]string{"cloud": "InvalidCloud"}, "", QueueEndpoint, true},
	{map[string]string{"cloud": "AzureUSGovernmentCloud"}, "queue.core.usgovcloudapi.net", QueueEndpoint, false},
	{map[string]string{"cloud": "Private"}, "", BlobEndpoint, true},
	{map[string]string{"cloud": "Private", "endpointSuffix": "blob.core.private.cloud"}, "blob.core.private.cloud", BlobEndpoint, false},
	{map[string]string{"endpointSuffix": "ignored"}, "blob.core.windows.net", BlobEndpoint, false},
}

func TestParseAzureStorageEndpointSuffix(t *testing.T) {
	for _, testData := range parseAzureStorageEndpointSuffixTestDataset {
		endpointSuffix, err := ParseAzureStorageEndpointSuffix(testData.metadata, testData.endpointType)
		if !testData.isError && err != nil {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if err == nil {
			if endpointSuffix != testData.endpointSuffix {
				t.Error(
					"For", testData.metadata,
					"expected endpointSuffix=", testData.endpointSuffix,
					"but got", endpointSuffix)
			}
		}
	}
}
