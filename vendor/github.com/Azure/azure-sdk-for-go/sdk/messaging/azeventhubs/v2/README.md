# Azure Event Hubs Client Module for Go

[Azure Event Hubs](https://azure.microsoft.com/services/event-hubs/) is a big data streaming platform and event ingestion service from Microsoft. For more information about Event Hubs see: [link](https://learn.microsoft.com/azure/event-hubs/event-hubs-about).

Use the client library `github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2` in your application to:

- Send events to an event hub.
- Consume events from an event hub.

Key links:
- [Source code][source]
- [API Reference Documentation][godoc]
- [Product documentation](https://azure.microsoft.com/services/event-hubs/)
- [Samples][godoc_examples]

## Getting started

### Install the package

Install the Azure Event Hubs client module for Go with `go get`:

```bash
go get github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2
```

### Prerequisites

- [Supported](https://aka.ms/azsdk/go/supported-versions) version of Go
- An [Azure subscription](https://azure.microsoft.com/free/)
- An [Event Hub namespace](https://learn.microsoft.com/azure/event-hubs/).
- An Event Hub. You can create an event hub in your Event Hubs Namespace using the [Azure Portal](https://learn.microsoft.com/azure/event-hubs/event-hubs-create), or the [Azure CLI](https://learn.microsoft.com/azure/event-hubs/event-hubs-quickstart-cli).

### Authenticate the client

Event Hub clients are created using a TokenCredential from the [Azure Identity package][azure_identity_pkg], like [DefaultAzureCredential][default_azure_credential].
You can also create a client using a connection string.

#### Using a service principal
 - ConsumerClient: [link](https://aka.ms/azsdk/go/eventhubs/pkg#example-NewConsumerClient)
 - ProducerClient: [link](https://aka.ms/azsdk/go/eventhubs/pkg#example-NewProducerClient)

  For Event Hubs roles, see [Built-in roles for Azure Event Hubs](https://learn.microsoft.com/azure/event-hubs/authenticate-application#built-in-roles-for-azure-event-hubs).

#### Using a connection string
 - ConsumerClient: [link](https://aka.ms/azsdk/go/eventhubs/pkg#example-NewConsumerClientFromConnectionString)
 - ProducerClient: [link](https://aka.ms/azsdk/go/eventhubs/pkg#example-NewProducerClientFromConnectionString)

# Key concepts

An Event Hub [**namespace**](https://learn.microsoft.com/azure/event-hubs/event-hubs-features#namespace) can have multiple event hubs. Each event hub, in turn, contains [**partitions**](https://learn.microsoft.com/azure/event-hubs/event-hubs-features#partitions) which store events.

Events are published to an event hub using an [event publisher](https://learn.microsoft.com/azure/event-hubs/event-hubs-features#event-publishers). In this package, the event publisher is the [ProducerClient](https://aka.ms/azsdk/go/eventhubs/pkg#ProducerClient)

Events can be consumed from an event hub using an [event consumer](https://learn.microsoft.com/azure/event-hubs/event-hubs-features#event-consumers). In this package there are two types for consuming events: 
- The basic event consumer is the, in the [ConsumerClient](https://aka.ms/azsdk/go/eventhubs/pkg#ConsumerClient). This consumer is useful if you already known which partitions you want to receive from.
- A distributed event consumer, which uses Azure Blobs for checkpointing and coordination. This is implemented in the [Processor](https://aka.ms/azsdk/go/eventhubs/pkg#Processor). This is useful when you want to have the partition assignment be dynamically chosen, and balanced with other Processor instances.

For more information about Event Hubs features and terminology can be found here: [link](https://learn.microsoft.com/azure/event-hubs/event-hubs-features)

# Examples

Examples for various scenarios can be found on [pkg.go.dev](https://aka.ms/azsdk/go/eventhubs/pkg#pkg-examples) or in the example*_test.go files in our GitHub repo for [azeventhubs](https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs).

# Troubleshooting

For detailed troubleshooting information, refer to the [Event Hubs Troubleshooting Guide][eventhubs_troubleshooting].

### Logging

This module uses the classification-based logging implementation in `azcore`. To enable console logging for all SDK modules, set the environment variable `AZURE_SDK_GO_LOGGING` to `all`. 

Use the `azcore/log` package to control log event output or to enable logs for `azeventhubs/v2` only. For example:

```go
import (
  "fmt"
  azlog "github.com/Azure/azure-sdk-for-go/sdk/azcore/log"
  "github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2"
)

// print log output to stdout
azlog.SetListener(func(event azlog.Event, s string) {
    fmt.Printf("[%s] %s\n", event, s)
})

// pick the set of events to log
azlog.SetEvents(
  azeventhubs.EventConn,
  azeventhubs.EventAuth,
  azeventhubs.EventProducer,
  azeventhubs.EventConsumer,
)
```

## Contributing
For details on contributing to this repository, see the [contributing guide][azure_sdk_for_go_contributing].

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.microsoft.com.

When you submit a pull request, a CLA-bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., label, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

### Additional Helpful Links for Contributors  
Many people all over the world have helped make this project better.  You'll want to check out:

* [What are some good first issues for new contributors to the repo?](https://github.com/azure/azure-sdk-for-go/issues?q=is%3Aopen+is%3Aissue+label%3A%22up+for+grabs%22)
* [How to build and test your change][azure_sdk_for_go_contributing_developer_guide]
* [How you can make a change happen!][azure_sdk_for_go_contributing_pull_requests]
* Frequently Asked Questions (FAQ) and Conceptual Topics in the detailed [Azure SDK for Go wiki](https://github.com/azure/azure-sdk-for-go/wiki).

<!-- ### Community-->
### Reporting security issues and security bugs

Security issues and bugs should be reported privately, via email, to the Microsoft Security Response Center (MSRC) <secure@microsoft.com>. You should receive a response within 24 hours. If for some reason you do not, please follow up via email to ensure we received your original message. Further information, including the MSRC PGP key, can be found in the [Security TechCenter](https://www.microsoft.com/msrc/faqs-report-an-issue).

### License

Azure SDK for Go is licensed under the [MIT](https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs/LICENSE.txt) license.

<!-- LINKS -->
[azure_sdk_for_go_contributing]: https://github.com/Azure/azure-sdk-for-go/blob/main/CONTRIBUTING.md
[azure_sdk_for_go_contributing_developer_guide]: https://github.com/Azure/azure-sdk-for-go/blob/main/CONTRIBUTING.md#developer-guide
[azure_sdk_for_go_contributing_pull_requests]: https://github.com/Azure/azure-sdk-for-go/blob/main/CONTRIBUTING.md#pull-requests

[azure_identity_pkg]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity
[default_azure_credential]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#NewDefaultAzureCredential
[eventhubs_troubleshooting]: https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs/TROUBLESHOOTING.md
[source]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/messaging/azeventhubs
[godoc]: https://aka.ms/azsdk/go/eventhubs/pkg
[godoc_examples]: https://aka.ms/azsdk/go/eventhubs/pkg#pkg-examples
