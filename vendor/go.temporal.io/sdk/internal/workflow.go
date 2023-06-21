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
	"strings"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/log"
)

var (
	errWorkflowIDNotSet              = errors.New("workflowId is not set")
	errLocalActivityParamsBadRequest = errors.New("missing local activity parameters through context, check LocalActivityOptions")
	errSearchAttributesNotSet        = errors.New("search attributes is empty")
	errMemoNotSet                    = errors.New("memo is empty")
)

type (
	// SendChannel is a write only view of the Channel
	SendChannel interface {
		// Send blocks until the data is sent.
		Send(ctx Context, v interface{})

		// SendAsync try to send without blocking. It returns true if the data was sent, otherwise it returns false.
		SendAsync(v interface{}) (ok bool)

		// Close close the Channel, and prohibit subsequent sends.
		Close()
	}

	// ReceiveChannel is a read only view of the Channel
	ReceiveChannel interface {
		// Receive blocks until it receives a value, and then assigns the received value to the provided pointer.
		// Returns false when Channel is closed.
		// Parameter valuePtr is a pointer to the expected data structure to be received. For example:
		//  var v string
		//  c.Receive(ctx, &v)
		//
		// Note, values should not be reused for extraction here because merging on
		// top of existing values may result in unexpected behavior similar to
		// json.Unmarshal.
		Receive(ctx Context, valuePtr interface{}) (more bool)

		// ReceiveWithTimeout blocks up to timeout until it receives a value, and then assigns the received value to the
		// provided pointer.
		// Returns more value of false when Channel is closed.
		// Returns ok value of false when no value was found in the channel for the duration of timeout or
		// the ctx was canceled.
		// The valuePtr is not modified if ok is false.
		// Parameter valuePtr is a pointer to the expected data structure to be received. For example:
		//  var v string
		//  c.ReceiveWithTimeout(ctx, time.Minute, &v)
		//
		// Note, values should not be reused for extraction here because merging on
		// top of existing values may result in unexpected behavior similar to
		// json.Unmarshal.
		ReceiveWithTimeout(ctx Context, timeout time.Duration, valuePtr interface{}) (ok, more bool)

		// ReceiveAsync try to receive from Channel without blocking. If there is data available from the Channel, it
		// assign the data to valuePtr and returns true. Otherwise, it returns false immediately.
		//
		// Note, values should not be reused for extraction here because merging on
		// top of existing values may result in unexpected behavior similar to
		// json.Unmarshal.
		ReceiveAsync(valuePtr interface{}) (ok bool)

		// ReceiveAsyncWithMoreFlag is same as ReceiveAsync with extra return value more to indicate if there could be
		// more value from the Channel. The more is false when Channel is closed.
		//
		// Note, values should not be reused for extraction here because merging on
		// top of existing values may result in unexpected behavior similar to
		// json.Unmarshal.
		ReceiveAsyncWithMoreFlag(valuePtr interface{}) (ok bool, more bool)

		// Len returns the number of buffered messages plus the number of blocked Send calls.
		Len() int
	}

	// Channel must be used instead of native go channel by workflow code.
	// Use workflow.NewChannel(ctx) method to create Channel instance.
	Channel interface {
		SendChannel
		ReceiveChannel
	}

	// Selector must be used instead of native go select by workflow code.
	// Create through workflow.NewSelector(ctx).
	Selector interface {
		// AddReceive registers a callback function to be called when a channel has a message to receive.
		// The callback is called when Select(ctx) is called.
		// The message is expected be consumed by the callback function.
		// The branch is automatically removed after the channel is closed and callback function is called once
		// with more parameter set to false.
		AddReceive(c ReceiveChannel, f func(c ReceiveChannel, more bool)) Selector
		// AddSend registers a callback function to be called when sending message to channel is not going to block.
		// The callback is called when Select(ctx) is called.
		// The sending message to the channel is expected to be done by the callback function
		AddSend(c SendChannel, v interface{}, f func()) Selector
		// AddFuture registers a callback function to be called when a future is ready.
		// The callback is called when Select(ctx) is called.
		// The callback is called once per ready future even if Select is called multiple times for the same
		// Selector instance.
		AddFuture(future Future, f func(f Future)) Selector
		// AddDefault register callback function to be called if none of other branches matched.
		// The callback is called when Select(ctx) is called.
		// When the default branch is registered Select never blocks.
		AddDefault(f func())
		// Select checks if any of the registered branches satisfies its condition blocking if necessary.
		// When a branch becomes eligible its callback is invoked.
		// If multiple branches are eligible only one of them (picked randomly) is invoked per Select call.
		// It is OK to call Select multiple times for the same Selector instance.
		Select(ctx Context)
		// HasPending returns true if call to Select is guaranteed to not block.
		HasPending() bool
	}

	// WaitGroup must be used instead of native go sync.WaitGroup by
	// workflow code.  Use workflow.NewWaitGroup(ctx) method to create
	// a new WaitGroup instance
	WaitGroup interface {
		Add(delta int)
		Done()
		Wait(ctx Context)
	}

	// Future represents the result of an asynchronous computation.
	Future interface {
		// Get blocks until the future is ready. When ready it either returns non nil error or assigns result value to
		// the provided pointer.
		// Example:
		//  var v string
		//  if err := f.Get(ctx, &v); err != nil {
		//      return err
		//  }
		//
		// The valuePtr parameter can be nil when the encoded result value is not needed.
		// Example:
		//  err = f.Get(ctx, nil)
		//
		// Note, values should not be reused for extraction here because merging on
		// top of existing values may result in unexpected behavior similar to
		// json.Unmarshal.
		Get(ctx Context, valuePtr interface{}) error

		// When true Get is guaranteed to not block
		IsReady() bool
	}

	// Settable is used to set value or error on a future.
	// See more: workflow.NewFuture(ctx).
	Settable interface {
		Set(value interface{}, err error)
		SetValue(value interface{})
		SetError(err error)
		Chain(future Future) // EncodedValue (or error) of the future become the same of the chained one.
	}

	// ChildWorkflowFuture represents the result of a child workflow execution
	ChildWorkflowFuture interface {
		Future
		// GetChildWorkflowExecution returns a future that will be ready when child workflow execution started. You can
		// get the WorkflowExecution of the child workflow from the future. Then you can use Workflow ID and RunID of
		// child workflow to cancel or send signal to child workflow.
		//  childWorkflowFuture := workflow.ExecuteChildWorkflow(ctx, child, ...)
		//  var childWE workflow.Execution
		//  if err := childWorkflowFuture.GetChildWorkflowExecution().Get(ctx, &childWE); err == nil {
		//      // child workflow started, you can use childWE to get the WorkflowID and RunID of child workflow
		//  }
		GetChildWorkflowExecution() Future

		// SignalChildWorkflow sends a signal to the child workflow. This call will block until child workflow is started.
		SignalChildWorkflow(ctx Context, signalName string, data interface{}) Future
	}

	// WorkflowType identifies a workflow type.
	WorkflowType struct {
		Name string
	}

	// WorkflowExecution details.
	WorkflowExecution struct {
		ID    string
		RunID string
	}

	// EncodedValue is type used to encapsulate/extract encoded result from workflow/activity.
	EncodedValue struct {
		value         *commonpb.Payloads
		dataConverter converter.DataConverter
	}
	// Version represents a change version. See GetVersion call.
	Version int

	// ChildWorkflowOptions stores all child workflow specific parameters that will be stored inside of a Context.
	// The current timeout resolution implementation is in seconds and uses math.Ceil(d.Seconds()) as the duration. But is
	// subjected to change in the future.
	ChildWorkflowOptions struct {
		// Namespace of the child workflow.
		// Optional: the current workflow (parent)'s namespace will be used if this is not provided.
		Namespace string

		// WorkflowID of the child workflow to be scheduled.
		// Optional: an auto generated workflowID will be used if this is not provided.
		WorkflowID string

		// TaskQueue that the child workflow needs to be scheduled on.
		// Optional: the parent workflow task queue will be used if this is not provided.
		TaskQueue string

		// WorkflowExecutionTimeout - The end to end timeout for the child workflow execution including retries
		// and continue as new.
		// Optional: defaults to unlimited.
		WorkflowExecutionTimeout time.Duration

		// WorkflowRunTimeout - The timeout for a single run of the child workflow execution. Each retry or
		// continue as new should obey this timeout. Use WorkflowExecutionTimeout to specify how long the parent
		// is willing to wait for the child completion.
		// Optional: defaults to WorkflowExecutionTimeout
		WorkflowRunTimeout time.Duration

		// WorkflowTaskTimeout - Maximum execution time of a single Workflow Task. In the majority of cases there is
		// no need to change this timeout. Note that this timeout is not related to the overall Workflow duration in
		// any way. It defines for how long the Workflow can get blocked in the case of a Workflow Worker crash.
		// Default is 10 seconds. Maximum value allowed by the Temporal Server is 1 minute.
		WorkflowTaskTimeout time.Duration

		// WaitForCancellation - Whether to wait for canceled child workflow to be ended (child workflow can be ended
		// as: completed/failed/timedout/terminated/canceled)
		// Optional: default false
		WaitForCancellation bool

		// WorkflowIDReusePolicy - Whether server allow reuse of workflow ID, can be useful
		// for dedup logic if set to WorkflowIdReusePolicyRejectDuplicate
		WorkflowIDReusePolicy enumspb.WorkflowIdReusePolicy

		// RetryPolicy specify how to retry child workflow if error happens.
		// Optional: default is no retry
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
		CronSchedule string

		// Memo - Optional non-indexed info that will be shown in list workflow.
		Memo map[string]interface{}

		// SearchAttributes - Optional indexed info that can be used in query of List/Scan/Count workflow APIs (only
		// supported when Temporal server is using ElasticSearch). The key and value type must be registered on Temporal server side.
		// Use GetSearchAttributes API to get valid key and corresponding value type.
		SearchAttributes map[string]interface{}

		// ParentClosePolicy - Optional policy to decide what to do for the child.
		// Default is Terminate (if onboarded to this feature)
		ParentClosePolicy enumspb.ParentClosePolicy

		// VersioningIntent specifies whether this child workflow should run on a worker with a
		// compatible build ID or not. See VersioningIntent.
		// WARNING: Worker versioning is currently experimental
		VersioningIntent VersioningIntent
	}

	// RegisterWorkflowOptions consists of options for registering a workflow
	RegisterWorkflowOptions struct {
		// Custom name for this workflow instead of the function name.
		//
		// If this is set, users are strongly recommended to set
		// worker.Options.DisableRegistrationAliasing at the worker level to prevent
		// ambiguity between string names and function references. Also users should
		// always use this string name when executing this workflow from a client or
		// inside a workflow as a child workflow.
		Name                          string
		DisableAlreadyRegisteredCheck bool
	}

	localActivityContext struct {
		fn       interface{}
		isMethod bool
	}

	// UpdateHandlerOptions consists of options for executing a named workflow update.
	//
	// NOTE: Experimental
	UpdateHandlerOptions struct {
		// Validator is an optional (i.e. can be left nil) func with exactly the
		// same type signature as the required update handler func but returning
		// only a single value of type error. The implementation of this
		// function MUST NOT alter workflow state in any way however it need not
		// be pure - it is permissible to observe workflow state without
		// mutating it as part of performing validation. The prohibition against
		// mutating workflow state includes normal variable mutation/assignment
		// as well as workflow actions such as scheduling activities and
		// performing side-effects. A panic from this function will be treated
		// as equivalent to returning an error.
		Validator interface{}
	}
)

// Await blocks the calling thread until condition() returns true
// Returns CanceledError if the ctx is canceled.
func Await(ctx Context, condition func() bool) error {
	state := getState(ctx)
	defer state.unblocked()

	for !condition() {
		doneCh := ctx.Done()
		// TODO: Consider always returning a channel
		if doneCh != nil {
			if _, more := doneCh.ReceiveAsyncWithMoreFlag(nil); !more {
				return NewCanceledError("Await context canceled")
			}
		}
		state.yield("Await")
	}
	return nil
}

// AwaitWithTimeout blocks the calling thread until condition() returns true
// Returns ok equals to false if timed out and err equals to CanceledError if the ctx is canceled.
func AwaitWithTimeout(ctx Context, timeout time.Duration, condition func() bool) (ok bool, err error) {
	state := getState(ctx)
	defer state.unblocked()
	timer := NewTimer(ctx, timeout)
	for !condition() {
		doneCh := ctx.Done()
		// TODO: Consider always returning a channel
		if doneCh != nil {
			if _, more := doneCh.ReceiveAsyncWithMoreFlag(nil); !more {
				return false, NewCanceledError("AwaitWithTimeout context canceled")
			}
		}
		if timer.IsReady() {
			return false, nil
		}
		state.yield("AwaitWithTimeout")
	}
	return true, nil
}

// NewChannel create new Channel instance
func NewChannel(ctx Context) Channel {
	state := getState(ctx)
	state.dispatcher.channelSequence++
	return NewNamedChannel(ctx, fmt.Sprintf("chan-%v", state.dispatcher.channelSequence))
}

// NewNamedChannel create new Channel instance with a given human readable name.
// Name appears in stack traces that are blocked on this channel.
func NewNamedChannel(ctx Context, name string) Channel {
	env := getWorkflowEnvironment(ctx)
	return &channelImpl{name: name, dataConverter: getDataConverterFromWorkflowContext(ctx), env: env}
}

// NewBufferedChannel create new buffered Channel instance
func NewBufferedChannel(ctx Context, size int) Channel {
	env := getWorkflowEnvironment(ctx)
	return &channelImpl{size: size, dataConverter: getDataConverterFromWorkflowContext(ctx), env: env}
}

// NewNamedBufferedChannel create new BufferedChannel instance with a given human readable name.
// Name appears in stack traces that are blocked on this Channel.
func NewNamedBufferedChannel(ctx Context, name string, size int) Channel {
	env := getWorkflowEnvironment(ctx)
	return &channelImpl{name: name, size: size, dataConverter: getDataConverterFromWorkflowContext(ctx), env: env}
}

// NewSelector creates a new Selector instance.
func NewSelector(ctx Context) Selector {
	state := getState(ctx)
	state.dispatcher.selectorSequence++
	return NewNamedSelector(ctx, fmt.Sprintf("selector-%v", state.dispatcher.selectorSequence))
}

// NewNamedSelector creates a new Selector instance with a given human readable name.
// Name appears in stack traces that are blocked on this Selector.
func NewNamedSelector(_ Context, name string) Selector {
	return &selectorImpl{name: name}
}

// NewWaitGroup creates a new WaitGroup instance.
func NewWaitGroup(ctx Context) WaitGroup {
	f, s := NewFuture(ctx)
	return &waitGroupImpl{future: f, settable: s}
}

// Go creates a new coroutine. It has similar semantic to goroutine in a context of the workflow.
func Go(ctx Context, f func(ctx Context)) {
	state := getState(ctx)
	state.dispatcher.interceptor.Go(ctx, "", f)
}

// GoNamed creates a new coroutine with a given human readable name.
// It has similar semantic to goroutine in a context of the workflow.
// Name appears in stack traces that are blocked on this Channel.
func GoNamed(ctx Context, name string, f func(ctx Context)) {
	state := getState(ctx)
	state.dispatcher.interceptor.Go(ctx, name, f)
}

// NewFuture creates a new future as well as associated Settable that is used to set its value.
func NewFuture(ctx Context) (Future, Settable) {
	impl := &futureImpl{channel: NewChannel(ctx).(*channelImpl)}
	return impl, impl
}

func (wc *workflowEnvironmentInterceptor) HandleSignal(ctx Context, in *HandleSignalInput) error {
	// Remove header from the context
	ctx = workflowContextWithoutHeader(ctx)

	eo := getWorkflowEnvOptions(ctx)
	// We don't want this code to be blocked ever, using sendAsync().
	ch := eo.getSignalChannel(ctx, in.SignalName).(*channelImpl)
	if !ch.SendAsync(in.Arg) {
		return fmt.Errorf("exceeded channel buffer size for signal: %v", in.SignalName)
	}
	return nil
}

func (wc *workflowEnvironmentInterceptor) ValidateUpdate(ctx Context, in *UpdateInput) error {
	eo := getWorkflowEnvOptions(ctx)

	handler, ok := eo.updateHandlers[in.Name]
	if !ok {
		keys := make([]string, 0, len(eo.updateHandlers))
		for k := range eo.updateHandlers {
			keys = append(keys, k)
		}
		return fmt.Errorf("unknown update %v. KnownUpdates=%v", in.Name, keys)
	}
	return handler.validate(ctx, in.Args)
}

func (wc *workflowEnvironmentInterceptor) ExecuteUpdate(ctx Context, in *UpdateInput) (interface{}, error) {
	eo := getWorkflowEnvOptions(ctx)

	handler, ok := eo.updateHandlers[in.Name]
	if !ok {
		keys := make([]string, 0, len(eo.updateHandlers))
		for k := range eo.updateHandlers {
			keys = append(keys, k)
		}
		return nil, fmt.Errorf("unknown update %v. KnownUpdates=%v", in.Name, keys)
	}
	return handler.execute(ctx, in.Args)
}

func (wc *workflowEnvironmentInterceptor) HandleQuery(ctx Context, in *HandleQueryInput) (interface{}, error) {
	eo := getWorkflowEnvOptions(ctx)
	handler, ok := eo.queryHandlers[in.QueryType]
	// Should never happen because its presence is checked before this call too
	if !ok {
		keys := []string{QueryTypeStackTrace, QueryTypeOpenSessions}
		for k := range eo.queryHandlers {
			keys = append(keys, k)
		}
		return nil, fmt.Errorf("unknown queryType %v. KnownQueryTypes=%v", in.QueryType, keys)
	}
	return handler.execute(in.Args)
}

func (wc *workflowEnvironmentInterceptor) ExecuteWorkflow(ctx Context, in *ExecuteWorkflowInput) (interface{}, error) {
	// Remove header from the context
	ctx = workflowContextWithoutHeader(ctx)

	// Always put the context first
	args := append([]interface{}{ctx}, in.Args...)
	return executeFunction(wc.fn, args)
}

func (wc *workflowEnvironmentInterceptor) Init(outbound WorkflowOutboundInterceptor) error {
	wc.outboundInterceptor = outbound
	return nil
}

// ExecuteActivity requests activity execution in the context of a workflow.
// Context can be used to pass the settings for this activity.
// For example: task queue that this need to be routed, timeouts that need to be configured.
// Use ActivityOptions to pass down the options.
//
//	 ao := ActivityOptions{
//		    TaskQueue: "exampleTaskQueue",
//		    ScheduleToStartTimeout: 10 * time.Second,
//		    StartToCloseTimeout: 5 * time.Second,
//		    ScheduleToCloseTimeout: 10 * time.Second,
//		    HeartbeatTimeout: 0,
//		}
//		ctx := WithActivityOptions(ctx, ao)
//
// Or to override a single option
//
//	ctx := WithTaskQueue(ctx, "exampleTaskQueue")
//
// Input activity is either an activity name (string) or a function representing an activity that is getting scheduled.
// Input args are the arguments that need to be passed to the scheduled activity.
//
// If the activity failed to complete then the future get error would indicate the failure.
// The error will be of type *ActivityError. It will have important activity information and actual error that caused
// activity failure. Use errors.Unwrap to get this error or errors.As to check it type which can be one of
// *ApplicationError, *TimeoutError, *CanceledError, or *PanicError.
//
// You can cancel the pending activity using context(workflow.WithCancel(ctx)) and that will fail the activity with
// *CanceledError set as cause for *ActivityError.
//
// ExecuteActivity returns Future with activity result or failure.
func ExecuteActivity(ctx Context, activity interface{}, args ...interface{}) Future {
	i := getWorkflowOutboundInterceptor(ctx)
	registry := getRegistryFromWorkflowContext(ctx)
	activityType := getActivityFunctionName(registry, activity)
	// Put header on context before executing
	ctx = workflowContextWithNewHeader(ctx)
	return i.ExecuteActivity(ctx, activityType, args...)
}

func (wc *workflowEnvironmentInterceptor) ExecuteActivity(ctx Context, typeName string, args ...interface{}) Future {
	// Validate type and its arguments.
	dataConverter := getDataConverterFromWorkflowContext(ctx)
	registry := getRegistryFromWorkflowContext(ctx)
	future, settable := newDecodeFuture(ctx, typeName)
	activityType, err := getValidatedActivityFunction(typeName, args, registry)
	if err != nil {
		settable.Set(nil, err)
		return future
	}
	// Validate context options.
	options := getActivityOptions(ctx)

	// Validate session state.
	if sessionInfo := getSessionInfo(ctx); sessionInfo != nil {
		isCreationActivity := isSessionCreationActivity(typeName)
		if sessionInfo.SessionState == SessionStateFailed && !isCreationActivity {
			settable.Set(nil, ErrSessionFailed)
			return future
		}
		if sessionInfo.SessionState == SessionStateOpen && !isCreationActivity {
			// Use session taskqueue
			oldTaskQueueName := options.TaskQueueName
			options.TaskQueueName = sessionInfo.taskqueue
			defer func() {
				options.TaskQueueName = oldTaskQueueName
			}()
		}
	}

	// Retrieve headers from context to pass them on
	envOptions := getWorkflowEnvOptions(ctx)
	header, err := workflowHeaderPropagated(ctx, envOptions.ContextPropagators)
	if err != nil {
		settable.Set(nil, err)
		return future
	}

	input, err := encodeArgs(dataConverter, args)
	if err != nil {
		panic(err)
	}

	params := ExecuteActivityParams{
		ExecuteActivityOptions: *options,
		ActivityType:           *activityType,
		Input:                  input,
		DataConverter:          dataConverter,
		Header:                 header,
	}

	ctxDone, cancellable := ctx.Done().(*channelImpl)
	cancellationCallback := &receiveCallback{}
	a := getWorkflowEnvironment(ctx).ExecuteActivity(params, func(r *commonpb.Payloads, e error) {
		settable.Set(r, e)
		if cancellable {
			// future is done, we don't need the cancellation callback anymore.
			ctxDone.removeReceiveCallback(cancellationCallback)
		}
	})

	if cancellable {
		cancellationCallback.fn = func(v interface{}, more bool) bool {
			if ctx.Err() == ErrCanceled {
				wc.env.RequestCancelActivity(a)
			}
			return false
		}
		_, ok, more := ctxDone.receiveAsyncImpl(cancellationCallback)
		if ok || !more {
			cancellationCallback.fn(nil, more)
		}
	}
	return future
}

// ExecuteLocalActivity requests to run a local activity. A local activity is like a regular activity with some key
// differences:
// * Local activity is scheduled and run by the workflow worker locally.
// * Local activity does not need Temporal server to schedule activity task and does not rely on activity worker.
// * No need to register local activity.
// * Local activity is for short living activities (usually finishes within seconds).
// * Local activity cannot heartbeat.
//
// Context can be used to pass the settings for this local activity.
// For now there is only one setting for timeout to be set:
//
//	 lao := LocalActivityOptions{
//		    ScheduleToCloseTimeout: 5 * time.Second,
//		}
//		ctx := WithLocalActivityOptions(ctx, lao)
//
// The timeout here should be relative shorter than the WorkflowTaskTimeout of the workflow. If you need a
// longer timeout, you probably should not use local activity and instead should use regular activity. Local activity is
// designed to be used for short living activities (usually finishes within seconds).
//
// Input args are the arguments that will to be passed to the local activity. The input args will be hand over directly
// to local activity function without serialization/deserialization because we don't need to pass the input across process
// boundary. However, the result will still go through serialization/deserialization because we need to record the result
// as history to temporal server so if the workflow crashes, a different worker can replay the history without running
// the local activity again.
//
// If the activity failed to complete then the future get error would indicate the failure.
// The error will be of type *ActivityError. It will have important activity information and actual error that caused
// activity failure. Use errors.Unwrap to get this error or errors.As to check it type which can be one of
// *ApplicationError, *TimeoutError, *CanceledError, or *PanicError.
//
// You can cancel the pending activity using context(workflow.WithCancel(ctx)) and that will fail the activity with
// *CanceledError set as cause for *ActivityError.
//
// ExecuteLocalActivity returns Future with local activity result or failure.
func ExecuteLocalActivity(ctx Context, activity interface{}, args ...interface{}) Future {
	i := getWorkflowOutboundInterceptor(ctx)
	env := getWorkflowEnvironment(ctx)
	activityType, isMethod := getFunctionName(activity)
	if alias, ok := env.GetRegistry().getActivityAlias(activityType); ok {
		activityType = alias
	}
	var fn interface{}
	if _, ok := activity.(string); ok {
		fn = nil
	} else {
		fn = activity
	}
	localCtx := &localActivityContext{
		fn:       fn,
		isMethod: isMethod,
	}
	ctx = WithValue(ctx, localActivityFnContextKey, localCtx)
	// Put header on context before executing
	ctx = workflowContextWithNewHeader(ctx)
	return i.ExecuteLocalActivity(ctx, activityType, args...)
}

func (wc *workflowEnvironmentInterceptor) ExecuteLocalActivity(ctx Context, typeName string, args ...interface{}) Future {
	future, settable := newDecodeFuture(ctx, typeName)

	envOptions := getWorkflowEnvOptions(ctx)
	header, err := workflowHeaderPropagated(ctx, envOptions.ContextPropagators)
	if err != nil {
		settable.Set(nil, err)
		return future
	}

	var activityFn interface{}
	localCtx := ctx.Value(localActivityFnContextKey).(*localActivityContext)
	if localCtx == nil {
		panic("ExecuteLocalActivity: Expected context key " + localActivityFnContextKey + " is missing")
	}

	if localCtx.isMethod {
		registry := getRegistryFromWorkflowContext(ctx)
		activity, ok := registry.GetActivity(typeName)
		// Uses registered function if found as the registration is required with a nil receiver.
		// Calls function directly if not registered. It is to support legacy applications
		// that called local activities using non nil receiver.
		if ok {
			activityFn = activity.GetFunction()
		} else {
			if err := validateFunctionArgs(localCtx.fn, args, false); err != nil {
				settable.Set(nil, err)
				return future
			}
			activityFn = localCtx.fn
		}
	} else if localCtx.fn == nil {
		registry := getRegistryFromWorkflowContext(ctx)
		activityType, err := getValidatedActivityFunction(typeName, args, registry)
		if err != nil {
			settable.Set(nil, err)
			return future
		}
		activity, ok := registry.GetActivity(activityType.Name)
		if ok {
			activityFn = activity.GetFunction()
		} else if IsReplayNamespace(GetWorkflowInfo(ctx).Namespace) {
			// When running the replayer (but not necessarily during all replays), we
			// don't require the activities to be registered, so use a dummy function
			activityFn = func(context.Context) error { panic("dummy replayer function") }
		} else {
			settable.Set(nil, fmt.Errorf("local activity %s is not registered by the worker", activityType.Name))
			return future
		}
	} else {
		if err := validateFunctionArgs(localCtx.fn, args, false); err != nil {
			settable.Set(nil, err)
			return future
		}

		activityFn = localCtx.fn
	}

	options, err := getValidatedLocalActivityOptions(ctx)
	if err != nil {
		settable.Set(nil, err)
		return future
	}

	params := &ExecuteLocalActivityParams{
		ExecuteLocalActivityOptions: *options,
		ActivityFn:                  activityFn,
		ActivityType:                typeName,
		InputArgs:                   args,
		WorkflowInfo:                GetWorkflowInfo(ctx),
		DataConverter:               getDataConverterFromWorkflowContext(ctx),
		ScheduledTime:               Now(ctx), // initial scheduled time
		Header:                      header,
		Attempt:                     1, // Attempts always start at one
	}

	Go(ctx, func(ctx Context) {
		for {
			f := wc.scheduleLocalActivity(ctx, params)
			var result *commonpb.Payloads
			err := f.Get(ctx, &result)
			if retryErr, ok := err.(*needRetryError); ok && retryErr.Backoff > 0 {
				// Backoff for retry
				_ = Sleep(ctx, retryErr.Backoff)
				// increase the attempt, and retry the local activity
				params.Attempt = retryErr.Attempt + 1
				continue
			}

			// not more retry, return whatever is received.
			settable.Set(result, err)
			return
		}
	})

	return future
}

type needRetryError struct {
	Backoff time.Duration
	Attempt int32
}

func (e *needRetryError) Error() string {
	return fmt.Sprintf("Retry backoff: %v, Attempt: %v", e.Backoff, e.Attempt)
}

func (wc *workflowEnvironmentInterceptor) scheduleLocalActivity(ctx Context, params *ExecuteLocalActivityParams) Future {
	f := &futureImpl{channel: NewChannel(ctx).(*channelImpl)}
	ctxDone, cancellable := ctx.Done().(*channelImpl)
	cancellationCallback := &receiveCallback{}
	la := wc.env.ExecuteLocalActivity(*params, func(lar *LocalActivityResultWrapper) {
		if cancellable {
			// future is done, we don't need cancellation anymore
			ctxDone.removeReceiveCallback(cancellationCallback)
		}

		if lar.Err == nil || IsCanceledError(lar.Err) || lar.Backoff <= 0 {
			f.Set(lar.Result, lar.Err)
			return
		}

		// set retry error, and it will be handled by workflow.ExecuteLocalActivity().
		f.Set(nil, &needRetryError{Backoff: lar.Backoff, Attempt: lar.Attempt})
	})

	if cancellable {
		cancellationCallback.fn = func(v interface{}, more bool) bool {
			if ctx.Err() == ErrCanceled {
				getWorkflowEnvironment(ctx).RequestCancelLocalActivity(la)
			}
			return false
		}
		_, ok, more := ctxDone.receiveAsyncImpl(cancellationCallback)
		if ok || !more {
			cancellationCallback.fn(nil, more)
		}
	}

	return f
}

// ExecuteChildWorkflow requests child workflow execution in the context of a workflow.
// Context can be used to pass the settings for the child workflow.
// For example: task queue that this child workflow should be routed, timeouts that need to be configured.
// Use ChildWorkflowOptions to pass down the options.
//
//	 cwo := ChildWorkflowOptions{
//		    WorkflowExecutionTimeout: 10 * time.Minute,
//		    WorkflowTaskTimeout: time.Minute,
//		}
//	 ctx := WithChildWorkflowOptions(ctx, cwo)
//
// Input childWorkflow is either a workflow name or a workflow function that is getting scheduled.
// Input args are the arguments that need to be passed to the child workflow function represented by childWorkflow.
//
// If the child workflow failed to complete then the future get error would indicate the failure.
// The error will be of type *ChildWorkflowExecutionError. It will have important child workflow information and actual error that caused
// child workflow failure. Use errors.Unwrap to get this error or errors.As to check it type which can be one of
// *ApplicationError, *TimeoutError, or *CanceledError.
//
// You can cancel the pending child workflow using context(workflow.WithCancel(ctx)) and that will fail the workflow with
// *CanceledError set as cause for *ChildWorkflowExecutionError.
//
// ExecuteChildWorkflow returns ChildWorkflowFuture.
func ExecuteChildWorkflow(ctx Context, childWorkflow interface{}, args ...interface{}) ChildWorkflowFuture {
	i := getWorkflowOutboundInterceptor(ctx)
	env := getWorkflowEnvironment(ctx)
	workflowType, err := getWorkflowFunctionName(env.GetRegistry(), childWorkflow)
	if err != nil {
		panic(err)
	}
	// Put header on context before executing
	ctx = workflowContextWithNewHeader(ctx)
	return i.ExecuteChildWorkflow(ctx, workflowType, args...)
}

func (wc *workflowEnvironmentInterceptor) ExecuteChildWorkflow(ctx Context, childWorkflowType string, args ...interface{}) ChildWorkflowFuture {
	mainFuture, mainSettable := newDecodeFuture(ctx, childWorkflowType)
	executionFuture, executionSettable := NewFuture(ctx)
	result := &childWorkflowFutureImpl{
		decodeFutureImpl: mainFuture.(*decodeFutureImpl),
		executionFuture:  executionFuture.(*futureImpl),
	}

	// Immediately return if the context has an error without spawning the child workflow
	if ctx.Err() != nil {
		executionSettable.Set(nil, ctx.Err())
		mainSettable.Set(nil, ctx.Err())
		return result
	}

	workflowOptionsFromCtx := getWorkflowEnvOptions(ctx)
	dc := WithWorkflowContext(ctx, workflowOptionsFromCtx.DataConverter)
	env := getWorkflowEnvironment(ctx)
	wfType, input, err := getValidatedWorkflowFunction(childWorkflowType, args, dc, env.GetRegistry())
	if err != nil {
		executionSettable.Set(nil, err)
		mainSettable.Set(nil, err)
		return result
	}

	options := getWorkflowEnvOptions(ctx)
	options.DataConverter = dc
	options.ContextPropagators = workflowOptionsFromCtx.ContextPropagators
	options.Memo = workflowOptionsFromCtx.Memo
	options.SearchAttributes = workflowOptionsFromCtx.SearchAttributes
	options.VersioningIntent = workflowOptionsFromCtx.VersioningIntent

	header, err := workflowHeaderPropagated(ctx, options.ContextPropagators)
	if err != nil {
		executionSettable.Set(nil, err)
		mainSettable.Set(nil, err)
		return result
	}

	params := ExecuteWorkflowParams{
		WorkflowOptions: *options,
		Input:           input,
		WorkflowType:    wfType,
		Header:          header,
		scheduledTime:   Now(ctx), /* this is needed for test framework, and is not send to server */
		attempt:         1,
	}

	ctxDone, cancellable := ctx.Done().(*channelImpl)
	cancellationCallback := &receiveCallback{}
	getWorkflowEnvironment(ctx).ExecuteChildWorkflow(params, func(r *commonpb.Payloads, e error) {
		mainSettable.Set(r, e)
		if cancellable {
			// future is done, we don't need cancellation anymore
			ctxDone.removeReceiveCallback(cancellationCallback)
		}
	}, func(r WorkflowExecution, e error) {
		if e == nil {
			// We must wait for Workflow initiation to finish before registering the cancellation handler.
			// Otherwise, we risk firing the cancel handler and then having the workflow "initiate" afterwards,
			// which would result in an uncanceled workflow.
			if cancellable {
				cancellationCallback.fn = func(v interface{}, _ bool) bool {
					if ctx.Err() == ErrCanceled && !mainFuture.IsReady() {
						// child workflow started, and ctx canceled
						getWorkflowEnvironment(ctx).RequestCancelChildWorkflow(options.Namespace, r.ID)
					}
					return false
				}
				_, ok, more := ctxDone.receiveAsyncImpl(cancellationCallback)
				if ok || !more {
					cancellationCallback.fn(nil, more)
				}
			}
		}

		executionSettable.Set(r, e)
	})

	return result
}

// WorkflowInfo information about currently executing workflow
type WorkflowInfo struct {
	WorkflowExecution WorkflowExecution
	// The original runID before resetting. Using it instead of current runID can make workflow decision deterministic after reset. See also FirstRunId
	OriginalRunID string
	// The very first original RunId of the current Workflow Execution preserved along the chain of ContinueAsNew, Retry, Cron and Reset. Identifies the whole Runs chain of Workflow Execution.
	FirstRunID               string
	WorkflowType             WorkflowType
	TaskQueueName            string
	WorkflowExecutionTimeout time.Duration
	WorkflowRunTimeout       time.Duration
	WorkflowTaskTimeout      time.Duration
	Namespace                string
	Attempt                  int32 // Attempt starts from 1 and increased by 1 for every retry if retry policy is specified.
	// Time of the workflow start.
	// workflow.Now at the beginning of a workflow can return a later time if the Workflow Worker was down.
	WorkflowStartTime       time.Time
	lastCompletionResult    *commonpb.Payloads
	lastFailure             *failurepb.Failure
	CronSchedule            string
	ContinuedExecutionRunID string
	ParentWorkflowNamespace string
	ParentWorkflowExecution *WorkflowExecution
	Memo                    *commonpb.Memo             // Value can be decoded using data converter (defaultDataConverter, or custom one if set).
	SearchAttributes        *commonpb.SearchAttributes // Value can be decoded using defaultDataConverter.
	RetryPolicy             *RetryPolicy
	// BinaryChecksum represents the value persisted by the last worker to complete a task in this workflow. It may be
	// an explicitly set or implicitly derived binary checksum of the worker binary, or, if this worker has opted into
	// build-id based versioning, is the explicitly set worker build id. If this is the first worker to operate on the
	// workflow, it is this worker's current value.
	BinaryChecksum string

	currentHistoryLength int
}

// GetBinaryChecksum return binary checksum.
func (wInfo *WorkflowInfo) GetBinaryChecksum() string {
	if wInfo.BinaryChecksum == "" {
		return getBinaryChecksum()
	}
	return wInfo.BinaryChecksum
}

// GetCurrentHistoryLength returns the current length of history when called.
// This value may change throughout the life of the workflow.
func (wInfo *WorkflowInfo) GetCurrentHistoryLength() int {
	return wInfo.currentHistoryLength
}

// GetWorkflowInfo extracts info of a current workflow from a context.
func GetWorkflowInfo(ctx Context) *WorkflowInfo {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.GetInfo(ctx)
}

func (wc *workflowEnvironmentInterceptor) GetInfo(ctx Context) *WorkflowInfo {
	return wc.env.WorkflowInfo()
}

// GetLogger returns a logger to be used in workflow's context
func GetLogger(ctx Context) log.Logger {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.GetLogger(ctx)
}

func (wc *workflowEnvironmentInterceptor) GetLogger(ctx Context) log.Logger {
	return wc.env.GetLogger()
}

// GetMetricsHandler returns a metrics handler to be used in workflow's context
func GetMetricsHandler(ctx Context) metrics.Handler {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.GetMetricsHandler(ctx)
}

func (wc *workflowEnvironmentInterceptor) GetMetricsHandler(ctx Context) metrics.Handler {
	return wc.env.GetMetricsHandler()
}

// Now returns the current time in UTC. It corresponds to the time when the workflow task is started or replayed.
// Workflow needs to use this method to get the wall clock time instead of the one from the golang library.
func Now(ctx Context) time.Time {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.Now(ctx).UTC()
}

func (wc *workflowEnvironmentInterceptor) Now(ctx Context) time.Time {
	return wc.env.Now()
}

// NewTimer returns immediately and the future becomes ready after the specified duration d. The workflow needs to use
// this NewTimer() to get the timer instead of the Go lang library one(timer.NewTimer()). You can cancel the pending
// timer by cancel the Context (using context from workflow.WithCancel(ctx)) and that will cancel the timer. After timer
// is canceled, the returned Future become ready, and Future.Get() will return *CanceledError.
func NewTimer(ctx Context, d time.Duration) Future {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.NewTimer(ctx, d)
}

func (wc *workflowEnvironmentInterceptor) NewTimer(ctx Context, d time.Duration) Future {
	future, settable := NewFuture(ctx)
	if d <= 0 {
		settable.Set(true, nil)
		return future
	}

	ctxDone, cancellable := ctx.Done().(*channelImpl)
	cancellationCallback := &receiveCallback{}
	timerID := wc.env.NewTimer(d, func(r *commonpb.Payloads, e error) {
		settable.Set(nil, e)
		if cancellable {
			// future is done, we don't need cancellation anymore
			ctxDone.removeReceiveCallback(cancellationCallback)
		}
	})

	if timerID != nil && cancellable {
		cancellationCallback.fn = func(v interface{}, more bool) bool {
			if !future.IsReady() {
				wc.env.RequestCancelTimer(*timerID)
			}
			return false
		}
		_, ok, more := ctxDone.receiveAsyncImpl(cancellationCallback)
		if ok || !more {
			cancellationCallback.fn(nil, more)
		}
	}
	return future
}

// Sleep pauses the current workflow for at least the duration d. A negative or zero duration causes Sleep to return
// immediately. Workflow code needs to use this Sleep() to sleep instead of the Go lang library one(timer.Sleep()).
// You can cancel the pending sleep by cancel the Context (using context from workflow.WithCancel(ctx)).
// Sleep() returns nil if the duration d is passed, or it returns *CanceledError if the ctx is canceled. There are 2
// reasons the ctx could be canceled: 1) your workflow code cancel the ctx (with workflow.WithCancel(ctx));
// 2) your workflow itself is canceled by external request.
func Sleep(ctx Context, d time.Duration) (err error) {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.Sleep(ctx, d)
}

func (wc *workflowEnvironmentInterceptor) Sleep(ctx Context, d time.Duration) (err error) {
	t := NewTimer(ctx, d)
	err = t.Get(ctx, nil)
	return
}

// RequestCancelExternalWorkflow can be used to request cancellation of an external workflow.
// Input workflowID is the workflow ID of target workflow.
// Input runID indicates the instance of a workflow. Input runID is optional (default is ""). When runID is not specified,
// then the currently running instance of that workflowID will be used.
// By default, the current workflow's namespace will be used as target namespace. However, you can specify a different namespace
// of the target workflow using the context like:
//
//	ctx := WithWorkflowNamespace(ctx, "namespace")
//
// RequestCancelExternalWorkflow return Future with failure or empty success result.
func RequestCancelExternalWorkflow(ctx Context, workflowID, runID string) Future {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.RequestCancelExternalWorkflow(ctx, workflowID, runID)
}

func (wc *workflowEnvironmentInterceptor) RequestCancelExternalWorkflow(ctx Context, workflowID, runID string) Future {
	ctx1 := setWorkflowEnvOptionsIfNotExist(ctx)
	options := getWorkflowEnvOptions(ctx1)
	future, settable := NewFuture(ctx1)

	if workflowID == "" {
		settable.Set(nil, errWorkflowIDNotSet)
		return future
	}

	resultCallback := func(result *commonpb.Payloads, err error) {
		settable.Set(result, err)
	}

	wc.env.RequestCancelExternalWorkflow(
		options.Namespace,
		workflowID,
		runID,
		resultCallback,
	)

	return future
}

// SignalExternalWorkflow can be used to send signal info to an external workflow.
// Input workflowID is the workflow ID of target workflow.
// Input runID indicates the instance of a workflow. Input runID is optional (default is ""). When runID is not specified,
// then the currently running instance of that workflowID will be used.
// By default, the current workflow's namespace will be used as target namespace. However, you can specify a different namespace
// of the target workflow using the context like:
//
//	ctx := WithWorkflowNamespace(ctx, "namespace")
//
// SignalExternalWorkflow return Future with failure or empty success result.
func SignalExternalWorkflow(ctx Context, workflowID, runID, signalName string, arg interface{}) Future {
	i := getWorkflowOutboundInterceptor(ctx)
	// Put header on context before executing
	ctx = workflowContextWithNewHeader(ctx)
	return i.SignalExternalWorkflow(ctx, workflowID, runID, signalName, arg)
}

func (wc *workflowEnvironmentInterceptor) SignalExternalWorkflow(ctx Context, workflowID, runID, signalName string, arg interface{}) Future {
	const childWorkflowOnly = false // this means we are not limited to child workflow
	return signalExternalWorkflow(ctx, workflowID, runID, signalName, arg, childWorkflowOnly)
}

func (wc *workflowEnvironmentInterceptor) SignalChildWorkflow(ctx Context, workflowID, signalName string, arg interface{}) Future {
	const childWorkflowOnly = true // this means we are limited to child workflow
	// Empty run ID to indicate current one
	return signalExternalWorkflow(ctx, workflowID, "", signalName, arg, childWorkflowOnly)
}

func signalExternalWorkflow(ctx Context, workflowID, runID, signalName string, arg interface{}, childWorkflowOnly bool) Future {
	env := getWorkflowEnvironment(ctx)
	ctx1 := setWorkflowEnvOptionsIfNotExist(ctx)
	options := getWorkflowEnvOptions(ctx1)
	future, settable := NewFuture(ctx1)

	if workflowID == "" {
		settable.Set(nil, errWorkflowIDNotSet)
		return future
	}

	dataConverter := getDataConverterFromWorkflowContext(ctx)
	input, err := encodeArg(dataConverter, arg)
	if err != nil {
		settable.Set(nil, err)
		return future
	}

	// Get header
	header, err := workflowHeaderPropagated(ctx, options.ContextPropagators)
	if err != nil {
		settable.Set(nil, err)
		return future
	}

	resultCallback := func(result *commonpb.Payloads, err error) {
		settable.Set(result, err)
	}
	env.SignalExternalWorkflow(
		options.Namespace,
		workflowID,
		runID,
		signalName,
		input,
		arg,
		header,
		childWorkflowOnly,
		resultCallback,
	)

	return future
}

// UpsertSearchAttributes is used to add or update workflow search attributes.
// The search attributes can be used in query of List/Scan/Count workflow APIs.
// The key and value type must be registered on temporal server side;
// The value has to deterministic when replay;
// The value has to be Json serializable.
// UpsertSearchAttributes will merge attributes to existing map in workflow, for example workflow code:
//
//	  func MyWorkflow(ctx workflow.Context, input string) error {
//		   attr1 := map[string]interface{}{
//			   "CustomIntField": 1,
//			   "CustomBoolField": true,
//		   }
//		   workflow.UpsertSearchAttributes(ctx, attr1)
//
//		   attr2 := map[string]interface{}{
//			   "CustomIntField": 2,
//			   "CustomKeywordField": "seattle",
//		   }
//		   workflow.UpsertSearchAttributes(ctx, attr2)
//	  }
//
// will eventually have search attributes:
//
//	map[string]interface{}{
//		"CustomIntField": 2,
//		"CustomBoolField": true,
//		"CustomKeywordField": "seattle",
//	}
//
// This is only supported when using ElasticSearch.
func UpsertSearchAttributes(ctx Context, attributes map[string]interface{}) error {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.UpsertSearchAttributes(ctx, attributes)
}

func (wc *workflowEnvironmentInterceptor) UpsertSearchAttributes(ctx Context, attributes map[string]interface{}) error {
	if _, ok := attributes[TemporalChangeVersion]; ok {
		return errors.New("TemporalChangeVersion is a reserved key that cannot be set, please use other key")
	}
	return wc.env.UpsertSearchAttributes(attributes)
}

// UpsertMemo is used to add or update workflow memo.
// UpsertMemo will merge keys to the existing map in workflow. For example:
//
//	func MyWorkflow(ctx workflow.Context, input string) error {
//		memo1 := map[string]interface{}{
//			"Key1": 1,
//			"Key2": true,
//		}
//		workflow.UpsertMemo(ctx, memo1)
//
//		memo2 := map[string]interface{}{
//			"Key1": 2,
//			"Key3": "seattle",
//		}
//		workflow.UpsertMemo(ctx, memo2)
//	}
//
// The workflow memo will eventually be:
//
//	map[string]interface{}{
//		"Key1": 2,
//		"Key2": true,
//		"Key3": "seattle",
//	}
//
// This is only supported with Temporal Server 1.18+
func UpsertMemo(ctx Context, memo map[string]interface{}) error {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.UpsertMemo(ctx, memo)
}

func (wc *workflowEnvironmentInterceptor) UpsertMemo(ctx Context, memo map[string]interface{}) error {
	return wc.env.UpsertMemo(memo)
}

// WithChildWorkflowOptions adds all workflow options to the context.
// The current timeout resolution implementation is in seconds and uses math.Ceil(d.Seconds()) as the duration. But is
// subjected to change in the future.
func WithChildWorkflowOptions(ctx Context, cwo ChildWorkflowOptions) Context {
	ctx1 := setWorkflowEnvOptionsIfNotExist(ctx)
	wfOptions := getWorkflowEnvOptions(ctx1)
	if len(cwo.Namespace) > 0 {
		wfOptions.Namespace = cwo.Namespace
	}
	if len(cwo.TaskQueue) > 0 {
		wfOptions.TaskQueueName = cwo.TaskQueue
	}
	wfOptions.WorkflowID = cwo.WorkflowID
	wfOptions.WorkflowExecutionTimeout = cwo.WorkflowExecutionTimeout
	wfOptions.WorkflowRunTimeout = cwo.WorkflowRunTimeout
	wfOptions.WorkflowTaskTimeout = cwo.WorkflowTaskTimeout
	wfOptions.WaitForCancellation = cwo.WaitForCancellation
	wfOptions.WorkflowIDReusePolicy = cwo.WorkflowIDReusePolicy
	wfOptions.RetryPolicy = convertToPBRetryPolicy(cwo.RetryPolicy)
	wfOptions.CronSchedule = cwo.CronSchedule
	wfOptions.Memo = cwo.Memo
	wfOptions.SearchAttributes = cwo.SearchAttributes
	wfOptions.ParentClosePolicy = cwo.ParentClosePolicy
	wfOptions.VersioningIntent = cwo.VersioningIntent

	return ctx1
}

// GetChildWorkflowOptions returns all workflow options present on the context.
func GetChildWorkflowOptions(ctx Context) ChildWorkflowOptions {
	opts := getWorkflowEnvOptions(ctx)
	if opts == nil {
		return ChildWorkflowOptions{}
	}
	return ChildWorkflowOptions{
		Namespace:                opts.Namespace,
		WorkflowID:               opts.WorkflowID,
		TaskQueue:                opts.TaskQueueName,
		WorkflowExecutionTimeout: opts.WorkflowExecutionTimeout,
		WorkflowRunTimeout:       opts.WorkflowRunTimeout,
		WorkflowTaskTimeout:      opts.WorkflowTaskTimeout,
		WaitForCancellation:      opts.WaitForCancellation,
		WorkflowIDReusePolicy:    opts.WorkflowIDReusePolicy,
		RetryPolicy:              convertFromPBRetryPolicy(opts.RetryPolicy),
		CronSchedule:             opts.CronSchedule,
		Memo:                     opts.Memo,
		SearchAttributes:         opts.SearchAttributes,
		ParentClosePolicy:        opts.ParentClosePolicy,
		VersioningIntent:         opts.VersioningIntent,
	}
}

// WithWorkflowNamespace adds a namespace to the context.
func WithWorkflowNamespace(ctx Context, name string) Context {
	ctx1 := setWorkflowEnvOptionsIfNotExist(ctx)
	getWorkflowEnvOptions(ctx1).Namespace = name
	return ctx1
}

// WithWorkflowTaskQueue adds a task queue to the context.
func WithWorkflowTaskQueue(ctx Context, name string) Context {
	if name == "" {
		panic("empty task queue name")
	}
	ctx1 := setWorkflowEnvOptionsIfNotExist(ctx)
	getWorkflowEnvOptions(ctx1).TaskQueueName = name
	return ctx1
}

// WithWorkflowID adds a workflowID to the context.
func WithWorkflowID(ctx Context, workflowID string) Context {
	ctx1 := setWorkflowEnvOptionsIfNotExist(ctx)
	getWorkflowEnvOptions(ctx1).WorkflowID = workflowID
	return ctx1
}

// WithWorkflowRunTimeout adds a run timeout to the context.
// The current timeout resolution implementation is in seconds and uses math.Ceil(d.Seconds()) as the duration. But is
// subjected to change in the future.
func WithWorkflowRunTimeout(ctx Context, d time.Duration) Context {
	ctx1 := setWorkflowEnvOptionsIfNotExist(ctx)
	getWorkflowEnvOptions(ctx1).WorkflowRunTimeout = d
	return ctx1
}

// WithWorkflowTaskTimeout adds a workflow task timeout to the context.
// The current timeout resolution implementation is in seconds and uses math.Ceil(d.Seconds()) as the duration. But is
// subjected to change in the future.
func WithWorkflowTaskTimeout(ctx Context, d time.Duration) Context {
	ctx1 := setWorkflowEnvOptionsIfNotExist(ctx)
	getWorkflowEnvOptions(ctx1).WorkflowTaskTimeout = d
	return ctx1
}

// WithDataConverter adds DataConverter to the context.
func WithDataConverter(ctx Context, dc converter.DataConverter) Context {
	if dc == nil {
		panic("data converter is nil for WithDataConverter")
	}
	ctx1 := setWorkflowEnvOptionsIfNotExist(ctx)
	getWorkflowEnvOptions(ctx1).DataConverter = dc
	return ctx1
}

// WithWorkflowVersioningIntent is used to set the VersioningIntent before constructing a
// ContinueAsNewError with NewContinueAsNewError.
// WARNING: Worker versioning is currently experimental
func WithWorkflowVersioningIntent(ctx Context, intent VersioningIntent) Context {
	ctx1 := setWorkflowEnvOptionsIfNotExist(ctx)
	getWorkflowEnvOptions(ctx1).VersioningIntent = intent
	return ctx1
}

// withContextPropagators adds ContextPropagators to the context.
func withContextPropagators(ctx Context, contextPropagators []ContextPropagator) Context {
	ctx1 := setWorkflowEnvOptionsIfNotExist(ctx)
	getWorkflowEnvOptions(ctx1).ContextPropagators = contextPropagators
	return ctx1
}

// GetSignalChannel returns channel corresponding to the signal name.
func GetSignalChannel(ctx Context, signalName string) ReceiveChannel {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.GetSignalChannel(ctx, signalName)
}

func (wc *workflowEnvironmentInterceptor) GetSignalChannel(ctx Context, signalName string) ReceiveChannel {
	return getWorkflowEnvOptions(ctx).getSignalChannel(ctx, signalName)
}

func newEncodedValue(value *commonpb.Payloads, dc converter.DataConverter) converter.EncodedValue {
	if dc == nil {
		dc = converter.GetDefaultDataConverter()
	}
	return &EncodedValue{value, dc}
}

// Get extract data from encoded data to desired value type. valuePtr is pointer to the actual value type.
func (b EncodedValue) Get(valuePtr interface{}) error {
	if !b.HasValue() {
		return ErrNoData
	}
	return decodeArg(b.dataConverter, b.value, valuePtr)
}

// HasValue return whether there is value
func (b EncodedValue) HasValue() bool {
	return b.value != nil
}

// SideEffect executes the provided function once, records its result into the workflow history. The recorded result on
// history will be returned without executing the provided function during replay. This guarantees the deterministic
// requirement for workflow as the exact same result will be returned in replay.
// Common use case is to run some short non-deterministic code in workflow, like getting random number or new UUID.
// The only way to fail SideEffect is to panic which causes workflow task failure. The workflow task after timeout is
// rescheduled and re-executed giving SideEffect another chance to succeed.
//
// Caution: do not use SideEffect to modify closures. Always retrieve result from SideEffect's encoded return value.
// For example this code is BROKEN:
//
//	// Bad example:
//	var random int
//	workflow.SideEffect(func(ctx workflow.Context) interface{} {
//	       random = rand.Intn(100)
//	       return nil
//	})
//	// random will always be 0 in replay, thus this code is non-deterministic
//	if random < 50 {
//	       ....
//	} else {
//	       ....
//	}
//
// On replay the provided function is not executed, the random will always be 0, and the workflow could takes a
// different path breaking the determinism.
//
// Here is the correct way to use SideEffect:
//
//	// Good example:
//	encodedRandom := SideEffect(func(ctx workflow.Context) interface{} {
//	      return rand.Intn(100)
//	})
//	var random int
//	encodedRandom.Get(&random)
//	if random < 50 {
//	       ....
//	} else {
//	       ....
//	}
func SideEffect(ctx Context, f func(ctx Context) interface{}) converter.EncodedValue {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.SideEffect(ctx, f)
}

func (wc *workflowEnvironmentInterceptor) SideEffect(ctx Context, f func(ctx Context) interface{}) converter.EncodedValue {
	dc := getDataConverterFromWorkflowContext(ctx)
	future, settable := NewFuture(ctx)
	wrapperFunc := func() (*commonpb.Payloads, error) {
		r := f(ctx)
		return encodeArg(dc, r)
	}
	resultCallback := func(result *commonpb.Payloads, err error) {
		settable.Set(EncodedValue{result, dc}, err)
	}
	wc.env.SideEffect(wrapperFunc, resultCallback)
	var encoded EncodedValue
	if err := future.Get(ctx, &encoded); err != nil {
		panic(err)
	}
	return encoded
}

// MutableSideEffect executes the provided function once, then it looks up the history for the value with the given id.
// If there is no existing value, then it records the function result as a value with the given id on history;
// otherwise, it compares whether the existing value from history has changed from the new function result by calling
// theprovided equals function. If they are equal, it returns the value without recording a new one in history;
//
//	otherwise, it records the new value with the same id on history.
//
// Caution: do not use MutableSideEffect to modify closures. Always retrieve result from MutableSideEffect's encoded
// return value.
//
// The difference between MutableSideEffect() and SideEffect() is that every new SideEffect() call in non-replay will
// result in a new marker being recorded on history. However, MutableSideEffect() only records a new marker if the value
// changed. During replay, MutableSideEffect() will not execute the function again, but it will return the exact same
// value as it was returning during the non-replay run.
//
// One good use case of MutableSideEffect() is to access dynamically changing config without breaking determinism.
func MutableSideEffect(ctx Context, id string, f func(ctx Context) interface{}, equals func(a, b interface{}) bool) converter.EncodedValue {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.MutableSideEffect(ctx, id, f, equals)
}

func (wc *workflowEnvironmentInterceptor) MutableSideEffect(ctx Context, id string, f func(ctx Context) interface{}, equals func(a, b interface{}) bool) converter.EncodedValue {
	wrapperFunc := func() interface{} {
		return f(ctx)
	}
	return wc.env.MutableSideEffect(id, wrapperFunc, equals)
}

// DefaultVersion is a version returned by GetVersion for code that wasn't versioned before
const DefaultVersion Version = -1

// TemporalChangeVersion is used as search attributes key to find workflows with specific change version.
const TemporalChangeVersion = "TemporalChangeVersion"

// GetVersion is used to safely perform backwards incompatible changes to workflow definitions.
// It is not allowed to update workflow code while there are workflows running as it is going to break
// determinism. The solution is to have both old code that is used to replay existing workflows
// as well as the new one that is used when it is executed for the first time.
// GetVersion returns maxSupported version when is executed for the first time. This version is recorded into the
// workflow history as a marker event. Even if maxSupported version is changed the version that was recorded is
// returned on replay. DefaultVersion constant contains version of code that wasn't versioned before.
// For example initially workflow has the following code:
//
//	err = workflow.ExecuteActivity(ctx, foo).Get(ctx, nil)
//
// it should be updated to
//
//	err = workflow.ExecuteActivity(ctx, bar).Get(ctx, nil)
//
// The backwards compatible way to execute the update is
//
//	v :=  GetVersion(ctx, "fooChange", DefaultVersion, 1)
//	if v  == DefaultVersion {
//	    err = workflow.ExecuteActivity(ctx, foo).Get(ctx, nil)
//	} else {
//	    err = workflow.ExecuteActivity(ctx, bar).Get(ctx, nil)
//	}
//
// Then bar has to be changed to baz:
//
//	v :=  GetVersion(ctx, "fooChange", DefaultVersion, 2)
//	if v  == DefaultVersion {
//	    err = workflow.ExecuteActivity(ctx, foo).Get(ctx, nil)
//	} else if v == 1 {
//	    err = workflow.ExecuteActivity(ctx, bar).Get(ctx, nil)
//	} else {
//	    err = workflow.ExecuteActivity(ctx, baz).Get(ctx, nil)
//	}
//
// Later when there are no workflow executions running DefaultVersion the correspondent branch can be removed:
//
//	v :=  GetVersion(ctx, "fooChange", 1, 2)
//	if v == 1 {
//	    err = workflow.ExecuteActivity(ctx, bar).Get(ctx, nil)
//	} else {
//	    err = workflow.ExecuteActivity(ctx, baz).Get(ctx, nil)
//	}
//
// It is recommended to keep the GetVersion() call even if single branch is left:
//
//	GetVersion(ctx, "fooChange", 2, 2)
//	err = workflow.ExecuteActivity(ctx, baz).Get(ctx, nil)
//
// The reason to keep it is: 1) it ensures that if there is older version execution still running, it will fail here
// and not proceed; 2) if you ever need to make more changes for “fooChange”, for example change activity from baz to qux,
// you just need to update the maxVersion from 2 to 3.
//
// Note that, you only need to preserve the first call to GetVersion() for each changeID. All subsequent call to GetVersion()
// with same changeID are safe to remove. However, if you really want to get rid of the first GetVersion() call as well,
// you can do so, but you need to make sure: 1) all older version executions are completed; 2) you can no longer use “fooChange”
// as changeID. If you ever need to make changes to that same part like change from baz to qux, you would need to use a
// different changeID like “fooChange-fix2”, and start minVersion from DefaultVersion again. The code would looks like:
//
//	v := workflow.GetVersion(ctx, "fooChange-fix2", workflow.DefaultVersion, 1)
//	if v == workflow.DefaultVersion {
//	  err = workflow.ExecuteActivity(ctx, baz, data).Get(ctx, nil)
//	} else {
//	  err = workflow.ExecuteActivity(ctx, qux, data).Get(ctx, nil)
//	}
func GetVersion(ctx Context, changeID string, minSupported, maxSupported Version) Version {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.GetVersion(ctx, changeID, minSupported, maxSupported)
}

func (wc *workflowEnvironmentInterceptor) GetVersion(ctx Context, changeID string, minSupported, maxSupported Version) Version {
	return wc.env.GetVersion(changeID, minSupported, maxSupported)
}

// SetQueryHandler sets the query handler to handle workflow query. The queryType specify which query type this handler
// should handle. The handler must be a function that returns 2 values. The first return value must be a serializable
// result. The second return value must be an error. The handler function could receive any number of input parameters.
// All the input parameter must be serializable. You should call workflow.SetQueryHandler() at the beginning of the workflow
// code. When client calls Client.QueryWorkflow() to temporal server, a task will be generated on server that will be dispatched
// to a workflow worker, which will replay the history events and then execute a query handler based on the query type.
// The query handler will be invoked out of the context of the workflow, meaning that the handler code must not use temporal
// context to do things like workflow.NewChannel(), workflow.Go() or to call any workflow blocking functions like
// Channel.Get() or Future.Get(). Trying to do so in query handler code will fail the query and client will receive
// QueryFailedError.
// Example of workflow code that support query type "current_state":
//
//	func MyWorkflow(ctx workflow.Context, input string) error {
//	  currentState := "started" // this could be any serializable struct
//	  err := workflow.SetQueryHandler(ctx, "current_state", func() (string, error) {
//	    return currentState, nil
//	  })
//	  if err != nil {
//	    currentState = "failed to register query handler"
//	    return err
//	  }
//	  // your normal workflow code begins here, and you update the currentState as the code makes progress.
//	  currentState = "waiting timer"
//	  err = NewTimer(ctx, time.Hour).Get(ctx, nil)
//	  if err != nil {
//	    currentState = "timer failed"
//	    return err
//	  }
//
//	  currentState = "waiting activity"
//	  ctx = WithActivityOptions(ctx, myActivityOptions)
//	  err = ExecuteActivity(ctx, MyActivity, "my_input").Get(ctx, nil)
//	  if err != nil {
//	    currentState = "activity failed"
//	    return err
//	  }
//	  currentState = "done"
//	  return nil
//	}
func SetQueryHandler(ctx Context, queryType string, handler interface{}) error {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.SetQueryHandler(ctx, queryType, handler)
}

// SetUpdateHandler binds an update handler function to the specified
// name such that update invocations specifying that name will invoke the
// handler.  The handler function can take as input any number of parameters so
// long as they can be serialized/deserialized by the system. The handler can
// take a workflow.Context as its first parameter but this is not required. The
// update handler must return either a single error or a single serializable
// object along with a single error. The update handler function is invoked in
// the context of the workflow and thus is subject to the same restrictions as
// workflow code, namely, the update handler must be deterministic. As with
// other workflow code, update code is free to invoke and wait on the results of
// activities. Update handler code is free to mutate workflow state.
//
// This registration can optionally specify (through UpdateHandlerOptions) an
// update validation function. If provided, this function will be invoked before
// the update handler itself is invoked and if this function returns an error,
// the update request will be considered to have been rejected and as such will
// not occupy any space in the workflow history. Validation functions must take
// as inputs the same parameters as the associated update handler but my vary
// from said handler by the presence/absence of a workflow.Context as the first
// parameter. Validation handlers must only return a single error. Validation
// handlers must be deterministic and can observe workflow state but must not
// mutate workflow state in any way.
//
// NOTE: Experimental
func SetUpdateHandler(ctx Context, updateName string, handler interface{}, opts UpdateHandlerOptions) error {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.SetUpdateHandler(ctx, updateName, handler, opts)
}

func (wc *workflowEnvironmentInterceptor) SetQueryHandler(ctx Context, queryType string, handler interface{}) error {
	if strings.HasPrefix(queryType, "__") {
		return errors.New("queryType starts with '__' is reserved for internal use")
	}
	return setQueryHandler(ctx, queryType, handler)
}

func (wc *workflowEnvironmentInterceptor) SetUpdateHandler(ctx Context, name string, handler interface{}, opts UpdateHandlerOptions) error {
	if strings.HasPrefix(name, "__") {
		return errors.New("update names starting with '__' are reserved for internal use")
	}
	return setUpdateHandler(ctx, name, handler, opts)
}

// IsReplaying returns whether the current workflow code is replaying.
//
// Warning! Never make commands, like schedule activity/childWorkflow/timer or send/wait on future/channel, based on
// this flag as it is going to break workflow determinism requirement.
// The only reasonable use case for this flag is to avoid some external actions during replay, like custom logging or
// metric reporting. Please note that Temporal already provide standard logging/metric via workflow.GetLogger(ctx) and
// workflow.GetMetricsHandler(ctx), and those standard mechanism are replay-aware and it will automatically suppress
// during replay. Only use this flag if you need custom logging/metrics reporting, for example if you want to log to
// kafka.
//
// Warning! Any action protected by this flag should not fail or if it does fail should ignore that failure or panic
// on the failure. If workflow don't want to be blocked on those failure, it should ignore those failure; if workflow do
// want to make sure it proceed only when that action succeed then it should panic on that failure. Panic raised from a
// workflow causes workflow task to fail and temporal server will rescheduled later to retry.
func IsReplaying(ctx Context) bool {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.IsReplaying(ctx)
}

func (wc *workflowEnvironmentInterceptor) IsReplaying(ctx Context) bool {
	return wc.env.IsReplaying()
}

// HasLastCompletionResult checks if there is completion result from previous runs.
// This is used in combination with cron schedule. A workflow can be started with an optional cron schedule.
// If a cron workflow wants to pass some data to next schedule, it can return any data and that data will become
// available when next run starts.
// This HasLastCompletionResult() checks if there is such data available passing down from previous successful run.
func HasLastCompletionResult(ctx Context) bool {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.HasLastCompletionResult(ctx)
}

func (wc *workflowEnvironmentInterceptor) HasLastCompletionResult(ctx Context) bool {
	info := wc.GetInfo(ctx)
	return info.lastCompletionResult != nil
}

// GetLastCompletionResult extract last completion result from previous run for this cron workflow.
// This is used in combination with cron schedule. A workflow can be started with an optional cron schedule.
// If a cron workflow wants to pass some data to next schedule, it can return any data and that data will become
// available when next run starts.
// This GetLastCompletionResult() extract the data into expected data structure.
//
// Note, values should not be reused for extraction here because merging on top
// of existing values may result in unexpected behavior similar to
// json.Unmarshal.
func GetLastCompletionResult(ctx Context, d ...interface{}) error {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.GetLastCompletionResult(ctx, d...)
}

func (wc *workflowEnvironmentInterceptor) GetLastCompletionResult(ctx Context, d ...interface{}) error {
	info := wc.GetInfo(ctx)
	if info.lastCompletionResult == nil {
		return ErrNoData
	}

	encodedVal := newEncodedValues(info.lastCompletionResult, getDataConverterFromWorkflowContext(ctx))
	return encodedVal.Get(d...)
}

// GetLastError extracts the latest failure from any from previous run for this workflow, if one has failed. If none
// have failed, nil is returned.
//
// See TestWorkflowEnvironment.SetLastError() for unit test support.
func GetLastError(ctx Context) error {
	i := getWorkflowOutboundInterceptor(ctx)
	return i.GetLastError(ctx)
}

func (wc *workflowEnvironmentInterceptor) GetLastError(ctx Context) error {
	info := wc.GetInfo(ctx)
	return wc.env.GetFailureConverter().FailureToError(info.lastFailure)
}

// Needed so this can properly be considered an inbound interceptor
func (*workflowEnvironmentInterceptor) mustEmbedWorkflowInboundInterceptorBase() {}

// Needed so this can properly be considered an outbound interceptor
func (*workflowEnvironmentInterceptor) mustEmbedWorkflowOutboundInterceptorBase() {}

// WithActivityOptions adds all options to the copy of the context.
// The current timeout resolution implementation is in seconds and uses math.Ceil(d.Seconds()) as the duration. But is
// subjected to change in the future.
func WithActivityOptions(ctx Context, options ActivityOptions) Context {
	ctx1 := setActivityParametersIfNotExist(ctx)
	eap := getActivityOptions(ctx1)

	if len(options.TaskQueue) > 0 {
		eap.TaskQueueName = options.TaskQueue
	}
	eap.ScheduleToCloseTimeout = options.ScheduleToCloseTimeout
	eap.StartToCloseTimeout = options.StartToCloseTimeout
	eap.ScheduleToStartTimeout = options.ScheduleToStartTimeout
	eap.HeartbeatTimeout = options.HeartbeatTimeout
	eap.WaitForCancellation = options.WaitForCancellation
	eap.ActivityID = options.ActivityID
	eap.RetryPolicy = convertToPBRetryPolicy(options.RetryPolicy)
	eap.DisableEagerExecution = options.DisableEagerExecution
	eap.VersioningIntent = options.VersioningIntent
	return ctx1
}

// WithLocalActivityOptions adds local activity options to the copy of the context.
// The current timeout resolution implementation is in seconds and uses math.Ceil(d.Seconds()) as the duration. But is
// subjected to change in the future.
func WithLocalActivityOptions(ctx Context, options LocalActivityOptions) Context {
	ctx1 := setLocalActivityParametersIfNotExist(ctx)
	opts := getLocalActivityOptions(ctx1)

	opts.ScheduleToCloseTimeout = options.ScheduleToCloseTimeout
	opts.StartToCloseTimeout = options.StartToCloseTimeout
	opts.RetryPolicy = applyRetryPolicyDefaultsForLocalActivity(options.RetryPolicy)
	return ctx1
}

func applyRetryPolicyDefaultsForLocalActivity(policy *RetryPolicy) *RetryPolicy {
	if policy == nil {
		policy = &RetryPolicy{}
	}
	if policy.BackoffCoefficient == 0 {
		policy.BackoffCoefficient = 2
	}
	if policy.InitialInterval == 0 {
		policy.InitialInterval = 1 * time.Second
	}
	if policy.MaximumInterval == 0 {
		policy.MaximumInterval = policy.InitialInterval * 100
	}
	return policy
}

// WithTaskQueue adds a task queue to the copy of the context.
func WithTaskQueue(ctx Context, name string) Context {
	ctx1 := setActivityParametersIfNotExist(ctx)
	getActivityOptions(ctx1).TaskQueueName = name
	return ctx1
}

// GetActivityOptions returns all activity options present on the context.
func GetActivityOptions(ctx Context) ActivityOptions {
	opts := getActivityOptions(ctx)
	if opts == nil {
		return ActivityOptions{}
	}
	return ActivityOptions{
		TaskQueue:              opts.TaskQueueName,
		ScheduleToCloseTimeout: opts.ScheduleToCloseTimeout,
		ScheduleToStartTimeout: opts.ScheduleToStartTimeout,
		StartToCloseTimeout:    opts.StartToCloseTimeout,
		HeartbeatTimeout:       opts.HeartbeatTimeout,
		WaitForCancellation:    opts.WaitForCancellation,
		ActivityID:             opts.ActivityID,
		RetryPolicy:            convertFromPBRetryPolicy(opts.RetryPolicy),
		DisableEagerExecution:  opts.DisableEagerExecution,
		VersioningIntent:       opts.VersioningIntent,
	}
}

// GetLocalActivityOptions returns all local activity options present on the context.
func GetLocalActivityOptions(ctx Context) LocalActivityOptions {
	opts := getLocalActivityOptions(ctx)
	if opts == nil {
		return LocalActivityOptions{}
	}
	return LocalActivityOptions{
		ScheduleToCloseTimeout: opts.ScheduleToCloseTimeout,
		StartToCloseTimeout:    opts.StartToCloseTimeout,
		RetryPolicy:            opts.RetryPolicy,
	}
}

// WithScheduleToCloseTimeout adds a timeout to the copy of the context.
// The current timeout resolution implementation is in seconds and uses math.Ceil(d.Seconds()) as the duration. But is
// subjected to change in the future.
func WithScheduleToCloseTimeout(ctx Context, d time.Duration) Context {
	ctx1 := setActivityParametersIfNotExist(ctx)
	getActivityOptions(ctx1).ScheduleToCloseTimeout = d
	return ctx1
}

// WithScheduleToStartTimeout adds a timeout to the copy of the context.
// The current timeout resolution implementation is in seconds and uses math.Ceil(d.Seconds()) as the duration. But is
// subjected to change in the future.
func WithScheduleToStartTimeout(ctx Context, d time.Duration) Context {
	ctx1 := setActivityParametersIfNotExist(ctx)
	getActivityOptions(ctx1).ScheduleToStartTimeout = d
	return ctx1
}

// WithStartToCloseTimeout adds a timeout to the copy of the context.
// The current timeout resolution implementation is in seconds and uses math.Ceil(d.Seconds()) as the duration. But is
// subjected to change in the future.
func WithStartToCloseTimeout(ctx Context, d time.Duration) Context {
	ctx1 := setActivityParametersIfNotExist(ctx)
	getActivityOptions(ctx1).StartToCloseTimeout = d
	return ctx1
}

// WithHeartbeatTimeout adds a timeout to the copy of the context.
// The current timeout resolution implementation is in seconds and uses math.Ceil(d.Seconds()) as the duration. But is
// subjected to change in the future.
func WithHeartbeatTimeout(ctx Context, d time.Duration) Context {
	ctx1 := setActivityParametersIfNotExist(ctx)
	getActivityOptions(ctx1).HeartbeatTimeout = d
	return ctx1
}

// WithWaitForCancellation adds wait for the cacellation to the copy of the context.
func WithWaitForCancellation(ctx Context, wait bool) Context {
	ctx1 := setActivityParametersIfNotExist(ctx)
	getActivityOptions(ctx1).WaitForCancellation = wait
	return ctx1
}

// WithRetryPolicy adds retry policy to the copy of the context
func WithRetryPolicy(ctx Context, retryPolicy RetryPolicy) Context {
	ctx1 := setActivityParametersIfNotExist(ctx)
	getActivityOptions(ctx1).RetryPolicy = convertToPBRetryPolicy(&retryPolicy)
	return ctx1
}

func convertToPBRetryPolicy(retryPolicy *RetryPolicy) *commonpb.RetryPolicy {
	if retryPolicy == nil {
		return nil
	}

	return &commonpb.RetryPolicy{
		MaximumInterval:        &retryPolicy.MaximumInterval,
		InitialInterval:        &retryPolicy.InitialInterval,
		BackoffCoefficient:     retryPolicy.BackoffCoefficient,
		MaximumAttempts:        retryPolicy.MaximumAttempts,
		NonRetryableErrorTypes: retryPolicy.NonRetryableErrorTypes,
	}
}

func convertFromPBRetryPolicy(retryPolicy *commonpb.RetryPolicy) *RetryPolicy {
	if retryPolicy == nil {
		return nil
	}

	p := RetryPolicy{
		BackoffCoefficient:     retryPolicy.BackoffCoefficient,
		MaximumAttempts:        retryPolicy.MaximumAttempts,
		NonRetryableErrorTypes: retryPolicy.NonRetryableErrorTypes,
	}

	// Avoid nil pointer dereferences
	if v := retryPolicy.MaximumInterval; v != nil {
		p.MaximumInterval = *v
	}
	if v := retryPolicy.InitialInterval; v != nil {
		p.InitialInterval = *v
	}

	return &p
}

// GetLastCompletionResultFromWorkflowInfo returns value of last completion result.
func GetLastCompletionResultFromWorkflowInfo(info *WorkflowInfo) *commonpb.Payloads {
	return info.lastCompletionResult
}
