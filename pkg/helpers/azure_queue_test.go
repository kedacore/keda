package helpers

import (
	"strings"
	"context"
	"testing"
)

type parseConnectionStringTestData struct {
	connectionString string
	accountName      string
	accountKey       string
	isError          bool
}

var parseConnectionStringTestDataset = []parseConnectionStringTestData{
	{"AccountName=testing;AccountKey=key==;DnsSuffix=example.com", "testing", "key==", false},
	{"AccountName=testing;AccountKey=key==", "testing", "key==", false},
	{"AccountName=testingAccountKey=key==", "", "", true},
	{"", "", "", true},
}

func TestParseConnectionString(t *testing.T) {
	for _, testData := range parseConnectionStringTestDataset {
		accountName, accountKey, err := ParseAzureStorageConnectionString(testData.connectionString)

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
	length, err := GetAzureQueueLength(context.TODO(), "", "queueName")
	if length != -1 {
		t.Error("Expected length to be -1, but got", length)
	}

	if err == nil {
		t.Error("Expected error for empty connection string, but got nil")
	}

	if !strings.Contains(err.Error(), "parse connection string") {
		t.Error("Expected error to contain parsing error message, but got", err.Error())
	}

	length, err = GetAzureQueueLength(context.TODO(), "AccountName=name;AccountKey=key==", "queueName")

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