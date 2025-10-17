package internal

import (
	"context"
	"time"

	deploymentpb "go.temporal.io/api/deployment/v1"
	enumspb "go.temporal.io/api/enums/v1"
)

var (
	_ PollerBehavior = (*pollerBehaviorSimpleMaximum)(nil)
	_ PollerBehavior = (*pollerBehaviorAutoscaling)(nil)
)

type (
	pollerBehaviorSimpleMaximum struct {
		// maximumNumberOfPollers is the maximum number of pollers the worker is allowed to start.
		maximumNumberOfPollers int
	}

	pollerBehaviorAutoscaling struct {
		// initialNumberOfPollers is the initial number of pollers to start.
		initialNumberOfPollers int
		// maximumNumberOfPollers is the maximum number of pollers the worker is allowed scale up to.
		maximumNumberOfPollers int
		// minimumNumberOfPollers is the minimum number of pollers the worker is allowed scale down to.
		minimumNumberOfPollers int
	}

	// PollerBehavior is used to configure the behavior of the poller.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/worker.PollerBehavior]
	PollerBehavior interface {
		isPollerBehavior()
	}

	// PollerBehaviorAutoscalingOptions is the options for NewPollerBehaviorAutoscaling.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/worker.PollerBehaviorAutoscalingOptions]
	PollerBehaviorAutoscalingOptions struct {
		// InitialNumberOfPollers is the initial number of pollers to start.
		//
		// Default: 5
		InitialNumberOfPollers int

		// MinimumNumberOfPollers is the minimum number of pollers the worker is allowed scale down to.
		//
		// Default: 1
		MinimumNumberOfPollers int

		// MaximumNumberOfPollers is the maximum number of pollers the worker is allowed scale up to.
		//
		// Default: 100
		MaximumNumberOfPollers int
	}

	// PollerBehaviorSimpleMaximumOptions is the options for NewPollerBehaviorSimpleMaximum.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/worker.PollerBehaviorSimpleMaximumOptions]
	PollerBehaviorSimpleMaximumOptions struct {
		// MaximumNumberOfPollers is the maximum number of pollers the worker is allowed
		// to start.
		//
		// Default: 2
		MaximumNumberOfPollers int
	}

	// WorkerDeploymentOptions provides configuration for Worker Deployment Versioning.
	//
	// NOTE: [WorkerDeploymentOptions.UseVersioning] must be set to enable Worker Deployment
	// Versioning.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/worker.DeploymentOptions]
	WorkerDeploymentOptions struct {
		// If set, opts this worker into the Worker Deployment Versioning feature. It will only
		// operate on workflows it claims to be compatible with. You must set [Version] if this flag
		// is true.
		//
		// NOTE: Experimental
		//
		// NOTE: Cannot be enabled at the same time as [WorkerOptions.EnableSessionWorker]
		UseVersioning bool

		// Assign a Deployment Version identifier to this worker. If [Version] is set
		// [WorkerOptions.BuildID] will be ignored.
		//
		// NOTE: Experimental
		Version WorkerDeploymentVersion

		// Optional: Provides a default Versioning Behavior to workflows that do not set one with
		// the registration option [RegisterWorkflowOptions.VersioningBehavior]. It is an error to
		// set this without [UseVersioning] being true.
		//
		// NOTE: When the new Deployment-based Worker Versioning feature is on, and
		// [DefaultVersioningBehavior] is unspecified, workflows that do not set the Versioning
		// Behavior will fail at registration time.
		//
		// NOTE: Experimental
		DefaultVersioningBehavior VersioningBehavior
	}

	// WorkerOptions is used to configure a worker instance.
	// The current timeout resolution implementation is in seconds and uses math.Ceil(d.Seconds()) as the duration. But is
	// subjected to change in the future.
	//
	// Exposed as: [go.temporal.io/sdk/worker.Options]
	WorkerOptions struct {
		// Optional: To set the maximum concurrent activity executions this worker can have.
		// The zero value of this uses the default value.
		//
		// default: defaultMaxConcurrentActivityExecutionSize(1k)
		MaxConcurrentActivityExecutionSize int

		// Optional: Sets the rate limiting on number of activities that can be executed per second per
		// worker. This can be used to limit resources used by the worker.
		// Notice that the number is represented in float, so that you can set it to less than
		// 1 if needed. For example, set the number to 0.1 means you want your activity to be executed
		// once for every 10 seconds. This can be used to protect down stream services from flooding.
		// The zero value of this uses the default value
		//
		// default: 100k
		WorkerActivitiesPerSecond float64

		// Optional: To set the maximum concurrent local activity executions this worker can have.
		// The zero value of this uses the default value.
		//
		// default: 1k
		MaxConcurrentLocalActivityExecutionSize int

		// Optional: Sets the rate limiting on number of local activities that can be executed per second per
		// worker. This can be used to limit resources used by the worker.
		// Notice that the number is represented in float, so that you can set it to less than
		// 1 if needed. For example, set the number to 0.1 means you want your local activity to be executed
		// once for every 10 seconds. This can be used to protect down stream services from flooding.
		// The zero value of this uses the default value
		//
		// default: 100k
		WorkerLocalActivitiesPerSecond float64

		// Optional: Sets the rate limiting on number of activities that can be executed per second.
		// This is managed by the server and controls activities per second for your entire taskqueue
		// whereas WorkerActivityTasksPerSecond controls activities only per worker.
		// Notice that the number is represented in float, so that you can set it to less than
		// 1 if needed. For example, set the number to 0.1 means you want your activity to be executed
		// once for every 10 seconds. This can be used to protect down stream services from flooding.
		// The zero value of this uses the default value.
		//
		// default: 100k
		//
		// NOTE: Setting this to a non zero value will also disable eager activities.
		TaskQueueActivitiesPerSecond float64

		// Optional: Sets the maximum number of goroutines that will concurrently poll the
		// temporal-server to retrieve activity tasks. Changing this value will affect the
		// rate at which the worker is able to consume tasks from a task queue.
		//
		// NOTE: This option is mutually exclusive with WorkflowTaskPollerBehavior.
		//
		// default: 2
		MaxConcurrentActivityTaskPollers int

		// Optional: To set the maximum concurrent workflow task executions this worker can have.
		// The zero value of this uses the default value. Due to internal logic where pollers
		// alternate between stick and non-sticky queues, this
		// value cannot be 1 and will panic if set to that value.
		//
		// default: defaultMaxConcurrentTaskExecutionSize(1k)
		MaxConcurrentWorkflowTaskExecutionSize int

		// Optional: Sets the maximum number of goroutines that will concurrently poll the
		// temporal-server to retrieve workflow tasks. Changing this value will affect the
		// rate at which the worker is able to consume tasks from a task queue. Due to
		// internal logic where pollers alternate between stick and non-sticky queues, this
		// value cannot be 1 and will panic if set to that value.
		//
		// NOTE: This option is mutually exclusive with WorkflowTaskPollerBehavior.
		//
		// default: 2
		MaxConcurrentWorkflowTaskPollers int

		// Optional: Sets the maximum concurrent nexus task executions this worker can have.
		// The zero value of this uses the default value.
		//
		// default: defaultMaxConcurrentTaskExecutionSize(1k)
		MaxConcurrentNexusTaskExecutionSize int

		// Optional: Sets the maximum number of goroutines that will concurrently poll the
		// temporal-server to retrieve nexus tasks. Changing this value will affect the
		// rate at which the worker is able to consume tasks from a task queue.
		//
		// NOTE: This option is mutually exclusive with NexusTaskPollerBehavior.
		//
		// default: 2
		MaxConcurrentNexusTaskPollers int

		// Optional: Enable logging in replay.
		// In the workflow code you can use workflow.GetLogger(ctx) to write logs. By default, the logger will skip log
		// entry during replay mode so you won't see duplicate logs. This option will enable the logging in replay mode.
		// This is only useful for debugging purpose.
		//
		// default: false
		EnableLoggingInReplay bool

		// Optional: Sticky schedule to start timeout.
		// The resolution is seconds.
		//
		// Sticky Execution is to run the workflow tasks for one workflow execution on same worker host. This is an
		// optimization for workflow execution. When sticky execution is enabled, worker keeps the workflow state in
		// memory. New workflow task contains the new history events will be dispatched to the same worker. If this
		// worker crashes, the sticky workflow task will timeout after StickyScheduleToStartTimeout, and temporal server
		// will clear the stickiness for that workflow execution and automatically reschedule a new workflow task that
		// is available for any worker to pick up and resume the progress.
		//
		// default: 5s
		StickyScheduleToStartTimeout time.Duration

		// Optional: sets root context for all activities. The context can be used to pass external dependencies
		// like DB connections to activity functions.
		// Note that this method of passing dependencies is not recommended anymore.
		// Instead, use a structure with fields that contain dependencies and activities
		// as the structure member functions. Then pass all the dependencies on the structure initialization.
		BackgroundActivityContext context.Context

		// Optional: Sets how workflow worker deals with non-deterministic history events
		// (presumably arising from non-deterministic workflow definitions or non-backward compatible workflow
		// definition changes) and other panics raised from workflow code.
		//
		// default: BlockWorkflow, which just logs error but doesn't fail workflow.
		WorkflowPanicPolicy WorkflowPanicPolicy

		// Optional: worker graceful stop timeout
		//
		// default: 0s
		WorkerStopTimeout time.Duration

		// Optional: Enable running session workers.
		// Session workers is for activities within a session.
		// Enable this option to allow worker to process sessions.
		//
		// default: false
		EnableSessionWorker bool

		// Uncomment this option when we support automatic restablish failed sessions.
		// Optional: The identifier of the resource consumed by sessions.
		// It's the user's responsibility to ensure there's only one worker using this resourceID.
		// For now, if user doesn't specify one, a new uuid will be used as the resourceID.
		// SessionResourceID string

		// Optional: Sets the maximum number of concurrently running sessions the resource supports.
		//
		// default: 1000
		MaxConcurrentSessionExecutionSize int

		// Optional: If set to true, a workflow worker is not started for this
		// worker and workflows cannot be registered with this worker. Use this if
		// you only want your worker to execute activities.
		//
		// default: false
		DisableWorkflowWorker bool

		// Optional: If set to true worker will only handle workflow tasks and local activities.
		// Non-local activities will not be executed by this worker.
		//
		// default: false
		LocalActivityWorkerOnly bool

		// Optional: If set overwrites the client level Identity value.
		//
		// default: client identity
		Identity string

		// Optional: If set defines maximum amount of time that workflow task will be allowed to run. Defaults to 1 sec.
		DeadlockDetectionTimeout time.Duration

		// Optional: The maximum amount of time between sending each pending heartbeat to the server. Regardless of
		// heartbeat timeout, no pending heartbeat will wait longer than this amount of time to send. To effectively disable
		// heartbeat throttling, this can be set to something like 1 nanosecond, but it is not recommended.
		//
		// default: 60 seconds
		MaxHeartbeatThrottleInterval time.Duration

		// Optional: The default amount of time between sending each pending heartbeat to the server. This is used if the
		// ActivityOptions do not provide a HeartbeatTimeout. Otherwise, the interval becomes a value a bit smaller than the
		// given HeartbeatTimeout.
		//
		// default: 30 seconds
		DefaultHeartbeatThrottleInterval time.Duration

		// Interceptors to apply to the worker. Earlier interceptors wrap later
		// interceptors.
		//
		// When worker interceptors are here and in client options, the ones in
		// client options wrap the ones here. The same interceptor should not be set
		// here and in client options.
		Interceptors []WorkerInterceptor

		// Optional: Callback invoked on fatal error. Immediately after this
		// returns, Worker.Stop() will be called.
		OnFatalError func(error)

		// Optional: Disable eager activities. If set to true, activities will not
		// be requested to execute eagerly from the same workflow regardless of
		// MaxConcurrentEagerActivityExecutionSize.
		//
		// Eager activity execution means the server returns requested eager
		// activities directly from the workflow task back to this worker which is
		// faster than non-eager which may be dispatched to a separate worker.
		//
		// NOTE: Eager activities will automatically be disabled if TaskQueueActivitiesPerSecond is set.
		DisableEagerActivities bool

		// Optional: Maximum number of eager activities that can be running.
		//
		// When non-zero, eager activity execution will not be requested for
		// activities schedule by the workflow if it would cause the total number of
		// running eager activities to exceed this value. For example, if this is
		// set to 1000 and there are already 998 eager activities executing and a
		// workflow task schedules 3 more, only the first 2 will request eager
		// execution.
		//
		// The default of 0 means unlimited and therefore only bound by
		// MaxConcurrentActivityExecutionSize.
		//
		// See DisableEagerActivities for a description of eager activity execution.
		MaxConcurrentEagerActivityExecutionSize int

		// Optional: Disable allowing workflow and activity functions that are
		// registered with custom names from being able to be called with their
		// function references.
		//
		// Users are strongly recommended to set this as true if they register any
		// workflow or activity functions with custom names. By leaving this as
		// false, the historical default, ambiguity can occur between function names
		// and aliased names when not using string names when executing child
		// workflow or activities.
		DisableRegistrationAliasing bool

		// Assign a BuildID to this worker. This replaces the deprecated binary checksum concept,
		// and is used to provide a unique identifier for a set of worker code, and is necessary
		// to opt in to the Worker Versioning feature. See [UseBuildIDForVersioning].
		//
		// Deprecated: Use [WorkerDeploymentOptions.Version]
		BuildID string

		// If set, opts this worker into the Worker Versioning feature. It will only
		// operate on workflows it claims to be compatible with. You must set BuildID if this flag
		// is true.
		//
		// Deprecated: Use [WorkerDeploymentOptions.UseVersioning]
		//
		// NOTE: Cannot be enabled at the same time as [WorkerOptions.EnableSessionWorker]
		UseBuildIDForVersioning bool

		// Optional: If set it configures Worker Versioning for this worker. See [WorkerDeploymentOptions]
		// for more.
		//
		// NOTE: Experimental
		DeploymentOptions WorkerDeploymentOptions

		// Optional: If set, use a custom tuner for this worker. See WorkerTuner for more.
		// Mutually exclusive with MaxConcurrentWorkflowTaskExecutionSize,
		// MaxConcurrentActivityExecutionSize, and MaxConcurrentLocalActivityExecutionSize.
		//
		// NOTE: Experimental
		Tuner WorkerTuner

		// Optional: If set, the worker will use the provided poller behavior when polling for workflow tasks.
		// This is mutually exclusive with MaxConcurrentWorkflowTaskPollers.
		//
		// NOTE: This option is mutually exclusive with MaxConcurrentWorkflowTaskPollers.
		//
		// NOTE: Experimental
		WorkflowTaskPollerBehavior PollerBehavior

		// Optional: If set, the worker will use the provided poller behavior when polling for activity tasks.
		// This is mutually exclusive with MaxConcurrentActivityTaskPollers.
		//
		// NOTE: This option is mutually exclusive with MaxConcurrentActivityTaskPollers.
		//
		// NOTE: Experimental
		ActivityTaskPollerBehavior PollerBehavior

		// Optional: If set, the worker will use the provided poller behavior when polling for nexus tasks.
		// This is mutually exclusive with MaxConcurrentNexusTaskPollers.
		//
		// NOTE: This option is mutually exclusive with MaxConcurrentNexusTaskPollers.
		//
		// NOTE: Experimental
		NexusTaskPollerBehavior PollerBehavior
	}
)

// WorkflowPanicPolicy is used for configuring how worker deals with workflow
// code panicking which includes non backwards compatible changes to the workflow code without appropriate
// versioning (see workflow.GetVersion).
// The default behavior is to block workflow execution until the problem is fixed.
//
// Exposed as: [go.temporal.io/sdk/worker.WorkflowPanicPolicy]
type WorkflowPanicPolicy int

const (
	// BlockWorkflow is the default policy for handling workflow panics and detected non-determinism.
	// This option causes workflow to get stuck in the workflow task retry loop.
	// It is expected that after the problem is discovered and fixed the workflows are going to continue
	// without any additional manual intervention.
	//
	// Exposed as: [go.temporal.io/sdk/worker.BlockWorkflow]
	BlockWorkflow WorkflowPanicPolicy = iota
	// FailWorkflow immediately fails workflow execution if workflow code throws panic or detects non-determinism.
	// This feature is convenient during development.
	// WARNING: enabling this in production can cause all open workflows to fail on a single bug or bad deployment.
	//
	// Exposed as: [go.temporal.io/sdk/worker.FailWorkflow]
	FailWorkflow
)

// ReplayNamespace is namespace for replay because startEvent doesn't contain it
const ReplayNamespace = "ReplayNamespace"

// IsReplayNamespace checks if the namespace is from replay
func IsReplayNamespace(dn string) bool {
	return ReplayNamespace == dn
}

// NewWorker creates an instance of worker for managing workflow and activity executions.
// client   - client created with client.Dial() or client.NewLazyClient().
// taskQueue - is the task queue name you use to identify your client worker,
// also identifies group of workflow and activity implementations that are
// hosted by a single worker process.
//
// options 	- configure any worker specific options.
//
// Exposed as: [go.temporal.io/sdk/worker.New]
func NewWorker(
	client Client,
	taskQueue string,
	options WorkerOptions,
) *AggregatedWorker {
	workflowClient, ok := client.(*WorkflowClient)
	if !ok {
		panic("Client must be created with client.Dial() or client.NewLazyClient()")
	}
	return NewAggregatedWorker(workflowClient, taskQueue, options)
}

func workerDeploymentOptionsToProto(useVersioning bool, version WorkerDeploymentVersion) *deploymentpb.WorkerDeploymentOptions {
	if (version != WorkerDeploymentVersion{}) {
		var workerVersioningMode enumspb.WorkerVersioningMode
		if useVersioning {
			workerVersioningMode = enumspb.WORKER_VERSIONING_MODE_VERSIONED
		} else {
			workerVersioningMode = enumspb.WORKER_VERSIONING_MODE_UNVERSIONED
		}
		return &deploymentpb.WorkerDeploymentOptions{
			DeploymentName:       version.DeploymentName,
			BuildId:              version.BuildId,
			WorkerVersioningMode: workerVersioningMode,
		}
	}
	return nil
}

// isPollerBehavior implements PollerBehavior.
func (p *pollerBehaviorSimpleMaximum) isPollerBehavior() {
}

// isPollerBehavior implements PollerBehavior.
func (p *pollerBehaviorAutoscaling) isPollerBehavior() {
}

// NewPollerBehaviorSimpleMaximum creates a PollerBehavior that allows the worker to start up to a maximum number of pollers.
//
// NOTE: Experimental
//
// Exposed as: [go.temporal.io/sdk/worker.NewPollerBehaviorSimpleMaximum]
func NewPollerBehaviorSimpleMaximum(
	options PollerBehaviorSimpleMaximumOptions,
) PollerBehavior {
	if options.MaximumNumberOfPollers <= 0 {
		options.MaximumNumberOfPollers = defaultConcurrentPollRoutineSize // Default maximum number of pollers.
	}
	return &pollerBehaviorSimpleMaximum{
		maximumNumberOfPollers: options.MaximumNumberOfPollers,
	}
}

// NewPollerBehaviorAutoscaling creates a PollerBehavior that allows the worker to scale the number of pollers within a given range.
// based on the workflow and feedback from the server.
//
// NOTE: Experimental
//
// Exposed as: [go.temporal.io/sdk/worker.NewPollerBehaviorAutoscaling]
func NewPollerBehaviorAutoscaling(
	options PollerBehaviorAutoscalingOptions,
) PollerBehavior {
	initialNumberOfPollers := options.InitialNumberOfPollers
	if initialNumberOfPollers <= 0 {
		initialNumberOfPollers = defaultAutoscalingInitialNumberOfPollers // Default initial number of pollers.
	}
	minimumNumberOfPollers := options.MinimumNumberOfPollers
	if minimumNumberOfPollers <= 0 {
		minimumNumberOfPollers = defaultAutoscalingMinimumNumberOfPollers // Default minimum number of pollers.
	}
	maximumNumberOfPollers := options.MaximumNumberOfPollers
	if maximumNumberOfPollers <= 0 {
		maximumNumberOfPollers = defaultAutoscalingMaximumNumberOfPollers // Default maximum number of pollers.
	}
	return &pollerBehaviorAutoscaling{
		initialNumberOfPollers: initialNumberOfPollers,
		minimumNumberOfPollers: minimumNumberOfPollers,
		maximumNumberOfPollers: maximumNumberOfPollers,
	}
}
