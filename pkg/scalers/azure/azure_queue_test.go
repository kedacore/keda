package azure

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

func TestGetQueueLength(t *testing.T) {
	length, err := GetAzureQueueLength(context.TODO(), kedav1alpha1.AuthPodIdentity{}, "", "queueName", "", "")
	if length != -1 {
		t.Error("Expected length to be -1, but got", length)
	}

	if err == nil {
		t.Error("Expected error for empty connection string, but got nil")
	}

	if !errors.Is(err, ErrAzureConnectionStringKeyName) {
		t.Error("Expected error to contain parsing error message, but got", err.Error())
	}

	length, err = GetAzureQueueLength(context.TODO(), kedav1alpha1.AuthPodIdentity{}, "DefaultEndpointsProtocol=https;AccountName=name;AccountKey=key==;EndpointSuffix=core.windows.net", "queueName", "", "")

	if length != -1 {
		t.Error("Expected length to be -1, but got", length)
	}

	if err == nil {
		t.Error("Expected error for empty connection string, but got nil")
	}

	var base64Error base64.CorruptInputError
	if !errors.As(err, &base64Error) {
		t.Error("Expected error to contain base64 error message, but got", err.Error())
	}
}
