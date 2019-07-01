package scalers

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Azure/azure-storage-queue-go/azqueue"
)

// GetAzureQueueLength returns the length of a queue in int
func GetAzureQueueLength(ctx context.Context, connectionString, queueName string) (int32, error) {
	_, accountName, accountKey, _, err := ParseAzureStorageConnectionString(connectionString)

	if err != nil {
		return -1, err
	}

	credential, err := azqueue.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return -1, err
	}

	p := azqueue.NewPipeline(credential, azqueue.PipelineOptions{})
	u, _ := url.Parse(fmt.Sprintf("https://%s.queue.core.windows.net", accountName))
	serviceURL := azqueue.NewServiceURL(*u, p)
	queueURL := serviceURL.NewQueueURL(queueName)
	_, err = queueURL.Create(ctx, azqueue.Metadata{})
	if err != nil {
		return -1, err
	}

	props, err := queueURL.GetProperties(ctx)
	if err != nil {
		return -1, err
	}

	return props.ApproximateMessagesCount(), nil
}
