package internal

import (
	"context"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/log"
)

// InterceptorBase is a default implementation of Interceptor meant for
// embedding. See documentation in the interceptor package for more details.
//
// Exposed as: [go.temporal.io/sdk/interceptor.InterceptorBase]
type InterceptorBase struct {
	ClientInterceptorBase
	WorkerInterceptorBase
}

// WorkerInterceptorBase is a default implementation of WorkerInterceptor meant
// for embedding. See documentation in the interceptor package for more details.
//
// Exposed as: [go.temporal.io/sdk/interceptor.WorkerInterceptorBase]
type WorkerInterceptorBase struct{}

// Exposed as: [go.temporal.io/sdk/interceptor.WorkerInterceptor]
var _ WorkerInterceptor = &WorkerInterceptorBase{}

// InterceptActivity implements WorkerInterceptor.InterceptActivity.
func (*WorkerInterceptorBase) InterceptActivity(
	ctx context.Context,
	next ActivityInboundInterceptor,
) ActivityInboundInterceptor {
	return &ActivityInboundInterceptorBase{Next: next}
}

// InterceptWorkflow implements WorkerInterceptor.InterceptWorkflow.
func (*WorkerInterceptorBase) InterceptWorkflow(
	ctx Context,
	next WorkflowInboundInterceptor,
) WorkflowInboundInterceptor {
	return &WorkflowInboundInterceptorBase{Next: next}
}

// InterceptNexusOperation implements WorkerInterceptor.
func (w *WorkerInterceptorBase) InterceptNexusOperation(ctx context.Context, next NexusOperationInboundInterceptor) NexusOperationInboundInterceptor {
	return &NexusOperationInboundInterceptorBase{Next: next}
}

func (*WorkerInterceptorBase) mustEmbedWorkerInterceptorBase() {}

// ActivityInboundInterceptorBase is a default implementation of
// ActivityInboundInterceptor meant for embedding. See documentation in the
// interceptor package for more details.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ActivityInboundInterceptorBase]
type ActivityInboundInterceptorBase struct {
	Next ActivityInboundInterceptor
}

// Exposed as: [go.temporal.io/sdk/interceptor.ActivityInboundInterceptor]
var _ ActivityInboundInterceptor = &ActivityInboundInterceptorBase{}

// Init implements ActivityInboundInterceptor.Init.
func (a *ActivityInboundInterceptorBase) Init(outbound ActivityOutboundInterceptor) error {
	return a.Next.Init(outbound)
}

// ExecuteActivity implements ActivityInboundInterceptor.ExecuteActivity.
func (a *ActivityInboundInterceptorBase) ExecuteActivity(
	ctx context.Context,
	in *ExecuteActivityInput,
) (interface{}, error) {
	return a.Next.ExecuteActivity(ctx, in)
}

func (*ActivityInboundInterceptorBase) mustEmbedActivityInboundInterceptorBase() {}

// ActivityOutboundInterceptorBase is a default implementation of
// ActivityOutboundInterceptor meant for embedding. See documentation in the
// interceptor package for more details.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ActivityOutboundInterceptorBase]
type ActivityOutboundInterceptorBase struct {
	Next ActivityOutboundInterceptor
}

// Exposed as: [go.temporal.io/sdk/interceptor.ActivityOutboundInterceptor]
var _ ActivityOutboundInterceptor = &ActivityOutboundInterceptorBase{}

// GetInfo implements ActivityOutboundInterceptor.GetInfo.
func (a *ActivityOutboundInterceptorBase) GetInfo(ctx context.Context) ActivityInfo {
	return a.Next.GetInfo(ctx)
}

// GetLogger implements ActivityOutboundInterceptor.GetLogger.
func (a *ActivityOutboundInterceptorBase) GetLogger(ctx context.Context) log.Logger {
	return a.Next.GetLogger(ctx)
}

// GetMetricsHandler implements ActivityOutboundInterceptor.GetMetricsHandler.
func (a *ActivityOutboundInterceptorBase) GetMetricsHandler(ctx context.Context) metrics.Handler {
	return a.Next.GetMetricsHandler(ctx)
}

// RecordHeartbeat implements ActivityOutboundInterceptor.RecordHeartbeat.
func (a *ActivityOutboundInterceptorBase) RecordHeartbeat(ctx context.Context, details ...interface{}) {
	a.Next.RecordHeartbeat(ctx, details...)
}

// HasHeartbeatDetails implements
// ActivityOutboundInterceptor.HasHeartbeatDetails.
func (a *ActivityOutboundInterceptorBase) HasHeartbeatDetails(ctx context.Context) bool {
	return a.Next.HasHeartbeatDetails(ctx)
}

// GetHeartbeatDetails implements
// ActivityOutboundInterceptor.GetHeartbeatDetails.
func (a *ActivityOutboundInterceptorBase) GetHeartbeatDetails(ctx context.Context, d ...interface{}) error {
	return a.Next.GetHeartbeatDetails(ctx, d...)
}

// GetWorkerStopChannel implements
// ActivityOutboundInterceptor.GetWorkerStopChannel.
func (a *ActivityOutboundInterceptorBase) GetWorkerStopChannel(ctx context.Context) <-chan struct{} {
	return a.Next.GetWorkerStopChannel(ctx)
}

// GetClient implements
// ActivityOutboundInterceptor.GetClient
func (a *ActivityOutboundInterceptorBase) GetClient(ctx context.Context) Client {
	return a.Next.GetClient(ctx)
}

func (*ActivityOutboundInterceptorBase) mustEmbedActivityOutboundInterceptorBase() {}

// WorkflowInboundInterceptorBase is a default implementation of
// WorkflowInboundInterceptor meant for embedding. See documentation in the
// interceptor package for more details.
//
// Exposed as: [go.temporal.io/sdk/interceptor.WorkflowInboundInterceptorBase]
type WorkflowInboundInterceptorBase struct {
	Next WorkflowInboundInterceptor
}

// Exposed as: [go.temporal.io/sdk/interceptor.WorkflowInboundInterceptor]
var _ WorkflowInboundInterceptor = &WorkflowInboundInterceptorBase{}

// Init implements WorkflowInboundInterceptor.Init.
func (w *WorkflowInboundInterceptorBase) Init(outbound WorkflowOutboundInterceptor) error {
	return w.Next.Init(outbound)
}

// ExecuteWorkflow implements WorkflowInboundInterceptor.ExecuteWorkflow.
func (w *WorkflowInboundInterceptorBase) ExecuteWorkflow(ctx Context, in *ExecuteWorkflowInput) (interface{}, error) {
	return w.Next.ExecuteWorkflow(ctx, in)
}

// HandleSignal implements WorkflowInboundInterceptor.HandleSignal.
func (w *WorkflowInboundInterceptorBase) HandleSignal(ctx Context, in *HandleSignalInput) error {
	return w.Next.HandleSignal(ctx, in)
}

// ExecuteUpdate implements WorkflowInboundInterceptor.ExecuteUpdate.
func (w *WorkflowInboundInterceptorBase) ExecuteUpdate(ctx Context, in *UpdateInput) (interface{}, error) {
	return w.Next.ExecuteUpdate(ctx, in)
}

// ValidateUpdate implements WorkflowInboundInterceptor.ValidateUpdate.
func (w *WorkflowInboundInterceptorBase) ValidateUpdate(ctx Context, in *UpdateInput) error {
	return w.Next.ValidateUpdate(ctx, in)
}

// HandleQuery implements WorkflowInboundInterceptor.HandleQuery.
func (w *WorkflowInboundInterceptorBase) HandleQuery(ctx Context, in *HandleQueryInput) (interface{}, error) {
	return w.Next.HandleQuery(ctx, in)
}

func (*WorkflowInboundInterceptorBase) mustEmbedWorkflowInboundInterceptorBase() {}

// WorkflowOutboundInterceptorBase is a default implementation of
// WorkflowOutboundInterceptor meant for embedding. See documentation in the
// interceptor package for more details.
//
// Exposed as: [go.temporal.io/sdk/interceptor.WorkflowOutboundInterceptorBase]
type WorkflowOutboundInterceptorBase struct {
	Next WorkflowOutboundInterceptor
}

// Exposed as: [go.temporal.io/sdk/interceptor.WorkflowOutboundInterceptor]
var _ WorkflowOutboundInterceptor = &WorkflowOutboundInterceptorBase{}

// Go implements WorkflowOutboundInterceptor.Go.
func (w *WorkflowOutboundInterceptorBase) Go(ctx Context, name string, f func(ctx Context)) Context {
	return w.Next.Go(ctx, name, f)
}

// ExecuteActivity implements WorkflowOutboundInterceptor.ExecuteActivity.
func (w *WorkflowOutboundInterceptorBase) ExecuteActivity(ctx Context, activityType string, args ...interface{}) Future {
	return w.Next.ExecuteActivity(ctx, activityType, args...)
}

// Await implements WorkflowOutboundInterceptor.Await.
func (w *WorkflowOutboundInterceptorBase) Await(ctx Context, condition func() bool) error {
	return w.Next.Await(ctx, condition)
}

// AwaitWithTimeout implements WorkflowOutboundInterceptor.AwaitWithTimeout.
func (w *WorkflowOutboundInterceptorBase) AwaitWithTimeout(ctx Context, timeout time.Duration, condition func() bool) (bool, error) {
	return w.Next.AwaitWithTimeout(ctx, timeout, condition)
}

// AwaitWithOptions implements WorkflowOutboundInterceptor.AwaitWithOptions.
//
// NOTE: Experimental
func (w *WorkflowOutboundInterceptorBase) AwaitWithOptions(ctx Context, options AwaitOptions, condition func() bool) (bool, error) {
	return w.Next.AwaitWithOptions(ctx, options, condition)
}

// ExecuteLocalActivity implements WorkflowOutboundInterceptor.ExecuteLocalActivity.
func (w *WorkflowOutboundInterceptorBase) ExecuteLocalActivity(
	ctx Context,
	activityType string,
	args ...interface{},
) Future {
	return w.Next.ExecuteLocalActivity(ctx, activityType, args...)
}

// ExecuteChildWorkflow implements WorkflowOutboundInterceptor.ExecuteChildWorkflow.
func (w *WorkflowOutboundInterceptorBase) ExecuteChildWorkflow(
	ctx Context,
	childWorkflowType string,
	args ...interface{},
) ChildWorkflowFuture {
	return w.Next.ExecuteChildWorkflow(ctx, childWorkflowType, args...)
}

// GetInfo implements WorkflowOutboundInterceptor.GetInfo.
func (w *WorkflowOutboundInterceptorBase) GetInfo(ctx Context) *WorkflowInfo {
	return w.Next.GetInfo(ctx)
}

// GetTypedSearchAttributes implements WorkflowOutboundInterceptor.GetTypedSearchAttributes.
func (w *WorkflowOutboundInterceptorBase) GetTypedSearchAttributes(ctx Context) SearchAttributes {
	return w.Next.GetTypedSearchAttributes(ctx)
}

// GetCurrentUpdateInfo implements WorkflowOutboundInterceptor.GetCurrentUpdateInfo.
func (w *WorkflowOutboundInterceptorBase) GetCurrentUpdateInfo(ctx Context) *UpdateInfo {
	return w.Next.GetCurrentUpdateInfo(ctx)
}

// GetLogger implements WorkflowOutboundInterceptor.GetLogger.
func (w *WorkflowOutboundInterceptorBase) GetLogger(ctx Context) log.Logger {
	return w.Next.GetLogger(ctx)
}

// GetMetricsHandler implements WorkflowOutboundInterceptor.GetMetricsHandler.
func (w *WorkflowOutboundInterceptorBase) GetMetricsHandler(ctx Context) metrics.Handler {
	return w.Next.GetMetricsHandler(ctx)
}

// Now implements WorkflowOutboundInterceptor.Now.
func (w *WorkflowOutboundInterceptorBase) Now(ctx Context) time.Time {
	return w.Next.Now(ctx)
}

// NewTimer implements WorkflowOutboundInterceptor.NewTimer.
func (w *WorkflowOutboundInterceptorBase) NewTimer(ctx Context, d time.Duration) Future {
	return w.Next.NewTimer(ctx, d)
}

// NewTimerWithOptions implements WorkflowOutboundInterceptor.NewTimerWithOptions.
//
// NOTE: Experimental
func (w *WorkflowOutboundInterceptorBase) NewTimerWithOptions(
	ctx Context,
	d time.Duration,
	options TimerOptions,
) Future {
	return w.Next.NewTimerWithOptions(ctx, d, options)
}

// Sleep implements WorkflowOutboundInterceptor.Sleep.
func (w *WorkflowOutboundInterceptorBase) Sleep(ctx Context, d time.Duration) (err error) {
	return w.Next.Sleep(ctx, d)
}

// RequestCancelExternalWorkflow implements
// WorkflowOutboundInterceptor.RequestCancelExternalWorkflow.
func (w *WorkflowOutboundInterceptorBase) RequestCancelExternalWorkflow(
	ctx Context,
	workflowID string,
	runID string,
) Future {
	return w.Next.RequestCancelExternalWorkflow(ctx, workflowID, runID)
}

// SignalExternalWorkflow implements
// WorkflowOutboundInterceptor.SignalExternalWorkflow.
func (w *WorkflowOutboundInterceptorBase) SignalExternalWorkflow(
	ctx Context,
	workflowID string,
	runID string,
	signalName string,
	arg interface{},
) Future {
	return w.Next.SignalExternalWorkflow(ctx, workflowID, runID, signalName, arg)
}

// SignalChildWorkflow implements
// WorkflowOutboundInterceptor.SignalChildWorkflow.
func (w *WorkflowOutboundInterceptorBase) SignalChildWorkflow(
	ctx Context,
	workflowID string,
	signalName string,
	arg interface{},
) Future {
	return w.Next.SignalChildWorkflow(ctx, workflowID, signalName, arg)
}

// UpsertSearchAttributes implements
// WorkflowOutboundInterceptor.UpsertSearchAttributes.
func (w *WorkflowOutboundInterceptorBase) UpsertSearchAttributes(ctx Context, attributes map[string]interface{}) error {
	return w.Next.UpsertSearchAttributes(ctx, attributes)
}

// UpsertTypedSearchAttributes implements
// WorkflowOutboundInterceptor.UpsertTypedSearchAttributes.
func (w *WorkflowOutboundInterceptorBase) UpsertTypedSearchAttributes(ctx Context, attributes ...SearchAttributeUpdate) error {
	return w.Next.UpsertTypedSearchAttributes(ctx, attributes...)
}

// UpsertMemo implements
// WorkflowOutboundInterceptor.UpsertMemo.
func (w *WorkflowOutboundInterceptorBase) UpsertMemo(ctx Context, memo map[string]interface{}) error {
	return w.Next.UpsertMemo(ctx, memo)
}

// GetSignalChannel implements WorkflowOutboundInterceptor.GetSignalChannel.
func (w *WorkflowOutboundInterceptorBase) GetSignalChannel(ctx Context, signalName string) ReceiveChannel {
	return w.Next.GetSignalChannel(ctx, signalName)
}

// GetSignalChannelWithOptions implements WorkflowOutboundInterceptor.GetSignalChannelWithOptions.
//
// NOTE: Experimental
func (w *WorkflowOutboundInterceptorBase) GetSignalChannelWithOptions(
	ctx Context,
	signalName string,
	options SignalChannelOptions,
) ReceiveChannel {
	return w.Next.GetSignalChannelWithOptions(ctx, signalName, options)
}

// SideEffect implements WorkflowOutboundInterceptor.SideEffect.
func (w *WorkflowOutboundInterceptorBase) SideEffect(
	ctx Context,
	f func(ctx Context) interface{},
) converter.EncodedValue {
	return w.Next.SideEffect(ctx, f)
}

// MutableSideEffect implements WorkflowOutboundInterceptor.MutableSideEffect.
func (w *WorkflowOutboundInterceptorBase) MutableSideEffect(
	ctx Context,
	id string,
	f func(ctx Context) interface{},
	equals func(a, b interface{}) bool,
) converter.EncodedValue {
	return w.Next.MutableSideEffect(ctx, id, f, equals)
}

// GetVersion implements WorkflowOutboundInterceptor.GetVersion.
func (w *WorkflowOutboundInterceptorBase) GetVersion(
	ctx Context,
	changeID string,
	minSupported Version,
	maxSupported Version,
) Version {
	return w.Next.GetVersion(ctx, changeID, minSupported, maxSupported)
}

// SetQueryHandler implements WorkflowOutboundInterceptor.SetQueryHandler.
func (w *WorkflowOutboundInterceptorBase) SetQueryHandler(ctx Context, queryType string, handler interface{}) error {
	return w.Next.SetQueryHandler(ctx, queryType, handler)
}

// SetQueryHandlerWithOptions implements WorkflowOutboundInterceptor.SetQueryHandlerWithOptions.
//
// NOTE: Experimental
func (w *WorkflowOutboundInterceptorBase) SetQueryHandlerWithOptions(
	ctx Context,
	queryType string,
	handler interface{},
	options QueryHandlerOptions,
) error {
	return w.Next.SetQueryHandlerWithOptions(ctx, queryType, handler, options)
}

// SetUpdateHandler implements WorkflowOutboundInterceptor.SetUpdateHandler.
func (w *WorkflowOutboundInterceptorBase) SetUpdateHandler(ctx Context, updateName string, handler interface{}, opts UpdateHandlerOptions) error {
	return w.Next.SetUpdateHandler(ctx, updateName, handler, opts)
}

// IsReplaying implements WorkflowOutboundInterceptor.IsReplaying.
func (w *WorkflowOutboundInterceptorBase) IsReplaying(ctx Context) bool {
	return w.Next.IsReplaying(ctx)
}

// HasLastCompletionResult implements
// WorkflowOutboundInterceptor.HasLastCompletionResult.
func (w *WorkflowOutboundInterceptorBase) HasLastCompletionResult(ctx Context) bool {
	return w.Next.HasLastCompletionResult(ctx)
}

// GetLastCompletionResult implements
// WorkflowOutboundInterceptor.GetLastCompletionResult.
func (w *WorkflowOutboundInterceptorBase) GetLastCompletionResult(ctx Context, d ...interface{}) error {
	return w.Next.GetLastCompletionResult(ctx, d...)
}

// GetLastError implements WorkflowOutboundInterceptor.GetLastError.
func (w *WorkflowOutboundInterceptorBase) GetLastError(ctx Context) error {
	return w.Next.GetLastError(ctx)
}

// NewContinueAsNewError implements
// WorkflowOutboundInterceptor.NewContinueAsNewError.
func (w *WorkflowOutboundInterceptorBase) NewContinueAsNewError(
	ctx Context,
	wfn interface{},
	args ...interface{},
) error {
	return w.Next.NewContinueAsNewError(ctx, wfn, args...)
}

// ExecuteNexusOperation implements
// WorkflowOutboundInterceptor.ExecuteNexusOperation.
func (w *WorkflowOutboundInterceptorBase) ExecuteNexusOperation(
	ctx Context,
	input ExecuteNexusOperationInput,
) NexusOperationFuture {
	return w.Next.ExecuteNexusOperation(ctx, input)
}

// RequestCancelNexusOperation implements
// WorkflowOutboundInterceptor.RequestCancelNexusOperation.
func (w *WorkflowOutboundInterceptorBase) RequestCancelNexusOperation(ctx Context, input RequestCancelNexusOperationInput) {
	w.Next.RequestCancelNexusOperation(ctx, input)
}

func (*WorkflowOutboundInterceptorBase) mustEmbedWorkflowOutboundInterceptorBase() {}

// ClientInterceptorBase is a default implementation of ClientInterceptor meant
// for embedding. See documentation in the interceptor package for more details.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ClientInterceptorBase]
type ClientInterceptorBase struct{}

// Exposed as: [go.temporal.io/sdk/interceptor.ClientInterceptor]
var _ ClientInterceptor = &ClientInterceptorBase{}

// InterceptClient implements ClientInterceptor.InterceptClient.
func (*ClientInterceptorBase) InterceptClient(
	next ClientOutboundInterceptor,
) ClientOutboundInterceptor {
	return &ClientOutboundInterceptorBase{Next: next}
}

func (*ClientInterceptorBase) mustEmbedClientInterceptorBase() {}

// ClientOutboundInterceptorBase is a default implementation of
// ClientOutboundInterceptor meant for embedding. See documentation in the
// interceptor package for more details.
//
// Exposed as: [go.temporal.io/sdk/interceptor.ClientOutboundInterceptorBase]
type ClientOutboundInterceptorBase struct {
	Next ClientOutboundInterceptor
}

// Exposed as: [go.temporal.io/sdk/interceptor.ClientOutboundInterceptor]
var _ ClientOutboundInterceptor = &ClientOutboundInterceptorBase{}

func (c *ClientOutboundInterceptorBase) UpdateWorkflow(
	ctx context.Context,
	in *ClientUpdateWorkflowInput,
) (WorkflowUpdateHandle, error) {
	return c.Next.UpdateWorkflow(ctx, in)
}

func (c *ClientOutboundInterceptorBase) PollWorkflowUpdate(
	ctx context.Context,
	in *ClientPollWorkflowUpdateInput,
) (*ClientPollWorkflowUpdateOutput, error) {
	return c.Next.PollWorkflowUpdate(ctx, in)
}

func (c *ClientOutboundInterceptorBase) UpdateWithStartWorkflow(
	ctx context.Context,
	in *ClientUpdateWithStartWorkflowInput,
) (WorkflowUpdateHandle, error) {
	return c.Next.UpdateWithStartWorkflow(ctx, in)
}

// ExecuteWorkflow implements ClientOutboundInterceptor.ExecuteWorkflow.
func (c *ClientOutboundInterceptorBase) ExecuteWorkflow(
	ctx context.Context,
	in *ClientExecuteWorkflowInput,
) (WorkflowRun, error) {
	return c.Next.ExecuteWorkflow(ctx, in)
}

// SignalWorkflow implements ClientOutboundInterceptor.SignalWorkflow.
func (c *ClientOutboundInterceptorBase) SignalWorkflow(ctx context.Context, in *ClientSignalWorkflowInput) error {
	return c.Next.SignalWorkflow(ctx, in)
}

// SignalWithStartWorkflow implements
// ClientOutboundInterceptor.SignalWithStartWorkflow.
func (c *ClientOutboundInterceptorBase) SignalWithStartWorkflow(
	ctx context.Context,
	in *ClientSignalWithStartWorkflowInput,
) (WorkflowRun, error) {
	return c.Next.SignalWithStartWorkflow(ctx, in)
}

// CancelWorkflow implements ClientOutboundInterceptor.CancelWorkflow.
func (c *ClientOutboundInterceptorBase) CancelWorkflow(ctx context.Context, in *ClientCancelWorkflowInput) error {
	return c.Next.CancelWorkflow(ctx, in)
}

// TerminateWorkflow implements ClientOutboundInterceptor.TerminateWorkflow.
func (c *ClientOutboundInterceptorBase) TerminateWorkflow(ctx context.Context, in *ClientTerminateWorkflowInput) error {
	return c.Next.TerminateWorkflow(ctx, in)
}

// QueryWorkflow implements ClientOutboundInterceptor.QueryWorkflow.
func (c *ClientOutboundInterceptorBase) QueryWorkflow(
	ctx context.Context,
	in *ClientQueryWorkflowInput,
) (converter.EncodedValue, error) {
	return c.Next.QueryWorkflow(ctx, in)
}

// DescribeWorkflow implements ClientOutboundInterceptor.DescribeWorkflow.
func (c *ClientOutboundInterceptorBase) DescribeWorkflow(
	ctx context.Context,
	in *ClientDescribeWorkflowInput,
) (*ClientDescribeWorkflowOutput, error) {
	return c.Next.DescribeWorkflow(ctx, in)
}

// ExecuteWorkflow implements ClientOutboundInterceptor.CreateSchedule.
func (c *ClientOutboundInterceptorBase) CreateSchedule(ctx context.Context, in *ScheduleClientCreateInput) (ScheduleHandle, error) {
	return c.Next.CreateSchedule(ctx, in)
}

func (*ClientOutboundInterceptorBase) mustEmbedClientOutboundInterceptorBase() {}

// NexusOperationInboundInterceptorBase is a default implementation of [NexusOperationInboundInterceptor] that
// forwards calls to the next inbound interceptor.
//
// Note: Experimental
type NexusOperationInboundInterceptorBase struct {
	Next NexusOperationInboundInterceptor
}

// CancelOperation implements NexusOperationInboundInterceptor.
func (n *NexusOperationInboundInterceptorBase) CancelOperation(ctx context.Context, input NexusCancelOperationInput) error {
	return n.Next.CancelOperation(ctx, input)
}

// Init implements NexusOperationInboundInterceptor.
func (n *NexusOperationInboundInterceptorBase) Init(ctx context.Context, outbound NexusOperationOutboundInterceptor) error {
	return n.Next.Init(ctx, outbound)
}

// StartOperation implements NexusOperationInboundInterceptor.
func (n *NexusOperationInboundInterceptorBase) StartOperation(ctx context.Context, input NexusStartOperationInput) (nexus.HandlerStartOperationResult[any], error) {
	return n.Next.StartOperation(ctx, input)
}

// mustEmbedNexusOperationInboundInterceptorBase implements NexusOperationInboundInterceptor.
func (n *NexusOperationInboundInterceptorBase) mustEmbedNexusOperationInboundInterceptorBase() {}

var _ NexusOperationInboundInterceptor = &NexusOperationInboundInterceptorBase{}

// NexusOperationOutboundInterceptorBase is a default implementation of [NexusOperationOutboundInterceptor] that
// forwards calls to the next outbound interceptor.
//
// Note: Experimental
type NexusOperationOutboundInterceptorBase struct {
	Next NexusOperationOutboundInterceptor
}

// GetClient implements NexusOperationOutboundInterceptor.
func (n *NexusOperationOutboundInterceptorBase) GetClient(ctx context.Context) Client {
	return n.Next.GetClient(ctx)
}

// GetLogger implements NexusOperationOutboundInterceptor.
func (n *NexusOperationOutboundInterceptorBase) GetLogger(ctx context.Context) log.Logger {
	return n.Next.GetLogger(ctx)
}

// GetMetricsHandler implements NexusOperationOutboundInterceptor.
func (n *NexusOperationOutboundInterceptorBase) GetMetricsHandler(ctx context.Context) metrics.Handler {
	return n.Next.GetMetricsHandler(ctx)
}

// mustEmbedNexusOperationOutboundInterceptorBase implements NexusOperationOutboundInterceptor.
func (n *NexusOperationOutboundInterceptorBase) mustEmbedNexusOperationOutboundInterceptorBase() {}

var _ NexusOperationOutboundInterceptor = &NexusOperationOutboundInterceptorBase{}
