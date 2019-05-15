package scalers

import (
	"context"
	"os"
	"testing"
)

func TestGetQueueLengthReal(t *testing.T) {
	queueURL := os.Getenv("AWS_SQS_QUEUE_URL")

	t.Log("This test will use the environment variable AWS_SQS_QUEUE_URL if it is set")
	t.Log("Ensure that AWS credentials are configured to be able to access the queue")
	t.Log("If set, it will connect to the specified SQS Queue & check:")
	t.Logf("\tQueue '%s' has 0 message\n", queueURL)

	length, err := GetAwsSqsQueueLength(context.TODO(), queueURL)
	if err != nil {
		t.Error(err)
	}
	t.Log("QueueLength = ", length)

	if length != 0 {
		t.Error("Expected length to be 0, but got", length)
	}
}
