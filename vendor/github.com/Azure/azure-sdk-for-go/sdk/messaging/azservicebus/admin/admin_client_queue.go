// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package admin

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/atom"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/auth"
)

// QueueProperties represents the static properties of the queue.
type QueueProperties struct {
	// LockDuration is the duration a message is locked when using the PeekLock receive mode.
	// Default is 1 minute.
	LockDuration *string

	// MaxSizeInMegabytes - The maximum size of the queue in megabytes, which is the size of memory
	// allocated for the queue.
	// Default is 1024.
	MaxSizeInMegabytes *int32

	// RequiresDuplicateDetection indicates if this queue requires duplicate detection.
	RequiresDuplicateDetection *bool

	// RequiresSession indicates whether the queue supports the concept of sessions.
	// Sessionful-messages follow FIFO ordering.
	// Default is false.
	RequiresSession *bool

	// DefaultMessageTimeToLive is the duration after which the message expires, starting from when
	// the message is sent to Service Bus. This is the default value used when TimeToLive is not
	// set on a message itself.
	DefaultMessageTimeToLive *string

	// DeadLetteringOnMessageExpiration indicates whether this queue has dead letter
	// support when a message expires.
	DeadLetteringOnMessageExpiration *bool

	// DuplicateDetectionHistoryTimeWindow is the duration of duplicate detection history.
	// Default value is 10 minutes.
	DuplicateDetectionHistoryTimeWindow *string

	// MaxDeliveryCount is the maximum amount of times a message can be delivered before it is automatically
	// sent to the dead letter queue.
	// Default value is 10.
	MaxDeliveryCount *int32

	// EnableBatchedOperations indicates whether server-side batched operations are enabled.
	EnableBatchedOperations *bool

	// Status is the current status of the queue.
	Status *EntityStatus

	// AutoDeleteOnIdle is the idle interval after which the queue is automatically deleted.
	AutoDeleteOnIdle *string

	// EnablePartitioning indicates whether the queue is to be partitioned across multiple message brokers.
	EnablePartitioning *bool

	// ForwardTo is the name of the recipient entity to which all the messages sent to the queue
	// are forwarded to.
	ForwardTo *string

	// ForwardDeadLetteredMessagesTo is the absolute URI of the entity to forward dead letter messages
	ForwardDeadLetteredMessagesTo *string

	// UserMetadata is custom metadata that user can associate with the queue.
	UserMetadata *string

	// AuthorizationRules are the authorization rules for this entity.
	AuthorizationRules []AuthorizationRule

	// Maximum size (in KB) of the message payload that can be accepted by the queue. This feature is only available when
	// using Service Bus Premium.
	MaxMessageSizeInKilobytes *int64
}

// QueueRuntimeProperties represent dynamic properties of a queue, such as the ActiveMessageCount.
type QueueRuntimeProperties struct {
	// SizeInBytes - The size of the queue, in bytes.
	SizeInBytes int64

	// CreatedAt is when the entity was created.
	CreatedAt time.Time

	// UpdatedAt is when the entity was last updated.
	UpdatedAt time.Time

	// AccessedAt is when the entity was last updated.
	AccessedAt time.Time

	// TotalMessageCount is the number of messages in the queue.
	TotalMessageCount int64

	// ActiveMessageCount is the number of active messages in the entity.
	ActiveMessageCount int32

	// DeadLetterMessageCount is the number of dead-lettered messages in the entity.
	DeadLetterMessageCount int32

	// ScheduledMessageCount is the number of messages that are scheduled to be enqueued.
	ScheduledMessageCount int32

	// TransferDeadLetterMessageCount is the number of messages transfer-messages which are dead-lettered
	// into transfer-dead-letter subqueue.
	TransferDeadLetterMessageCount int32

	// TransferMessageCount is the number of messages which are yet to be transferred/forwarded to destination entity.
	TransferMessageCount int32
}

// CreateQueueOptions contains the optional parameters for Client.CreateQueue
type CreateQueueOptions struct {
	// Properties for the queue.
	Properties *QueueProperties
}

// CreateQueueResponse contains the response fields for Client.CreateQueue
type CreateQueueResponse struct {
	// QueueName is the name of the queue.
	QueueName string

	QueueProperties
}

// CreateQueue creates a queue with configurable properties.
func (ac *Client) CreateQueue(ctx context.Context, queueName string, options *CreateQueueOptions) (CreateQueueResponse, error) {
	var properties *QueueProperties

	if options != nil {
		properties = options.Properties
	}

	newProps, _, err := ac.createOrUpdateQueueImpl(ctx, queueName, properties, true)

	if err != nil {
		return CreateQueueResponse{}, err
	}

	return CreateQueueResponse{
		QueueName:       queueName,
		QueueProperties: *newProps,
	}, nil
}

// UpdateQueueResponse contains the response fields for Client.UpdateQueue
type UpdateQueueResponse struct {
	// QueueName is the name of the queue.
	QueueName string

	QueueProperties
}

// UpdateQueueOptions contains optional parameters for Client.UpdateQueue
type UpdateQueueOptions struct {
	// for future expansion
}

// UpdateQueue updates an existing queue.
func (ac *Client) UpdateQueue(ctx context.Context, queueName string, properties QueueProperties, options *UpdateQueueOptions) (UpdateQueueResponse, error) {
	newProps, _, err := ac.createOrUpdateQueueImpl(ctx, queueName, &properties, false)

	if err != nil {
		return UpdateQueueResponse{}, err
	}

	return UpdateQueueResponse{
		QueueName:       queueName,
		QueueProperties: *newProps,
	}, err
}

// GetQueueResponse contains the response fields for Client.GetQueue
type GetQueueResponse struct {
	// QueueName is the name of the queue.
	QueueName string

	QueueProperties
}

// GetQueueOptions contains the optional parameters for Client.GetQueue
type GetQueueOptions struct {
	// For future expansion
}

// GetQueue gets a queue by name.
// If the entity does not exist this function will return a nil GetQueueResponse and a nil error.
func (ac *Client) GetQueue(ctx context.Context, queueName string, options *GetQueueOptions) (*GetQueueResponse, error) {
	var atomResp *atom.QueueEnvelope
	_, err := ac.em.Get(ctx, "/"+queueName, &atomResp)

	if err != nil {
		return mapATOMError[GetQueueResponse](err)
	}

	queueItem, err := newQueueItem(atomResp)

	if err != nil {
		return nil, err
	}

	return &GetQueueResponse{
		QueueName:       queueName,
		QueueProperties: queueItem.QueueProperties,
	}, nil
}

// GetQueueRuntimePropertiesResponse contains response fields for Client.GetQueueRuntimeProperties
type GetQueueRuntimePropertiesResponse struct {
	// QueueName is the name of the queue.
	QueueName string

	QueueRuntimeProperties
}

// GetQueueRuntimePropertiesOptions contains optional parameters for client.GetQueueRuntimeProperties
type GetQueueRuntimePropertiesOptions struct {
	// For future expansion
}

// GetQueueRuntimeProperties gets runtime properties of a queue, like the SizeInBytes, or ActiveMessageCount.
// If the entity does not exist this function will return a nil GetQueueRuntimePropertiesResponse and a nil error.
func (ac *Client) GetQueueRuntimeProperties(ctx context.Context, queueName string, options *GetQueueRuntimePropertiesOptions) (*GetQueueRuntimePropertiesResponse, error) {
	var atomResp *atom.QueueEnvelope
	_, err := ac.em.Get(ctx, "/"+queueName, &atomResp)

	if err != nil {
		return mapATOMError[GetQueueRuntimePropertiesResponse](err)
	}

	item, err := newQueueRuntimePropertiesItem(atomResp)

	if err != nil {
		return nil, err
	}

	return &GetQueueRuntimePropertiesResponse{
		QueueName:              queueName,
		QueueRuntimeProperties: item.QueueRuntimeProperties,
	}, nil
}

// DeleteQueueResponse contains response fields for Client.DeleteQueue
type DeleteQueueResponse struct {
}

// DeleteQueueOptions contains optional parameters for Client.DeleteQueue
type DeleteQueueOptions struct {
	// for future expansion
}

// DeleteQueue deletes a queue.
func (ac *Client) DeleteQueue(ctx context.Context, queueName string, options *DeleteQueueOptions) (DeleteQueueResponse, error) {
	resp, err := ac.em.Delete(ctx, "/"+queueName)
	defer atom.CloseRes(ctx, resp)
	return DeleteQueueResponse{}, err
}

// ListQueuesOptions can be used to configure the ListQueues method.
type ListQueuesOptions struct {
	// MaxPageSize is the maximum size of each page of results.
	MaxPageSize int32
}

// ListQueuesResponse contains the response fields for QueuePager.PageResponse
type ListQueuesResponse struct {
	Queues []QueueItem
}

// QueueItem contains the data from the Client.ListQueues pager
type QueueItem struct {
	QueueName string
	QueueProperties
}

// NewListQueuesPager creates a pager that can be used to list queues.
func (ac *Client) NewListQueuesPager(options *ListQueuesOptions) *runtime.Pager[ListQueuesResponse] {
	var pageSize int32

	if options != nil {
		pageSize = options.MaxPageSize
	}

	ep := &entityPager[atom.QueueFeed, atom.QueueEnvelope, QueueItem]{
		convertFn:    newQueueItem,
		baseFragment: "/$Resources/Queues",
		maxPageSize:  pageSize,
		em:           ac.em,
	}

	return runtime.NewPager(runtime.PagingHandler[ListQueuesResponse]{
		More: func(ltr ListQueuesResponse) bool {
			return ep.More()
		},
		Fetcher: func(ctx context.Context, t *ListQueuesResponse) (ListQueuesResponse, error) {
			items, err := ep.Fetcher(ctx)

			if err != nil {
				return ListQueuesResponse{}, err
			}

			return ListQueuesResponse{
				Queues: items,
			}, nil
		},
	})
}

// ListQueuesRuntimePropertiesOptions can be used to configure the ListQueuesRuntimeProperties method.
type ListQueuesRuntimePropertiesOptions struct {
	// MaxPageSize is the maximum size of each page of results.
	MaxPageSize int32
}

// ListQueuesRuntimePropertiesResponse contains the page response for QueueRuntimePropertiesPager.PageResponse
type ListQueuesRuntimePropertiesResponse struct {
	QueueRuntimeProperties []QueueRuntimePropertiesItem
}

// QueueRuntimePropertiesItem contains a single item in the page response for QueueRuntimePropertiesPager.PageResponse
type QueueRuntimePropertiesItem struct {
	// QueueName is the name of the queue.
	QueueName string

	QueueRuntimeProperties
}

// NewListQueuesRuntimePropertiesPager creates a pager that lists the runtime properties for queues.
func (ac *Client) NewListQueuesRuntimePropertiesPager(options *ListQueuesRuntimePropertiesOptions) *runtime.Pager[ListQueuesRuntimePropertiesResponse] {
	var pageSize int32

	if options != nil {
		pageSize = options.MaxPageSize
	}

	ep := &entityPager[atom.QueueFeed, atom.QueueEnvelope, QueueRuntimePropertiesItem]{
		convertFn:    newQueueRuntimePropertiesItem,
		baseFragment: "/$Resources/Queues",
		maxPageSize:  pageSize,
		em:           ac.em,
	}

	return runtime.NewPager(runtime.PagingHandler[ListQueuesRuntimePropertiesResponse]{
		More: func(ltr ListQueuesRuntimePropertiesResponse) bool {
			return ep.More()
		},
		Fetcher: func(ctx context.Context, t *ListQueuesRuntimePropertiesResponse) (ListQueuesRuntimePropertiesResponse, error) {
			items, err := ep.Fetcher(ctx)

			if err != nil {
				return ListQueuesRuntimePropertiesResponse{}, err
			}

			return ListQueuesRuntimePropertiesResponse{
				QueueRuntimeProperties: items,
			}, nil
		},
	})
}

func (ac *Client) createOrUpdateQueueImpl(ctx context.Context, queueName string, props *QueueProperties, creating bool) (*QueueProperties, *http.Response, error) {
	if props == nil {
		props = &QueueProperties{}
	}

	env := newQueueEnvelope(props, ac.em.TokenProvider())

	if !creating {
		ctx = runtime.WithHTTPHeader(ctx, http.Header{
			"If-Match": []string{"*"},
		})
	}

	executeOpts := &atom.ExecuteOptions{
		ForwardTo:           props.ForwardTo,
		ForwardToDeadLetter: props.ForwardDeadLetteredMessagesTo,
	}

	var atomResp *atom.QueueEnvelope

	resp, err := ac.em.Put(ctx, "/"+queueName, env, &atomResp, executeOpts)

	if err != nil {
		return nil, nil, err
	}

	item, err := newQueueItem(atomResp)

	if err != nil {
		return nil, nil, err
	}

	return &item.QueueProperties, resp, nil
}

func newQueueEnvelope(props *QueueProperties, tokenProvider auth.TokenProvider) *atom.QueueEnvelope {
	qpr := &atom.QueueDescription{
		LockDuration:                        props.LockDuration,
		MaxSizeInMegabytes:                  props.MaxSizeInMegabytes,
		RequiresDuplicateDetection:          props.RequiresDuplicateDetection,
		RequiresSession:                     props.RequiresSession,
		DefaultMessageTimeToLive:            props.DefaultMessageTimeToLive,
		DeadLetteringOnMessageExpiration:    props.DeadLetteringOnMessageExpiration,
		DuplicateDetectionHistoryTimeWindow: props.DuplicateDetectionHistoryTimeWindow,
		MaxDeliveryCount:                    props.MaxDeliveryCount,
		EnableBatchedOperations:             props.EnableBatchedOperations,
		Status:                              (*atom.EntityStatus)(props.Status),
		AutoDeleteOnIdle:                    props.AutoDeleteOnIdle,
		EnablePartitioning:                  props.EnablePartitioning,
		ForwardTo:                           props.ForwardTo,
		ForwardDeadLetteredMessagesTo:       props.ForwardDeadLetteredMessagesTo,
		UserMetadata:                        props.UserMetadata,
		AuthorizationRules:                  publicAccessRightsToInternal(props.AuthorizationRules),
		MaxMessageSizeInKilobytes:           props.MaxMessageSizeInKilobytes,
	}

	return atom.WrapWithQueueEnvelope(qpr, tokenProvider)
}

func newQueueItem(env *atom.QueueEnvelope) (*QueueItem, error) {
	desc := env.Content.QueueDescription

	props := &QueueProperties{
		LockDuration:                        desc.LockDuration,
		MaxSizeInMegabytes:                  desc.MaxSizeInMegabytes,
		RequiresDuplicateDetection:          desc.RequiresDuplicateDetection,
		RequiresSession:                     desc.RequiresSession,
		DefaultMessageTimeToLive:            desc.DefaultMessageTimeToLive,
		DeadLetteringOnMessageExpiration:    desc.DeadLetteringOnMessageExpiration,
		DuplicateDetectionHistoryTimeWindow: desc.DuplicateDetectionHistoryTimeWindow,
		MaxDeliveryCount:                    desc.MaxDeliveryCount,
		EnableBatchedOperations:             desc.EnableBatchedOperations,
		Status:                              (*EntityStatus)(desc.Status),
		AutoDeleteOnIdle:                    desc.AutoDeleteOnIdle,
		EnablePartitioning:                  desc.EnablePartitioning,
		ForwardTo:                           desc.ForwardTo,
		ForwardDeadLetteredMessagesTo:       desc.ForwardDeadLetteredMessagesTo,
		UserMetadata:                        desc.UserMetadata,
		AuthorizationRules:                  internalAccessRightsToPublic(desc.AuthorizationRules),
		MaxMessageSizeInKilobytes:           desc.MaxMessageSizeInKilobytes,
	}

	return &QueueItem{
		QueueName:       env.Title,
		QueueProperties: *props,
	}, nil
}

func newQueueRuntimePropertiesItem(env *atom.QueueEnvelope) (*QueueRuntimePropertiesItem, error) {
	desc := env.Content.QueueDescription

	if desc.CountDetails == nil {
		return nil, errors.New("invalid queue runtime properties: no CountDetails element")
	}

	qrt := &QueueRuntimeProperties{
		SizeInBytes:                    int64OrZero(desc.SizeInBytes),
		TotalMessageCount:              int64OrZero(desc.MessageCount),
		ActiveMessageCount:             int32OrZero(desc.CountDetails.ActiveMessageCount),
		DeadLetterMessageCount:         int32OrZero(desc.CountDetails.DeadLetterMessageCount),
		ScheduledMessageCount:          int32OrZero(desc.CountDetails.ScheduledMessageCount),
		TransferDeadLetterMessageCount: int32OrZero(desc.CountDetails.TransferDeadLetterMessageCount),
		TransferMessageCount:           int32OrZero(desc.CountDetails.TransferMessageCount),
	}

	var err error

	if qrt.CreatedAt, err = atom.StringToTime(desc.CreatedAt); err != nil {
		return nil, err
	}

	if qrt.UpdatedAt, err = atom.StringToTime(desc.UpdatedAt); err != nil {
		return nil, err
	}

	if qrt.AccessedAt, err = atom.StringToTime(desc.AccessedAt); err != nil {
		return nil, err
	}

	return &QueueRuntimePropertiesItem{
		QueueName:              env.Title,
		QueueRuntimeProperties: *qrt,
	}, nil
}

func int32OrZero(i *int32) int32 {
	if i == nil {
		return 0
	}

	return *i
}

func int64OrZero(i *int64) int64 {
	if i == nil {
		return 0
	}

	return *i
}
