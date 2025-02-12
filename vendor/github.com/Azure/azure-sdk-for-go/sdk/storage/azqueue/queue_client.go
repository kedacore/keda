//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package azqueue

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/internal/base"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/internal/generated"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/internal/shared"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/queueerror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/sas"
	"time"
)

// QueueClient represents a URL to the Azure Queue Storage service allowing you to manipulate queues.
type QueueClient base.CompositeClient[generated.QueueClient, generated.MessagesClient]

func (q *QueueClient) queueClient() *generated.QueueClient {
	queue, _ := base.InnerClients((*base.CompositeClient[generated.QueueClient, generated.MessagesClient])(q))
	return queue
}

func (q *QueueClient) messagesClient() *generated.MessagesClient {
	_, messages := base.InnerClients((*base.CompositeClient[generated.QueueClient, generated.MessagesClient])(q))
	return messages
}

func (q *QueueClient) getMessageIDURL(messageID string) string {
	return runtime.JoinPaths(q.queueClient().Endpoint(), "messages", messageID)
}

func (q *QueueClient) sharedKey() *SharedKeyCredential {
	return base.SharedKeyComposite((*base.CompositeClient[generated.QueueClient, generated.MessagesClient])(q))
}

// URL returns the URL endpoint used by the ServiceClient object.
func (q *QueueClient) URL() string {
	return q.queueClient().Endpoint()
}

// NewQueueClient creates an instance of ServiceClient with the specified values.
//   - serviceURL - the URL of the storage account e.g. https://<account>.queue.core.windows.net/
//   - cred - an Azure AD credential, typically obtained via the azidentity module
//   - options - client options; pass nil to accept the default values
func NewQueueClient(queueURL string, cred azcore.TokenCredential, options *ClientOptions) (*QueueClient, error) {
	authPolicy := runtime.NewBearerTokenPolicy(cred, []string{shared.TokenScope}, nil)
	conOptions := shared.GetClientOptions(options)
	conOptions.PerRetryPolicies = append(conOptions.PerRetryPolicies, authPolicy)
	pl := runtime.NewPipeline(exported.ModuleName, exported.ModuleVersion, runtime.PipelineOptions{}, &conOptions.ClientOptions)

	return (*QueueClient)(base.NewQueueClient(queueURL, pl, nil)), nil
}

// NewQueueClientWithNoCredential creates an instance of QueueClient with the specified values.
// This is used to anonymously access a storage account or with a shared access signature (SAS) token.
//   - serviceURL - the URL of the storage account e.g. https://<account>.queue.core.windows.net/?<sas token>
//   - options - client options; pass nil to accept the default values
func NewQueueClientWithNoCredential(queueURL string, options *ClientOptions) (*QueueClient, error) {
	conOptions := shared.GetClientOptions(options)
	pl := runtime.NewPipeline(exported.ModuleName, exported.ModuleVersion, runtime.PipelineOptions{}, &conOptions.ClientOptions)

	return (*QueueClient)(base.NewQueueClient(queueURL, pl, nil)), nil
}

// NewQueueClientWithSharedKeyCredential creates an instance of ServiceClient with the specified values.
//   - serviceURL - the URL of the storage account e.g. https://<account>.queue.core.windows.net/
//   - cred - a SharedKeyCredential created with the matching storage account and access key
//   - options - client options; pass nil to accept the default values
func NewQueueClientWithSharedKeyCredential(queueURL string, cred *SharedKeyCredential, options *ClientOptions) (*QueueClient, error) {
	authPolicy := exported.NewSharedKeyCredPolicy(cred)
	conOptions := shared.GetClientOptions(options)
	conOptions.PerRetryPolicies = append(conOptions.PerRetryPolicies, authPolicy)
	pl := runtime.NewPipeline(exported.ModuleName, exported.ModuleVersion, runtime.PipelineOptions{}, &conOptions.ClientOptions)

	return (*QueueClient)(base.NewQueueClient(queueURL, pl, cred)), nil
}

// NewQueueClientFromConnectionString creates an instance of ServiceClient with the specified values.
//   - connectionString - a connection string for the desired storage account
//   - options - client options; pass nil to accept the default values
func NewQueueClientFromConnectionString(connectionString string, queueName string, options *ClientOptions) (*QueueClient, error) {
	parsed, err := shared.ParseConnectionString(connectionString)
	if err != nil {
		return nil, err
	}
	parsed.ServiceURL = runtime.JoinPaths(parsed.ServiceURL, queueName)
	if parsed.AccountKey != "" && parsed.AccountName != "" {
		credential, err := exported.NewSharedKeyCredential(parsed.AccountName, parsed.AccountKey)
		if err != nil {
			return nil, err
		}
		return NewQueueClientWithSharedKeyCredential(parsed.ServiceURL, credential, options)
	}

	return NewQueueClientWithNoCredential(parsed.ServiceURL, options)
}

// Create creates a new queue within a storage account. If a queue with the specified name already exists, and
// the existing metadata is identical to the metadata that's specified on the Create Queue request,
// status code 204 (No Content) is returned. If the existing metadata doesn't match the metadata provided with the Create Queue request,
// the operation fails and status code 409 (Conflict) is returned.
// For more information, see https://learn.microsoft.com/en-us/rest/api/storageservices/create-queue4.
func (q *QueueClient) Create(ctx context.Context, options *CreateOptions) (CreateResponse, error) {
	opts := options.format()
	resp, err := q.queueClient().Create(ctx, opts)
	return resp, err
}

// Delete deletes the specified queue.
// For more information, see https://learn.microsoft.com/en-us/rest/api/storageservices/delete-queue3.
func (q *QueueClient) Delete(ctx context.Context, options *DeleteOptions) (DeleteResponse, error) {
	opts := options.format()
	resp, err := q.queueClient().Delete(ctx, opts)
	return resp, err
}

// SetMetadata sets the metadata for the queue.
// For more information, see https://learn.microsoft.com/en-us/rest/api/storageservices/set-queue-metadata.
func (q *QueueClient) SetMetadata(ctx context.Context, options *SetMetadataOptions) (SetMetadataResponse, error) {
	opts := options.format()
	resp, err := q.queueClient().SetMetadata(ctx, opts)
	return resp, err
}

// GetProperties gets properties including metadata of a queue.
// For more information, see https://learn.microsoft.com/en-us/rest/api/storageservices/get-queue-metadata.
func (q *QueueClient) GetProperties(ctx context.Context, options *GetQueuePropertiesOptions) (GetQueuePropertiesResponse, error) {
	opts := options.format()
	resp, err := q.queueClient().GetProperties(ctx, opts)
	return resp, err
}

// GetAccessPolicy returns the queue's access policy.
// For more information, see https://learn.microsoft.com/en-us/rest/api/storageservices/get-queue-acl.
func (q *QueueClient) GetAccessPolicy(ctx context.Context, o *GetAccessPolicyOptions) (GetAccessPolicyResponse, error) {
	options := o.format()
	resp, err := q.queueClient().GetAccessPolicy(ctx, options)
	return resp, err
}

// SetAccessPolicy sets the queue's permissions.
// For more information, see https://learn.microsoft.com/en-us/rest/api/storageservices/set-queue-acl.
func (q *QueueClient) SetAccessPolicy(ctx context.Context, o *SetAccessPolicyOptions) (SetAccessPolicyResponse, error) {
	opts, acl, err := o.format()
	if err != nil {
		return SetAccessPolicyResponse{}, err
	}
	resp, err := q.queueClient().SetAccessPolicy(ctx, acl, opts)
	return resp, err
}

// EnqueueMessage adds a message to the queue.
// For more information, see https://learn.microsoft.com/en-us/rest/api/storageservices/put-message.
func (q *QueueClient) EnqueueMessage(ctx context.Context, content string, o *EnqueueMessageOptions) (EnqueueMessagesResponse, error) {
	opts := o.format()
	message := generated.QueueMessage{MessageText: &content}
	resp, err := q.messagesClient().Enqueue(ctx, message, opts)
	return resp, err
}

// DequeueMessage removes one message from the queue.
// For more information, see https://learn.microsoft.com/en-us/rest/api/storageservices/get-messages.
func (q *QueueClient) DequeueMessage(ctx context.Context, o *DequeueMessageOptions) (DequeueMessagesResponse, error) {
	opts := o.format()
	resp, err := q.messagesClient().Dequeue(ctx, opts)
	return resp, err
}

// UpdateMessage updates a message from the queue with the given popReceipt.
// For more information, see https://learn.microsoft.com/en-us/rest/api/storageservices/update-message.
func (q *QueueClient) UpdateMessage(ctx context.Context, messageID string, popReceipt string, content string, o *UpdateMessageOptions) (UpdateMessageResponse, error) {
	opts := o.format()
	message := generated.QueueMessage{MessageText: &content}
	messageClient := generated.NewMessageIDClient(q.getMessageIDURL(messageID), q.queueClient().Pipeline())
	resp, err := messageClient.Update(ctx, popReceipt, message, opts)
	return resp, err
}

// DeleteMessage deletes message from queue with the given popReceipt.
// For more information, see https://learn.microsoft.com/en-us/rest/api/storageservices/delete-message2.
func (q *QueueClient) DeleteMessage(ctx context.Context, messageID string, popReceipt string, o *DeleteMessageOptions) (DeleteMessageResponse, error) {
	opts := o.format()
	messageClient := generated.NewMessageIDClient(q.getMessageIDURL(messageID), q.queueClient().Pipeline())
	resp, err := messageClient.Delete(ctx, popReceipt, opts)
	return resp, err
}

// PeekMessage peeks the first message from the queue.
// For more information, see https://learn.microsoft.com/en-us/rest/api/storageservices/peek-messages.
func (q *QueueClient) PeekMessage(ctx context.Context, o *PeekMessageOptions) (PeekMessagesResponse, error) {
	opts := o.format()
	resp, err := q.messagesClient().Peek(ctx, opts)
	return resp, err
}

// DequeueMessages removes one or more messages from the queue.
// For more information, see https://learn.microsoft.com/en-us/rest/api/storageservices/get-messages.
func (q *QueueClient) DequeueMessages(ctx context.Context, o *DequeueMessagesOptions) (DequeueMessagesResponse, error) {
	opts := o.format()
	resp, err := q.messagesClient().Dequeue(ctx, opts)
	return resp, err
}

// PeekMessages peeks one or more messages from the queue
// For more information, see https://learn.microsoft.com/en-us/rest/api/storageservices/peek-messages.
func (q *QueueClient) PeekMessages(ctx context.Context, o *PeekMessagesOptions) (PeekMessagesResponse, error) {
	opts := o.format()
	resp, err := q.messagesClient().Peek(ctx, opts)
	return resp, err
}

// ClearMessages deletes all messages from the queue.
// For more information, see https://learn.microsoft.com/en-us/rest/api/storageservices/clear-messages.
func (q *QueueClient) ClearMessages(ctx context.Context, o *ClearMessagesOptions) (ClearMessagesResponse, error) {
	opts := o.format()
	resp, err := q.messagesClient().Clear(ctx, opts)
	return resp, err
}

// GetSASURL is a convenience method for generating a SAS token for the currently pointed at account.
// It can only be used if the credential supplied during creation was a SharedKeyCredential.
// This validity can be checked with CanGetAccountSASToken().
func (q *QueueClient) GetSASURL(permissions sas.QueuePermissions, expiry time.Time, o *GetSASURLOptions) (string, error) {
	if q.sharedKey() == nil {
		return "", queueerror.MissingSharedKeyCredential
	}

	st := o.format()
	urlParts, err := ParseURL(q.URL())
	if err != nil {
		return "", err
	}
	qps, err := sas.QueueSignatureValues{
		Version:     sas.Version,
		Protocol:    sas.ProtocolHTTPS,
		StartTime:   st,
		ExpiryTime:  expiry.UTC(),
		Permissions: permissions.String(),
		QueueName:   urlParts.QueueName,
	}.SignWithSharedKey(q.sharedKey())
	if err != nil {
		return "", err
	}

	endpoint := q.URL() + "?" + qps.Encode()

	return endpoint, nil
}
