# Azure Event Grid Publisher Client Module for Go

[Azure Event Grid](https://learn.microsoft.com/azure/event-grid/overview) is a highly scalable, fully managed Pub Sub message distribution service that offers flexible message consumption patterns. For more information about Event Grid see: [link](https://learn.microsoft.com/azure/event-grid/overview).

The client in this package can publish events to [Event Grid topics](https://learn.microsoft.com/azure/event-grid/concepts).

> NOTE: This client does NOT work with Event Grid namespaces. Use the [Client][godoc_client] in the root package of this module instead.

Key links:
- [Source code][source]
- [API Reference Documentation][godoc]
- [Product documentation](https://azure.microsoft.com/services/event-grid/)
- [Samples][godoc_examples]

## Getting started

### Install the package

Install the Azure Event Grid client module for Go with `go get`:

```bash
go get github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventgrid
```

### Prerequisites

- Go, version 1.18 or higher
- An [Azure subscription](https://azure.microsoft.com/free/)
- An Event Grid topic. You can create an Event Grid topic using the [Azure Portal](https://learn.microsoft.com/azure/event-grid/custom-event-quickstart-portal).

### Authenticate the client

Event Grid publisher clients authenticate using either:
- A TokenCredential. An example of that can be viewed here: [ExampleNewClientWithSharedKeyCredential][godoc_example_newclient].
- A shared key credential. An example of that can be viewed here: [ExampleNewClientWithSharedKeyCredential][godoc_example_newclientsk]
- A Shared Access Signature (SAS). An example of that can be viewed here: [ExampleNewClientWithSharedKeyCredential][godoc_example_newclientsas].

# Key concepts

The client in this package can publish events to Azure Event Grid topics. Topics are published to using the [publisher.Client][godoc_publisher_client]. The topic
you publish to will be configured to accept events of a certain format: EventGrid, CloudEvent or Custom. Separate functions are available on the publisher client for each format.

# Examples

Examples for various scenarios can be found on [pkg.go.dev][godoc_examples] or in the example*_test.go files in our GitHub repo for [azeventgrid][gh].

# Troubleshooting

### Logging

This module uses the classification-based logging implementation in `azcore`. To enable console logging for all SDK modules, set the environment variable `AZURE_SDK_GO_LOGGING` to `all`. 

Use the `azcore/log` package to control log event output.

```go
import (
  "fmt"
  azlog "github.com/Azure/azure-sdk-for-go/sdk/azcore/log"
)

// print log output to stdout
azlog.SetListener(func(event azlog.Event, s string) {
    fmt.Printf("[%s] %s\n", event, s)
})
```

# Next steps

More sample code should go here, along with links out to the appropriate example tests.

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

Azure SDK for Go is licensed under the [MIT](https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/template/aztemplate/LICENSE.txt) license.

<!-- LINKS -->
[azure_sdk_for_go_contributing]: https://github.com/Azure/azure-sdk-for-go/blob/main/CONTRIBUTING.md
[azure_sdk_for_go_contributing_developer_guide]: https://github.com/Azure/azure-sdk-for-go/blob/main/CONTRIBUTING.md#developer-guide
[azure_sdk_for_go_contributing_pull_requests]: https://github.com/Azure/azure-sdk-for-go/blob/main/CONTRIBUTING.md#pull-requests
[azure_cli]: https://docs.microsoft.com/cli/azure
[azure_pattern_circuit_breaker]: https://docs.microsoft.com/azure/architecture/patterns/circuit-breaker
[azure_pattern_retry]: https://docs.microsoft.com/azure/architecture/patterns/retry
[azure_portal]: https://portal.azure.com
[azure_sub]: https://azure.microsoft.com/free/
[cloud_shell]: https://docs.microsoft.com/azure/cloud-shell/overview
[cloud_shell_bash]: https://shell.azure.com/bash
[source]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/messaging/azeventgrid
[godoc_client]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventgrid#Client

<!-- Temp links until the PR goes in, replacements are below -->
[godoc]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventgrid/
[godoc_examples]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventgrid/
[godoc_publisher_client]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventgrid/
[godoc_example_newclient]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventgrid/
[godoc_example_newclientsk]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventgrid/
[godoc_example_newclientsas]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventgrid/
[gh]: https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventgrid/

<!-- these links are all broken until I complete a PR
[godoc]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventgrid/publisher
[godoc_examples]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventgrid/publisher/#pkg-examples
[godoc_publisher_client]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventgrid/publisher#Client
[godoc_example_newclient]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventgrid/publisher/#example-NewClient
[godoc_example_newclientsk]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventgrid/publisher/#example-NewClientWithSharedKeyCredential
[godoc_example_newclientsas]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventgrid/publisher/#example-NewClientWithSAS
[gh]: https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventgrid/publisher
 -->
