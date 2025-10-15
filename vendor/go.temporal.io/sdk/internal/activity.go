package internal

import (
	"context"
	"fmt"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/log"
)

type (
	// ActivityType identifies an activity type.
	//
	// Exposed as: [go.temporal.io/sdk/activity.Type]
	ActivityType struct {
		Name string
	}

	// ActivityInfo contains information about a currently executing activity.
	//
	// Exposed as: [go.temporal.io/sdk/activity.Info]
	ActivityInfo struct {
		TaskToken              []byte
		WorkflowType           *WorkflowType
		WorkflowNamespace      string
		WorkflowExecution      WorkflowExecution
		ActivityID             string
		ActivityType           ActivityType
		TaskQueue              string
		HeartbeatTimeout       time.Duration // Maximum time between heartbeats. 0 means no heartbeat needed.
		ScheduleToCloseTimeout time.Duration // Schedule to close timeout set by the activity options.
		StartToCloseTimeout    time.Duration // Start to close timeout set by the activity options.
		ScheduledTime          time.Time     // Time of activity scheduled by a workflow
		StartedTime            time.Time     // Time of activity start
		Deadline               time.Time     // Time of activity timeout
		Attempt                int32         // Attempt starts from 1, and increased by 1 for every retry if retry policy is specified.
		IsLocalActivity        bool          // true if it is a local activity
		// Priority settings that control relative ordering of task processing when activity tasks are backed up in a queue.
		// If no priority is set, the default value is the zero value.
		//
		// WARNING: Task queue priority is currently experimental.
		Priority Priority
	}

	// RegisterActivityOptions consists of options for registering an activity.
	//
	// Exposed as: [go.temporal.io/sdk/activity.RegisterOptions]
	RegisterActivityOptions struct {
		// When an activity is a function the name is an actual activity type name.
		// When an activity is part of a structure then each member of the structure becomes an activity with
		// this Name as a prefix + activity function name.
		//
		// If this is set, users are strongly recommended to set
		// worker.Options.DisableRegistrationAliasing at the worker level to prevent
		// ambiguity between string names and function references. Also users should
		// always use this string name when executing this activity.
		Name                          string
		DisableAlreadyRegisteredCheck bool

		// When registering a struct with activities, skip functions that are not valid activities. If false,
		// registration panics.
		SkipInvalidStructFunctions bool
	}

	// ActivityOptions stores all activity-specific parameters that will be stored inside of a context.
	// The current timeout resolution implementation is in seconds and uses math.Ceil(d.Seconds()) as the duration. But is
	// subjected to change in the future.
	//
	// Exposed as: [go.temporal.io/sdk/workflow.ActivityOptions]
	ActivityOptions struct {
		// TaskQueue - Name of the task queue that the activity needs to be scheduled on.
		//
		// Optional: The default task queue with the same name as the workflow task queue.
		TaskQueue string

		// ScheduleToCloseTimeout - Total time that a workflow is willing to wait for an Activity to complete.
		// ScheduleToCloseTimeout limits the total time of an Activity's execution including retries
		// 		(use StartToCloseTimeout to limit the time of a single attempt).
		// The zero value of this uses default value.
		// Either this option or StartToCloseTimeout is required: Defaults to unlimited.
		ScheduleToCloseTimeout time.Duration

		// ScheduleToStartTimeout - Time that the Activity Task can stay in the Task Queue before it is picked up by
		// a Worker. Do not specify this timeout unless using host specific Task Queues for Activity Tasks are being
		// used for routing. In almost all situations that don't involve routing activities to specific hosts, it is
		// better to rely on the default value.
		// ScheduleToStartTimeout is always non-retryable. Retrying after this timeout doesn't make sense, as it would
		// just put the Activity Task back into the same Task Queue.
		//
		// Optional: Defaults to unlimited.
		ScheduleToStartTimeout time.Duration

		// StartToCloseTimeout - Maximum time of a single Activity execution attempt.
		// Note that the Temporal Server doesn't detect Worker process failures directly. It relies on this timeout
		// to detect that an Activity that didn't complete on time. So this timeout should be as short as the longest
		// possible execution of the Activity body. Potentially long running Activities must specify HeartbeatTimeout
		// and call Activity.RecordHeartbeat(ctx, "my-heartbeat") periodically for timely failure detection.
		// Either this option or ScheduleToCloseTimeout is required: Defaults to the ScheduleToCloseTimeout value.
		StartToCloseTimeout time.Duration

		// HeartbeatTimeout - Heartbeat interval. Activity must call Activity.RecordHeartbeat(ctx, "my-heartbeat")
		// before this interval passes after the last heartbeat or the Activity starts.
		HeartbeatTimeout time.Duration

		// WaitForCancellation - Whether to wait for canceled activity to be completed(
		// activity can be failed, completed, cancel accepted)
		//
		// Optional: default false
		WaitForCancellation bool

		// ActivityID - Business level activity ID, this is not needed for most of the cases if you have
		// to specify this then talk to the temporal team. This is something will be done in the future.
		//
		// Optional: default empty string
		ActivityID string

		// RetryPolicy - Specifies how to retry an Activity if an error occurs.
		// More details are available at docs.temporal.io.
		// RetryPolicy is optional. If one is not specified, a default RetryPolicy is provided by the server.
		// The default RetryPolicy provided by the server specifies:
		//  - InitialInterval of 1 second
		//  - BackoffCoefficient of 2.0
		//  - MaximumInterval of 100 x InitialInterval
		//  - MaximumAttempts of 0 (unlimited)
		// To disable retries, set MaximumAttempts to 1.
		// The default RetryPolicy provided by the server can be overridden by the dynamic config.
		RetryPolicy *RetryPolicy

		// If true, eager execution will not be requested, regardless of worker settings.
		// If false, eager execution may still be disabled at the worker level or
		// may not be requested due to lack of available slots.
		//
		// Eager activity execution means the server returns requested eager
		// activities directly from the workflow task back to this worker. This is
		// faster than non-eager, which may be dispatched to a separate worker.
		DisableEagerExecution bool

		// VersioningIntent - Specifies whether this activity should run on a worker with a compatible
		// build ID or not. See temporal.VersioningIntent.
		// WARNING: Worker versioning is currently experimental
		VersioningIntent VersioningIntent

		// Summary is a single-line summary for this activity that will appear in UI/CLI. This can be
		// in single-line Temporal Markdown format.
		//
		// Optional: defaults to none/empty.
		//
		// NOTE: Experimental
		Summary string

		// Priority - Optional priority settings that control relative ordering of
		// task processing when tasks are backed up in a queue.
		//
		// WARNING: Task queue priority is currently experimental.
		Priority Priority
	}

	// LocalActivityOptions stores local activity specific parameters that will be stored inside of a context.
	//
	// Exposed as: [go.temporal.io/sdk/workflow.LocalActivityOptions]
	LocalActivityOptions struct {
		// ScheduleToCloseTimeout - The end to end timeout for the local activity, including retries.
		// At least one of ScheduleToCloseTimeout or StartToCloseTimeout is required.
		// Defaults to StartToCloseTimeout if not set.
		ScheduleToCloseTimeout time.Duration

		// StartToCloseTimeout - The timeout for a single execution of the local activity.
		// At least one of ScheduleToCloseTimeout or StartToCloseTimeout is required.
		// Defaults to ScheduleToCloseTimeout if not set.
		StartToCloseTimeout time.Duration

		// RetryPolicy - Specify how to retry activity if error happens.
		//
		// Optional: default is to retry according to the default retry policy up to ScheduleToCloseTimeout
		// with 1sec initial delay between retries and 2x backoff.
		RetryPolicy *RetryPolicy

		// Summary is a single-line summary for this activity that will appear in UI/CLI. This can be
		// in single-line Temporal Markdown format.
		//
		// Optional: defaults to none/empty.
		//
		// NOTE: Experimental
		Summary string
	}
)

// GetActivityInfo returns information about the currently executing activity.
//
// Exposed as: [go.temporal.io/sdk/activity.GetInfo]
func GetActivityInfo(ctx context.Context) ActivityInfo {
	return getActivityOutboundInterceptor(ctx).GetInfo(ctx)
}

// HasHeartbeatDetails checks if there are heartbeat details from last attempt.
//
// Exposed as: [go.temporal.io/sdk/activity.HasHeartbeatDetails]
func HasHeartbeatDetails(ctx context.Context) bool {
	return getActivityOutboundInterceptor(ctx).HasHeartbeatDetails(ctx)
}

// IsActivity checks if the context is an activity context from a normal or local activity.
//
// Exposed as: [go.temporal.io/sdk/activity.IsActivity]
func IsActivity(ctx context.Context) bool {
	a := ctx.Value(activityInterceptorContextKey)
	return a != nil
}

// GetHeartbeatDetails extracts heartbeat details from the last failed attempt. This is used in combination with the retry policy.
// An activity could be scheduled with an optional retry policy on ActivityOptions. If the activity failed, then server
// would attempt to dispatch another activity task to retry according to the retry policy. If there were heartbeat
// details reported by activity from the failed attempt, the details would be delivered along with the activity task for
// the retry attempt. An activity can extract the details from GetHeartbeatDetails() and resume progress from there.
//
// Note: Values should not be reused for extraction here because merging on top
// of existing values may result in unexpected behavior similar to json.Unmarshal.
//
// Exposed as: [go.temporal.io/sdk/activity.GetHeartbeatDetails]
func GetHeartbeatDetails(ctx context.Context, d ...interface{}) error {
	return getActivityOutboundInterceptor(ctx).GetHeartbeatDetails(ctx, d...)
}

// GetActivityLogger returns a logger that can be used in the activity.
//
// Exposed as: [go.temporal.io/sdk/activity.GetLogger]
func GetActivityLogger(ctx context.Context) log.Logger {
	return getActivityOutboundInterceptor(ctx).GetLogger(ctx)
}

// GetActivityMetricsHandler returns a metrics handler that can be used in the activity.
//
// Exposed as: [go.temporal.io/sdk/activity.GetMetricsHandler]
func GetActivityMetricsHandler(ctx context.Context) metrics.Handler {
	return getActivityOutboundInterceptor(ctx).GetMetricsHandler(ctx)
}

// GetWorkerStopChannel returns a read-only channel. The closure of this channel indicates the activity worker is stopping.
// When the worker is stopping, it will close this channel and wait until the worker stop timeout finishes. After the timeout
// hits, the worker will cancel the activity context and then exit. The timeout can be defined by worker option: WorkerStopTimeout.
// Use this channel to handle a graceful activity exit when the activity worker stops.
//
// Exposed as: [go.temporal.io/sdk/activity.GetWorkerStopChannel]
func GetWorkerStopChannel(ctx context.Context) <-chan struct{} {
	return getActivityOutboundInterceptor(ctx).GetWorkerStopChannel(ctx)
}

// RecordActivityHeartbeat sends a heartbeat for the currently executing activity.
// If the activity is either canceled or workflow/activity doesn't exist, then we would cancel
// the context with error context.Canceled.
//
//	TODO: Implement automatic heartbeating with cancellation through ctx.
//
// details - The details that you provided here can be seen in the workflow when it receives TimeoutError. You
// can check error TimeoutType()/Details().
//
// Exposed as: [go.temporal.io/sdk/activity.RecordHeartbeat]
func RecordActivityHeartbeat(ctx context.Context, details ...interface{}) {
	getActivityOutboundInterceptor(ctx).RecordHeartbeat(ctx, details...)
}

// GetClient returns a client that can be used to interact with the Temporal
// service from an activity.
//
// Exposed as: [go.temporal.io/sdk/activity.GetClient]
func GetClient(ctx context.Context) Client {
	return getActivityOutboundInterceptor(ctx).GetClient(ctx)
}

// ServiceInvoker abstracts calls to the Temporal service from an activity implementation.
// Implement to unit test activities.
type ServiceInvoker interface {
	// Returns ActivityTaskCanceledError if activity is canceled
	Heartbeat(ctx context.Context, details *commonpb.Payloads, skipBatching bool) error
	Close(ctx context.Context, flushBufferedHeartbeat bool)
	GetClient(options ClientOptions) Client
}

// WithActivityTask adds activity specific information into context.
// Use this method to unit test activity implementations that use context extractor methodshared.
func WithActivityTask(
	ctx context.Context,
	task *workflowservice.PollActivityTaskQueueResponse,
	taskQueue string,
	invoker ServiceInvoker,
	logger log.Logger,
	metricsHandler metrics.Handler,
	dataConverter converter.DataConverter,
	workerStopChannel <-chan struct{},
	contextPropagators []ContextPropagator,
	interceptors []WorkerInterceptor,
	client *WorkflowClient,
) (context.Context, error) {
	scheduled := task.GetScheduledTime().AsTime()
	started := task.GetStartedTime().AsTime()
	scheduleToCloseTimeout := task.GetScheduleToCloseTimeout().AsDuration()
	startToCloseTimeout := task.GetStartToCloseTimeout().AsDuration()
	heartbeatTimeout := task.GetHeartbeatTimeout().AsDuration()
	deadline := calculateActivityDeadline(scheduled, scheduleToCloseTimeout, startToCloseTimeout)

	logger = log.With(logger,
		tagActivityID, task.ActivityId,
		tagActivityType, task.ActivityType.GetName(),
		tagAttempt, task.Attempt,
		tagWorkflowType, task.WorkflowType.GetName(),
		tagWorkflowID, task.WorkflowExecution.WorkflowId,
		tagRunID, task.WorkflowExecution.RunId,
	)

	return newActivityContext(ctx, interceptors, &activityEnvironment{
		taskToken:      task.TaskToken,
		serviceInvoker: invoker,
		activityType:   ActivityType{Name: task.ActivityType.GetName()},
		activityID:     task.ActivityId,
		workflowExecution: WorkflowExecution{
			RunID: task.WorkflowExecution.RunId,
			ID:    task.WorkflowExecution.WorkflowId},
		logger:                 logger,
		metricsHandler:         metricsHandler,
		deadline:               deadline,
		heartbeatTimeout:       heartbeatTimeout,
		scheduleToCloseTimeout: scheduleToCloseTimeout,
		startToCloseTimeout:    startToCloseTimeout,
		scheduledTime:          scheduled,
		startedTime:            started,
		taskQueue:              taskQueue,
		dataConverter:          dataConverter,
		attempt:                task.GetAttempt(),
		priority:               task.GetPriority(),
		heartbeatDetails:       task.HeartbeatDetails,
		workflowType: &WorkflowType{
			Name: task.WorkflowType.GetName(),
		},
		workflowNamespace:  task.WorkflowNamespace,
		workerStopChannel:  workerStopChannel,
		contextPropagators: contextPropagators,
		client:             client,
	})
}

// WithLocalActivityTask adds local activity specific information into context.
func WithLocalActivityTask(
	ctx context.Context,
	task *localActivityTask,
	logger log.Logger,
	metricsHandler metrics.Handler,
	dataConverter converter.DataConverter,
	interceptors []WorkerInterceptor,
	client *WorkflowClient,
	workerStopChannel <-chan struct{},
) (context.Context, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	workflowTypeLocal := task.params.WorkflowInfo.WorkflowType
	workflowType := task.params.WorkflowInfo.WorkflowType.Name
	activityType := task.params.ActivityType
	logger = log.With(logger,
		tagActivityID, task.activityID,
		tagActivityType, activityType,
		tagAttempt, task.attempt,
		tagWorkflowType, workflowType,
		tagWorkflowID, task.params.WorkflowInfo.WorkflowExecution.ID,
		tagRunID, task.params.WorkflowInfo.WorkflowExecution.RunID,
	)
	startedTime := time.Now()
	scheduleToCloseTimeout := task.params.ScheduleToCloseTimeout
	startToCloseTimeout := task.params.StartToCloseTimeout

	if startToCloseTimeout == 0 {
		startToCloseTimeout = scheduleToCloseTimeout
	}
	if scheduleToCloseTimeout == 0 {
		scheduleToCloseTimeout = startToCloseTimeout
	}
	deadline := calculateActivityDeadline(task.scheduledTime, scheduleToCloseTimeout, startToCloseTimeout)
	if task.attempt > 1 && !task.expireTime.IsZero() && task.expireTime.Before(deadline) {
		// this is attempt and expire time is before SCHEDULE_TO_CLOSE timeout
		deadline = task.expireTime
	}
	return newActivityContext(ctx, interceptors, &activityEnvironment{
		workflowType:           &workflowTypeLocal,
		workflowNamespace:      task.params.WorkflowInfo.Namespace,
		taskQueue:              task.params.WorkflowInfo.TaskQueueName,
		activityType:           ActivityType{Name: activityType},
		activityID:             fmt.Sprintf("%v", task.activityID),
		workflowExecution:      task.params.WorkflowInfo.WorkflowExecution,
		logger:                 logger,
		metricsHandler:         metricsHandler,
		scheduleToCloseTimeout: scheduleToCloseTimeout,
		startToCloseTimeout:    startToCloseTimeout,
		isLocalActivity:        true,
		deadline:               deadline,
		scheduledTime:          task.scheduledTime,
		startedTime:            startedTime,
		dataConverter:          dataConverter,
		attempt:                task.attempt,
		client:                 client,
		workerStopChannel:      workerStopChannel,
	})
}

func newActivityContext(
	ctx context.Context,
	interceptors []WorkerInterceptor,
	env *activityEnvironment,
) (context.Context, error) {
	ctx = context.WithValue(ctx, activityEnvContextKey, env)

	// Create interceptor with default inbound and outbound values and put on
	// context
	envInterceptor := &activityEnvironmentInterceptor{env: env}
	envInterceptor.inboundInterceptor = envInterceptor
	envInterceptor.outboundInterceptor = envInterceptor
	ctx = context.WithValue(ctx, activityEnvInterceptorContextKey, envInterceptor)
	ctx = context.WithValue(ctx, activityInterceptorContextKey, envInterceptor.outboundInterceptor)

	// Intercept, run init, and put the new outbound interceptor on the context
	for i := len(interceptors) - 1; i >= 0; i-- {
		envInterceptor.inboundInterceptor = interceptors[i].InterceptActivity(ctx, envInterceptor.inboundInterceptor)
	}
	err := envInterceptor.inboundInterceptor.Init(envInterceptor)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, activityInterceptorContextKey, envInterceptor.outboundInterceptor)

	return ctx, nil
}

func calculateActivityDeadline(scheduled time.Time, scheduleToCloseTimeout, startToCloseTimeout time.Duration) time.Time {
	startToCloseDeadline := time.Now().Add(startToCloseTimeout)
	if scheduleToCloseTimeout > 0 {
		scheduleToCloseDeadline := scheduled.Add(scheduleToCloseTimeout)
		// Minimum of the two deadlines.
		if scheduleToCloseDeadline.Before(startToCloseDeadline) {
			return scheduleToCloseDeadline
		}
	}
	return startToCloseDeadline
}
