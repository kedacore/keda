package scalers

import (
	"context"
	"strings"
	"testing"
)

func TestGetServiceBusQueueLength(t *testing.T) {
	length, err := GetAzureServiceBusQueueLength(context.TODO(), "", "queueName")
	if length != -1 {
		t.Error("Expected length to be -1, but got", length)
	}

	if err == nil {
		t.Error("Expected error for empty connection string, but got nil")
	}

	if !strings.Contains(err.Error(), "failed parsing connection string") {
		t.Error("Expected error to contain parsing error message, but got", err.Error())
	}
}
