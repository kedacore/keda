# Guide to migrate from `azure-service-bus-go` to `azservicebus`

This guide is intended to assist in the migration from the pre-release `azure-service-bus-go` package to the latest beta releases (and eventual GA) of the `github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus`.

# Migration benefits

The redesign of the Service Bus SDK offers better integration with Azure Identity, a simpler API surface that allows you to uniformly work with queues, topics, subscriptions and subqueues (for instance: dead letter queues).

## Simplified API surface

The redesign for the API surface of Service Bus involves changing the way that clients are created. We wanted to simplify the number of types needed to get started, while also providing clarity on how, as a user of the SDK, to manage the resources the SDK creates (connections, links, etc...)

- [`Namespace` to `Client` migration](#namespace-to-client-migration)
- [Sending messages](#sending-messages)
- [Sending messages in batches](#sending-messages-in-batches)
- [Processing and receiving messages](#processing-and-receiving-messages)
- [Using dead letter queues](#using-dead-letter-queues)

### Namespace to Client migration

One big change is that the top level "client" is now [Client](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus#Client), not `Namespace`:

Previous code:

```go
// previous code

ns, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString())
```

New (using `azservicebus`):

```go
// new code

client, err := azservicebus.NewClientFromConnectionString(connectionString, nil)
```

You can also use `azidentity` credentials. See the [Azure Identity integration](#azure-identity-integration) section
below.

### Sending messages

Sending is done from a [Sender](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus#Sender), which
works the same for queues or topics:

```go
sender, err := client.NewSender(queueOrTopicName, nil)

sender.SendMessage(context.TODO(), &azservicebus.Message{
  Body: []byte("hello world"),
}, nil)
```

### Sending messages in batches

Sending messages in batches is similar, except that the focus has been moved more
towards giving the user full control using the `MessageBatch` type.

```go
// Create a message batch. It will automatically be sized for the Service Bus
// Namespace's maximum message size.
messageBatch, err := sender.NewMessageBatch(context.TODO(), nil)

if err != nil {
  panic(err)
}

// Add a message to our message batch. This can be called multiple times.
err = messageBatch.AddMessage(&azservicebus.Message{
    Body: []byte(fmt.Sprintf("hello world")),
}, nil)

if errors.Is(err, azservicebus.ErrMessageTooLarge) {
  fmt.Printf("Message batch is full. We should send it and create a new one.\n")

  // send what we have since the batch is full
  err := sender.SendMessageBatch(context.TODO(), messageBatch, nil)

  if err != nil {
    panic(err)
  }
  
  // Create a new batch, add this message and start again.
} else if err != nil {
  panic(err)
}
```

### Processing and receiving messages

Receiving has been changed to be pull-based, rather than using callbacks. 

You can receive messages using the [Receiver](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus#Receiver), for receiving of messages in batches.

### Receivers

Receivers allow you to request messages in batches:

```go
receiver, err := client.NewReceiverForQueue(queue, nil)
// or for a subscription
receiver, err := client.NewReceiverForSubscription(topicName, subscriptionName, nil)

// receiving multiple messages at a time. 
messages, err := receiver.ReceiveMessages(context.TODO(), numMessages, nil)
```

### Using dead letter queues

Previously, you created a receiver through an entity struct, like Queue or Subscription:

```go
// previous code

queue, err := ns.NewQueue()
deadLetterReceiver, err := queue.NewDeadLetterReceiver()

// or

topic, err := ns.NewTopic("topic")
subscription, err := topic.NewSubscription("subscription")
deadLetterReceiver, err := subscription.NewDeadLetterReceiver()

// the resulting receiver was a `ReceiveOner` which had different
// functions than some of the more full-fledged receiving types.
```

Now, in `azservicebus`:

```go
// new code

receiver, err := client.NewReceiverForQueue(
	queueName,
	&azservicebus.ReceiverOptions{
		ReceiveMode: azservicebus.ReceiveModePeekLock,
		SubQueue:    azservicebus.SubQueueDeadLetter,
	})

//or

receiver, err := client.NewReceiverForSubscription(
  topicName,
  subscriptionName,
  &azservicebus.ReceiverOptions{
    ReceiveMode: azservicebus.ReceiveModePeekLock,
    SubQueue:    azservicebus.SubQueueDeadLetter,
  })
```

The `Receiver` type for a dead letter queue is the same as the receiver for a 
queue or subscription, making things more consistent.

### Message settlement

Message settlement functions have moved to the `Receiver`, rather than being on the `Message`. 

Previously:

```go
// previous code

receiver.Listen(ctx, servicebus.HandlerFunc(func(c context.Context, m *servicebus.Message) error {
  m.Complete(ctx)
  return nil
}))
```

Now, using `azservicebus`:

```go
// new code

// with a Receiver
messages, err := receiver.ReceiveMessages(ctx, 10, nil)

for _, message := range messages {
  err = receiver.CompleteMessage(ctx, message, nil)
}
```

# Azure Identity integration

Azure Identity has been directly integrated into the `Client` via the `NewClient()` function. This allows you to take advantage of conveniences like [DefaultAzureCredential](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#section-readme) or any of the supported types within the package.

In `azservicebus`:

```go
// import "github.com/Azure/azure-sdk-for-go/sdk/azidentity"

credential, err := azidentity.NewDefaultAzureCredential(nil)
client, err := azservicebus.NewClient("<ex: myservicebus.servicebus.windows.net>", credential, nil)
```

# Entity management using admin.Client

Administration features, like creating queues, topics and subscriptions, has been moved into a dedicated client (admin.Client).

```go
// note: import "github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/admin"
adminClient, err := admin.NewClientFromConnectionString(connectionString, nil)

// create a queue with default properties
resp, err := adminClient.CreateQueue(context.TODO(), "queue-name", nil)

// or create a queue and configure some properties
```

# Receiving with session entities

Entities that use sessions can now be be received from:

```go
// to get a specific session by ID
sessionReceiver, err := client.AcceptSessionForQueue(context.TODO(), "queue", "session-id", nil)
// or client.AcceptSessionForSubscription

// to get the next available session from Service Bus (service-assigned)
sessionReceiver, err := client.AcceptNextSessionForQueue(context.TODO(), "queue", nil)

// SessionReceiver's are similar to Receiver's with some additional functions:

// managing session state
sessionData, err := sessionReceiver.GetSessionState(context.TODO())
err = sessionReceiver.SetSessionState(context.TODO(), []byte("data"))

// renewing the lock associated with the session
err = sessionReceiver.RenewSessionLock(context.TODO())
```
