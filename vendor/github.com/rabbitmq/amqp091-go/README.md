# Go RabbitMQ Client Library

[![amqp091-go](https://github.com/rabbitmq/amqp091-go/actions/workflows/tests.yml/badge.svg)](https://github.com/rabbitmq/amqp091-go/actions/workflows/tests.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/rabbitmq/amqp091-go.svg)](https://pkg.go.dev/github.com/rabbitmq/amqp091-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/rabbitmq/amqp091-go)](https://goreportcard.com/report/github.com/rabbitmq/amqp091-go)

This is a Go AMQP 0.9.1 client maintained by the [RabbitMQ core team](https://github.com/rabbitmq).
It was [originally developed by Sean Treadway](https://github.com/streadway/amqp).

## Differences from streadway/amqp

Some things are different compared to the original client,
others haven't changed.

### Package Name

This library uses a different package name. If moving from `streadway/amqp`,
using an alias may reduce the number of changes needed:

``` go
amqp "github.com/rabbitmq/amqp091-go"
```

### License

This client uses the same 2-clause BSD license as the original project.

### Public API Evolution

 This client retains key API elements as practically possible.
 It is, however, open to reasonable breaking public API changes suggested by the community.
 We don't have the "no breaking public API changes ever" rule and fully recognize
 that a good client API evolves over time.


## Project Maturity

This project is based on a mature Go client that's been around for over a decade.


## Supported Go Versions

This client supports two most recent Go release series.


## Supported RabbitMQ Versions

This project supports RabbitMQ versions starting with `2.0` but primarily tested
against [currently supported RabbitMQ release series](https://www.rabbitmq.com/versions.html).

Some features and behaviours may be server version-specific.

## Goals

Provide a functional interface that closely represents the AMQP 0.9.1 model
targeted to RabbitMQ as a server. This includes the minimum necessary to
interact the semantics of the protocol.

## Non-goals

Things not intended to be supported.

  * Auto reconnect and re-synchronization of client and server topologies.
    * Reconnection would require understanding the error paths when the
      topology cannot be declared on reconnect.  This would require a new set
      of types and code paths that are best suited at the call-site of this
      package.  AMQP has a dynamic topology that needs all peers to agree. If
      this doesn't happen, the behavior is undefined.  Instead of producing a
      possible interface with undefined behavior, this package is designed to
      be simple for the caller to implement the necessary connection-time
      topology declaration so that reconnection is trivial and encapsulated in
      the caller's application code.
  * AMQP Protocol negotiation for forward or backward compatibility.
    * 0.9.1 is stable and widely deployed.  AMQP 1.0 is a divergent
      specification (a different protocol) and belongs to a different library.
  * Anything other than PLAIN and EXTERNAL authentication mechanisms.
    * Keeping the mechanisms interface modular makes it possible to extend
      outside of this package.  If other mechanisms prove to be popular, then
      we would accept patches to include them in this package.
  * Support for [`basic.return` and `basic.ack` frame ordering](https://www.rabbitmq.com/confirms.html#when-publishes-are-confirmed).
    This client uses Go channels for certain protocol events and ordering between
    events sent to two different channels generally cannot be guaranteed.

## Usage

See the [_examples](_examples) subdirectory for simple producers and consumers executables.
If you have a use-case in mind which isn't well-represented by the examples,
please file an issue.

## Documentation

 * [Godoc API reference](http://godoc.org/github.com/rabbitmq/amqp091-go)
 * [RabbitMQ tutorials in Go](https://github.com/rabbitmq/rabbitmq-tutorials/tree/master/go)

## Contributing

Pull requests are very much welcomed.  Create your pull request on a non-main
branch, make sure a test or example is included that covers your change, and
your commits represent coherent changes that include a reason for the change.

See [CONTRIBUTING.md](CONTRIBUTING.md) for more information.

## License

BSD 2 clause, see LICENSE for more details.
