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

	"github.com/Azure/azure-storage-queue-go/azqueue"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/util"
)

// GetAzureQueueLength returns the length of a queue in int, see https://learn.microsoft.com/en-us/azure/storage/queues/storage-dotnet-how-to-use-queues?tabs=dotnet#get-the-queue-length
func GetAzureQueueLength(ctx context.Context, httpClient util.HTTPDoer, podIdentity kedav1alpha1.AuthPodIdentity, connectionString, queueName, accountName, endpointSuffix string) (int64, error) {
	credential, endpoint, err := ParseAzureStorageQueueConnection(ctx, httpClient, podIdentity, connectionString, accountName, endpointSuffix)
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

	return int64(props.ApproximateMessagesCount()), nil
}
