package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/nexus-rpc/sdk-go/nexus"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/encoding/protojson"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/log"
)

// NexusOperationInfo contains information about a currently executing Nexus operation.
//
// Exposed as: [go.temporal.io/sdk/temporalnexus.OperationInfo]
type NexusOperationInfo struct {
	// The namespace of the worker handling this Nexus operation.
	Namespace string
	// The task queue of the worker handling this Nexus operation.
	TaskQueue string
}

// NexusOperationContext is an internal only struct that holds fields used by the temporalnexus functions.
type NexusOperationContext struct {
	client         Client
	Namespace      string
	TaskQueue      string
	metricsHandler metrics.Handler
	log            log.Logger
	registry       *registry
}

func (nc *NexusOperationContext) ResolveWorkflowName(wf any) (string, error) {
	return getWorkflowFunctionName(nc.registry, wf)
}

type nexusOperationEnvironment struct {
	NexusOperationOutboundInterceptorBase
}

func (nc *nexusOperationEnvironment) GetOperationInfo(ctx context.Context) NexusOperationInfo {
	nctx, ok := NexusOperationContextFromGoContext(ctx)
	if !ok {
		panic("temporalnexus GetInfo: Not a valid Nexus context")
	}
	return NexusOperationInfo{
		Namespace: nctx.Namespace,
		TaskQueue: nctx.TaskQueue,
	}
}

func (nc *nexusOperationEnvironment) GetMetricsHandler(ctx context.Context) metrics.Handler {
	nctx, ok := NexusOperationContextFromGoContext(ctx)
	if !ok {
		panic("temporalnexus GetMetricsHandler: Not a valid Nexus context")
	}
	return nctx.metricsHandler
}

// GetLogger returns a logger to be used in a Nexus operation's context.
func (nc *nexusOperationEnvironment) GetLogger(ctx context.Context) log.Logger {
	nctx, ok := NexusOperationContextFromGoContext(ctx)
	if !ok {
		panic("temporalnexus GetMetricsHandler: Not a valid Nexus context")
	}
	return nctx.log
}

// GetClient returns a client to be used in a Nexus operation's context, this is the same client that the worker was
// created with. Client methods will panic when called from the test environment.
func (nc *nexusOperationEnvironment) GetClient(ctx context.Context) Client {
	nctx, ok := NexusOperationContextFromGoContext(ctx)
	if !ok {
		panic("temporalnexus GetMetricsHandler: Not a valid Nexus context")
	}
	return nctx.client
}

type nexusOperationOutboundInterceptorKeyType struct{}

// nexusOperationOutboundInterceptorKey is a key for associating a [NexusOperationOutboundInterceptor] with a [context.Context].
var nexusOperationOutboundInterceptorKey = nexusOperationOutboundInterceptorKeyType{}

// nexusOperationOutboundInterceptorFromGoContext gets the [NexusOperationOutboundInterceptor] associated with the given [context.Context].
func nexusOperationOutboundInterceptorFromGoContext(ctx context.Context) (nctx NexusOperationOutboundInterceptor, ok bool) {
	nctx, ok = ctx.Value(nexusOperationOutboundInterceptorKey).(NexusOperationOutboundInterceptor)
	return
}

// IsNexusOperation checks if the provided context is a Nexus operation context.
//
// Exposed as: [go.temporal.io/sdk/temporalnexus.IsNexusOperation]
func IsNexusOperation(ctx context.Context) bool {
	_, ok := NexusOperationContextFromGoContext(ctx)
	return ok
}

// GetNexusOperationInfo returns information about the currently executing Nexus operation.
//
// Exposed as: [go.temporal.io/sdk/temporalnexus.GetOperationInfo]
func GetNexusOperationInfo(ctx context.Context) NexusOperationInfo {
	interceptor, ok := nexusOperationOutboundInterceptorFromGoContext(ctx)
	if !ok {
		panic("temporalnexus GetOperationInfo: Not a valid Nexus context")
	}
	return interceptor.GetOperationInfo(ctx)
}

// GetNexusOperationMetricsHandler returns a metrics handler to be used in a Nexus operation's context.
//
// Exposed as: [go.temporal.io/sdk/temporalnexus.GetMetricsHandler]
func GetNexusOperationMetricsHandler(ctx context.Context) metrics.Handler {
	interceptor, ok := nexusOperationOutboundInterceptorFromGoContext(ctx)
	if !ok {
		panic("temporalnexus GetMetricsHandler: Not a valid Nexus context")
	}
	return interceptor.GetMetricsHandler(ctx)
}

// GetNexusOperationLogger returns a logger to be used in a Nexus operation's context.
//
// Exposed as: [go.temporal.io/sdk/temporalnexus.GetLogger]
func GetNexusOperationLogger(ctx context.Context) log.Logger {
	interceptor, ok := nexusOperationOutboundInterceptorFromGoContext(ctx)
	if !ok {
		panic("temporalnexus GetLogger: Not a valid Nexus context")
	}
	return interceptor.GetLogger(ctx)
}

// GetNexusOperationClient returns a client to be used in a Nexus operation's context, this is the same client that the
// worker was created with. Client methods will panic when called from the test environment.
//
// Exposed as: [go.temporal.io/sdk/temporalnexus.GetClient]
func GetNexusOperationClient(ctx context.Context) Client {
	interceptor, ok := nexusOperationOutboundInterceptorFromGoContext(ctx)
	if !ok {
		panic("temporalnexus GetClient: Not a valid Nexus context")
	}
	return interceptor.GetClient(ctx)
}

type nexusOperationContextKeyType struct{}

// nexusOperationContextKey is a key for associating a [NexusOperationContext] with a [context.Context].
var nexusOperationContextKey = nexusOperationContextKeyType{}

type isWorkflowRunOpContextKeyType struct{}

// IsWorkflowRunOpContextKey is a key to mark that the current context is used within a workflow run operation.
// The fake test env client verifies this key is set on the context to decide whether it should execute a method or
// panic as we don't want to expose a partial client to sync operations.
var IsWorkflowRunOpContextKey = isWorkflowRunOpContextKeyType{}

type nexusOperationRequestIDKeyType struct{}

var NexusOperationRequestIDKey = nexusOperationRequestIDKeyType{}

type nexusOperationLinksKeyType struct{}

var NexusOperationLinksKey = nexusOperationLinksKeyType{}

// NexusOperationContextFromGoContext gets the [NexusOperationContext] associated with the given [context.Context].
func NexusOperationContextFromGoContext(ctx context.Context) (nctx *NexusOperationContext, ok bool) {
	nctx, ok = ctx.Value(nexusOperationContextKey).(*NexusOperationContext)
	return
}

// nexusMiddleware constructs an adapter from Temporal WorkerInterceptors to a Nexus MiddlewareFunc.
func nexusMiddleware(interceptors []WorkerInterceptor) nexus.MiddlewareFunc {
	return func(ctx context.Context, next nexus.OperationHandler[any, any]) (nexus.OperationHandler[any, any], error) {
		root := &nexusInterceptorToMiddlewareAdapter{handler: next}
		var in NexusOperationInboundInterceptor = root
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			in = interceptor.InterceptNexusOperation(ctx, in)
		}
		if err := in.Init(ctx, &nexusOperationEnvironment{}); err != nil {
			return nil, err
		}
		return newNexusHandler(in, root.outboundInterceptor), nil
	}
}

// nexusMiddlewareToInterceptorAdapter is an adapter from the Nexus Handler interface to the Temporal interceptor interface.
type nexusMiddlewareToInterceptorAdapter struct {
	nexus.UnimplementedOperation[any, any]
	inboundInterceptor  NexusOperationInboundInterceptor
	outboundInterceptor NexusOperationOutboundInterceptor
}

func newNexusHandler(inbound NexusOperationInboundInterceptor, outbound NexusOperationOutboundInterceptor) nexus.OperationHandler[any, any] {
	return &nexusMiddlewareToInterceptorAdapter{inboundInterceptor: inbound, outboundInterceptor: outbound}
}

func (h *nexusMiddlewareToInterceptorAdapter) Start(ctx context.Context, input any, options nexus.StartOperationOptions) (nexus.HandlerStartOperationResult[any], error) {
	ctx = context.WithValue(ctx, nexusOperationOutboundInterceptorKey, h.outboundInterceptor)
	return h.inboundInterceptor.StartOperation(ctx, NexusStartOperationInput{
		Input:   input,
		Options: options,
	})
}

func (h *nexusMiddlewareToInterceptorAdapter) Cancel(ctx context.Context, token string, options nexus.CancelOperationOptions) error {
	ctx = context.WithValue(ctx, nexusOperationOutboundInterceptorKey, h.outboundInterceptor)
	return h.inboundInterceptor.CancelOperation(ctx, NexusCancelOperationInput{
		Token:   token,
		Options: options,
	})
}

// nexusInterceptorToMiddlewareAdapter is an adapter from the Temporal interceptor interface to the Nexus Handler interface.
type nexusInterceptorToMiddlewareAdapter struct {
	NexusOperationInboundInterceptorBase
	handler             nexus.OperationHandler[any, any]
	outboundInterceptor NexusOperationOutboundInterceptor
}

// CancelOperation implements NexusOperationInboundInterceptor.
func (n *nexusInterceptorToMiddlewareAdapter) CancelOperation(ctx context.Context, input NexusCancelOperationInput) error {
	return n.handler.Cancel(ctx, input.Token, input.Options)
}

// Init implements NexusOperationInboundInterceptor.
func (n *nexusInterceptorToMiddlewareAdapter) Init(ctx context.Context, outbound NexusOperationOutboundInterceptor) error {
	n.outboundInterceptor = outbound
	return nil
}

// StartOperation implements NexusOperationInboundInterceptor.
func (n *nexusInterceptorToMiddlewareAdapter) StartOperation(ctx context.Context, input NexusStartOperationInput) (nexus.HandlerStartOperationResult[any], error) {
	return n.handler.Start(ctx, input.Input, input.Options)
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////
// Most of the helpers in this section were duplicated from the server codebase at common/nexus/failure.go.
///////////////////////////////////////////////////////////////////////////////////////////////////////////

var failureTypeString = string((&failurepb.Failure{}).ProtoReflect().Descriptor().FullName())

// ProtoFailureToNexusFailure converts a proto Nexus Failure to a Nexus SDK Failure.
func protoFailureToNexusFailure(failure *nexuspb.Failure) nexus.Failure {
	return nexus.Failure{
		Message:  failure.GetMessage(),
		Metadata: failure.GetMetadata(),
		Details:  failure.GetDetails(),
	}
}

// nexusOperationFailure is a utility in use by the test environment.
func nexusOperationFailure(params executeNexusOperationParams, token string, cause *failurepb.Failure) *failurepb.Failure {
	return &failurepb.Failure{
		Message: "nexus operation completed unsuccessfully",
		FailureInfo: &failurepb.Failure_NexusOperationExecutionFailureInfo{
			NexusOperationExecutionFailureInfo: &failurepb.NexusOperationFailureInfo{
				Endpoint:       params.client.Endpoint(),
				Service:        params.client.Service(),
				Operation:      params.operation,
				OperationToken: token,
				OperationId:    token, // Also populate ID for backwards compatibility.
			},
		},
		Cause: cause,
	}
}

// nexusFailureToAPIFailure converts a Nexus Failure to an API proto Failure.
// If the failure metadata "type" field is set to the fullname of the temporal API Failure message, the failure is
// reconstructed using protojson.Unmarshal on the failure details field.
func nexusFailureToAPIFailure(failure nexus.Failure, retryable bool) (*failurepb.Failure, error) {
	apiFailure := &failurepb.Failure{}

	if failure.Metadata != nil && failure.Metadata["type"] == failureTypeString {
		if err := protojson.Unmarshal(failure.Details, apiFailure); err != nil {
			return nil, err
		}
	} else {
		payloads, err := nexusFailureMetadataToPayloads(failure)
		if err != nil {
			return nil, err
		}
		apiFailure.FailureInfo = &failurepb.Failure_ApplicationFailureInfo{
			ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{
				// Make up a type here, it's not part of the Nexus Failure spec.
				Type:         "NexusFailure",
				Details:      payloads,
				NonRetryable: !retryable,
			},
		}
	}
	// Ensure this always gets written.
	apiFailure.Message = failure.Message
	return apiFailure, nil
}

func nexusFailureMetadataToPayloads(failure nexus.Failure) (*commonpb.Payloads, error) {
	if len(failure.Metadata) == 0 && len(failure.Details) == 0 {
		return nil, nil
	}
	// Delete before serializing.
	failure.Message = ""
	data, err := json.Marshal(failure)
	if err != nil {
		return nil, err
	}
	return &commonpb.Payloads{
		Payloads: []*commonpb.Payload{
			{
				Metadata: map[string][]byte{
					"encoding": []byte("json/plain"),
				},
				Data: data,
			},
		},
	}, err
}

func apiOperationErrorToNexusOperationError(opErr *nexuspb.UnsuccessfulOperationError) *nexus.OperationError {
	return &nexus.OperationError{
		State: nexus.OperationState(opErr.GetOperationState()),
		Cause: &nexus.FailureError{
			Failure: protoFailureToNexusFailure(opErr.GetFailure()),
		},
	}
}

func apiHandlerErrorToNexusHandlerError(apiErr *nexuspb.HandlerError, failureConverter converter.FailureConverter) (*nexus.HandlerError, error) {
	var retryBehavior nexus.HandlerErrorRetryBehavior
	// nolint:exhaustive // unspecified is the default
	switch apiErr.GetRetryBehavior() {
	case enums.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_RETRYABLE:
		retryBehavior = nexus.HandlerErrorRetryBehaviorRetryable
	case enums.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_NON_RETRYABLE:
		retryBehavior = nexus.HandlerErrorRetryBehaviorNonRetryable
	}

	nexusErr := &nexus.HandlerError{
		Type:          nexus.HandlerErrorType(apiErr.GetErrorType()),
		RetryBehavior: retryBehavior,
	}

	failure, err := nexusFailureToAPIFailure(protoFailureToNexusFailure(apiErr.GetFailure()), nexusErr.Retryable())
	if err != nil {
		return nil, err
	}
	nexusErr.Cause = failureConverter.FailureToError(failure)
	return nexusErr, nil
}

func operationErrorToTemporalFailure(opErr *nexus.OperationError) (*failurepb.Failure, error) {
	var nexusFailure nexus.Failure
	failureErr, ok := opErr.Cause.(*nexus.FailureError)
	if ok {
		nexusFailure = failureErr.Failure
	} else if opErr.Cause != nil {
		nexusFailure = nexus.Failure{Message: opErr.Cause.Error()}
	}

	// Canceled must be translated into a CanceledFailure to match the SDK expectation.
	if opErr.State == nexus.OperationStateCanceled {
		if nexusFailure.Metadata != nil && nexusFailure.Metadata["type"] == failureTypeString {
			temporalFailure, err := nexusFailureToAPIFailure(nexusFailure, false)
			if err != nil {
				return nil, err
			}
			if temporalFailure.GetCanceledFailureInfo() != nil {
				// We already have a CanceledFailure, use it.
				return temporalFailure, nil
			}
			// Fallback to encoding the Nexus failure into a Temporal canceled failure, we expect operations that end up
			// as canceled to have a CanceledFailureInfo object.
		}
		payloads, err := nexusFailureMetadataToPayloads(nexusFailure)
		if err != nil {
			return nil, err
		}
		return &failurepb.Failure{
			Message: nexusFailure.Message,
			FailureInfo: &failurepb.Failure_CanceledFailureInfo{
				CanceledFailureInfo: &failurepb.CanceledFailureInfo{
					Details: payloads,
				},
			},
		}, nil
	}

	return nexusFailureToAPIFailure(nexusFailure, false)
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////
// END Nexus failure section.
///////////////////////////////////////////////////////////////////////////////////////////////////////////

// testSuiteClientForNexusOperations is a partial [Client] implementation for the test workflow environment used to
// support running the workflow run operation - and only this operation, all methods will panic when this client is
// passed to sync operations.
type testSuiteClientForNexusOperations struct {
	env *testWorkflowEnvironmentImpl
}

// DescribeWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) DescribeWorkflow(ctx context.Context, workflowID string, runID string) (*WorkflowExecutionDescription, error) {
	panic("not implemented in the test environment")
}

// CancelWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) CancelWorkflow(ctx context.Context, workflowID string, runID string) error {
	if set, ok := ctx.Value(IsWorkflowRunOpContextKey).(bool); !ok || !set {
		panic("not implemented in the test environment")
	}
	doneCh := make(chan error)
	t.env.cancelWorkflowByID(workflowID, runID, func(result *commonpb.Payloads, err error) {
		doneCh <- err
	})
	return <-doneCh
}

// CheckHealth implements Client.
func (t *testSuiteClientForNexusOperations) CheckHealth(ctx context.Context, request *CheckHealthRequest) (*CheckHealthResponse, error) {
	return &CheckHealthResponse{}, nil
}

// Close implements Client.
func (t *testSuiteClientForNexusOperations) Close() {
	// No op.
}

// CompleteActivity implements Client.
func (t *testSuiteClientForNexusOperations) CompleteActivity(ctx context.Context, taskToken []byte, result interface{}, err error) error {
	panic("not implemented in the test environment")
}

// CompleteActivityByID implements Client.
func (t *testSuiteClientForNexusOperations) CompleteActivityByID(ctx context.Context, namespace string, workflowID string, runID string, activityID string, result interface{}, err error) error {
	panic("not implemented in the test environment")
}

// CountWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) CountWorkflow(ctx context.Context, request *workflowservice.CountWorkflowExecutionsRequest) (*workflowservice.CountWorkflowExecutionsResponse, error) {
	panic("not implemented in the test environment")
}

// DescribeTaskQueue implements Client.
func (t *testSuiteClientForNexusOperations) DescribeTaskQueue(ctx context.Context, taskqueue string, taskqueueType enums.TaskQueueType) (*workflowservice.DescribeTaskQueueResponse, error) {
	panic("not implemented in the test environment")
}

// DescribeTaskQueueEnhanced implements Client.
func (t *testSuiteClientForNexusOperations) DescribeTaskQueueEnhanced(ctx context.Context, options DescribeTaskQueueEnhancedOptions) (TaskQueueDescription, error) {
	panic("unimplemented in the test environment")
}

// DescribeWorkflowExecution implements Client.
func (t *testSuiteClientForNexusOperations) DescribeWorkflowExecution(ctx context.Context, workflowID string, runID string) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	panic("not implemented in the test environment")
}

// ExecuteWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) ExecuteWorkflow(ctx context.Context, options StartWorkflowOptions, workflow interface{}, args ...interface{}) (WorkflowRun, error) {
	if set, ok := ctx.Value(IsWorkflowRunOpContextKey).(bool); !ok || !set {
		panic("not implemented in the test environment")
	}
	wfType, input, err := getValidatedWorkflowFunction(workflow, args, t.env.dataConverter, t.env.GetRegistry())
	if err != nil {
		return nil, fmt.Errorf("cannot validate workflow function: %w", err)
	}

	run := &testEnvWorkflowRunForNexusOperations{}
	startedErrCh := make(chan error, 1)
	doneCh := make(chan error)

	var callback *commonpb.Callback

	if len(options.callbacks) > 0 {
		callback = options.callbacks[0]
	}

	t.env.postCallback(func() {
		t.env.executeChildWorkflowWithDelay(options.StartDelay, ExecuteWorkflowParams{
			// Not propagating Header as this client does not support context propagation.
			WorkflowType: wfType,
			Input:        input,
			WorkflowOptions: WorkflowOptions{
				WaitForCancellation:      true,
				Namespace:                t.env.workflowInfo.Namespace,
				TaskQueueName:            t.env.workflowInfo.TaskQueueName,
				WorkflowID:               options.ID,
				WorkflowExecutionTimeout: options.WorkflowExecutionTimeout,
				WorkflowRunTimeout:       options.WorkflowRunTimeout,
				WorkflowTaskTimeout:      options.WorkflowTaskTimeout,
				DataConverter:            t.env.dataConverter,
				WorkflowIDReusePolicy:    options.WorkflowIDReusePolicy,
				WorkflowIDConflictPolicy: options.WorkflowIDConflictPolicy,
				OnConflictOptions:        options.onConflictOptions,
				ContextPropagators:       t.env.contextPropagators,
				SearchAttributes:         options.SearchAttributes,
				TypedSearchAttributes:    options.TypedSearchAttributes,
				ParentClosePolicy:        enums.PARENT_CLOSE_POLICY_ABANDON,
				Memo:                     options.Memo,
				CronSchedule:             options.CronSchedule,
				RetryPolicy:              convertToPBRetryPolicy(options.RetryPolicy),
				Priority:                 convertToPBPriority(options.Priority),
			},
		}, func(result *commonpb.Payloads, wfErr error) {
			// This callback handles async completion of Nexus operations. If there was an error when
			// starting the workflow, then the operation failed synchronously and this callback doesn't
			// need to be executed.
			startedErr := <-startedErrCh
			if startedErr != nil {
				return
			}

			ncb := callback.GetNexus()
			if ncb == nil {
				return
			}
			seqStr := ncb.GetHeader()["operation-sequence"]
			if seqStr == "" {
				return
			}
			seq, err := strconv.ParseInt(seqStr, 10, 64)
			if err != nil {
				panic(fmt.Errorf("unexpected operation sequence in callback header: %s: %w", seqStr, err))
			}

			// Send the operation token to account for a race when the completion comes in before the response to the
			// StartOperation call is recorded.
			// The token is extracted from the callback header which is attached in ExecuteUntypedWorkflow.
			var operationToken string
			if len(options.callbacks) == 1 {
				if cbHeader := options.callbacks[0].GetNexus().GetHeader(); cbHeader != nil {
					operationToken = cbHeader[nexus.HeaderOperationToken]
				}
			}

			if wfErr != nil {
				t.env.resolveNexusOperation(seq, operationToken, nil, wfErr)
			} else {
				var payload *commonpb.Payload
				if len(result.GetPayloads()) > 0 {
					payload = result.Payloads[0]
				}
				t.env.resolveNexusOperation(seq, operationToken, payload, nil)
			}
		}, func(r WorkflowExecution, err error) {
			run.WorkflowExecution = r
			startedErrCh <- err
			close(startedErrCh)
			doneCh <- err
		})
	}, false)
	err = <-doneCh
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (t *testSuiteClientForNexusOperations) NewWithStartWorkflowOperation(options StartWorkflowOptions, workflow interface{}, args ...interface{}) WithStartWorkflowOperation {
	panic("not implemented in the test environment")
}

// GetSearchAttributes implements Client.
func (t *testSuiteClientForNexusOperations) GetSearchAttributes(ctx context.Context) (*workflowservice.GetSearchAttributesResponse, error) {
	panic("not implemented in the test environment")
}

// GetWorkerBuildIdCompatibility implements Client.
func (t *testSuiteClientForNexusOperations) GetWorkerBuildIdCompatibility(ctx context.Context, options *GetWorkerBuildIdCompatibilityOptions) (*WorkerBuildIDVersionSets, error) {
	panic("not implemented in the test environment")
}

// GetWorkerTaskReachability implements Client.
func (t *testSuiteClientForNexusOperations) GetWorkerTaskReachability(ctx context.Context, options *GetWorkerTaskReachabilityOptions) (*WorkerTaskReachability, error) {
	panic("not implemented in the test environment")
}

// GetWorkerVersioningRules implements Client.
func (t *testSuiteClientForNexusOperations) GetWorkerVersioningRules(ctx context.Context, options GetWorkerVersioningOptions) (*WorkerVersioningRules, error) {
	panic("unimplemented in the test environment")
}

// GetWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) GetWorkflow(ctx context.Context, workflowID string, runID string) WorkflowRun {
	panic("not implemented in the test environment")
}

// GetWorkflowHistory implements Client.
func (t *testSuiteClientForNexusOperations) GetWorkflowHistory(ctx context.Context, workflowID string, runID string, isLongPoll bool, filterType enums.HistoryEventFilterType) HistoryEventIterator {
	panic("not implemented in the test environment")
}

// GetWorkflowUpdateHandle implements Client.
func (t *testSuiteClientForNexusOperations) GetWorkflowUpdateHandle(GetWorkflowUpdateHandleOptions) WorkflowUpdateHandle {
	panic("not implemented in the test environment")
}

// ListArchivedWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) ListArchivedWorkflow(ctx context.Context, request *workflowservice.ListArchivedWorkflowExecutionsRequest) (*workflowservice.ListArchivedWorkflowExecutionsResponse, error) {
	panic("not implemented in the test environment")
}

// ListClosedWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) ListClosedWorkflow(ctx context.Context, request *workflowservice.ListClosedWorkflowExecutionsRequest) (*workflowservice.ListClosedWorkflowExecutionsResponse, error) {
	panic("not implemented in the test environment")
}

// ListOpenWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) ListOpenWorkflow(ctx context.Context, request *workflowservice.ListOpenWorkflowExecutionsRequest) (*workflowservice.ListOpenWorkflowExecutionsResponse, error) {
	panic("not implemented in the test environment")
}

// ListWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) ListWorkflow(ctx context.Context, request *workflowservice.ListWorkflowExecutionsRequest) (*workflowservice.ListWorkflowExecutionsResponse, error) {
	panic("not implemented in the test environment")
}

// OperatorService implements Client.
func (t *testSuiteClientForNexusOperations) OperatorService() operatorservice.OperatorServiceClient {
	panic("not implemented in the test environment")
}

// QueryWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) QueryWorkflow(ctx context.Context, workflowID string, runID string, queryType string, args ...interface{}) (converter.EncodedValue, error) {
	panic("not implemented in the test environment")
}

// QueryWorkflowWithOptions implements Client.
func (t *testSuiteClientForNexusOperations) QueryWorkflowWithOptions(ctx context.Context, request *QueryWorkflowWithOptionsRequest) (*QueryWorkflowWithOptionsResponse, error) {
	panic("not implemented in the test environment")
}

// RecordActivityHeartbeat implements Client.
func (t *testSuiteClientForNexusOperations) RecordActivityHeartbeat(ctx context.Context, taskToken []byte, details ...interface{}) error {
	panic("not implemented in the test environment")
}

// RecordActivityHeartbeatByID implements Client.
func (t *testSuiteClientForNexusOperations) RecordActivityHeartbeatByID(ctx context.Context, namespace string, workflowID string, runID string, activityID string, details ...interface{}) error {
	panic("not implemented in the test environment")
}

// ResetWorkflowExecution implements Client.
func (t *testSuiteClientForNexusOperations) ResetWorkflowExecution(ctx context.Context, request *workflowservice.ResetWorkflowExecutionRequest) (*workflowservice.ResetWorkflowExecutionResponse, error) {
	panic("not implemented in the test environment")
}

// ScanWorkflow implements Client.
//
//lint:ignore SA1019 the server API was deprecated.
func (t *testSuiteClientForNexusOperations) ScanWorkflow(ctx context.Context, request *workflowservice.ScanWorkflowExecutionsRequest) (*workflowservice.ScanWorkflowExecutionsResponse, error) {
	panic("not implemented in the test environment")
}

// ScheduleClient implements Client.
func (t *testSuiteClientForNexusOperations) ScheduleClient() ScheduleClient {
	panic("not implemented in the test environment")
}

// SignalWithStartWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) SignalWithStartWorkflow(ctx context.Context, workflowID string, signalName string, signalArg interface{}, options StartWorkflowOptions, workflow interface{}, workflowArgs ...interface{}) (WorkflowRun, error) {
	panic("not implemented in the test environment")
}

// SignalWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) SignalWorkflow(ctx context.Context, workflowID string, runID string, signalName string, arg interface{}) error {
	panic("not implemented in the test environment")
}

// TerminateWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) TerminateWorkflow(ctx context.Context, workflowID string, runID string, reason string, details ...interface{}) error {
	panic("not implemented in the test environment")
}

// UpdateWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) UpdateWorkflow(ctx context.Context, options UpdateWorkflowOptions) (WorkflowUpdateHandle, error) {
	panic("unimplemented in the test environment")
}

// UpdateWithStartWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) UpdateWithStartWorkflow(ctx context.Context, options UpdateWithStartWorkflowOptions) (WorkflowUpdateHandle, error) {
	panic("unimplemented in the test environment")
}

// UpdateWorkerBuildIdCompatibility implements Client.
func (t *testSuiteClientForNexusOperations) UpdateWorkerBuildIdCompatibility(ctx context.Context, options *UpdateWorkerBuildIdCompatibilityOptions) error {
	panic("not implemented in the test environment")
}

// UpdateWorkerVersioningRules implements Client.
func (t *testSuiteClientForNexusOperations) UpdateWorkerVersioningRules(ctx context.Context, options UpdateWorkerVersioningRulesOptions) (*WorkerVersioningRules, error) {
	panic("unimplemented in the test environment")
}

// WorkflowService implements Client.
func (t *testSuiteClientForNexusOperations) WorkflowService() workflowservice.WorkflowServiceClient {
	panic("not implemented in the test environment")
}

// DeploymentClient implements Client.
func (t *testSuiteClientForNexusOperations) DeploymentClient() DeploymentClient {
	panic("not implemented in the test environment")
}

// WorkerDeploymentClient implements Client.
func (t *testSuiteClientForNexusOperations) WorkerDeploymentClient() WorkerDeploymentClient {
	panic("not implemented in the test environment")
}

// UpdateWorkflowExecutionOptions implements Client.
func (t *testSuiteClientForNexusOperations) UpdateWorkflowExecutionOptions(ctx context.Context, options UpdateWorkflowExecutionOptionsRequest) (WorkflowExecutionOptions, error) {
	panic("not implemented in the test environment")
}

var _ Client = &testSuiteClientForNexusOperations{}

// testEnvWorkflowRunForNexusOperations is a partial [WorkflowRun] implementation for the test workflow environment used
// to support basic Nexus functionality.
type testEnvWorkflowRunForNexusOperations struct {
	WorkflowExecution
}

// Get implements WorkflowRun.
func (t *testEnvWorkflowRunForNexusOperations) Get(ctx context.Context, valuePtr interface{}) error {
	panic("not implemented in the test environment")
}

// GetID implements WorkflowRun.
func (t *testEnvWorkflowRunForNexusOperations) GetID() string {
	return t.ID
}

// GetRunID implements WorkflowRun.
func (t *testEnvWorkflowRunForNexusOperations) GetRunID() string {
	return t.RunID
}

// GetWithOptions implements WorkflowRun.
func (t *testEnvWorkflowRunForNexusOperations) GetWithOptions(ctx context.Context, valuePtr interface{}, options WorkflowRunGetOptions) error {
	panic("not implemented in the test environment")
}

// Exposed as: [go.temporal.io/sdk/client.WorkflowRun]
var _ WorkflowRun = &testEnvWorkflowRunForNexusOperations{}
