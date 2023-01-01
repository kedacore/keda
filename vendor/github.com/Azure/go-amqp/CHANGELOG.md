# Release History

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
