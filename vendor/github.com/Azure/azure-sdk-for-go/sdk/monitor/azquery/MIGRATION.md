# Guide to migrate from `operationalinsights` and monitor `insights` to `azquery`

This guide is intended to assist in the migration to the `azquery` module. `azquery` allows users to retrieve log and metric data from Azure Monitor.

## Package consolidation

 Azure Monitor allows users to retrieve telemetry data for their Azure resources. The main two data catagories for Azure Monitor are [metrics](https://learn.microsoft.com/azure/azure-monitor/essentials/data-platform-metrics) and [logs](https://learn.microsoft.com/azure/azure-monitor/logs/data-platform-logs). 
 
 There have been a number of [terminology](https://learn.microsoft.com/azure/azure-monitor/terminology) changes for Azure Monitor over the years which resulted in the operations being spread over multiple packages. For Go, metrics methods were contained in `github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/<version-number>/insights` and logs methods resided in `github.com/Azure/azure-sdk-for-go/services/operationalinsights/v1/operationalinsights`.

The new `azquery` module condenses metrics and logs functionality into one package for simpler access. The `azquery` module contains two clients: [LogsClient](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery#LogsClient) and [MetricsClient](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery#MetricsClient).

Transitioning to a single package has resulted in a number of name changes, as detailed below.

### Log name changes

| `operationalinsights`    | `azquery` |
| ----------- | ----------- |
| QueryClient.Execute      | LogsClient.QueryWorkspace     |
| MetadataClient.Get and MetadataClient.Post | N/A |

The `azquery` module does not contain the `MetadataClient`. For that functionality, please use the old [`operationalinsights`](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/services/operationalinsights/v1/operationalinsights) module or [file an issue in our github repo](https://github.com/Azure/azure-sdk-for-go/issues), so we can prioritize adding it to `azquery`.

### Metrics name changes 

| `insights` | `azquery` |
| ----------- | ----------- |
| MetricsClient.List     | MetricsClient.QueryResource       |
| MetricDefinitionsClient.List   | MetricsClient.NewListDefinitionsPager        |
| MetricNamespacesClient.List   | MetricsClient.NewListNamespacesPager        |

## Query Logs

### `operationalinsights`
```go
import (
    "context"

    "github.com/Azure/azure-sdk-for-go/services/operationalinsights/v1/operationalinsights"
    "github.com/Azure/go-autorest/autorest"
)

// create the client
client := operationalinsights.NewQueryClient()
client.Authorizer = autorest.NewAPIKeyAuthorizerWithHeaders(map[string]interface{}{
    "x-api-key": "DEMO_KEY",
})

// execute the query
query := "<kusto query>"
timespan := "2023-12-25/2023-12-26"

res, err := client.Execute(context.TODO(), "DEMO_WORKSPACE", operationalinsights.QueryBody{Query: &query, Timespan: &timespan})
if err != nil {
    //TODO: handle error
}
```

### `azquery`

Compared to previous versions, querying logs with the new `azquery` module is clearer and simpler. There are a number of name changes for clarity, like how the old `Execute` method is now [`QueryWorkspace`](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery#LogsClient.QueryWorkspace). In addition, there is improved time support. Before if a user added a timespan over which to query the request, it had to be a string constructed in the ISO8601 interval format. Users frequently made mistakes when constructing this string. With the new `QueryWorkspace` method, the type of timespan has been changed from a string to a new type named [`TimeInterval`](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery#TimeInterval). `TimeInterval` has a contructor that allows users to take advantage of Go's time package, allowing easier creation.

```go
import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
)

// create the logs client
cred, err := azidentity.NewDefaultAzureCredential(nil)
if err != nil {
    //TODO: handle error
}
client, err := azquery.NewLogsClient(cred, nil)
if err != nil {
    //TODO: handle error
}

// execute the logs query
res, err := client.QueryWorkspace(context.TODO(), workspaceID,
    azquery.Body{
        Query:    to.Ptr("<kusto query>"),
        Timespan: to.Ptr(azquery.NewTimeInterval(time.Date(2022, 12, 25, 0, 0, 0, 0, time.UTC), time.Date(2022, 12, 25, 12, 0, 0, 0, time.UTC))),
    },
    nil)
if err != nil {
    //TODO: handle error
}
if res.Error != nil {
    //TODO: handle partial error
}
```

## Query Metrics

### `insights`

```go
import (
    "context"

    "github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2022-10-01-preview/insights"
    "github.com/Azure/go-autorest/autorest/azure/auth"
)

// create the client
client := insights.NewMetricsClient("<subscriptionID>")
authorizer, err := auth.NewAuthorizerFromCLI()
if err == nil {
    client.Authorizer = authorizer
}

// execute the query
timespan := "2023-12-25/2023-12-26"
interval := "PT1M"
metricnames := ""
aggregation := "Average"
top := 3
orderby := "Average asc"
filter := "BlobType eq '*'"
resultType := insights.ResultTypeData
metricnamespace := "Microsoft.Storage/storageAccounts/blobServices"

res, err := client.List(context.TODO(), resourceURI, timespan, &interval, metricnames, aggregation, &top, orderby, filter, resultType, metricnamespace)
if err != nil {
    //TODO: handle error
}
```

### `azquery`

The main difference between the old and new methods of querying metrics is in the naming. The new method has an updated convention for clarity. For example, the old name of the method was simply `List`. Now, it's `QueryResource`. There have also been a number of casing fixes and the query options have been moved into the options struct.

```go
import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
)

// create the metrics client
cred, err := azidentity.NewDefaultAzureCredential(nil)
if err != nil {
    //TODO: handle error
}
client, err := azquery.NewMetricsClient(cred, nil)
if err != nil {
    //TODO: handle error
}

// execute the metrics query
res, err := metricsClient.QueryResource(context.TODO(), resourceURI,
    &azquery.MetricsClientQueryResourceOptions{
        Timespan:        to.Ptr(azquery.NewTimeInterval(time.Date(2022, 12, 25, 0, 0, 0, 0, time.UTC), time.Date(2022, 12, 25, 12, 0, 0, 0, time.UTC))),
        Interval:        to.Ptr("PT1M"),
        MetricNames:     nil,
        Aggregation:     to.SliceOfPtrs(azquery.AggregationTypeAverage, azquery.AggregationTypeCount),
        Top:             to.Ptr[int32](3),
        OrderBy:         to.Ptr("Average asc"),
        Filter:          to.Ptr("BlobType eq '*'"),
        ResultType:      nil,
        MetricNamespace: to.Ptr("Microsoft.Storage/storageAccounts/blobServices"),
    })
if err != nil {
    //TODO: handle error
}
```



