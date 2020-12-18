package azure

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestGetQueueLength(t *testing.T) {
	length, err := GetAzureQueueLength(context.TODO(), http.DefaultClient, "", "", "queueName", "")
	if length != -1 {
		t.Error("Expected length to be -1, but got", length)
	}

	if err == nil {
		t.Error("Expected error for empty connection string, but got nil")
	}

	if !strings.Contains(err.Error(), "parse storage connection string") {
		t.Error("Expected error to contain parsing error message, but got", err.Error())
	}

	length, err = GetAzureQueueLength(context.TODO(), http.DefaultClient, "", "DefaultEndpointsProtocol=https;AccountName=name;AccountKey=key==;EndpointSuffix=core.windows.net", "queueName", "")

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
