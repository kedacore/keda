# Release History

## 1.5.1 (2026-01-07)

### Bugs Fixed

* Fixed attach frame nil terminus check expectations

## 1.5.0 (2025-09-04)

* Added support for marshaling and unmarshaling arrays of maps

## 1.4.0 (2025-02-19)

### Features Added

* Added support in both `Sender` and `Receiver` to set the `desired-capabilities` in their ATTACH frames, using DesiredCapabilities in their respective Options.
* Added Receiver.DrainCredit, which allows you to drain credits from a link.

### Bugs Fixed

* Fixed encoding and decoding larger timestamp values, like .NET's DateTime.MaxValue.

## 1.4.0-beta.1 (2024-12-05)

### Features Added

* Added `Sender` support for delayed confirmation of message settlement and retrieval of delivery state.
  * `Sender.SendWithReceipt` sends a message and returns a `SendReceipt`.
  * `SendReceipt.Wait` waits for confirmation of settlement and returns the message's delivery state.
  * The `DeliveryState` interface encapsulates concrete delivery outcomes `StateAccepted`, `StateModified`, `StateRejected`, `StateReleased` and
    non-terminal delivery state `StateReceived`.

## 1.3.0 (2024-12-03)

### Features Added

* Added methods `Done` and `Err` to `Conn`
  * `Done` returns a channel that's closed when `Conn` has closed.
  * `Err` explains why `Conn` was closed.
* encoding.Symbol was exposed as a public type `Symbol`.

## 1.2.0 (2024-09-30)

### Features Added

* Added type `Null` used to send an AMQP `null` message value.
* Added method `Properties` to `Conn`, `Session`, `Receiver`, and `Sender` which contains the peer's respective properties.

### Bugs Fixed

* Fixed a rare race in `Conn.start` that could cause goroutines to be leaked if the provided context was canceled/expired.

### Other Changes

* The field `MessageHeader.Durable` is not omitted when it's `false`.

## 1.1.0 (2024-08-20)

### Features Added

* ConnError, SessionError and LinkError now work with errors.As(), making it easier to write generalized error handling code that wants to deal with *amqp.Error's.

## 1.0.5 (2024-03-04)

### Bugs Fixed

* Fixed an issue that could cause delays when parsing small frames.

## 1.0.4 (2024-01-16)

### Other Changes

* A `Receiver`'s unsettled messages are tracked as a count (currently used for diagnostic purposes only).

## 1.0.3 (2024-01-09)

### Bugs Fixed

* Fixed an issue that could cause a memory leak when settling messages across `Receiver` instances.

## 1.0.2 (2023-09-05)

### Bugs Fixed

* Fixed an issue that could cause frames to be sent even when the provided `context.Context` was cancelled.
* Fixed a potential hang in `Sender.Send()` that could happen in rare circumstances.
* Ensure that `Sender`'s delivery count and link credit are updated when a transfer fails to send due to context cancellation/timeout.

## 1.0.1 (2023-06-08)

### Bugs Fixed

* Fixed an issue that could cause links to terminate with error "received disposition frame with unknown link handle X".

## 1.0.0 (2023-05-04)

### Features Added

* Added `ConnOptions.WriteTimeout` to control the write deadline when writing to `net.Conn`.

### Bugs Fixed

* Calling `Dial()` with a cancelled context doesn't create a connection.
* Context cancellation is properly honored in calls to `Dial()` and `NewConn()`.
* Fixed potential race during `Conn.Close()`.
* Disable sending frames when closing `Session`, `Sender`, and `Receiver`.
* Don't leak in-flight messages when a message settlement API is cancelled or times out waiting for acknowledgement.
* `Sender.Send()` will return an `*amqp.Error` with condition `amqp.ErrCondTransferLimitExceeded` when attempting to send a transfer on a link with no credit.
* `Sender.Send()` will return an `*amqp.Error` with condition `amqp.ErrCondMessageSizeExceeded` if the message or delivery tag size exceeds the maximum allowed size for the link.

### Other Changes

* Debug logging includes the address of the object that's writing a log entry.
* Context expiration or cancellation when creating instances of `Session`, `Receiver`, and `Sender` no longer result in the potential for `Conn` to unexpectedly terminate.
* Session channel and link handle exhaustion will now return `*ConnError` and `*SessionError` respectively, closing the respective `Conn` or `Session`.
* If a `context.Context` contains a deadline/timeout, that value will be used as the write deadline when writing to `net.Conn`.

## 0.19.1 (2023-03-31)

### Bugs Fixed

* Fixed a race closing a `Session`, `Receiver`, or `Sender` in succession when the first attempt times out.
* Check the `LinkError.RemoteErr` field when determining if a link was cleanly closed.

## 0.19.0 (2023-03-30)

### Breaking Changes

* `Dial()` and `NewConn()` now require a `context.Context` as their first parameter.
  * As a result, the `ConnOptions.Timeout` field has been removed.
* Methods `Sender.Send()` and `Receiver.Receive()` now take their respective options-type as the final argument.
* The `ManualCredits` field in `ReceiverOptions` has been consolidated into field `Credit`.
* Renamed fields in the `ReceiverOptions` for configuring options on the source.
* Renamed `DetachError` to `LinkError` as "detach" has a specific meaning which doesn't equate to the returned link errors.
* The `Receiver.DrainCredit()` API has been removed.
* Removed fields `Batching` and `BatchMaxAge` in `ReceiverOptions`.
* The `IncomingWindow` and `OutgoingWindow` fields in `SessionOptions` have been removed.
* The field `SenderOptions.IgnoreDispositionErrors` has been removed.
  * By default, messages that are rejected by the peer no longer close the `Sender`.
* The field `SendSettled` in type `Message` has been moved to type `SendOptions` and renamed as `Settled`.
* The following type aliases have been removed.
  * `Address`, `Binary`, `MessageID`, `SequenceNumber`, `Symbol`
* Method `Message.LinkName()` has been removed.

### Bugs Fixed

* Don't discard incoming frames while closing a Session.
* Client-side termination of a Session due to invalid state will wait for the peer to acknowledge the Session's end.
* Fixed an issue that could cause `creditor.Drain()` to return the wrong error when a link is terminated.
* Ensure that `Receiver.Receive()` drains prefetched messages when the link closed.
* Fixed an issue that could cause closing a `Receiver` to hang under certain circumstances.
* In `Receiver.Drain()`, wake up `Receiver.mux()` after the drain bit has been set.

### Other Changes

* Debug logging has been cleaned up to reduce the number of redundant entries and consolidate the entry format.
  * DEBUG_LEVEL 1 now captures all sent/received frames along with basic flow control information.
  * Higher debug levels add entries when a frame transitions across mux boundaries and other diagnostics info.
* Document default values for incoming and outgoing windows.
* Refactored handling of incoming frames to eliminate potential deadlocks due to "mux pumping".
* Disallow sending of frames once the end performative has been sent.
* Clean up client-side state when a `context.Context` expires or is cancelled and document the potential side-effects.
* Unexpected frames will now terminate a `Session`, `Receiver`, or `Sender` as required.
* Cleaned up tests that triggered the race detector.

## 0.18.1 (2023-01-17)

### Bugs Fixed

* Fixed an issue that could cause `Conn.connReader()` to become blocked in rare circumstances.
* Fixed an issue that could cause outgoing transfers to be rejected by some brokers due to out-of-sequence delivery IDs.
* Fixed an issue that could cause senders and receivers within the same session to deadlock if the receiver was configured with `ReceiverSettleModeFirst`.
* Enabled support for senders in an at-most-once configuration.

### Other Changes

* The connection mux goroutine has been removed, eliminating a potential source of deadlocks.
* Automatic link flow control is built on the manual creditor.
* Clarified docs that messages received from a sender configured in a mode other than `SenderSettleModeSettled` must be acknowledged.
* Clarified default value for `Conn.IdleTimeout` and removed unit prefix.

## 0.18.0 (2022-12-06)

### Features Added
* Added `ConnError` type that's returned when a connection is no longer functional.
* Added `SessionError` type that's returned when a session has been closed.
* Added `SASLType` used when configuring the SASL authentication mechanism.
* Added `Ptr()` method to `SenderSettleMode` and `ReceiverSettleMode` types.

### Breaking Changes
* The minimum version of Go required to build this module is now 1.18.
* The type `Client` has been renamed to `Conn`, and its constructor `New()` renamed to `NewConn()`.
* Removed `ErrConnClosed`, `ErrSessionClosed`, `ErrLinkClosed`, and `ErrTimeout` sentinel error types.
* The following methods now require a `context.Context` as their first parameter.
  * `Conn.NewSession()`, `Session.NewReceiver()`, `Session.NewSender()`
* Removed `context.Context` parameter and `error` return from method `Receiver.Prefetched()`.
* The following type names had the prefix `AMQP` removed to prevent stuttering.
  * `AMQPAddress`, `AMQPMessageID`, `AMQPSymbol`, `AMQPSequenceNumber`, `AMQPBinary`
* Various `Default*` constants are no longer exported.
* The args to `Receiver.ModifyMessage()` have changed.
* The "variadic config" pattern for `Conn`, `Session`, `Sender`, and `Receiver` constructors has been replaced with a struct-based config.
  * This removes the `ConnOption`, `SessionOption`, and `LinkOption` types and all of the associated configuration funcs.
  * The sender and receiver specific link options have been moved into their respective options types.
  * The `ConnTLS()` option was removed as part of this change.
* The `Dial()` and `New()` constructors now require an `*ConnOptions` parameter.
* `Conn.NewSession()` now requires a `*SessionOptions` parameter.
* `Session.NewSender()` now requires `target` address and `*SenderOptions` parameters.
* `Session.NewReceiver()` now requires `source` address and `*ReceiverOptions` parameters.
* The various SASL configuration funcs have been slightly renamed.
* The following constant types had their values renamed in accordance with the SDK design guidelines.
  * `SenderSettleMode`, `ReceiverSettleMode`, `ExpiryPolicy`
* Constant type `ErrorCondition` has been renamed to `ErrCond`.
  * The `ErrCond` values have had their names updated to include the `ErrCond` prefix.
* `LinkFilterSource` and `LinkFilterSelector` have been renamed to `NewLinkFilter` and `NewSelectorFilter` respectively.
* The `RemoteError` field in `DetachError` has been renamed.

### Bugs Fixed
* Fixed potential panic in `muxHandleFrame()` when checking for manual creditor.
* Fixed potential panic in `attachLink()` when copying source filters.
* `NewConn()` will no longer return a broken `*Conn` in some instances.
* Incoming transfer frames received during initial link detach are no longer discarded.
* Session will no longer flood peer with flow frames when half its incoming window is consumed.
* Newly created `Session` won't leak if the context passed to `Conn.NewSession()` expires before exit.
* Newly created `link` won't leak if the context passed to `link.attach()` expires before exit.
* Fixed an issue causing dispositions to hang indefinitely with batching enabled when the receiver link is detached.

### Other Changes
* Errors when reading/writing to the underlying `net.Conn` are now wrapped in a `ConnError` type.
* Disambiguate error message for distinct cases where a session wasn't found for the specified remote channel.
* Removed `link.Paused` as it didn't add much value and was broken in some cases.
* Only send one flow frame when a drain has been requested.
* Session window size increased to 5000.
* Creation and deletion of `Session` instances have been made deterministic.
* Allocation and deallocation of link handles has been made deterministic.
