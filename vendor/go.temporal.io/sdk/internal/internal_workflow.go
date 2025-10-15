package internal

// All code in this file is private to the package.

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/sdk/v1"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/log"
)

const (
	defaultSignalChannelSize    = 100000 // really large buffering size(100K)
	defaultCoroutineExitTimeout = 100 * time.Millisecond

	panicIllegalAccessCoroutineState = "getState: illegal access from outside of workflow context"
	unhandledUpdateWarningMessage    = "[TMPRL1102] Workflow finished while update handlers are still running. This may have interrupted work that the" +
		" update handler was doing, and the client that sent the update will receive a 'workflow execution" +
		" already completed' RPCError instead of the update result. You can wait for all update" +
		" handlers to complete by using `workflow.Await(ctx, func() bool { return workflow.AllHandlersFinished(ctx) })`. Alternatively, if both you and the clients sending the update" +
		" are okay with interrupting running handlers when the workflow finishes, and causing clients to" +
		" receive errors, then you can disable this warning via UnfinishedPolicy in UpdateHandlerOptions."
)

type (
	syncWorkflowDefinition struct {
		workflow   workflow
		dispatcher dispatcher
		cancel     CancelFunc
		rootCtx    Context
	}

	workflowResult struct {
		workflowResult *commonpb.Payloads
		error          error
	}

	futureImpl struct {
		value   interface{}
		err     error
		ready   bool
		channel *channelImpl
		chained []asyncFuture // Futures that are chained to this one
	}

	// Implements WaitGroup interface
	waitGroupImpl struct {
		n        int      // the number of coroutines to wait on
		waiting  bool     // indicates whether WaitGroup.Wait() has been called yet for the WaitGroup
		future   Future   // future to signal that all awaited members of the WaitGroup have completed
		settable Settable // used to unblock the future when all coroutines have completed
	}

	// Implements Mutex interface
	mutexImpl struct {
		locked bool
	}

	// Implements Semaphore interface
	semaphoreImpl struct {
		size int64
		cur  int64
	}

	// Dispatcher is a container of a set of coroutines.
	dispatcher interface {
		// ExecuteUntilAllBlocked executes coroutines one by one in deterministic order
		// until all of them are completed or blocked on Channel or Selector or timeout is reached.
		ExecuteUntilAllBlocked(deadlockDetectionTimeout time.Duration) (err error)
		// IsDone returns true when all of coroutines are completed
		IsDone() bool
		IsClosed() bool
		IsExecuting() bool
		Close()             // Destroys all coroutines without waiting for their completion
		StackTrace() string // Stack trace of all coroutines owned by the Dispatcher instance

		// Create coroutine. To be called from within other coroutine.
		// Used by the interceptors
		NewCoroutine(ctx Context, name string, highPriority bool, f func(ctx Context)) Context
	}

	// Workflow is an interface that any workflow should implement.
	// Code of a workflow must be deterministic. It must use workflow.Channel, workflow.Selector, and workflow.Go instead of
	// native channels, select and go. It also must not use range operation over map as it is randomized by go runtime.
	// All time manipulation should use current time returned by GetTime(ctx) method.
	// Note that workflow.Context is used instead of context.Context to avoid use of raw channels.
	workflow interface {
		Execute(ctx Context, input *commonpb.Payloads) (result *commonpb.Payloads, err error)
	}

	sendCallback struct {
		value interface{}
		fn    func() bool // false indicates that callback didn't accept the value
	}

	receiveCallback struct {
		// false result means that callback didn't accept the value and it is still up for delivery
		fn func(v interface{}, more bool) bool
	}

	channelImpl struct {
		name            string                  // human readable channel name
		size            int                     // Channel buffer size. 0 for non buffered.
		buffer          []interface{}           // buffered messages
		blockedSends    []*sendCallback         // puts waiting when buffer is full.
		blockedReceives []*receiveCallback      // receives waiting when no messages are available.
		closed          bool                    // true if channel is closed.
		recValue        *interface{}            // Used only while receiving value, this is used as pre-fetch buffer value from the channel.
		dataConverter   converter.DataConverter // for decode data
		env             WorkflowEnvironment
	}

	// Single case statement of the Select
	selectCase struct {
		channel     *channelImpl                       // Channel of this case.
		receiveFunc *func(c ReceiveChannel, more bool) // function to call when channel has a message. nil for send case.

		sendFunc   *func()         // function to call when channel accepted a message. nil for receive case.
		sendValue  *interface{}    // value to send to the channel. Used only for send case.
		future     asyncFuture     // Used for future case
		futureFunc *func(f Future) // function to call when Future is ready
	}

	// Implements Selector interface
	selectorImpl struct {
		name        string
		cases       []*selectCase // cases that this select is comprised from
		defaultFunc *func()       // default case
	}

	// unblockFunc is passed evaluated by a coroutine yield. When it returns false the yield returns to a caller.
	// stackDepth is the depth of stack from the last blocking call relevant to user.
	// Used to truncate internal stack frames from thread stack.
	unblockFunc func(status string, stackDepth int) (keepBlocked bool)

	coroutineState struct {
		name         string
		dispatcher   *dispatcherImpl  // dispatcher this context belongs to
		aboutToBlock chan bool        // used to notify dispatcher that coroutine that owns this context is about to block
		unblock      chan unblockFunc // used to notify coroutine that it should continue executing.
		keptBlocked  bool             // true indicates that coroutine didn't make any progress since the last yield unblocking
		closed       atomic.Bool      // indicates that owning coroutine has finished execution
		blocked      atomic.Bool
		panicError   error // non nil if coroutine had unhandled panic
	}

	dispatcherImpl struct {
		sequence         int
		channelSequence  int // used to name channels
		selectorSequence int // used to name channels
		coroutines       []*coroutineState
		executing        bool       // currently running ExecuteUntilAllBlocked. Used to avoid recursive calls to it.
		mutex            sync.Mutex // used to synchronize executing
		closed           bool
		interceptor      WorkflowOutboundInterceptor
		logger           log.Logger
		deadlockDetector *deadlockDetector
		readOnly         bool
		// allBlockedCallback is called when all coroutines are blocked,
		// returns true if the callback updated any coroutines state and there may be more work
		allBlockedCallback func() bool
		newEagerCoroutines []*coroutineState
	}

	// WorkflowOptions options passed to the workflow function
	// The current timeout resolution implementation is in seconds and uses math.Ceil() as the duration. But is
	// subjected to change in the future.
	WorkflowOptions struct {
		TaskQueueName            string
		WorkflowExecutionTimeout time.Duration
		WorkflowRunTimeout       time.Duration
		WorkflowTaskTimeout      time.Duration
		Namespace                string
		WorkflowID               string
		WaitForCancellation      bool
		WorkflowIDReusePolicy    enumspb.WorkflowIdReusePolicy
		// WorkflowIDConflictPolicy and OnConflictOptions are only used in test environment for
		// running Nexus operations as child workflow.
		WorkflowIDConflictPolicy enumspb.WorkflowIdConflictPolicy
		OnConflictOptions        *OnConflictOptions
		DataConverter            converter.DataConverter
		RetryPolicy              *commonpb.RetryPolicy
		Priority                 *commonpb.Priority
		CronSchedule             string
		ContextPropagators       []ContextPropagator
		Memo                     map[string]interface{}
		SearchAttributes         map[string]interface{}
		TypedSearchAttributes    SearchAttributes
		ParentClosePolicy        enumspb.ParentClosePolicy
		StaticSummary            string
		StaticDetails            string
		signalChannels           map[string]Channel
		requestedSignalChannels  map[string]*requestedSignalChannel
		queryHandlers            map[string]*queryHandler
		updateHandlers           map[string]*updateHandler
		// runningUpdatesHandles is a map of update handlers that are currently running.
		runningUpdatesHandles map[string]UpdateInfo
		VersioningIntent      VersioningIntent
		// currentDetails is the user-set string returned on metadata query as
		// WorkflowMetadata.current_details
		currentDetails string
	}

	// ExecuteWorkflowParams parameters of the workflow invocation
	ExecuteWorkflowParams struct {
		WorkflowOptions
		WorkflowType         *WorkflowType
		Input                *commonpb.Payloads
		Header               *commonpb.Header
		attempt              int32              // used by test framework to support child workflow retry
		scheduledTime        time.Time          // used by test framework to support child workflow retry
		lastCompletionResult *commonpb.Payloads // used by test framework to support cron
	}

	// decodeFutureImpl
	decodeFutureImpl struct {
		*futureImpl
		fn interface{}
	}

	childWorkflowFutureImpl struct {
		*decodeFutureImpl             // for child workflow result
		executionFuture   *futureImpl // for child workflow execution future
	}

	nexusOperationFutureImpl struct {
		*decodeFutureImpl             // for the result
		executionFuture   *futureImpl // for the NexusOperationExecution
	}

	asyncFuture interface {
		Future
		// Used by selectorImpl
		// If Future is ready returns its value immediately.
		// If not registers callback which is called when it is ready.
		GetAsync(callback *receiveCallback) (v interface{}, ok bool, err error)

		// Used by selectorImpl
		RemoveReceiveCallback(callback *receiveCallback)

		// This future will added to list of dependency futures.
		ChainFuture(f Future)

		// Gets the current value and error.
		// Make sure this is called once the future is ready.
		GetValueAndError() (v interface{}, err error)

		Set(value interface{}, err error)
	}

	requestedSignalChannel struct {
		options SignalChannelOptions
	}

	queryHandler struct {
		fn            interface{}
		queryType     string
		dataConverter converter.DataConverter
		options       QueryHandlerOptions
	}

	// updateSchedulerImpl adapts the coro dispatcher to the UpdateScheduler interface
	updateSchedulerImpl struct {
		dispatcher dispatcher
	}
)

const (
	workflowEnvironmentContextKey    = "workflowEnv"
	workflowInterceptorContextKey    = "workflowInterceptor"
	localActivityFnContextKey        = "localActivityFn"
	workflowEnvInterceptorContextKey = "envInterceptor"
	workflowResultContextKey         = "workflowResult"
	coroutinesContextKey             = "coroutines"
	workflowEnvOptionsContextKey     = "wfEnvOptions"
	updateInfoContextKey             = "updateInfo"
)

// Assert that structs do indeed implement the interfaces
var _ Channel = (*channelImpl)(nil)
var _ Selector = (*selectorImpl)(nil)
var _ WaitGroup = (*waitGroupImpl)(nil)
var _ dispatcher = (*dispatcherImpl)(nil)

// 1MB buffer to fit combined stack trace of all active goroutines
var stackBuf [1024 * 1024]byte

var (
	errCoroStackNotFound   = errors.New("coroutine stack not found")
	errStackTraceTruncated = errors.New("stack trace truncated: stackBuf is too small")
)

// Pointer to pointer to workflow result
func getWorkflowResultPointerPointer(ctx Context) **workflowResult {
	rpp := ctx.Value(workflowResultContextKey)
	if rpp == nil {
		panic("getWorkflowResultPointerPointer: Not a workflow context")
	}
	return rpp.(**workflowResult)
}

func getWorkflowEnvironment(ctx Context) WorkflowEnvironment {
	wc := ctx.Value(workflowEnvironmentContextKey)
	if wc == nil {
		panic("getWorkflowContext: Not a workflow context")
	}
	return wc.(WorkflowEnvironment)
}

func getWorkflowEnvironmentInterceptor(ctx Context) *workflowEnvironmentInterceptor {
	wc := ctx.Value(workflowEnvInterceptorContextKey)
	if wc == nil {
		panic("getWorkflowContext: Not a workflow context")
	}
	return wc.(*workflowEnvironmentInterceptor)
}

type workflowEnvironmentInterceptor struct {
	env                 WorkflowEnvironment
	dispatcher          dispatcher
	inboundInterceptor  WorkflowInboundInterceptor
	fn                  interface{}
	outboundInterceptor WorkflowOutboundInterceptor
}

func (wc *workflowEnvironmentInterceptor) Go(ctx Context, name string, f func(ctx Context)) Context {
	return wc.dispatcher.NewCoroutine(ctx, name, false, f)
}

func getWorkflowOutboundInterceptor(ctx Context) WorkflowOutboundInterceptor {
	wc := ctx.Value(workflowInterceptorContextKey)
	if wc == nil {
		panic("getWorkflowOutboundInterceptor: Not a workflow context")
	}
	return wc.(WorkflowOutboundInterceptor)
}

func (f *futureImpl) Get(ctx Context, valuePtr interface{}) error {
	assertNotInReadOnlyState(ctx)
	more := f.channel.Receive(ctx, nil)
	if more {
		panic("not closed")
	}
	if !f.ready {
		panic("not ready")
	}
	if f.err != nil || f.value == nil || valuePtr == nil {
		return f.err
	}
	rf := reflect.ValueOf(valuePtr)
	if rf.Type().Kind() != reflect.Ptr {
		return errors.New("valuePtr parameter is not a pointer")
	}

	if payload, ok := f.value.(*commonpb.Payloads); ok {
		if _, ok2 := valuePtr.(**commonpb.Payloads); !ok2 {
			if err := decodeArg(getDataConverterFromWorkflowContext(ctx), payload, valuePtr); err != nil {
				return err
			}
			return f.err
		}
	}

	fv := reflect.ValueOf(f.value)
	// If the value set was a pointer and is the same type as the wanted result,
	// instead of panicking because it is not a pointer to a pointer, we will just
	// set the pointer
	if fv.Kind() == reflect.Ptr && fv.Type() == rf.Type() {
		rf.Elem().Set(fv.Elem())
	} else {
		rf.Elem().Set(fv)
	}
	return f.err
}

// Used by selectorImpl
// If Future is ready returns its value immediately.
// If not registers callback which is called when it is ready.
func (f *futureImpl) GetAsync(callback *receiveCallback) (v interface{}, ok bool, err error) {
	_, _, more := f.channel.receiveAsyncImpl(callback)
	// Future uses Channel.Close to indicate that it is ready.
	// So more being true (channel is still open) indicates future is not ready.
	if more {
		return nil, false, nil
	}
	if !f.ready {
		panic("not ready")
	}
	return f.value, true, f.err
}

// RemoveReceiveCallback removes the callback from future's channel to avoid closure leak.
// Used by selectorImpl
func (f *futureImpl) RemoveReceiveCallback(callback *receiveCallback) {
	f.channel.removeReceiveCallback(callback)
}

func (f *futureImpl) IsReady() bool {
	return f.ready
}

func (f *futureImpl) Set(value interface{}, err error) {
	if f.ready {
		panic("already set")
	}
	f.value = value
	f.err = err
	f.ready = true
	f.channel.Close()
	for _, ch := range f.chained {
		ch.Set(f.value, f.err)
	}
}

func (f *futureImpl) SetValue(value interface{}) {
	if f.ready {
		panic("already set")
	}
	f.Set(value, nil)
}

func (f *futureImpl) SetError(err error) {
	if f.ready {
		panic("already set")
	}
	f.Set(nil, err)
}

func (f *futureImpl) Chain(future Future) {
	if f.ready {
		panic("already set")
	}

	ch, ok := future.(asyncFuture)
	if !ok {
		panic("cannot chain Future that wasn't created with workflow.NewFuture")
	}
	if !ch.IsReady() {
		ch.ChainFuture(f)
		return
	}
	val, err := ch.GetValueAndError()
	f.value = val
	f.err = err
	f.ready = true
}

func (f *futureImpl) ChainFuture(future Future) {
	f.chained = append(f.chained, future.(asyncFuture))
}

func (f *futureImpl) GetValueAndError() (interface{}, error) {
	return f.value, f.err
}

func (f *childWorkflowFutureImpl) GetChildWorkflowExecution() Future {
	return f.executionFuture
}

func (f *childWorkflowFutureImpl) SignalChildWorkflow(ctx Context, signalName string, data interface{}) Future {
	assertNotInReadOnlyState(ctx)
	var childExec WorkflowExecution
	if err := f.GetChildWorkflowExecution().Get(ctx, &childExec); err != nil {
		return f.GetChildWorkflowExecution()
	}

	i := getWorkflowOutboundInterceptor(ctx)
	// Put header on context before executing
	ctx = workflowContextWithNewHeader(ctx)
	return i.SignalChildWorkflow(ctx, childExec.ID, signalName, data)
}

func (f *nexusOperationFutureImpl) GetNexusOperationExecution() Future {
	return f.executionFuture
}

func newWorkflowContext(
	env WorkflowEnvironment,
	interceptors []WorkerInterceptor,
) (*workflowEnvironmentInterceptor, Context, error) {
	// Create context with default values
	ctx := WithValue(background, workflowEnvironmentContextKey, env)
	var resultPtr *workflowResult
	ctx = WithValue(ctx, workflowResultContextKey, &resultPtr)
	info := env.WorkflowInfo()
	ctx = WithWorkflowNamespace(ctx, info.Namespace)
	ctx = WithWorkflowTaskQueue(ctx, info.TaskQueueName)
	getWorkflowEnvOptions(ctx).WorkflowExecutionTimeout = info.WorkflowExecutionTimeout
	ctx = WithWorkflowRunTimeout(ctx, info.WorkflowRunTimeout)
	ctx = WithWorkflowTaskTimeout(ctx, info.WorkflowTaskTimeout)
	ctx = WithTaskQueue(ctx, info.TaskQueueName)
	ctx = WithDataConverter(ctx, env.GetDataConverter())
	ctx = withContextPropagators(ctx, env.GetContextPropagators())
	getActivityOptions(ctx).OriginalTaskQueueName = info.TaskQueueName

	// Create interceptor and put it on context as inbound and put it on context
	// as the default outbound interceptor before init
	envInterceptor := &workflowEnvironmentInterceptor{env: env}
	envInterceptor.inboundInterceptor = envInterceptor
	envInterceptor.outboundInterceptor = envInterceptor
	ctx = WithValue(ctx, workflowEnvInterceptorContextKey, envInterceptor)
	ctx = WithValue(ctx, workflowInterceptorContextKey, envInterceptor.outboundInterceptor)

	// Intercept, run init, and put the new outbound interceptor on the context
	for i := len(interceptors) - 1; i >= 0; i-- {
		envInterceptor.inboundInterceptor = interceptors[i].InterceptWorkflow(ctx, envInterceptor.inboundInterceptor)
	}
	err := envInterceptor.inboundInterceptor.Init(envInterceptor)
	if err != nil {
		return nil, nil, err
	}
	ctx = WithValue(ctx, workflowInterceptorContextKey, envInterceptor.outboundInterceptor)

	return envInterceptor, ctx, nil
}

func (d *syncWorkflowDefinition) Execute(env WorkflowEnvironment, header *commonpb.Header, input *commonpb.Payloads) {
	envInterceptor, rootCtx, err := newWorkflowContext(env, env.GetRegistry().interceptors)
	if err != nil {
		panic(err)
	}
	dispatcher, rootCtx := newDispatcher(
		rootCtx,
		envInterceptor,
		func(ctx Context) {
			r := &workflowResult{}

			// We want to execute the user workflow definition from the first workflow task started,
			// so they can see everything before that. Here we would have all initialization done, hence
			// we are yielding.
			state := getState(d.rootCtx)
			state.yield("yield before executing to setup state")
			state.unblocked()

			r.workflowResult, r.error = d.workflow.Execute(d.rootCtx, input)
			rpp := getWorkflowResultPointerPointer(ctx)
			*rpp = r
		}, getWorkflowEnvironment(rootCtx).DrainUnhandledUpdates)

	// set the information from the headers that is to be propagated in the workflow context
	rootCtx, err = workflowContextWithHeaderPropagated(rootCtx, header, env.GetContextPropagators())
	if err != nil {
		panic(err)
	}

	d.rootCtx, d.cancel = WithCancel(rootCtx)
	d.dispatcher = dispatcher
	envInterceptor.dispatcher = dispatcher

	getWorkflowEnvironment(d.rootCtx).RegisterCancelHandler(func() {
		// It is ok to call this method multiple times.
		// it doesn't do anything new, the context remains canceled.
		d.cancel()
	})

	getWorkflowEnvironment(d.rootCtx).RegisterSignalHandler(
		func(name string, input *commonpb.Payloads, header *commonpb.Header) error {
			// Put the header on context
			rootCtx, err := workflowContextWithHeaderPropagated(d.rootCtx, header, env.GetContextPropagators())
			if err != nil {
				return err
			}
			return envInterceptor.inboundInterceptor.HandleSignal(rootCtx, &HandleSignalInput{SignalName: name, Arg: input})
		},
	)

	getWorkflowEnvironment(d.rootCtx).RegisterUpdateHandler(
		func(name string, id string, serializedArgs *commonpb.Payloads, header *commonpb.Header, callbacks UpdateCallbacks) {
			defaultUpdateHandler(d.rootCtx, name, id, serializedArgs, header, callbacks, updateSchedulerImpl{d.dispatcher})
		})

	getWorkflowEnvironment(d.rootCtx).RegisterQueryHandler(
		func(queryType string, queryArgs *commonpb.Payloads, header *commonpb.Header) (*commonpb.Payloads, error) {
			// Put the header on context if server supports it
			rootCtx, err := workflowContextWithHeaderPropagated(d.rootCtx, header, env.GetContextPropagators())
			if err != nil {
				return nil, err
			}

			// As a special case, we handle __temporal_workflow_metadata query
			// here instead of in workflowExecutionEventHandlerImpl.ProcessQuery
			// because we need the context environment to do so.
			if queryType == QueryTypeWorkflowMetadata {
				if result, err := getWorkflowMetadata(rootCtx); err != nil {
					return nil, err
				} else {
					// Use raw value built from default converter because we don't want to use
					// user-conversion
					resultPayload, err := converter.GetDefaultDataConverter().ToPayload(result)
					if err != nil {
						return nil, err
					}
					return encodeArg(getDataConverterFromWorkflowContext(rootCtx), converter.NewRawValue(resultPayload))
				}
			}

			eo := getWorkflowEnvOptions(rootCtx)
			// A handler must be present since it is needed for argument decoding,
			// even if the interceptor intercepts query handling
			handler, ok := eo.queryHandlers[queryType]
			if !ok {
				keys := []string{QueryTypeStackTrace, QueryTypeOpenSessions, QueryTypeWorkflowMetadata}
				for k := range eo.queryHandlers {
					keys = append(keys, k)
				}
				return nil, fmt.Errorf("unknown queryType %v. KnownQueryTypes=%v", queryType, keys)
			}

			// Decode the arguments
			args, err := decodeArgsToRawValues(handler.dataConverter, reflect.TypeOf(handler.fn), queryArgs)
			if err != nil {
				return nil, fmt.Errorf("unable to decode the input for queryType: %v, with error: %w", handler.queryType, err)
			}

			// Invoke
			result, err := envInterceptor.inboundInterceptor.HandleQuery(
				rootCtx,
				&HandleQueryInput{QueryType: queryType, Args: args},
			)

			// Encode the result
			var serializedResult *commonpb.Payloads
			if err == nil {
				serializedResult, err = encodeArg(handler.dataConverter, result)
			}
			return serializedResult, err
		},
	)
}

func (d *syncWorkflowDefinition) OnWorkflowTaskStarted(deadlockDetectionTimeout time.Duration) {
	executeDispatcher(d.rootCtx, d.dispatcher, deadlockDetectionTimeout)
}

func (d *syncWorkflowDefinition) StackTrace() string {
	return d.dispatcher.StackTrace()
}

func (d *syncWorkflowDefinition) Close() {
	if d.dispatcher != nil {
		d.dispatcher.Close()
	}
}

// NewDispatcher creates a new Dispatcher instance with a root coroutine function.
// Context passed to the root function is child of the passed rootCtx.
// This way rootCtx can be used to pass values to the coroutine code.
func newDispatcher(rootCtx Context, interceptor *workflowEnvironmentInterceptor, root func(ctx Context), allBlockedCallback func() bool) (*dispatcherImpl, Context) {
	env := getWorkflowEnvironment(rootCtx)

	result := &dispatcherImpl{
		interceptor:        interceptor.outboundInterceptor,
		logger:             env.GetLogger(),
		deadlockDetector:   newDeadlockDetector(),
		allBlockedCallback: allBlockedCallback,
	}
	interceptor.dispatcher = result
	ctxWithState := result.interceptor.Go(rootCtx, "root", root)
	return result, ctxWithState
}

// executeDispatcher executed coroutines in the calling thread and calls workflow completion callbacks
// if root workflow function returned
func executeDispatcher(ctx Context, dispatcher dispatcher, timeout time.Duration) {
	env := getWorkflowEnvironment(ctx)
	panicErr := dispatcher.ExecuteUntilAllBlocked(timeout)
	if panicErr != nil {
		env.Complete(nil, panicErr)
		return
	}

	rp := *getWorkflowResultPointerPointer(ctx)
	if rp == nil {
		// Result is not set, so workflow is still executing
		return
	}

	weo := getWorkflowEnvOptions(ctx)
	us := weo.getUnhandledSignalNames()
	if len(us) > 0 {
		env.GetLogger().Warn("Workflow has unhandled signals", "SignalNames", us)
	}
	// Warn if there are any update handlers still running
	type warnUpdate struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	}
	var updatesToWarn []warnUpdate
	for _, info := range weo.getRunningUpdateHandles() {
		if weo.updateHandlers[info.Name].unfinishedPolicy == HandlerUnfinishedPolicyWarnAndAbandon {
			updatesToWarn = append(updatesToWarn, warnUpdate{
				Name: info.Name,
				ID:   info.ID,
			})
		}
	}

	// Verify that the workflow did not fail. If it did we will not warn about unhandled updates.
	var canceledErr *CanceledError
	var contErr *ContinueAsNewError
	if len(updatesToWarn) > 0 && (rp.error == nil || errors.As(rp.error, &canceledErr) || errors.As(rp.error, &contErr)) {
		env.GetLogger().Warn(unhandledUpdateWarningMessage, "Updates", updatesToWarn)
	}

	env.Complete(rp.workflowResult, rp.error)
}

// For troubleshooting stack pretty printing only.
// Set to true to see full stack trace that includes framework methods.
const disableCleanStackTraces = false

func getState(ctx Context) *coroutineState {
	s := ctx.Value(coroutinesContextKey)
	if s == nil {
		panic("getState: not workflow context")
	}
	state := s.(*coroutineState)
	if !state.dispatcher.IsExecuting() {
		panic(panicIllegalAccessCoroutineState)
	}
	return state
}

func assertNotInReadOnlyState(ctx Context) {
	state := getState(ctx)
	// use the dispatcher state instead of the coroutine state because contexts can be
	// shared
	if state.dispatcher.getIsReadOnly() {
		panic(panicIllegalAccessCoroutineState)
	}
}

func assertNotInReadOnlyStateCancellation(ctx Context) {
	s := ctx.Value(coroutinesContextKey)
	if s == nil {
		panic("assertNotInReadOnlyStateCtxCancellation: not workflow context")
	}
	state := s.(*coroutineState)
	// For cancellation the dispatcher may not be running because workflow cancellation
	// is sent outside of the dispatchers loop.
	if state.dispatcher.IsClosed() {
		panic(panicIllegalAccessCoroutineState)
	}
	// use the dispatcher state instead of the coroutine state because contexts can be
	// shared
	if state.dispatcher.getIsReadOnly() {
		panic(panicIllegalAccessCoroutineState)
	}
}

func getStateIfRunning(ctx Context) *coroutineState {
	if ctx == nil {
		return nil
	}
	s := ctx.Value(coroutinesContextKey)
	if s == nil {
		return nil
	}
	state := s.(*coroutineState)
	if !state.dispatcher.IsExecuting() {
		return nil
	}
	return state
}

func (c *channelImpl) Name() string {
	return c.name
}

func (c *channelImpl) CanReceiveWithoutBlocking() bool {
	return c.recValue != nil || len(c.buffer) > 0 || len(c.blockedSends) > 0 || c.closed
}

func (c *channelImpl) CanSendWithoutBlocking() bool {
	return len(c.buffer) < c.size || len(c.blockedReceives) > 0
}

func (c *channelImpl) Receive(ctx Context, valuePtr interface{}) (more bool) {
	assertNotInReadOnlyState(ctx)
	state := getState(ctx)
	hasResult := false
	var result interface{}
	callback := &receiveCallback{
		fn: func(v interface{}, m bool) bool {
			result = v
			hasResult = true
			more = m
			return true
		},
	}

	for {
		hasResult = false
		v, ok, m := c.receiveAsyncImpl(callback)

		if !ok && !m { // channel closed and empty
			return m
		}

		if ok || !m {
			err := c.assignValue(v, valuePtr)
			if err == nil {
				state.unblocked()
				return m
			}
			continue // corrupt signal. Drop and reset process
		}
		for {
			if hasResult {
				err := c.assignValue(result, valuePtr)
				if err == nil {
					state.unblocked()
					return more
				}
				break // Corrupt signal. Drop and reset process.
			}
			state.yield("blocked on " + c.name + ".Receive")
		}
	}

}

func (c *channelImpl) ReceiveWithTimeout(ctx Context, timeout time.Duration, valuePtr interface{}) (ok, more bool) {
	okAwait, err := AwaitWithTimeout(ctx, timeout, func() bool { return c.Len() > 0 })
	if err != nil { // context canceled
		return false, true
	}
	if !okAwait { // timed out
		return false, true
	}
	ok, more = c.ReceiveAsyncWithMoreFlag(valuePtr)
	if !ok {
		panic("unexpected empty channel")
	}
	return true, more
}

func (c *channelImpl) ReceiveAsync(valuePtr interface{}) (ok bool) {
	ok, _ = c.ReceiveAsyncWithMoreFlag(valuePtr)
	return ok
}

func (c *channelImpl) ReceiveAsyncWithMoreFlag(valuePtr interface{}) (ok bool, more bool) {
	for {
		v, ok, more := c.receiveAsyncImpl(nil)
		if !ok && !more { // channel closed and empty
			return ok, more
		}

		err := c.assignValue(v, valuePtr)
		if err != nil {
			continue
			// keep consuming until a good signal is hit or channel is drained
		}
		return ok, more
	}
}

func (c *channelImpl) Len() int {
	result := len(c.buffer) + len(c.blockedSends)
	if c.recValue != nil {
		result = result + 1
	}
	return result
}

// ok = true means that value was received
// more = true means that channel is not closed and more deliveries are possible
func (c *channelImpl) receiveAsyncImpl(callback *receiveCallback) (v interface{}, ok bool, more bool) {
	if c.recValue != nil {
		r := *c.recValue
		c.recValue = nil
		return r, true, true
	}
	if len(c.buffer) > 0 {
		r := c.buffer[0]
		c.buffer[0] = nil
		c.buffer = c.buffer[1:]

		// Move blocked sends into buffer
		for len(c.blockedSends) > 0 {
			b := c.blockedSends[0]
			c.blockedSends[0] = nil
			c.blockedSends = c.blockedSends[1:]
			if b.fn() {
				c.buffer = append(c.buffer, b.value)
				break
			}
		}

		return r, true, true
	}
	if c.closed {
		return nil, false, false
	}
	for len(c.blockedSends) > 0 {
		b := c.blockedSends[0]
		c.blockedSends[0] = nil
		c.blockedSends = c.blockedSends[1:]
		if b.fn() {
			return b.value, true, true
		}
	}
	if callback != nil {
		c.blockedReceives = append(c.blockedReceives, callback)
	}
	return nil, false, true
}

func (c *channelImpl) removeReceiveCallback(callback *receiveCallback) {
	for i, blockedCallback := range c.blockedReceives {
		if callback == blockedCallback {
			c.blockedReceives = append(c.blockedReceives[:i], c.blockedReceives[i+1:]...)
			break
		}
	}
}

func (c *channelImpl) removeSendCallback(callback *sendCallback) {
	for i, blockedCallback := range c.blockedSends {
		if callback == blockedCallback {
			c.blockedSends = append(c.blockedSends[:i], c.blockedSends[i+1:]...)
			break
		}
	}
}

func (c *channelImpl) Send(ctx Context, v interface{}) {
	state := getState(ctx)
	valueConsumed := false
	callback := &sendCallback{
		value: v,
		fn: func() bool {
			valueConsumed = true
			return true
		},
	}
	ok := c.sendAsyncImpl(v, callback)
	if ok {
		state.unblocked()
		return
	}
	for {
		if valueConsumed {
			state.unblocked()
			return
		}

		// Check for closed in the loop as close can be called when send is blocked
		if c.closed {
			panic("Closed channel")
		}
		state.yield("blocked on " + c.name + ".Send")
	}
}

func (c *channelImpl) SendAsync(v interface{}) (ok bool) {
	return c.sendAsyncImpl(v, nil)
}

func (c *channelImpl) sendAsyncImpl(v interface{}, pair *sendCallback) (ok bool) {
	if c.closed {
		panic("Closed channel")
	}
	for len(c.blockedReceives) > 0 {
		blockedGet := c.blockedReceives[0].fn
		c.blockedReceives[0] = nil
		c.blockedReceives = c.blockedReceives[1:]
		// false from callback indicates that value wasn't consumed
		if blockedGet(v, true) {
			return true
		}
	}
	if len(c.buffer) < c.size {
		c.buffer = append(c.buffer, v)
		return true
	}
	if pair != nil {
		c.blockedSends = append(c.blockedSends, pair)
	}
	return false
}

func (c *channelImpl) Close() {
	c.closed = true
	// Use a copy of blockedReceives for iteration as invoking callback could result in modification
	copy := append(c.blockedReceives[:0:0], c.blockedReceives...)
	for _, callback := range copy {
		callback.fn(nil, false)
	}
	// All blocked sends are going to panic
}

// Takes a value and assigns that 'to' value. logs a metric if it is unable to deserialize
func (c *channelImpl) assignValue(from interface{}, to interface{}) error {
	err := decodeAndAssignValue(c.dataConverter, from, to)
	// add to metrics
	if err != nil {
		c.env.GetLogger().Error(fmt.Sprintf("Deserialization error. Corrupted signal received on channel %s.", c.name), tagError, err)
		c.env.GetMetricsHandler().Counter(metrics.CorruptedSignalsCounter).Inc(1)
	}
	return err
}

// initialYield called at the beginning of the coroutine execution
// stackDepth is the depth of top of the stack to omit when stack trace is generated
// to hide frames internal to the framework.
func (s *coroutineState) initialYield(stackDepth int, status string) {
	if s.blocked.Swap(true) {
		panic("trying to block on coroutine which is already blocked, most likely a wrong Context is used to do blocking" +
			" call (like Future.Get() or Channel.Receive()")
	}
	keepBlocked := true
	for keepBlocked {
		f := <-s.unblock
		keepBlocked = f(status, stackDepth+1)
	}
	s.blocked.Swap(false)
}

// yield indicates that coroutine cannot make progress and should sleep
// this call blocks
func (s *coroutineState) yield(status string) {
	s.aboutToBlock <- true
	s.initialYield(3, status) // omit three levels of stack. To adjust change to 0 and count the lines to remove.
	s.keptBlocked = true
}

func getStackTrace(coroutineName, status string, stackDepth int) string {
	top := fmt.Sprintf("coroutine %s [%s]:", coroutineName, status)
	// Omit top stackDepth frames + top status line.
	// Omit bottom two frames which is wrapping of coroutine in a goroutine.
	return getStackTraceRaw(top, stackDepth*2+1, 4)
}

func getStackTraceRaw(top string, omitTop, omitBottom int) string {
	stack := stackBuf[:runtime.Stack(stackBuf[:], false)]
	outStack := filterStackTrace(string(stack), omitTop, omitBottom)
	return strings.Join([]string{top, outStack}, "\n")
}

func filterStackTrace(stack string, omitTop, omitBottom int) string {
	stack = strings.TrimRightFunc(stack, unicode.IsSpace)
	if disableCleanStackTraces {
		return stack
	}

	lines := strings.Split(stack, "\n")
	omitEnd := len(lines) - omitBottom
	// If the start is after the end, the depth was invalid originally so return
	// the entire raw stack
	if omitTop > omitEnd {
		return stack
	}
	return strings.Join(lines[omitTop:omitEnd], "\n")
}

func getCoroStackTrace(crt *coroutineState, status string, stackDepth int) (string, error) {
	// Can't dump goroutines selectively :(
	// Instead, we identify a coroutine's stack trace by the *coroutineState pointer address
	// in its function arguments. To avoid false positives, we also match on the fixed
	// member function name.
	stacks := stackBuf[:runtime.Stack(stackBuf[:], true)]
	needle := []byte(fmt.Sprintf("/internal.(*coroutineState).run(%p,", crt))
	idx := bytes.Index(stacks, needle)
	if idx == -1 {
		if len(stacks) == len(stackBuf) {
			return "", fmt.Errorf("coroutine not found: %w", errStackTraceTruncated)
		}
		// NOTE: This could happen if coroutineState is moved between runtime.Stack(...)
		// and formatting needle. However, Go's GC is currently non-moving.
		return "", errCoroStackNotFound
	}

	// coroStack spans from the stackDelim before idx to the stackDelim after idx
	stackDelim := []byte("\n\n")
	coroStack := stacks
	if start := bytes.LastIndex(stacks[:idx], stackDelim); start != -1 {
		start += len(stackDelim) // skip over delimiter
		coroStack = stacks[start:]
	}
	coroStack, _, _ = bytes.Cut(coroStack, stackDelim)

	// Omit top stackDepth frames + top status line.
	// Omit bottom two frames which is wrapping of coroutine in a goroutine.
	outStack := filterStackTrace(string(coroStack), stackDepth*2+1, 4)
	return fmt.Sprintf("coroutine %s [%s]:\n%s", crt.name, status, outStack), nil
}

// unblocked is called by coroutine to indicate that since the last time yield was unblocked channel or select
// where unblocked versus calling yield again after checking their condition
func (s *coroutineState) unblocked() {
	s.keptBlocked = false
}

func (s *coroutineState) call(timeout time.Duration) {
	s.unblock <- func(status string, stackDepth int) bool {
		return false // unblock
	}

	// Defaults are populated in the worker options during worker startup, but test environment
	// may have no default value for the deadlock detection timeout, so we also need to set it here for
	// backwards compatibility.
	if timeout == 0 {
		timeout = defaultDeadlockDetectionTimeout
		if debugMode {
			timeout = unlimitedDeadlockDetectionTimeout
		}
	}
	deadlockTicker := s.dispatcher.deadlockDetector.begin(timeout)
	defer deadlockTicker.end()

	select {
	case <-s.aboutToBlock:
	case <-deadlockTicker.reached():
		// Use workflowPanicError since this used to call panic(msg)
		st, err := getCoroStackTrace(s, "running", 0)
		if err != nil {
			st = fmt.Sprintf("<%s>", err)
		}
		msg := fmt.Sprintf("[TMPRL1101] Potential deadlock detected: "+
			"workflow goroutine %q didn't yield for over a second", s.name)
		s.closed.Store(true)
		s.panicError = newWorkflowPanicError(msg, st)
	}
}

func (s *coroutineState) close() {
	s.closed.Store(true)
	s.aboutToBlock <- true
}

// exit tries to run Goexit on the coroutine and wait for it to exit
// within timeout. If it doesn't exit within timeout, it will log a warning.
func (s *coroutineState) exit(logger log.Logger, warnTimeout time.Duration) {
	if !s.closed.Load() {
		s.unblock <- func(status string, stackDepth int) bool {
			runtime.Goexit()
			return true
		}

		timer := time.NewTimer(warnTimeout)
		defer timer.Stop()

		select {
		case <-s.aboutToBlock:
			return
		case <-timer.C:
			st, err := getCoroStackTrace(s, "running", 0)
			if err != nil {
				st = fmt.Sprintf("<%s>", err)
			}

			logger.Warn(fmt.Sprintf("Workflow coroutine %q didn't exit within %v", s.name, warnTimeout), "stackTrace", st)
		}
		// We need to make sure the coroutine is closed, otherwise we risk concurrent coroutines running
		// at the same time causing a race condition.
		<-s.aboutToBlock
	}
}

func (s *coroutineState) stackTrace() string {
	if s.closed.Load() {
		return ""
	}
	stackCh := make(chan string, 1)
	s.unblock <- func(status string, stackDepth int) bool {
		stackCh <- getStackTrace(s.name, status, stackDepth+2)
		return true
	}
	return <-stackCh
}

func (s *coroutineState) run(ctx Context, f func(ctx Context)) {
	defer runtime.KeepAlive(&s) // keep receiver argument alive for getCoroStackTrace
	defer s.close()
	defer func() {
		if r := recover(); r != nil {
			st := getStackTrace(s.name, "panic", 4)
			s.panicError = newWorkflowPanicError(r, st)
		}
	}()
	s.initialYield(1, "")
	f(ctx)
}

func (d *dispatcherImpl) NewCoroutine(ctx Context, name string, highPriority bool, f func(ctx Context)) Context {
	if name == "" {
		name = fmt.Sprintf("%v", d.sequence+1)
	}
	state := d.newState(name, highPriority)
	spawned := WithValue(ctx, coroutinesContextKey, state)
	go state.run(spawned, f)
	return spawned
}

func (d *dispatcherImpl) newState(name string, highPriority bool) *coroutineState {
	c := &coroutineState{
		name:         name,
		dispatcher:   d,
		aboutToBlock: make(chan bool, 1),
		unblock:      make(chan unblockFunc),
	}
	d.sequence++
	if highPriority {
		// Update requests need to be added to the front of the dispatchers coroutine list so they
		// are handled before the root coroutine.
		d.newEagerCoroutines = append(d.newEagerCoroutines, c)
	} else {
		d.coroutines = append(d.coroutines, c)
	}
	return c
}

func (d *dispatcherImpl) IsClosed() bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.closed
}

func (d *dispatcherImpl) ExecuteUntilAllBlocked(deadlockDetectionTimeout time.Duration) (err error) {
	d.mutex.Lock()
	if d.closed {
		d.mutex.Unlock()
		panic("dispatcher is closed")
	}
	if d.executing {
		d.mutex.Unlock()
		panic("call to ExecuteUntilAllBlocked (possibly from a coroutine) while it is already running")
	}
	d.executing = true
	d.mutex.Unlock()
	defer func() {
		d.mutex.Lock()
		d.executing = false
		d.mutex.Unlock()
	}()
	allBlocked := false
	// Keep executing until at least one goroutine made some progress
	for !allBlocked || d.allBlockedCallback() {
		d.coroutines = append(d.newEagerCoroutines, d.coroutines...)
		d.newEagerCoroutines = nil
		// Give every coroutine chance to execute removing closed ones
		allBlocked = true
		lastSequence := d.sequence
		for i := 0; i < len(d.coroutines); i++ {
			c := d.coroutines[i]
			if !c.closed.Load() {
				// TODO: Support handling of panic in a coroutine by dispatcher.
				// TODO: Dump all outstanding coroutines if one of them panics
				c.call(deadlockDetectionTimeout)
			}
			// c.call() can close the context so check again
			if c.closed.Load() {
				// remove the closed one from the slice
				d.coroutines = append(d.coroutines[:i],
					d.coroutines[i+1:]...)
				i--
				if c.panicError != nil {
					return c.panicError
				}
				allBlocked = false

			} else {
				allBlocked = allBlocked && (c.keptBlocked || c.closed.Load())
			}
			// If any eager coroutines were created by the last coroutine we
			// need to schedule them now.
			if len(d.newEagerCoroutines) > 0 {
				d.coroutines = slices.Insert(d.coroutines, i+1, d.newEagerCoroutines...)
				d.newEagerCoroutines = nil
				allBlocked = false
			}
		}
		// Set allBlocked to false if new coroutines where created
		allBlocked = allBlocked && lastSequence == d.sequence
	}
	return nil
}

func (d *dispatcherImpl) IsDone() bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return len(d.coroutines) == 0
}

func (d *dispatcherImpl) IsExecuting() bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.executing
}

func (d *dispatcherImpl) getIsReadOnly() bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.readOnly
}

func (d *dispatcherImpl) setIsReadOnly(readOnly bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.readOnly = readOnly
}

func (d *dispatcherImpl) Close() {
	d.mutex.Lock()
	if d.closed {
		d.mutex.Unlock()
		return
	}
	d.closed = true
	d.mutex.Unlock()
	// We need to exit the coroutines in a separate goroutine because:
	// 	* The coroutine may be stuck and won't respond to the exit request.
	// 	* On exit the coroutines defers will still run and that may block.
	go func() {
		for _, c := range d.coroutines {
			c.exit(d.logger, defaultDeadlockDetectionTimeout)
		}
	}()
}

func (d *dispatcherImpl) StackTrace() string {
	var result string
	for i := 0; i < len(d.coroutines); i++ {
		c := d.coroutines[i]
		if !c.closed.Load() {
			if len(result) > 0 {
				result += "\n\n"
			}
			result += c.stackTrace()
		}
	}
	return result
}

func (s *selectorImpl) AddReceive(c ReceiveChannel, f func(c ReceiveChannel, more bool)) Selector {
	s.cases = append(s.cases, &selectCase{channel: c.(*channelImpl), receiveFunc: &f})
	return s
}

func (s *selectorImpl) AddSend(c SendChannel, v interface{}, f func()) Selector {
	s.cases = append(s.cases, &selectCase{channel: c.(*channelImpl), sendFunc: &f, sendValue: &v})
	return s
}

func (s *selectorImpl) AddFuture(future Future, f func(future Future)) Selector {
	asyncF, ok := future.(asyncFuture)
	if !ok {
		panic("cannot chain Future that wasn't created with workflow.NewFuture")
	}
	s.cases = append(s.cases, &selectCase{future: asyncF, futureFunc: &f})
	return s
}

func (s *selectorImpl) AddDefault(f func()) {
	s.defaultFunc = &f
}

func (s *selectorImpl) HasPending() bool {
	for _, pair := range s.cases {
		if pair.receiveFunc != nil && pair.channel.CanReceiveWithoutBlocking() {
			return true
		} else if pair.sendFunc != nil && pair.channel.CanSendWithoutBlocking() {
			return true
		} else if pair.futureFunc != nil && pair.future.IsReady() {
			return true
		}
	}
	return false
}

func (s *selectorImpl) Select(ctx Context) {
	assertNotInReadOnlyState(ctx)
	state := getState(ctx)
	var readyBranch func()
	var cleanups []func()
	defer func() {
		for _, c := range cleanups {
			c()
		}
	}()

	for _, pair := range s.cases {
		if pair.receiveFunc != nil {
			f := *pair.receiveFunc
			c := pair.channel
			callback := &receiveCallback{
				fn: func(v interface{}, more bool) bool {
					if readyBranch != nil {
						return false
					}
					// readyBranch is not executed when AddDefault is specified,
					// setting the value here prevents the signal from being dropped
					env := getWorkflowEnvironment(ctx)
					var dropSignalFlag bool
					if unblockSelectorSignal {
						dropSignalFlag = env.TryUse(SDKFlagBlockedSelectorSignalReceive)
					} else {
						dropSignalFlag = env.GetFlag(SDKFlagBlockedSelectorSignalReceive)
					}

					if dropSignalFlag {
						c.recValue = &v
					}

					readyBranch = func() {
						if !dropSignalFlag {
							c.recValue = &v
						}
						f(c, more)
					}
					return true
				},
			}
			v, ok, more := c.receiveAsyncImpl(callback)
			if ok || !more {
				// Select() returns in this case/branch. The callback won't be called for this case. However, callback
				// will be called for previous cases/branches. We should set readyBranch so that when other case/branch
				// become ready they won't consume the value for this Select() call.
				readyBranch = func() {
				}
				// Avoid assigning pointer to nil interface which makes
				// c.RecValue != nil and breaks the nil check at the beginning of receiveAsyncImpl
				if more {
					c.recValue = &v
				} else {
					pair.receiveFunc = nil
				}
				f(c, more)
				return
			}
			// callback closure is added to channel's blockedReceives, we need to clean it up to avoid closure leak
			cleanups = append(cleanups, func() {
				c.removeReceiveCallback(callback)
			})
		} else if pair.sendFunc != nil {
			f := *pair.sendFunc
			c := pair.channel
			callback := &sendCallback{
				value: *pair.sendValue,
				fn: func() bool {
					if readyBranch != nil {
						return false
					}
					readyBranch = func() {
						f()
					}
					return true
				},
			}
			ok := c.sendAsyncImpl(*pair.sendValue, callback)
			if ok {
				// Select() returns in this case/branch. The callback won't be called for this case. However, callback
				// will be called for previous cases/branches. We should set readyBranch so that when other case/branch
				// become ready they won't consume the value for this Select() call.
				readyBranch = func() {
				}
				f()
				return
			}
			// callback closure is added to channel's blockedSends, we need to clean it up to avoid closure leak
			cleanups = append(cleanups, func() {
				c.removeSendCallback(callback)
			})
		} else if pair.futureFunc != nil {
			p := pair
			f := *p.futureFunc
			callback := &receiveCallback{
				fn: func(v interface{}, more bool) bool {
					if readyBranch != nil {
						return false
					}
					readyBranch = func() {
						p.futureFunc = nil
						f(p.future)
					}
					return true
				},
			}

			_, ok, _ := p.future.GetAsync(callback)
			if ok {
				// Select() returns in this case/branch. The callback won't be called for this case. However, callback
				// will be called for previous cases/branches. We should set readyBranch so that when other case/branch
				// become ready they won't consume the value for this Select() call.
				readyBranch = func() {
				}
				p.futureFunc = nil
				f(p.future)
				return
			}
			// callback closure is added to future's channel's blockedReceives, need to clean up to avoid leak
			cleanups = append(cleanups, func() {
				p.future.RemoveReceiveCallback(callback)
			})
		}
	}
	if s.defaultFunc != nil {
		f := *s.defaultFunc
		f()
		return
	}
	for {
		if readyBranch != nil {
			readyBranch()
			state.unblocked()
			return
		}
		state.yield("blocked on " + s.name + ".Select")
	}
}

// NewWorkflowDefinition creates a WorkflowDefinition from a Workflow
func newSyncWorkflowDefinition(workflow workflow) *syncWorkflowDefinition {
	return &syncWorkflowDefinition{workflow: workflow}
}

func getValidatedWorkflowFunction(workflowFunc interface{}, args []interface{}, dataConverter converter.DataConverter, r *registry) (*WorkflowType, *commonpb.Payloads, error) {
	if err := validateFunctionArgs(workflowFunc, args, true); err != nil {
		return nil, nil, err
	}

	fnName, err := getWorkflowFunctionName(r, workflowFunc)
	if err != nil {
		return nil, nil, err
	}

	if dataConverter == nil {
		dataConverter = converter.GetDefaultDataConverter()
	}
	input, err := encodeArgs(dataConverter, args)
	if err != nil {
		return nil, nil, err
	}
	return &WorkflowType{Name: fnName}, input, nil
}

func getWorkflowEnvOptions(ctx Context) *WorkflowOptions {
	options := ctx.Value(workflowEnvOptionsContextKey)
	if options != nil {
		return options.(*WorkflowOptions)
	}
	return nil
}

func setWorkflowEnvOptionsIfNotExist(ctx Context) Context {
	options := getWorkflowEnvOptions(ctx)
	var newOptions WorkflowOptions
	if options != nil {
		newOptions = *options
	} else {
		newOptions.signalChannels = make(map[string]Channel)
		newOptions.requestedSignalChannels = make(map[string]*requestedSignalChannel)
		newOptions.queryHandlers = make(map[string]*queryHandler)
		newOptions.updateHandlers = make(map[string]*updateHandler)
		newOptions.runningUpdatesHandles = make(map[string]UpdateInfo)
	}
	if newOptions.DataConverter == nil {
		newOptions.DataConverter = converter.GetDefaultDataConverter()
	}

	return WithValue(ctx, workflowEnvOptionsContextKey, &newOptions)
}

func getDataConverterFromWorkflowContext(ctx Context) converter.DataConverter {
	options := getWorkflowEnvOptions(ctx)
	var dataConverter converter.DataConverter

	if options != nil && options.DataConverter != nil {
		dataConverter = options.DataConverter
	} else {
		dataConverter = converter.GetDefaultDataConverter()
	}

	return WithWorkflowContext(ctx, dataConverter)
}

func getRegistryFromWorkflowContext(ctx Context) *registry {
	env := getWorkflowEnvironment(ctx)
	return env.GetRegistry()
}

// getSignalChannel finds the associated channel for the signal.
func (w *WorkflowOptions) getSignalChannel(ctx Context, signalName string) ReceiveChannel {
	if ch, ok := w.signalChannels[signalName]; ok {
		return ch
	}
	ch := NewNamedBufferedChannel(ctx, signalName, defaultSignalChannelSize)
	w.signalChannels[signalName] = ch
	return ch
}

// GetUnhandledSignalNames returns signal names that have unconsumed signals.
func GetUnhandledSignalNames(ctx Context) []string {
	return getWorkflowEnvOptions(ctx).getUnhandledSignalNames()
}

// GetCurrentDetails gets the previously-set current details.
//
// NOTE: Experimental
func GetCurrentDetails(ctx Context) string {
	return getWorkflowEnvOptions(ctx).currentDetails
}

// SetCurrentDetails sets the current details.
//
// NOTE: Experimental
func SetCurrentDetails(ctx Context, details string) {
	getWorkflowEnvOptions(ctx).currentDetails = details
}

func getWorkflowMetadata(ctx Context) (*sdk.WorkflowMetadata, error) {
	info := GetWorkflowInfo(ctx)
	eo := getWorkflowEnvOptions(ctx)
	ret := &sdk.WorkflowMetadata{
		Definition: &sdk.WorkflowDefinition{
			Type: info.WorkflowType.Name,
			QueryDefinitions: []*sdk.WorkflowInteractionDefinition{
				{
					Name:        QueryTypeStackTrace,
					Description: "Current stack trace",
				},
				{
					Name:        QueryTypeOpenSessions,
					Description: "Open sessions on the workflow",
				},
				{
					Name:        QueryTypeWorkflowMetadata,
					Description: "Metadata about the workflow",
				},
			},
		},
		CurrentDetails: eo.currentDetails,
	}
	// Queries
	for k, v := range eo.queryHandlers {
		ret.Definition.QueryDefinitions = append(ret.Definition.QueryDefinitions, &sdk.WorkflowInteractionDefinition{
			Name:        k,
			Description: v.options.Description,
		})
	}
	// Signals
	for k, v := range eo.requestedSignalChannels {
		ret.Definition.SignalDefinitions = append(ret.Definition.SignalDefinitions, &sdk.WorkflowInteractionDefinition{
			Name:        k,
			Description: v.options.Description,
		})
	}
	// Updates
	for k, v := range eo.updateHandlers {
		ret.Definition.UpdateDefinitions = append(ret.Definition.UpdateDefinitions, &sdk.WorkflowInteractionDefinition{
			Name:        k,
			Description: v.description,
		})
	}
	// Sort interaction definitions
	sortWorkflowInteractionDefinitions(ret.Definition.QueryDefinitions)
	sortWorkflowInteractionDefinitions(ret.Definition.SignalDefinitions)
	sortWorkflowInteractionDefinitions(ret.Definition.UpdateDefinitions)
	return ret, nil
}

func sortWorkflowInteractionDefinitions(defns []*sdk.WorkflowInteractionDefinition) {
	sort.Slice(defns, func(i, j int) bool { return defns[i].Name < defns[j].Name })
}

// getUnhandledSignalNames returns signal names that have unconsumed signals.
func (w *WorkflowOptions) getUnhandledSignalNames() []string {
	var unhandledSignals []string
	for k, c := range w.signalChannels {
		ch := c.(*channelImpl)
		v, ok, _ := ch.receiveAsyncImpl(nil)
		if ok {
			unhandledSignals = append(unhandledSignals, k)
			ch.recValue = &v
		}
	}
	return unhandledSignals
}

func (w *WorkflowOptions) getRunningUpdateHandles() map[string]UpdateInfo {
	return w.runningUpdatesHandles
}

func (d *decodeFutureImpl) Get(ctx Context, valuePtr interface{}) error {
	more := d.futureImpl.channel.Receive(ctx, nil)
	if more {
		panic("not closed")
	}
	if !d.futureImpl.ready {
		panic("not ready")
	}
	if d.futureImpl.err != nil || d.futureImpl.value == nil || valuePtr == nil {
		return d.futureImpl.err
	}
	rf := reflect.ValueOf(valuePtr)
	if rf.Type().Kind() != reflect.Ptr {
		return errors.New("valuePtr parameter is not a pointer")
	}
	dataConverter := getDataConverterFromWorkflowContext(ctx)
	err := dataConverter.FromPayloads(d.futureImpl.value.(*commonpb.Payloads), valuePtr)
	if err != nil {
		return err
	}
	return d.futureImpl.err
}

// newDecodeFuture creates a new future as well as associated Settable that is used to set its value.
// fn - the decoded value needs to be validated against a function.
func newDecodeFuture(ctx Context, fn interface{}) (Future, Settable) {
	impl := &decodeFutureImpl{
		&futureImpl{channel: NewChannel(ctx).(*channelImpl)}, fn}
	return impl, impl
}

// setQueryHandler sets query handler for given queryType.
func setQueryHandler(ctx Context, queryType string, handler interface{}, options QueryHandlerOptions) error {
	qh := &queryHandler{
		fn:            handler,
		queryType:     queryType,
		dataConverter: getDataConverterFromWorkflowContext(ctx),
		options:       options,
	}
	err := validateQueryHandlerFn(qh.fn)
	if err != nil {
		return err
	}

	getWorkflowEnvOptions(ctx).queryHandlers[queryType] = qh
	return nil
}

// setUpdateHandler sets update handler for a given update name.
func setUpdateHandler(ctx Context, updateName string, handler interface{}, opts UpdateHandlerOptions) error {
	uh, err := newUpdateHandler(updateName, handler, opts)
	if err != nil {
		return err
	}
	getWorkflowEnvOptions(ctx).updateHandlers[updateName] = uh
	if getWorkflowEnvironment(ctx).TryUse(SDKPriorityUpdateHandling) {
		getWorkflowEnvironment(ctx).HandleQueuedUpdates(updateName)
		state := getState(ctx)
		defer state.unblocked()
		state.yield("letting any updates waiting on a handler run")
	}
	return nil
}

// validateEquivalentParams verifies that both arguments are functions and that
// said functions take the exact same parameter types in the same order but not
// considering the presence or absence of a workflow.Context parameter in the
// zeroth position.
func validateEquivalentParams(fn1, fn2 interface{}) error {
	fn1Type := reflect.TypeOf(fn1)
	fn2Type := reflect.TypeOf(fn2)

	if fn1Type.Kind() != reflect.Func {
		return fmt.Errorf("type must be function but was %s", fn1Type.Kind())
	}

	if fn2Type.Kind() != reflect.Func {
		return fmt.Errorf("type must be function but was %s", fn1Type.Kind())
	}

	ctxType := reflect.TypeOf(new(Context)).Elem()
	extractRelevantParamTypes := func(t reflect.Type) []reflect.Type {
		out := make([]reflect.Type, 0, t.NumIn())
		for i := 0; i < t.NumIn(); i++ {
			paramType := t.In(i)
			if i == 0 && paramType.Implements(ctxType) {
				// ignore the presence of a workflow.Context as a first param
				continue
			}
			out = append(out, paramType)
		}
		return out
	}

	fn1ParamTypes := extractRelevantParamTypes(fn1Type)
	fn2ParamTypes := extractRelevantParamTypes(fn2Type)

	if len(fn1ParamTypes) != len(fn2ParamTypes) {
		return errors.New("functions have different numbers of parameters")
	}

	for i := 0; i < len(fn1ParamTypes); i++ {
		fn1ParamType := fn1ParamTypes[i]
		fn2ParamType := fn2ParamTypes[i]
		if fn1ParamType != fn2ParamType {
			return fmt.Errorf("functions differ at parameter %v; %v != %v", i, fn1ParamType, fn2ParamType)
		}
	}
	return nil
}

func validateQueryHandlerFn(fn interface{}) error {
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("handler must be function but was %s", fnType.Kind())
	}

	if fnType.NumOut() != 2 {
		return fmt.Errorf(
			"handler must return 2 values (serializable result and error), but found %d return values", fnType.NumOut(),
		)
	}

	if !isValidResultType(fnType.Out(0)) {
		return fmt.Errorf(
			"first return value of handler must be serializable but found: %v", fnType.Out(0).Kind(),
		)
	}
	if !isError(fnType.Out(1)) {
		return fmt.Errorf(
			"second return value of handler must be error but found %v", fnType.Out(fnType.NumOut()-1).Kind(),
		)
	}
	return nil
}

func (h *queryHandler) execute(input []interface{}) (result interface{}, err error) {
	// if query handler panic, convert it to error
	defer func() {
		if p := recover(); p != nil {
			result = nil
			st := getStackTraceRaw("query handler [panic]:", 7, 0)
			if p == panicIllegalAccessCoroutineState {
				// query handler code try to access workflow functions outside of workflow context, make error message
				// more descriptive and clear.
				p = "query handler must not use temporal context to do things like workflow.NewChannel(), " +
					"workflow.Go() or to call any workflow blocking functions like Channel.Get() or Future.Get()"
			}
			err = fmt.Errorf("query handler panic: %v, stack trace: %v", p, st)
		}
	}()

	return executeFunction(h.fn, input)
}

// Add adds delta, which may be negative, to the WaitGroup counter.
// If the counter becomes zero, all goroutines blocked on Wait are released.
// If the counter goes negative, Add panics.
//
// Note that calls with a positive delta that occur when the counter is zero
// must happen before a Wait. Calls with a negative delta, or calls with a
// positive delta that start when the counter is greater than zero, may happen
// at any time.
// Typically this means the calls to Add should execute before the statement
// creating the goroutine or other event to be waited for.
// If a WaitGroup is reused to wait for several independent sets of events,
// new Add calls must happen after all previous Wait calls have returned.
//
// param delta int -> the value to increment the WaitGroup counter by
func (wg *waitGroupImpl) Add(delta int) {
	wg.n = wg.n + delta
	if wg.n < 0 {
		panic("negative WaitGroup counter")
	}
	if (wg.n > 0) || (!wg.waiting) {
		return
	}
	if wg.n == 0 {
		wg.settable.Set(false, nil)
	}
}

// Done decrements the WaitGroup counter by 1, indicating
// that a coroutine in the WaitGroup has completed
func (wg *waitGroupImpl) Done() {
	wg.Add(-1)
}

// Wait blocks and waits for specified number of coroutines to
// finish executing and then unblocks once the counter has reached 0.
//
// param ctx Context -> workflow context
func (wg *waitGroupImpl) Wait(ctx Context) {
	assertNotInReadOnlyState(ctx)
	if wg.n <= 0 {
		return
	}
	if wg.waiting {
		panic("WaitGroup is reused before previous Wait has returned")
	}

	wg.waiting = true
	if err := wg.future.Get(ctx, &wg.waiting); err != nil {
		panic(err)
	}
	wg.future, wg.settable = NewFuture(ctx)
}

func (wg *waitGroupImpl) Go(ctx Context, f func(Context)) {
	wg.Add(1)
	Go(ctx, func(ctx Context) {
		defer wg.Done()
		f(ctx)
	})
}

// Spawn starts a new coroutine with Dispatcher.NewCoroutine
func (us updateSchedulerImpl) Spawn(ctx Context, name string, highPriority bool, f func(Context)) Context {
	return us.dispatcher.NewCoroutine(ctx, name, highPriority, f)
}

// Yield calls the yield function on the coroutineState associated with the
// supplied workflow context.
func (us updateSchedulerImpl) Yield(ctx Context, reason string) {
	getState(ctx).yield(reason)
}

func (m *mutexImpl) Lock(ctx Context) error {
	err := Await(ctx, func() bool {
		return !m.locked
	})
	if err != nil {
		return err
	}
	m.locked = true
	return nil
}

func (m *mutexImpl) TryLock(ctx Context) bool {
	assertNotInReadOnlyState(ctx)
	if m.locked {
		return false
	}
	m.locked = true
	return true
}

func (m *mutexImpl) Unlock() {
	if !m.locked {
		panic("Mutex.Unlock() was called on an unlocked mutex")
	}
	m.locked = false
}

func (m *mutexImpl) IsLocked() bool {
	return m.locked
}

func (s *semaphoreImpl) Acquire(ctx Context, n int64) error {
	err := Await(ctx, func() bool {
		return s.size-s.cur >= n
	})
	if err != nil {
		return err
	}
	s.cur += n
	return nil
}

func (s *semaphoreImpl) TryAcquire(ctx Context, n int64) bool {
	assertNotInReadOnlyState(ctx)
	success := s.size-s.cur >= n
	if success {
		s.cur += n
	}
	return success
}

func (s *semaphoreImpl) Release(n int64) {
	s.cur -= n
	if s.cur < 0 {
		panic("Semaphore.Release() released more than held")
	}
}

func incrementWorkflowTaskFailureCounter(metricsHandler metrics.Handler, failureReason string) {
	metricsHandler.WithTags(metrics.WorkflowTaskFailedTags(failureReason)).Counter(metrics.WorkflowTaskExecutionFailureCounter).Inc(1)
}
