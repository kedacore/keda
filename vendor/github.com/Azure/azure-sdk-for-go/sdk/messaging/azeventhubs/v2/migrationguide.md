# Guide to migrate from `azure-event-hubs-go` to `azeventhubs`

This guide is intended to assist in the migration from the `azure-event-hubs-go` package to the `github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2`.

Our goal with this newest package was to export components that can be easily integrated into multiple styles of application, while still mapping close to the underlying resources for AMQP. This includes making TCP connection sharing simple (a must when multiplexing across multiple partitions), making batching boundaries more explicit and also integrating with the `azidentity` package, opening up a large number of authentication methods.

These changes are described in more detail, below.

### TCP connection sharing

In AMQP there are is a concept of a connection and links. AMQP Connections are TCP connections. Links are a logical conduit within an AMQP connection and there are typically many of them but they use the same connection and do not require their own socket.

The prior version of this package did not allow you to share an AMQP connection when sending events, which meant sending to multiple partitions would require a TCP connection per partition. If your application used more than a few partitions this could use up a scarce resource.

In the newer version of the library each top-level client (ProducerClient or ConsumerClient) owns their own TCP connection. For instance, in ProducerClient, sending to separate partitions creates multiple links internally, but not multiple TCP connections. ConsumerClient works similarly - it has a single TCP connection and calling ConsumerClient.NewPartitionClient creates new links, but not new TCP connections.

If you want to split activity across multiple TCP connections you can still do so by creating multiple instances of ProducerClient or ConsumerClient.

Some examples:

```go
// consumerClient will own a TCP connection.
consumerClient, err := azeventhubs.NewConsumerClient(/* arguments elided for example */)      

// Close the TCP connection (and any child links)
defer consumerClient.Close(context.TODO())    

// this call will lazily create a set of AMQP links using the consumerClient's TCP connection.
partClient0, err := consumerClient.NewPartitionClient("0", nil)
defer partClient0.Close(context.TODO())     // will close the AMQP link, not the connection

// this call will also lazily create a set of AMQP links using the consumerClient's TCP connection.
partClient1, err := consumerClient.NewPartitionClient("1", nil)
defer partClient1.Close(context.TODO())     // will close the AMQP link, not the connection
```

```go
// will lazily create an AMQP connection
producerClient, err := azeventhubs.NewProducerClient(/* arguments elided for example */)

// close the TCP connection (and any child links created for sending events)
defer producerClient.Close(context.TODO())

// these calls will lazily create a set of AMQP links using the producerClient's TCP connection.
producerClient.SendEventDataBatch(context.TODO(), eventDataBatchForPartition0, nil)
producerClient.SendEventDataBatch(context.TODO(), eventDataBatchForPartition1, nil)
```

## Clients

The `Hub` type has been replaced by two types:

* Consuming events, using the `azeventhubs.ConsumerClient`: [docs](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2#ConsumerClient) | [example](https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs/example_consuming_events_test.go)
* Sending events, use the `azeventhubs.ProducerClient`: [docs](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2#ProducerClient) | [example](https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs/example_producing_events_test.go)

`EventProcessorHost` has been replaced by the `azeventhubs.Processor` type: [docs](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2#Processor) | [example](https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs/example_consuming_with_checkpoints_test.go)

## Authentication

The older Event Hubs package provided some authentication methods like hub.NewHubFromEnvironment. These have been replaced by by using Azure Identity credentials from [azidentity](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#section-readme). 

You can also still authenticate using connection strings.

* `azeventhubs.ConsumerClient`: [using azidentity](https://github.com/Azure/azure-sdk-for-go/blob/a46bd74e113d6a045541b82a0f3f6497011d8417/sdk/messaging/azeventhubs/example_consumerclient_test.go#L16) | [using a connection string](https://github.com/Azure/azure-sdk-for-go/blob/a46bd74e113d6a045541b82a0f3f6497011d8417/sdk/messaging/azeventhubs/example_consumerclient_test.go#L30)

* `azeventhubs.ProducerClient`: [using azidentity](https://github.com/Azure/azure-sdk-for-go/blob/a46bd74e113d6a045541b82a0f3f6497011d8417/sdk/messaging/azeventhubs/example_producerclient_test.go#L16) | [using a connection string](https://github.com/Azure/azure-sdk-for-go/blob/a46bd74e113d6a045541b82a0f3f6497011d8417/sdk/messaging/azeventhubs/example_producerclient_test.go#L30)

## EventBatchIterator

Sending events has changed to be more explicit about when batches are formed and sent.

The older module had a type (EventBatchIterator). This type has been removed and replaced
with explicit batching, using `azeventhubs.EventDataBatch`. See here for an example: [link](https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs/example_producing_events_test.go).

## Getting hub/partition information

In the older module functions to get the partition IDs, as well as runtime properties
like the last enqueued sequence number were on the `Hub` type. These are now on both
of the client types instead (`ProducerClient`, `ConsumerClient`).

```go
// old
hub.GetPartitionInformation(context.TODO(), "0")
hub.GetRuntimeInformation(context.TODO())
```

```go
// new

// equivalent to: hub.GetRuntimeInformation(context.TODO())
consumerClient.GetEventHubProperties(context.TODO(), nil)   

// equivalent to: hub.GetPartitionInformation
consumerClient.GetPartitionProperties(context.TODO(), "partition-id", nil)  

//
// or, using the ProducerClient
//

producerClient.GetEventHubProperties(context.TODO(), nil)
producerClient.GetPartitionProperties(context.TODO(), "partition-id", nil)
```

## Migrating from a previous checkpoint store

See here for an example: [link](https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs/example_checkpoint_migration_test.go)
