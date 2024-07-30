## Go

``` yaml
title: MonitorQueryClient
description: Azure Monitor Query Go Client
generated-metadata: false

clear-output-folder: false
export-clients: true
go: true
input-file: 
    - https://github.com/Azure/azure-rest-api-specs/blob/72427ef3ff5875bd8409ef112ef5e6f3cf2b8795/specification/operationalinsights/data-plane/Microsoft.OperationalInsights/stable/2022-10-27/OperationalInsights.json
    - https://github.com/Azure/azure-rest-api-specs/blob/dba6ed1f03bda88ac6884c0a883246446cc72495/specification/monitor/resource-manager/Microsoft.Insights/stable/2018-01-01/metricDefinitions_API.json
    - https://github.com/Azure/azure-rest-api-specs/blob/dba6ed1f03bda88ac6884c0a883246446cc72495/specification/monitor/resource-manager/Microsoft.Insights/stable/2018-01-01/metrics_API.json
    - https://github.com/Azure/azure-rest-api-specs/blob/dba6ed1f03bda88ac6884c0a883246446cc72495/specification/monitor/resource-manager/Microsoft.Insights/preview/2017-12-01-preview/metricNamespaces_API.json
license-header: MICROSOFT_MIT_NO_VERSION
module: github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery
openapi-type: "data-plane"
output-folder: ../azquery
override-client-name: LogsClient
security: "AADToken"
use: "@autorest/go@4.0.0-preview.46"
version: "^3.0.0"

directive:
  # delete extra endpoints
  - from: swagger-document
    where: $["paths"]
    transform: >
        delete $["/workspaces/{workspaceId}/metadata"];
  - from: swagger-document
    where: $["x-ms-paths"]
    transform: >
        delete $["/{resourceId}/query?disambiguation_dummy"];

  # delete extra operations
  - remove-operation: Metadata_Post
  - remove-operation: Metadata_Get
  - remove-operation: Query_Get
  - remove-operation: Query_ResourceGet
  - remove-operation: Query_ResourceExecuteXms
  - remove-operation: Query_ResourceGetXms

  # delete metadata models
  - remove-model: metadataResults
  - remove-model: metadataCategory
  - remove-model: metadataSolution
  - remove-model: metadataResourceType
  - remove-model: metadataTable
  - remove-model: metadataFunction
  - remove-model: metadataQuery
  - remove-model: metadataApplication
  - remove-model: metadataWorkspace
  - remove-model: metadataResource
  - remove-model: metadataPermissions

 # rename operations to generate into metrics and logs clients
  - rename-operation:
      from: Query_Execute
      to: Logs_QueryWorkspace
  - rename-operation:
      from: Query_Batch
      to: Logs_QueryBatch
  - rename-operation:
      from: Query_ResourceExecute
      to: Logs_QueryResource
  - rename-operation:
      from: Metrics_List
      to: Metrics_QueryResource
  - rename-operation:
      from: MetricDefinitions_List
      to: Metrics_ListDefinitions
  - rename-operation:
      from: MetricNamespaces_List
      to: Metrics_ListNamespaces

  # rename some metrics fields
  - from: swagger-document
    where: $.definitions.Metric.properties.timeseries
    transform: $["x-ms-client-name"] = "TimeSeries"
  - from: swagger-document
    where: $.definitions.TimeSeriesElement.properties.metadatavalues
    transform: $["x-ms-client-name"] = "MetadataValues"
  - from: swagger-document
    where: $.definitions.Response.properties.resourceregion
    transform: $["x-ms-client-name"] = "ResourceRegion"
  - from: swagger-document
    where: $.parameters.MetricNamespaceParameter
    transform: $["x-ms-client-name"] = "MetricNamespace"
  - from: swagger-document
    where: $.parameters.MetricNamesParameter
    transform: $["x-ms-client-name"] = "MetricNames"
  - from: swagger-document
    where: $.parameters.OrderByParameter
    transform: $["x-ms-client-name"] = "OrderBy"

  # rename Body.Workspaces to Body.AdditionalWorkspaces
  - from: swagger-document
    where: $.definitions.queryBody.properties.workspaces
    transform: $["x-ms-client-name"] = "AdditionalWorkspaces"
  
  # rename Render to Visualization
  - from: swagger-document
    where: $.definitions.queryResults.properties.render
    transform: $["x-ms-client-name"] = "Visualization"
  - from: swagger-document
    where: $.definitions.batchQueryResults.properties.render
    transform: $["x-ms-client-name"] = "Visualization"

  # rename BatchQueryRequest.ID to BatchQueryRequest.CorrelationID
  - from: swagger-document
    where: $.definitions.batchQueryRequest.properties.id
    transform: $["x-ms-client-name"] = "CorrelationID"
  - from: swagger-document
    where: $.definitions.batchQueryResponse.properties.id
    transform: $["x-ms-client-name"] = "CorrelationID"

  # rename BatchQueryRequest.Workspace to BatchQueryRequest.WorkspaceID
  - from: swagger-document
    where: $.definitions.batchQueryRequest.properties.workspace
    transform: $["x-ms-client-name"] = "WorkspaceID"
  
  # rename Prefer to Options
  - from: swagger-document
    where: $.parameters.PreferHeaderParameter
    transform: $["x-ms-client-name"] = "Options"
  - from: models.go
    where: $
    transform: return $.replace(/Options \*string/g, "Options *LogsQueryOptions");
  - from: logs_client.go
    where: $
    transform: return $.replace(/\*options\.Options/g, "options.Options.preferHeader()");
  
  # add default values for batch request path and method attributes
  - from: swagger-document
    where: $.definitions.batchQueryRequest.properties.path
    transform: $["x-ms-client-default"] = "/query"
  - from: swagger-document
    where: $.definitions.batchQueryRequest.properties.method
    transform: $["x-ms-client-default"] = "POST"
  - from: swagger-document
    where: $.definitions.batchQueryRequest.properties.path.x-ms-enum
    transform: $["modelAsString"] = true
  - from: swagger-document
    where: $.definitions.batchQueryRequest.properties.path.x-ms-enum
    transform: $["name"] = "BatchQueryRequestPath"
  - from: swagger-document
    where: $.definitions.batchQueryRequest.properties.method.x-ms-enum
    transform: $["modelAsString"] = true
  - from: swagger-document
    where: $.definitions.batchQueryRequest.properties.method.x-ms-enum
    transform: $["name"] = "BatchQueryRequestMethod"

  # add descriptions for models and constants that don't have them
  - from: constants.go
    where: $
    transform: return $.replace(/type ResultType string/, "//ResultType - Reduces the set of data collected. The syntax allowed depends on the operation. See the operation's description for details.\ntype ResultType string");

  # delete unused error models
  - from: models.go
    where: $
    transform: return $.replace(/(?:\/\/.*\s)+type (?:ErrorResponse|ErrorResponseAutoGenerated|ErrorInfo|ErrorDetail).+\{(?:\s.+\s)+\}\s/g, "");
  - from: models_serde.go
    where: $
    transform: return $.replace(/(?:\/\/.*\s)+func \(\w \*?(?:ErrorResponse|ErrorResponseAutoGenerated|ErrorInfo|ErrorDetail)\).*\{\s(?:.+\s)+\}\s/g, "");

  # point the clients to the correct host url
  - from: 
         - logs_client.go
         - metrics_client.go
    where: $
    transform: return $.replace(/host/g, "client.host");
  - from: 
         - logs_client.go
         - metrics_client.go
    where: $
    transform: return $.replace(/internal \*azcore.Client/g, "host string\n internal *azcore.Client");

  # delete generated host url
  - from: constants.go
    where: $
    transform: return $.replace(/const host = "(.*?)"/, "");

  # change Table.Rows from type [][]interface{} to type []Row
  - from: models.go
    where: $
    transform: return $.replace(/Rows \[\]\[\]byte/, "Rows []Row");

  # change type of timespan from *string to *TimeInterval
  - from: models.go
    where: $
    transform: return $.replace(/Timespan \*string/g, "Timespan *TimeInterval");
  - from: metrics_client.go
    where: $
    transform: return $.replace(/reqQP\.Set\(\"timespan\", \*options\.Timespan\)/g, "reqQP.Set(\"timespan\", string(*options.Timespan))");

  # change type of MetricsClientQueryResourceOptions.Aggregation from *string to []*AggregationType
  - from: models.go
    where: $
    transform: return $.replace(/Aggregation \*string/g, "Aggregation []*AggregationType");
  - from: metrics_client.go
    where: $
    transform: return $.replace(/\*options.Aggregation/g, "aggregationTypeToString(options.Aggregation)");
  - from: swagger-document
    where: $.parameters.AggregationsParameter
    transform: $["description"] = "The list of aggregation types to retrieve"