# Troubleshooting Azure Monitor Query client library issues

This troubleshooting guide contains instructions to diagnose frequently encountered issues while using the Azure
Monitor Query client library for Go.

## Table of contents

* [General Troubleshooting](#general-troubleshooting)
    * [Error Handling](#error-handling)
    * [Logging](#logging)
    * [Troubleshooting authentication issues with logs and metrics query requests](#authentication-errors)
* [Troubleshooting Logs Query](#troubleshooting-logs-query)
    * [Troubleshooting authorization errors](#troubleshooting-authorization-errors-for-logs-query)
    * [Troubleshooting invalid Kusto query](#troubleshooting-invalid-kusto-query)
    * [Troubleshooting empty log query results](#troubleshooting-empty-log-query-results)
    * [Troubleshooting server timeouts when executing logs query request](#troubleshooting-server-timeouts-when-executing-logs-query-request)
* [Troubleshooting Metrics Query](#troubleshooting-metrics-query)
    * [Troubleshooting authorization errors](#troubleshooting-authorization-errors-for-metrics-query)
    * [Troubleshooting unsupported granularity for metrics query](#troubleshooting-unsupported-granularity-for-metrics-query)

## General Troubleshooting

### Error Handling

All methods which send HTTP requests return `*azcore.ResponseError` when these requests fail. `ResponseError` has error details and the raw response from Monitor Query.

For Logs, an error may also be returned in the response's `ErrorInfo` struct, usually to indicate a partial error from the service.

### Logging

This module uses the logging implementation in `azcore`. To turn on logging for all Azure SDK modules, set `AZURE_SDK_GO_LOGGING` to `all`. By default, the logger writes to stderr. Use the `azcore/log` package to control log output. For example, logging only HTTP request and response events, and printing them to stdout:

```go
import azlog "github.com/Azure/azure-sdk-for-go/sdk/azcore/log"

// Print log events to stdout
azlog.SetListener(func(cls azlog.Event, msg string) {
	fmt.Println(msg)
})

// Includes only requests and responses in credential logs
azlog.SetEvents(azlog.EventRequest, azlog.EventResponse)
```

### Authentication errors

Azure Monitor Query supports Azure Active Directory authentication. Both LogsClient and
MetricsClient take in a `credential` as a parameter in their constructors. To provide a valid credential, you can use
`azidentity` package. For more details on getting started, refer to
the [README][readme_authentication]
of Azure Monitor Query library. For details on the credential types supported in `azidentity`, see the [Azure Identity library's documentation][azidentity_docs].

For more help with troubleshooting authentication errors, see the Azure Identity client library [troubleshooting guide][azidentity_troubleshooting].

## Troubleshooting Logs Query

### Troubleshooting authorization errors for logs query

If you get an HTTP error with status code 403 (Forbidden), it means that the provided credentials does not have
sufficient permissions to query the workspace.
```text
{"error":{"message":"The provided credentials have insufficient access to perform the requested operation","code":"InsufficientAccessError","correlationId":""}}
```

1. Check that the application or user that is making the request has sufficient permissions:
    * You can refer to this document to [manage access to workspaces][workspace_access]
2. If the user or application is granted sufficient privileges to query the workspace, make sure you are
   authenticating as that user/application. If you are authenticating using the
   [DefaultAzureCredential][default_azure_cred]
   then check the logs to verify that the credential used is the one you expected. To enable logging, see [enable
   client logging](#logging) section above.

For more help with troubleshooting authentication errors, see the Azure Identity client library [troubleshooting guide][azidentity_troubleshooting].

### Troubleshooting invalid Kusto query

If you get an HTTP error with status code 400 (Bad Request), you may have an error in your Kusto query and you'll
see an error message similar to the one below.

```text
{"error":{"message":"The request had some invalid properties","code":"BadArgumentError","correlationId":"","innererror":{"code":"SyntaxError","message":"A recognition error occurred in the query.","innererror":{"code":"SYN0002","message":"Query could not be parsed at 'joi' on line [2,244]","line":2,"pos":244,"token":"joi"}}}}
```

The error message in `innererror` may include the location where the Kusto query has an error plus further details. You may also refer to the [Kusto Query Language][kusto] reference docs to learn more about querying logs using KQL.

### Troubleshooting empty log query results

If your Kusto query returns empty with no logs, please validate the following:

- You have the right workspace ID or resource ID
- You are setting the correct time interval for the query. Try lengthening the time interval for your query to see if that
  returns any results.
- If your Kusto query also has a time interval, the query is evaluated for the intersection of the time interval in the
  query string and the time interval set in the `Body.Timespan` field of the request query. The intersection of
  these time intervals may not have any logs. To avoid any confusion, it's recommended to remove any time interval in
  the Kusto query string and use `Body.Timespan` explicitly.
- Your workspace or resource actually has logs to query. Sometimes, especially with newly created resources,
  there are no logs yet to query.

### Troubleshooting server timeouts when executing logs query request

Some complex Kusto queries can take a long time to complete. These queries are aborted by the service if they run for more than 3 minutes. For such scenarios, the query APIs on LogsClient, provide options to configure the timeout on the server. The server timeout can be extended up to 10 minutes.

You may see an error as follows:

```
Code: GatewayTimeout
Message: Gateway timeout
Inner error: {
    "code": "GatewayTimeout",
    "message": "Unable to unzip response"
}
```

The following code shows an example of setting the server timeout. By setting this server timeout, the Azure Monitor Query library will automatically extend the client timeout to wait for 10 minutes for the server to respond. 

```go
workspaceID := "<workspace_id>"
options := &azquery.LogsClientQueryWorkspaceOptions{
		Options: &azquery.LogsQueryOptions{
			Wait:          to.Ptr(600), // increases wait time to ten minutes
		},
	}

res, err := logsClient.QueryWorkspace(context.Background(), 
             workspaceID, 
             azquery.Body{Query: to.Ptr("AzureActivity
                    | summarize Count = count() by ResourceGroup
                    | top 10 by Count
                    | project ResourceGroup")}, 
             options)
if err != nil {
    //TODO: handle error
}
_ = res
```

## Troubleshooting Metrics Query

### Troubleshooting authorization errors for metrics query

If you get an HTTP error with status code 403 (Forbidden), it means that the provided credentials does not have
sufficient permissions to query the workspace.
```text
{"error":{"code":"AuthorizationFailed","message":"The client '71d56230-5920-4856-8f33-c030b269d870' with object id '71d56230-5920-4856-8f33-c030b269d870' does not have authorization to perform action 'microsoft.insights/metrics/read' over scope '/subscriptions/faa080af-c1d8-40ad-9cce-e1a450ca5b57/resourceGroups/srnagar-azuresdkgroup/providers/Microsoft.CognitiveServices/accounts/srnagara-textanalytics/providers/microsoft.insights' or the scope is invalid. If access was recently granted, please refresh your credentials."}}
```

1. Check that the application or user that is making the request has sufficient permissions.
2. If the user or application is granted sufficient privileges to query the resource, make sure you are
   authenticating as that user/application. If you are authenticating using the
   [DefaultAzureCredential][default_azure_cred]
   then check the logs to verify that the credential used is the one you expected. To enable logging, see [enable
   client logging](#logging) section above.

For more help on troubleshooting authentication errors, please see the Azure Identity client library [troubleshooting
guide][azidentity_troubleshooting]

### Troubleshooting unsupported granularity for metrics query

If you notice the following exception, this is due to an invalid time granularity in the metrics query request. Your
query might look something like the following where `MetricsClientQueryResourceOptions.Interval` is set to an unsupported
duration.

```text
{"code":"BadRequest","message":"Invalid time grain duration: PT10M, supported ones are: 00:01:00,00:05:00,00:15:00,00:30:00,01:00:00,06:00:00,12:00:00,1.00:00:00"}
```

As documented in the error message, the supported granularity for metrics queries are 1 minute, 5 minutes, 15 minutes,
30 minutes, 1 hour, 6 hours, 12 hours and 1 day.

<!-- LINKS -->
[azidentity_docs]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity
[azidentity_troubleshooting]: https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/azidentity/TROUBLESHOOTING.md
[default_azure_cred]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/azidentity#defaultazurecredential
[kusto]: https://learn.microsoft.com/azure/data-explorer/kusto/query
[readme_authentication]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/monitor/azquery#authentication
[workspace_access]: https://learn.microsoft.com/azure/azure-monitor/logs/manage-access#manage-access-using-workspace-permissions
