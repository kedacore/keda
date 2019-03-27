package scalers

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/azure-storage-queue-go/azqueue"
)

// GetAzureQueueLength returns the length of a queue in int
func GetAzureQueueLength(ctx context.Context, connectionString, queueName string) (int32, error) {
	accountName, accountKey, err := ParseAzureStorageConnectionString(connectionString)

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

// ParseAzureStorageConnectionString parses a storage account connection string into (accountName, key)
func ParseAzureStorageConnectionString(connectionString string) (string, string, error) {
	parts := strings.Split(connectionString, ";")

	var name, key string
	for _, v := range parts {
		if strings.HasPrefix(v, "AccountName") {
			accountParts := strings.SplitN(v, "=", 2)
			if len(accountParts) == 2 {
				name = accountParts[1]
			}
		} else if strings.HasPrefix(v, "AccountKey") {
			keyParts := strings.SplitN(v, "=", 2)
			if len(keyParts) == 2 {
				key = keyParts[1]
			}
		}
	}
	if name == "" || key == "" {
		return "", "", errors.New("Can't parse connection string")
	}

	return name, key, nil
}
