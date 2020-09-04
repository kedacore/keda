package azure

import (
	"context"

	"github.com/Azure/azure-storage-queue-go/azqueue"
)

// GetAzureQueueLength returns the length of a queue in int
func GetAzureQueueLength(ctx context.Context, podIdentity string, connectionString, queueName string, accountName string) (int32, error) {

	credential, endpoint, err := ParseAzureStorageQueueConnection(podIdentity, connectionString, accountName)
	if err != nil {
		return -1, err
	}

	p := azqueue.NewPipeline(credential, azqueue.PipelineOptions{})
	serviceURL := azqueue.NewServiceURL(*endpoint, p)
	queueURL := serviceURL.NewQueueURL(queueName)
	props, err := queueURL.GetProperties(ctx)
	if err != nil {
		return -1, err
	}

	visibleMessageCount, err := getVisibleCount(&queueURL, 32)
	if err != nil {
		return -1, err
	}
	approximateMessageCount := props.ApproximateMessagesCount()

	if visibleMessageCount == 32 {
		return approximateMessageCount, nil
	}
	
	return visibleMessageCount, nil
}

func getVisibleCount(queueURL *azqueue.QueueURL, maxCount int32) (int32, error) {
	messagesURL := queueURL.NewMessagesURL()
	ctx := context.Background()
	queue, err := messagesURL.Peek(ctx, maxCount)
	if err != nil {
		return 0, err
	}
	num := queue.NumMessages()
	return num, nil
}
