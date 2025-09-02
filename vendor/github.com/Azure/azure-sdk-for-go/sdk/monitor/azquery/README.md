# Azure Monitor Query client module for Go

> Use the [azmetrics][azmetrics] package for access to data plane metrics

The Azure Monitor Query client module is used to execute read-only queries against [Azure Monitor][azure_monitor_overview]'s two data platforms:

- [Logs][logs_overview] - Collects and organizes log and performance data from monitored resources. Data from different sources such as platform logs from Azure services, log and performance data from virtual machines agents, and usage and performance data from apps can be consolidated into a single [Azure Log Analytics workspace][log_analytics_workspace]. The various data types can be analyzed together using the [Kusto Query Language][kusto_query_language]. See the [Kusto to SQL cheat sheet][kusto_to_sql] for more information.
- [Metrics][metrics_overview] - Collects numeric data from monitored resources into a time series database. Metrics are numerical values that are collected at regular intervals and describe some aspect of a system at a particular time. Metrics are lightweight and capable of supporting near real-time scenarios, making them particularly useful for alerting and fast detection of issues.

[Source code][azquery_repo] | [Package (pkg.go.dev)][azquery_pkg_go] | [REST API documentation][monitor_rest_docs] | [Product documentation][monitor_docs] | [Samples][azquery_pkg_go_samples]

## Getting started

### Prerequisites

* [Supported](https://aka.ms/azsdk/go/supported-versions) version of Go - [Install Go](https://go.dev/doc/install)
* Azure subscription - [Create a free account][azure_sub]
* To query Logs, you need one of the following things:
  * An [Azure Log Analytics workspace][log_analytics_workspace_create]
  * The resource URI of an Azure resource (Storage Account, Key Vault, Cosmos DB, etc.)
* To query Metrics, the resource URI of an Azure resource (Storage Account, Key Vault, CosmosDB, etc.) that you plan to monitor

### Install the packages

Install the `azquery` and `azidentity` modules with `go get`:

```bash
go get github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery
go get github.com/Azure/azure-sdk-for-go/sdk/azidentity
```

The [azidentity][azure_identity] module is used for Azure Active Directory authentication during client construction.

### Authentication

An authenticated client object is required to execute a query. The examples demonstrate using [azidentity.NewDefaultAzureCredential][default_cred_ref] to authenticate; however, the client accepts any [azidentity][azure_identity] credential. See the [azidentity][azure_identity] documentation for more information about other credential types.

The clients default to the Azure public cloud. For other cloud configurations, see the [cloud][cloud_documentation] package documentation.

#### Create a logs client

Example [logs client][example_logs_client]

#### Create a metrics client

Example [metrics client][example_metrics_client]

## Key concepts

### Timespan

It's best practice to always query with a timespan (type `TimeInterval`) to prevent excessive queries of the entire logs or metrics data set. Log queries use the ISO8601 Time Interval Standard. All time should be represented in UTC. If the timespan is included in both the Kusto query string and `Timespan` field, the timespan is the intersection of the two values.

Use the `NewTimeInterval()` method for easy creation.

### Metrics data structure

Each set of metric values is a time series with the following characteristics:

- The time the value was collected
- The resource associated with the value
- A namespace that acts like a category for the metric
- A metric name
- The value itself
- Some metrics may have multiple dimensions as described in [multi-dimensional metrics][multi-metrics]. Custom metrics can have up to 10 dimensions.

### Logs query rate limits and throttling

The Log Analytics service applies throttling when the request rate is too high. Limits, such as the maximum number of rows returned, are also applied on the Kusto queries. For more information, see [Query API][service_limits].

If you're executing a batch logs query, a throttled request will return a `ErrorInfo` object. That object's `code` value will be `ThrottledError`.

### Advanced logs queries

#### Query multiple workspaces

To run the same query against multiple Log Analytics workspaces, add the additional workspace ID strings to the `AdditionalWorkspaces` slice in the `Body` struct. 

When multiple workspaces are included in the query, the logs in the result table are not grouped according to the workspace from which they were retrieved.

#### Increase wait time, include statistics, include render (visualization)

The `LogsQueryOptions` type is used for advanced logs options.

* By default, your query will run for up to three minutes. To increase the default timeout, set `LogsQueryOptions.Wait` to the desired number of seconds. The maximum wait time is 10 minutes (600 seconds).

* To get logs query execution statistics, such as CPU and memory consumption, set `LogsQueryOptions.Statistics` to `true`.

* To get visualization data for logs queries, set `LogsQueryOptions.Visualization` to `true`.

```go
azquery.LogsClientQueryWorkspaceOptions{
			Options: &azquery.LogsQueryOptions{
				Statistics:    to.Ptr(true),
				Visualization: to.Ptr(true),
				Wait:          to.Ptr(600),
			},
		}
```

To do the same with `QueryBatch`, set the values in the `BatchQueryRequest.Headers` map with a key of "prefer", or use the `NewBatchQueryRequest` method.

## Examples

Get started with our [examples][azquery_pkg_go_samples].

* For the majority of log queries, use the `LogsClient.QueryWorkspace` or the `LogsClient.QueryResource` method. Only use the `LogsClient.QueryBatch` method in advanced scenerios.

* Use `MetricsClient.QueryResource` for metric queries.

## Troubleshooting

See our [troubleshooting guide][troubleshooting_guide] for details on how to diagnose various failure scenarios.

## Contributing

This project welcomes contributions and suggestions. Most contributions require you to agree to a [Contributor License Agreement (CLA)][cla] declaring that you have the right to, and actually do, grant us the rights to use your contribution.

When you submit a pull request, a CLA-bot will automatically determine whether you need to provide a CLA and decorate
the PR appropriately (e.g., label, comment). Simply follow the instructions provided by the bot. You will only need to
do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct][coc]. For more information, see
the [Code of Conduct FAQ][coc_faq] or contact [opencode@microsoft.com][coc_contact] with any additional questions or
comments.

<!-- LINKS -->
[azmetrics]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics
[azquery_repo]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/monitor/azquery
[azquery_pkg_go]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery
[azquery_pkg_go_docs]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery#section-documentation
[azquery_pkg_go_samples]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery#pkg-examples
[azure_identity]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity
[azure_sub]: https://azure.microsoft.com/free/
[azure_monitor_overview]: https://learn.microsoft.com/azure/azure-monitor/overview
[context]: https://pkg.go.dev/context
[cloud_documentation]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud
[default_cred_ref]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/azidentity#defaultazurecredential
[example_logs_client]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery#example-NewLogsClient
[example_metrics_client]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery#example-NewMetricsClient
[go_samples]: (https://github.com/Azure-Samples/azure-sdk-for-go-samples)
[kusto_query_language]: https://learn.microsoft.com/azure/data-explorer/kusto/query/
[kusto_to_sql]: https://learn.microsoft.com/azure/data-explorer/kusto/query/sqlcheatsheet
[log_analytics_workspace]: https://learn.microsoft.com/azure/azure-monitor/logs/log-analytics-workspace-overview
[log_analytics_workspace_create]: https://learn.microsoft.com/azure/azure-monitor/logs/quick-create-workspace
[logs_overview]: https://learn.microsoft.com/azure/azure-monitor/logs/data-platform-logs
[metrics_overview]: https://learn.microsoft.com/azure/azure-monitor/essentials/data-platform-metrics
[metric_namespaces]: https://learn.microsoft.com/azure/azure-monitor/reference/supported-metrics/metrics-index#metrics-by-resource-provider
[monitor_docs]: https://learn.microsoft.com/azure/azure-monitor/
[monitor_rest_docs]: https://learn.microsoft.com/rest/api/monitor/
[multi-metrics]: https://learn.microsoft.com/azure/azure-monitor/essentials/data-platform-metrics#multi-dimensional-metrics
[service_limits]: https://learn.microsoft.com/azure/azure-monitor/service-limits#la-query-api
[troubleshooting_guide]: https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/monitor/azquery/TROUBLESHOOTING.md
[cla]: https://cla.microsoft.com
[coc]: https://opensource.microsoft.com/codeofconduct/
[coc_faq]: https://opensource.microsoft.com/codeofconduct/faq/
[coc_contact]: mailto:opencode@microsoft.com
