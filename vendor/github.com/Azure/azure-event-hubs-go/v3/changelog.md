# Change Log

## `v3.6.2`

- Update dependencies.

## `v3.6.1`

- Deprecate module. Use github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs instead.

## `v3.6.0`

- Updated to latest `azure-amqp-common-go` and GA version of `go-amqp` modules.

## `v3.5.0`

- Updated to latest `azure-amqp-common-go` and `go-amqp` modules.

## `v3.4.1`

- Updated `golang.org/x/net` to the latest version. [#286](https://github.com/Azure/azure-event-hubs-go/pull/286)

## `v3.4.0`

- Updated to latest `azure-amqp-common-go` and `go-amqp` modules.

## `v3.3.20`

- Fixed issue with a lease existing on start [#277](https://github.com/Azure/azure-event-hubs-go/pull/277)

## `v3.3.19`

- Vendor a copy of `azblob` to avoid compilation errors with mismatched versions [#261](https://github.com/Azure/azure-event-hubs-go/issues/261)

## `v3.3.18`

- Fixing issue where the LeaserCheckpointer could fail with a "ContainerAlreadyExists" error. (#253)

## `v3.3.17`

Updating to the latest go-amqp and azure-amqp-common-go to take advantage of some underlying reliability and interface improvements (#245)

## `v3.3.16`

- Exporting a subset of AMQP message properties for the Dapr project.

## `v3.3.13`

- We no longer close the link when we receive disposition errors on sending. This allows
  us to return errors properly when doing parallel sends on a link that is being
  throttled. [PR#234](https://github.com/Azure/azure-event-hubs-go/pull/234)

## `v3.3.12`

- Fix bug in sender.Recover() where recovery could get stuck when a link was throttled. [PR#232](#https://github.com/Azure/azure-event-hubs-go/pull/232)

## `v3.3.11`

- Allow for controlling the maximum retry count when sending messages. [#226](https://github.com/Azure/azure-event-hubs-go/issues/226)

## `v3.3.10`

- Fix sender.Recover() to be goroutine safe [#218](https://github.com/Azure/azure-event-hubs-go/issues/218)
- Skip calling sender.Recover() for some errors [#219](https://github.com/Azure/azure-event-hubs-go/issues/219)

## `v3.3.9`

- update the checkpoint after the receiver options are applied
- return the error from reading an old checkpoint when initializing the receiver

## `v3.3.8`

- add option to customise initial checkpoint

## `v3.3.7`

- add option to prefix checkpoint blob paths

## `v3.3.6`

- fix goroutine leak on listener close

## `v3.3.5`

- Remove the check for temporary network errors in sender.go [#80](https://github.com/Azure/azure-event-hubs-go/issues/80)

## `v3.3.4`

- read AZURE_ENVIRONMENT variable from environment to use the specified value when creating NewHub

## `v3.3.3`

- EventBatchIterator drops messages which bigger than 1MB with an error

## `v3.3.2`

- passing a context to internal calls that use go-amqp that now expect a context
- updating dependencies in go.mod

## `v3.3.1`

- fixed panic caused by interface conversion in event.go [#182](https://github.com/Azure/azure-event-hubs-go/issues/182)
- apply Receive options after populating last stored checkpoint

## `v3.3.0`

- add support for sending and receiving custom annotations

## `v3.2.0`

- add IoT Hub system properties

## `v3.1.2`

- fix errors in message handling being ignored [#155](https://github.com/Azure/azure-event-hubs-go/issues/155)

## `v3.1.1`

- Azure storage SAS token regeneration fix [#157](https://github.com/Azure/azure-event-hubs-go/issues/157)

## `v3.1.0`

- add support for websocket connections with eph with `eph.WithWebSocketConnection()`

## `v2.0.4`

- add comment on the `PartitionID` field in `SystemProperties` to clarify that it will always return a nil value [#131](https://github.com/Azure/azure-event-hubs-go/issues/131)

## `v2.0.3`

- fix send on closed channel for GetLeases [#142](https://github.com/Azure/azure-event-hubs-go/issues/142)

## `v2.0.2`

- enable partitionKey for sendBatch to fix [#128](https://github.com/Azure/azure-event-hubs-go/issues/128)
- ensure sender receives ack'd messages from EH [#126](https://github.com/Azure/azure-event-hubs-go/issues/126)
- close `leaseCh` on function return in storage.(\*LeaserCheckpointer).GetLeases to fix [#136](https://github.com/Azure/azure-event-hubs-go/issues/136)

## `v2.0.1`

- update to amqp 0.11.2 & common 2.1.0 to fix [#115](https://github.com/Azure/azure-event-hubs-go/issues/115)
- added checkpoint attribute to receiver to fix [#95](https://github.com/Azure/azure-event-hubs-go/issues/95) and [#118](https://github.com/Azure/azure-event-hubs-go/issues/118)

## `v2.0.0`

- **breaking change:** moved github.com/Azure/azure-amqp-common-go/persist to
  github.com/Azure/azure-event-hubs-go/persist
- **breaking change:** changed batch message sending to use a safe batch iterator rather than leaving batch sizing to
  the consumer.
- move tracing to devigned/tab so to not have to take a direct dependency on opentracing or opencensus

## `v1.3.1`

- cleanup connection after making management request

## `v1.3.0`

- add `SystemProperties` to `Event` which contains immutable broker provided metadata (squence number, offset,
  enqueued time)

## `v1.2.0`

- add websocket support

## `v1.1.5`

- add sender recovery handling for `amqp.ErrLinkClose`, `amqp.ErrConnClosed` and `amqp.ErrSessionClosed`

## `v1.1.4`

- update to amqp 0.11.0 and change sender to use unsettled rather than receiver second mode

## `v1.1.3`

- fix leak in partition persistence
- fix discarding event properties on batch sending

## `v1.1.2`

- take dep on updated amqp common which has more permissive RPC status description parsing

## `v1.1.1`

- close sender when hub is closed
- ensure links, session and connections are closed gracefully

## `v1.1.0`

- add receive option to receive from a timestamp
- fix sender recovery on temporary network failures
- add LeasePersistenceInterval to Azure Storage LeaserCheckpointer to allow for customization of persistence interval
  duration

## `v1.0.1`

- fix the breaking change from storage; this is not a breaking change for this library
- move from dep to go modules

## `v1.0.0`

- change from OpenTracing to OpenCensus
- add more documentation for EPH
- variadic mgmt options

## `v0.4.0`

- add partition key to received event [#43](https://github.com/Azure/azure-event-hubs-go/pull/43)
- remove `Receive` in eph in favor of `RegisterHandler`, `UnregisterHandler` and `RegisteredHandlerIDs` [#45](https://github.com/Azure/azure-event-hubs-go/pull/45)

## `v0.3.1`

- simplify environmental construction by preferring SAS

## `v0.3.0`

- pin version of amqp

## `v0.2.1`

- update dependency on common to 0.3.2 to fix retry returning nil error

## `v0.2.0`

- add opentracing support
- add context to close functions (breaking change)

## `v0.1.2`

- remove an extraneous dependency on satori/uuid

## `v0.1.1`

- update common dependency to 0.2.4
- provide more feedback when sending using testhub
- retry send upon server-busy
- use a new connection for each sender and receiver

## `v0.1.0`

- initial release
- basic send and receive
- batched send
- offset persistence
- alpha event host processor with Azure storage persistence
- enabled prefetch batching
