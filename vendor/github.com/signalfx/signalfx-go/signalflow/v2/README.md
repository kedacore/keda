# SignalFx SignalFlow Go Client

This is a client for [SignalFlow](https://dev.splunk.com/observability/docs/signalflow) that lets
you stream and analyze metric data in real-time for your organization.

## Installation

**You must use Go 1.19+ for the v2 of this client.**

```
go get github.com/signalfx/signalfx-go/signalflow/v2@latest
```

If you do not want to upgrade Go you can use the v1 version of the old client that supports older Go
versions:

```
go get github.com/signalfx/signalfx-go@v1.30.0
```

## Usage

The package must be imported like so:

```
import (
  "github.com/signalfx/signalfx-go/signalflow/v2"
)
```

See [./example/main.go]() for an example of how to use the client.

SignalFlow itself is documented at https://dev.splunk.com/observability/docs/signalflow/messages.


## Migration from v1

If you previously used v1 of this module, you can migrate to v2 by doing the following:

 - You must use Go 1.19+.

 - Remove any uses of the `MetadataTimeout` client option.  This has been replaced by the addition
   of a `ctx` argument on all of the metadata getters (see below).

 - Add a `context.Context` as the first argument to any of the `Computation` metadata getter
   methods.  You can control how long to wait for metadata by using a context with a timeout or
   cancel.  See [the example](./example/main.go).

 - Add a `context.Context` as the first argument to each of the `Client` SignalFlow method calls.
   This context can be used to cancel the calls in case of connection trouble.  Previously these
   calls could hang indefinitely.

 - Remove references to the `Computation.Done()` method, which returned a channel that would be
   closed when the computation was finished. You can know if the computation is finished based on
   when the `Data` channel is closed.

 - `Computation.Events` was removed entirely as it wasn't implemented correctly.  Reach out if you
   have a desire for it.
