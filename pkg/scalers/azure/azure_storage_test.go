package azure

import "testing"

type parseConnectionStringTestData struct {
	connectionString string
	accountName      string
	accountKey       string
	endpoint         string
	endpointType     StorageEndpointType
	isError          bool
}

var parseConnectionStringTestDataset = []parseConnectionStringTestData{
	{"DefaultEndpointsProtocol=https;AccountName=testing;AccountKey=key==;EndpointSuffix=core.windows.net", "testing", "key==", "https://testing.queue.core.windows.net", QueueEndpoint, false},
	{"DefaultEndpointsProtocol=https;AccountName=testing;AccountKey=key==;EndpointSuffix=core.windows.net", "testing", "key==", "https://testing.blob.core.windows.net", BlobEndpoint, false},
	{"DefaultEndpointsProtocol=https;AccountName=testing;AccountKey=key==;EndpointSuffix=core.windows.net", "testing", "key==", "https://testing.table.core.windows.net", TableEndpoint, false},
	{"DefaultEndpointsProtocol=https;AccountName=testing;AccountKey=key==;EndpointSuffix=core.windows.net", "testing", "key==", "https://testing.file.core.windows.net", FileEndpoint, false},
	{"AccountName=testingAccountKey=key==", "", "", "", QueueEndpoint, true},
	{"", "", "", "", QueueEndpoint, true},
	{"DefaultEndpointsProtocol=https;AccountName=testing;AccountKey=key==;EndpointSuffix=core.windows.net;QueueEndpoint=https://queue.net", "testing", "key==", "https://queue.net", QueueEndpoint, false},
	{"DefaultEndpointsProtocol=https;AccountName=testing;AccountKey=key==;EndpointSuffix=core.windows.net;BlobEndpoint=https://blob.net", "testing", "key==", "https://blob.net", BlobEndpoint, false},
	{"DefaultEndpointsProtocol=https;AccountName=testing;AccountKey=key==;EndpointSuffix=core.windows.net;TableEndpoint=https://table.net", "testing", "key==", "https://table.net", TableEndpoint, false},
	{"DefaultEndpointsProtocol=https;AccountName=testing;AccountKey=key==;EndpointSuffix=core.windows.net;FileEndpoint=https://file.net", "testing", "key==", "https://file.net", FileEndpoint, false},
	{"QueueEndpoint=https://queue.net;SharedAccessSignature=sv=2012-02-12&st=2009-02-09&se=2009-02-10&sr=c&sp=r&si=YWJjZGVmZw%3d%3d&sig=dD80ihBh5jfNpymO5Hg1IdiJIEvHcJpCMiCMnN%2fRnbI%3d", "", "", "https://queue.net?sv=2012-02-12&st=2009-02-09&se=2009-02-10&sr=c&sp=r&si=YWJjZGVmZw%3d%3d&sig=dD80ihBh5jfNpymO5Hg1IdiJIEvHcJpCMiCMnN%2fRnbI%3d", QueueEndpoint, false},
	{"BlobEndpoint=https://blob.net;SharedAccessSignature=sv=2012-02-12&st=2009-02-09&se=2009-02-10&sr=c&sp=r&si=YWJjZGVmZw%3d%3d&sig=dD80ihBh5jfNpymO5Hg1IdiJIEvHcJpCMiCMnN%2fRnbI%3d", "", "", "https://blob.net?sv=2012-02-12&st=2009-02-09&se=2009-02-10&sr=c&sp=r&si=YWJjZGVmZw%3d%3d&sig=dD80ihBh5jfNpymO5Hg1IdiJIEvHcJpCMiCMnN%2fRnbI%3d", BlobEndpoint, false},
}

func TestParseStorageConnectionString(t *testing.T) {
	for _, testData := range parseConnectionStringTestDataset {
		endpoint, accountName, accountKey, err := parseAzureStorageConnectionString(testData.connectionString, testData.endpointType)

		if !testData.isError && err != nil {
			t.Error("Expected success but got err", err)
		}

		if testData.isError && err == nil {
			t.Error("Expected error but got nil")
		}

		if accountName != testData.accountName {
			t.Error(
				"For", testData.connectionString,
				"expected accountName=", testData.accountName,
				"but got", accountName)
		}

		if accountKey != testData.accountKey {
			t.Error(
				"For", testData.connectionString,
				"expected accountKey=", testData.accountKey,
				"but got", accountKey)
		}

		if err == nil {
			if endpoint.String() != testData.endpoint {
				t.Error(
					"For", testData.connectionString,
					"expected endpoint=", testData.endpoint,
					"but got", endpoint)
			}
		}
	}
}

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
