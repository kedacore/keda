// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package internal

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync/atomic"
	"time"

	"go.temporal.io/api/cloud/cloudservice/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/metrics"
	ilog "go.temporal.io/sdk/internal/log"
	"go.temporal.io/sdk/log"
)

const (
	// DefaultNamespace is the namespace name which is used if not passed with options.
	//
	// Exposed as: [go.temporal.io/sdk/client.DefaultNamespace]
	DefaultNamespace = "default"

	// QueryTypeStackTrace is the build in query type for Client.QueryWorkflow() call. Use this query type to get the call
	// stack of the workflow. The result will be a string encoded in the EncodedValue.
	//
	// Exposed as: [go.temporal.io/sdk/client.QueryTypeStackTrace]
	QueryTypeStackTrace string = "__stack_trace"

	// QueryTypeOpenSessions is the build in query type for Client.QueryWorkflow() call. Use this query type to get all open
	// sessions in the workflow. The result will be a list of SessionInfo encoded in the EncodedValue.
	//
	// Exposed as: [go.temporal.io/sdk/client.QueryTypeOpenSessions]
	QueryTypeOpenSessions string = "__open_sessions"

	// QueryTypeWorkflowMetadata is the query name for the workflow metadata.
	QueryTypeWorkflowMetadata string = "__temporal_workflow_metadata"
)

type (
	// Client is the client for starting and getting information about a workflow executions as well as
	// completing activities asynchronously.
	Client interface {
		// ExecuteWorkflow starts a workflow execution and return a WorkflowRun instance and error
		// The user can use this to start using a function or workflow type name.
		// Either by
		//     ExecuteWorkflow(ctx, options, "workflowTypeName", arg1, arg2, arg3)
		//     or
		//     ExecuteWorkflow(ctx, options, workflowExecuteFn, arg1, arg2, arg3)
		// The errors it can return:
		//  - serviceerror.NamespaceNotFound, if namespace does not exist
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//
		// The current timeout resolution implementation is in seconds and uses math.Ceil(d.Seconds()) as the duration. But is
		// subjected to change in the future.
		//
		// WorkflowRun has three methods:
		//  - GetID() string: which return workflow ID (which is same as StartWorkflowOptions.ID if provided)
		//  - GetRunID() string: which return the first started workflow run ID (please see below)
		//  - Get(ctx context.Context, valuePtr interface{}) error: which will fill the workflow
		//    execution result to valuePtr, if workflow execution is a success, or return corresponding
		//    error. This is a blocking API.
		// NOTE: if the started workflow return ContinueAsNewError during the workflow execution, the
		// return result of GetRunID() will be the started workflow run ID, not the new run ID caused by ContinueAsNewError,
		// however, Get(ctx context.Context, valuePtr interface{}) will return result from the run which did not return ContinueAsNewError.
		// Say ExecuteWorkflow started a workflow, in its first run, has run ID "run ID 1", and returned ContinueAsNewError,
		// the second run has run ID "run ID 2" and return some result other than ContinueAsNewError:
		// GetRunID() will always return "run ID 1" and  Get(ctx context.Context, valuePtr interface{}) will return the result of second run.
		// NOTE: DO NOT USE THIS API INSIDE A WORKFLOW, USE workflow.ExecuteChildWorkflow instead
		ExecuteWorkflow(ctx context.Context, options StartWorkflowOptions, workflow interface{}, args ...interface{}) (WorkflowRun, error)

		// GetWorkflow retrieves a workflow execution and return a WorkflowRun instance
		//  - workflow ID of the workflow.
		//  - runID can be default(empty string). if empty string then it will pick the last running execution of that workflow ID.
		//
		// WorkflowRun has three methods:
		//  - GetID() string: which return workflow ID (which is same as StartWorkflowOptions.ID if provided)
		//  - GetRunID() string: which return the first started workflow run ID (please see below)
		//  - Get(ctx context.Context, valuePtr interface{}) error: which will fill the workflow
		//    execution result to valuePtr, if workflow execution is a success, or return corresponding
		//    error. This is a blocking API.
		// NOTE: if the retrieved workflow returned ContinueAsNewError during the workflow execution, the
		// return result of GetRunID() will be the retrieved workflow run ID, not the new run ID caused by ContinueAsNewError,
		// however, Get(ctx context.Context, valuePtr interface{}) will return result from the run which did not return ContinueAsNewError.
		GetWorkflow(ctx context.Context, workflowID string, runID string) WorkflowRun

		// SignalWorkflow sends a signals to a workflow in execution
		//  - workflow ID of the workflow.
		//  - runID can be default(empty string). if empty string then it will pick the running execution of that workflow ID.
		//  - signalName name to identify the signal.
		// The errors it can return:
		//  - serviceerror.NotFound
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		SignalWorkflow(ctx context.Context, workflowID string, runID string, signalName string, arg interface{}) error

		// SignalWithStartWorkflow sends a signal to a running workflow.
		// If the workflow is not running or not found, it starts the workflow and then sends the signal in transaction.
		//  - workflowID, signalName, signalArg are same as SignalWorkflow's parameters
		//  - options, workflow, workflowArgs are same as StartWorkflow's parameters
		//  - the workflowID parameter is used instead of options.ID. If the latter is present, it must match the workflowID.
		// Note: options.WorkflowIDReusePolicy is default to AllowDuplicate.
		// The errors it can return:
		//  - serviceerror.NotFound
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		SignalWithStartWorkflow(ctx context.Context, workflowID string, signalName string, signalArg interface{},
			options StartWorkflowOptions, workflow interface{}, workflowArgs ...interface{}) (WorkflowRun, error)

		// NewWithStartWorkflowOperation returns a WithStartWorkflowOperation for use in UpdateWithStartWorkflow.
		// NOTE: Experimental
		NewWithStartWorkflowOperation(options StartWorkflowOptions, workflow interface{}, args ...interface{}) WithStartWorkflowOperation

		// CancelWorkflow cancels a workflow in execution
		//  - workflow ID of the workflow.
		//  - runID can be default(empty string). if empty string then it will pick the running execution of that workflow ID.
		// The errors it can return:
		//  - serviceerror.NotFound
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		CancelWorkflow(ctx context.Context, workflowID string, runID string) error

		// TerminateWorkflow terminates a workflow execution.
		// workflowID is required, other parameters are optional.
		//  - workflow ID of the workflow.
		//  - runID can be default(empty string). if empty string then it will pick the running execution of that workflow ID.
		// The errors it can return:
		//  - serviceerror.NotFound
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		TerminateWorkflow(ctx context.Context, workflowID string, runID string, reason string, details ...interface{}) error

		// GetWorkflowHistory gets history events of a particular workflow
		//  - workflow ID of the workflow.
		//  - runID can be default(empty string). if empty string then it will pick the last running execution of that workflow ID.
		//  - whether use long poll for tracking new events: when the workflow is running, there can be new events generated during iteration
		//    of HistoryEventIterator, if isLongPoll == true, then iterator will do long poll, tracking new history event, i.e. the iteration
		//   will not be finished until workflow is finished; if isLongPoll == false, then iterator will only return current history events.
		//  - whether return all history events or just the last event, which contains the workflow execution end result
		// Example:-
		//  To iterate all events,
		//    iter := GetWorkflowHistory(ctx, workflowID, runID, isLongPoll, filterType)
		//    events := []*shared.HistoryEvent{}
		//    for iter.HasNext() {
		//      event, err := iter.Next()
		//      if err != nil {
		//        return err
		//      }
		//      events = append(events, event)
		//    }
		GetWorkflowHistory(ctx context.Context, workflowID string, runID string, isLongPoll bool, filterType enumspb.HistoryEventFilterType) HistoryEventIterator

		// CompleteActivity reports activity completed.
		// activity Execute method can return activity.ErrResultPending to
		// indicate the activity is not completed when it's Execute method returns. In that case, this CompleteActivity() method
		// should be called when that activity is completed with the actual result and error. If err is nil, activity task
		// completed event will be reported; if err is CanceledError, activity task canceled event will be reported; otherwise,
		// activity task failed event will be reported.
		// An activity implementation should use GetActivityInfo(ctx).TaskToken function to get task token to use for completion.
		// Example:-
		//  To complete with a result.
		//    CompleteActivity(token, "Done", nil)
		//  To fail the activity with an error.
		//      CompleteActivity(token, nil, temporal.NewApplicationError("reason", details)
		// The activity can fail with below errors ApplicationError, TimeoutError, CanceledError.
		CompleteActivity(ctx context.Context, taskToken []byte, result interface{}, err error) error

		// CompleteActivityByID reports activity completed.
		// Similar to CompleteActivity, but may save user from keeping taskToken info.
		// activity Execute method can return activity.ErrResultPending to
		// indicate the activity is not completed when it's Execute method returns. In that case, this CompleteActivityById() method
		// should be called when that activity is completed with the actual result and error. If err is nil, activity task
		// completed event will be reported; if err is CanceledError, activity task canceled event will be reported; otherwise,
		// activity task failed event will be reported.
		// An activity implementation should use activityID provided in ActivityOption to use for completion.
		// namespace name, workflowID, activityID are required, runID is optional.
		// The errors it can return:
		//  - ApplicationError
		//  - TimeoutError
		//  - CanceledError
		CompleteActivityByID(ctx context.Context, namespace, workflowID, runID, activityID string, result interface{}, err error) error

		// RecordActivityHeartbeat records heartbeat for an activity.
		// details - is the progress you want to record along with heart beat for this activity.
		// The errors it can return:
		//  - serviceerror.NotFound
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		RecordActivityHeartbeat(ctx context.Context, taskToken []byte, details ...interface{}) error

		// RecordActivityHeartbeatByID records heartbeat for an activity.
		// details - is the progress you want to record along with heart beat for this activity.
		// The errors it can return:
		//  - serviceerror.NotFound
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		RecordActivityHeartbeatByID(ctx context.Context, namespace, workflowID, runID, activityID string, details ...interface{}) error

		// ListClosedWorkflow gets closed workflow executions based on request filters
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//  - serviceerror.NamespaceNotFound
		ListClosedWorkflow(ctx context.Context, request *workflowservice.ListClosedWorkflowExecutionsRequest) (*workflowservice.ListClosedWorkflowExecutionsResponse, error)

		// ListOpenWorkflow gets open workflow executions based on request filters
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//  - serviceerror.NamespaceNotFound
		ListOpenWorkflow(ctx context.Context, request *workflowservice.ListOpenWorkflowExecutionsRequest) (*workflowservice.ListOpenWorkflowExecutionsResponse, error)

		// ListWorkflow gets workflow executions based on query.The query is basically the SQL WHERE clause,
		// examples:
		//  - "(WorkflowID = 'wid1' or (WorkflowType = 'type2' and WorkflowID = 'wid2'))".
		//  - "CloseTime between '2019-08-27T15:04:05+00:00' and '2019-08-28T15:04:05+00:00'".
		//  - to list only open workflow use "CloseTime is null"
		// Retrieved workflow executions are sorted by StartTime in descending order when list open workflow,
		// and sorted by CloseTime in descending order for other queries.
		// For supported operations on different server versions see [Visibility].
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//
		// [Visibility]: https://docs.temporal.io/visibility
		ListWorkflow(ctx context.Context, request *workflowservice.ListWorkflowExecutionsRequest) (*workflowservice.ListWorkflowExecutionsResponse, error)

		// ListArchivedWorkflow gets archived workflow executions based on query. This API will return BadRequest if Temporal
		// cluster or target namespace is not configured for visibility archival or read is not enabled. The query is basically the SQL WHERE clause.
		// However, different visibility archivers have different limitations on the query. Please check the documentation of the visibility archiver used
		// by your namespace to see what kind of queries are accept and whether retrieved workflow executions are ordered or not.
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		ListArchivedWorkflow(ctx context.Context, request *workflowservice.ListArchivedWorkflowExecutionsRequest) (*workflowservice.ListArchivedWorkflowExecutionsResponse, error)

		// ScanWorkflow gets workflow executions based on query. The query is basically the SQL WHERE clause
		// (see ListWorkflow for query examples).
		// ScanWorkflow should be used when retrieving large amount of workflows and order is not needed.
		// It will use more resources than ListWorkflow, but will be several times faster
		// when retrieving millions of workflows.
		// For supported operations on different server versions see [Visibility].
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		// [Visibility]: https://docs.temporal.io/visibility
		ScanWorkflow(ctx context.Context, request *workflowservice.ScanWorkflowExecutionsRequest) (*workflowservice.ScanWorkflowExecutionsResponse, error)

		// CountWorkflow gets number of workflow executions based on query. The query is basically the SQL WHERE clause
		// (see ListWorkflow for query examples).
		// For supported operations on different server versions see [Visibility].
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//
		// [Visibility]: https://docs.temporal.io/visibility
		CountWorkflow(ctx context.Context, request *workflowservice.CountWorkflowExecutionsRequest) (*workflowservice.CountWorkflowExecutionsResponse, error)

		// GetSearchAttributes returns valid search attributes keys and value types.
		// The search attributes can be used in query of List/Scan/Count APIs. Adding new search attributes requires temporal server
		// to update dynamic config ValidSearchAttributes.
		GetSearchAttributes(ctx context.Context) (*workflowservice.GetSearchAttributesResponse, error)

		// QueryWorkflow queries a given workflow execution and returns the query result synchronously. Parameter workflowID
		// and queryType are required, other parameters are optional. The workflowID and runID (optional) identify the
		// target workflow execution that this query will be send to. If runID is not specified (empty string), server will
		// use the currently running execution of that workflowID. The queryType specifies the type of query you want to
		// run. By default, temporal supports "__stack_trace" as a standard query type, which will return string value
		// representing the call stack of the target workflow. The target workflow could also setup different query handler
		// to handle custom query types.
		// See comments at workflow.SetQueryHandler(ctx Context, queryType string, handler interface{}) for more details
		// on how to setup query handler within the target workflow.
		//  - workflowID is required.
		//  - runID can be default(empty string). if empty string then it will pick the running execution of that workflow ID.
		//  - queryType is the type of the query.
		//  - args... are the optional query parameters.
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//  - serviceerror.NotFound
		//  - serviceerror.QueryFailed
		QueryWorkflow(ctx context.Context, workflowID string, runID string, queryType string, args ...interface{}) (converter.EncodedValue, error)

		// QueryWorkflowWithOptions queries a given workflow execution and returns the query result synchronously.
		// See QueryWorkflowWithOptionsRequest and QueryWorkflowWithOptionsResponse for more information.
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//  - serviceerror.NotFound
		//  - serviceerror.QueryFailed
		QueryWorkflowWithOptions(ctx context.Context, request *QueryWorkflowWithOptionsRequest) (*QueryWorkflowWithOptionsResponse, error)

		// DescribeWorkflowExecution returns information about the specified workflow execution.
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//  - serviceerror.NotFound
		DescribeWorkflowExecution(ctx context.Context, workflowID, runID string) (*workflowservice.DescribeWorkflowExecutionResponse, error)

		// UpdateWorkflowExecutionOptions partially overrides the [WorkflowExecutionOptions] of an existing workflow execution
		// and returns the new [WorkflowExecutionOptions] after applying the changes.
		// It is intended for building tools that can selectively apply ad-hoc workflow configuration changes.
		// NOTE: Experimental
		UpdateWorkflowExecutionOptions(ctx context.Context, options UpdateWorkflowExecutionOptionsRequest) (WorkflowExecutionOptions, error)

		// DescribeTaskQueue returns information about the target taskqueue, right now this API returns the
		// pollers which polled this taskqueue in last few minutes.
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//  - serviceerror.NotFound
		DescribeTaskQueue(ctx context.Context, taskqueue string, taskqueueType enumspb.TaskQueueType) (*workflowservice.DescribeTaskQueueResponse, error)

		// ResetWorkflowExecution reset an existing workflow execution to WorkflowTaskFinishEventId(exclusive).
		// And it will immediately terminating the current execution instance.
		// RequestId is used to deduplicate requests. It will be autogenerated if not set.
		ResetWorkflowExecution(ctx context.Context, request *workflowservice.ResetWorkflowExecutionRequest) (*workflowservice.ResetWorkflowExecutionResponse, error)

		// UpdateWorkerBuildIdCompatibility allows you to update the worker-build-id based version sets for a particular
		// task queue. This is used in conjunction with workers who specify their build id and thus opt into the
		// feature.
		UpdateWorkerBuildIdCompatibility(ctx context.Context, options *UpdateWorkerBuildIdCompatibilityOptions) error

		// GetWorkerBuildIdCompatibility returns the worker-build-id based version sets for a particular task queue.
		GetWorkerBuildIdCompatibility(ctx context.Context, options *GetWorkerBuildIdCompatibilityOptions) (*WorkerBuildIDVersionSets, error)

		// GetWorkerTaskReachability returns which versions are is still in use by open or closed workflows.
		GetWorkerTaskReachability(ctx context.Context, options *GetWorkerTaskReachabilityOptions) (*WorkerTaskReachability, error)

		// DescribeTaskQueueEnhanced returns information about the target task queue, broken down by Build Id:
		//   - List of pollers
		//   - Workflow Reachability status
		//   - Backlog info for Workflow and/or Activity tasks
		// When not supported by the server, it returns an empty [TaskQueueDescription] if there is no information
		// about the task queue, or an error when the response identifies an unsupported server.
		// Note that using a sticky queue as target is not supported.
		// Also, workflow reachability status is eventually consistent, and it could take a few minutes to update.
		// WARNING: Worker versioning is currently experimental, and requires server 1.24+
		DescribeTaskQueueEnhanced(ctx context.Context, options DescribeTaskQueueEnhancedOptions) (TaskQueueDescription, error)

		// UpdateWorkerVersioningRules allows updating the worker-build-id based assignment and redirect rules for a given
		// task queue. This is used in conjunction with workers who specify their build id and thus opt into the feature.
		// The errors it can return:
		//  - serviceerror.FailedPrecondition when the conflict token is invalid
		// WARNING: Worker versioning is currently experimental, and requires server 1.24+
		UpdateWorkerVersioningRules(ctx context.Context, options UpdateWorkerVersioningRulesOptions) (*WorkerVersioningRules, error)

		// GetWorkerVersioningRules returns the worker-build-id assignment and redirect rules for a task queue.
		// WARNING: Worker versioning is currently experimental, and requires server 1.24+
		GetWorkerVersioningRules(ctx context.Context, options GetWorkerVersioningOptions) (*WorkerVersioningRules, error)

		// CheckHealth performs a server health check using the gRPC health check
		// API. If the check fails, an error is returned.
		CheckHealth(ctx context.Context, request *CheckHealthRequest) (*CheckHealthResponse, error)

		// UpdateWorkflow issues an update request to the
		// specified workflow execution and returns a handle to the update that
		// is running in in parallel with the calling thread. Errors returned
		// from the server will be exposed through the return value of
		// WorkflowExecutionUpdateHandle.Get(). Errors that occur before the
		// update is requested (e.g. if the required workflow ID field is
		// missing from the UpdateWorkflowOptions) are returned
		// directly from this function call.
		UpdateWorkflow(ctx context.Context, options UpdateWorkflowOptions) (WorkflowUpdateHandle, error)

		// UpdateWithStartWorkflow issues an update-with-start request. A
		// WorkflowIDConflictPolicy must be set. If the specified workflow is
		// not running, then a new workflow execution is started and the update
		// is sent in the first workflow task. Alternatively if the specified
		// workflow is running then, if the WorkflowIDConflictPolicy is
		// USE_EXISTING, the update is issued against the specified workflow,
		// and if the WorkflowIDConflictPolicy is FAIL, an error is returned.
		//
		// NOTE: Experimental
		UpdateWithStartWorkflow(ctx context.Context, options UpdateWithStartWorkflowOptions) (WorkflowUpdateHandle, error)

		// GetWorkflowUpdateHandle creates a handle to the referenced update
		// which can be polled for an outcome. Note that runID is optional and
		// if not specified the most recent runID will be used.
		GetWorkflowUpdateHandle(GetWorkflowUpdateHandleOptions) WorkflowUpdateHandle

		// WorkflowService provides access to the underlying gRPC service. This should only be used for advanced use cases
		// that cannot be accomplished via other Client methods. Unlike calls to other Client methods, calls directly to the
		// service are not configured with internal semantics such as automatic retries.
		WorkflowService() workflowservice.WorkflowServiceClient

		// OperatorService creates a new operator service client with the same gRPC connection as this client.
		OperatorService() operatorservice.OperatorServiceClient

		// Schedule creates a new shedule client with the same gRPC connection as this client.
		ScheduleClient() ScheduleClient

		// DeploymentClient creates a new deployment client with the same gRPC connection as this client.
		DeploymentClient() DeploymentClient

		// Close client and clean up underlying resources.
		Close()
	}

	// ClientOptions are optional parameters for Client creation.
	//
	// Exposed as: [go.temporal.io/sdk/client.Options]
	ClientOptions struct {
		// Optional: To set the host:port for this client to connect to.
		// default: localhost:7233
		//
		// This is a gRPC address and therefore can also support a special-formatted address of "<resolver>:///<value>" that
		// will use a registered resolver. By default all hosts returned from the resolver will be used in a round-robin
		// fashion.
		//
		// The "dns" resolver is registered by and used by default.
		//
		// A custom resolver can be created to provide multiple hosts in other ways. For example, to manually provide
		// multiple IPs to round-robin across, a google.golang.org/grpc/resolver/manual resolver can be created and
		// registered with google.golang.org/grpc/resolver with a custom scheme:
		//    builder := manual.NewBuilderWithScheme("myresolver")
		//    builder.InitialState(resolver.State{Addresses: []resolver.Address{{Addr: "1.2.3.4:1234"}, {Addr: "2.3.4.5:2345"}}})
		//    resolver.Register(builder)
		//    c, err := client.Dial(client.Options{HostPort: "myresolver:///ignoredvalue"})
		// Other more advanced resolvers can also be registered.
		HostPort string

		// Optional: To set the namespace name for this client to work with.
		// default: default
		Namespace string

		// Optional: Set the credentials for this client.
		Credentials Credentials

		// Optional: Logger framework can use to log.
		// default: default logger provided.
		Logger log.Logger

		// Optional: Metrics handler for reporting metrics.
		// default: no metrics.
		MetricsHandler metrics.Handler

		// Optional: Sets an identify that can be used to track this host for debugging.
		// default: default identity that include hostname, groupName and process ID.
		Identity string

		// Optional: Sets DataConverter to customize serialization/deserialization of arguments in Temporal
		// default: defaultDataConverter, an combination of google protobuf converter, gogo protobuf converter and json converter
		DataConverter converter.DataConverter

		// Optional: Sets FailureConverter to customize serialization/deserialization of errors.
		// default: temporal.DefaultFailureConverter, does not encode any fields of the error. Use temporal.NewDefaultFailureConverter
		// options to configure or create a custom converter.
		FailureConverter converter.FailureConverter

		// Optional: Sets ContextPropagators that allows users to control the context information passed through a workflow
		// default: nil
		ContextPropagators []ContextPropagator

		// Optional: Sets options for server connection that allow users to control features of connections such as TLS settings.
		// default: no extra options
		ConnectionOptions ConnectionOptions

		// Optional: HeadersProvider will be invoked on every outgoing gRPC request and gives user ability to
		// set custom request headers. This can be used to set auth headers for example.
		HeadersProvider HeadersProvider

		// Optional parameter that is designed to be used *in tests*. It gets invoked last in
		// the gRPC interceptor chain and can be used to induce artificial failures in test scenarios.
		TrafficController TrafficController

		// Interceptors to apply to some calls of the client. Earlier interceptors
		// wrap later interceptors.
		//
		// Any interceptors that also implement Interceptor (meaning they implement
		// WorkerInterceptor in addition to ClientInterceptor) will be used for
		// worker interception as well. When worker interceptors are here and in
		// worker options, the ones here wrap the ones in worker options. The same
		// interceptor should not be set here and in worker options.
		Interceptors []ClientInterceptor

		// If set true, error code labels will not be included on request failure metrics.
		DisableErrorCodeMetricTags bool
	}

	CloudOperationsClient interface {
		CloudService() cloudservice.CloudServiceClient
		Close()
	}

	// CloudOperationsClientOptions are parameters for CloudOperationsClient creation.
	//
	// WARNING: Cloud operations client is currently experimental.
	//
	// Exposed as: [go.temporal.io/sdk/client.CloudOperationsClientOptions]
	CloudOperationsClientOptions struct {
		// Optional: The credentials for this client. This is essentially required.
		// See [go.temporal.io/sdk/client.NewAPIKeyStaticCredentials],
		// [go.temporal.io/sdk/client.NewAPIKeyDynamicCredentials], and
		// [go.temporal.io/sdk/client.NewMTLSCredentials].
		// Default: No credentials.
		Credentials Credentials

		// Optional: Version header for safer mutations. May or may not be required
		// depending on cloud settings.
		// Default: No header.
		Version string

		// Optional: Advanced server connection options such as TLS settings. Not
		// usually needed.
		ConnectionOptions ConnectionOptions

		// Optional: Logger framework can use to log.
		// Default: Default logger provided.
		Logger log.Logger

		// Optional: Metrics handler for reporting metrics.
		// Default: No metrics
		MetricsHandler metrics.Handler

		// Optional: Overrides the specific host to connect to. Not usually needed.
		// Default: saas-api.tmprl.cloud:443
		HostPort string

		// Optional: Disable TLS.
		// Default: false (i.e. TLS enabled)
		DisableTLS bool
	}

	// HeadersProvider returns a map of gRPC headers that should be used on every request.
	HeadersProvider interface {
		GetHeaders(ctx context.Context) (map[string]string, error)
	}
	// TrafficController is getting called in the interceptor chain with API invocation parameters.
	// Result is either nil if API call is allowed or an error, in which case request would be interrupted and
	// the error will be propagated back through the interceptor chain.
	TrafficController interface {
		CheckCallAllowed(ctx context.Context, method string, req, reply interface{}) error
	}

	// ConnectionOptions is provided by SDK consumers to control optional connection params.
	//
	// Exposed as: [go.temporal.io/sdk/client.ConnectionOptions]
	ConnectionOptions struct {
		// TLS configures connection level security credentials.
		TLS *tls.Config

		// Authority specifies the value to be used as the :authority pseudo-header.
		// This value only used when TLS is nil.
		Authority string

		// Disable keep alive ping from client to the server.
		DisableKeepAliveCheck bool

		// After a duration of this time if the client doesn't see any activity it
		// pings the server to see if the transport is still alive.
		// If set below 10s, a minimum value of 10s will be used instead.
		// default: 30s
		KeepAliveTime time.Duration

		// After having pinged for keepalive check, the client waits for a duration
		// of Timeout and if no activity is seen even after that the connection is
		// closed.
		// default: 15s
		KeepAliveTimeout time.Duration

		// GetSystemInfoTimeout is the timeout for the RPC made by the
		// client to fetch server capabilities.
		GetSystemInfoTimeout time.Duration

		// if true, when there are no active RPCs, Time and Timeout will be ignored and no
		// keepalive pings will be sent.
		// If false, client sends keepalive pings even with no active RPCs
		// default: false
		DisableKeepAlivePermitWithoutStream bool

		// MaxPayloadSize is a number of bytes that gRPC would allow to travel to and from server. Defaults to 128 MB.
		MaxPayloadSize int

		// Advanced dial options for gRPC connections. These are applied after the internal default dial options are
		// applied. Therefore any dial options here may override internal ones. Dial options WithBlock, WithTimeout,
		// WithReturnConnectionError, and FailOnNonTempDialError are ignored since [grpc.NewClient] is used.
		//
		// For gRPC interceptors, internal interceptors such as error handling, metrics, and retrying are done via
		// grpc.WithChainUnaryInterceptor. Therefore to add inner interceptors that are wrapped by those, a
		// grpc.WithChainUnaryInterceptor can be added as an option here. To add a single outer interceptor, a
		// grpc.WithUnaryInterceptor option can be added since grpc.WithUnaryInterceptor is prepended to chains set with
		// grpc.WithChainUnaryInterceptor.
		DialOptions []grpc.DialOption

		// Hidden for use by client overloads.
		disableEagerConnection bool

		// Internal atomic that, when true, will not retry internal errors like
		// other gRPC errors. If not present during service client creation, it will
		// be created as false. This is set to true when server capabilities are
		// fetched.
		excludeInternalFromRetry *atomic.Bool
	}

	// StartWorkflowOptions configuration parameters for starting a workflow execution.
	// The current timeout resolution implementation is in seconds and uses math.Ceil(d.Seconds()) as the duration. But is
	// subjected to change in the future.
	//
	// Exposed as: [go.temporal.io/sdk/client.StartWorkflowOptions]
	StartWorkflowOptions struct {
		// ID - The business identifier of the workflow execution.
		// Optional: defaulted to a uuid.
		ID string

		// TaskQueue - The workflow tasks of the workflow are scheduled on the queue with this name.
		// This is also the name of the activity task queue on which activities are scheduled.
		// The workflow author can choose to override this using activity options.
		// Mandatory: No default.
		TaskQueue string

		// WorkflowExecutionTimeout - The timeout for duration of workflow execution.
		// It includes retries and continue as new. Use WorkflowRunTimeout to limit execution time
		// of a single workflow run.
		// The resolution is seconds.
		// Optional: defaulted to unlimited.
		WorkflowExecutionTimeout time.Duration

		// WorkflowRunTimeout - The timeout for duration of a single workflow run.
		// The resolution is seconds.
		// Optional: defaulted to WorkflowExecutionTimeout.
		WorkflowRunTimeout time.Duration

		// WorkflowTaskTimeout - The timeout for processing workflow task from the time the worker
		// pulled this task. If a workflow task is lost, it is retried after this timeout.
		// The resolution is seconds.
		// Optional: defaulted to 10 secs.
		WorkflowTaskTimeout time.Duration

		// WorkflowIDReusePolicy - Specifies server behavior if a *completed* workflow with the same id exists.
		// This can be useful for dedupe logic if set to RejectDuplicate
		// Optional: defaulted to AllowDuplicate.
		WorkflowIDReusePolicy enumspb.WorkflowIdReusePolicy

		// WorkflowIDConflictPolicy - Specifies server behavior if a *running* workflow with the same id exists.
		// This cannot be set if WorkflowIDReusePolicy is set to TerminateIfRunning.
		// Optional: defaulted to Fail.
		WorkflowIDConflictPolicy enumspb.WorkflowIdConflictPolicy

		// When WorkflowExecutionErrorWhenAlreadyStarted is true, Client.ExecuteWorkflow will return an error if the
		// workflow id has already been used and WorkflowIDReusePolicy or WorkflowIDConflictPolicy would
		// disallow a re-run. If it is set to false, rather than erroring a WorkflowRun instance representing
		// the current or last run will be returned. However, when WithStartOperation is set, this field is ignored and
		// the WorkflowIDConflictPolicy UseExisting must be used instead to prevent erroring.
		//
		// Optional: defaults to false
		WorkflowExecutionErrorWhenAlreadyStarted bool

		// RetryPolicy - Optional retry policy for workflow. If a retry policy is specified, in case of workflow failure
		// server will start new workflow execution if needed based on the retry policy.
		RetryPolicy *RetryPolicy

		// CronSchedule - Optional cron schedule for workflow. If a cron schedule is specified, the workflow will run
		// as a cron based on the schedule. The scheduling will be based on UTC time. Schedule for next run only happen
		// after the current run is completed/failed/timeout. If a RetryPolicy is also supplied, and the workflow failed
		// or timeout, the workflow will be retried based on the retry policy. While the workflow is retrying, it won't
		// schedule its next run. If next schedule is due while workflow is running (or retrying), then it will skip that
		// schedule. Cron workflow will not stop until it is terminated or canceled (by returning temporal.CanceledError).
		// The cron spec is as following:
		// ┌───────────── minute (0 - 59)
		// │ ┌───────────── hour (0 - 23)
		// │ │ ┌───────────── day of the month (1 - 31)
		// │ │ │ ┌───────────── month (1 - 12)
		// │ │ │ │ ┌───────────── day of the week (0 - 6) (Sunday to Saturday)
		// │ │ │ │ │
		// │ │ │ │ │
		// * * * * *
		// Cannot be set the same time as a StartDelay or WithStartOperation.
		CronSchedule string

		// Memo - Optional non-indexed info that will be shown in list workflow.
		Memo map[string]interface{}

		// SearchAttributes - Optional indexed info that can be used in query of List/Scan/Count workflow APIs. The key and value type must be registered on Temporal server side.
		// Use GetSearchAttributes API to get valid key and corresponding value type.
		// For supported operations on different server versions see [Visibility].
		//
		// Deprecated: use TypedSearchAttributes instead.
		//
		// [Visibility]: https://docs.temporal.io/visibility
		SearchAttributes map[string]interface{}

		// TypedSearchAttributes - Specifies Search Attributes that will be attached to the Workflow. Search Attributes are
		// additional indexed information attributed to workflow and used for search and visibility. The search attributes
		// can be used in query of List/Scan/Count workflow APIs. The key and its value type must be registered on Temporal
		// server side. For supported operations on different server versions see [Visibility].
		//
		// Optional: default to none.
		//
		// [Visibility]: https://docs.temporal.io/visibility
		TypedSearchAttributes SearchAttributes

		// EnableEagerStart - request eager execution for this workflow, if a local worker is available.
		// Cannot be set the same time as a WithStartOperation.
		//
		// WARNING: Eager start does not respect worker versioning. An eagerly started workflow may run on
		// any available local worker even if that worker is not in the default build ID set.
		//
		// NOTE: Experimental
		EnableEagerStart bool

		// StartDelay - Time to wait before dispatching the first workflow task.
		// A signal from signal with start will not trigger a workflow task.
		// Cannot be set the same time as a CronSchedule or WithStartOperation.
		StartDelay time.Duration

		// StaticSummary - Single-line fixed summary for this workflow execution that will appear in UI/CLI. This can be
		// in single-line Temporal markdown format.
		//
		// Optional: defaults to none/empty.
		//
		// NOTE: Experimental
		StaticSummary string

		// Details - General fixed details for this workflow execution that will appear in UI/CLI. This can be in
		// Temporal markdown format and can span multiple lines. This is a fixed value on the workflow that cannot be
		// updated. For details that can be updated, use SetCurrentDetails within the workflow.
		//
		// Optional: defaults to none/empty.
		//
		// NOTE: Experimental
		StaticDetails string

		// VersioningOverride - Sets the versioning configuration of a specific workflow execution, ignoring current
		// server or worker default policies. This enables running canary tests without affecting existing workflows.
		// To unset the override after the workflow is running, use [UpdateWorkflowExecutionOptions].
		// Optional: defaults to no override.
		//
		// NOTE: Experimental
		VersioningOverride VersioningOverride

		// request ID. Only settable by the SDK - e.g. [temporalnexus.workflowRunOperation].
		requestID string
		// workflow completion callback. Only settable by the SDK - e.g. [temporalnexus.workflowRunOperation].
		callbacks []*commonpb.Callback
		// links. Only settable by the SDK - e.g. [temporalnexus.workflowRunOperation].
		links []*commonpb.Link
	}

	// WithStartWorkflowOperation defines how to start a workflow when using UpdateWithStartWorkflow.
	// See [NewWithStartWorkflowOperation] and [UpdateWithStartWorkflow].
	// NOTE: Experimental
	WithStartWorkflowOperation interface {
		// Get returns the WorkflowRun that was targeted by the UpdateWithStartWorkflow call.
		// This is a blocking API.
		Get(ctx context.Context) (WorkflowRun, error)
	}

	withStartWorkflowOperationImpl struct {
		input *ClientExecuteWorkflowInput
		// flag to ensure the operation is only executed once
		executed atomic.Bool
		// channel to indicate that handle or err is available
		doneCh chan struct{}
		// workflowRun and err cannot be accessed before doneCh is closed
		workflowRun WorkflowRun
		err         error
	}

	// RetryPolicy defines the retry policy.
	// Note that the history of activity with retry policy will be different: the started event will be written down into
	// history only when the activity completes or "finally" timeouts/fails. And the started event only records the last
	// started time. Because of that, to check an activity has started or not, you cannot rely on history events. Instead,
	// you can use CLI to describe the workflow to see the status of the activity:
	//     tctl --ns <namespace> wf desc -w <wf-id>
	//
	// Exposed as: [go.temporal.io/sdk/temporal.RetryPolicy]
	RetryPolicy struct {
		// Backoff interval for the first retry. If BackoffCoefficient is 1.0 then it is used for all retries.
		// If not set or set to 0, a default interval of 1s will be used.
		InitialInterval time.Duration

		// Coefficient used to calculate the next retry backoff interval.
		// The next retry interval is previous interval multiplied by this coefficient.
		// Must be 1 or larger. Default is 2.0.
		BackoffCoefficient float64

		// Maximum backoff interval between retries. Exponential backoff leads to interval increase.
		// This value is the cap of the interval. Default is 100x of initial interval.
		MaximumInterval time.Duration

		// Maximum number of attempts. When exceeded the retries stop even if not expired yet.
		// If not set or set to 0, it means unlimited, and rely on activity ScheduleToCloseTimeout to stop.
		MaximumAttempts int32

		// Non-Retriable errors. This is optional. Temporal server will stop retry if error type matches this list.
		// Note:
		//  - cancellation is not a failure, so it won't be retried,
		//  - only StartToClose or Heartbeat timeouts are retryable.
		NonRetryableErrorTypes []string
	}

	// NamespaceClient is the client for managing operations on the namespace.
	// CLI, tools, ... can use this layer to manager operations on namespace.
	NamespaceClient interface {
		// Register a namespace with temporal server
		// The errors it can throw:
		//  - NamespaceAlreadyExistsError
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		Register(ctx context.Context, request *workflowservice.RegisterNamespaceRequest) error

		// Describe a namespace. The namespace has 3 part of information
		// NamespaceInfo - Which has Name, Status, Description, Owner Email
		// NamespaceConfiguration - Configuration like Workflow Execution Retention Period In Days, Whether to emit metrics.
		// ReplicationConfiguration - replication config like clusters and active cluster name
		// The errors it can throw:
		//  - serviceerror.NamespaceNotFound
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		Describe(ctx context.Context, name string) (*workflowservice.DescribeNamespaceResponse, error)

		// Update a namespace.
		// The errors it can throw:
		//  - serviceerror.NamespaceNotFound
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		Update(ctx context.Context, request *workflowservice.UpdateNamespaceRequest) error

		// Close client and clean up underlying resources.
		Close()
	}
)

// Credentials are optional credentials that can be specified in ClientOptions.
//
// Exposed as: [go.temporal.io/sdk/client.Credentials]
type Credentials interface {
	applyToOptions(*ConnectionOptions) error
	// Can return nil to have no interceptor
	gRPCInterceptor() grpc.UnaryClientInterceptor
}

// DialClient creates a client and attempts to connect to the server.
//
// Exposed as: [go.temporal.io/sdk/client.DialContext]
func DialClient(ctx context.Context, options ClientOptions) (Client, error) {
	options.ConnectionOptions.disableEagerConnection = false
	return NewClient(ctx, options)
}

// NewLazyClient creates a client and does not attempt to connect to the server.
//
// Exposed as: [go.temporal.io/sdk/client.NewLazyClient]
func NewLazyClient(options ClientOptions) (Client, error) {
	options.ConnectionOptions.disableEagerConnection = true
	return NewClient(context.Background(), options)
}

// NewClient creates an instance of a workflow client
//
// Deprecated: Use DialClient or NewLazyClient instead.
//
// Exposed as: [go.temporal.io/sdk/client.NewClient]
func NewClient(ctx context.Context, options ClientOptions) (Client, error) {
	return newClient(ctx, options, nil)
}

// NewClientFromExisting creates a new client using the same connection as the
// existing client.
//
// Exposed as: [go.temporal.io/sdk/client.NewClientFromExistingWithContext]
func NewClientFromExisting(ctx context.Context, existingClient Client, options ClientOptions) (Client, error) {
	existing, _ := existingClient.(*WorkflowClient)
	if existing == nil {
		return nil, fmt.Errorf("existing client must have been created directly from a client package call")
	}
	return newClient(ctx, options, existing)
}

func newClient(ctx context.Context, options ClientOptions, existing *WorkflowClient) (Client, error) {
	if options.Namespace == "" {
		options.Namespace = DefaultNamespace
	}

	// Initialize root tags
	if options.MetricsHandler == nil {
		options.MetricsHandler = metrics.NopHandler
	}
	options.MetricsHandler = options.MetricsHandler.WithTags(metrics.RootTags(options.Namespace))

	if options.HostPort == "" {
		options.HostPort = LocalHostPort
	}

	if options.Logger == nil {
		options.Logger = ilog.NewDefaultLogger()
		options.Logger.Info("No logger configured for temporal client. Created default one.")
	}

	if options.Credentials != nil {
		if err := options.Credentials.applyToOptions(&options.ConnectionOptions); err != nil {
			return nil, err
		}
	}

	// Dial or use existing connection
	var connection *grpc.ClientConn
	var err error
	if existing == nil {
		options.ConnectionOptions.excludeInternalFromRetry = &atomic.Bool{}
		connection, err = dial(newDialParameters(&options, options.ConnectionOptions.excludeInternalFromRetry))
		if err != nil {
			return nil, err
		}
	} else {
		connection = existing.conn
	}

	client := NewServiceClient(workflowservice.NewWorkflowServiceClient(connection), connection, options)

	// If using existing connection, always load its capabilities and use them for
	// the new connection. Otherwise, only load server capabilities eagerly if not
	// disabled.
	if existing != nil {
		if client.capabilities, err = existing.loadCapabilities(ctx, options.ConnectionOptions.GetSystemInfoTimeout); err != nil {
			return nil, err
		}
		client.unclosedClients = existing.unclosedClients
	} else {
		if !options.ConnectionOptions.disableEagerConnection {
			if _, err := client.loadCapabilities(ctx, options.ConnectionOptions.GetSystemInfoTimeout); err != nil {
				client.Close()
				return nil, err
			}
		}
		var unclosedClients int32
		client.unclosedClients = &unclosedClients
	}
	atomic.AddInt32(client.unclosedClients, 1)

	return client, nil
}

func newDialParameters(options *ClientOptions, excludeInternalFromRetry *atomic.Bool) dialParameters {
	return dialParameters{
		UserConnectionOptions: options.ConnectionOptions,
		HostPort:              options.HostPort,
		RequiredInterceptors:  requiredInterceptors(options, excludeInternalFromRetry),
		DefaultServiceConfig:  defaultServiceConfig,
	}
}

// NewServiceClient creates workflow client from workflowservice.WorkflowServiceClient. Must be used internally in unit tests only.
func NewServiceClient(workflowServiceClient workflowservice.WorkflowServiceClient, conn *grpc.ClientConn, options ClientOptions) *WorkflowClient {
	// Namespace can be empty in unit tests.
	if options.Namespace == "" {
		options.Namespace = DefaultNamespace
	}

	if options.Identity == "" {
		options.Identity = getWorkerIdentity("")
	}

	if options.DataConverter == nil {
		options.DataConverter = converter.GetDefaultDataConverter()
	}

	if options.FailureConverter == nil {
		options.FailureConverter = GetDefaultFailureConverter()
	}

	if options.MetricsHandler == nil {
		options.MetricsHandler = metrics.NopHandler
	}

	if options.ConnectionOptions.excludeInternalFromRetry == nil {
		options.ConnectionOptions.excludeInternalFromRetry = &atomic.Bool{}
	}

	// Collect set of applicable worker interceptors
	var workerInterceptors []WorkerInterceptor
	for _, interceptor := range options.Interceptors {
		if workerInterceptor, _ := interceptor.(WorkerInterceptor); workerInterceptor != nil {
			workerInterceptors = append(workerInterceptors, workerInterceptor)
		}
	}

	client := &WorkflowClient{
		workflowService:          workflowServiceClient,
		conn:                     conn,
		namespace:                options.Namespace,
		registry:                 newRegistry(),
		metricsHandler:           options.MetricsHandler,
		logger:                   options.Logger,
		identity:                 options.Identity,
		dataConverter:            options.DataConverter,
		failureConverter:         options.FailureConverter,
		contextPropagators:       options.ContextPropagators,
		workerInterceptors:       workerInterceptors,
		excludeInternalFromRetry: options.ConnectionOptions.excludeInternalFromRetry,
		eagerDispatcher: &eagerWorkflowDispatcher{
			workersByTaskQueue: make(map[string]map[eagerWorker]struct{}),
		},
	}

	// Create outbound interceptor by wrapping backwards through chain
	client.interceptor = &workflowClientInterceptor{client: client}
	for i := len(options.Interceptors) - 1; i >= 0; i-- {
		client.interceptor = options.Interceptors[i].InterceptClient(client.interceptor)
	}

	return client
}

// DialCloudOperationsClient creates a cloud client to perform cloud-management
// operations.
//
// Exposed as: [go.temporal.io/sdk/client.DialCloudOperationsClient]
func DialCloudOperationsClient(ctx context.Context, options CloudOperationsClientOptions) (CloudOperationsClient, error) {
	// Set defaults
	if options.MetricsHandler == nil {
		options.MetricsHandler = metrics.NopHandler
	}
	if options.Logger == nil {
		options.Logger = ilog.NewDefaultLogger()
	}
	if options.HostPort == "" {
		options.HostPort = "saas-api.tmprl.cloud:443"
	}
	if options.Version != "" {
		options.ConnectionOptions.DialOptions = append(
			options.ConnectionOptions.DialOptions,
			grpc.WithChainUnaryInterceptor(func(
				ctx context.Context, method string, req, reply any,
				cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption,
			) error {
				ctx = metadata.AppendToOutgoingContext(ctx, "temporal-cloud-api-version", options.Version)
				return invoker(ctx, method, req, reply, cc, opts...)
			}),
		)
	}
	if options.Credentials != nil {
		if err := options.Credentials.applyToOptions(&options.ConnectionOptions); err != nil {
			return nil, err
		}
	}
	if options.ConnectionOptions.TLS == nil && !options.DisableTLS {
		options.ConnectionOptions.TLS = &tls.Config{}
	}
	// Exclude internal from retry by default
	options.ConnectionOptions.excludeInternalFromRetry = &atomic.Bool{}
	options.ConnectionOptions.excludeInternalFromRetry.Store(true)
	// TODO(cretz): Pass through context on dial
	conn, err := dial(newDialParameters(&ClientOptions{
		HostPort:          options.HostPort,
		ConnectionOptions: options.ConnectionOptions,
		MetricsHandler:    options.MetricsHandler,
		Credentials:       options.Credentials,
	}, options.ConnectionOptions.excludeInternalFromRetry))
	if err != nil {
		return nil, err
	}
	return &cloudOperationsClient{
		conn:               conn,
		logger:             options.Logger,
		cloudServiceClient: cloudservice.NewCloudServiceClient(conn),
	}, nil
}

func (op *withStartWorkflowOperationImpl) Get(ctx context.Context) (WorkflowRun, error) {
	select {
	case <-op.doneCh:
		return op.workflowRun, op.err
	case <-ctx.Done():
		if !op.executed.Load() {
			return nil, fmt.Errorf("%w: %w", ctx.Err(), fmt.Errorf("operation was not executed"))
		}
		return nil, ctx.Err()
	}
}

func (op *withStartWorkflowOperationImpl) markExecuted() error {
	if op.executed.Swap(true) {
		return fmt.Errorf("was already executed")
	}
	return nil
}

func (op *withStartWorkflowOperationImpl) set(workflowRun WorkflowRun, err error) {
	op.workflowRun = workflowRun
	op.err = err
	close(op.doneCh)
}

// NewNamespaceClient creates an instance of a namespace client, to manager lifecycle of namespaces.
//
// Exposed as: [go.temporal.io/sdk/client.NewNamespaceClient]
func NewNamespaceClient(options ClientOptions) (NamespaceClient, error) {
	// Initialize root tags
	if options.MetricsHandler == nil {
		options.MetricsHandler = metrics.NopHandler
	}
	options.MetricsHandler = options.MetricsHandler.WithTags(metrics.RootTags(metrics.NoneTagValue))

	if options.HostPort == "" {
		options.HostPort = LocalHostPort
	}

	connection, err := dial(newDialParameters(&options, nil))
	if err != nil {
		return nil, err
	}

	return newNamespaceServiceClient(workflowservice.NewWorkflowServiceClient(connection), connection, options), nil
}

func newNamespaceServiceClient(workflowServiceClient workflowservice.WorkflowServiceClient, clientConn *grpc.ClientConn, options ClientOptions) NamespaceClient {
	if options.Identity == "" {
		options.Identity = getWorkerIdentity("")
	}

	return &namespaceClient{
		workflowService:  workflowServiceClient,
		connectionCloser: clientConn,
		metricsHandler:   options.MetricsHandler,
		logger:           options.Logger,
		identity:         options.Identity,
	}
}

// NewValue creates a new converter.EncodedValue which can be used to decode binary data returned by Temporal.  For example:
// User had Activity.RecordHeartbeat(ctx, "my-heartbeat") and then got response from calling Client.DescribeWorkflowExecution.
// The response contains binary field PendingActivityInfo.HeartbeatDetails,
// which can be decoded by using:
//
//	var result string // This need to be same type as the one passed to RecordHeartbeat
//	NewValue(data).Get(&result)
//
// Exposed as: [go.temporal.io/sdk/client.NewValue]
func NewValue(data *commonpb.Payloads) converter.EncodedValue {
	return newEncodedValue(data, nil)
}

// NewValues creates a new converter.EncodedValues which can be used to decode binary data returned by Temporal. For example:
// User had Activity.RecordHeartbeat(ctx, "my-heartbeat", 123) and then got response from calling Client.DescribeWorkflowExecution.
// The response contains binary field PendingActivityInfo.HeartbeatDetails,
// which can be decoded by using:
//
//	var result1 string
//	var result2 int // These need to be same type as those arguments passed to RecordHeartbeat
//	NewValues(data).Get(&result1, &result2)
//
// Exposed as: [go.temporal.io/sdk/client.NewValues]
func NewValues(data *commonpb.Payloads) converter.EncodedValues {
	return newEncodedValues(data, nil)
}

type apiKeyCredentials func(context.Context) (string, error)

// Exposed as: [go.temporal.io/sdk/client.NewAPIKeyStaticCredentials]
func NewAPIKeyStaticCredentials(apiKey string) Credentials {
	return NewAPIKeyDynamicCredentials(func(ctx context.Context) (string, error) { return apiKey, nil })
}

// Exposed as: [go.temporal.io/sdk/client.NewAPIKeyDynamicCredentials]
func NewAPIKeyDynamicCredentials(apiKeyCallback func(context.Context) (string, error)) Credentials {
	return apiKeyCredentials(apiKeyCallback)
}

func (apiKeyCredentials) applyToOptions(*ConnectionOptions) error { return nil }

func (a apiKeyCredentials) gRPCInterceptor() grpc.UnaryClientInterceptor { return a.gRPCIntercept }

func (a apiKeyCredentials) gRPCIntercept(
	ctx context.Context,
	method string,
	req any,
	reply any,
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	if apiKey, err := a(ctx); err != nil {
		return err
	} else if apiKey != "" {
		// Only add API key if it doesn't already exist
		if md, _ := metadata.FromOutgoingContext(ctx); len(md.Get("authorization")) == 0 {
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+apiKey)
		}
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}

type mTLSCredentials tls.Certificate

// Exposed as: [go.temporal.io/sdk/client.NewMTLSCredentials]
func NewMTLSCredentials(certificate tls.Certificate) Credentials { return mTLSCredentials(certificate) }

func (m mTLSCredentials) applyToOptions(opts *ConnectionOptions) error {
	if opts.TLS == nil {
		opts.TLS = &tls.Config{}
	} else if len(opts.TLS.Certificates) != 0 {
		return fmt.Errorf("cannot apply mTLS credentials, certificates already exist on TLS options")
	}
	opts.TLS.Certificates = append(opts.TLS.Certificates, tls.Certificate(m))
	return nil
}

func (mTLSCredentials) gRPCInterceptor() grpc.UnaryClientInterceptor { return nil }

// WorkflowUpdateServiceTimeoutOrCanceledError is an error that occurs when an update call times out or is cancelled.
//
// Note, this is not related to any general concept of timing out or cancelling a running update, this is only related to the client call itself.
//
// Exposed as: [go.temporal.io/sdk/client.WorkflowUpdateServiceTimeoutOrCanceledError]
type WorkflowUpdateServiceTimeoutOrCanceledError struct {
	cause error
}

// NewWorkflowUpdateServiceTimeoutOrCanceledError creates a new WorkflowUpdateServiceTimeoutOrCanceledError.
//
// Exposed as: [go.temporal.io/sdk/client.NewWorkflowUpdateServiceTimeoutOrCanceledError]
func NewWorkflowUpdateServiceTimeoutOrCanceledError(err error) *WorkflowUpdateServiceTimeoutOrCanceledError {
	return &WorkflowUpdateServiceTimeoutOrCanceledError{
		cause: err,
	}
}

func (e *WorkflowUpdateServiceTimeoutOrCanceledError) Error() string {
	return fmt.Sprintf("Timeout or cancellation waiting for update: %v", e.cause)
}

func (e *WorkflowUpdateServiceTimeoutOrCanceledError) Unwrap() error { return e.cause }

// SetRequestIDOnStartWorkflowOptions is an internal only method for setting a requestID on StartWorkflowOptions.
// RequestID is purposefully not exposed to users for the time being.
func SetRequestIDOnStartWorkflowOptions(opts *StartWorkflowOptions, requestID string) {
	opts.requestID = requestID
}

// SetCallbacksOnStartWorkflowOptions is an internal only method for setting callbacks on StartWorkflowOptions.
// Callbacks are purposefully not exposed to users for the time being.
func SetCallbacksOnStartWorkflowOptions(opts *StartWorkflowOptions, callbacks []*commonpb.Callback) {
	opts.callbacks = callbacks
}

// SetLinksOnStartWorkflowOptions is an internal only method for setting links on StartWorkflowOptions.
// Links are purposefully not exposed to users for the time being.
func SetLinksOnStartWorkflowOptions(opts *StartWorkflowOptions, links []*commonpb.Link) {
	opts.links = links
}
