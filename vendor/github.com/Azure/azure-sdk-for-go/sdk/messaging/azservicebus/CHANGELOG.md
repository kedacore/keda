# Release History

## 1.1.3 (2022-11-16)

### Bugs Fixed

- Removing changes for client-side idle timer and closing without timeout. Combined these are 
  causing issues with links not properly recovering or closing. Investigating an alternative
  for a future release.

## 1.1.2 (2022-11-08)

### Features Added

- Added a client-side idle timer which will reset Receiver links, transparently, if the link is idle for 
  5 minutes.

### Bugs Fixed

- $cbs link is properly closed, even on cancellation (#19492)

## 1.1.1 (2022-10-11)

### Bugs Fixed

- AcceptNextSessionForQueue and AcceptNextSessionForSubscription now return an azservicebus.Error with 
  Code set to CodeTimeout when they fail due to no sessions being available. Examples for this have 
  been added for `AcceptNextSessionForQueue`. PR#19113.
- Retries now respect cancellation when they're in the "delay before next try" phase.

## 1.1.0 (2022-08-09)

### Features Added

- Full access to send and receive all AMQP message properties. (#18413) 
  - Send AMQP messages using the new `AMQPAnnotatedMessage` type and `Sender.SendAMQPAnnotatedMessage()`.
  - AMQP messages can be added to MessageBatch's as well using `MessageBatch.AddAMQPAnnotatedMessage()`.
  - AMQP messages can be scheduled using `Sender.ScheduleAMQPAnnotatedMessages`.
  - Access the full set of AMQP message properties when receiving using the `ReceivedMessage.RawAMQPMessage` property. 

### Bugs Fixed

- Changed receive messages algorithm to avoid messages being excessively locked in Service Bus without 
  being transferred to the client. (PR#18657)
- Updating go-amqp, which fixes several bugs related to incorrect message locking (PR#18599)
  - Requesting large quantities of messages in a single ReceiveMessages() call could result in messages 
    not being delivered, but still incrementing their delivery count and requiring the message lock 
    timeout to expire.
  - Link detach could result in messages being ignored, requiring the message lock timeout to expire.
- Subscription rules weren't deserializing properly when created from the portal (PR#18813)

## 1.0.2-beta.0 (2022-07-07)

### Features Added

- Full access to send and receive all AMQP message properties. (#18413) 
  - Send AMQP messages using the new `AMQPAnnotatedMessage` type and `Sender.SendAMQPAnnotatedMessage()`.
  - AMQP messages can be added to MessageBatch's as well using `MessageBatch.AddAMQPAnnotatedMessage()`.
  - AMQP messages can be scheduled using `Sender.ScheduleAMQPAnnotatedMessages`.
  - Access the full set of AMQP message properties when receiving using the `ReceivedMessage.RawAMQPMessage` property. 

### Bugs Fixed

- Settlement of a message could hang if the link had been detached/closed. (#18532)

## 1.0.1 (2022-06-07)

### Features Added

- Adding in (QueueProperties|TopicProperties).MaxMessageSizeInKilobytes property, which can be used to increase the max message
  size for Service Bus Premium namespaces. (#18310)

### Bugs Fixed

- Handle a missing CountDetails node in the returned responses for Get<Entity>RuntimeProperties which could cause a panic. (#18213)
- Adding the `associated-link-name` property to management operations (RenewLock, settlement and others), which 
  can help extend link lifetime (#18291)
- Namespace closing didn't reset the internal client, which could lead to connection recovery thrashing. (#18323)

## 1.0.0 (2022-05-16)

### Features Added

- First stable release of the azservicebus package.

## 0.4.1 (2022-05-12)

### Features Added

- Exported log.Event constants for azservicebus. This will make them easier to
  discover and they are also documented. NOTE: The log messages themselves 
  are not guaranteed to be stable. (#17596)
- `admin.Client` can now manage authorization rules and subscription filters and 
  actions. (#17616)
- Exported an official `*azservicebus.Error` type that gets returned if the failure is
  actionable. This can indicate if the connection was lost and could not be
  recovered with the configured retries or if a message lock was lost, which would cause
  message settlement to fail. 

  See the `ExampleReceiver_ReceiveMessages` in example_receiver_test.go for an example 
  on how to use it. (#17786)

### Breaking Changes

- `admin.Client` can now be configured using `azcore.Options`. (#17796)
- `ReceivedMessage.TransactionPartitionKey` has been removed as this library doesn't support transactions.
- `ReceivedMessage.Body()` is now a field. `Body` will be nil in the cases where it would have returned an error (where the underlying AMQP message had a payload in .Value, .Sequence or had multiple byte slices in .Data). (#17888)

### Bugs Fixed

- Fixing issue where the AcceptNextSessionForQueue and AcceptNextSessionForSubscription 
  couldn't be cancelled, forcing the user to wait for the service to timeout. (#17598)
- Fixing bug where there was a chance that internally cached messages would not be returned when
  the receiver was draining. (#17893)

## 0.4.0 (2022-04-06)

### Features Added

- Support for using a SharedAccessSignature in a connection string. Ex: `Endpoint=sb://<sb>.servicebus.windows.net;SharedAccessSignature=SharedAccessSignature sr=<sb>.servicebus.windows.net&sig=<base64-sig>&se=<expiry>&skn=<keyname>` (#17314)

### Bugs Fixed

- Fixed bug where message batch size calculation was inaccurate, resulting in batches that were too large to be sent. (#17318)
- Fixing an issue with an entity not being found leading to a longer timeout than needed. (#17279)
- Fixed the RPCLink so it does better handling of connection/link failures. (#17389)
- Fixed issue where a message lock expiring would cause unnecessary retries. These retries could cause message settlement calls (ex: Receiver.CompleteMessage) 
  to appear to hang. (#17382)
- Fixed issue where a cancellation on ReceiveMessages() would work, but wouldn't return the proper cancellation error. (#17422)

### Breaking Changes

- This module now requires Go 1.18
- Multiple functions have had `options` parameters added.
- `SessionReceiver.RenewMessageLock` has been removed - it isn't used for sessions. SessionReceivers should use `SessionReceiver.RenewSessionLock`.
- The `admin.Client` type has been changed to conform with the latest Azure Go SDK guidelines. As part of this:
  - Embedded `*Result` structs in `admin.Client`'s APIs have been removed. Inner *Properties values have been hoisted up to the `*Response` instead.
  - `.Response` fields have been removed for successful results. These will be added back using a     different pattern in the next release.
  - Fields that were of type `time.Duration` have been changed to `*string`, where the value of the string is an ISO8601 timestamp. 
    Affected fields from Queues, Topics and Subscriptions: AutoDeleteOnIdle, DefaultMessageTimeToLive, DuplicateDetectionHistoryTimeWindow, LockDuration.    
  - Properties that were passed as a parameter to CreateQueue, CreateTopic or CreateSubscription are now in the `options` parameter (as they were optional):
    Previously:
    ```go
    // older code
    adminClient.CreateQueue(context.Background(), queueName, &queueProperties, nil)	  
    ```

    And now:
    ```go
    // new code
    adminClient.CreateQueue(context.Background(), queueName, &admin.CreateQueueOptions{
      Properties: queueProperties,
    })
    ```  
  - Pagers have been changed to use the new generics-based `runtime.Pager`:
  
    Previously:
    ```go
    // older code
    for queuePager.NextPage(context.TODO()) {
		  for _, queue := range queuePager.PageResponse().Items {
			  fmt.Printf("Queue name: %s, max size in MB: %d\n", queue.QueueName, *queue.MaxSizeInMegabytes)
		  }
	  }
    
    if err := queuePager.Err(); err != nil {
      panic(err)
    }
    ```
    And now:

    ```go
    // new code
    for queuePager.More() {
		  page, err := queuePager.NextPage(context.TODO())

		  if err != nil {
			  panic(err)
		  }

		  for _, queue := range page.Queues {
			  fmt.Printf("Queue name: %s, max size in MB: %d\n", queue.QueueName, *queue.MaxSizeInMegabytes)
		  }
	  }
    ```

## 0.3.6 (2022-03-08)

### Bugs Fixed

- Fix connection recovery in situations where network errors bubble up from go-amqp. (#17048)
- Quicker reattach for idle links. (#17205)
- Quick exit on receiver reconnects to avoid potentially returning duplicate messages. (#17157)

### Breaking Changes

- The following 'Get' APIs have been changed to return a nil result when an item is not found: (#17229)
  - GetQueue, GetQueueRuntimeProperties
  - GetTopic, GetTopicRuntimeProperties
  - GetSubscription, GetSubscriptionRuntimeProperties

## 0.3.5 (2022-02-10)

### Bugs Fixed

- Fix panic() when go-amqp was returning an incorrect error on drain failures. (#17036)

## 0.3.4 (2022-02-08)

### Features Added

- Allow RetryOptions to be configured in the options for azservicebus.Client as well and admin.Client(#16831)
- Add in the MessageState property to the ReceivedMessage. (#16985)

### Bugs Fixed

- Fix unaligned 64-bit atomic operation on mips.  Thanks to @jackesdavid for contributing this fix. (#16847)
- Multiple fixes to address connection/link recovery (#16831)
- Fixing panic() when the links haven't been initialized (early cancellation) (#16941)
- Handle 500 as a retryable code (no recovery needed) (#16925)

## 0.3.3 (2022-01-12)

### Features Added

- Support the pass-through of an Application ID when constructing an Azure Service Bus Client. PR#16558 (thanks halspang!)

### Bugs Fixed 

- Fixing connection/link recovery in Sender.SendMessages() and Sender.SendMessageBatch(). PR#16790
- Fixing bug in the management link which could cause it to panic during recovery. PR#16790

## 0.3.2 (2021-12-08)

### Features Added

- Enabling websocket support via `ClientOptions.NewWebSocketConn`. For an example, see the `ExampleNewClient_usingWebsockets` 
  function in `example_client_test.go`.

### Breaking Changes

- Message properties that come from the standard AMQP message have been made into pointers, to allow them to be 
  properly omitted (or indicate that they've been omitted) when sending and receiving.  

### Bugs Fixed

- Session IDs can now be blank - prior to this release it would cause an error. PR#16530
- Drain will no longer hang if there is a link failure. Thanks to @flexarts for reporting this issue: PR#16530
- Attempting to settle messages received in ReceiveAndDelete mode would cause a panic. PR#16255

### Other Changes
- Removed legacy dependencies, resulting in a much smaller package.

## 0.3.1 (2021-11-16)

### Bugs Fixed

- Updating go-amqp to v0.16.4 to fix a race condition found when running `go test -race`.  Thanks to @peterzeller for reporting this issue. PR: #16168

## 0.3.0 (2021-11-12)

### Features Added

- AbandonMessage and DeferMessage now take an additional `PropertiesToModify` option, allowing
  the message properties to be modified when they are settled.
- Missing fields for entities in the admin.Client have been added (UserMetadata, etc..)

### Breaking Changes

- AdminClient has been moved into the `admin` subpackage.
- ReceivedMessage.Body is now a function that returns a ([]byte, error), rather than being a field.
  This protects against a potential data-loss scenario where a message is received with a payload 
  encoded in the sequence or value sections of an AMQP message, which cannot be properly represented
  in the .Body. This will now return an error.
- Functions that have options or might have options in the future have an additional *options parameter.
  As usual, passing 'nil' ignores the options, and will cause the function to use defaults.
- MessageBatch.Add() has been renamed to MessageBatch.AddMessage(). AddMessage() now returns only an `error`, 
  with a sentinel error (ErrMessageTooLarge) signaling that the batch cannot fit a new message.
- Sender.SendMessages() has been removed in favor of simplifications made in MessageBatch.

### Bugs Fixed

- ReceiveMessages has been tuned to match the .NET limits (which has worked well in practice). This partly addresses #15963, 
  as our default limit was far higher than needed.

## 0.2.0 (2021-11-02)

### Features Added

- Scheduling messages to be delivered at a later date, via the `Sender.ScheduleMessage(s)` function or 
  setting `Message.ScheduledEnqueueTime`.
- Added in the `Sender.SendMessages([slice of sendable messages])` function, which batches messages 
  automatically. Useful when you're sending multiple messages that you are already sure will be small
  enough to fit into a single batch.
- Receiving from sessions using a SessionReceiver, created using Client.AcceptSessionFor(Queue|Subscription)
  or Client.AcceptNextSessionFor(Queue|Subscription).
- Can fully create, update, delete and list queues, topics and subscriptions using the `AdministrationClient`.
- Can renew message and session locks, using Receiver.RenewMessageLock() and SessionReceiver.RenewSessionLock(), respectively.

### Bugs Fixed

- Receiver.ReceiveMessages() had a bug where multiple calls could result in the link no longer receiving messages.
  This was fixed with an update in go-amqp.

## 0.1.0 (2021-10-05)

- Initial preview for the new version of the Azure Service Bus Go SDK. 
