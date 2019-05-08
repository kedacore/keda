package scalers

import (
	"context"
	"testing"
)

func TestGetQueueLengthReal(t *testing.T) {
	length, err := GetAwsSqsQueueLength(context.TODO(), "")
	t.Error(err)
	t.Log("QueueLength = ", length)

	if length != -1 {
		t.Error("Expected length to be -1, but got", length)
	}
}
