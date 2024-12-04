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
)

// TopicProperties represents the static properties of the topic.
type TopicProperties struct {
	// MaxSizeInMegabytes - The maximum size of the topic in megabytes, which is the size of memory
	// allocated for the topic.
	// Default is 1024.
	MaxSizeInMegabytes *int32

	// RequiresDuplicateDetection indicates if this topic requires duplicate detection.
	RequiresDuplicateDetection *bool

	// DefaultMessageTimeToLive is the duration after which the message expires, starting from when
	// the message is sent to Service Bus. This is the default value used when TimeToLive is not
	// set on a message itself.
	DefaultMessageTimeToLive *string

	// DuplicateDetectionHistoryTimeWindow is the duration of duplicate detection history.
	// Default value is 10 minutes.
	DuplicateDetectionHistoryTimeWindow *string

	// EnableBatchedOperations indicates whether server-side batched operations are enabled.
	EnableBatchedOperations *bool

	// Status is the current status of the topic.
	Status *EntityStatus

	// AutoDeleteOnIdle is the idle interval after which the topic is automatically deleted.
	AutoDeleteOnIdle *string

	// EnablePartitioning indicates whether the topic is to be partitioned across multiple message brokers.
	EnablePartitioning *bool

	// SupportOrdering defines whether ordering needs to be maintained. If true, messages
	// sent to topic will be forwarded to the subscription, in order.
	SupportOrdering *bool

	// UserMetadata is custom metadata that user can associate with the topic.
	UserMetadata *string

	// AuthorizationRules are the authorization rules for this entity.
	AuthorizationRules []AuthorizationRule

	// Maximum size (in KB) of the message payload that can be accepted by the topic. This feature is only available when
	// using Service Bus Premium.
	MaxMessageSizeInKilobytes *int64
}

// TopicRuntimeProperties represent dynamic properties of a topic, such as the ActiveMessageCount.
type TopicRuntimeProperties struct {
	// SizeInBytes - The size of the topic, in bytes.
	SizeInBytes int64

	// CreatedAt is when the entity was created.
	CreatedAt time.Time

	// UpdatedAt is when the entity was last updated.
	UpdatedAt time.Time

	// AccessedAt is when the entity was last updated.
	AccessedAt time.Time

	// SubscriptionCount is the number of subscriptions to the topic.
	SubscriptionCount int32

	// ScheduledMessageCount is the number of messages that are scheduled to be entopicd.
	ScheduledMessageCount int32
}

// CreateTopicResponse contains response fields for Client.CreateTopic
type CreateTopicResponse struct {
	// TopicName is the name of the topic.
	TopicName string

	TopicProperties
}

// CreateTopicOptions contains optional parameters for Client.CreateTopic
type CreateTopicOptions struct {
	// Properties for the topic.
	Properties *TopicProperties
}

// CreateTopic creates a topic using defaults for all options.
func (ac *Client) CreateTopic(ctx context.Context, topicName string, options *CreateTopicOptions) (CreateTopicResponse, error) {
	var properties *TopicProperties

	if options != nil {
		properties = options.Properties
	}

	newProps, _, err := ac.createOrUpdateTopicImpl(ctx, topicName, properties, true)

	if err != nil {
		return CreateTopicResponse{}, err
	}

	return CreateTopicResponse{
		TopicName:       topicName,
		TopicProperties: *newProps,
	}, nil
}

// GetTopicResponse contains response fields for Client.GetTopic
type GetTopicResponse struct {
	// TopicName is the name of the topic.
	TopicName string

	TopicProperties
}

// GetTopicOptions contains optional parameters for Client.GetTopic
type GetTopicOptions struct {
	// For future expansion
}

// GetTopic gets a topic by name.
// If the entity does not exist this function will return a nil GetTopicResponse and a nil error.
func (ac *Client) GetTopic(ctx context.Context, topicName string, options *GetTopicOptions) (*GetTopicResponse, error) {
	var atomResp *atom.TopicEnvelope
	_, err := ac.em.Get(ctx, "/"+topicName, &atomResp)

	if err != nil {
		return mapATOMError[GetTopicResponse](err)
	}

	topicItem, err := newTopicItem(atomResp)

	if err != nil {
		return nil, err
	}

	return &GetTopicResponse{
		TopicName:       topicName,
		TopicProperties: topicItem.TopicProperties,
	}, nil
}

// GetTopicRuntimePropertiesResponse contains the result for Client.GetTopicRuntimeProperties
type GetTopicRuntimePropertiesResponse struct {
	// TopicName is the name of the topic.
	TopicName string

	// Value is the result of the request.
	TopicRuntimeProperties
}

// GetTopicRuntimePropertiesOptions contains optional parameters for Client.GetTopicRuntimeProperties
type GetTopicRuntimePropertiesOptions struct {
	// For future expansion
}

// GetTopicRuntimeProperties gets runtime properties of a topic, like the SizeInBytes, or SubscriptionCount.
// If the entity does not exist this function will return a nil GetTopicRuntimePropertiesResponse and a nil error.
func (ac *Client) GetTopicRuntimeProperties(ctx context.Context, topicName string, options *GetTopicRuntimePropertiesOptions) (*GetTopicRuntimePropertiesResponse, error) {
	var atomResp *atom.TopicEnvelope
	_, err := ac.em.Get(ctx, "/"+topicName, &atomResp)

	if err != nil {
		return mapATOMError[GetTopicRuntimePropertiesResponse](err)
	}

	item, err := newTopicRuntimePropertiesItem(atomResp)

	if err != nil {
		return nil, err
	}

	return &GetTopicRuntimePropertiesResponse{
		TopicName:              topicName,
		TopicRuntimeProperties: item.TopicRuntimeProperties,
	}, nil
}

// TopicItem is the data returned by the Client.ListTopics pager
type TopicItem struct {
	TopicProperties

	TopicName string
}

// ListTopicsResponse contains response fields for the Client.PageResponse method
type ListTopicsResponse struct {
	// Topics is the result of the request.
	Topics []TopicItem
}

// ListTopicsOptions can be used to configure the ListTopics method.
type ListTopicsOptions struct {
	// MaxPageSize is the maximum size of each page of results.
	MaxPageSize int32
}

// NewListTopicsPager creates a pager that can list topics.
func (ac *Client) NewListTopicsPager(options *ListTopicsOptions) *runtime.Pager[ListTopicsResponse] {
	var pageSize int32

	if options != nil {
		pageSize = options.MaxPageSize
	}

	ep := &entityPager[atom.TopicFeed, atom.TopicEnvelope, TopicItem]{
		convertFn:    newTopicItem,
		baseFragment: "/$Resources/Topics",
		maxPageSize:  pageSize,
		em:           ac.em,
	}

	return runtime.NewPager(runtime.PagingHandler[ListTopicsResponse]{
		More: func(ltr ListTopicsResponse) bool {
			return ep.More()
		},
		Fetcher: func(ctx context.Context, t *ListTopicsResponse) (ListTopicsResponse, error) {
			items, err := ep.Fetcher(ctx)

			if err != nil {
				return ListTopicsResponse{}, err
			}

			return ListTopicsResponse{
				Topics: items,
			}, nil
		},
	})
}

// TopicRuntimePropertiesItem contains fields for the Client.ListTopicsRuntimeProperties method
type TopicRuntimePropertiesItem struct {
	TopicRuntimeProperties

	TopicName string
}

// ListTopicsRuntimePropertiesResponse contains response fields for TopicRuntimePropertiesPager.PageResponse
type ListTopicsRuntimePropertiesResponse struct {
	// TopicRuntimeProperties is the result of the request.
	TopicRuntimeProperties []TopicRuntimePropertiesItem
}

// ListTopicsRuntimePropertiesOptions can be used to configure the ListTopicsRuntimeProperties method.
type ListTopicsRuntimePropertiesOptions struct {
	// MaxPageSize is the maximum size of each page of results.
	MaxPageSize int32
}

// NewListTopicsRuntimePropertiesPager creates a pager than can list runtime properties for topics.
func (ac *Client) NewListTopicsRuntimePropertiesPager(options *ListTopicsRuntimePropertiesOptions) *runtime.Pager[ListTopicsRuntimePropertiesResponse] {
	var pageSize int32

	if options != nil {
		pageSize = options.MaxPageSize
	}

	ep := &entityPager[atom.TopicFeed, atom.TopicEnvelope, TopicRuntimePropertiesItem]{
		convertFn:    newTopicRuntimePropertiesItem,
		baseFragment: "/$Resources/Topics",
		maxPageSize:  pageSize,
		em:           ac.em,
	}

	return runtime.NewPager(runtime.PagingHandler[ListTopicsRuntimePropertiesResponse]{
		More: func(ltr ListTopicsRuntimePropertiesResponse) bool {
			return ep.More()
		},
		Fetcher: func(ctx context.Context, t *ListTopicsRuntimePropertiesResponse) (ListTopicsRuntimePropertiesResponse, error) {
			items, err := ep.Fetcher(ctx)

			if err != nil {
				return ListTopicsRuntimePropertiesResponse{}, err
			}

			return ListTopicsRuntimePropertiesResponse{
				TopicRuntimeProperties: items,
			}, nil
		},
	})
}

// UpdateTopicResponse contains response fields for Client.UpdateTopic
type UpdateTopicResponse struct {
	// TopicName is the name of the topic.
	TopicName string

	TopicProperties
}

// UpdateTopicOptions contains optional parameters for Client.UpdateTopic
type UpdateTopicOptions struct {
	// For future expansion
}

// UpdateTopic updates an existing topic.
func (ac *Client) UpdateTopic(ctx context.Context, topicName string, properties TopicProperties, options *UpdateTopicOptions) (UpdateTopicResponse, error) {
	newProps, _, err := ac.createOrUpdateTopicImpl(ctx, topicName, &properties, false)

	if err != nil {
		return UpdateTopicResponse{}, err
	}

	return UpdateTopicResponse{
		TopicName:       topicName,
		TopicProperties: *newProps,
	}, nil
}

// DeleteTopicResponse contains the response fields for Client.DeleteTopic
type DeleteTopicResponse struct {
	// Value is the result of the request.
	Value *TopicProperties
}

// DeleteTopicOptions contains optional parameters for Client.DeleteTopic
type DeleteTopicOptions struct {
	// For future expansion
}

// DeleteTopic deletes a topic.
func (ac *Client) DeleteTopic(ctx context.Context, topicName string, options *DeleteTopicOptions) (DeleteTopicResponse, error) {
	resp, err := ac.em.Delete(ctx, "/"+topicName)
	defer atom.CloseRes(ctx, resp)
	return DeleteTopicResponse{}, err
}

func (ac *Client) createOrUpdateTopicImpl(ctx context.Context, topicName string, props *TopicProperties, creating bool) (*TopicProperties, *http.Response, error) {
	if props == nil {
		props = &TopicProperties{}
	}

	env := newTopicEnvelope(props)

	if !creating {
		ctx = runtime.WithHTTPHeader(ctx, http.Header{
			"If-Match": []string{"*"},
		})
	}

	var atomResp *atom.TopicEnvelope
	resp, err := ac.em.Put(ctx, "/"+topicName, env, &atomResp, nil)

	if err != nil {
		return nil, nil, err
	}

	topicItem, err := newTopicItem(atomResp)

	if err != nil {
		return nil, nil, err
	}

	return &topicItem.TopicProperties, resp, nil
}

func newTopicEnvelope(props *TopicProperties) *atom.TopicEnvelope {
	desc := &atom.TopicDescription{
		DefaultMessageTimeToLive:            props.DefaultMessageTimeToLive,
		MaxSizeInMegabytes:                  props.MaxSizeInMegabytes,
		RequiresDuplicateDetection:          props.RequiresDuplicateDetection,
		DuplicateDetectionHistoryTimeWindow: props.DuplicateDetectionHistoryTimeWindow,
		EnableBatchedOperations:             props.EnableBatchedOperations,

		Status:                    (*atom.EntityStatus)(props.Status),
		UserMetadata:              props.UserMetadata,
		SupportOrdering:           props.SupportOrdering,
		AutoDeleteOnIdle:          props.AutoDeleteOnIdle,
		EnablePartitioning:        props.EnablePartitioning,
		AuthorizationRules:        publicAccessRightsToInternal(props.AuthorizationRules),
		MaxMessageSizeInKilobytes: props.MaxMessageSizeInKilobytes,
	}

	return atom.WrapWithTopicEnvelope(desc)
}

func newTopicItem(te *atom.TopicEnvelope) (*TopicItem, error) {
	td := te.Content.TopicDescription

	return &TopicItem{
		TopicName: te.Title,
		TopicProperties: TopicProperties{
			MaxSizeInMegabytes:                  td.MaxSizeInMegabytes,
			RequiresDuplicateDetection:          td.RequiresDuplicateDetection,
			DefaultMessageTimeToLive:            td.DefaultMessageTimeToLive,
			DuplicateDetectionHistoryTimeWindow: td.DuplicateDetectionHistoryTimeWindow,
			EnableBatchedOperations:             td.EnableBatchedOperations,
			Status:                              (*EntityStatus)(td.Status),
			UserMetadata:                        td.UserMetadata,
			AutoDeleteOnIdle:                    td.AutoDeleteOnIdle,
			EnablePartitioning:                  td.EnablePartitioning,
			SupportOrdering:                     td.SupportOrdering,
			AuthorizationRules:                  internalAccessRightsToPublic(td.AuthorizationRules),
			MaxMessageSizeInKilobytes:           td.MaxMessageSizeInKilobytes,
		},
	}, nil
}

func newTopicRuntimePropertiesItem(env *atom.TopicEnvelope) (*TopicRuntimePropertiesItem, error) {
	desc := env.Content.TopicDescription

	if desc.CountDetails == nil {
		return nil, errors.New("invalid topic runtime properties: no CountDetails element")
	}

	props := &TopicRuntimeProperties{
		SizeInBytes:           int64OrZero(desc.SizeInBytes),
		ScheduledMessageCount: int32OrZero(desc.CountDetails.ScheduledMessageCount),
		SubscriptionCount:     int32OrZero(desc.SubscriptionCount),
	}

	var err error

	if props.CreatedAt, err = atom.StringToTime(desc.CreatedAt); err != nil {
		return nil, err
	}

	if props.UpdatedAt, err = atom.StringToTime(desc.UpdatedAt); err != nil {
		return nil, err
	}

	if props.AccessedAt, err = atom.StringToTime(desc.AccessedAt); err != nil {
		return nil, err
	}

	return &TopicRuntimePropertiesItem{
		TopicName:              env.Title,
		TopicRuntimeProperties: *props,
	}, nil
}
