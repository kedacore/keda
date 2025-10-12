# Azure Monitor Query Metrics client module for Go

* Query metrics (this module): execute read-only queries against [Azure Monitor Metrics][metrics_overview]
* Query logs ([query/azlogs][azlogs]): execute read-only queries against [Azure Monitor Logs][logs_overview]
* Upload logs ([ingestion/azlogs][ingestion_azlogs]): send custom logs to [Azure Monitor][azure_monitor_overview] using the [Logs Ingestion API][ingestion_overview]

[Source code][azmetrics_repo] | [Package (pkg.go.dev)][azmetrics_pkg_go] | [REST API documentation][monitor_rest_docs] | [Product documentation][monitor_docs] | [Samples][examples]

## Getting started

### Prerequisites

* [Supported](https://aka.ms/azsdk/go/supported-versions) version of Go - [Install Go](https://go.dev/doc/install)
* Azure subscription - [Create a free account][azure_sub]
* The resource URI of an Azure resource (Storage Account, Key Vault, CosmosDB, etc.) that you plan to monitor
* Regional endpoint when instantiating the client (for example, "https://westus3.metrics.monitor.azure.com")

### Install the packages

Install the `azmetrics` and `azidentity` modules with `go get`:

```bash
go get github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics
go get github.com/Azure/azure-sdk-for-go/sdk/azidentity
```

The [azidentity][azure_identity] module is used for client authentication.

### Authentication

An authenticated client object is required to execute a query. The examples demonstrate using [azidentity.NewDefaultAzureCredential][default_cred_ref] to authenticate; however, the client accepts any [azidentity][azure_identity] credential. See the [azidentity][azure_identity] documentation for more information about other credential types.

The client defaults to the Azure public cloud. For other cloud configurations, see the [cloud][cloud_documentation] package documentation.

#### Create a client

Example [client][example_client]

## Key concepts

[Azure Monitor Metrics][metrics_overview] collects numeric data from monitored resources into a time series database. Metrics are numerical values that are collected at regular intervals and describe some aspect of a system at a particular time. Metrics are lightweight and capable of supporting near real-time scenarios, making them particularly useful for alerting and fast detection of issues.

### Metrics data structure

Each set of metric values is a time series with the following characteristics:

- The time the value was collected
- The resource associated with the value
- A namespace that acts like a category for the metric
- A metric name
- The value itself
- Some metrics may have multiple dimensions as described in [multi-dimensional metrics][multi-metrics]. Custom metrics can have up to 10 dimensions.

To discover the metrics available to query, please reference the [supported metrics documentation][supported_metrics]. The documentation also describes important details like the valid unit, aggregation, and time grain for each metric.

## Examples

Get started with our [examples][examples].

## Contributing

This project welcomes contributions and suggestions. Most contributions require you to agree to a [Contributor License Agreement (CLA)][cla] declaring that you have the right to, and actually do, grant us the rights to use your contribution.

When you submit a pull request, a CLA-bot will automatically determine whether you need to provide a CLA and decorate
the PR appropriately (e.g., label, comment). Simply follow the instructions provided by the bot. You will only need to
do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct][coc]. For more information, see
the [Code of Conduct FAQ][coc_faq] or contact [opencode@microsoft.com][coc_contact] with any additional questions or
comments.

<!-- LINKS -->
[azlogs]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azlogs
[azmetrics_repo]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/monitor/query/azmetrics
[azmetrics_pkg_go]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics
[azure_identity]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity
[azure_monitor_overview]: https://learn.microsoft.com/azure/azure-monitor/overview
[azure_sub]: https://azure.microsoft.com/free/
[cloud_documentation]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud
[default_cred_ref]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/azidentity#defaultazurecredential
[examples]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics#pkg-examples
[example_client]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics#example-NewClient
[ingestion_azlogs]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/monitor/ingestion/azlogs
[ingestion_overview]: https://learn.microsoft.com/azure/azure-monitor/logs/logs-ingestion-api-overview
[logs_overview]: https://learn.microsoft.com/azure/azure-monitor/logs/data-platform-logs
[metrics_overview]: https://learn.microsoft.com/azure/azure-monitor/essentials/data-platform-metrics
[monitor_docs]: https://learn.microsoft.com/azure/azure-monitor/
[monitor_rest_docs]: https://learn.microsoft.com/rest/api/monitor/
[multi-metrics]: https://learn.microsoft.com/azure/azure-monitor/essentials/data-platform-metrics#multi-dimensional-metrics
[supported_metrics]: https://learn.microsoft.com/azure/azure-monitor/reference/supported-metrics/metrics-index#metrics-by-resource-provider

[cla]: https://cla.microsoft.com
[coc]: https://opensource.microsoft.com/codeofconduct/
[coc_faq]: https://opensource.microsoft.com/codeofconduct/faq/
[coc_contact]: mailto:opencode@microsoft.com
