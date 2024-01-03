// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package admin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/atom"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/auth"
)

// SubscriptionProperties represents the static properties of the subscription.
type SubscriptionProperties struct {
	// LockDuration is the duration a message is locked when using the PeekLock receive mode.
	// Default is 1 minute.
	LockDuration *string

	// RequiresSession indicates whether the subscription supports the concept of sessions.
	// Sessionful-messages follow FIFO ordering.
	// Default is false.
	RequiresSession *bool

	// DefaultMessageTimeToLive is the duration after which the message expires, starting from when
	// the message is sent to Service Bus. This is the default value used when TimeToLive is not
	// set on a message itself.
	DefaultMessageTimeToLive *string

	// DeadLetteringOnMessageExpiration indicates whether this subscription has dead letter
	// support when a message expires.
	DeadLetteringOnMessageExpiration *bool

	// EnableDeadLetteringOnFilterEvaluationExceptions indicates whether messages need to be
	// forwarded to dead-letter sub queue when subscription rule evaluation fails.
	EnableDeadLetteringOnFilterEvaluationExceptions *bool

	// MaxDeliveryCount is the maximum amount of times a message can be delivered before it is automatically
	// sent to the dead letter queue.
	// Default value is 10.
	MaxDeliveryCount *int32

	// Status is the current status of the subscription.
	Status *EntityStatus

	// AutoDeleteOnIdle is the idle interval after which the subscription is automatically deleted.
	AutoDeleteOnIdle *string

	// ForwardTo is the name of the recipient entity to which all the messages sent to the topic
	// are forwarded to.
	ForwardTo *string

	// ForwardDeadLetteredMessagesTo is the absolute URI of the entity to forward dead letter messages
	ForwardDeadLetteredMessagesTo *string

	// EnableBatchedOperations indicates whether server-side batched operations are enabled.
	EnableBatchedOperations *bool

	// UserMetadata is custom metadata that user can associate with the subscription.
	UserMetadata *string

	// DefaultRule is a rule that is added to the subscription as soon as it is created.
	DefaultRule *RuleProperties
}

// SubscriptionRuntimeProperties represent dynamic properties of a subscription, such as the ActiveMessageCount.
type SubscriptionRuntimeProperties struct {
	// TotalMessageCount is the number of messages in the subscription.
	TotalMessageCount int64

	// ActiveMessageCount is the number of active messages in the entity.
	ActiveMessageCount int32

	// DeadLetterMessageCount is the number of dead-lettered messages in the entity.
	DeadLetterMessageCount int32

	// TransferMessageCount is the number of messages which are yet to be transferred/forwarded to destination entity.
	TransferMessageCount int32

	// TransferDeadLetterMessageCount is the number of messages transfer-messages which are dead-lettered
	// into transfer-dead-letter subqueue.
	TransferDeadLetterMessageCount int32

	// AccessedAt is when the entity was last updated.
	AccessedAt time.Time

	// CreatedAt is when the entity was created.
	CreatedAt time.Time

	// UpdatedAt is when the entity was last updated.
	UpdatedAt time.Time
}

// CreateSubscriptionResponse contains response fields for Client.CreateSubscription
type CreateSubscriptionResponse struct {
	// SubscriptionName is the name of the subscription.
	SubscriptionName string

	// TopicName is the name of the topic for this subscription.
	TopicName string

	SubscriptionProperties
}

// CreateSubscriptionOptions contains optional parameters for Client.CreateSubscription
type CreateSubscriptionOptions struct {
	// Properties for the subscription.
	Properties *SubscriptionProperties
}

// CreateSubscription creates a subscription to a topic with configurable properties
func (ac *Client) CreateSubscription(ctx context.Context, topicName string, subscriptionName string, options *CreateSubscriptionOptions) (CreateSubscriptionResponse, error) {
	var properties *SubscriptionProperties

	if options != nil {
		properties = options.Properties
	}

	newProps, _, err := ac.createOrUpdateSubscriptionImpl(ctx, topicName, subscriptionName, properties, true)

	if err != nil {
		return CreateSubscriptionResponse{}, err
	}

	return CreateSubscriptionResponse{
		SubscriptionName:       subscriptionName,
		TopicName:              topicName,
		SubscriptionProperties: *newProps,
	}, nil
}

// GetSubscriptionResponse contains response fields for Client.GetSubscription
type GetSubscriptionResponse struct {
	// SubscriptionName is the name of the subscription.
	SubscriptionName string

	// TopicName is the name of the topic for this subscription.
	TopicName string

	SubscriptionProperties
}

// GetSubscriptionOptions contains optional parameters for Client.GetSubscription
type GetSubscriptionOptions struct {
	// For future expansion
}

// GetSubscription gets a subscription by name.
// If the entity does not exist this function will return a nil GetSubscriptionResponse and a nil error.
func (ac *Client) GetSubscription(ctx context.Context, topicName string, subscriptionName string, options *GetSubscriptionOptions) (*GetSubscriptionResponse, error) {
	var atomResp *atom.SubscriptionEnvelope
	_, err := ac.em.Get(ctx, fmt.Sprintf("/%s/Subscriptions/%s", topicName, subscriptionName), &atomResp)

	if err != nil {
		return mapATOMError[GetSubscriptionResponse](err)
	}

	item, err := newSubscriptionItem(atomResp, topicName)

	if err != nil {
		return nil, err
	}

	return &GetSubscriptionResponse{
		SubscriptionName:       subscriptionName,
		TopicName:              topicName,
		SubscriptionProperties: item.SubscriptionProperties,
	}, nil
}

// GetSubscriptionRuntimePropertiesResponse contains response fields for Client.GetSubscriptionRuntimeProperties
type GetSubscriptionRuntimePropertiesResponse struct {
	// TopicName is the name of the topic.
	TopicName string

	// SubscriptionName is the name of the subscription.
	SubscriptionName string

	SubscriptionRuntimeProperties
}

// GetSubscriptionRuntimePropertiesOptions contains optional parameters for Client.GetSubscriptionRuntimeProperties
type GetSubscriptionRuntimePropertiesOptions struct {
	// For future expansion
}

// GetSubscriptionRuntimeProperties gets runtime properties of a subscription, like the SizeInBytes, or SubscriptionCount.
// If the entity does not exist this function will return a nil GetSubscriptionRuntimePropertiesResponse and a nil error.
func (ac *Client) GetSubscriptionRuntimeProperties(ctx context.Context, topicName string, subscriptionName string, options *GetSubscriptionRuntimePropertiesOptions) (*GetSubscriptionRuntimePropertiesResponse, error) {
	var atomResp *atom.SubscriptionEnvelope
	_, err := ac.em.Get(ctx, fmt.Sprintf("/%s/Subscriptions/%s", topicName, subscriptionName), &atomResp)

	if err != nil {
		return mapATOMError[GetSubscriptionRuntimePropertiesResponse](err)
	}

	item, err := newSubscriptionRuntimePropertiesItem(atomResp, topicName)

	if err != nil {
		return nil, err
	}

	return &GetSubscriptionRuntimePropertiesResponse{
		TopicName:                     topicName,
		SubscriptionName:              subscriptionName,
		SubscriptionRuntimeProperties: item.SubscriptionRuntimeProperties,
	}, nil
}

// ListSubscriptionsOptions can be used to configure the ListSusbscriptions method.
type ListSubscriptionsOptions struct {
	// MaxPageSize is the maximum size of each page of results.
	MaxPageSize int32
}

// SubscriptionPropertiesItem contains a single item for SubscriptionPager.PageResponse
type SubscriptionPropertiesItem struct {
	SubscriptionProperties

	// TopicName is the name of the topic.
	TopicName string

	// SubscriptionName is the name of the subscription.
	SubscriptionName string
}

// ListSubscriptionsResponse contains the response fields for SubscriptionPager.PageResponse
type ListSubscriptionsResponse struct {
	// Value is the result of the request.
	Subscriptions []SubscriptionPropertiesItem
}

// NewListSubscriptionsPager creates a pager than can list subscriptions for a topic.
func (ac *Client) NewListSubscriptionsPager(topicName string, options *ListSubscriptionsOptions) *runtime.Pager[ListSubscriptionsResponse] {
	var pageSize int32

	if options != nil {
		pageSize = options.MaxPageSize
	}

	ep := &entityPager[atom.SubscriptionFeed, atom.SubscriptionEnvelope, SubscriptionPropertiesItem]{
		convertFn: func(env *atom.SubscriptionEnvelope) (*SubscriptionPropertiesItem, error) {
			return newSubscriptionItem(env, topicName)
		},
		baseFragment: fmt.Sprintf("/%s/Subscriptions?", topicName),
		maxPageSize:  pageSize,
		em:           ac.em,
	}

	return runtime.NewPager(runtime.PagingHandler[ListSubscriptionsResponse]{
		More: func(ltr ListSubscriptionsResponse) bool {
			return ep.More()
		},
		Fetcher: func(ctx context.Context, t *ListSubscriptionsResponse) (ListSubscriptionsResponse, error) {
			items, err := ep.Fetcher(ctx)

			if err != nil {
				return ListSubscriptionsResponse{}, err
			}

			return ListSubscriptionsResponse{
				Subscriptions: items,
			}, nil
		},
	})
}

// ListSubscriptionsRuntimePropertiesOptions can be used to configure the ListSubscriptionsRuntimeProperties method.
type ListSubscriptionsRuntimePropertiesOptions struct {
	// MaxPageSize is the maximum size of each page of results.
	MaxPageSize int32
}

// SubscriptionRuntimePropertiesItem contains the data from a SubscriptionRuntimePropertiesPager.PageResponse method
type SubscriptionRuntimePropertiesItem struct {
	SubscriptionRuntimeProperties

	// TopicName is the name of the topic.
	TopicName string

	// SubscriptionName is the name of the subscription.
	SubscriptionName string
}

// ListSubscriptionsRuntimePropertiesResponse contains the response fields for SubscriptionRuntimePropertiesPager.PageResponse
type ListSubscriptionsRuntimePropertiesResponse struct {
	// Value is the result of the request.
	SubscriptionRuntimeProperties []SubscriptionRuntimePropertiesItem
}

// NewListSubscriptionsRuntimePropertiesPager creates a pager than can list runtime properties for subscriptions for a topic.
func (ac *Client) NewListSubscriptionsRuntimePropertiesPager(topicName string, options *ListSubscriptionsRuntimePropertiesOptions) *runtime.Pager[ListSubscriptionsRuntimePropertiesResponse] {
	var pageSize int32

	if options != nil {
		pageSize = options.MaxPageSize
	}

	ep := &entityPager[atom.SubscriptionFeed, atom.SubscriptionEnvelope, SubscriptionRuntimePropertiesItem]{
		convertFn: func(env *atom.SubscriptionEnvelope) (*SubscriptionRuntimePropertiesItem, error) {
			return newSubscriptionRuntimePropertiesItem(env, topicName)
		},
		baseFragment: fmt.Sprintf("/%s/Subscriptions?", topicName),
		maxPageSize:  pageSize,
		em:           ac.em,
	}

	return runtime.NewPager(runtime.PagingHandler[ListSubscriptionsRuntimePropertiesResponse]{
		More: func(ltr ListSubscriptionsRuntimePropertiesResponse) bool {
			return ep.More()
		},
		Fetcher: func(ctx context.Context, t *ListSubscriptionsRuntimePropertiesResponse) (ListSubscriptionsRuntimePropertiesResponse, error) {
			items, err := ep.Fetcher(ctx)

			if err != nil {
				return ListSubscriptionsRuntimePropertiesResponse{}, err
			}

			return ListSubscriptionsRuntimePropertiesResponse{
				SubscriptionRuntimeProperties: items,
			}, nil
		},
	})
}

// UpdateSubscriptionResponse contains the response fields for Client.UpdateSubscription
type UpdateSubscriptionResponse struct {
	// TopicName is the name of the topic.
	TopicName string

	// SubscriptionName is the name of the subscription.
	SubscriptionName string

	SubscriptionProperties
}

// UpdateSubscriptionOptions contains the optional parameters for Client.UpdateSubscription
type UpdateSubscriptionOptions struct {
	// For future expansion
}

// UpdateSubscription updates an existing subscription.
func (ac *Client) UpdateSubscription(ctx context.Context, topicName string, subscriptionName string, properties SubscriptionProperties, options *UpdateSubscriptionOptions) (UpdateSubscriptionResponse, error) {
	newProps, _, err := ac.createOrUpdateSubscriptionImpl(ctx, topicName, subscriptionName, &properties, false)

	if err != nil {
		return UpdateSubscriptionResponse{}, err
	}

	return UpdateSubscriptionResponse{
		TopicName:              topicName,
		SubscriptionName:       subscriptionName,
		SubscriptionProperties: *newProps,
	}, nil
}

// DeleteSubscriptionOptions contains optional parameters for Client.DeleteSubscription
type DeleteSubscriptionOptions struct {
	// For future expansion
}

// DeleteSubscriptionResponse contains response fields for Client.DeleteSubscription
type DeleteSubscriptionResponse struct {
}

// DeleteSubscription deletes a subscription.
func (ac *Client) DeleteSubscription(ctx context.Context, topicName string, subscriptionName string, options *DeleteSubscriptionOptions) (DeleteSubscriptionResponse, error) {
	resp, err := ac.em.Delete(ctx, fmt.Sprintf("/%s/Subscriptions/%s", topicName, subscriptionName))
	defer atom.CloseRes(ctx, resp)
	return DeleteSubscriptionResponse{}, err
}

func (ac *Client) createOrUpdateSubscriptionImpl(ctx context.Context, topicName string, subscriptionName string, props *SubscriptionProperties, creating bool) (*SubscriptionProperties, *http.Response, error) {
	if props == nil {
		props = &SubscriptionProperties{}
	}

	env, err := newSubscriptionEnvelope(props, ac.em.TokenProvider())

	if err != nil {
		return nil, nil, err
	}

	if !creating {
		ctx = runtime.WithHTTPHeader(ctx, http.Header{
			"If-Match": []string{"*"},
		})
	}

	executeOpts := &atom.ExecuteOptions{
		ForwardTo:           props.ForwardTo,
		ForwardToDeadLetter: props.ForwardDeadLetteredMessagesTo,
	}

	var atomResp *atom.SubscriptionEnvelope
	resp, err := ac.em.Put(ctx, fmt.Sprintf("/%s/Subscriptions/%s", topicName, subscriptionName), env, &atomResp, executeOpts)

	if err != nil {
		return nil, nil, err
	}

	item, err := newSubscriptionItem(atomResp, topicName)

	if err != nil {
		return nil, nil, err
	}

	return &item.SubscriptionProperties, resp, nil
}

func newSubscriptionEnvelope(props *SubscriptionProperties, tokenProvider auth.TokenProvider) (*atom.SubscriptionEnvelope, error) {
	defaultRuleDescription, err := newDefaultRuleDescription(props.DefaultRule)

	if err != nil {
		return nil, err
	}

	desc := &atom.SubscriptionDescription{
		DefaultMessageTimeToLive:                  props.DefaultMessageTimeToLive,
		LockDuration:                              props.LockDuration,
		RequiresSession:                           props.RequiresSession,
		DeadLetteringOnMessageExpiration:          props.DeadLetteringOnMessageExpiration,
		DeadLetteringOnFilterEvaluationExceptions: props.EnableDeadLetteringOnFilterEvaluationExceptions,
		MaxDeliveryCount:                          props.MaxDeliveryCount,
		ForwardTo:                                 props.ForwardTo,
		ForwardDeadLetteredMessagesTo:             props.ForwardDeadLetteredMessagesTo,
		UserMetadata:                              props.UserMetadata,
		EnableBatchedOperations:                   props.EnableBatchedOperations,
		AutoDeleteOnIdle:                          props.AutoDeleteOnIdle,
		DefaultRuleDescription:                    defaultRuleDescription,
	}

	return atom.WrapWithSubscriptionEnvelope(desc), nil
}

func newDefaultRuleDescription(properties *RuleProperties) (*atom.DefaultRuleDescription, error) {
	if properties == nil {
		return nil, nil
	}

	ruleDescription := atom.DefaultRuleDescription{
		Name: makeRuleNameForProperties(properties),
	}

	filter, err := convertRuleFilterToFilterDescription(&properties.Filter)

	if err != nil {
		return nil, err
	}

	// Filter can never be nil because it's default is TrueFilter
	ruleDescription.Filter = filter

	action, err := convertRuleActionToActionDescription(&properties.Action)

	if err != nil {
		return nil, err
	}

	ruleDescription.Action = action

	return &ruleDescription, nil
}

func newSubscriptionItem(env *atom.SubscriptionEnvelope, topicName string) (*SubscriptionPropertiesItem, error) {
	desc := env.Content.SubscriptionDescription

	props := SubscriptionProperties{
		RequiresSession:                                 desc.RequiresSession,
		DeadLetteringOnMessageExpiration:                desc.DeadLetteringOnMessageExpiration,
		EnableDeadLetteringOnFilterEvaluationExceptions: desc.DeadLetteringOnFilterEvaluationExceptions,
		MaxDeliveryCount:                                desc.MaxDeliveryCount,
		ForwardTo:                                       desc.ForwardTo,
		ForwardDeadLetteredMessagesTo:                   desc.ForwardDeadLetteredMessagesTo,
		UserMetadata:                                    desc.UserMetadata,
		LockDuration:                                    desc.LockDuration,
		DefaultMessageTimeToLive:                        desc.DefaultMessageTimeToLive,
		EnableBatchedOperations:                         desc.EnableBatchedOperations,
		Status:                                          (*EntityStatus)(desc.Status),
		AutoDeleteOnIdle:                                desc.AutoDeleteOnIdle,
	}

	return &SubscriptionPropertiesItem{
		TopicName:              topicName,
		SubscriptionName:       env.Title,
		SubscriptionProperties: props,
	}, nil
}

func newSubscriptionRuntimePropertiesItem(env *atom.SubscriptionEnvelope, topicName string) (*SubscriptionRuntimePropertiesItem, error) {
	desc := env.Content.SubscriptionDescription

	if desc.CountDetails == nil {
		return nil, errors.New("invalid subscription runtime properties: no CountDetails element")
	}

	rtp := SubscriptionRuntimeProperties{
		TotalMessageCount:              *desc.MessageCount,
		ActiveMessageCount:             *desc.CountDetails.ActiveMessageCount,
		DeadLetterMessageCount:         *desc.CountDetails.DeadLetterMessageCount,
		TransferMessageCount:           *desc.CountDetails.TransferMessageCount,
		TransferDeadLetterMessageCount: *desc.CountDetails.TransferDeadLetterMessageCount,
	}

	var err error

	if rtp.CreatedAt, err = atom.StringToTime(desc.CreatedAt); err != nil {
		return nil, err
	}

	if rtp.UpdatedAt, err = atom.StringToTime(desc.UpdatedAt); err != nil {
		return nil, err
	}

	if rtp.AccessedAt, err = atom.StringToTime(desc.AccessedAt); err != nil {
		return nil, err
	}

	return &SubscriptionRuntimePropertiesItem{
		SubscriptionRuntimeProperties: rtp,
		TopicName:                     topicName,
		SubscriptionName:              env.Title,
	}, nil
}
