# AMQP 1.0 Client Module for Go

[![PkgGoDev](https://pkg.go.dev/badge/github.com/Azure/go-amqp)](https://pkg.go.dev/github.com/Azure/go-amqp)
[![Build Status](https://dev.azure.com/azure-sdk/public/_apis/build/status/go/Azure.go-amqp?branchName=main)](https://dev.azure.com/azure-sdk/public/_build/latest?definitionId=1292&branchName=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/Azure/go-amqp)](https://goreportcard.com/report/github.com/Azure/go-amqp)
[![MIT licensed](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/Azure/go-amqp/main/LICENSE)

The [amqp][godoc_amqp] module is an AMQP 1.0 client implementation for Go.

[AMQP 1.0][amqp_spec] is not compatible with AMQP 0-9-1 or 0-10.

## Getting Started

### Prerequisites

- Go 1.18 or later
- An AMQP 1.0 compliant [broker][broker_listing]

### Install the module

```sh
go get github.com/Azure/go-amqp
```

### Connect to a broker

Call [amqp.Dial()][godoc_dial] to connect to an AMQP broker. This creates an [*amqp.Conn][godoc_conn].

```go
conn, err := amqp.Dial(context.TODO(), "amqp[s]://<host name of AMQP 1.0 broker>", nil)
if err != nil {
	// handle error
}
```

### Sending and receiving messages

In order to send or receive messages, first create an [*amqp.Session][godoc_session] from the [*amqp.Conn][godoc_conn] by calling [Conn.NewSession()][godoc_conn_session].

```go
session, err := conn.NewSession(context.TODO(), nil)
if err != nil {
	// handle error
}
```

Once the session has been created, create an [*amqp.Sender][godoc_sender] to send messages and/or an [*amqp.Receiver][godoc_receiver] to receive messages by calling [Session.NewSender()][godoc_session_sender] and/or [Session.NewReceiver()][godoc_session_receiver] respectively.

```go
// create a new sender
sender, err := session.NewSender(context.TODO(), "<name of peer's receiving terminus>", nil)
if err != nil {
	// handle error
}

// send a message
err = sender.Send(context.TODO(), amqp.NewMessage([]byte("Hello!")), nil)
if err != nil {
	// handle error
}

// create a new receiver
receiver, err := session.NewReceiver(context.TODO(), "<name of peer's sending terminus>", nil)
if err != nil {
	// handle error
}

// receive the next message
msg, err := receiver.Receive(context.TODO(), nil)
if err != nil {
	// handle error
}
```

## Key concepts

- An [*amqp.Conn][godoc_conn] connects a client to a broker (e.g. Azure Service Bus).
- Once a connection has been established, create one or more [*amqp.Session][godoc_session] instances.
- From an [*amqp.Session][godoc_session] instance, create one or more senders and/or receivers.
  - An [*amqp.Sender][godoc_sender] is used to send messages from the client to a broker.
  - An [*amqp.Receiver][godoc_receiver] is used to receive messages from a broker to the client.

For a complete overview of AMQP's conceptual model, please consult section [2.1 Transport][section_2_1] of the AMQP 1.0 specification.

## Examples

The following examples cover common scenarios for sending and receiving messages:

- [Create a message](#create-a-message)
- [Send message](#send-message)
- [Receive messages](#receive-messages)

### Create a message

A message can be created in two different ways.  The first is to simply instantiate a new instance of the [*amqp.Message][godoc_message] type, populating the required fields.

```go
msg := &amqp.Message{
	// populate fields (Data is the most common)
}
```

The second is the [amqp.NewMessage][godoc_message_ctor] constructor. It passes the provided `[]byte` to the first entry in the `*amqp.Message.Data` slice.

```go
msg := amqp.NewMessage(/* some []byte */)
```

This is purely a convenience constructor as many AMQP brokers expect a message's data in the `Data` field.

### Send message

Once an [*amqp.Session][godoc_session] has been created, create an [*amqp.Sender][godoc_sender] in order to send messages.

```go
sender, err := session.NewSender(context.TODO(), "<name of peer's receiving terminus>", nil)
```

Once the [*amqp.Sender][godoc_sender] has been created, call [Sender.Send()][godoc_sender_send] to send an [*amqp.Message][godoc_message].

```go
err := sender.Send(context.TODO(), msg, nil)
```

Depending on the sender's configuration, the call to [Sender.Send()][godoc_sender_send] will block until the peer has acknowledged the message was received.
The amount of time the call will block is dependent upon network latency and the peer's load, but is usually in a few dozen milliseconds.

### Receive messages

Once an [*amqp.Session][godoc_session] has been created, create an [*amqp.Receiver][godoc_receiver] in order to receive messages.

```go
receiver, err := session.NewReceiver(context.TODO(), "<name of peer's sending terminus>", nil)
```

Once the [*amqp.Receiver][godoc_receiver] has been created, call [Receiver.Receive()][godoc_receiver_receive] to wait for an incoming message.

```go
msg, err := receiver.Receive(context.TODO(), nil)
```

Note that calls to [Receiver.Receive()][godoc_receiver_receive] will block until either a message has been received or, if applicable, the provided [context.Context][godoc_context] has been cancelled and/or its deadline exceeded.

After an [*amqp.Message][godoc_message] message has been received and processed, as the final step it's **imperative** that the [*amqp.Message][godoc_message] is passed to one of the acknowledgement methods on the [*amqp.Receiver][godoc_receiver].

- [Receiver.AcceptMessage][godoc_receiver_accept] - the client has accepted the message and no redelivery is required (most common)
- [Receiver.ModifyMessage][godoc_receiver_modify] - the client has modified the message and released it for redelivery with the specified modifications
- [Receiver.RejectMessage][godoc_receiver_reject] - the message is invalid and therefore cannot be processed
- [Receiver.ReleaseMessage][godoc_receiver_release] - the client has released the message for redelivery without any modifications

```go
err := receiver.AcceptMessage(context.TODO(), msg)
```

## Next steps

See the [examples][godoc_examples] for complete end-to-end examples on how to use this module.

## Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.opensource.microsoft.com.

When you submit a pull request, a CLA bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

[amqp_spec]: http://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-overview-v1.0-os.html
[broker_listing]: https://github.com/xinchen10/awesome-amqp
[section_2_1]: http://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-transport-v1.0-os.html#section-transport
[godoc_amqp]: https://pkg.go.dev/github.com/Azure/go-amqp
[godoc_examples]: https://pkg.go.dev/github.com/Azure/go-amqp#pkg-examples
[godoc_conn]: https://pkg.go.dev/github.com/Azure/go-amqp#Conn
[godoc_conn_session]: https://pkg.go.dev/github.com/Azure/go-amqp#Conn.NewSession
[godoc_dial]: https://pkg.go.dev/github.com/Azure/go-amqp#Dial
[godoc_context]: https://pkg.go.dev/context#Context
[godoc_message]: https://pkg.go.dev/github.com/Azure/go-amqp#Message
[godoc_message_ctor]: https://pkg.go.dev/github.com/Azure/go-amqp#NewMessage
[godoc_session]: https://pkg.go.dev/github.com/Azure/go-amqp#Session
[godoc_session_sender]: https://pkg.go.dev/github.com/Azure/go-amqp#Session.NewSender
[godoc_session_receiver]: https://pkg.go.dev/github.com/Azure/go-amqp#Session.NewReceiver
[godoc_sender]: https://pkg.go.dev/github.com/Azure/go-amqp#Sender
[godoc_sender_send]: https://pkg.go.dev/github.com/Azure/go-amqp#Sender.Send
[godoc_receiver]: https://pkg.go.dev/github.com/Azure/go-amqp#Receiver
[godoc_receiver_accept]: https://pkg.go.dev/github.com/Azure/go-amqp#Receiver.AcceptMessage
[godoc_receiver_modify]: https://pkg.go.dev/github.com/Azure/go-amqp#Receiver.ModifyMessage
[godoc_receiver_reject]: https://pkg.go.dev/github.com/Azure/go-amqp#Receiver.RejectMessage
[godoc_receiver_release]: https://pkg.go.dev/github.com/Azure/go-amqp#Receiver.ReleaseMessage
[godoc_receiver_receive]: https://pkg.go.dev/github.com/Azure/go-amqp#Receiver.Receive
