# Azure Service Bus Client Module for Go

[Azure Service Bus](https://azure.microsoft.com/services/service-bus/) is a highly-reliable cloud messaging service from Microsoft.

Use the client library `github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus` in your application to:

- Send messages to an Azure Service Bus Queue or Topic
- Receive messages from an Azure Service Bus Queue or Subscription

Key links:
- [Source code][source]
- [API Reference Documentation][godoc]
- [Product documentation](https://azure.microsoft.com/services/service-bus/)
- [Samples][godoc_examples]
- [Migration guide for `azure-service-bus-go` users](https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azservicebus/migrationguide.md)

## Getting started

### Install the package

Install the Azure Service Bus client module for Go with `go get`:

```bash
go get github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus
```

### Prerequisites
- Go, version 1.18 or higher
- An [Azure subscription](https://azure.microsoft.com/free/)
- A [Service Bus Namespace](https://docs.microsoft.com/azure/service-bus-messaging/).
- A Service Bus Queue, Topic or Subscription. You can create an entity in your Service Bus Namespace using the [Azure Portal](https://docs.microsoft.com/azure/service-bus-messaging/service-bus-quickstart-portal), or the [Azure CLI](https://docs.microsoft.com/azure/service-bus-messaging/service-bus-quickstart-cli).

### Authenticate the client

The Service Bus [Client][godoc_client] can be created using a Service Bus connection string or a credential from the [Azure Identity package][azure_identity_pkg], like [DefaultAzureCredential][default_azure_credential].

#### Using a Service Principal

```go
import (
  "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
  "github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

func main() {
  // For more information about the DefaultAzureCredential:
  // https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#NewDefaultAzureCredential
  credential, err := azidentity.NewDefaultAzureCredential(nil)

  if err != nil {
    panic(err)
  }

  // The service principal specified by the credential needs to be added to the appropriate Service Bus roles for your
  // resource. More information about Service Bus roles can be found here:
  // https://docs.microsoft.com/azure/service-bus-messaging/service-bus-managed-service-identity#azure-built-in-roles-for-azure-service-bus
  client, err := azservicebus.NewClient("<ex: myservicebus.servicebus.windows.net>", credential, nil)

  if err != nil {
    panic(err)
  }
}
```

#### Using a connection string

```go
import (
  "github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

func main() {
  // See here for instructions on how to get a Service Bus connection string:
  // https://docs.microsoft.com/azure/service-bus-messaging/service-bus-quickstart-portal#get-the-connection-string
  client, err := azservicebus.NewClientFromConnectionString(connectionString, nil)

  if err != nil {
    panic(err)
  }
}
```

## Key concepts

Once you've created a [Client][godoc_client], you can interact with resources within a Service Bus Namespace:

- [Queues][queue_concept]: Allows for sending and receiving messages. Often used for point-to-point communication.
- [Topics][topic_concept]: A topic is similar to a queue but splits the receiving and sending into separate entities. Messages are sent to a topic and are broadcast to Subscriptions, where they can be consumed independently, and in parallel by Receivers.
- [Subscriptions][subscription_concept]: The mechanism to consume from a Topic. Each subscription is independent, and receives a copy of each message sent to the topic. Rules and Filters can be used to tailor which messages are received by a specific subscription.

For more information about these resources, see [What is Azure Service Bus?][service_bus_overview].

Using a `Client` you can do the following:

- Send messages, to a queue or topic, using a [Sender][godoc_sender] created using [Client.NewSender()][godoc_newsender]. You can see an example here: [(link)](#send-messages)
- Receive messages, from either a queue or a subscription, using a [Receiver][godoc_receiver] created using [client.NewReceiverForQueue()][godoc_newreceiver_queue] or [client.NewReceiverForSubscription()][godoc_newreceiver_subscription]. You can see an example here: [(link)](#receive-messages)

Please note that the Queues, Topics and Subscriptions should be created prior to using this library.

## Examples

The following sections provide examples that cover common tasks using Azure Service Bus:

- [Send messages](#send-messages)
- [Receive messages](#receive-messages)
- [Dead lettering and subqueues](#dead-letter-queue)

### Send messages

Once you've created a [Client][godoc_client] you can create a [Sender][godoc_sender], which will allow you to send messages.

> NOTE: Creating a `azservicebus.Client` is covered in the ["Authenticate the client"](#authenticate-the-client) section of the readme, using a [Service Principal](#using-a-service-principal) or a [Service Bus connection string](#using-a-connection-string).

Sending a single message:

```go
err := sender.SendMessage(context.TODO(), &azservicebus.Message{
  Body: []byte("hello world!"),
}, nil)
```

#### Sending multiple messages using a batch

You can also send messages in batches, which can be more efficient than sending them individually

```go
// Create a message batch. It will automatically be sized for the Service Bus
// Namespace's maximum message size.
currentMessageBatch, err := sender.NewMessageBatch(context.TODO(), nil)

if err != nil {
  panic(err)
}

messagesToSend := []*azservicebus.Message{
  // any messages that you'd want to send would be here, or sourced from
  // somewhere else.
}

for i := 0; i < len(messagesToSend); i++ {
  // Add a message to our message batch. This can be called multiple times.
  err = currentMessageBatch.AddMessage(messagesToSend[i], nil)

  if errors.Is(err, azservicebus.ErrMessageTooLarge) {
    if currentMessageBatch.NumMessages() == 0 {
      // This means the message itself is too large to be sent, even on its own.
      // This will require intervention from the user.
      panic("Single message is too large to be sent in a batch.")
    }

    fmt.Printf("Message batch is full. Sending it and creating a new one.\n")

    // send what we have since the batch is full
    err := sender.SendMessageBatch(context.TODO(), currentMessageBatch, nil)

    if err != nil {
      panic(err)
    }

    // Create a new batch and retry adding this message to our batch.
    newBatch, err := sender.NewMessageBatch(context.TODO(), nil)

    if err != nil {
      panic(err)
    }

    currentMessageBatch = newBatch

    // rewind the counter and attempt to add the message again (this batch
    // was full so it didn't go out with the previous SendMessageBatch call).
    i--
  } else if err != nil {
    panic(err)
  }
}

// check if any messages are remaining to be sent.
if currentMessageBatch.NumMessages() > 0 {
  err := sender.SendMessageBatch(context.TODO(), currentMessageBatch, nil)

  if err != nil {
    panic(err)
  }
}
```

### Receive messages

Once you've created a [Client][godoc_client] you can create a [Receiver][godoc_receiver], which will allow you to receive messages.

> NOTE: Creating a `azservicebus.Client` is covered in the ["Authenticate the client"](#authenticate-the-client) section of the readme, using a [Service Principal](#using-a-service-principal) or a [Service Bus connection string](#using-a-connection-string).

```go
receiver, err := client.NewReceiverForQueue(
  "<queue>",
  nil,
)
// or
// client.NewReceiverForSubscription("<topic>", "<subscription>")

// ReceiveMessages respects the passed in context, and will gracefully stop
// receiving when 'ctx' is cancelled.
ctx, cancel := context.WithTimeout(context.TODO(), 60*time.Second)
defer cancel()

messages, err := receiver.ReceiveMessages(ctx,
  // The number of messages to receive. Note this is merely an upper
  // bound. It is possible to get fewer message (or zero), depending
  // on the contents of the remote queue or subscription and network
  // conditions.
  1,
  nil,
)

if err != nil {
  panic(err)
}

for _, message := range messages {
  // The message body is a []byte. For this example we're just assuming that the body
  // was a string, converted to bytes but any []byte payload is valid.
  var body []byte = message.Body
  fmt.Printf("Message received with body: %s\n", string(body))

  // For more information about settling messages:
  // https://docs.microsoft.com/azure/service-bus-messaging/message-transfers-locks-settlement#settling-receive-operations
  err = receiver.CompleteMessage(context.TODO(), message, nil)

  if err != nil {
    var sbErr *azservicebus.Error

    if errors.As(err, &sbErr) && sbErr.Code == azservicebus.CodeLockLost {
      // The message lock has expired. This isn't fatal for the client, but it does mean
      // that this message can be received by another Receiver (or potentially this one!).
      fmt.Printf("Message lock expired\n")

      // You can extend the message lock by calling receiver.RenewMessageLock(msg) before the
      // message lock has expired.
      continue
    }

    panic(err)
	}

  fmt.Printf("Received and completed the message\n")
}
```

### Dead letter queue

The dead letter queue is a **sub-queue**. Each queue or subscription has its own dead letter queue. Dead letter queues store
messages that have been explicitly dead lettered using the [Receiver.DeadLetterMessage][godoc_receiver_deadlettermessage] function.

Opening a dead letter queue is just a configuration option when creating a [Receiver][godoc_receiver].

> NOTE: Creating a `azservicebus.Client` is covered in the ["Authenticate the client"](#authenticate-the-client) section of the readme, using a [Service Principal](#using-a-service-principal) or a [Service Bus connection string](#using-a-connection-string).

```go
deadLetterReceiver, err := client.NewReceiverForQueue("<queue>",
  &azservicebus.ReceiverOptions{
    SubQueue: azservicebus.SubQueueDeadLetter,
  })
// or 
// client.NewReceiverForSubscription("<topic>", "<subscription>", 
//   &azservicebus.ReceiverOptions{
//     SubQueue: azservicebus.SubQueueDeadLetter,
//   })
```

To see some example code for receiving messages using the Receiver see the ["Receive messages"](#receive-messages) sample.

## Troubleshooting

### Logging

This module uses the classification-based logging implementation in `azcore`. To enable console logging for all SDK modules, set the environment variable `AZURE_SDK_GO_LOGGING` to `all`. 

Use the `azcore/log` package to control log event output or to enable logs for `azservicebus` only. For example:

```go
import (
  "fmt"
  azlog "github.com/Azure/azure-sdk-for-go/sdk/azcore/log"
)

// print log output to stdout
azlog.SetListener(func(event azlog.Event, s string) {
    fmt.Printf("[%s] %s\n", event, s)
})

// pick the set of events to log
azlog.SetEvents(
  // EventConn is used whenever we create a connection or any links (ie: receivers, senders).
  azservicebus.EventConn,
  // EventAuth is used when we're doing authentication/claims negotiation.
  azservicebus.EventAuth,
  // EventReceiver represents operations that happen on Receivers.
  azservicebus.EventReceiver,
  // EventSender represents operations that happen on Senders.
  azservicebus.EventSender,
  // EventAdmin is used for operations in the azservicebus/admin.Client
  azservicebus.EventAdmin,
)
```


## Next steps

Please take a look at the [samples][godoc_examples] for detailed examples on how to use this library to send and receive messages to/from [Service Bus Queues, Topics and Subscriptions](https://docs.microsoft.com/azure/service-bus-messaging/service-bus-messaging-overview).

## Contributing

If you'd like to contribute to this library, please read the [contributing guide](https://github.com/Azure/azure-sdk-for-go/blob/main/CONTRIBUTING.md) to learn more about how to build and test the code.

![Impressions](https://azure-sdk-impressions.azurewebsites.net/api/impressions/azure-sdk-for-go%2Fsdk%2Fmessaging%2Fazservicebus%2FREADME.png)

[new_issue]: https://github.com/Azure/azure-sdk-for-go/issues/new
[azure_identity_pkg]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity
[default_azure_credential]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#NewDefaultAzureCredential
[queue_concept]: https://docs.microsoft.com/azure/service-bus-messaging/service-bus-messaging-overview#queues
[topic_concept]: https://docs.microsoft.com/azure/service-bus-messaging/service-bus-messaging-overview#topics
[subscription_concept]: https://docs.microsoft.com/azure/service-bus-messaging/service-bus-queues-topics-subscriptions#topics-and-subscriptions
[service_bus_overview]: https://docs.microsoft.com/azure/service-bus-messaging/service-bus-messaging-overview
[msdoc_settling]: https://docs.microsoft.com/azure/service-bus-messaging/message-transfers-locks-settlement#settling-receive-operations
[source]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/messaging/azservicebus
[godoc]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus
[godoc_examples]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus#pkg-examples
[godoc_client]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/#Client
[godoc_sender]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/#Sender
[godoc_receiver]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/#Receiver
[godoc_receiver_completemessage]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/#Receiver.CompleteMessage
[godoc_receiver_deadlettermessage]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/#Receiver.DeadLetterMessage
[godoc_newsender]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/#Client.NewSender
[godoc_newreceiver_queue]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/#Client.NewReceiverForQueue
[godoc_newreceiver_subscription]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/#Client.NewReceiverForSubscription
