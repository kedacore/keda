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
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pborman/uuid"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/operatorservice/v1"
	querypb "go.temporal.io/api/query/v1"
	"go.temporal.io/api/serviceerror"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	updatepb "go.temporal.io/api/update/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/internal/common/retry"
	"go.temporal.io/sdk/internal/common/serializer"
	"go.temporal.io/sdk/internal/common/util"
	"go.temporal.io/sdk/log"
	uberatomic "go.uber.org/atomic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// Assert that structs do indeed implement the interfaces
var (
	_ Client          = (*WorkflowClient)(nil)
	_ NamespaceClient = (*namespaceClient)(nil)
)

const (
	defaultGetHistoryTimeout = 65 * time.Second

	getSystemInfoTimeout = 5 * time.Second

	pollUpdateTimeout = 60 * time.Second
)

var maxListArchivedWorkflowTimeout = time.Minute * 3

type (
	// WorkflowClient is the client for starting a workflow execution.
	WorkflowClient struct {
		workflowService          workflowservice.WorkflowServiceClient
		conn                     *grpc.ClientConn
		namespace                string
		registry                 *registry
		logger                   log.Logger
		metricsHandler           metrics.Handler
		identity                 string
		dataConverter            converter.DataConverter
		failureConverter         converter.FailureConverter
		contextPropagators       []ContextPropagator
		workerInterceptors       []WorkerInterceptor
		interceptor              ClientOutboundInterceptor
		excludeInternalFromRetry *uberatomic.Bool
		capabilities             *workflowservice.GetSystemInfoResponse_Capabilities
		capabilitiesLock         sync.RWMutex

		// The pointer value is shared across multiple clients. If non-nil, only
		// access/mutate atomically.
		unclosedClients *int32
	}

	// namespaceClient is the client for managing namespaces.
	namespaceClient struct {
		workflowService  workflowservice.WorkflowServiceClient
		connectionCloser io.Closer
		metricsHandler   metrics.Handler
		logger           log.Logger
		identity         string
	}

	// WorkflowRun represents a started non child workflow
	WorkflowRun interface {
		// GetID return workflow ID, which will be same as StartWorkflowOptions.ID if provided.
		GetID() string

		// GetRunID return the first started workflow run ID (please see below) -
		// empty string if no such run. Note, this value may change after Get is
		// called if there was a later run for this run.
		GetRunID() string

		// Get will fill the workflow execution result to valuePtr, if workflow
		// execution is a success, or return corresponding error. This is a blocking
		// API.
		//
		// This call will follow execution runs to the latest result for this run
		// instead of strictly returning this run's result. This means that if the
		// workflow returned ContinueAsNewError, has a more recent cron execution,
		// or has a new run ID on failure (i.e. a retry), this will wait and return
		// the result for the latest run in the chain. To strictly get the result
		// for this run without following to the latest, use GetWithOptions and set
		// the DisableFollowingRuns option to true.
		//
		// Note, values should not be reused for extraction here because merging on
		// top of existing values may result in unexpected behavior similar to
		// json.Unmarshal.
		Get(ctx context.Context, valuePtr interface{}) error

		// GetWithOptions will fill the workflow execution result to valuePtr, if
		// workflow execution is a success, or return corresponding error. This is a
		// blocking API.
		//
		// Note, values should not be reused for extraction here because merging on
		// top of existing values may result in unexpected behavior similar to
		// json.Unmarshal.
		GetWithOptions(ctx context.Context, valuePtr interface{}, options WorkflowRunGetOptions) error
	}

	// WorkflowRunGetOptions are options for WorkflowRun.GetWithOptions.
	WorkflowRunGetOptions struct {
		// DisableFollowingRuns, if true, will not follow execution chains to the
		// latest run. By default when this is false, getting the result of a
		// workflow may not use the literal run ID but instead follow to later runs
		// if the workflow returned a ContinueAsNewError, has a later cron, or is
		// retried on failure.
		DisableFollowingRuns bool
	}

	// workflowRunImpl is an implementation of WorkflowRun
	workflowRunImpl struct {
		workflowType     string
		workflowID       string
		firstRunID       string
		currentRunID     *util.OnceCell
		iterFn           func(ctx context.Context, runID string) HistoryEventIterator
		dataConverter    converter.DataConverter
		failureConverter converter.FailureConverter
		registry         *registry
	}

	// HistoryEventIterator represents the interface for
	// history event iterator
	HistoryEventIterator interface {
		// HasNext return whether this iterator has next value
		HasNext() bool
		// Next returns the next history events and error
		// The errors it can return:
		//	- serviceerror.NotFound
		//	- serviceerror.InvalidArgument
		//	- serviceerror.Internal
		//	- serviceerror.Unavailable
		Next() (*historypb.HistoryEvent, error)
	}

	// historyEventIteratorImpl is the implementation of HistoryEventIterator
	historyEventIteratorImpl struct {
		// whether this iterator is initialized
		initialized bool
		// local cached history events and corresponding consuming index
		nextEventIndex int
		events         []*historypb.HistoryEvent
		// token to get next page of history events
		nexttoken []byte
		// err when getting next page of history events
		err error
		// func which use a next token to get next page of history events
		paginate func(nexttoken []byte) (*workflowservice.GetWorkflowExecutionHistoryResponse, error)
	}
)

// ExecuteWorkflow starts a workflow execution and returns a WorkflowRun that will allow you to wait until this workflow
// reaches the end state, such as workflow finished successfully or timeout.
// The user can use this to start using a functor like below and get the workflow execution result, as EncodedValue
// Either by
//
//	ExecuteWorkflow(options, "workflowTypeName", arg1, arg2, arg3)
//	or
//	ExecuteWorkflow(options, workflowExecuteFn, arg1, arg2, arg3)
//
// The current timeout resolution implementation is in seconds and uses math.Ceil(d.Seconds()) as the duration. But is
// subjected to change in the future.
// NOTE: the context.Context should have a fairly large timeout, since workflow execution may take a while to be finished
func (wc *WorkflowClient) ExecuteWorkflow(ctx context.Context, options StartWorkflowOptions, workflow interface{}, args ...interface{}) (WorkflowRun, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}

	// Default workflow ID
	if options.ID == "" {
		options.ID = uuid.New()
	}

	// Validate function and get name
	if err := validateFunctionArgs(workflow, args, true); err != nil {
		return nil, err
	}
	workflowType, err := getWorkflowFunctionName(wc.registry, workflow)
	if err != nil {
		return nil, err
	}

	// Set header before interceptor run
	ctx = contextWithNewHeader(ctx)

	// Run via interceptor
	return wc.interceptor.ExecuteWorkflow(ctx, &ClientExecuteWorkflowInput{
		Options:      &options,
		WorkflowType: workflowType,
		Args:         args,
	})
}

// GetWorkflow gets a workflow execution and returns a WorkflowRun that will allow you to wait until this workflow
// reaches the end state, such as workflow finished successfully or timeout.
// The current timeout resolution implementation is in seconds and uses math.Ceil(d.Seconds()) as the duration. But is
// subjected to change in the future.
func (wc *WorkflowClient) GetWorkflow(ctx context.Context, workflowID string, runID string) WorkflowRun {
	// We intentionally don't "ensureIntialized" here because there is no direct
	// error return path. Rather we let GetWorkflowHistory do it.

	iterFn := func(fnCtx context.Context, fnRunID string) HistoryEventIterator {
		return wc.GetWorkflowHistory(fnCtx, workflowID, fnRunID, true, enumspb.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT)
	}

	// The ID may not actually have been set - if not, we have to (lazily) ask the server for info about the workflow
	// execution and extract run id from there. This is definitely less efficient than it could be if there was a more
	// specific rpc method for this, or if there were more granular history filters - in which case it could be
	// extracted from the `iterFn` inside of `workflowRunImpl`
	var runIDCell util.OnceCell
	if runID == "" {
		fetcher := func() string {
			execData, _ := wc.DescribeWorkflowExecution(ctx, workflowID, runID)
			wei := execData.GetWorkflowExecutionInfo()
			if wei != nil {
				execution := wei.GetExecution()
				if execution != nil {
					return execution.RunId
				}
			}
			return ""
		}
		runIDCell = util.LazyOnceCell(fetcher)
	} else {
		runIDCell = util.PopulatedOnceCell(runID)
	}

	return &workflowRunImpl{
		workflowID:       workflowID,
		firstRunID:       runID,
		currentRunID:     &runIDCell,
		iterFn:           iterFn,
		dataConverter:    wc.dataConverter,
		failureConverter: wc.failureConverter,
		registry:         wc.registry,
	}
}

// SignalWorkflow signals a workflow in execution.
func (wc *WorkflowClient) SignalWorkflow(ctx context.Context, workflowID string, runID string, signalName string, arg interface{}) error {
	if err := wc.ensureInitialized(); err != nil {
		return err
	}

	// Set header before interceptor run
	ctx = contextWithNewHeader(ctx)

	return wc.interceptor.SignalWorkflow(ctx, &ClientSignalWorkflowInput{
		WorkflowID: workflowID,
		RunID:      runID,
		SignalName: signalName,
		Arg:        arg,
	})
}

// SignalWithStartWorkflow sends a signal to a running workflow.
// If the workflow is not running or not found, it starts the workflow and then sends the signal in transaction.
func (wc *WorkflowClient) SignalWithStartWorkflow(ctx context.Context, workflowID string, signalName string, signalArg interface{},
	options StartWorkflowOptions, workflowFunc interface{}, workflowArgs ...interface{},
) (WorkflowRun, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}

	// Due to the ambiguous way to provide workflow IDs, if options contains an
	// ID, it must match the parameter
	if options.ID != "" && options.ID != workflowID {
		return nil, fmt.Errorf("workflow ID from options not used, must be unset or match workflow ID parameter")
	}

	// Default workflow ID to UUID
	options.ID = workflowID
	if options.ID == "" {
		options.ID = uuid.New()
	}

	// Validate function and get name
	if err := validateFunctionArgs(workflowFunc, workflowArgs, true); err != nil {
		return nil, err
	}
	workflowType, err := getWorkflowFunctionName(wc.registry, workflowFunc)
	if err != nil {
		return nil, err
	}

	// Set header before interceptor run
	ctx = contextWithNewHeader(ctx)

	// Run via interceptor
	return wc.interceptor.SignalWithStartWorkflow(ctx, &ClientSignalWithStartWorkflowInput{
		SignalName:   signalName,
		SignalArg:    signalArg,
		Options:      &options,
		WorkflowType: workflowType,
		Args:         workflowArgs,
	})
}

// CancelWorkflow cancels a workflow in execution.  It allows workflow to properly clean up and gracefully close.
// workflowID is required, other parameters are optional.
// If runID is omit, it will terminate currently running workflow (if there is one) based on the workflowID.
func (wc *WorkflowClient) CancelWorkflow(ctx context.Context, workflowID string, runID string) error {
	if err := wc.ensureInitialized(); err != nil {
		return err
	}

	return wc.interceptor.CancelWorkflow(ctx, &ClientCancelWorkflowInput{WorkflowID: workflowID, RunID: runID})
}

// TerminateWorkflow terminates a workflow execution.
// workflowID is required, other parameters are optional.
// If runID is omit, it will terminate currently running workflow (if there is one) based on the workflowID.
func (wc *WorkflowClient) TerminateWorkflow(ctx context.Context, workflowID string, runID string, reason string, details ...interface{}) error {
	if err := wc.ensureInitialized(); err != nil {
		return err
	}

	return wc.interceptor.TerminateWorkflow(ctx, &ClientTerminateWorkflowInput{
		WorkflowID: workflowID,
		RunID:      runID,
		Reason:     reason,
		Details:    details,
	})
}

// GetWorkflowHistory return a channel which contains the history events of a given workflow
func (wc *WorkflowClient) GetWorkflowHistory(
	ctx context.Context,
	workflowID string,
	runID string,
	isLongPoll bool,
	filterType enumspb.HistoryEventFilterType,
) HistoryEventIterator {
	return wc.getWorkflowHistory(ctx, workflowID, runID, isLongPoll, filterType, wc.metricsHandler)
}

func (wc *WorkflowClient) getWorkflowHistory(
	ctx context.Context,
	workflowID string,
	runID string,
	isLongPoll bool,
	filterType enumspb.HistoryEventFilterType,
	rpcMetricsHandler metrics.Handler,
) HistoryEventIterator {
	namespace := wc.namespace
	paginate := func(nextToken []byte) (*workflowservice.GetWorkflowExecutionHistoryResponse, error) {
		request := &workflowservice.GetWorkflowExecutionHistoryRequest{
			Namespace: namespace,
			Execution: &commonpb.WorkflowExecution{
				WorkflowId: workflowID,
				RunId:      runID,
			},
			WaitNewEvent:           isLongPoll,
			HistoryEventFilterType: filterType,
			NextPageToken:          nextToken,
			SkipArchival:           isLongPoll,
		}

		var response *workflowservice.GetWorkflowExecutionHistoryResponse
		var err error
	Loop:
		for {
			response, err = wc.getWorkflowExecutionHistory(ctx, rpcMetricsHandler, isLongPoll, request, filterType)
			if err != nil {
				return nil, err
			}
			if isLongPoll && len(response.History.Events) == 0 && len(response.NextPageToken) != 0 {
				request.NextPageToken = response.NextPageToken
				continue Loop
			}
			break Loop
		}
		return response, nil
	}

	return &historyEventIteratorImpl{
		paginate: paginate,
	}
}

func (wc *WorkflowClient) getWorkflowExecutionHistory(ctx context.Context, rpcMetricsHandler metrics.Handler, isLongPoll bool,
	request *workflowservice.GetWorkflowExecutionHistoryRequest, filterType enumspb.HistoryEventFilterType,
) (*workflowservice.GetWorkflowExecutionHistoryResponse, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}

	grpcCtx, cancel := newGRPCContext(ctx, grpcMetricsHandler(rpcMetricsHandler), grpcLongPoll(isLongPoll), defaultGrpcRetryParameters(ctx), func(builder *grpcContextBuilder) {
		if isLongPoll {
			builder.Timeout = defaultGetHistoryTimeout
		}
	})

	defer cancel()
	response, err := wc.workflowService.GetWorkflowExecutionHistory(grpcCtx, request)
	if err != nil {
		return nil, err
	}

	if response.RawHistory != nil {
		history, err := serializer.DeserializeBlobDataToHistoryEvents(response.RawHistory, filterType)
		if err != nil {
			return nil, err
		}
		response.History = history
	}
	return response, err
}

// CompleteActivity reports activity completed. activity Execute method can return activity.ErrResultPending to
// indicate the activity is not completed when it's Execute method returns. In that case, this CompleteActivity() method
// should be called when that activity is completed with the actual result and error. If err is nil, activity task
// completed event will be reported; if err is CanceledError, activity task canceled event will be reported; otherwise,
// activity task failed event will be reported.
func (wc *WorkflowClient) CompleteActivity(ctx context.Context, taskToken []byte, result interface{}, err error) error {
	if err := wc.ensureInitialized(); err != nil {
		return err
	}

	if taskToken == nil {
		return errors.New("invalid task token provided")
	}

	dataConverter := WithContext(ctx, wc.dataConverter)
	var data *commonpb.Payloads
	if result != nil {
		var err0 error
		data, err0 = encodeArg(dataConverter, result)
		if err0 != nil {
			return err0
		}
	}

	// We do allow canceled error to be passed here
	cancelAllowed := true
	request := convertActivityResultToRespondRequest(wc.identity, taskToken,
		data, err, wc.dataConverter, wc.failureConverter, wc.namespace, cancelAllowed, nil)
	return reportActivityComplete(ctx, wc.workflowService, request, wc.metricsHandler)
}

// CompleteActivityByID reports activity completed. Similar to CompleteActivity
// It takes namespace name, workflowID, runID, activityID as arguments.
func (wc *WorkflowClient) CompleteActivityByID(ctx context.Context, namespace, workflowID, runID, activityID string,
	result interface{}, err error,
) error {
	if activityID == "" || workflowID == "" || namespace == "" {
		return errors.New("empty activity or workflow id or namespace")
	}

	dataConverter := WithContext(ctx, wc.dataConverter)
	var data *commonpb.Payloads
	if result != nil {
		var err0 error
		data, err0 = encodeArg(dataConverter, result)
		if err0 != nil {
			return err0
		}
	}

	// We do allow canceled error to be passed here
	cancelAllowed := true
	request := convertActivityResultToRespondRequestByID(wc.identity, namespace, workflowID, runID, activityID,
		data, err, wc.dataConverter, wc.failureConverter, cancelAllowed)
	return reportActivityCompleteByID(ctx, wc.workflowService, request, wc.metricsHandler)
}

// RecordActivityHeartbeat records heartbeat for an activity.
func (wc *WorkflowClient) RecordActivityHeartbeat(ctx context.Context, taskToken []byte, details ...interface{}) error {
	if err := wc.ensureInitialized(); err != nil {
		return err
	}

	dataConverter := WithContext(ctx, wc.dataConverter)
	data, err := encodeArgs(dataConverter, details)
	if err != nil {
		return err
	}
	return recordActivityHeartbeat(ctx, wc.workflowService, wc.metricsHandler, wc.identity, taskToken, data)
}

// RecordActivityHeartbeatByID records heartbeat for an activity.
func (wc *WorkflowClient) RecordActivityHeartbeatByID(ctx context.Context,
	namespace, workflowID, runID, activityID string, details ...interface{},
) error {
	if err := wc.ensureInitialized(); err != nil {
		return err
	}

	dataConverter := WithContext(ctx, wc.dataConverter)
	data, err := encodeArgs(dataConverter, details)
	if err != nil {
		return err
	}
	return recordActivityHeartbeatByID(ctx, wc.workflowService, wc.metricsHandler, wc.identity, namespace, workflowID, runID, activityID, data)
}

// ListClosedWorkflow gets closed workflow executions based on request filters
// The errors it can throw:
//   - serviceerror.InvalidArgument
//   - serviceerror.Internal
//   - serviceerror.Unavailable
//   - serviceerror.NamespaceNotFound
func (wc *WorkflowClient) ListClosedWorkflow(ctx context.Context, request *workflowservice.ListClosedWorkflowExecutionsRequest) (*workflowservice.ListClosedWorkflowExecutionsResponse, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}

	if request.GetNamespace() == "" {
		request.Namespace = wc.namespace
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	response, err := wc.workflowService.ListClosedWorkflowExecutions(grpcCtx, request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// ListOpenWorkflow gets open workflow executions based on request filters
// The errors it can throw:
//   - serviceerror.InvalidArgument
//   - serviceerror.Internal
//   - serviceerror.Unavailable
//   - serviceerror.NamespaceNotFound
func (wc *WorkflowClient) ListOpenWorkflow(ctx context.Context, request *workflowservice.ListOpenWorkflowExecutionsRequest) (*workflowservice.ListOpenWorkflowExecutionsResponse, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}

	if request.GetNamespace() == "" {
		request.Namespace = wc.namespace
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	response, err := wc.workflowService.ListOpenWorkflowExecutions(grpcCtx, request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// ListWorkflow implementation
func (wc *WorkflowClient) ListWorkflow(ctx context.Context, request *workflowservice.ListWorkflowExecutionsRequest) (*workflowservice.ListWorkflowExecutionsResponse, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}

	if request.GetNamespace() == "" {
		request.Namespace = wc.namespace
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	response, err := wc.workflowService.ListWorkflowExecutions(grpcCtx, request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// ListArchivedWorkflow implementation
func (wc *WorkflowClient) ListArchivedWorkflow(ctx context.Context, request *workflowservice.ListArchivedWorkflowExecutionsRequest) (*workflowservice.ListArchivedWorkflowExecutionsResponse, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}

	if request.GetNamespace() == "" {
		request.Namespace = wc.namespace
	}
	timeout := maxListArchivedWorkflowTimeout
	now := time.Now()
	if ctx != nil {
		if expiration, ok := ctx.Deadline(); ok && expiration.After(now) {
			timeout = expiration.Sub(now)
			if timeout > maxListArchivedWorkflowTimeout {
				timeout = maxListArchivedWorkflowTimeout
			} else if timeout < minRPCTimeout {
				timeout = minRPCTimeout
			}
		}
	}
	grpcCtx, cancel := newGRPCContext(ctx, grpcTimeout(timeout), defaultGrpcRetryParameters(ctx))
	defer cancel()
	response, err := wc.workflowService.ListArchivedWorkflowExecutions(grpcCtx, request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// ScanWorkflow implementation
func (wc *WorkflowClient) ScanWorkflow(ctx context.Context, request *workflowservice.ScanWorkflowExecutionsRequest) (*workflowservice.ScanWorkflowExecutionsResponse, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}

	if request.GetNamespace() == "" {
		request.Namespace = wc.namespace
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	response, err := wc.workflowService.ScanWorkflowExecutions(grpcCtx, request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// CountWorkflow implementation
func (wc *WorkflowClient) CountWorkflow(ctx context.Context, request *workflowservice.CountWorkflowExecutionsRequest) (*workflowservice.CountWorkflowExecutionsResponse, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}

	if request.GetNamespace() == "" {
		request.Namespace = wc.namespace
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	response, err := wc.workflowService.CountWorkflowExecutions(grpcCtx, request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetSearchAttributes implementation
func (wc *WorkflowClient) GetSearchAttributes(ctx context.Context) (*workflowservice.GetSearchAttributesResponse, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}

	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	response, err := wc.workflowService.GetSearchAttributes(grpcCtx, &workflowservice.GetSearchAttributesRequest{})
	if err != nil {
		return nil, err
	}
	return response, nil
}

// DescribeWorkflowExecution returns information about the specified workflow execution.
// The errors it can return:
//   - serviceerror.InvalidArgument
//   - serviceerror.Internal
//   - serviceerror.Unavailable
//   - serviceerror.NotFound
func (wc *WorkflowClient) DescribeWorkflowExecution(ctx context.Context, workflowID, runID string) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}

	request := &workflowservice.DescribeWorkflowExecutionRequest{
		Namespace: wc.namespace,
		Execution: &commonpb.WorkflowExecution{
			WorkflowId: workflowID,
			RunId:      runID,
		},
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	response, err := wc.workflowService.DescribeWorkflowExecution(grpcCtx, request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// QueryWorkflow queries a given workflow execution
// workflowID and queryType are required, other parameters are optional.
// - workflow ID of the workflow.
// - runID can be default(empty string). if empty string then it will pick the running execution of that workflow ID.
// - taskQueue can be default(empty string). If empty string then it will pick the taskQueue of the running execution of that workflow ID.
// - queryType is the type of the query.
// - args... are the optional query parameters.
// The errors it can return:
//   - serviceerror.InvalidArgument
//   - serviceerror.Internal
//   - serviceerror.Unavailable
//   - serviceerror.NotFound
//   - serviceerror.QueryFailed
func (wc *WorkflowClient) QueryWorkflow(ctx context.Context, workflowID string, runID string, queryType string, args ...interface{}) (converter.EncodedValue, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}

	// Set header before interceptor run
	ctx = contextWithNewHeader(ctx)

	return wc.interceptor.QueryWorkflow(ctx, &ClientQueryWorkflowInput{
		WorkflowID: workflowID,
		RunID:      runID,
		QueryType:  queryType,
		Args:       args,
	})
}

// UpdateWorkflowWithOptionsRequest is the request to UpdateWorkflowWithOptions
type UpdateWorkflowWithOptionsRequest struct {
	// UpdateID is an application-layer identifier for the requested update. It
	// must be unique within the scope of a Namespace+WorkflowID+RunID.
	UpdateID string

	// WorkflowID is a required field indicating the workflow which should be
	// updated.
	WorkflowID string

	// RunID is an optional field used to identify a specific run of the target
	// workflow.  If RunID is not provided the latest run will be used.
	RunID string

	// UpdateName is a required field which specifies the update you want to run.
	// See comments at workflow.SetUpdateHandler(ctx Context, updateName string, handler interface{}, opts UpdateHandlerOptions)
	// for more details on how to setup update handlers within the target workflow.
	UpdateName string

	// Args is an optional field used to identify the arguments passed to the
	// update.
	Args []interface{}

	// FirstExecutionRunID specifies the RunID expected to identify the first
	// run in the workflow execution chain. If this expectation does not match
	// then the server will reject the update request with an error.
	FirstExecutionRunID string

	// How this RPC should block on the server before returning.
	WaitPolicy *updatepb.WaitPolicy
}

// WorkflowUpdateHandle is a handle to a workflow execution update process. The
// update may or may not have completed so an instance of this type functions
// simlar to a Future with respect to the outcome of the update. If the update
// is rejected or returns an error, the Get function on this type will return
// that error through the output valuePtr.
// NOTE: Experimental
type WorkflowUpdateHandle interface {
	// WorkflowID observes the update's workflow ID.
	WorkflowID() string

	// RunID observes the update's run ID.
	RunID() string

	// UpdateID observes the update's ID.
	UpdateID() string

	// Get blocks on the outcome of the update.
	Get(ctx context.Context, valuePtr interface{}) error
}

// GetWorkflowUpdateHandleOptions encapsulates the parameters needed to unambiguously
// refer to a Workflow Update.
// NOTE: Experimental
type GetWorkflowUpdateHandleOptions struct {
	// WorkflowID of the target update
	WorkflowID string

	// RunID of the target workflow. If blank, use the most recent run
	RunID string

	// UpdateID of the target update
	UpdateID string
}

type baseUpdateHandle struct {
	ref *updatepb.UpdateRef
}

// completedUpdateHandle is an UpdateHandle impelementation for use when the outcome
// of the update is already known and the Get call can return immediately.
type completedUpdateHandle struct {
	baseUpdateHandle
	value converter.EncodedValue
	err   error
}

// lazyUpdateHandle represents and update that is not known to have completed
// yet (i.e. the associated updatepb.Outcome is not known) and thus calling Get
// will poll the server for the outcome.
type lazyUpdateHandle struct {
	baseUpdateHandle
	client *WorkflowClient
}

// QueryWorkflowWithOptionsRequest is the request to QueryWorkflowWithOptions
type QueryWorkflowWithOptionsRequest struct {
	// WorkflowID is a required field indicating the workflow which should be queried.
	WorkflowID string

	// RunID is an optional field used to identify a specific run of the queried workflow.
	// If RunID is not provided the latest run will be used.
	RunID string

	// QueryType is a required field which specifies the query you want to run.
	// By default, temporal supports "__stack_trace" as a standard query type, which will return string value
	// representing the call stack of the target workflow. The target workflow could also setup different query handler to handle custom query types.
	// See comments at workflow.SetQueryHandler(ctx Context, queryType string, handler interface{}) for more details on how to setup query handler within the target workflow.
	QueryType string

	// Args is an optional field used to identify the arguments passed to the query.
	Args []interface{}

	// QueryRejectCondition is an optional field used to reject queries based on workflow state.
	// QUERY_REJECT_CONDITION_NONE indicates that query should not be rejected.
	// QUERY_REJECT_CONDITION_NOT_OPEN indicates that query should be rejected if workflow is not open.
	// QUERY_REJECT_CONDITION_NOT_COMPLETED_CLEANLY indicates that query should be rejected if workflow did not complete cleanly (e.g. terminated, canceled timeout etc...).
	QueryRejectCondition enumspb.QueryRejectCondition

	// Header is an optional header to include with the query.
	Header *commonpb.Header
}

// QueryWorkflowWithOptionsResponse is the response to QueryWorkflowWithOptions
type QueryWorkflowWithOptionsResponse struct {
	// QueryResult contains the result of executing the query.
	// This will only be set if the query was completed successfully and not rejected.
	QueryResult converter.EncodedValue

	// QueryRejected contains information about the query rejection.
	QueryRejected *querypb.QueryRejected
}

// QueryWorkflowWithOptions queries a given workflow execution and returns the query result synchronously.
// See QueryWorkflowWithOptionsRequest and QueryWorkflowWithOptionsResult for more information.
// The errors it can return:
//   - serviceerror.InvalidArgument
//   - serviceerror.Internal
//   - serviceerror.Unavailable
//   - serviceerror.NotFound
//   - serviceerror.QueryFailed
func (wc *WorkflowClient) QueryWorkflowWithOptions(ctx context.Context, request *QueryWorkflowWithOptionsRequest) (*QueryWorkflowWithOptionsResponse, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}

	var input *commonpb.Payloads
	if len(request.Args) > 0 {
		var err error
		if input, err = encodeArgs(wc.dataConverter, request.Args); err != nil {
			return nil, err
		}
	}
	req := &workflowservice.QueryWorkflowRequest{
		Namespace: wc.namespace,
		Execution: &commonpb.WorkflowExecution{
			WorkflowId: request.WorkflowID,
			RunId:      request.RunID,
		},
		Query: &querypb.WorkflowQuery{
			QueryType: request.QueryType,
			QueryArgs: input,
			Header:    request.Header,
		},
		QueryRejectCondition: request.QueryRejectCondition,
	}

	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	resp, err := wc.workflowService.QueryWorkflow(grpcCtx, req)
	if err != nil {
		return nil, err
	}

	if resp.QueryRejected != nil {
		return &QueryWorkflowWithOptionsResponse{
			QueryRejected: resp.QueryRejected,
			QueryResult:   nil,
		}, nil
	}
	return &QueryWorkflowWithOptionsResponse{
		QueryRejected: nil,
		QueryResult:   newEncodedValue(resp.QueryResult, wc.dataConverter),
	}, nil
}

// DescribeTaskQueue returns information about the target taskqueue, right now this API returns the
// pollers which polled this taskqueue in last few minutes.
// - taskqueue name of taskqueue
// - taskqueueType type of taskqueue, can be workflow or activity
// The errors it can return:
//   - serviceerror.InvalidArgument
//   - serviceerror.Internal
//   - serviceerror.Unavailable
//   - serviceerror.NotFound
func (wc *WorkflowClient) DescribeTaskQueue(ctx context.Context, taskQueue string, taskQueueType enumspb.TaskQueueType) (*workflowservice.DescribeTaskQueueResponse, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}

	request := &workflowservice.DescribeTaskQueueRequest{
		Namespace:     wc.namespace,
		TaskQueue:     &taskqueuepb.TaskQueue{Name: taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
		TaskQueueType: taskQueueType,
	}

	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	resp, err := wc.workflowService.DescribeTaskQueue(grpcCtx, request)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// ResetWorkflowExecution reset an existing workflow execution to WorkflowTaskFinishEventId(exclusive).
// And it will immediately terminating the current execution instance.
// RequestId is used to deduplicate requests. It will be autogenerated if not set.
func (wc *WorkflowClient) ResetWorkflowExecution(ctx context.Context, request *workflowservice.ResetWorkflowExecutionRequest) (*workflowservice.ResetWorkflowExecutionResponse, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}

	if request != nil && request.GetRequestId() == "" {
		request.RequestId = uuid.New()
	}

	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	resp, err := wc.workflowService.ResetWorkflowExecution(grpcCtx, request)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// UpdateWorkerBuildIdCompatibility allows you to update the worker-build-id based version sets for a particular
// task queue. This is used in conjunction with workers who specify their build id and thus opt into the
// feature.
func (wc *WorkflowClient) UpdateWorkerBuildIdCompatibility(ctx context.Context, options *UpdateWorkerBuildIdCompatibilityOptions) error {
	if err := wc.ensureInitialized(); err != nil {
		return err
	}

	request, err := options.validateAndConvertToProto()
	if err != nil {
		return err
	}
	request.Namespace = wc.namespace

	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	_, err = wc.workflowService.UpdateWorkerBuildIdCompatibility(grpcCtx, request)
	return err
}

// GetWorkerBuildIdCompatibility returns the worker-build-id based version sets for a particular task queue.
func (wc *WorkflowClient) GetWorkerBuildIdCompatibility(ctx context.Context, options *GetWorkerBuildIdCompatibilityOptions) (*WorkerBuildIDVersionSets, error) {
	if options.MaxSets < 0 {
		return nil, errors.New("maxDepth must be >= 0")
	}
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}

	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	request := &workflowservice.GetWorkerBuildIdCompatibilityRequest{
		Namespace: wc.namespace,
		TaskQueue: options.TaskQueue,
		MaxSets:   int32(options.MaxSets),
	}
	resp, err := wc.workflowService.GetWorkerBuildIdCompatibility(grpcCtx, request)
	if err != nil {
		return nil, err
	}
	converted := workerVersionSetsFromProtoResponse(resp)
	return converted, nil
}

func (wc *WorkflowClient) UpdateWorkflowWithOptions(
	ctx context.Context,
	req *UpdateWorkflowWithOptionsRequest,
) (WorkflowUpdateHandle, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}
	// Default update ID
	updateID := req.UpdateID
	if updateID == "" {
		updateID = uuid.New()
	}

	ctx = contextWithNewHeader(ctx)

	return wc.interceptor.UpdateWorkflow(ctx, &ClientUpdateWorkflowInput{
		UpdateID:            updateID,
		WorkflowID:          req.WorkflowID,
		UpdateName:          req.UpdateName,
		Args:                req.Args,
		RunID:               req.RunID,
		FirstExecutionRunID: req.FirstExecutionRunID,
		WaitPolicy:          req.WaitPolicy,
	})
}

func (wc *WorkflowClient) GetWorkflowUpdateHandle(ref GetWorkflowUpdateHandleOptions) WorkflowUpdateHandle {
	return &lazyUpdateHandle{
		client: wc,
		baseUpdateHandle: baseUpdateHandle{
			ref: &updatepb.UpdateRef{
				WorkflowExecution: &commonpb.WorkflowExecution{
					WorkflowId: ref.WorkflowID,
					RunId:      ref.RunID,
				},
				UpdateId: ref.UpdateID,
			},
		},
	}
}

// PollWorkflowUpdate sends a request for the outcome of the specified update
// through the interceptor chain.
func (wc *WorkflowClient) PollWorkflowUpdate(
	ctx context.Context,
	ref *updatepb.UpdateRef,
) (converter.EncodedValue, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}
	ctx = contextWithNewHeader(ctx)
	return wc.interceptor.PollWorkflowUpdate(ctx, &ClientPollWorkflowUpdateInput{
		UpdateRef: ref,
	})
}

func (wc *WorkflowClient) UpdateWorkflow(
	ctx context.Context,
	workflowID string,
	workflowRunID string,
	updateName string,
	args ...interface{},
) (WorkflowUpdateHandle, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}

	ctx = contextWithNewHeader(ctx)

	return wc.interceptor.UpdateWorkflow(ctx, &ClientUpdateWorkflowInput{
		WorkflowID: workflowID,
		UpdateName: updateName,
		UpdateID:   uuid.New(),
		Args:       args,
	})
}

// CheckHealthRequest is a request for Client.CheckHealth.
type CheckHealthRequest struct{}

// CheckHealthResponse is a response for Client.CheckHealth.
type CheckHealthResponse struct{}

// CheckHealth performs a server health check using the gRPC health check
// API. If the check fails, an error is returned.
func (wc *WorkflowClient) CheckHealth(ctx context.Context, request *CheckHealthRequest) (*CheckHealthResponse, error) {
	if err := wc.ensureInitialized(); err != nil {
		return nil, err
	}

	// Ignore request/response for now, they are empty
	resp, err := healthpb.NewHealthClient(wc.conn).Check(ctx, &healthpb.HealthCheckRequest{
		Service: "temporal.api.workflowservice.v1.WorkflowService",
	})
	if err != nil {
		return nil, fmt.Errorf("health check error: %w", err)
	} else if resp.Status != healthpb.HealthCheckResponse_SERVING {
		return nil, fmt.Errorf("health check returned unhealthy status: %v", resp.Status)
	}
	return &CheckHealthResponse{}, nil
}

// WorkflowService implements Client.WorkflowService.
func (wc *WorkflowClient) WorkflowService() workflowservice.WorkflowServiceClient {
	return wc.workflowService
}

// OperatorService implements Client.OperatorService.
func (wc *WorkflowClient) OperatorService() operatorservice.OperatorServiceClient {
	return operatorservice.NewOperatorServiceClient(wc.conn)
}

// Get capabilities, lazily fetching from server if not already obtained.
func (wc *WorkflowClient) loadCapabilities() (*workflowservice.GetSystemInfoResponse_Capabilities, error) {
	// While we want to memoize the result here, we take care not to lock during
	// the call. This means that in racy situations where this is called multiple
	// times at once, it may result in multiple calls. This is far more preferable
	// than locking on the call itself.

	wc.capabilitiesLock.RLock()
	capabilities := wc.capabilities
	wc.capabilitiesLock.RUnlock()
	if capabilities != nil {
		return capabilities, nil
	}

	// Fetch the capabilities
	ctx, cancel := context.WithTimeout(context.Background(), getSystemInfoTimeout)
	defer cancel()
	resp, err := wc.workflowService.GetSystemInfo(ctx, &workflowservice.GetSystemInfoRequest{})
	// We ignore unimplemented
	if _, isUnimplemented := err.(*serviceerror.Unimplemented); err != nil && !isUnimplemented {
		return nil, fmt.Errorf("failed reaching server: %w", err)
	}
	if resp != nil && resp.Capabilities != nil {
		capabilities = resp.Capabilities
	} else {
		capabilities = &workflowservice.GetSystemInfoResponse_Capabilities{}
	}

	// Store and return. We intentionally don't check if we're overwriting as we
	// accept last-success-wins.
	wc.capabilitiesLock.Lock()
	wc.capabilities = capabilities
	// Also set whether we exclude internal from retry
	wc.excludeInternalFromRetry.Store(capabilities.InternalErrorDifferentiation)
	wc.capabilitiesLock.Unlock()
	return capabilities, nil
}

func (wc *WorkflowClient) ensureInitialized() error {
	// Just loading the capabilities is enough
	_, err := wc.loadCapabilities()
	return err
}

// ScheduleClient implements Client.ScheduleClient.
func (wc *WorkflowClient) ScheduleClient() ScheduleClient {
	return &scheduleClient{
		workflowClient: wc,
	}
}

// Close client and clean up underlying resources.
func (wc *WorkflowClient) Close() {
	// If there's a set of unclosed clients, we have to decrement it and then
	// set it to a new pointer of max to prevent decrementing on repeated Close
	// calls to this client. If the count has not reached zero, this close call is
	// ignored.
	if wc.unclosedClients != nil {
		remainingUnclosedClients := atomic.AddInt32(wc.unclosedClients, -1)
		// Set the unclosed clients to max value so we never try this again
		var maxUnclosedClients int32 = math.MaxInt32
		wc.unclosedClients = &maxUnclosedClients
		// If there are any remaining, do not close
		if remainingUnclosedClients > 0 {
			return
		}
	}

	if wc.conn != nil {
		if err := wc.conn.Close(); err != nil {
			wc.logger.Warn("unable to close connection", tagError, err)
		}
	}
}

// Register a namespace with temporal server
// The errors it can throw:
//   - NamespaceAlreadyExistsError
//   - serviceerror.InvalidArgument
//   - serviceerror.Internal
//   - serviceerror.Unavailable
func (nc *namespaceClient) Register(ctx context.Context, request *workflowservice.RegisterNamespaceRequest) error {
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	var err error
	_, err = nc.workflowService.RegisterNamespace(grpcCtx, request)
	return err
}

// Describe a namespace. The namespace has 3 part of information
// NamespaceInfo - Which has Name, Status, Description, Owner Email
// NamespaceConfiguration - Configuration like Workflow Execution Retention Period In Days, Whether to emit metrics.
// ReplicationConfiguration - replication config like clusters and active cluster name
// The errors it can throw:
//   - serviceerror.NamespaceNotFound
//   - serviceerror.InvalidArgument
//   - serviceerror.Internal
//   - serviceerror.Unavailable
func (nc *namespaceClient) Describe(ctx context.Context, namespace string) (*workflowservice.DescribeNamespaceResponse, error) {
	request := &workflowservice.DescribeNamespaceRequest{
		Namespace: namespace,
	}

	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	response, err := nc.workflowService.DescribeNamespace(grpcCtx, request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// Update a namespace.
// The errors it can throw:
//   - serviceerror.NamespaceNotFound
//   - serviceerror.InvalidArgument
//   - serviceerror.Internal
//   - serviceerror.Unavailable
func (nc *namespaceClient) Update(ctx context.Context, request *workflowservice.UpdateNamespaceRequest) error {
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	_, err := nc.workflowService.UpdateNamespace(grpcCtx, request)
	return err
}

// Close client and clean up underlying resources.
func (nc *namespaceClient) Close() {
	if nc.connectionCloser == nil {
		return
	}
	if err := nc.connectionCloser.Close(); err != nil {
		nc.logger.Warn("unable to close connection", tagError, err)
	}
}

func (iter *historyEventIteratorImpl) HasNext() bool {
	if iter.nextEventIndex < len(iter.events) || iter.err != nil {
		return true
	} else if !iter.initialized || len(iter.nexttoken) != 0 {
		iter.initialized = true
		response, err := iter.paginate(iter.nexttoken)
		iter.nextEventIndex = 0
		if err == nil {
			iter.events = response.History.Events
			iter.nexttoken = response.NextPageToken
			iter.err = nil
		} else {
			iter.events = nil
			iter.nexttoken = nil
			iter.err = err
		}

		if iter.nextEventIndex < len(iter.events) || iter.err != nil {
			return true
		}
		return false
	}

	return false
}

func (iter *historyEventIteratorImpl) Next() (*historypb.HistoryEvent, error) {
	// if caller call the Next() when iteration is over, just return nil, nil
	if !iter.HasNext() {
		panic("HistoryEventIterator Next() called without checking HasNext()")
	}

	// we have cached events
	if iter.nextEventIndex < len(iter.events) {
		index := iter.nextEventIndex
		iter.nextEventIndex++
		return iter.events[index], nil
	} else if iter.err != nil {
		// we have err, clear that iter.err and return err
		err := iter.err
		iter.err = nil
		return nil, err
	}

	panic("HistoryEventIterator Next() should return either a history event or a err")
}

func (workflowRun *workflowRunImpl) GetRunID() string {
	return workflowRun.currentRunID.Get()
}

func (workflowRun *workflowRunImpl) GetID() string {
	return workflowRun.workflowID
}

func (workflowRun *workflowRunImpl) Get(ctx context.Context, valuePtr interface{}) error {
	return workflowRun.GetWithOptions(ctx, valuePtr, WorkflowRunGetOptions{})
}

func (workflowRun *workflowRunImpl) GetWithOptions(
	ctx context.Context,
	valuePtr interface{},
	options WorkflowRunGetOptions,
) error {
	iter := workflowRun.iterFn(ctx, workflowRun.currentRunID.Get())
	if !iter.HasNext() {
		panic("could not get last history event for workflow")
	}
	closeEvent, err := iter.Next()
	if err != nil {
		return err
	}

	switch closeEvent.GetEventType() {
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED:
		attributes := closeEvent.GetWorkflowExecutionCompletedEventAttributes()
		if !options.DisableFollowingRuns && attributes.NewExecutionRunId != "" {
			return workflowRun.follow(ctx, valuePtr, attributes.NewExecutionRunId, options)
		}
		if valuePtr == nil || attributes.Result == nil {
			return nil
		}
		rf := reflect.ValueOf(valuePtr)
		if rf.Type().Kind() != reflect.Ptr {
			return errors.New("value parameter is not a pointer")
		}
		return workflowRun.dataConverter.FromPayloads(attributes.Result, valuePtr)
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED:
		attributes := closeEvent.GetWorkflowExecutionFailedEventAttributes()
		if !options.DisableFollowingRuns && attributes.NewExecutionRunId != "" {
			return workflowRun.follow(ctx, valuePtr, attributes.NewExecutionRunId, options)
		}
		err = workflowRun.failureConverter.FailureToError(attributes.GetFailure())
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED:
		attributes := closeEvent.GetWorkflowExecutionCanceledEventAttributes()
		details := newEncodedValues(attributes.Details, workflowRun.dataConverter)
		err = NewCanceledError(details)
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_TERMINATED:
		err = newTerminatedError()
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_TIMED_OUT:
		attributes := closeEvent.GetWorkflowExecutionTimedOutEventAttributes()
		if !options.DisableFollowingRuns && attributes.NewExecutionRunId != "" {
			return workflowRun.follow(ctx, valuePtr, attributes.NewExecutionRunId, options)
		}
		err = NewTimeoutError("Workflow timeout", enumspb.TIMEOUT_TYPE_START_TO_CLOSE, nil)
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CONTINUED_AS_NEW:
		attributes := closeEvent.GetWorkflowExecutionContinuedAsNewEventAttributes()
		if !options.DisableFollowingRuns {
			return workflowRun.follow(ctx, valuePtr, attributes.NewExecutionRunId, options)
		}
		err := &ContinueAsNewError{
			WorkflowType:  &WorkflowType{Name: attributes.GetWorkflowType().GetName()},
			Input:         attributes.Input,
			Header:        attributes.Header,
			TaskQueueName: attributes.GetTaskQueue().GetName(),
		}
		if attributes.WorkflowRunTimeout != nil {
			err.WorkflowRunTimeout = *attributes.WorkflowRunTimeout
		}
		if attributes.WorkflowTaskTimeout != nil {
			err.WorkflowTaskTimeout = *attributes.WorkflowTaskTimeout
		}
		return err
	default:
		return fmt.Errorf("unexpected event type %s when handling workflow execution result", closeEvent.GetEventType())
	}

	err = NewWorkflowExecutionError(
		workflowRun.workflowID,
		workflowRun.currentRunID.Get(),
		workflowRun.workflowType,
		err)

	return err
}

// follow is used by Get to follow a chain of executions linked by NewExecutionRunId, so that Get
// doesn't return until the chain finishes. These can be ContinuedAsNew events, Completed events
// (for workflows with a cron schedule), or Failed or TimedOut events (for workflows with a retry
// policy or cron schedule).
func (workflowRun *workflowRunImpl) follow(
	ctx context.Context,
	valuePtr interface{},
	newRunID string,
	options WorkflowRunGetOptions,
) error {
	curRunID := util.PopulatedOnceCell(newRunID)
	workflowRun.currentRunID = &curRunID
	return workflowRun.GetWithOptions(ctx, valuePtr, options)
}

func getWorkflowMemo(input map[string]interface{}, dc converter.DataConverter) (*commonpb.Memo, error) {
	if input == nil {
		return nil, nil
	}

	memo := make(map[string]*commonpb.Payload)
	for k, v := range input {
		// TODO (shtin): use dc here???
		memoBytes, err := converter.GetDefaultDataConverter().ToPayload(v)
		if err != nil {
			return nil, fmt.Errorf("encode workflow memo error: %v", err.Error())
		}
		memo[k] = memoBytes
	}
	return &commonpb.Memo{Fields: memo}, nil
}

func serializeSearchAttributes(input map[string]interface{}) (*commonpb.SearchAttributes, error) {
	if input == nil {
		return nil, nil
	}

	attr := make(map[string]*commonpb.Payload)
	for k, v := range input {
		// If search attribute value is already of Payload type, then use it directly.
		// This allows to copy search attributes from workflow info to child workflow options.
		if vp, ok := v.(*commonpb.Payload); ok {
			attr[k] = vp
			continue
		}
		var err error
		attr[k], err = converter.GetDefaultDataConverter().ToPayload(v)
		if err != nil {
			return nil, fmt.Errorf("encode search attribute [%s] error: %v", k, err)
		}
	}
	return &commonpb.SearchAttributes{IndexedFields: attr}, nil
}

type workflowClientInterceptor struct{ client *WorkflowClient }

func (w *workflowClientInterceptor) ExecuteWorkflow(
	ctx context.Context,
	in *ClientExecuteWorkflowInput,
) (WorkflowRun, error) {
	// This is always set before interceptor is invoked
	workflowID := in.Options.ID
	if workflowID == "" {
		return nil, fmt.Errorf("no workflow ID in options")
	}

	executionTimeout := in.Options.WorkflowExecutionTimeout
	runTimeout := in.Options.WorkflowRunTimeout
	workflowTaskTimeout := in.Options.WorkflowTaskTimeout

	dataConverter := WithContext(ctx, w.client.dataConverter)
	if dataConverter == nil {
		dataConverter = converter.GetDefaultDataConverter()
	}

	// Encode input
	input, err := encodeArgs(dataConverter, in.Args)
	if err != nil {
		return nil, err
	}

	memo, err := getWorkflowMemo(in.Options.Memo, dataConverter)
	if err != nil {
		return nil, err
	}

	searchAttr, err := serializeSearchAttributes(in.Options.SearchAttributes)
	if err != nil {
		return nil, err
	}

	// get workflow headers from the context
	header, err := headerPropagated(ctx, w.client.contextPropagators)
	if err != nil {
		return nil, err
	}

	// run propagators to extract information about tracing and other stuff, store in headers field
	startRequest := &workflowservice.StartWorkflowExecutionRequest{
		Namespace:                w.client.namespace,
		RequestId:                uuid.New(),
		WorkflowId:               workflowID,
		WorkflowType:             &commonpb.WorkflowType{Name: in.WorkflowType},
		TaskQueue:                &taskqueuepb.TaskQueue{Name: in.Options.TaskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
		Input:                    input,
		WorkflowExecutionTimeout: &executionTimeout,
		WorkflowRunTimeout:       &runTimeout,
		WorkflowTaskTimeout:      &workflowTaskTimeout,
		Identity:                 w.client.identity,
		WorkflowIdReusePolicy:    in.Options.WorkflowIDReusePolicy,
		RetryPolicy:              convertToPBRetryPolicy(in.Options.RetryPolicy),
		CronSchedule:             in.Options.CronSchedule,
		Memo:                     memo,
		SearchAttributes:         searchAttr,
		Header:                   header,
	}

	var response *workflowservice.StartWorkflowExecutionResponse

	grpcCtx, cancel := newGRPCContext(ctx, grpcMetricsHandler(
		w.client.metricsHandler.WithTags(metrics.RPCTags(in.WorkflowType, metrics.NoneTagValue, in.Options.TaskQueue))),
		defaultGrpcRetryParameters(ctx))
	defer cancel()

	response, err = w.client.workflowService.StartWorkflowExecution(grpcCtx, startRequest)

	// Allow already-started error
	var runID string
	if e, ok := err.(*serviceerror.WorkflowExecutionAlreadyStarted); ok && !in.Options.WorkflowExecutionErrorWhenAlreadyStarted {
		runID = e.RunId
	} else if err != nil {
		return nil, err
	} else {
		runID = response.RunId
	}

	iterFn := func(fnCtx context.Context, fnRunID string) HistoryEventIterator {
		metricsHandler := w.client.metricsHandler.WithTags(metrics.RPCTags(in.WorkflowType,
			metrics.NoneTagValue, in.Options.TaskQueue))
		return w.client.getWorkflowHistory(fnCtx, workflowID, fnRunID, true,
			enumspb.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT, metricsHandler)
	}

	curRunIDCell := util.PopulatedOnceCell(runID)
	return &workflowRunImpl{
		workflowType:     in.WorkflowType,
		workflowID:       workflowID,
		firstRunID:       runID,
		currentRunID:     &curRunIDCell,
		iterFn:           iterFn,
		dataConverter:    w.client.dataConverter,
		failureConverter: w.client.failureConverter,
		registry:         w.client.registry,
	}, nil
}

func (w *workflowClientInterceptor) SignalWorkflow(ctx context.Context, in *ClientSignalWorkflowInput) error {
	dataConverter := WithContext(ctx, w.client.dataConverter)
	input, err := encodeArg(dataConverter, in.Arg)
	if err != nil {
		return err
	}

	// get workflow headers from the context
	header, err := headerPropagated(ctx, w.client.contextPropagators)
	if err != nil {
		return err
	}

	request := &workflowservice.SignalWorkflowExecutionRequest{
		Namespace: w.client.namespace,
		RequestId: uuid.New(),
		WorkflowExecution: &commonpb.WorkflowExecution{
			WorkflowId: in.WorkflowID,
			RunId:      in.RunID,
		},
		SignalName: in.SignalName,
		Input:      input,
		Identity:   w.client.identity,
		Header:     header,
	}

	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	_, err = w.client.workflowService.SignalWorkflowExecution(grpcCtx, request)
	return err
}

func (w *workflowClientInterceptor) SignalWithStartWorkflow(
	ctx context.Context,
	in *ClientSignalWithStartWorkflowInput,
) (WorkflowRun, error) {
	dataConverter := WithContext(ctx, w.client.dataConverter)
	signalInput, err := encodeArg(dataConverter, in.SignalArg)
	if err != nil {
		return nil, err
	}

	executionTimeout := in.Options.WorkflowExecutionTimeout
	runTimeout := in.Options.WorkflowRunTimeout
	taskTimeout := in.Options.WorkflowTaskTimeout

	// Encode input
	input, err := encodeArgs(dataConverter, in.Args)
	if err != nil {
		return nil, err
	}

	memo, err := getWorkflowMemo(in.Options.Memo, dataConverter)
	if err != nil {
		return nil, err
	}

	searchAttr, err := serializeSearchAttributes(in.Options.SearchAttributes)
	if err != nil {
		return nil, err
	}

	// get workflow headers from the context
	header, err := headerPropagated(ctx, w.client.contextPropagators)
	if err != nil {
		return nil, err
	}

	signalWithStartRequest := &workflowservice.SignalWithStartWorkflowExecutionRequest{
		Namespace:                w.client.namespace,
		RequestId:                uuid.New(),
		WorkflowId:               in.Options.ID,
		WorkflowType:             &commonpb.WorkflowType{Name: in.WorkflowType},
		TaskQueue:                &taskqueuepb.TaskQueue{Name: in.Options.TaskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
		Input:                    input,
		WorkflowExecutionTimeout: &executionTimeout,
		WorkflowRunTimeout:       &runTimeout,
		WorkflowTaskTimeout:      &taskTimeout,
		SignalName:               in.SignalName,
		SignalInput:              signalInput,
		Identity:                 w.client.identity,
		RetryPolicy:              convertToPBRetryPolicy(in.Options.RetryPolicy),
		CronSchedule:             in.Options.CronSchedule,
		Memo:                     memo,
		SearchAttributes:         searchAttr,
		WorkflowIdReusePolicy:    in.Options.WorkflowIDReusePolicy,
		Header:                   header,
	}

	var response *workflowservice.SignalWithStartWorkflowExecutionResponse

	// Start creating workflow request.
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	response, err = w.client.workflowService.SignalWithStartWorkflowExecution(grpcCtx, signalWithStartRequest)
	if err != nil {
		return nil, err
	}

	iterFn := func(fnCtx context.Context, fnRunID string) HistoryEventIterator {
		metricsHandler := w.client.metricsHandler.WithTags(metrics.RPCTags(in.WorkflowType,
			metrics.NoneTagValue, in.Options.TaskQueue))
		return w.client.getWorkflowHistory(fnCtx, in.Options.ID, fnRunID, true,
			enumspb.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT, metricsHandler)
	}

	curRunIDCell := util.PopulatedOnceCell(response.GetRunId())
	return &workflowRunImpl{
		workflowType:     in.WorkflowType,
		workflowID:       in.Options.ID,
		firstRunID:       response.GetRunId(),
		currentRunID:     &curRunIDCell,
		iterFn:           iterFn,
		dataConverter:    w.client.dataConverter,
		failureConverter: w.client.failureConverter,
		registry:         w.client.registry,
	}, nil
}

func (w *workflowClientInterceptor) CancelWorkflow(ctx context.Context, in *ClientCancelWorkflowInput) error {
	request := &workflowservice.RequestCancelWorkflowExecutionRequest{
		Namespace: w.client.namespace,
		RequestId: uuid.New(),
		WorkflowExecution: &commonpb.WorkflowExecution{
			WorkflowId: in.WorkflowID,
			RunId:      in.RunID,
		},
		Identity: w.client.identity,
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	_, err := w.client.workflowService.RequestCancelWorkflowExecution(grpcCtx, request)
	return err
}

func (w *workflowClientInterceptor) TerminateWorkflow(ctx context.Context, in *ClientTerminateWorkflowInput) error {
	datailsPayload, err := w.client.dataConverter.ToPayloads(in.Details...)
	if err != nil {
		return err
	}

	request := &workflowservice.TerminateWorkflowExecutionRequest{
		Namespace: w.client.namespace,
		WorkflowExecution: &commonpb.WorkflowExecution{
			WorkflowId: in.WorkflowID,
			RunId:      in.RunID,
		},
		Reason:   in.Reason,
		Identity: w.client.identity,
		Details:  datailsPayload,
	}

	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	_, err = w.client.workflowService.TerminateWorkflowExecution(grpcCtx, request)
	return err
}

func (w *workflowClientInterceptor) QueryWorkflow(
	ctx context.Context,
	in *ClientQueryWorkflowInput,
) (converter.EncodedValue, error) {
	// get workflow headers from the context
	header, err := headerPropagated(ctx, w.client.contextPropagators)
	if err != nil {
		return nil, err
	}

	result, err := w.client.QueryWorkflowWithOptions(ctx, &QueryWorkflowWithOptionsRequest{
		WorkflowID: in.WorkflowID,
		RunID:      in.RunID,
		QueryType:  in.QueryType,
		Args:       in.Args,
		Header:     header,
	})
	if err != nil {
		return nil, err
	}
	return result.QueryResult, nil
}

func (w *workflowClientInterceptor) UpdateWorkflow(
	ctx context.Context,
	in *ClientUpdateWorkflowInput,
) (WorkflowUpdateHandle, error) {
	argPayloads, err := w.client.dataConverter.ToPayloads(in.Args...)
	if err != nil {
		return nil, err
	}
	header, _ := headerPropagated(ctx, w.client.contextPropagators)

	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	wfexec := &commonpb.WorkflowExecution{
		WorkflowId: in.WorkflowID,
		RunId:      in.RunID,
	}
	resp, err := w.client.workflowService.UpdateWorkflowExecution(grpcCtx, &workflowservice.UpdateWorkflowExecutionRequest{
		WaitPolicy:          in.WaitPolicy,
		Namespace:           w.client.namespace,
		WorkflowExecution:   wfexec,
		FirstExecutionRunId: in.FirstExecutionRunID,
		Request: &updatepb.Request{
			Meta: &updatepb.Meta{
				UpdateId: in.UpdateID,
				Identity: w.client.identity,
			},
			Input: &updatepb.Input{
				Header: header,
				Name:   in.UpdateName,
				Args:   argPayloads,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	switch v := resp.GetOutcome().GetValue().(type) {
	case nil:
		return &lazyUpdateHandle{
			client:           w.client,
			baseUpdateHandle: baseUpdateHandle{ref: resp.GetUpdateRef()},
		}, nil
	case *updatepb.Outcome_Failure:
		return &completedUpdateHandle{
			err:              w.client.failureConverter.FailureToError(v.Failure),
			baseUpdateHandle: baseUpdateHandle{ref: resp.GetUpdateRef()},
		}, nil
	case *updatepb.Outcome_Success:
		return &completedUpdateHandle{
			value:            newEncodedValue(v.Success, w.client.dataConverter),
			baseUpdateHandle: baseUpdateHandle{ref: resp.GetUpdateRef()},
		}, nil
	}
	return nil, fmt.Errorf("unsupported outcome type %T", resp.GetOutcome().GetValue())
}

func (w *workflowClientInterceptor) PollWorkflowUpdate(
	parentCtx context.Context,
	in *ClientPollWorkflowUpdateInput,
) (converter.EncodedValue, error) {
	// header, _ = headerPropagated(ctx, w.client.contextPropagators)
	// todo header not in PollWorkflowUpdate

	pollReq := workflowservice.PollWorkflowExecutionUpdateRequest{
		Namespace: w.client.namespace,
		UpdateRef: in.UpdateRef,
		Identity:  w.client.identity,
		WaitPolicy: &updatepb.WaitPolicy{
			LifecycleStage: enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_COMPLETED,
		},
	}
	for parentCtx.Err() == nil {
		ctx, cancel := newGRPCContext(
			parentCtx,
			grpcLongPoll(true),
			grpcTimeout(pollUpdateTimeout),
		)
		ctx = context.WithValue(
			ctx,
			retry.ConfigKey,
			createDynamicServiceRetryPolicy(ctx).GrpcRetryConfig(),
		)
		resp, err := w.client.workflowService.PollWorkflowExecutionUpdate(ctx, &pollReq)
		cancel()
		if err == context.DeadlineExceeded ||
			status.Code(err) == codes.DeadlineExceeded ||
			(err == nil && resp.GetOutcome() == nil) {
			continue
		}
		if err != nil {
			return nil, err
		}
		switch v := resp.GetOutcome().GetValue().(type) {
		case *updatepb.Outcome_Failure:
			return nil, w.client.failureConverter.FailureToError(v.Failure)
		case *updatepb.Outcome_Success:
			return newEncodedValue(v.Success, w.client.dataConverter), nil
		default:
			return nil, fmt.Errorf("unsupported outcome type %T", v)
		}
	}
	return nil, parentCtx.Err()
}

// Required to implement ClientOutboundInterceptor
func (*workflowClientInterceptor) mustEmbedClientOutboundInterceptorBase() {}

func (uh *baseUpdateHandle) WorkflowID() string {
	return uh.ref.GetWorkflowExecution().GetWorkflowId()
}

func (uh *baseUpdateHandle) RunID() string {
	return uh.ref.GetWorkflowExecution().GetRunId()
}

func (uh *baseUpdateHandle) UpdateID() string {
	return uh.ref.GetUpdateId()
}

func (ch *completedUpdateHandle) Get(ctx context.Context, valuePtr interface{}) error {
	if ch.err != nil || valuePtr == nil {
		return ch.err
	}
	if err := ch.value.Get(valuePtr); err != nil {
		return err
	}
	return nil
}

func (luh *lazyUpdateHandle) Get(ctx context.Context, valuePtr interface{}) error {
	enc, err := luh.client.PollWorkflowUpdate(ctx, luh.ref)
	if err != nil {
		return err
	}
	return enc.Get(valuePtr)
}
