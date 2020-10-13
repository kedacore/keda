package azure

import (
	"context"

	"github.com/Azure/azure-storage-queue-go/azqueue"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/util"
)

// GetAzureQueueLength returns the length of a queue in int
func GetAzureQueueLength(ctx context.Context, httpClient util.HTTPDoer, podIdentity kedav1alpha1.PodIdentityProvider, connectionString, queueName string, accountName string) (int32, error) {
	credential, endpoint, err := ParseAzureStorageQueueConnection(httpClient, podIdentity, connectionString, accountName)
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
