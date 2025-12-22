// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azeventhubs

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/internal/uuid"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/amqpwrap"
)

// ConsumerClientOptions configures optional parameters for a ConsumerClient.
type ConsumerClientOptions struct {
	// ApplicationID is used as the identifier when setting the User-Agent property.
	ApplicationID string

	// A custom endpoint address that can be used when establishing the connection to the service.
	CustomEndpoint string

	// InstanceID is a unique name used to identify the consumer. This can help with
	// diagnostics as this name will be returned in error messages. By default,
	// an identifier will be automatically generated.
	InstanceID string

	// NewWebSocketConn is a function that can create a net.Conn for use with websockets.
	// For an example, see ExampleNewClient_usingWebsockets() function in example_client_test.go.
	NewWebSocketConn func(ctx context.Context, args WebSocketConnParams) (net.Conn, error)

	// RetryOptions controls how often operations are retried from this client and any
	// Receivers and Senders created from this client.
	RetryOptions RetryOptions

	// TLSConfig configures a client with a custom *tls.Config.
	TLSConfig *tls.Config
}

// ConsumerClient can create PartitionClient instances, which can read events from
// a partition.
type ConsumerClient struct {
	consumerGroup string
	eventHub      string

	// instanceID is a customer supplied instanceID that can be passed to Event Hubs.
	// It'll be returned in error messages and can be useful for customers when
	// troubleshooting.
	instanceID string

	links        *internal.Links[amqpwrap.RPCLink]
	namespace    *internal.Namespace
	retryOptions RetryOptions
}

// NewConsumerClient creates a ConsumerClient which uses an azcore.TokenCredential for authentication. You
// MUST call [ConsumerClient.Close] on this client to avoid leaking resources.
//
// The fullyQualifiedNamespace is the Event Hubs namespace name (ex: myeventhub.servicebus.windows.net)
// The credential is one of the credentials in the [azidentity] package.
//
// [azidentity]: https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/azidentity
func NewConsumerClient(fullyQualifiedNamespace string, eventHub string, consumerGroup string, credential azcore.TokenCredential, options *ConsumerClientOptions) (*ConsumerClient, error) {
	return newConsumerClient(consumerClientArgs{
		consumerGroup:           consumerGroup,
		fullyQualifiedNamespace: fullyQualifiedNamespace,
		eventHub:                eventHub,
		credential:              credential,
	}, options)
}

// NewConsumerClientFromConnectionString creates a ConsumerClient from a connection string. You
// MUST call [ConsumerClient.Close] on this client to avoid leaking resources.
//
// connectionString can be one of two formats - with or without an EntityPath key.
//
// When the connection string does not have an entity path, as shown below, the eventHub parameter cannot
// be empty and should contain the name of your event hub.
//
//	Endpoint=sb://<your-namespace>.servicebus.windows.net/;SharedAccessKeyName=<key-name>;SharedAccessKey=<key>
//
// When the connection string DOES have an entity path, as shown below, the eventHub parameter must be empty.
//
//	Endpoint=sb://<your-namespace>.servicebus.windows.net/;SharedAccessKeyName=<key-name>;SharedAccessKey=<key>;EntityPath=<entity path>;
func NewConsumerClientFromConnectionString(connectionString string, eventHub string, consumerGroup string, options *ConsumerClientOptions) (*ConsumerClient, error) {
	props, err := parseConn(connectionString, eventHub)

	if err != nil {
		return nil, err
	}

	return newConsumerClient(consumerClientArgs{
		consumerGroup:    consumerGroup,
		connectionString: connectionString,
		eventHub:         *props.EntityPath,
	}, options)
}

// PartitionClientOptions provides options for the NewPartitionClient function.
type PartitionClientOptions struct {
	// StartPosition is the position we will start receiving events from,
	// either an offset (inclusive) with Offset, or receiving events received
	// after a specific time using EnqueuedTime.
	//
	// NOTE: you can also use the [Processor], which will automatically manage the start
	// value using a [CheckpointStore]. See [example_consuming_with_checkpoints_test.go] for an
	// example.
	//
	// [example_consuming_with_checkpoints_test.go]: https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs/example_consuming_with_checkpoints_test.go
	StartPosition StartPosition

	// OwnerLevel is the priority for this partition client, also known as the 'epoch' level.
	// When used, a partition client with a higher OwnerLevel will take ownership of a partition
	// from partition clients with a lower OwnerLevel.
	// Default is off.
	OwnerLevel *int64

	// Prefetch represents the size of the internal prefetch buffer. When set,
	// this client will attempt to always maintain an internal cache of events of
	// this size, asynchronously, increasing the odds that ReceiveEvents() will use
	// a locally stored cache of events, rather than having to wait for events to
	// arrive from the network.
	//
	// Defaults to 300 events if Prefetch == 0.
	// Disabled if Prefetch < 0.
	Prefetch int32
}

// NewPartitionClient creates a client that can receive events from a partition. By default it starts
// at the latest point in the partition. This can be changed using the options parameter.
// You MUST call [azeventhubs.PartitionClient.Close] on the returned client to avoid leaking resources.
func (cc *ConsumerClient) NewPartitionClient(partitionID string, options *PartitionClientOptions) (*PartitionClient, error) {
	return newPartitionClient(partitionClientArgs{
		namespace:     cc.namespace,
		eventHub:      cc.eventHub,
		partitionID:   partitionID,
		instanceID:    cc.instanceID,
		consumerGroup: cc.consumerGroup,
		retryOptions:  cc.retryOptions,
	}, options)
}

// GetEventHubProperties gets event hub properties, like the available partition IDs and when the Event Hub was created.
func (cc *ConsumerClient) GetEventHubProperties(ctx context.Context, options *GetEventHubPropertiesOptions) (EventHubProperties, error) {
	return getEventHubProperties(ctx, EventConsumer, cc.namespace, cc.links, cc.eventHub, cc.retryOptions, options)
}

// GetPartitionProperties gets properties for a specific partition. This includes data like the
// last enqueued sequence number, the first sequence number and when an event was last enqueued
// to the partition.
func (cc *ConsumerClient) GetPartitionProperties(ctx context.Context, partitionID string, options *GetPartitionPropertiesOptions) (PartitionProperties, error) {
	return getPartitionProperties(ctx, EventConsumer, cc.namespace, cc.links, cc.eventHub, partitionID, cc.retryOptions, options)
}

// InstanceID is the identifier for this ConsumerClient.
func (cc *ConsumerClient) InstanceID() string {
	return cc.instanceID
}

type consumerClientDetails struct {
	FullyQualifiedNamespace string
	ConsumerGroup           string
	EventHubName            string
	ClientID                string
}

func (cc *ConsumerClient) getDetails() consumerClientDetails {
	return consumerClientDetails{
		FullyQualifiedNamespace: cc.namespace.FQDN,
		ConsumerGroup:           cc.consumerGroup,
		EventHubName:            cc.eventHub,
		ClientID:                cc.InstanceID(),
	}
}

// Close releases resources for this client.
func (cc *ConsumerClient) Close(ctx context.Context) error {
	return cc.namespace.Close(ctx, true)
}

type consumerClientArgs struct {
	connectionString string

	// the Event Hubs namespace name (ex: myservicebus.servicebus.windows.net)
	fullyQualifiedNamespace string
	credential              azcore.TokenCredential

	consumerGroup string
	eventHub      string
}

func newConsumerClient(args consumerClientArgs, options *ConsumerClientOptions) (*ConsumerClient, error) {
	if options == nil {
		options = &ConsumerClientOptions{}
	}

	instanceID, err := getInstanceID(options.InstanceID)

	if err != nil {
		return nil, err
	}

	client := &ConsumerClient{
		consumerGroup: args.consumerGroup,
		eventHub:      args.eventHub,
		instanceID:    instanceID,
	}

	var nsOptions []internal.NamespaceOption

	if args.connectionString != "" {
		nsOptions = append(nsOptions, internal.NamespaceWithConnectionString(args.connectionString))
	} else if args.credential != nil {
		option := internal.NamespaceWithTokenCredential(
			args.fullyQualifiedNamespace,
			args.credential)

		nsOptions = append(nsOptions, option)
	}

	client.retryOptions = options.RetryOptions

	if options.TLSConfig != nil {
		nsOptions = append(nsOptions, internal.NamespaceWithTLSConfig(options.TLSConfig))
	}

	if options.NewWebSocketConn != nil {
		nsOptions = append(nsOptions, internal.NamespaceWithWebSocket(options.NewWebSocketConn))
	}

	if options.ApplicationID != "" {
		nsOptions = append(nsOptions, internal.NamespaceWithUserAgent(options.ApplicationID))
	}

	if options.CustomEndpoint != "" {
		nsOptions = append(nsOptions, internal.NamespaceWithCustomEndpoint(options.CustomEndpoint))
	}

	nsOptions = append(nsOptions, internal.NamespaceWithRetryOptions(options.RetryOptions))

	tempNS, err := internal.NewNamespace(nsOptions...)

	if err != nil {
		return nil, err
	}

	client.namespace = tempNS
	client.links = internal.NewLinks[amqpwrap.RPCLink](tempNS, fmt.Sprintf("%s/$management", client.eventHub), nil, nil)

	return client, nil
}

func getInstanceID(optionalID string) (string, error) {
	if optionalID != "" {
		return optionalID, nil
	}

	// generate a new one
	id, err := uuid.New()

	if err != nil {
		return "", err
	}

	return id.String(), nil
}
