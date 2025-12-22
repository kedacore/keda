// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azeventhubs

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	azlog "github.com/Azure/azure-sdk-for-go/sdk/internal/log"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/amqpwrap"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/exported"
	"github.com/Azure/go-amqp"
)

// WebSocketConnParams are passed to your web socket creation function (ClientOptions.NewWebSocketConn)
type WebSocketConnParams = exported.WebSocketConnParams

// RetryOptions represent the options for retries.
type RetryOptions = exported.RetryOptions

// ProducerClientOptions contains options for the `NewProducerClient` and `NewProducerClientFromConnectionString`
// functions.
type ProducerClientOptions struct {
	// Application ID that will be passed to the namespace.
	ApplicationID string

	// A custom endpoint address that can be used when establishing the connection to the service.
	CustomEndpoint string

	// NewWebSocketConn is a function that can create a net.Conn for use with websockets.
	// For an example, see ExampleNewClient_usingWebsockets() function in example_client_test.go.
	NewWebSocketConn func(ctx context.Context, params WebSocketConnParams) (net.Conn, error)

	// RetryOptions controls how often operations are retried from this client and any
	// Receivers and Senders created from this client.
	RetryOptions RetryOptions

	// TLSConfig configures a client with a custom *tls.Config.
	TLSConfig *tls.Config
}

// ProducerClient can be used to send events to an Event Hub.
type ProducerClient struct {
	eventHub     string
	links        *internal.Links[amqpwrap.AMQPSenderCloser]
	namespace    internal.NamespaceForProducerOrConsumer
	retryOptions RetryOptions
}

// anyPartitionID is what we target if we want to send a message and let Event Hubs pick a partition
// or if we're doing an operation that isn't partition specific, such as querying the management link
// to get event hub properties or partition properties.
const anyPartitionID = ""

// NewProducerClient creates a ProducerClient which uses an azcore.TokenCredential for authentication. You
// MUST call [ProducerClient.Close] on this client to avoid leaking resources.
//
// The fullyQualifiedNamespace is the Event Hubs namespace name (ex: myeventhub.servicebus.windows.net)
// The credential is one of the credentials in the [azidentity] package.
//
// [azidentity]: https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/azidentity
func NewProducerClient(fullyQualifiedNamespace string, eventHub string, credential azcore.TokenCredential, options *ProducerClientOptions) (*ProducerClient, error) {
	return newProducerClientImpl(producerClientCreds{
		fullyQualifiedNamespace: fullyQualifiedNamespace,
		credential:              credential,
		eventHub:                eventHub,
	}, options)
}

// NewProducerClientFromConnectionString creates a ProducerClient from a connection string. You
// MUST call [ProducerClient.Close] on this client to avoid leaking resources.
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
func NewProducerClientFromConnectionString(connectionString string, eventHub string, options *ProducerClientOptions) (*ProducerClient, error) {
	props, err := parseConn(connectionString, eventHub)

	if err != nil {
		return nil, err
	}

	return newProducerClientImpl(producerClientCreds{
		connectionString: connectionString,
		eventHub:         *props.EntityPath,
	}, options)
}

// EventDataBatchOptions contains optional parameters for the [ProducerClient.NewEventDataBatch] function.
//
// If both PartitionKey and PartitionID are nil, Event Hubs will choose an arbitrary partition
// for any events in this [EventDataBatch].
type EventDataBatchOptions struct {
	// MaxBytes overrides the max size (in bytes) for a batch.
	// By default NewEventDataBatch will use the max message size provided by the service.
	MaxBytes uint64

	// PartitionKey is hashed to calculate the partition assignment. Messages and message
	// batches with the same PartitionKey are guaranteed to end up in the same partition.
	// Note that if you use this option then PartitionID cannot be set.
	PartitionKey *string

	// PartitionID is the ID of the partition to send these messages to.
	// Note that if you use this option then PartitionKey cannot be set.
	PartitionID *string
}

// NewEventDataBatch can be used to create an EventDataBatch, which can contain multiple
// events.
//
// EventDataBatch contains logic to make sure that the it doesn't exceed the maximum size
// for the Event Hubs link, using it's [azeventhubs.EventDataBatch.AddEventData] function.
// A lower size limit can also be configured through the options.
//
// NOTE: if options is nil or empty, Event Hubs will choose an arbitrary partition for any
// events in this [EventDataBatch].
//
// If the operation fails it can return an azeventhubs.Error type if the failure is actionable.
func (pc *ProducerClient) NewEventDataBatch(ctx context.Context, options *EventDataBatchOptions) (*EventDataBatch, error) {
	var batch *EventDataBatch

	partitionID := anyPartitionID

	if options != nil && options.PartitionID != nil {
		partitionID = *options.PartitionID
	}

	err := pc.links.Retry(ctx, exported.EventProducer, "NewEventDataBatch", partitionID, pc.retryOptions, func(ctx context.Context, lwid internal.LinkWithID[amqpwrap.AMQPSenderCloser]) error {
		tmpBatch, err := newEventDataBatch(lwid.Link(), options)

		if err != nil {
			return err
		}

		batch = tmpBatch
		return nil
	})

	if err != nil {
		return nil, internal.TransformError(err)
	}

	return batch, nil
}

// SendEventDataBatchOptions contains optional parameters for the SendEventDataBatch function
type SendEventDataBatchOptions struct {
	// For future expansion
}

// SendEventDataBatch sends an event data batch to Event Hubs.
func (pc *ProducerClient) SendEventDataBatch(ctx context.Context, batch *EventDataBatch, options *SendEventDataBatchOptions) error {
	amqpMessage, err := batch.toAMQPMessage()

	if err != nil {
		return err
	}

	partID := getPartitionID(batch.partitionID)

	err = pc.links.Retry(ctx, exported.EventProducer, "SendEventDataBatch", partID, pc.retryOptions, func(ctx context.Context, lwid internal.LinkWithID[amqpwrap.AMQPSenderCloser]) error {
		azlog.Writef(EventProducer, "[%s] Sending message with ID %v to partition %q", lwid.String(), amqpMessage.Properties.MessageID, partID)
		return lwid.Link().Send(ctx, amqpMessage, nil)
	})
	return internal.TransformError(err)
}

// GetPartitionProperties gets properties for a specific partition. This includes data like the last enqueued sequence number, the first sequence
// number and when an event was last enqueued to the partition.
func (pc *ProducerClient) GetPartitionProperties(ctx context.Context, partitionID string, options *GetPartitionPropertiesOptions) (PartitionProperties, error) {
	return getPartitionProperties(ctx, EventProducer, pc.namespace, pc.links, pc.eventHub, partitionID, pc.retryOptions, options)
}

// GetEventHubProperties gets event hub properties, like the available partition IDs and when the Event Hub was created.
func (pc *ProducerClient) GetEventHubProperties(ctx context.Context, options *GetEventHubPropertiesOptions) (EventHubProperties, error) {
	return getEventHubProperties(ctx, EventProducer, pc.namespace, pc.links, pc.eventHub, pc.retryOptions, options)
}

// Close releases resources for this client.
func (pc *ProducerClient) Close(ctx context.Context) error {
	if err := pc.links.Close(ctx); err != nil {
		azlog.Writef(EventProducer, "Failed when closing links while shutting down producer client: %s", err.Error())
	}
	return pc.namespace.Close(ctx, true)
}

func (pc *ProducerClient) getEntityPath(partitionID string) string {
	if partitionID != anyPartitionID {
		return fmt.Sprintf("%s/Partitions/%s", pc.eventHub, partitionID)
	} else {
		// this is the "let Event Hubs" decide link - any sends that occur here will
		// end up getting distributed to different partitions on the service side, rather
		// then being specified in the client.
		return pc.eventHub
	}
}

func (pc *ProducerClient) newEventHubProducerLink(ctx context.Context, session amqpwrap.AMQPSession, entityPath string, partitionID string) (amqpwrap.AMQPSenderCloser, error) {
	sender, err := session.NewSender(ctx, entityPath, partitionID, &amqp.SenderOptions{
		SettlementMode:              to.Ptr(amqp.SenderSettleModeMixed),
		RequestedReceiverSettleMode: to.Ptr(amqp.ReceiverSettleModeFirst),
		DesiredCapabilities: []string{
			internal.CapabilityGeoDRReplication,
		},
	})

	if err != nil {
		return nil, err
	}

	return sender, nil
}

type producerClientCreds struct {
	connectionString string

	// the Event Hubs namespace name (ex: myservicebus.servicebus.windows.net)
	fullyQualifiedNamespace string
	credential              azcore.TokenCredential

	eventHub string
}

func newProducerClientImpl(creds producerClientCreds, options *ProducerClientOptions) (*ProducerClient, error) {
	client := &ProducerClient{
		eventHub: creds.eventHub,
	}

	var nsOptions []internal.NamespaceOption

	if creds.connectionString != "" {
		nsOptions = append(nsOptions, internal.NamespaceWithConnectionString(creds.connectionString))
	} else if creds.credential != nil {
		option := internal.NamespaceWithTokenCredential(
			creds.fullyQualifiedNamespace,
			creds.credential)

		nsOptions = append(nsOptions, option)
	}

	if options != nil {
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
	}

	tmpNS, err := internal.NewNamespace(nsOptions...)

	if err != nil {
		return nil, err
	}

	client.namespace = tmpNS

	client.links = internal.NewLinks(tmpNS, fmt.Sprintf("%s/$management", client.eventHub), client.getEntityPath, client.newEventHubProducerLink)

	return client, err
}

// parseConn parses the connection string and ensures that the returned [exported.ConnectionStringProperties]
// has an EntityPath set, either from the connection string or using the eventHub parameter.
//
// If the connection string has an EntityPath then eventHub must be empty.
// If the connection string does not have an entity path then the eventHub must contain a value.
func parseConn(connectionString string, eventHub string) (exported.ConnectionStringProperties, error) {
	props, err := exported.ParseConnectionString(connectionString)

	if err != nil {
		return exported.ConnectionStringProperties{}, err
	}

	if props.EntityPath == nil {
		if eventHub == "" {
			return exported.ConnectionStringProperties{}, errors.New("connection string does not contain an EntityPath. eventHub cannot be an empty string")
		}
		props.EntityPath = &eventHub
	} else if eventHub != "" {
		return exported.ConnectionStringProperties{}, errors.New("connection string contains an EntityPath. eventHub must be an empty string")
	}

	return props, nil
}

func getPartitionID(partitionID *string) string {
	if partitionID != nil {
		return *partitionID
	}

	return anyPartitionID
}
