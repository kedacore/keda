package kusto

// queryopts.go holds the varying QueryOption constructors as the list is so long that
// it clogs up the main kusto.go file.

import (
	"github.com/Azure/azure-kusto-go/kusto/kql"
	"time"

	"github.com/Azure/azure-kusto-go/kusto/data/value"
)

// requestProperties is a POD used by clients to describe specific needs from the service.
// For more information please look at: https://docs.microsoft.com/en-us/azure/kusto/api/netfx/request-properties
// Not all of the documented options are implemented.
type requestProperties struct {
	Options         map[string]interface{}
	Parameters      map[string]string
	Application     string         `json:"-"`
	User            string         `json:"-"`
	QueryParameters kql.Parameters `json:"-"`
	ClientRequestID string         `json:"-"`
}

type queryOptions struct {
	requestProperties *requestProperties
	queryIngestion    bool
}

const ResultsProgressiveEnabledValue = "results_progressive_enabled"
const NoRequestTimeoutValue = "norequesttimeout"
const NoTruncationValue = "notruncation"
const ServerTimeoutValue = "servertimeout"
const DeferPartialQueryFailuresValue = "deferpartialqueryfailures"
const MaxMemoryConsumptionPerQueryPerNodeValue = "max_memory_consumption_per_query_per_node"
const MaxMemoryConsumptionPerIteratorValue = "maxmemoryconsumptionperiterator"
const MaxOutputColumnsValue = "maxoutputcolumns"
const PushSelectionThroughAggregationValue = "push_selection_through_aggregation"
const QueryCursorAfterDefaultValue = "query_cursor_after_default"
const QueryCursorBeforeOrAtDefaultValue = "query_cursor_before_or_at_default"
const QueryCursorCurrentValue = "query_cursor_current"
const QueryCursorDisabledValue = "query_cursor_disabled"
const QueryCursorScopedTablesValue = "query_cursor_scoped_tables"
const QueryDatascopeValue = "query_datascope"
const QueryDateTimeScopeColumnValue = "query_datetimescope_column"
const QueryDateTimeScopeFromValue = "query_datetimescope_from"
const QueryDateTimeScopeToValue = "query_datetimescope_to"
const ClientMaxRedirectCountValue = "client_max_redirect_count"
const MaterializedViewShuffleValue = "materialized_view_shuffle"
const QueryBinAutoAtValue = "query_bin_auto_at"
const QueryBinAutoSizeValue = "query_bin_auto_size"
const QueryDistributionNodesSpanValue = "query_distribution_nodes_span"
const QueryFanoutNodesPercentValue = "query_fanout_nodes_percent"
const QueryFanoutThreadsPercentValue = "query_fanout_threads_percent"
const QueryForceRowLevelSecurityValue = "query_force_row_level_security"
const QueryLanguageValue = "query_language"
const QueryLogQueryParametersValue = "query_log_query_parameters"
const QueryMaxEntitiesInUnionValue = "query_max_entities_in_union"
const QueryNowValue = "query_now"
const QueryPythonDebugValue = "query_python_debug"
const QueryResultsApplyGetschemaValue = "query_results_apply_getschema"
const QueryResultsCacheMaxAgeValue = "query_results_cache_max_age"
const QueryResultsCachePerShardValue = "query_results_cache_per_shard"
const QueryResultsProgressiveRowCountValue = "query_results_progressive_row_count"
const QueryResultsProgressiveUpdatePeriodValue = "query_results_progressive_update_period"
const QueryTakeMaxRecordsValue = "query_take_max_records"
const QueryConsistencyValue = "queryconsistency"
const RequestAppNameValue = "request_app_name"
const RequestBlockRowLevelSecurityValue = "request_block_row_level_security"
const RequestCalloutDisabledValue = "request_callout_disabled"
const RequestDescriptionValue = "request_description"
const RequestExternalTableDisabledValue = "request_external_table_disabled"
const RequestImpersonationDisabledValue = "request_impersonation_disabled"
const RequestReadonlyValue = "request_readonly"
const RequestRemoteEntitiesDisabledValue = "request_remote_entities_disabled"
const RequestSandboxedExecutionDisabledValue = "request_sandboxed_execution_disabled"
const RequestUserValue = "request_user"
const TruncationMaxRecordsValue = "truncation_max_records"
const TruncationMaxSizeValue = "truncation_max_size"
const ValidatePermissionsValue = "validate_permissions"

// ClientRequestID sets the x-ms-client-request-id header, and can be used to identify the request in the `.show queries` output.
func ClientRequestID(clientRequestID string) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.ClientRequestID = clientRequestID
		return nil
	}
}

// Application sets the x-ms-app header, and can be used to identify the application making the request in the `.show queries` output.
func Application(appName string) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Application = appName
		return nil
	}
}

// QueryParameters sets the parameters to be used in the query.
func QueryParameters(queryParameters *kql.Parameters) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.QueryParameters = *queryParameters
		q.requestProperties.Parameters = queryParameters.ToParameterCollection()
		return nil
	}
}

// User sets the x-ms-user header, and can be used to identify the user making the request in the `.show queries` output.
func User(userName string) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.User = userName
		return nil
	}
}

// NoRequestTimeout enables setting the request timeout to its maximum value.
func NoRequestTimeout() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[NoRequestTimeoutValue] = true
		return nil
	}
}

// NoTruncation enables suppressing truncation of the query results returned to the caller.
func NoTruncation() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[NoTruncationValue] = true
		return nil
	}
}

// ResultsProgressiveEnabled enables the progressive query stream.
func ResultsProgressiveEnabled() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[ResultsProgressiveEnabledValue] = true
		return nil
	}
}

// ServerTimeout overrides the default request timeout.
func ServerTimeout(d time.Duration) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[ServerTimeoutValue] = value.Timespan{Valid: true, Value: d}.Marshal()
		return nil
	}
}

// CustomQueryOption exists to allow a QueryOption that is not defined in the Go SDK, as all options
// are not defined. Please Note: you should always use the type safe options provided below when available.
// Also note that Kusto does not error on non-existent parameter names or bad values, it simply doesn't
// work as expected.
func CustomQueryOption(paramName string, i interface{}) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[paramName] = i
		return nil
	}
}

// DeferPartialQueryFailures disables reporting partial query failures as part of the result set.
func DeferPartialQueryFailures() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[DeferPartialQueryFailuresValue] = true
		return nil
	}
}

// MaxMemoryConsumptionPerQueryPerNode overrides the default maximum amount of memory a whole query
// may allocate per node.
func MaxMemoryConsumptionPerQueryPerNode(i uint64) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[MaxMemoryConsumptionPerQueryPerNodeValue] = i
		return nil
	}
}

// MaxMemoryConsumptionPerIterator overrides the default maximum amount of memory a query operator may allocate.
func MaxMemoryConsumptionPerIterator(i uint64) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[MaxMemoryConsumptionPerIteratorValue] = i
		return nil
	}
}

// MaxOutputColumns overrides the default maximum number of columns a query is allowed to produce.
func MaxOutputColumns(i int) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[MaxOutputColumnsValue] = i
		return nil
	}
}

// PushSelectionThroughAggregation will push simple selection through aggregation .
func PushSelectionThroughAggregation() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[PushSelectionThroughAggregationValue] = true
		return nil
	}
}

// QueryCursorAfterDefault sets the default parameter value of the cursor_after() function when
// called without parameters.
func QueryCursorAfterDefault(s string) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryCursorAfterDefaultValue] = s
		return nil
	}
}

// QueryCursorBeforeOrAtDefault sets the default parameter value of the cursor_before_or_at() function when called
// without parameters.
func QueryCursorBeforeOrAtDefault(s string) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryCursorBeforeOrAtDefaultValue] = s
		return nil
	}
}

// QueryCursorCurrent overrides the cursor value returned by the cursor_current() or current_cursor() functions.
func QueryCursorCurrent(s string) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryCursorCurrentValue] = s
		return nil
	}
}

// QueryCursorDisabled overrides the cursor value returned by the cursor_current() or current_cursor() functions.
func QueryCursorDisabled(s string) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryCursorDisabledValue] = s
		return nil
	}
}

// QueryCursorScopedTables is a list of table names that should be scoped to cursor_after_default ..
// cursor_before_or_at_default (upper bound is optional).
func QueryCursorScopedTables(l []string) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryCursorScopedTablesValue] = l
		return nil
	}
}

// DataScope is used with QueryDataScope() to control a query's datascope.
type DataScope interface {
	isDataScope()
}

type dataScope string

func (dataScope) isDataScope() {}

const (
	// DSDefault is used to set a query's datascope to default.
	DSDefault dataScope = "default"
	// DSAll is used to set a query's datascope to all.
	DSAll dataScope = "all"
	// DSHotCache is used to set a query's datascope to hotcache.
	DSHotCache dataScope = "hotcache"
)

// QueryDataScope controls the query's datascope -- whether the query applies to all data or
// just part of it. ['default', 'all', or 'hotcache']
func QueryDataScope(ds DataScope) QueryOption {
	if ds == nil {
		return func(q *queryOptions) error {
			return nil
		}
	}
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryDatascopeValue] = string(ds.(dataScope))
		return nil
	}
}

// QueryDateTimeScopeColumn controls the column name for the query's datetime scope
// (query_datetimescope_to / query_datetimescope_from)
func QueryDateTimeScopeColumn(s string) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryDateTimeScopeColumnValue] = s
		return nil
	}
}

// QueryDateTimeScopeFrom controls the query's datetime scope (earliest) -- used as auto-applied filter on
// query_datetimescope_column only (if defined).
func QueryDateTimeScopeFrom(t time.Time) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryDateTimeScopeFromValue] = t.Format(time.RFC3339Nano)
		return nil
	}
}

// QueryDateTimeScopeTo controls the query's datetime scope (latest) -- used as auto-applied filter on
// query_datetimescope_column only (if defined).
func QueryDateTimeScopeTo(t time.Time) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryDateTimeScopeToValue] = t.Format(time.RFC3339Nano)
		return nil
	}
}

// ClientMaxRedirectCount If set and positive, indicates the maximum number of HTTP redirects that the client will process.
func ClientMaxRedirectCount(i int64) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[ClientMaxRedirectCountValue] = i
		return nil
	}
}

// MaterializedViewShuffle A hint to use shuffle strategy for materialized views that are referenced in the query.
// The property is an array of materialized views names and the shuffle keys to use.
// Examples: 'dynamic([ { "Name": "V1", "Keys" : [ "K1", "K2" ] } ])' (shuffle view V1 by K1, K2) or 'dynamic([ { "Name": "V1" } ])' (shuffle view V1 by all keys)
func MaterializedViewShuffle(s string) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[MaterializedViewShuffleValue] = s
		return nil
	}
}

// QueryBinAutoAt When evaluating the bin_auto() function, the start value to use.
func QueryBinAutoAt(s string) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryBinAutoAtValue] = s
		return nil
	}
}

// QueryBinAutoSize When evaluating the bin_auto() function, the bin size value to use.
func QueryBinAutoSize(s string) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryBinAutoSizeValue] = s
		return nil
	}
}

// QueryDistributionNodesSpan If set, controls the way the subquery merge behaves: the executing node will introduce an additional
// level in the query hierarchy for each subgroup of nodes; the size of the subgroup is set by this option.
func QueryDistributionNodesSpan(i int64) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryDistributionNodesSpanValue] = i
		return nil
	}
}

// QueryFanoutNodesPercent The percentage of nodes to fan out execution to.
func QueryFanoutNodesPercent(i int) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryFanoutNodesPercentValue] = i
		return nil
	}
}

// QueryFanoutThreadsPercent The percentage of threads to fan out execution to.
func QueryFanoutThreadsPercent(i int) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryFanoutThreadsPercentValue] = i
		return nil
	}
}

// QueryForceRowLevelSecurity If specified, forces Row Level Security rules, even if row_level_security policy is disabled
func QueryForceRowLevelSecurity() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryForceRowLevelSecurityValue] = true
		return nil
	}
}

// QueryLanguage Controls how the query text is to be interpreted (Kql or Sql).
func QueryLanguage(s string) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryLanguageValue] = s
		return nil
	}
}

// QueryLogQueryParameters Enables logging of the query parameters, so that they can be viewed later in the .show queries journal.
func QueryLogQueryParameters() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryLogQueryParametersValue] = true
		return nil
	}
}

// QueryMaxEntitiesInUnion Overrides the default maximum number of entities in a union.
func QueryMaxEntitiesInUnion(i int64) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryMaxEntitiesInUnionValue] = i
		return nil
	}
}

// QueryNow Overrides the datetime value returned by the now(0s) function.
func QueryNow(t time.Time) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryNowValue] = t.Format(time.RFC3339Nano)
		return nil
	}
}

// QueryPythonDebug If set, generate python debug query for the enumerated python node (default first).
func QueryPythonDebug(i int) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryPythonDebugValue] = i
		return nil
	}
}

// QueryResultsApplyGetschema If set, retrieves the schema of each tabular data in the results of the query instead of the data itself.
func QueryResultsApplyGetschema() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryResultsApplyGetschemaValue] = true
		return nil
	}
}

// QueryResultsCacheMaxAge If positive, controls the maximum age of the cached query results the service is allowed to return
func QueryResultsCacheMaxAge(d time.Duration) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryResultsCacheMaxAgeValue] = value.Timespan{Value: d, Valid: true}.Marshal()
		return nil
	}
}

// QueryResultsCachePerShard If set, enables per-shard query cache.
func QueryResultsCachePerShard() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryResultsCachePerShardValue] = true
		return nil
	}
}

// QueryResultsProgressiveRowCount Hint for Kusto as to how many records to send in each update (takes effect only if OptionResultsProgressiveEnabled is set)
func QueryResultsProgressiveRowCount(i int64) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryResultsProgressiveRowCountValue] = i
		return nil
	}
}

// QueryResultsProgressiveUpdatePeriod Hint for Kusto as to how often to send progress frames (takes effect only if OptionResultsProgressiveEnabled is set)
func QueryResultsProgressiveUpdatePeriod(i int32) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryResultsProgressiveUpdatePeriodValue] = i
		return nil
	}
}

// QueryTakeMaxRecords Enables limiting query results to this number of records.
func QueryTakeMaxRecords(i int64) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryTakeMaxRecordsValue] = i
		return nil
	}
}

// QueryConsistency Controls query consistency
func QueryConsistency(c string) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[QueryConsistencyValue] = c
		return nil
	}
}

// RequestAppName Request application name to be used in the reporting (e.g. show queries).
// Does not set the `Application` property in `.show queries`, see `Application` for that.
func RequestAppName(s string) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[RequestAppNameValue] = s
		return nil
	}
}

// RequestBlockRowLevelSecurity If specified, blocks access to tables for which row_level_security policy is enabled.
func RequestBlockRowLevelSecurity() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[RequestBlockRowLevelSecurityValue] = true
		return nil
	}
}

// RequestCalloutDisabled If specified, indicates that the request can't call-out to a user-provided service.
func RequestCalloutDisabled() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[RequestCalloutDisabledValue] = true
		return nil
	}
}

// RequestDescription Arbitrary text that the author of the request wants to include as the request description.
func RequestDescription(s string) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[RequestDescriptionValue] = s
		return nil
	}
}

// RequestExternalTableDisabled If specified, indicates that the request can't invoke code in the ExternalTable.
func RequestExternalTableDisabled() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[RequestExternalTableDisabledValue] = true
		return nil
	}
}

// RequestImpersonationDisabled If specified, indicates that the service should not impersonate the caller's identity.
func RequestImpersonationDisabled() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[RequestImpersonationDisabledValue] = true
		return nil
	}
}

// RequestReadonly If specified, indicates that the request can't write anything.
func RequestReadonly() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[RequestReadonlyValue] = true
		return nil
	}
}

// RequestRemoteEntitiesDisabled If specified, indicates that the request can't access remote databases and clusters.
func RequestRemoteEntitiesDisabled() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[RequestRemoteEntitiesDisabledValue] = true
		return nil
	}
}

// RequestSandboxedExecutionDisabled If specified, indicates that the request can't invoke code in the sandbox.
func RequestSandboxedExecutionDisabled() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[RequestSandboxedExecutionDisabledValue] = true
		return nil
	}
}

// RequestUser Request user to be used in the reporting (e.g. show queries).
// Does not set the `User` property in `.show queries`, see `User` for that.
func RequestUser(s string) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[RequestUserValue] = s
		return nil
	}
}

// TruncationMaxRecords Overrides the default maximum number of records a query is allowed to return to the caller (truncation).
func TruncationMaxRecords(i int64) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[TruncationMaxRecordsValue] = i
		return nil
	}
}

// TruncationMaxSize Overrides the default maximum data size a query is allowed to return to the caller (truncation).
func TruncationMaxSize(i int64) QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[TruncationMaxSizeValue] = i
		return nil
	}
}

// ValidatePermissions Validates user's permissions to perform the query and doesn't run the query itself.
func ValidatePermissions() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options[ValidatePermissionsValue] = true
		return nil
	}
}

// IngestionEndpoint will instruct the Mgmt call to connect to the ingest-[endpoint] instead of [endpoint].
// This is not often used by end users and can only be used with a Mgmt() call.
func IngestionEndpoint() QueryOption {
	return func(m *queryOptions) error {
		m.queryIngestion = true
		return nil
	}
}
