/*
Copyright 2021 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package azure

import (
	"context"
	"strings"

	"github.com/Azure/azure-storage-queue-go/azqueue"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

const (
	maxPeekMessages                int32  = 32
	QueueLengthStrategyAll         string = "all"
	QueueLengthStrategyVisibleOnly string = "visibleonly"
)

// GetAzureQueueLength returns the length of a queue in int, see https://learn.microsoft.com/en-us/azure/storage/queues/storage-dotnet-how-to-use-queues?tabs=dotnet#get-the-queue-length
func GetAzureQueueLength(ctx context.Context, podIdentity kedav1alpha1.AuthPodIdentity, connectionString, queueName, accountName, endpointSuffix, queueLengthStrategy string) (int64, error) {
	credential, endpoint, err := ParseAzureStorageQueueConnection(ctx, podIdentity, connectionString, accountName, endpointSuffix)
	if err != nil {
		return -1, err
	}

	p := azqueue.NewPipeline(credential, azqueue.PipelineOptions{})
	serviceURL := azqueue.NewServiceURL(*endpoint, p)
	queueURL := serviceURL.NewQueueURL(queueName)

	strategy := strings.ToLower(queueLengthStrategy)
	if strategy == QueueLengthStrategyVisibleOnly {
		visibleMessageCount, err := getVisibleCount(ctx, &queueURL, maxPeekMessages)
		if err != nil {
			return -1, err
		}

		// Queue has less messages than we allowed to peek for,
		// so no need to fall back to the 'all' strategy
		if visibleMessageCount < int64(maxPeekMessages) {
			return visibleMessageCount, nil
		}
	}

	// Use the 'all' strategy (visible + invisible messages)
	props, err := queueURL.GetProperties(ctx)
	if err != nil {
		return -1, err
	}

	return int64(props.ApproximateMessagesCount()), nil
}

func getVisibleCount(ctx context.Context, queueURL *azqueue.QueueURL, maxCount int32) (int64, error) {
	messagesURL := queueURL.NewMessagesURL()
	queue, err := messagesURL.Peek(ctx, maxCount)
	if err != nil {
		return 0, err
	}
	num := queue.NumMessages()
	return int64(num), nil
}
