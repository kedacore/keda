//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package azqueue

import (
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/internal/generated"
)

// CreateQueueResponse contains the response from method queue.ServiceClient.Create.
type CreateQueueResponse = generated.QueueClientCreateResponse

// DeleteQueueResponse contains the response from method queue.ServiceClient.Delete
type DeleteQueueResponse = generated.QueueClientDeleteResponse

// ListQueuesResponse contains the response from method ServiceClient.ListQueuesSegment.
type ListQueuesResponse = generated.ServiceClientListQueuesSegmentResponse

// GetServicePropertiesResponse contains the response from method ServiceClient.GetServiceProperties.
type GetServicePropertiesResponse = generated.ServiceClientGetPropertiesResponse

// SetPropertiesResponse contains the response from method ServiceClient.SetProperties.
type SetPropertiesResponse = generated.ServiceClientSetPropertiesResponse

// GetStatisticsResponse contains the response from method ServiceClient.GetStatistics.
type GetStatisticsResponse = generated.ServiceClientGetStatisticsResponse

//------------------------------------------ QUEUES -------------------------------------------------------------------

// CreateResponse contains the response from method QueueClient.Create.
type CreateResponse = generated.QueueClientCreateResponse

// DeleteResponse contains the response from method QueueClient.Delete.
type DeleteResponse = generated.QueueClientDeleteResponse

// SetMetadataResponse contains the response from method QueueClient.SetMetadata.
type SetMetadataResponse = generated.QueueClientSetMetadataResponse

// GetAccessPolicyResponse contains the response from method QueueClient.GetAccessPolicy.
type GetAccessPolicyResponse = generated.QueueClientGetAccessPolicyResponse

// SetAccessPolicyResponse contains the response from method QueueClient.SetAccessPolicy.
type SetAccessPolicyResponse = generated.QueueClientSetAccessPolicyResponse

// GetQueuePropertiesResponse contains the response from method QueueClient.GetProperties.
type GetQueuePropertiesResponse = generated.QueueClientGetPropertiesResponse

// EnqueueMessagesResponse contains the response from method QueueClient.EnqueueMessage.
type EnqueueMessagesResponse = generated.MessagesClientEnqueueResponse

// DequeueMessagesResponse contains the response from method QueueClient.DequeueMessage or QueueClient.DequeueMessages.
type DequeueMessagesResponse = generated.MessagesClientDequeueResponse

// UpdateMessageResponse contains the response from method QueueClient.UpdateMessage.
type UpdateMessageResponse = generated.MessageIDClientUpdateResponse

// DeleteMessageResponse contains the response from method QueueClient.DeleteMessage.
type DeleteMessageResponse = generated.MessageIDClientDeleteResponse

// PeekMessagesResponse contains the response from method QueueClient.PeekMessage or QueueClient.PeekMessages.
type PeekMessagesResponse = generated.MessagesClientPeekResponse

// ClearMessagesResponse contains the response from method QueueClient.ClearMessages.
type ClearMessagesResponse = generated.MessagesClientClearResponse
