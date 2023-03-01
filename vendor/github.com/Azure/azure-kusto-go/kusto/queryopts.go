package kusto

// queryopts.go holds the varying QueryOption constructors as the list is so long that
// it clogs up the main kusto.go file.

import (
	"time"

	"github.com/Azure/azure-kusto-go/kusto/data/errors"
	"github.com/Azure/azure-kusto-go/kusto/data/value"
)

// requestProperties is a POD used by clients to describe specific needs from the service.
// For more information please look at: https://docs.microsoft.com/en-us/azure/kusto/api/netfx/request-properties
// Not all of the documented options are implemented.
type requestProperties struct {
	Options    map[string]interface{}
	Parameters map[string]string
}

type queryOptions struct {
	requestProperties *requestProperties
}

// TODO(jdoak/daniel): These really need to be tested.  I didn't find that NoTruncation worked, I had to add the
// line in the query itself. NoRequestTimeout I'm not sure has value and I don't know how to test it. According to
// the docs, the server timeout can be set to a max of 1 hour. I'm not sure how that plays with server timeout. Maybe .Net
// was using this on the client side and if so, that is already taken care of. So not sure what this does. So if a customer
// needs these in the future, we should look deeper at these.
/*
// NoRequestTimeout enables setting the request timeout to its maximum value.
func NoRequestTimeout() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options["norequesttimeout"] = true
		return nil
	}
}

// NoTruncation enables suppressing truncation of the query results returned to the caller.
func NoTruncation() QueryOption {
	return func(q *queryOptions) error {
		q.requestProperties.Options["notruncation"] = true
		return nil
	}
}
*/

// ResultsProgressiveDisable disables the progressive query stream.
func ResultsProgressiveDisable() QueryOption {
	return func(q *queryOptions) error {
		delete(q.requestProperties.Options, "results_progressive_enabled")
		return nil
	}
}

// queryServerTimeout is the amount of time the server will allow a query to take.
// NOTE: I have made the serverTimeout private. For the moment, I'm going to use the context.Context timer
// to set timeouts via this private method.
func queryServerTimeout(d time.Duration) QueryOption {
	return func(q *queryOptions) error {
		if d > 1*time.Hour {
			return errors.ES(errors.OpQuery, errors.KClientArgs, "ServerTimeout option was set to %v, but can't be more than 1 hour", d)
		}
		q.requestProperties.Options["servertimeout"] = value.Timespan{Valid: true, Value: d}.Marshal()
		return nil
	}
}

/*
// CustomQueryOption exists to allow a QueryOption that is not defined in the Go SDK, as all options
// are not defined. Please Note: you should always use the type safe options provided below when available.
// Also note that Kusto does not error on non-existent paramater names or bad values, it simply doesn't
// work as expected.
func CustomQueryOption(paramName string, i interface{}) QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options[paramName] = i
	}
}

// BlockSplitting enables splitting of sequence blocks after aggregation operator
func BlockSplitting() QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options["block_splitting_enabled"] = true
	}
}

// DatabasePattern overrides database name and picks the 1st database that matches the pattern. '*' means
// any database that user has access to.
func DatabasePattern(s string) QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options["database_pattern"] = s
	}
}

// DebugQueryExternalDataProjectionFusionDisabled prevents fusing projection into ExternalData operator.
func DebugQueryExternalDataProjectionFusionDisabled() QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options["debug_query_externaldata_projection_fusion_disabled"] = true
	}
}

// DebugQueryFanoutThreadsPercentExternalData sets percentage of threads to fanout execution to for
// external data nodes.
func DebugQueryFanoutThreadsPercentExternalData(i int) QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options["debug_query_fanout_threads_percent_external_data"] = i
	}
}

// DeferPartialQueryFailures disables reporting partial query failures as part of the result set.
func DeferPartialQueryFailures() QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options["deferpartialqueryfailures"] = true
	}
}

// MaxMemoryConsumptionPerQueryPerNode overrides the default maximum amount of memory a whole query
// may allocate per node.
func MaxMemoryConsumptionPerQueryPerNode(i uint64) QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options["max_memory_consumption_per_query_per_node"] = i
	}
}

// MaxMemoryConsumptionPerIterator overrides the default maximum amount of memory a query operator may allocate.
func MaxMemoryConsumptionPerIterator(i uint64) QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options["maxmemoryconsumptionperiterator"] = i
	}
}

// MaxOutputColumns overrides the default maximum number of columns a query is allowed to produce.
func MaxOutputColumns(i int) QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options["maxoutputcolumns"] = i
	}
}

// PushSelectionThroughAggregation will push simple selection through aggregation .
func PushSelectionThroughAggregation() QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options["push_selection_through_aggregation"] = true
	}
}

// AdminSuperSlackerMode delegate execution of the query to another node.
func AdminSuperSlackerMode() QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options["query_admin_super_slacker_mode"] = true
	}
}

// QueryCursorAfterDefault sets the default parameter value of the cursor_after() function when
// called without parameters.
func QueryCursorAfterDefault(s string) QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options["query_cursor_after_default"] = s
	}
}

// QueryCursorAllowReferencingStreamingIngestionTables enable usage of cursor functions over databases which have streaming ingestion enabled.
func QueryCursorAllowReferencingStreamingIngestionTables() QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options["query_cursor_allow_referencing_streaming_ingestion_tables"] = true
	}
}

// QueryCursorBeforeOrAtDefault sets the default parameter value of the cursor_before_or_at() function when called
// without parameters.
func QueryCursorBeforeOrAtDefault(s string) QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options["query_cursor_before_or_at_default"] = s
	}
}

// QueryCursorCurrent overrides the cursor value returned by the cursor_current() or current_cursor() functions.
func QueryCursorCurrent(s string) QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options["query_cursor_current"] = s
	}
}

// QueryCursorScopedTables is a list of table names that should be scoped to cursor_after_default ..
// cursor_before_or_at_default (upper bound is optional).
func QueryCursorScopedTables(l []string) QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options["query_cursor_scoped_tables"] = l
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
		return func(q *queryOptions) {}
	}
	return func(q *queryOptions) {
		q.requestProperties.Options["query_datascope"] = string(ds.(dataScope))
	}
}

// QueryDateTimeScopeColumn controls the column name for the query's datetime scope
// (query_datetimescope_to / query_datetimescope_from)
func QueryDateTimeScopeColumn(s string) QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options["query_datetimescope_column"] = s
	}
}

// QueryDateTimeScopeFrom controls the query's datetime scope (earliest) -- used as auto-applied filter on
// query_datetimescope_column only (if defined).
func QueryDateTimeScopeFrom(t time.Time) QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options["query_datetimescope_from"] = t.Format(time.RFC3339Nano)
	}
}

// QueryDateTimeScopeTo controls the query's datetime scope (latest) -- used as auto-applied filter on
// query_datetimescope_column only (if defined).
func QueryDateTimeScopeTo(t time.Time) QueryOption {
	return func(q *queryOptions) {
		q.requestProperties.Options["query_datetimescope_to"] = t.Format(time.RFC3339Nano)
	}
}
*/

// Options we have not added.
/*
query_bin_auto_at (QueryBinAutoAt): When evaluating the bin_auto() function, the start value to use. [LiteralExpression]
query_bin_auto_size (QueryBinAutoSize): When evaluating the bin_auto() function, the bin size value to use. [LiteralExpression]

query_distribution_nodes_span (OptionQueryDistributionNodesSpanSize): If set, controls the way sub-query merge behaves: the executing node will introduce an additional level in the query hierarchy for each sub-group of nodes; the size of the sub-group is set by this option. [Int]
query_enable_jit_stream (OptionEnableJitStream): If true, enabled JIT streams when sending data from managed code to native code. [Boolean]
query_fanout_nodes_percent (OptionQueryFanoutNodesPercent): The percentage of nodes to fanout execution to. [Int]
query_fanout_threads_percent (OptionQueryFanoutThreadsPercent): The percentage of threads to fanout execution to. [Int]
query_language (OptionQueryLanguage): Controls how the query text is to be interpreted. ['csl','kql' or 'sql']
query_max_entities_in_union (OptionMaxEntitiesToUnion): Overrides the default maximum number of columns a query is allowed to produce. [Long]
query_now (OptionQueryNow): Overrides the datetime value returned by the now(0s) function. [DateTime]
query_results_cache_max_age (OptionQueryResultsCacheMaxAge): If positive, controls the maximum age of the cached query results which Kusto is allowed to return [TimeSpan]
query_results_progressive_row_count (OptionProgressiveQueryMinRowCountPerUpdate): Hint for Kusto as to how many records to send in each update (Takes effect only if OptionProgressiveQueryIsProgressive is set)
query_results_progressive_update_period (OptionProgressiveProgressReportPeriod): Hint for Kusto as to how often to send progress frames (Takes effect only if OptionProgressiveQueryIsProgressive is set)
query_shuffle_broadcast_join (ShuffleBroadcastJoin): Enables shuffling over broadcast join.
query_take_max_records (OptionTakeMaxRecords): Enables limiting query results to this number of records. [Long]
queryconsistency (OptionQueryConsistency): Controls query consistency. ['strongconsistency' or 'normalconsistency' or 'weakconsistency']
request_callout_disabled (OptionRequestCalloutDisabled): If specified, indicates that the request cannot call-out to a user-provided service. [Boolean]
request_external_table_disabled (OptionRequestExternalTableDisabled): If specified, indicates that the request cannot invoke code in the ExternalTable. [Boolean]
request_readonly (OptionRequestReadOnly): If specified, indicates that the request must not be able to write anything. [Boolean]
request_remote_entities_disabled (OptionRequestRemoteEntitiesDisabled): If specified, indicates that the request cannot access remote databases and clusters. [Boolean]
request_sandboxed_execution_disabled (OptionRequestSandboxedExecutionDisabled): If specified, indicates that the request cannot invoke code in the sandbox. [Boolean]
response_dynamic_serialization (OptionResponseDynamicSerialization): Controls the serialization of 'dynamic' values in result sets. ['string', 'json']
response_dynamic_serialization_2 (OptionResponseDynamicSerialization_2): Controls the serialization of 'dynamic' string and null values in result sets. ['legacy', 'current']
results_progressive_enabled (OptionResultsProgressiveEnabled): If set, enables the progressive query stream
truncationmaxrecords (OptionTruncationMaxRecords): Overrides the default maximum number of records a query is allowed to return to the caller (truncation). [Long]
truncationmaxsize (OptionTruncationMaxSize): Overrides the dfefault maximum data size a query is allowed to return to the caller (truncation). [Long]
validate_permissions (OptionValidatePermissions): Validates user's permissions to perform the query and doesn't run the query itself. [Boolean]
*/
