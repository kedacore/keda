package internal

// All code in this file is private to the package.

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"google.golang.org/protobuf/proto"

	commonpb "go.temporal.io/api/common/v1"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/log"
)

type (
	// activity is an interface of an activity implementation.
	activity interface {
		Execute(ctx context.Context, input *commonpb.Payloads) (*commonpb.Payloads, error)
		ActivityType() ActivityType
		GetFunction() interface{}
	}

	// ActivityID uniquely identifies an activity execution
	ActivityID struct {
		id string
	}

	// LocalActivityID uniquely identifies a local activity execution
	LocalActivityID struct {
		id string
	}

	// ExecuteActivityOptions option for executing an activity
	ExecuteActivityOptions struct {
		ActivityID             string // Users can choose IDs but our framework makes it optional to decrease the crust.
		TaskQueueName          string
		ScheduleToCloseTimeout time.Duration
		ScheduleToStartTimeout time.Duration
		StartToCloseTimeout    time.Duration
		HeartbeatTimeout       time.Duration
		WaitForCancellation    bool
		OriginalTaskQueueName  string
		RetryPolicy            *commonpb.RetryPolicy
		DisableEagerExecution  bool
		VersioningIntent       VersioningIntent
		Summary                string
		Priority               *commonpb.Priority
	}

	// ExecuteLocalActivityOptions options for executing a local activity
	ExecuteLocalActivityOptions struct {
		ScheduleToCloseTimeout time.Duration
		StartToCloseTimeout    time.Duration
		RetryPolicy            *RetryPolicy
		Summary                string
	}

	// ExecuteActivityParams parameters for executing an activity
	ExecuteActivityParams struct {
		ExecuteActivityOptions
		ActivityType  ActivityType
		Input         *commonpb.Payloads
		DataConverter converter.DataConverter
		Header        *commonpb.Header
	}

	// ExecuteLocalActivityParams parameters for executing a local activity
	ExecuteLocalActivityParams struct {
		ExecuteLocalActivityOptions
		ActivityFn    interface{} // local activity function pointer
		ActivityType  string      // local activity type
		InputArgs     []interface{}
		WorkflowInfo  *WorkflowInfo
		DataConverter converter.DataConverter
		Attempt       int32
		ScheduledTime time.Time
		Header        *commonpb.Header
	}

	// AsyncActivityClient for requesting activity execution
	AsyncActivityClient interface {
		// The ExecuteActivity schedules an activity with a callback handler.
		// If the activity failed to complete the callback error would indicate the failure
		// and it can be one of ActivityTaskFailedError, ActivityTaskTimeoutError, ActivityTaskCanceledError
		ExecuteActivity(parameters ExecuteActivityParams, callback ResultHandler) ActivityID

		// This only initiates cancel request for activity. if the activity is configured to not WaitForCancellation then
		// it would invoke the callback handler immediately with error code ActivityTaskCanceledError.
		// If the activity is not running(either scheduled or started) then it is a no-operation.
		RequestCancelActivity(activityID ActivityID)
	}

	// LocalActivityClient for requesting local activity execution
	LocalActivityClient interface {
		ExecuteLocalActivity(params ExecuteLocalActivityParams, callback LocalActivityResultHandler) LocalActivityID

		RequestCancelLocalActivity(activityID LocalActivityID)
	}

	activityEnvironment struct {
		taskToken              []byte
		workflowExecution      WorkflowExecution
		activityID             string
		activityType           ActivityType
		serviceInvoker         ServiceInvoker
		logger                 log.Logger
		metricsHandler         metrics.Handler
		isLocalActivity        bool
		heartbeatTimeout       time.Duration
		scheduleToCloseTimeout time.Duration
		startToCloseTimeout    time.Duration
		deadline               time.Time
		scheduledTime          time.Time
		startedTime            time.Time
		taskQueue              string
		dataConverter          converter.DataConverter
		attempt                int32 // starts from 1.
		heartbeatDetails       *commonpb.Payloads
		workflowType           *WorkflowType
		workflowNamespace      string
		workerStopChannel      <-chan struct{}
		contextPropagators     []ContextPropagator
		client                 *WorkflowClient
		priority               *commonpb.Priority
		retryPolicy            *RetryPolicy
	}

	// context.WithValue need this type instead of basic type string to avoid lint error
	contextKey string
)

const (
	activityEnvContextKey            contextKey = "activityEnv"
	activityOptionsContextKey        contextKey = "activityOptions"
	localActivityOptionsContextKey   contextKey = "localActivityOptions"
	activityInterceptorContextKey    contextKey = "activityInterceptor"
	activityEnvInterceptorContextKey contextKey = "activityEnvInterceptor"
)

func (i ActivityID) String() string {
	return i.id
}

// ParseActivityID returns ActivityID constructed from its string representation.
// The string representation should be obtained through ActivityID.String()
func ParseActivityID(id string) (ActivityID, error) {
	return ActivityID{id: id}, nil
}

func (i LocalActivityID) String() string {
	return i.id
}

// ParseLocalActivityID returns LocalActivityID constructed from its string representation.
// The string representation should be obtained through LocalActivityID.String()
func ParseLocalActivityID(v string) (LocalActivityID, error) {
	return LocalActivityID{id: v}, nil
}

func getActivityEnv(ctx context.Context) *activityEnvironment {
	env := ctx.Value(activityEnvContextKey)
	if env == nil {
		panic("getActivityEnv: Not an activity context")
	}
	return env.(*activityEnvironment)
}

func getActivityOptions(ctx Context) *ExecuteActivityOptions {
	eap := ctx.Value(activityOptionsContextKey)
	if eap == nil {
		return nil
	}
	return eap.(*ExecuteActivityOptions)
}

func getLocalActivityOptions(ctx Context) *ExecuteLocalActivityOptions {
	opts := ctx.Value(localActivityOptionsContextKey)
	if opts == nil {
		return nil
	}
	return opts.(*ExecuteLocalActivityOptions)
}

func getValidatedLocalActivityOptions(ctx Context) (*ExecuteLocalActivityOptions, error) {
	p := getLocalActivityOptions(ctx)
	if p == nil {
		return nil, errLocalActivityParamsBadRequest
	}
	if p.ScheduleToCloseTimeout < 0 {
		return nil, errors.New("negative ScheduleToCloseTimeout")
	}
	if p.StartToCloseTimeout < 0 {
		return nil, errors.New("negative StartToCloseTimeout")
	}
	if p.ScheduleToCloseTimeout == 0 && p.StartToCloseTimeout == 0 {
		return nil, errors.New("at least one of ScheduleToCloseTimeout and StartToCloseTimeout is required")
	}
	if p.ScheduleToCloseTimeout == 0 {
		p.ScheduleToCloseTimeout = p.StartToCloseTimeout
	}
	if p.StartToCloseTimeout == 0 {
		p.StartToCloseTimeout = p.ScheduleToCloseTimeout
	}
	return p, nil
}

func validateFunctionArgs(workflowFunc interface{}, args []interface{}, isWorkflow bool) error {
	fType := reflect.TypeOf(workflowFunc)
	switch getKind(fType) {
	case reflect.String:
		// We can't validate function passed as string.
		return nil
	case reflect.Func:
	default:
		return fmt.Errorf(
			"invalid type 'workflowFunc' parameter provided, it can be either worker function or function name: %v",
			workflowFunc)
	}

	fnName, _ := getFunctionName(workflowFunc)
	fnArgIndex := 0
	// Skip Context function argument.
	if fType.NumIn() > 0 {
		if isWorkflow && isWorkflowContext(fType.In(0)) {
			fnArgIndex++
		}
		if !isWorkflow && isActivityContext(fType.In(0)) {
			fnArgIndex++
		}
	}

	// Validate provided args match with function order match.
	if fType.NumIn()-fnArgIndex != len(args) {
		return fmt.Errorf(
			"expected %d args for function: %v but found %v",
			fType.NumIn()-fnArgIndex, fnName, len(args))
	}

	for i := 0; fnArgIndex < fType.NumIn(); fnArgIndex, i = fnArgIndex+1, i+1 {
		fnArgType := fType.In(fnArgIndex)
		argType := reflect.TypeOf(args[i])
		if argType != nil && !argType.AssignableTo(fnArgType) {
			return fmt.Errorf(
				"cannot assign function argument: %d from type: %s to type: %s",
				fnArgIndex+1, argType, fnArgType,
			)
		}
	}

	return nil
}

func getValidatedActivityFunction(f interface{}, args []interface{}, registry *registry) (*ActivityType, error) {
	fnName := ""
	fType := reflect.TypeOf(f)
	switch getKind(fType) {
	case reflect.String:
		fnName = reflect.ValueOf(f).String()
	case reflect.Func:
		if err := validateFunctionArgs(f, args, false); err != nil {
			return nil, err
		}
		fnName, _ = getFunctionName(f)
		if alias, ok := registry.getActivityAlias(fnName); ok {
			fnName = alias
		}

	default:
		return nil, fmt.Errorf(
			"invalid type 'f' parameter provided, it can be either activity function or name of the activity: %v", f)
	}

	return &ActivityType{Name: fnName}, nil
}

func getKind(fType reflect.Type) reflect.Kind {
	if fType == nil {
		return reflect.Invalid
	}
	return fType.Kind()
}

func isActivityContext(inType reflect.Type) bool {
	contextElem := reflect.TypeOf((*context.Context)(nil)).Elem()
	return inType != nil && inType.Implements(contextElem)
}

func setActivityParametersIfNotExist(ctx Context) Context {
	params := getActivityOptions(ctx)
	var newParams ExecuteActivityOptions
	if params != nil {
		newParams = *params
		if params.RetryPolicy != nil {
			newParams.RetryPolicy = proto.Clone(params.RetryPolicy).(*commonpb.RetryPolicy)
		}
	}
	return WithValue(ctx, activityOptionsContextKey, &newParams)
}

func setLocalActivityParametersIfNotExist(ctx Context) Context {
	params := getLocalActivityOptions(ctx)
	var newParams ExecuteLocalActivityOptions
	if params != nil {
		newParams = *params
	}
	return WithValue(ctx, localActivityOptionsContextKey, &newParams)
}

type activityEnvironmentInterceptor struct {
	env                 *activityEnvironment
	inboundInterceptor  ActivityInboundInterceptor
	outboundInterceptor ActivityOutboundInterceptor
	fn                  interface{}
}

func getActivityEnvironmentInterceptor(ctx context.Context) *activityEnvironmentInterceptor {
	a := ctx.Value(activityEnvInterceptorContextKey)
	if a == nil {
		panic("getActivityEnvironmentInterceptor: Not an activity context")
	}
	return a.(*activityEnvironmentInterceptor)
}

func getActivityOutboundInterceptor(ctx context.Context) ActivityOutboundInterceptor {
	a := ctx.Value(activityInterceptorContextKey)
	if a == nil {
		panic("getActivityOutboundInterceptor: Not an activity context")
	}
	return a.(ActivityOutboundInterceptor)
}

func (a *activityEnvironmentInterceptor) Init(outbound ActivityOutboundInterceptor) error {
	a.outboundInterceptor = outbound
	return nil
}

func (a *activityEnvironmentInterceptor) ExecuteActivity(
	ctx context.Context,
	in *ExecuteActivityInput,
) (interface{}, error) {
	// Remove header from context
	ctx = contextWithoutHeader(ctx)

	return executeFunctionWithContext(ctx, a.fn, in.Args)
}

func (a *activityEnvironmentInterceptor) GetInfo(ctx context.Context) ActivityInfo {
	return ActivityInfo{
		ActivityID:             a.env.activityID,
		ActivityType:           a.env.activityType,
		TaskToken:              a.env.taskToken,
		WorkflowExecution:      a.env.workflowExecution,
		HeartbeatTimeout:       a.env.heartbeatTimeout,
		ScheduleToCloseTimeout: a.env.scheduleToCloseTimeout,
		StartToCloseTimeout:    a.env.startToCloseTimeout,
		Deadline:               a.env.deadline,
		ScheduledTime:          a.env.scheduledTime,
		StartedTime:            a.env.startedTime,
		TaskQueue:              a.env.taskQueue,
		Attempt:                a.env.attempt,
		WorkflowType:           a.env.workflowType,
		WorkflowNamespace:      a.env.workflowNamespace,
		IsLocalActivity:        a.env.isLocalActivity,
		Priority:               convertFromPBPriority(a.env.priority),
		RetryPolicy:            a.env.retryPolicy,
	}
}

func (a *activityEnvironmentInterceptor) GetLogger(ctx context.Context) log.Logger {
	return a.env.logger
}

func (a *activityEnvironmentInterceptor) GetMetricsHandler(ctx context.Context) metrics.Handler {
	return a.env.metricsHandler
}

func (a *activityEnvironmentInterceptor) RecordHeartbeat(ctx context.Context, details ...interface{}) {
	if a.env.isLocalActivity {
		// no-op for local activity
		return
	}
	var data *commonpb.Payloads
	var err error
	// We would like to be able to pass in "nil" as part of details(that is no progress to report to)
	if len(details) > 1 || (len(details) == 1 && details[0] != nil) {
		data, err = encodeArgs(getDataConverterFromActivityCtx(ctx), details)
		if err != nil {
			panic(err)
		}
	}

	// Heartbeat error is logged inside ServiceInvoker.internalHeartBeat
	_ = a.env.serviceInvoker.Heartbeat(ctx, data, false)
}

func (a *activityEnvironmentInterceptor) HasHeartbeatDetails(ctx context.Context) bool {
	return a.env.heartbeatDetails != nil
}

func (a *activityEnvironmentInterceptor) GetHeartbeatDetails(ctx context.Context, d ...interface{}) error {
	if a.env.heartbeatDetails == nil {
		return ErrNoData
	}
	encoded := newEncodedValues(a.env.heartbeatDetails, a.env.dataConverter)
	return encoded.Get(d...)
}

func (a *activityEnvironmentInterceptor) GetWorkerStopChannel(ctx context.Context) <-chan struct{} {
	return a.env.workerStopChannel
}

func (a *activityEnvironmentInterceptor) GetClient(ctx context.Context) Client {
	return a.env.client
}

// Needed so this can properly be considered an inbound interceptor
func (a *activityEnvironmentInterceptor) mustEmbedActivityInboundInterceptorBase() {}

// Needed so this can properly be considered an outbound interceptor
func (a *activityEnvironmentInterceptor) mustEmbedActivityOutboundInterceptorBase() {}
