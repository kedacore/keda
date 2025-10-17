package internal

import (
	"context"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	updatepb "go.temporal.io/api/update/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/log"
)

// Interceptor is a common interface for all interceptors. See documentation in
// the interceptor package for more details.
//
// Exposed as: [go.temporal.io/sdk/interceptor.Interceptor]
type Interceptor interface {
	ClientInterceptor
	WorkerInterceptor
}

// WorkerInterceptor is a common interface for all interceptors. See
// documentation in the interceptor package for more details.
//
// Exposed as: [go.temporal.io/sdk/interceptor.WorkerInterceptor]
type WorkerInterceptor interface {
	// InterceptActivity is called before each activity interception needed with
	// the next interceptor in the chain.
	InterceptActivity(ctx context.Context, next ActivityInboundInterceptor) ActivityInboundInterceptor

	// InterceptWorkflow is called before each workflow interception needed with
	// the next interceptor in the chain.
	InterceptWorkflow(ctx Context, next WorkflowInboundInterceptor) WorkflowInboundInterceptor

	InterceptNexusOperation(ctx context.Context, next NexusOperationInboundInterceptor) NexusOperationInboundInterceptor

	mustEmbedWorkerInterceptorBase()
}

// ActivityInboundInterceptor is an interface for all activity calls originating
// from the server. See documentation in the interceptor package for more
// details.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ActivityInboundInterceptor]
type ActivityInboundInterceptor interface {
	// Init is the first call of this interceptor. Implementations can change/wrap
	// the outbound interceptor before calling Init on the next interceptor.
	Init(outbound ActivityOutboundInterceptor) error

	// ExecuteActivity is called when an activity is to be run on this worker.
	// interceptor.Header will return a non-nil map for this context.
	ExecuteActivity(ctx context.Context, in *ExecuteActivityInput) (interface{}, error)

	mustEmbedActivityInboundInterceptorBase()
}

// ExecuteActivityInput is the input to ActivityInboundInterceptor.ExecuteActivity.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ExecuteActivityInput]
type ExecuteActivityInput struct {
	Args []interface{}
}

// ActivityOutboundInterceptor is an interface for all activity calls
// originating from the SDK. See documentation in the interceptor package for
// more details.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ActivityOutboundInterceptor]
type ActivityOutboundInterceptor interface {
	// GetInfo intercepts activity.GetInfo.
	GetInfo(ctx context.Context) ActivityInfo

	// GetLogger intercepts activity.GetLogger.
	GetLogger(ctx context.Context) log.Logger

	// GetMetricsHandler intercepts activity.GetMetricsHandler.
	GetMetricsHandler(ctx context.Context) metrics.Handler

	// RecordHeartbeat intercepts activity.RecordHeartbeat.
	RecordHeartbeat(ctx context.Context, details ...interface{})

	// HasHeartbeatDetails intercepts activity.HasHeartbeatDetails.
	HasHeartbeatDetails(ctx context.Context) bool

	// GetHeartbeatDetails intercepts activity.GetHeartbeatDetails.
	GetHeartbeatDetails(ctx context.Context, d ...interface{}) error

	// GetWorkerStopChannel intercepts activity.GetWorkerStopChannel.
	GetWorkerStopChannel(ctx context.Context) <-chan struct{}

	// GetClient intercepts activity.GetClient.
	GetClient(ctx context.Context) Client

	mustEmbedActivityOutboundInterceptorBase()
}

// WorkflowInboundInterceptor is an interface for all workflow calls originating
// from the server. See documentation in the interceptor package for more
// details.
//
// Exposed as: [go.temporal.io/sdk/interceptor.WorkflowInboundInterceptor]
type WorkflowInboundInterceptor interface {
	// Init is the first call of this interceptor. Implementations can change/wrap
	// the outbound interceptor before calling Init on the next interceptor.
	Init(outbound WorkflowOutboundInterceptor) error

	// ExecuteWorkflow is called when a workflow is to be run on this worker.
	// interceptor.WorkflowHeader will return a non-nil map for this context.
	ExecuteWorkflow(ctx Context, in *ExecuteWorkflowInput) (interface{}, error)

	// HandleSignal is called when a signal is sent to a workflow on this worker.
	// interceptor.WorkflowHeader will return a non-nil map for this context.
	HandleSignal(ctx Context, in *HandleSignalInput) error

	// HandleQuery is called when a query is sent to a workflow on this worker.
	// interceptor.WorkflowHeader will return a non-nil map for this context.
	HandleQuery(ctx Context, in *HandleQueryInput) (interface{}, error)

	// ValidateUpdate is always called prior to executing an update, even if the
	// update handler for in.Name was not registered with a validation function
	// as part of its optional configuration. The same prohibition against
	// mutating workflow state that is demanded of UpdateOptions.Validator
	// functions also applies to this function.
	ValidateUpdate(ctx Context, in *UpdateInput) error

	// ExecuteUpdate is called after ValidateUpdate if and only if the latter
	// returns nil. interceptor.WorkflowHeader will return a non-nil map for
	// this context. ExecuteUpdate is allowed to mutate workflow state and
	// perform workflow actions such as scheduling activities, timers, etc.
	ExecuteUpdate(ctx Context, in *UpdateInput) (interface{}, error)

	mustEmbedWorkflowInboundInterceptorBase()
}

// ExecuteWorkflowInput is the input to
// WorkflowInboundInterceptor.ExecuteWorkflow.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ExecuteWorkflowInput]
type ExecuteWorkflowInput struct {
	Args []interface{}
}

// HandleSignalInput is the input to WorkflowInboundInterceptor.HandleSignal.
//
// Exposed as: [go.temporal.io/sdk/interceptor.HandleSignalInput]
type HandleSignalInput struct {
	SignalName string
	// Arg is the signal argument. It is presented as a primitive payload since
	// the type needed for decode is not available at the time of interception.
	Arg *commonpb.Payloads
}

// UpdateInput carries the name and arguments of a workflow update invocation.
//
// Exposed as: [go.temporal.io/sdk/interceptor.UpdateInput]
type UpdateInput struct {
	Name string
	Args []interface{}
}

// HandleQueryInput is the input to WorkflowInboundInterceptor.HandleQuery.
//
// Exposed as: [go.temporal.io/sdk/interceptor.HandleQueryInput]
type HandleQueryInput struct {
	QueryType string
	Args      []interface{}
}

// ExecuteNexusOperationInput is the input to WorkflowOutboundInterceptor.ExecuteNexusOperation.
//
// NOTE: Experimental
//
// Exposed as: [go.temporal.io/sdk/interceptor.ExecuteNexusOperationInput]
type ExecuteNexusOperationInput struct {
	// Client to start the operation with.
	Client NexusClient
	// Operation name or OperationReference from the Nexus SDK.
	Operation any
	// Operation input.
	Input any
	// Options for starting the operation.
	Options NexusOperationOptions
	// Header to attach to the request.
	NexusHeader nexus.Header
}

// RequestCancelNexusOperationInput is the input to WorkflowOutboundInterceptor.RequestCancelNexusOperation.
//
// NOTE: Experimental
//
// Exposed as: [go.temporal.io/sdk/interceptor.RequestCancelNexusOperationInput]
type RequestCancelNexusOperationInput struct {
	// Client that was used to start the operation.
	Client NexusClient
	// Operation name or OperationReference from the Nexus SDK.
	Operation any
	// Operation Token. May be empty if the operation is synchronous or has not started yet.
	Token string
	// seq number. For internal use only.
	seq int64
}

// WorkflowOutboundInterceptor is an interface for all workflow calls
// originating from the SDK. See documentation in the interceptor package for
// more details.
//
// Exposed as: [go.temporal.io/sdk/interceptor.WorkflowOutboundInterceptor]
type WorkflowOutboundInterceptor interface {
	// Go intercepts workflow.Go.
	Go(ctx Context, name string, f func(ctx Context)) Context

	// Await intercepts workflow.Await.
	Await(ctx Context, condition func() bool) error

	// AwaitWithTimeout intercepts workflow.AwaitWithTimeout.
	AwaitWithTimeout(ctx Context, timeout time.Duration, condition func() bool) (bool, error)

	// AwaitWithOptions intercepts workflow.AwaitWithOptions.
	//
	// NOTE: Experimental
	AwaitWithOptions(ctx Context, options AwaitOptions, condition func() bool) (bool, error)

	// ExecuteActivity intercepts workflow.ExecuteActivity.
	// interceptor.WorkflowHeader will return a non-nil map for this context.
	ExecuteActivity(ctx Context, activityType string, args ...interface{}) Future

	// ExecuteLocalActivity intercepts workflow.ExecuteLocalActivity.
	// interceptor.WorkflowHeader will return a non-nil map for this context.
	ExecuteLocalActivity(ctx Context, activityType string, args ...interface{}) Future

	// ExecuteChildWorkflow intercepts workflow.ExecuteChildWorkflow.
	// interceptor.WorkflowHeader will return a non-nil map for this context.
	ExecuteChildWorkflow(ctx Context, childWorkflowType string, args ...interface{}) ChildWorkflowFuture

	// GetInfo intercepts workflow.GetInfo.
	GetInfo(ctx Context) *WorkflowInfo

	// GetTypedSearchAttributes intercepts workflow.GetTypedSearchAttributes.
	GetTypedSearchAttributes(ctx Context) SearchAttributes

	// GetCurrentUpdateInfo intercepts workflow.GetCurrentUpdateInfo.
	GetCurrentUpdateInfo(ctx Context) *UpdateInfo

	// GetLogger intercepts workflow.GetLogger.
	GetLogger(ctx Context) log.Logger

	// GetMetricsHandler intercepts workflow.GetMetricsHandler.
	GetMetricsHandler(ctx Context) metrics.Handler

	// Now intercepts workflow.Now.
	Now(ctx Context) time.Time

	// NewTimer intercepts workflow.NewTimer.
	NewTimer(ctx Context, d time.Duration) Future

	// NewTimer intercepts workflow.NewTimerWithOptions.
	//
	// NOTE: Experimental
	NewTimerWithOptions(ctx Context, d time.Duration, options TimerOptions) Future

	// Sleep intercepts workflow.Sleep.
	Sleep(ctx Context, d time.Duration) (err error)

	// RequestCancelExternalWorkflow intercepts
	// workflow.RequestCancelExternalWorkflow.
	RequestCancelExternalWorkflow(ctx Context, workflowID, runID string) Future

	// SignalExternalWorkflow intercepts workflow.SignalExternalWorkflow.
	// interceptor.WorkflowHeader will return a non-nil map for this context.
	SignalExternalWorkflow(ctx Context, workflowID, runID, signalName string, arg interface{}) Future

	// SignalChildWorkflow intercepts
	// workflow.ChildWorkflowFuture.SignalChildWorkflow.
	// interceptor.WorkflowHeader will return a non-nil map for this context.
	SignalChildWorkflow(ctx Context, workflowID, signalName string, arg interface{}) Future

	// UpsertSearchAttributes intercepts workflow.UpsertSearchAttributes.
	UpsertSearchAttributes(ctx Context, attributes map[string]interface{}) error

	// UpsertTypedSearchAttributes intercepts workflow.UpsertTypedSearchAttributes.
	UpsertTypedSearchAttributes(ctx Context, attributes ...SearchAttributeUpdate) error

	// UpsertMemo intercepts workflow.UpsertMemo.
	UpsertMemo(ctx Context, memo map[string]interface{}) error

	// GetSignalChannel intercepts workflow.GetSignalChannel.
	GetSignalChannel(ctx Context, signalName string) ReceiveChannel

	// GetSignalChannelWithOptions intercepts workflow.GetSignalChannelWithOptions.
	//
	// NOTE: Experimental
	GetSignalChannelWithOptions(ctx Context, signalName string, options SignalChannelOptions) ReceiveChannel

	// SideEffect intercepts workflow.SideEffect.
	SideEffect(ctx Context, f func(ctx Context) interface{}) converter.EncodedValue

	// MutableSideEffect intercepts workflow.MutableSideEffect.
	MutableSideEffect(
		ctx Context,
		id string,
		f func(ctx Context) interface{},
		equals func(a, b interface{}) bool,
	) converter.EncodedValue

	// GetVersion intercepts workflow.GetVersion.
	GetVersion(ctx Context, changeID string, minSupported, maxSupported Version) Version

	// SetQueryHandler intercepts workflow.SetQueryHandler.
	SetQueryHandler(ctx Context, queryType string, handler interface{}) error

	// SetQueryHandlerWithOptions intercepts workflow.SetQueryHandlerWithOptions.
	//
	// NOTE: Experimental
	SetQueryHandlerWithOptions(ctx Context, queryType string, handler interface{}, options QueryHandlerOptions) error

	// SetUpdateHandler intercepts workflow.SetUpdateHandler.
	SetUpdateHandler(ctx Context, updateName string, handler interface{}, opts UpdateHandlerOptions) error

	// IsReplaying intercepts workflow.IsReplaying.
	IsReplaying(ctx Context) bool

	// HasLastCompletionResult intercepts workflow.HasLastCompletionResult.
	HasLastCompletionResult(ctx Context) bool

	// GetLastCompletionResult intercepts workflow.GetLastCompletionResult.
	GetLastCompletionResult(ctx Context, d ...interface{}) error

	// GetLastError intercepts workflow.GetLastError.
	GetLastError(ctx Context) error

	// NewContinueAsNewError intercepts workflow.NewContinueAsNewError.
	// interceptor.WorkflowHeader will return a non-nil map for this context.
	NewContinueAsNewError(ctx Context, wfn interface{}, args ...interface{}) error

	// ExecuteNexusOperation intercepts NexusClient.ExecuteOperation.
	//
	// NOTE: Experimental
	ExecuteNexusOperation(ctx Context, input ExecuteNexusOperationInput) NexusOperationFuture

	// RequestCancelNexusOperation intercepts Nexus Operation cancellation via context.
	//
	// NOTE: Experimental
	RequestCancelNexusOperation(ctx Context, input RequestCancelNexusOperationInput)

	mustEmbedWorkflowOutboundInterceptorBase()
}

// ClientInterceptor for providing a ClientOutboundInterceptor to intercept
// certain workflow-specific client calls from the SDK. See documentation in the
// interceptor package for more details.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ClientInterceptor]
type ClientInterceptor interface {
	// This is called on client creation if set via client options
	InterceptClient(next ClientOutboundInterceptor) ClientOutboundInterceptor

	mustEmbedClientInterceptorBase()
}

// ClientOutboundInterceptor is an interface for certain workflow-specific calls
// originating from the SDK. See documentation in the interceptor package for
// more details.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ClientOutboundInterceptor]
type ClientOutboundInterceptor interface {
	// ExecuteWorkflow intercepts client.Client.ExecuteWorkflow.
	// interceptor.Header will return a non-nil map for this context.
	ExecuteWorkflow(context.Context, *ClientExecuteWorkflowInput) (WorkflowRun, error)

	// CreateSchedule - Intercept a service call to CreateSchedule
	CreateSchedule(ctx context.Context, options *ScheduleClientCreateInput) (ScheduleHandle, error)

	// SignalWorkflow intercepts client.Client.SignalWorkflow.
	// interceptor.Header will return a non-nil map for this context.
	SignalWorkflow(context.Context, *ClientSignalWorkflowInput) error

	// SignalWithStartWorkflow intercepts client.Client.SignalWithStartWorkflow.
	// interceptor.Header will return a non-nil map for this context.
	SignalWithStartWorkflow(context.Context, *ClientSignalWithStartWorkflowInput) (WorkflowRun, error)

	// CancelWorkflow intercepts client.Client.CancelWorkflow.
	CancelWorkflow(context.Context, *ClientCancelWorkflowInput) error

	// TerminateWorkflow intercepts client.Client.TerminateWorkflow.
	TerminateWorkflow(context.Context, *ClientTerminateWorkflowInput) error

	// QueryWorkflow intercepts client.Client.QueryWorkflow.
	// If the query is rejected, QueryWorkflow will return an QueryRejectedError
	// interceptor.Header will return a non-nil map for this context.
	QueryWorkflow(context.Context, *ClientQueryWorkflowInput) (converter.EncodedValue, error)

	// UpdateWorkflow intercepts client.Client.UpdateWorkflow
	// interceptor.Header will return a non-nil map for this context.
	UpdateWorkflow(context.Context, *ClientUpdateWorkflowInput) (WorkflowUpdateHandle, error)

	// UpdateWithStartWorkflow intercepts client.Client.UpdateWithStartWorkflow.
	UpdateWithStartWorkflow(context.Context, *ClientUpdateWithStartWorkflowInput) (WorkflowUpdateHandle, error)

	// PollWorkflowUpdate requests the outcome of a specific update from the
	// server.
	PollWorkflowUpdate(context.Context, *ClientPollWorkflowUpdateInput) (*ClientPollWorkflowUpdateOutput, error)

	// DescribeWorkflow intercepts client.Client.DescribeWorkflow.
	DescribeWorkflow(context.Context, *ClientDescribeWorkflowInput) (*ClientDescribeWorkflowOutput, error)

	mustEmbedClientOutboundInterceptorBase()
}

// ClientUpdateWorkflowInput is the input to
// ClientOutboundInterceptor.UpdateWorkflow
//
// Exposed as: [go.temporal.io/sdk/interceptor.ClientUpdateWorkflowInput]
type ClientUpdateWorkflowInput struct {
	UpdateID            string
	WorkflowID          string
	UpdateName          string
	Args                []interface{}
	RunID               string
	FirstExecutionRunID string
	WaitForStage        WorkflowUpdateStage
}

// Exposed as: [go.temporal.io/sdk/interceptor.ClientUpdateWithStartWorkflowInput]
type ClientUpdateWithStartWorkflowInput struct {
	UpdateOptions          *UpdateWorkflowOptions
	StartWorkflowOperation WithStartWorkflowOperation
}

// ClientPollWorkflowUpdateInput is the input to
// ClientOutboundInterceptor.PollWorkflowUpdate.
type ClientPollWorkflowUpdateInput struct {
	UpdateRef *updatepb.UpdateRef
}

// ClientPollWorkflowUpdateOutput is the output to
// ClientOutboundInterceptor.PollWorkflowUpdate.
type ClientPollWorkflowUpdateOutput struct {
	// Result is the result of the update, if it has completed successfully.
	Result converter.EncodedValue
	// Error is the result of a failed update.
	Error error
}

// ScheduleClientCreateInput is the input to
// ClientOutboundInterceptor.CreateSchedule.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ScheduleClientCreateInput]
type ScheduleClientCreateInput struct {
	Options *ScheduleOptions
}

// ClientExecuteWorkflowInput is the input to
// ClientOutboundInterceptor.ExecuteWorkflow.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ClientExecuteWorkflowInput]
type ClientExecuteWorkflowInput struct {
	Options      *StartWorkflowOptions
	WorkflowType string
	Args         []interface{}
}

// ClientSignalWorkflowInput is the input to
// ClientOutboundInterceptor.SignalWorkflow.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ClientSignalWorkflowInput]
type ClientSignalWorkflowInput struct {
	WorkflowID string
	RunID      string
	SignalName string
	Arg        interface{}
}

// ClientSignalWithStartWorkflowInput is the input to
// ClientOutboundInterceptor.SignalWithStartWorkflow.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ClientSignalWithStartWorkflowInput]
type ClientSignalWithStartWorkflowInput struct {
	SignalName   string
	SignalArg    interface{}
	Options      *StartWorkflowOptions
	WorkflowType string
	Args         []interface{}
}

// ClientCancelWorkflowInput is the input to
// ClientOutboundInterceptor.CancelWorkflow.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ClientCancelWorkflowInput]
type ClientCancelWorkflowInput struct {
	WorkflowID string
	RunID      string
}

// ClientTerminateWorkflowInput is the input to
// ClientOutboundInterceptor.TerminateWorkflow.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ClientTerminateWorkflowInput]
type ClientTerminateWorkflowInput struct {
	WorkflowID string
	RunID      string
	Reason     string
	Details    []interface{}
}

// ClientQueryWorkflowInput is the input to
// ClientOutboundInterceptor.QueryWorkflow.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ClientQueryWorkflowInput]
type ClientQueryWorkflowInput struct {
	WorkflowID           string
	RunID                string
	QueryType            string
	Args                 []interface{}
	QueryRejectCondition enumspb.QueryRejectCondition
}

// ClientDescribeWorkflowInput is the input to
// ClientOutboundInterceptor.DescribeWorkflow.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ClientDescribeWorkflowInput]
type ClientDescribeWorkflowInput struct {
	WorkflowID string
	RunID      string
}

// ClientDescribeWorkflowInput is the output to
// ClientOutboundInterceptor.DescribeWorkflow.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ClientDescribeWorkflowOutput]
type ClientDescribeWorkflowOutput struct {
	Response *WorkflowExecutionDescription
}

// NexusOutboundInterceptor intercepts Nexus operation method invocations. See documentation in the interceptor package
// for more details.
//
// Exposed as: [go.temporal.io/sdk/interceptor.NexusOperationInboundInterceptor]
//
// NOTE: Experimental
type NexusOperationInboundInterceptor interface {
	// Init is the first call of this interceptor. Implementations can change/wrap
	// the outbound interceptor before calling Init on the next interceptor.
	Init(ctx context.Context, outbound NexusOperationOutboundInterceptor) error

	// StartOperation intercepts inbound Nexus StartOperation calls.
	StartOperation(ctx context.Context, input NexusStartOperationInput) (nexus.HandlerStartOperationResult[any], error)
	// StartOperation intercepts inbound Nexus CancelOperation calls.
	CancelOperation(ctx context.Context, input NexusCancelOperationInput) error

	mustEmbedNexusOperationInboundInterceptorBase()
}

// NexusOperationOutboundInterceptor intercepts methods exposed in the temporalnexus package. See documentation in the
// interceptor package for more details.
//
// Exposed as: [go.temporal.io/sdk/interceptor.NexusOperationOutboundInterceptor]
//
// Note: Experimental
type NexusOperationOutboundInterceptor interface {
	// GetClient intercepts temporalnexus.GetClient.
	GetClient(ctx context.Context) Client
	// GetLogger intercepts temporalnexus.GetLogger.
	GetLogger(ctx context.Context) log.Logger
	// GetMetricsHandler intercepts temporalnexus.GetMetricsHandler.
	GetMetricsHandler(ctx context.Context) metrics.Handler

	mustEmbedNexusOperationOutboundInterceptorBase()
}

// NexusStartOperationInput is the input to NexusOperationInboundInterceptor.StartOperation.
//
// Exposed as: [go.temporal.io/sdk/interceptor.NexusStartOperationInput]
//
// Note: Experimental
type NexusStartOperationInput struct {
	Input   any
	Options nexus.StartOperationOptions
}

// NexusCancelOperationInput is the input to NexusOperationInboundInterceptor.CancelOperation.
//
// Exposed as: [go.temporal.io/sdk/interceptor.NexusCancelOperationInput]
//
// Note: Experimental
type NexusCancelOperationInput struct {
	Token   string
	Options nexus.CancelOperationOptions
}
