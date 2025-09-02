# Release History

## 1.2.0 (2025-07-17)

### Other Changes
* Upgraded dependencies

## 1.2.0-beta.2 (2025-05-06)

### Features Added
* Added API Version support for `MetricsClient`. Users can now change the default API Version by setting `MetricsClientOptions.APIVersion`.
* Added `AutoAdjustTimegrain`, `RollUpBy`, and `ValidateDimensions` options to `MetricsClient.QueryResource`

### Breaking Changes
* Removed `MetricsBatchClient`. For data plane metrics, including batch metrics, see the azmetrics module (https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics).

### Other Changes
* Upgrade ARM metrics API version to `2024-02-01`

## 1.2.0-beta.1 (2023-11-16)

### Features Added
* Added `MetricsBatchClient` to support batch querying metrics from Azure resources

### Other Changes
* Enabled spans for distributed tracing.

## 1.1.0 (2023-05-09)

### Other Changes
* Updated doc comments

## 1.1.0-beta.1 (2023-04-11)

### Features Added
* Added the `LogsClient.QueryResource` method which allow users to query Azure resources directly without a Log Analytics workspace

### Other Changes
* Updated dependencies and documentation

## 1.0.0 (2023-02-08)

### Breaking Changes
* Removed `LogsQueryOptions.String()`
* Fix casing on some metrics fields

### Other Changes
* Doc and example updates

## 0.4.0 (2023-01-12)

### Features Added
* Added `TimeInterval` type with constructor to aid with timespan creation
* Added `NewBatchQueryRequest` constructor to aid with logs batch requests
* Added `LogsQueryOptions` model for easier setting of logs options

### Breaking Changes
* Changed type of `Body.Timespan`, `MetricsClientQueryResourceOptions.Timespan`, `Response.Timespan` from *string to *TimeInterval
* Remove `ColumnIndexLookup` field from Table struct
* Renamed `Body.Workspaces` to `Body.AdditionalWorkspaces`
* Renamed `Results.Render` and `BatchResponse.Render` to `Results.Visualization` and `BatchResponse.Visualization`

### Other Changes
* Doc and example updates

## 0.3.0 (2022-11-08)

### Features Added
* Added `ColumnIndexLookup` field to Table struct
* Added type `Row`
* Added sovereign cloud support

### Breaking Changes
* Added error return values to `NewLogsClient` and `NewMetricsClient`
* Rename `Batch` to `QueryBatch`
* Rename `NewListMetricDefinitionsPager` to `NewListDefinitionsPager`
* Rename `NewListMetricNamespacesPager` to `NewListNamespacesPager`
* Changed type of `Render` and `Statistics` from interface{} to []byte

### Other Changes
* Updated docs with more detailed examples

## 0.2.0 (2022-10-11)

### Breaking Changes
* Changed format of logs `ErrorInfo` struct to custom error type

## 0.1.0 (2022-09-08)
* This is the initial release of the `azquery` library
