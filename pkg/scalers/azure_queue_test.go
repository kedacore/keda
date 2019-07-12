package scalers

import (
	"context"
	"strings"
	"testing"
)

type parseConnectionStringTestData struct {
	connectionString string
	accountName      string
	accountKey       string
	isError          bool
}

var parseConnectionStringTestDataset = []parseConnectionStringTestData{
	{"DefaultEndpointsProtocol=https;AccountName=testing;AccountKey=key==;EndpointSuffix=core.windows.net", "testing", "key==", false},
	{"DefaultEndpointsProtocol=https;AccountName=testing;AccountKey=key==;EndpointSuffix=core.windows.net", "testing", "key==", false},
	{"AccountName=testingAccountKey=key==", "", "", true},
	{"", "", "", true},
}

func TestParseStorageConnectionString(t *testing.T) {
	for _, testData := range parseConnectionStringTestDataset {
		_, accountName, accountKey, _, err := ParseAzureStorageConnectionString(testData.connectionString)

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
	}
}

func TestGetQueueLength(t *testing.T) {
	length, err := GetAzureQueueLength(context.TODO(), false, "", "queueName", "")
	if length != -1 {
		t.Error("Expected length to be -1, but got", length)
	}

	if err == nil {
		t.Error("Expected error for empty connection string, but got nil")
	}

	if !strings.Contains(err.Error(), "parse storage connection string") {
		t.Error("Expected error to contain parsing error message, but got", err.Error())
	}

	length, err = GetAzureQueueLength(context.TODO(), false, "DefaultEndpointsProtocol=https;AccountName=name;AccountKey=key==;EndpointSuffix=core.windows.net", "queueName", "")

	if length != -1 {
		t.Error("Expected length to be -1, but got", length)
	}

	if err == nil {
		t.Error("Expected error for empty connection string, but got nil")
	}

	if !strings.Contains(err.Error(), "illegal base64") {
		t.Error("Expected error to contain base64 error message, but got", err.Error())
	}
}

var testAzQueueResolvedEnv = map[string]string{
	"CONNECTION": "SAMPLE",
}

type parseAzQueueMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

var testAzQueueMetadata = []parseAzQueueMetadataTestData{
	// nothing passed
	{map[string]string{}, true},
	// properly formed
	{map[string]string{"connection": "CONNECTION", "queueName": "sample", "queueLength": "5"}, false},
	// Empty queueName
	{map[string]string{"connection": "CONNECTION", "queueName": ""}, true},
	// improperly formed queueLength
	{map[string]string{"connection": "CONNECTION", "queueName": "sample", "queueLength": "AA"}, true},
	// useAAdPodIdentity with account name
	{map[string]string{"useAAdPodIdentity": "true", "accountName": "sample_acc", "queueName": "sample_queue"}, false},
	// useAAdPodIdentity without account name
	{map[string]string{"useAAdPodIdentity": "true", "accountName": "", "queueName": "sample_queue"}, true},
	// useAAdPodIdentity without queue name
	{map[string]string{"useAAdPodIdentity": "true", "accountName": "sample_acc", "queueName": ""}, true},
}

func TestAzQueueParseMetadata(t *testing.T) {
	for _, testData := range testAzQueueMetadata {
		_, err := parseAzureQueueMetadata(testData.metadata, testAzQueueResolvedEnv)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}
