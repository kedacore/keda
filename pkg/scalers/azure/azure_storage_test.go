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
