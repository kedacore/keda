package scalers

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/azure-storage-queue-go/azqueue"
)

func getQueueLength(ctx context.Context, connectionString, queueName string) (int32, error) {
	// From the Azure portal, get your Storage account's name and account key.
	accountName, accountKey, err := parseConnectionString(connectionString)

	if err != nil {
		return -1, err
	}

	// Use your Storage account's name and key to create a credential object; this is used to access your account.
	credential, err := azqueue.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return -1, err
	}

	// Create a request pipeline that is used to process HTTP(S) requests and responses. It requires
	// your account credentials. In more advanced scenarios, you can configure telemetry, retry policies,
	// logging, and other options. Also, you can configure multiple request pipelines for different scenarios.
	p := azqueue.NewPipeline(credential, azqueue.PipelineOptions{})

	// From the Azure portal, get your Storage account queue service URL endpoint.
	// The URL typically looks like this:
	// https throws in aks, investigate
	u, _ := url.Parse(fmt.Sprintf("http://%s.queue.core.windows.net", accountName))

	// Create an ServiceURL object that wraps the service URL and a request pipeline.
	serviceURL := azqueue.NewServiceURL(*u, p)

	// Now, you can use the serviceURL to perform various queue operations.

	// All HTTP operations allow you to specify a Go context.Context object to control cancellation/timeout.

	// Create a URL that references a queue in your Azure Storage account.
	// This returns a QueueURL object that wraps the queue's URL and a request pipeline (inherited from serviceURL)
	queueURL := serviceURL.NewQueueURL(queueName) // Queue names require lowercase

	// The code below shows how to create the queue. It is common to create a queue and never delete it:
	_, err = queueURL.Create(ctx, azqueue.Metadata{})
	if err != nil {
		return -1, err
	}

	// The code below shows how a client or server can determine the approximate count of messages in the queue:
	props, err := queueURL.GetProperties(ctx)
	if err != nil {
		return -1, err
	}

	return props.ApproximateMessagesCount(), nil
}

func parseConnectionString(connectionString string) (string, string, error) {
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
