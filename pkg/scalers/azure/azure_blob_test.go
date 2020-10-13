package azure

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestGetBlobLength(t *testing.T) {
	httpClient := http.DefaultClient
	length, err := GetAzureBlobListLength(context.TODO(), httpClient, "", "", "blobContainerName", "", "", "")
	if length != -1 {
		t.Error("Expected length to be -1, but got", length)
	}

	if err == nil {
		t.Error("Expected error for empty connection string, but got nil")
	}

	if !strings.Contains(err.Error(), "parse storage connection string") {
		t.Error("Expected error to contain parsing error message, but got", err.Error())
	}

	length, err = GetAzureBlobListLength(context.TODO(), httpClient, "", "DefaultEndpointsProtocol=https;AccountName=name;AccountKey=key==;EndpointSuffix=core.windows.net", "blobContainerName", "", "", "")

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
