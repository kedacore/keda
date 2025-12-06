# Release History

## 2.0.1 (2025-10-08)

### Bugs Fixed

- Fixed outdated documentation that incorrectly stated the library is in beta.

## 2.0.0 (2025-06-10)

First release of `github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2`. 

### Breaking Changes

This new major release is compatible with azeventhubs v1, with one difference - Checkpoint.Offset and ReceivedEventData.Offset's type have been changed to a string (from an integer). 
This change does NOT affect any stored checkpoints. Most customers will be unaffected by this change and can safely upgrade.

### Features Added

- Support for Event Hubs Geo-Replication (PR#24477)

## 2.0.0-beta.1 (2025-05-06)

### Features Added

- Support for Event Hubs Geo-Replication (PR#24477)

### Breaking Changes

- This package is compatible with azeventhubs v1, with one difference - Checkpoint.Offset and ReceivedEventData.Offset's type have been changed to a string (from an integer). 
  This change does NOT affect any stored checkpoints. Most customers will be unaffected by this change and can safely upgrade.

## 1.3.2 (2025-04-08)

### Bugs Fixed

- Processor now only lists checkpoints when it needs to start a new partition client, avoiding wasted calls to the checkpoint store. (PR#24383)

## 1.3.1 (2025-03-11)

### Bugs Fixed

- Removed a memory leak that could occur when the ConsumerClient was unable to open a partition. (PR#24198)

## 1.3.0 (2025-02-11)

### Features Added

- ProducerClient and ConsumerClient allow the endpoint to be overridden with CustomEndpoint, allowing the use of TCP proxies with AMQP.

## 1.3.0-beta.1 (2025-01-13)

### Features Added

- ProducerClient and ConsumerClient allow the endpoint to be overridden with CustomEndpoint, allowing the use of TCP proxies with AMQP.

## 1.2.3 (2024-10-14)

### Bugs Fixed

- Fixed bug where cancelling management link calls, such GetEventHubProperties() or GetPartitionProperties, could result in blocked calls. (PR#23400)
- Apply fix from @bcho for overflows with retries. (PR#23562)

## 1.2.2 (2024-08-15)

### Bugs Fixed

- Fixed a bug that where a short context deadline could prevent recovery from ever happening. The end result would be a broken PartitionClient/ConsumerClient that would never recover from the underlying failure. (PR#23337)

## 1.2.1 (2024-05-20)

### Bugs Fixed

- Emulator strings should allow for hosts other than localhost (PR#22898)

## 1.2.0 (2024-05-07)

### Bugs Fixed

Processor.Run had unclear behavior for some cases:

- Run() now returns an explicit error when called more than once on a single
  Processor instance or if multiple Run calls are made concurrently. (PR#22833)
- NextProcessorClient now properly terminates (and returns nil) if called on a
  stopped Processor. (PR#22833)

## 1.1.0 (2024-04-02)

### Features Added

- Add in ability to handle emulator connection strings. (PR#22663)

### Bugs Fixed

- Fixed a race condition between Processor.Run() and Processor.NextPartitionClient() where cancelling Run() quickly could lead to NextPartitionClient hanging indefinitely. (PR#22541)

## 1.0.4 (2024-03-05)

### Bugs Fixed

- Fixed case where closing a Receiver/Sender after an idle period would take > 20 seconds. (PR#22509)

## 1.0.3 (2024-01-16)

### Bugs Fixed

- Processor distributes partitions optimally, which would result in idle or over-assigned processors. (PR#22153)

## 1.0.2 (2023-11-07)

### Bugs Fixed

- Processor now relinquishes ownership of partitions when it shuts down, making them immediately available to other active Processor instances. (PR#21899)

## 1.0.1 (2023-06-06)

### Bugs Fixed

- GetPartitionProperties and GetEventHubProperties now retry properly on failures. (PR#20893)
- Connection recovery could artifically fail, prolonging recovery. (PR#20883)

## 1.0.0 (2023-05-09)

### Features Added

- First stable release of the azeventhubs package.
- Authentication errors are indicated with an `azeventhubs.Error`, with a `Code` of `azeventhubs.ErrorCodeUnauthorizedAccess`. (PR#20450)

### Bugs Fixed

- Authentication errors could cause unnecessary retries, making calls taking longer to fail. (PR#20450)
- Recovery now includes internal timeouts and also handles restarting a connection if AMQP primitives aren't closed cleanly.
- Potential leaks for $cbs and $management when there was a partial failure. (PR#20564)
- Latest go-amqp changes have been merged in with fixes for robustness.
- Sending a message to an entity that is full will no longer retry. (PR#20722)
- Checkpoint store handles multiple initial owners properly, allowing only one through. (PR#20727)

## 0.6.0 (2023-03-07)

### Features Added

- Added the `ConsumerClientOptions.InstanceID` field. This optional field can enhance error messages from
  Event Hubs. For example, error messages related to ownership changes for a partition will contain the
  name of the link that has taken ownership, which can help with traceability.

### Breaking Changes

- `ConsumerClient.ID()` renamed to `ConsumerClient.InstanceID()`.

### Bugs Fixed

- Recover the connection when the $cbs Receiver/Sender is not closed properly. This would cause
  clients to return an error saying "$cbs node has already been opened." (PR#20334)

## 0.5.0 (2023-02-07)

### Features Added

- Adds ProcessorOptions.Prefetch field, allowing configuration of Prefetch values for PartitionClients created using the Processor. (PR#19786)
- Added new function to parse connection string into values using `ParseConnectionString` and `ConnectionStringProperties`. (PR#19855)

### Breaking Changes

- ProcessorOptions.OwnerLevel has been removed. The Processor uses 0 as the owner level.
- Uses the public release of `github.com/Azure/azure-sdk-for-go/sdk/storage/azblob` package rather than using an internal copy.
  For an example, see [example_consuming_with_checkpoints_test.go](https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs/example_consuming_with_checkpoints_test.go).

## 0.4.0 (2023-01-10)

### Bugs Fixed

- User-Agent was incorrectly formatted in our AMQP-based clients. (PR#19712)
- Connection recovery has been improved, removing some unnecessasry retries as well as adding a bound around
  some operations (Close) that could potentially block recovery for a long time. (PR#19683)

## 0.3.0 (2022-11-10)

### Bugs Fixed

- $cbs link is properly closed, even on cancellation (#19492)

### Breaking Changes

- ProducerClient.SendEventBatch renamed to ProducerClient.SendEventDataBatch, to align with
  the name of the type.

## 0.2.0 (2022-10-17)

### Features Added

- Raw AMQP message support, including full support for encoding Body (Value, Sequence and also multiple byte slices for Data). See ExampleEventDataBatch_AddEventData_rawAMQPMessages for some concrete examples. (PR#19156)
- Prefetch is now enabled by default. Prefetch allows the Event Hubs client to maintain a continuously full cache of events, controlled by PartitionClientOptions.Prefetch. (PR#19281)
- ConsumerClient.ID() returns a unique ID representing each instance of ConsumerClient.

### Breaking Changes

- EventDataBatch.NumMessages() renamed to EventDataBatch.NumEvents()
- Prefetch is now enabled by default. To disable it set PartitionClientOptions.Prefetch to -1.
- NewWebSocketConnArgs renamed to WebSocketConnParams
- Code renamed to ErrorCode, including associated constants like `ErrorCodeOwnershipLost`.
- OwnershipData, CheckpointData, and CheckpointStoreAddress have been folded into their individual structs: Ownership and Checkpoint.
- StartPosition and OwnerLevel were erroneously included in the ConsumerClientOptions struct - they've been removed. These can be
  configured in the PartitionClientOptions.

### Bugs Fixed

- Retries now respect cancellation when they're in the "delay before next try" phase. (PR#19295)
- Fixed a potential leak which could cause us to open and leak a $cbs link connection, resulting in errors. (PR#19326)

## 0.1.1 (2022-09-08)

### Features Added

- Adding in the new Processor type, which can be used to do distributed (and load balanced) consumption of events, using a
  CheckpointStore. The built-in checkpoints.BlobStore uses Azure Blob Storage for persistence. A full example is
  in [example_consuming_with_checkpoints_test.go](https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs/example_consuming_with_checkpoints_test.go).

### Breaking Changes

- In the first beta, ConsumerClient took constructor parameter that required a partition ID, which meant you had to create
  multiple ConsumerClients if you wanted to consume multiple partitions. ConsumerClient can now create multiple PartitionClient
  instances (using ConsumerClient.NewPartitionClient), which allows you to share the same AMQP connection and receive from multiple
  partitions simultaneously.
- Changes to EventData/ReceivedEventData:

  - ReceivedEventData now embeds EventData for fields common between the two, making it easier to change and resend.
  - `ApplicationProperties` renamed to `Properties`.
  - `PartitionKey` removed from `EventData`. To send events using a PartitionKey you must set it in the options
    when creating the EventDataBatch:

    ```go
    batch, err := producerClient.NewEventDataBatch(context.TODO(), &azeventhubs.NewEventDataBatchOptions{
      PartitionKey: to.Ptr("partition key"),
    })
    ```

### Bugs Fixed

- ReceivedEventData.Offset was incorrectly parsed, resulting in it always being 0.
- Added missing fields to ReceivedEventData and EventData (CorrelationID)
- PartitionKey property was not populated for messages sent via batch.

## 0.1.0 (2022-08-11)

- Initial preview for the new version of the Azure Event Hubs Go SDK.
