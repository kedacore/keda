package internal

// All code in this file is private to the package.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/nexus-rpc/sdk-go/nexus"
	commonpb "go.temporal.io/api/common/v1"
	deploymentpb "go.temporal.io/api/deployment/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/temporalproto"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/api/workflowservicemock/v1"
	"google.golang.org/protobuf/proto"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/internal/common/serializer"
	"go.temporal.io/sdk/internal/common/util"
	ilog "go.temporal.io/sdk/internal/log"
	"go.temporal.io/sdk/log"
)

const (
	// Set to 2 pollers for now, can adjust later if needed. The typical RTT (round-trip time) is below 1ms within data
	// center. And the poll API latency is about 5ms. With 2 poller, we could achieve around 300~400 RPS.
	defaultConcurrentPollRoutineSize = 2

	defaultAutoscalingInitialNumberOfPollers = 5   // Default initial number of pollers when using autoscaling.
	defaultAutoscalingMinimumNumberOfPollers = 1   // Default minimum number of pollers when using autoscaling.
	defaultAutoscalingMaximumNumberOfPollers = 100 // Default maximum number of pollers when using autoscaling.

	defaultMaxConcurrentActivityExecutionSize = 1000   // Large concurrent activity execution size (1k)
	defaultWorkerActivitiesPerSecond          = 100000 // Large activity executions/sec (unlimited)

	defaultMaxConcurrentLocalActivityExecutionSize = 1000   // Large concurrent activity execution size (1k)
	defaultWorkerLocalActivitiesPerSecond          = 100000 // Large activity executions/sec (unlimited)

	defaultTaskQueueActivitiesPerSecond = 100000.0 // Large activity executions/sec (unlimited)

	defaultMaxConcurrentTaskExecutionSize = 1000   // hardcoded max task execution size.
	defaultWorkerTaskExecutionRate        = 100000 // Large task execution rate (unlimited)

	defaultPollerRate = 1000

	defaultMaxConcurrentSessionExecutionSize = 1000 // Large concurrent session execution size (1k)

	defaultDeadlockDetectionTimeout = time.Second // By default kill workflow tasks that are running more than 1 sec.
	// Unlimited deadlock detection timeout is used when we want to allow workflow tasks to run indefinitely, such
	// as during debugging.
	unlimitedDeadlockDetectionTimeout = math.MaxInt64

	testTagsContextKey = "temporal-testTags"
)

type (
	// WorkflowWorker wraps the code for hosting workflow types.
	// And worker is mapped 1:1 with task queue. If the user want's to poll multiple
	// task queue names they might have to manage 'n' workers for 'n' task queues.
	workflowWorker struct {
		executionParameters workerExecutionParameters
		workflowService     workflowservice.WorkflowServiceClient
		worker              *baseWorker
		localActivityWorker *baseWorker
		identity            string
		stopC               chan struct{}
		localActivityStopC  chan struct{}
	}

	// ActivityWorker wraps the code for hosting activity types.
	// TODO: Worker doing heartbeating automatically while activity task is running
	activityWorker struct {
		executionParameters workerExecutionParameters
		workflowService     workflowservice.WorkflowServiceClient
		poller              taskPoller
		worker              *baseWorker
		identity            string
		stopC               chan struct{}
	}

	// sessionWorker wraps the code for hosting session creation, completion and
	// activities within a session. The creationWorker polls from a global taskqueue,
	// while the activityWorker polls from a resource specific taskqueue.
	sessionWorker struct {
		creationWorker *activityWorker
		activityWorker *activityWorker
	}

	// Worker overrides.
	workerOverrides struct {
		workflowTaskHandler WorkflowTaskHandler
		activityTaskHandler ActivityTaskHandler
		slotSupplier        SlotSupplier
	}

	// workerExecutionParameters defines worker configure/execution options.
	workerExecutionParameters struct {
		// Namespace name.
		Namespace string

		// Task queue name to poll.
		TaskQueue string

		// The tuner for the worker.
		Tuner WorkerTuner

		// Defines rate limiting on number of activity tasks that can be executed per second per worker.
		WorkerActivitiesPerSecond float64

		// Defines rate limiting on number of local activities that can be executed per second per worker.
		WorkerLocalActivitiesPerSecond float64

		// TaskQueueActivitiesPerSecond is the throttling limit for activity tasks controlled by the server.
		TaskQueueActivitiesPerSecond float64

		// User can provide an identity for the debuggability. If not provided the framework has
		// a default option.
		Identity string

		// The worker's build ID used for versioning, if one was set.
		WorkerBuildID string

		// If true the worker is opting in to build ID based versioning.
		UseBuildIDForVersioning bool

		// The worker deployment version identifier.
		// If non-empty, the [WorkerBuildID] from it, ignoring any previous value.
		WorkerDeploymentVersion WorkerDeploymentVersion

		// The Versioning Behavior for workflows that do not set one when registering the workflow type.
		DefaultVersioningBehavior VersioningBehavior

		MetricsHandler metrics.Handler

		Logger log.Logger

		// Enable logging in replay mode
		EnableLoggingInReplay bool

		// Context to store user provided key/value pairs
		BackgroundContext context.Context

		// Context cancel function to cancel user context
		BackgroundContextCancel context.CancelCauseFunc

		StickyScheduleToStartTimeout time.Duration

		// WorkflowPanicPolicy is used for configuring how client's workflow task handler deals with workflow
		// code panicking which includes non backwards compatible changes to the workflow code without appropriate
		// versioning (see workflow.GetVersion).
		// The default behavior is to block workflow execution until the problem is fixed.
		WorkflowPanicPolicy WorkflowPanicPolicy

		DataConverter converter.DataConverter

		FailureConverter converter.FailureConverter

		// WorkerStopTimeout is the time delay before hard terminate worker
		WorkerStopTimeout time.Duration

		// WorkerStopChannel is a read only channel listen on worker close. The worker will close the channel before exit.
		WorkerStopChannel <-chan struct{}

		// WorkerFatalErrorCallback is a callback for fatal errors that should stop
		// the worker.
		WorkerFatalErrorCallback func(error)

		// SessionResourceID is a unique identifier of the resource the session will consume
		SessionResourceID string

		ContextPropagators []ContextPropagator

		// DeadlockDetectionTimeout specifies workflow task timeout.
		DeadlockDetectionTimeout time.Duration

		DefaultHeartbeatThrottleInterval time.Duration

		MaxHeartbeatThrottleInterval time.Duration

		// WorkflowTaskPollerBehavior defines the behavior of the workflow task poller.
		WorkflowTaskPollerBehavior PollerBehavior

		// ActivityTaskPollerBehavior defines the behavior of the activity task poller.
		ActivityTaskPollerBehavior PollerBehavior

		// NexusTaskPollerBehavior defines the behavior of the nexus task poller.
		NexusTaskPollerBehavior PollerBehavior

		// Pointer to the shared worker cache
		cache *WorkerCache

		eagerActivityExecutor *eagerActivityExecutor

		capabilities *workflowservice.GetSystemInfoResponse_Capabilities
	}

	// HistoryJSONOptions are options for HistoryFromJSON.
	HistoryJSONOptions struct {
		// LastEventID, if set, will only load history up to this ID (inclusive).
		LastEventID int64
	}

	// Represents the version of a specific worker deployment.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/worker.WorkerDeploymentVersion]
	WorkerDeploymentVersion struct {
		// The name of the deployment this worker version belongs to
		DeploymentName string
		// The build id specific to this worker
		BuildID string
	}
)

var debugMode = os.Getenv("TEMPORAL_DEBUG") != ""

// newWorkflowWorker returns an instance of the workflow worker.
func newWorkflowWorker(client *WorkflowClient, params workerExecutionParameters, ppMgr pressurePointMgr, registry *registry) *workflowWorker {
	return newWorkflowWorkerInternal(client, params, ppMgr, nil, registry)
}

func ensureRequiredParams(params *workerExecutionParameters) {
	if params.Identity == "" {
		params.Identity = getWorkerIdentity(params.TaskQueue)
	}
	if params.Logger == nil {
		// create default logger if user does not supply one (should happen in tests only).
		params.Logger = ilog.NewDefaultLogger()
		params.Logger.Info("No logger configured for temporal worker. Created default one.")
	}
	if params.MetricsHandler == nil {
		params.MetricsHandler = metrics.NopHandler
		params.Logger.Info("No metrics handler configured for temporal worker. Use NopHandler as default.")
	}
	if params.DataConverter == nil {
		params.DataConverter = converter.GetDefaultDataConverter()
		params.Logger.Info("No DataConverter configured for temporal worker. Use default one.")
	}
	if params.FailureConverter == nil {
		params.FailureConverter = GetDefaultFailureConverter()
	}
	if params.Tuner == nil {
		// Err cannot happen since these slot numbers are guaranteed valid
		params.Tuner, _ = NewFixedSizeTuner(
			FixedSizeTunerOptions{
				NumWorkflowSlots:      defaultMaxConcurrentTaskExecutionSize,
				NumActivitySlots:      defaultMaxConcurrentActivityExecutionSize,
				NumLocalActivitySlots: defaultMaxConcurrentLocalActivityExecutionSize,
				NumNexusSlots:         defaultMaxConcurrentTaskExecutionSize,
			})
	}
}

// getBuildID returns either the user-defined build ID if it was provided, or an autogenerated one
// using getBinaryChecksum
func (params *workerExecutionParameters) getBuildID() string {
	if params.WorkerBuildID != "" {
		return params.WorkerBuildID
	}
	return getBinaryChecksum()
}

// Returns true if this worker is part of our system namespace or per-namespace system task queue
func (params *workerExecutionParameters) isInternalWorker() bool {
	return params.Namespace == "temporal-system" || params.TaskQueue == "temporal-sys-per-ns-tq"
}

// verifyNamespaceExist does a DescribeNamespace operation on the specified namespace with backoff/retry
func verifyNamespaceExist(
	client workflowservice.WorkflowServiceClient,
	metricsHandler metrics.Handler,
	namespace string,
	logger log.Logger,
) error {
	ctx := context.Background()
	if namespace == "" {
		return errors.New("namespace cannot be empty")
	}
	grpcCtx, cancel := newGRPCContext(ctx, grpcMetricsHandler(metricsHandler), defaultGrpcRetryParameters(ctx))
	defer cancel()
	_, err := client.DescribeNamespace(grpcCtx, &workflowservice.DescribeNamespaceRequest{Namespace: namespace})
	return err
}

func newWorkflowWorkerInternal(client *WorkflowClient, params workerExecutionParameters, ppMgr pressurePointMgr, overrides *workerOverrides, registry *registry) *workflowWorker {
	workerStopChannel := make(chan struct{})
	params.WorkerStopChannel = getReadOnlyChannel(workerStopChannel)
	// Get a workflow task handler.
	ensureRequiredParams(&params)
	var taskHandler WorkflowTaskHandler
	if overrides != nil && overrides.workflowTaskHandler != nil {
		taskHandler = overrides.workflowTaskHandler
	} else {
		taskHandler = newWorkflowTaskHandler(params, ppMgr, registry)
	}
	return newWorkflowTaskWorkerInternal(taskHandler, taskHandler, client, params, workerStopChannel, registry.interceptors)
}

func newWorkflowTaskWorkerInternal(
	taskHandler WorkflowTaskHandler,
	contextManager WorkflowContextManager,
	client *WorkflowClient,
	params workerExecutionParameters,
	stopC chan struct{},
	interceptors []WorkerInterceptor,
) *workflowWorker {
	ensureRequiredParams(&params)
	var service workflowservice.WorkflowServiceClient
	if client != nil {
		service = client.workflowService
	}
	taskProcessor := newWorkflowTaskProcessor(taskHandler, contextManager, service, params)

	var scalableTaskPollers []scalableTaskPoller
	switch params.WorkflowTaskPollerBehavior.(type) {
	case *pollerBehaviorSimpleMaximum:
		scalableTaskPollers = []scalableTaskPoller{
			newScalableTaskPoller(taskProcessor.createPoller(Mixed), params.Logger, params.WorkflowTaskPollerBehavior),
		}
	case *pollerBehaviorAutoscaling:
		scalableTaskPollers = []scalableTaskPoller{
			newScalableTaskPoller(taskProcessor.createPoller(NonSticky), params.Logger, params.WorkflowTaskPollerBehavior),
		}
		if taskProcessor.stickyCacheSize > 0 {
			scalableTaskPollers = append(scalableTaskPollers, newScalableTaskPoller(taskProcessor.createPoller(Sticky), params.Logger, params.WorkflowTaskPollerBehavior))
		}
	}

	bwo := baseWorkerOptions{
		pollerRate:       defaultPollerRate,
		slotSupplier:     params.Tuner.GetWorkflowTaskSlotSupplier(),
		maxTaskPerSecond: defaultWorkerTaskExecutionRate,
		taskPollers:      scalableTaskPollers,
		taskProcessor:    taskProcessor,
		workerType:       "WorkflowWorker",
		identity:         params.Identity,
		buildId:          params.getBuildID(),
		logger:           params.Logger,
		stopTimeout:      params.WorkerStopTimeout,
		fatalErrCb:       params.WorkerFatalErrorCallback,
		metricsHandler:   params.MetricsHandler,
		slotReservationData: slotReservationData{
			taskQueue: params.TaskQueue,
		},
	}

	worker := newBaseWorker(bwo)

	// We want a separate stop channel for local activities because when a worker shuts down,
	// we need to allow pending local activities to finish running for that workflow task.
	// After all pending local activities are handled, we then close the local activity stop channel.
	laStopChannel := make(chan struct{})
	laParams := params
	laParams.WorkerStopChannel = laStopChannel

	// laTunnel is the glue that hookup 3 parts
	laTunnel := newLocalActivityTunnel(getReadOnlyChannel(laStopChannel))

	// 1) workflow handler will send local activity task to laTunnel
	if handlerImpl, ok := taskHandler.(*workflowTaskHandlerImpl); ok {
		handlerImpl.laTunnel = laTunnel
	}

	// 2) local activity task poller will poll from laTunnel, and result will be pushed to laTunnel
	localActivityTaskPoller := newLocalActivityPoller(laParams, laTunnel, interceptors, client, stopC)
	localActivityWorker := newBaseWorker(baseWorkerOptions{
		slotSupplier:     laParams.Tuner.GetLocalActivitySlotSupplier(),
		maxTaskPerSecond: laParams.WorkerLocalActivitiesPerSecond,
		taskPollers: []scalableTaskPoller{
			newScalableTaskPoller(localActivityTaskPoller, params.Logger, NewPollerBehaviorSimpleMaximum(
				PollerBehaviorSimpleMaximumOptions{
					MaximumNumberOfPollers: 2,
				},
			)),
		},
		taskProcessor:  localActivityTaskPoller,
		workerType:     "LocalActivityWorker",
		identity:       laParams.Identity,
		buildId:        laParams.getBuildID(),
		logger:         laParams.Logger,
		stopTimeout:    laParams.WorkerStopTimeout,
		fatalErrCb:     laParams.WorkerFatalErrorCallback,
		metricsHandler: laParams.MetricsHandler,
		slotReservationData: slotReservationData{
			taskQueue: params.TaskQueue,
		},
	},
	)

	// 3) the result pushed to laTunnel will be sent as task to workflow worker to process.
	worker.taskQueueCh = laTunnel.resultCh

	return &workflowWorker{
		executionParameters: params,
		workflowService:     service,
		worker:              worker,
		localActivityWorker: localActivityWorker,
		identity:            params.Identity,
		stopC:               stopC,
		localActivityStopC:  laStopChannel,
	}
}

// Start the worker.
func (ww *workflowWorker) Start() error {
	err := verifyNamespaceExist(ww.workflowService, ww.executionParameters.MetricsHandler, ww.executionParameters.Namespace, ww.worker.logger)
	if err != nil {
		return err
	}
	ww.localActivityWorker.Start()
	ww.worker.Start()
	return nil // TODO: propagate error
}

// Stop the worker.
func (ww *workflowWorker) Stop() {
	close(ww.stopC)
	// TODO: remove the stop methods in favor of the workerStopChannel
	ww.worker.Stop()
	close(ww.localActivityStopC)
	ww.localActivityWorker.Stop()
}

func newSessionWorker(client *WorkflowClient, params workerExecutionParameters, env *registry, maxConcurrentSessionExecutionSize int) *sessionWorker {
	if params.Identity == "" {
		params.Identity = getWorkerIdentity(params.TaskQueue)
	}
	// For now resourceID is hidden from user so we will always create a unique one for each worker.
	if params.SessionResourceID == "" {
		params.SessionResourceID = uuid.NewString()
	}
	sessionEnvironment := newSessionEnvironment(params.SessionResourceID, maxConcurrentSessionExecutionSize)

	creationTaskqueue := getCreationTaskqueue(params.TaskQueue)
	params.BackgroundContext = context.WithValue(params.BackgroundContext, sessionEnvironmentContextKey, sessionEnvironment)
	params.TaskQueue = sessionEnvironment.GetResourceSpecificTaskqueue()
	activityWorker := newActivityWorker(client, params,
		&workerOverrides{slotSupplier: params.Tuner.GetSessionActivitySlotSupplier()}, env, nil)

	params.ActivityTaskPollerBehavior = NewPollerBehaviorSimpleMaximum(
		PollerBehaviorSimpleMaximumOptions{
			MaximumNumberOfPollers: 1,
		},
	)
	params.TaskQueue = creationTaskqueue
	// Although we have session token bucket to limit session size across creation
	// and recreation, we also limit it here for creation only
	overrides := &workerOverrides{}
	overrides.slotSupplier, _ = NewFixedSizeSlotSupplier(maxConcurrentSessionExecutionSize)
	creationWorker := newActivityWorker(client, params, overrides, env, sessionEnvironment.GetTokenBucket())

	return &sessionWorker{
		creationWorker: creationWorker,
		activityWorker: activityWorker,
	}
}

func (sw *sessionWorker) Start() error {
	err := sw.creationWorker.Start()
	if err != nil {
		return err
	}

	err = sw.activityWorker.Start()
	if err != nil {
		sw.creationWorker.Stop()
		return err
	}
	return nil
}

func (sw *sessionWorker) Stop() {
	sw.creationWorker.Stop()
	sw.activityWorker.Stop()
}

func newActivityWorker(
	client *WorkflowClient,
	params workerExecutionParameters,
	overrides *workerOverrides,
	env *registry,
	sessionTokenBucket *sessionTokenBucket,
) *activityWorker {
	var service workflowservice.WorkflowServiceClient
	if client != nil {
		service = client.workflowService
	}
	workerStopChannel := make(chan struct{}, 1)
	params.WorkerStopChannel = getReadOnlyChannel(workerStopChannel)
	ensureRequiredParams(&params)

	// Get a activity task handler.
	var taskHandler ActivityTaskHandler
	if overrides != nil && overrides.activityTaskHandler != nil {
		taskHandler = overrides.activityTaskHandler
	} else {
		taskHandler = newActivityTaskHandler(client, params, env)
	}

	poller := newActivityTaskPoller(taskHandler, service, params)
	var slotSupplier SlotSupplier
	if overrides != nil && overrides.slotSupplier != nil {
		slotSupplier = overrides.slotSupplier
	} else {
		slotSupplier = params.Tuner.GetActivityTaskSlotSupplier()
	}

	bwo := baseWorkerOptions{
		pollerRate:       defaultPollerRate,
		slotSupplier:     slotSupplier,
		maxTaskPerSecond: params.WorkerActivitiesPerSecond,
		taskPollers: []scalableTaskPoller{
			newScalableTaskPoller(poller, params.Logger, params.ActivityTaskPollerBehavior),
		},
		taskProcessor:           poller,
		workerType:              "ActivityWorker",
		identity:                params.Identity,
		buildId:                 params.getBuildID(),
		logger:                  params.Logger,
		stopTimeout:             params.WorkerStopTimeout,
		fatalErrCb:              params.WorkerFatalErrorCallback,
		backgroundContextCancel: params.BackgroundContextCancel,
		metricsHandler:          params.MetricsHandler,
		sessionTokenBucket:      sessionTokenBucket,
		slotReservationData: slotReservationData{
			taskQueue: params.TaskQueue,
		},
	}

	base := newBaseWorker(bwo)
	return &activityWorker{
		executionParameters: params,
		workflowService:     service,
		worker:              base,
		poller:              poller,
		identity:            params.Identity,
		stopC:               workerStopChannel,
	}
}

// Start the worker.
func (aw *activityWorker) Start() error {
	err := verifyNamespaceExist(aw.workflowService, aw.executionParameters.MetricsHandler, aw.executionParameters.Namespace, aw.worker.logger)
	if err != nil {
		return err
	}
	aw.worker.Start()
	return nil // TODO: propagate errors
}

// Stop the worker.
func (aw *activityWorker) Stop() {
	close(aw.stopC)
	aw.worker.Stop()
}

type registry struct {
	sync.Mutex
	nexusServices                 map[string]*nexus.Service
	workflowFuncMap               map[string]interface{}
	workflowAliasMap              map[string]string
	workflowVersioningBehaviorMap map[string]VersioningBehavior
	activityFuncMap               map[string]activity
	activityAliasMap              map[string]string
	dynamicWorkflow               interface{}
	dynamicWorkflowOptions        DynamicRegisterWorkflowOptions
	dynamicActivity               activity
	_                             DynamicRegisterActivityOptions
	interceptors                  []WorkerInterceptor
}

type registryOptions struct {
	disableAliasing bool
}

func (r *registry) RegisterWorkflow(af interface{}) {
	r.RegisterWorkflowWithOptions(af, RegisterWorkflowOptions{})
}

func (r *registry) RegisterWorkflowWithOptions(
	wf interface{},
	options RegisterWorkflowOptions,
) {
	// Support direct registration of WorkflowDefinition
	factory, ok := wf.(WorkflowDefinitionFactory)
	if ok {
		if len(options.Name) == 0 {
			panic("WorkflowDefinitionFactory must be registered with a name")
		}
		if strings.HasPrefix(options.Name, temporalPrefix) {
			panic(temporalPrefixError)
		}
		r.Lock()
		defer r.Unlock()
		r.workflowFuncMap[options.Name] = factory
		r.workflowVersioningBehaviorMap[options.Name] = options.VersioningBehavior
		return
	}
	// Validate that it is a function
	fnType := reflect.TypeOf(wf)
	if err := validateFnFormat(fnType, true, false); err != nil {
		panic(err)
	}
	fnName, _ := getFunctionName(wf)
	alias := options.Name
	registerName := fnName
	if len(alias) > 0 {
		registerName = alias
	}

	if strings.HasPrefix(alias, temporalPrefix) || strings.HasPrefix(registerName, temporalPrefix) {
		panic(temporalPrefixError)
	}

	r.Lock()
	defer r.Unlock()

	if !options.DisableAlreadyRegisteredCheck {
		if _, ok := r.workflowFuncMap[registerName]; ok {
			panic(fmt.Sprintf("workflow name \"%v\" is already registered", registerName))
		}
	}
	r.workflowFuncMap[registerName] = wf
	r.workflowVersioningBehaviorMap[registerName] = options.VersioningBehavior

	if len(alias) > 0 && r.workflowAliasMap != nil {
		r.workflowAliasMap[fnName] = alias
	}
}

func (r *registry) RegisterDynamicWorkflow(wf interface{}, options DynamicRegisterWorkflowOptions) {
	r.Lock()
	defer r.Unlock()
	// Support direct registration of WorkflowDefinition
	factory, ok := wf.(WorkflowDefinitionFactory)
	if ok {
		r.dynamicWorkflow = factory
		r.dynamicWorkflowOptions = options
		return
	}

	// Validate that it is a function
	fnType := reflect.TypeOf(wf)
	if err := validateFnFormat(fnType, true, true); err != nil {
		panic(err)
	}
	if r.dynamicWorkflow != nil {
		panic("dynamic workflow already registered")
	}
	r.dynamicWorkflow = wf
	r.dynamicWorkflowOptions = options
}

func (r *registry) RegisterActivity(af interface{}) {
	r.RegisterActivityWithOptions(af, RegisterActivityOptions{})
}

func (r *registry) RegisterActivityWithOptions(
	af interface{},
	options RegisterActivityOptions,
) {
	// Support direct registration of activity
	a, ok := af.(activity)
	if ok {
		if options.Name == "" {
			panic("registration of activity interface requires name")
		}
		if strings.HasPrefix(options.Name, temporalPrefix) {
			panic(temporalPrefixError)
		}
		r.addActivityWithLock(options.Name, a)
		return
	}
	// Validate that it is a function
	fnType := reflect.TypeOf(af)
	if fnType.Kind() == reflect.Ptr && fnType.Elem().Kind() == reflect.Struct {
		registerErr := r.registerActivityStructWithOptions(af, options)
		if registerErr != nil {
			panic(registerErr)
		}
		return
	}
	if err := validateFnFormat(fnType, false, false); err != nil {
		panic(err)
	}
	fnName, _ := getFunctionName(af)
	alias := options.Name
	registerName := fnName
	if len(alias) > 0 {
		registerName = alias
	}

	if strings.HasPrefix(alias, temporalPrefix) || strings.HasPrefix(registerName, temporalPrefix) {
		panic(temporalPrefixError)
	}

	r.Lock()
	defer r.Unlock()

	if !options.DisableAlreadyRegisteredCheck {
		if _, ok := r.activityFuncMap[registerName]; ok {
			panic(fmt.Sprintf("activity type \"%v\" is already registered", registerName))
		}
	}
	r.activityFuncMap[registerName] = &activityExecutor{name: registerName, fn: af}
	if len(alias) > 0 && r.activityAliasMap != nil {
		r.activityAliasMap[fnName] = alias
	}
}

func (r *registry) registerActivityStructWithOptions(aStruct interface{}, options RegisterActivityOptions) error {
	r.Lock()
	defer r.Unlock()

	structValue := reflect.ValueOf(aStruct)
	structType := structValue.Type()
	count := 0
	for i := 0; i < structValue.NumMethod(); i++ {
		methodValue := structValue.Method(i)
		method := structType.Method(i)
		// skip private method
		if method.PkgPath != "" {
			continue
		}
		name := method.Name
		if err := validateFnFormat(method.Type, false, false); err != nil {
			if options.SkipInvalidStructFunctions {
				continue
			}

			return fmt.Errorf("method %s of %s: %w", name, structType.Name(), err)
		}
		registerName := options.Name + name
		if !options.DisableAlreadyRegisteredCheck {
			if _, ok := r.getActivityNoLock(registerName); ok {
				return fmt.Errorf("activity type \"%v\" is already registered", registerName)
			}
		}
		r.activityFuncMap[registerName] = &activityExecutor{name: registerName, fn: methodValue.Interface()}
		count++
	}
	if count == 0 {
		return fmt.Errorf("no activities (public methods) found at %v structure", structType.Name())
	}
	return nil
}

func (r *registry) RegisterDynamicActivity(af interface{}, options DynamicRegisterActivityOptions) {
	r.Lock()
	defer r.Unlock()
	// Support direct registration of activity
	a, ok := af.(activity)
	if ok {
		r.dynamicActivity = a
		return
	}
	// Validate that it is a function
	fnType := reflect.TypeOf(af)
	if err := validateFnFormat(fnType, false, true); err != nil {
		panic(err)
	}
	if r.dynamicActivity != nil {
		panic("dynamic activity already registered")
	}
	r.dynamicActivity = &activityExecutor{name: "", fn: af, dynamic: true}
}

func (r *registry) RegisterNexusService(service *nexus.Service) {
	if service.Name == "" {
		panic(fmt.Errorf("tried to register a service with no name"))
	}

	r.Lock()
	defer r.Unlock()

	if _, ok := r.nexusServices[service.Name]; ok {
		panic(fmt.Sprintf("service name \"%v\" is already registered", service.Name))
	}
	r.nexusServices[service.Name] = service
}

func (r *registry) getWorkflowAlias(fnName string) (string, bool) {
	r.Lock()
	defer r.Unlock()
	alias, ok := r.workflowAliasMap[fnName]
	return alias, ok
}

func (r *registry) getWorkflowFn(fnName string) (interface{}, bool) {
	r.Lock()
	defer r.Unlock()
	if fn, ok := r.workflowFuncMap[fnName]; ok {
		return fn, ok
	}

	if r.dynamicWorkflow != nil {
		return "dynamic", true
	}
	return nil, false
}

func (r *registry) getRegisteredWorkflowTypes() []string {
	r.Lock()
	defer r.Unlock()
	var result []string
	for t := range r.workflowFuncMap {
		result = append(result, t)
	}
	return result
}

func (r *registry) getActivityAlias(fnName string) (string, bool) {
	r.Lock()
	defer r.Unlock()
	alias, ok := r.activityAliasMap[fnName]
	return alias, ok
}

func (r *registry) addActivityWithLock(fnName string, a activity) {
	r.Lock()
	defer r.Unlock()
	r.activityFuncMap[fnName] = a
}

func (r *registry) GetActivity(fnName string) (activity, bool) {
	r.Lock()
	defer r.Unlock()
	if a, ok := r.activityFuncMap[fnName]; ok {
		return a, ok
	}
	if r.dynamicActivity != nil {
		return r.dynamicActivity, true
	}
	return nil, false
}

func (r *registry) getActivityNoLock(fnName string) (activity, bool) {
	a, ok := r.activityFuncMap[fnName]
	return a, ok
}

func (r *registry) getRegisteredActivities() []activity {
	r.Lock()
	defer r.Unlock()
	numActivities := len(r.activityFuncMap)
	if r.dynamicActivity != nil {
		numActivities++
	}
	activities := make([]activity, 0, numActivities)
	for _, a := range r.activityFuncMap {
		activities = append(activities, a)
	}
	if r.dynamicActivity != nil {
		activities = append(activities, r.dynamicActivity)
	}
	return activities
}

func (r *registry) getRegisteredActivityTypes() []string {
	r.Lock()
	defer r.Unlock()
	var result []string
	for name := range r.activityFuncMap {
		result = append(result, name)
	}
	return result
}

func (r *registry) getWorkflowDefinition(wt WorkflowType) (WorkflowDefinition, error) {
	lookup := wt.Name
	if alias, ok := r.getWorkflowAlias(lookup); ok {
		lookup = alias
	}
	wf, ok := r.getWorkflowFn(lookup)
	if !ok {
		supported := strings.Join(r.getRegisteredWorkflowTypes(), ", ")
		return nil, fmt.Errorf("unable to find workflow type: %v. Supported types: [%v]", lookup, supported)
	}
	wdf, ok := wf.(WorkflowDefinitionFactory)
	if ok {
		return wdf.NewWorkflowDefinition(), nil
	}
	var dynamic bool
	if d, ok := wf.(string); ok && d == "dynamic" {
		wf = r.dynamicWorkflow
		dynamic = true
	}
	executor := &workflowExecutor{workflowType: lookup, fn: wf, interceptors: r.interceptors, dynamic: dynamic}
	return newSyncWorkflowDefinition(executor), nil
}

func (r *registry) getWorkflowVersioningBehavior(wt WorkflowType) (VersioningBehavior, bool) {
	lookup := wt.Name
	if alias, ok := r.getWorkflowAlias(lookup); ok {
		lookup = alias
	}
	r.Lock()
	defer r.Unlock()
	if behavior, ok := r.workflowVersioningBehaviorMap[lookup]; ok {
		return behavior, behavior != VersioningBehaviorUnspecified
	}
	if r.dynamicWorkflowOptions.LoadDynamicRuntimeOptions != nil {
		config := LoadDynamicRuntimeOptionsDetails{WorkflowType: wt}
		if behavior, err := r.dynamicWorkflowOptions.LoadDynamicRuntimeOptions(config); err == nil {
			return behavior.VersioningBehavior, true
		}
	}
	return VersioningBehaviorUnspecified, false
}

func (r *registry) getNexusService(service string) *nexus.Service {
	r.Lock()
	defer r.Unlock()
	return r.nexusServices[service]
}

func (r *registry) getRegisteredNexusServices() []*nexus.Service {
	r.Lock()
	defer r.Unlock()
	result := make([]*nexus.Service, 0, len(r.nexusServices))
	for _, s := range r.nexusServices {
		result = append(result, s)
	}
	return result
}

// Validate function parameters.
func validateFnFormat(fnType reflect.Type, isWorkflow, isDynamic bool) error {
	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("expected a func as input but was %s", fnType.Kind())
	}
	if isWorkflow {
		if fnType.NumIn() < 1 {
			return fmt.Errorf(
				"expected at least one argument of type workflow.Context in function, found %d input arguments",
				fnType.NumIn(),
			)
		}
		if !isWorkflowContext(fnType.In(0)) {
			return fmt.Errorf("expected first argument to be workflow.Context but found %s", fnType.In(0))
		}
	} else {
		// For activities, check that workflow context is not accidentally provided
		// Activities registered with structs will have their receiver as the first argument so confirm it is not
		// in the first two arguments
		for i := 0; i < fnType.NumIn() && i < 2; i++ {
			if isWorkflowContext(fnType.In(i)) {
				return fmt.Errorf("unexpected use of workflow context for an activity")
			}
		}
	}

	if isDynamic {
		if fnType.NumIn() != 2 {
			return fmt.Errorf(
				"expected function to have two arguments, first being workflow.Context and second being an EncodedValues type, found %d arguments", fnType.NumIn(),
			)
		}
		if fnType.In(1) != reflect.TypeOf((*converter.EncodedValues)(nil)).Elem() {
			return fmt.Errorf("expected function to EncodedValues as second argument, got %s", fnType.In(1).Elem())
		}
	}

	// Return values
	// We expect either
	// 	<result>, error
	//	(or) just error
	if fnType.NumOut() < 1 || fnType.NumOut() > 2 {
		return fmt.Errorf(
			"expected function to return result, error or just error, but found %d return values", fnType.NumOut(),
		)
	}
	if fnType.NumOut() > 1 && !isValidResultType(fnType.Out(0)) {
		return fmt.Errorf(
			"expected function first return value to return valid type but found: %v", fnType.Out(0).Kind(),
		)
	}
	if !isError(fnType.Out(fnType.NumOut() - 1)) {
		return fmt.Errorf(
			"expected function second return value to return error but found %v", fnType.Out(fnType.NumOut()-1).Kind(),
		)
	}
	return nil
}

func newRegistry() *registry { return newRegistryWithOptions(registryOptions{}) }

func newRegistryWithOptions(options registryOptions) *registry {
	r := &registry{
		workflowFuncMap:               make(map[string]interface{}),
		workflowVersioningBehaviorMap: make(map[string]VersioningBehavior),
		activityFuncMap:               make(map[string]activity),
		nexusServices:                 make(map[string]*nexus.Service),
	}
	if !options.disableAliasing {
		r.workflowAliasMap = make(map[string]string)
		r.activityAliasMap = make(map[string]string)
	}
	return r
}

// Wrapper to execute workflow functions.
type workflowExecutor struct {
	workflowType string
	fn           interface{}
	interceptors []WorkerInterceptor
	dynamic      bool
}

func (we *workflowExecutor) Execute(ctx Context, input *commonpb.Payloads) (*commonpb.Payloads, error) {
	dataConverter := WithWorkflowContext(ctx, getWorkflowEnvOptions(ctx).DataConverter)
	fnType := reflect.TypeOf(we.fn)

	var args []interface{}
	var err error
	if we.dynamic {
		// Dynamic workflows take in a single EncodedValues, encode all data into single EncodedValues
		args = []interface{}{newEncodedValues(input, dataConverter)}
	} else {
		args, err = decodeArgsToRawValues(dataConverter, fnType, input)
		if err != nil {
			return nil, fmt.Errorf(
				"unable to decode the workflow function input payload with error: %w, function name: %v",
				err, we.workflowType)
		}
	}

	envInterceptor := getWorkflowEnvironmentInterceptor(ctx)
	envInterceptor.fn = we.fn

	// Execute and serialize result
	result, err := envInterceptor.inboundInterceptor.ExecuteWorkflow(ctx, &ExecuteWorkflowInput{Args: args})
	var serializedResult *commonpb.Payloads
	if err == nil && result != nil {
		serializedResult, err = encodeArg(dataConverter, result)
	}
	return serializedResult, err
}

// Wrapper to execute activity functions.
type activityExecutor struct {
	name             string
	fn               interface{}
	skipInterceptors bool
	dynamic          bool
}

func (ae *activityExecutor) ActivityType() ActivityType {
	return ActivityType{Name: ae.name}
}

func (ae *activityExecutor) GetFunction() interface{} {
	return ae.fn
}

func (ae *activityExecutor) Execute(ctx context.Context, input *commonpb.Payloads) (*commonpb.Payloads, error) {
	fnType := reflect.TypeOf(ae.fn)
	dataConverter := getDataConverterFromActivityCtx(ctx)

	var args []interface{}
	var err error
	if ae.dynamic {
		// Dynamic activities take in a single EncodedValues, encode all data into single EncodedValues
		args = []interface{}{newEncodedValues(input, dataConverter)}
	} else {
		args, err = decodeArgsToRawValues(dataConverter, fnType, input)
		if err != nil {
			return nil, fmt.Errorf(
				"unable to decode the activity function input payload with error: %w for function name: %v",
				err, ae.name)
		}
	}

	return ae.ExecuteWithActualArgs(ctx, args)
}

func (ae *activityExecutor) ExecuteWithActualArgs(ctx context.Context, args []interface{}) (*commonpb.Payloads, error) {
	dataConverter := getDataConverterFromActivityCtx(ctx)

	envInterceptor := getActivityEnvironmentInterceptor(ctx)
	envInterceptor.fn = ae.fn

	// Execute and serialize result
	interceptor := envInterceptor.inboundInterceptor
	if ae.skipInterceptors {
		interceptor = envInterceptor
	}
	result, resultErr := interceptor.ExecuteActivity(ctx, &ExecuteActivityInput{Args: args})
	var serializedResult *commonpb.Payloads
	if result != nil {
		// As a special case, if the result is already a payload, just use it
		var ok bool
		if serializedResult, ok = result.(*commonpb.Payloads); !ok {
			var err error
			if serializedResult, err = encodeArg(dataConverter, result); err != nil {
				return nil, err
			}
		}
	}
	return serializedResult, resultErr
}

func getDataConverterFromActivityCtx(ctx context.Context) converter.DataConverter {
	var dataConverter converter.DataConverter

	env := getActivityEnvironmentFromCtx(ctx)
	if env != nil && env.dataConverter != nil {
		dataConverter = env.dataConverter
	} else {
		dataConverter = converter.GetDefaultDataConverter()
	}
	return WithContext(ctx, dataConverter)
}

func getNamespaceFromActivityCtx(ctx context.Context) string {
	env := getActivityEnvironmentFromCtx(ctx)
	if env == nil {
		return ""
	}
	return env.workflowNamespace
}

func getActivityEnvironmentFromCtx(ctx context.Context) *activityEnvironment {
	if ctx == nil || ctx.Value(activityEnvContextKey) == nil {
		return nil
	}
	return ctx.Value(activityEnvContextKey).(*activityEnvironment)
}

// AggregatedWorker combines management of both workflowWorker and activityWorker worker lifecycle.
type AggregatedWorker struct {
	// Stored for creating a nexus worker on Start.
	executionParams workerExecutionParameters
	// Memoized start function. Ensures start runs once and returns the same error when called multiple times.
	memoizedStart func() error

	client         *WorkflowClient
	workflowWorker *workflowWorker
	activityWorker *activityWorker
	sessionWorker  *sessionWorker
	nexusWorker    *nexusWorker
	logger         log.Logger
	registry       *registry
	// Stores a boolean indicating whether the worker has already been started.
	started      atomic.Bool
	stopC        chan struct{}
	fatalErr     error
	fatalErrLock sync.Mutex
	capabilities *workflowservice.GetSystemInfoResponse_Capabilities
}

// RegisterWorkflow registers workflow implementation with the AggregatedWorker
func (aw *AggregatedWorker) RegisterWorkflow(w interface{}) {
	if aw.workflowWorker == nil {
		panic("workflow worker disabled, cannot register workflow")
	}
	if aw.executionParams.UseBuildIDForVersioning &&
		(aw.executionParams.WorkerDeploymentVersion != WorkerDeploymentVersion{}) &&
		aw.executionParams.DefaultVersioningBehavior == VersioningBehaviorUnspecified {
		panic("workflow type does not have a versioning behavior")
	}
	aw.registry.RegisterWorkflow(w)
}

// RegisterWorkflowWithOptions registers workflow implementation with the AggregatedWorker
func (aw *AggregatedWorker) RegisterWorkflowWithOptions(w interface{}, options RegisterWorkflowOptions) {
	if aw.workflowWorker == nil {
		panic("workflow worker disabled, cannot register workflow")
	}
	if options.VersioningBehavior == VersioningBehaviorUnspecified &&
		(aw.executionParams.WorkerDeploymentVersion != WorkerDeploymentVersion{}) &&
		aw.executionParams.UseBuildIDForVersioning &&
		aw.executionParams.DefaultVersioningBehavior == VersioningBehaviorUnspecified {
		panic("workflow type does not have a versioning behavior")
	}
	aw.registry.RegisterWorkflowWithOptions(w, options)
}

// RegisterDynamicWorkflow registers dynamic workflow implementation with the AggregatedWorker
func (aw *AggregatedWorker) RegisterDynamicWorkflow(w interface{}, options DynamicRegisterWorkflowOptions) {
	if aw.workflowWorker == nil {
		panic("workflow worker disabled, cannot register workflow")
	}
	if options.LoadDynamicRuntimeOptions == nil && aw.executionParams.UseBuildIDForVersioning &&
		(aw.executionParams.WorkerDeploymentVersion != WorkerDeploymentVersion{}) &&
		aw.executionParams.DefaultVersioningBehavior == VersioningBehaviorUnspecified {
		panic("dynamic workflow does not have a versioning behavior")
	}
	aw.registry.RegisterDynamicWorkflow(w, options)
}

// RegisterActivity registers activity implementation with the AggregatedWorker
func (aw *AggregatedWorker) RegisterActivity(a interface{}) {
	aw.registry.RegisterActivity(a)
}

// RegisterActivityWithOptions registers activity implementation with the AggregatedWorker
func (aw *AggregatedWorker) RegisterActivityWithOptions(a interface{}, options RegisterActivityOptions) {
	aw.registry.RegisterActivityWithOptions(a, options)
}

// RegisterDynamicActivity registers the dynamic activity function with options.
// Registering activities via a structure is not supported for dynamic activities.
func (aw *AggregatedWorker) RegisterDynamicActivity(a interface{}, options DynamicRegisterActivityOptions) {
	aw.registry.RegisterDynamicActivity(a, options)
}

func (aw *AggregatedWorker) RegisterNexusService(service *nexus.Service) {
	if aw.started.Load() {
		panic(errors.New("cannot register Nexus services after worker start"))
	}
	aw.registry.RegisterNexusService(service)
}

// Start the worker in a non-blocking fashion.
// The actual work is done in the memoized "start" function to ensure duplicate calls are returned a consistent error.
func (aw *AggregatedWorker) Start() error {
	aw.assertNotStopped()
	return aw.memoizedStart()
}

// start the worker. This method is memoized using sync.OnceValue in memoizedStart.
func (aw *AggregatedWorker) start() error {
	aw.started.Store(true)

	if err := initBinaryChecksum(); err != nil {
		return fmt.Errorf("failed to get executable checksum: %v", err)
	} else if err = aw.client.ensureInitialized(context.Background()); err != nil {
		return err
	}
	// Populate the capabilities. This should be the only time it is written too.
	capabilities, err := aw.client.loadCapabilities(context.Background())
	if err != nil {
		return err
	}
	proto.Merge(aw.capabilities, capabilities)

	if !util.IsInterfaceNil(aw.workflowWorker) {
		if err := aw.workflowWorker.Start(); err != nil {
			return err
		}
		if aw.client.eagerDispatcher != nil {
			aw.client.eagerDispatcher.registerWorker(aw.workflowWorker)
		}
	}
	if !util.IsInterfaceNil(aw.activityWorker) {
		if err := aw.activityWorker.Start(); err != nil {
			// stop workflow worker.
			if !util.IsInterfaceNil(aw.workflowWorker) {
				if aw.workflowWorker.worker.isWorkerStarted {
					if aw.client.eagerDispatcher != nil {
						aw.client.eagerDispatcher.deregisterWorker(aw.workflowWorker)
					}
					aw.workflowWorker.Stop()
				}
			}
			return err
		}
	}

	if !util.IsInterfaceNil(aw.sessionWorker) && len(aw.registry.getRegisteredActivities()) > 0 {
		aw.logger.Info("Starting session worker")
		if err := aw.sessionWorker.Start(); err != nil {
			// stop workflow worker and activity worker.
			if !util.IsInterfaceNil(aw.workflowWorker) {
				if aw.workflowWorker.worker.isWorkerStarted {
					aw.workflowWorker.Stop()
				}
			}
			if !util.IsInterfaceNil(aw.activityWorker) {
				if aw.activityWorker.worker.isWorkerStarted {
					aw.activityWorker.Stop()
				}
			}
			return err
		}
	}
	nexusServices := aw.registry.getRegisteredNexusServices()
	if len(nexusServices) > 0 {
		reg := nexus.NewServiceRegistry()
		for _, service := range nexusServices {
			if err := reg.Register(service); err != nil {
				return fmt.Errorf("failed to create a nexus worker: %w", err)
			}
		}
		reg.Use(nexusMiddleware(aw.registry.interceptors))
		handler, err := reg.NewHandler()
		if err != nil {
			return fmt.Errorf("failed to create a nexus worker: %w", err)
		}
		aw.nexusWorker, err = newNexusWorker(nexusWorkerOptions{
			executionParameters: aw.executionParams,
			client:              aw.client,
			workflowService:     aw.client.workflowService,
			handler:             handler,
			registry:            aw.registry,
		})
		if err != nil {
			return fmt.Errorf("failed to create a nexus worker: %w", err)
		}
		if err := aw.nexusWorker.Start(); err != nil {
			return fmt.Errorf("failed to start a nexus worker: %w", err)
		}
	}
	aw.logger.Info("Started Worker")
	return nil
}

func (aw *AggregatedWorker) assertNotStopped() {
	stopped := true
	select {
	case <-aw.stopC:
	default:
		stopped = false
	}
	if stopped {
		panic("attempted to start a worker that has been stopped before")
	}
}

var (
	binaryChecksum     string
	binaryChecksumLock sync.Mutex
)

// SetBinaryChecksum sets the identifier of the binary(aka BinaryChecksum).
// The identifier is mainly used in recording reset points when respondWorkflowTaskCompleted. For each workflow, the very first
// workflow task completed by a binary will be associated as a auto-reset point for the binary. So that when a customer wants to
// mark the binary as bad, the workflow will be reset to that point -- which means workflow will forget all progress generated
// by the binary.
// On another hand, once the binary is marked as bad, the bad binary cannot poll workflow queue and make any progress any more.
func SetBinaryChecksum(checksum string) {
	binaryChecksumLock.Lock()
	defer binaryChecksumLock.Unlock()

	binaryChecksum = checksum
}

func initBinaryChecksum() error {
	binaryChecksumLock.Lock()
	defer binaryChecksumLock.Unlock()

	return initBinaryChecksumLocked()
}

func getBinaryChecksum() string {
	binaryChecksumLock.Lock()
	defer binaryChecksumLock.Unlock()

	if len(binaryChecksum) == 0 {
		err := initBinaryChecksumLocked()
		if err != nil {
			panic(err)
		}
	}

	return binaryChecksum
}

// Run the worker in a blocking fashion. Stop the worker when interruptCh receives signal.
// Pass worker.InterruptCh() to stop the worker with SIGINT or SIGTERM.
// Pass nil to stop the worker with external Stop() call.
// Pass any other `<-chan interface{}` and Run will wait for signal from that channel.
// Returns error if the worker fails to start or there is a fatal error
// during execution.
func (aw *AggregatedWorker) Run(interruptCh <-chan interface{}) error {
	if err := aw.Start(); err != nil {
		return err
	}
	select {
	case s := <-interruptCh:
		aw.logger.Info("Worker has been stopped.", "Signal", s)
		aw.Stop()
	case <-aw.stopC:
		aw.fatalErrLock.Lock()
		defer aw.fatalErrLock.Unlock()
		// This may be nil if this wasn't stopped due to fatal error
		return aw.fatalErr
	}
	return nil
}

// Stop the worker.
func (aw *AggregatedWorker) Stop() {
	// Only attempt stop if we haven't attempted before
	select {
	case <-aw.stopC:
		return
	default:
		close(aw.stopC)
	}

	if !util.IsInterfaceNil(aw.workflowWorker) {
		if aw.client.eagerDispatcher != nil {
			aw.client.eagerDispatcher.deregisterWorker(aw.workflowWorker)
		}
		aw.workflowWorker.Stop()
	}
	if !util.IsInterfaceNil(aw.activityWorker) {
		aw.activityWorker.Stop()
	}
	if !util.IsInterfaceNil(aw.sessionWorker) {
		aw.sessionWorker.Stop()
	}
	if !util.IsInterfaceNil(aw.nexusWorker) {
		aw.nexusWorker.Stop()
	}

	aw.logger.Info("Stopped Worker")
}

// WorkflowReplayer is used to replay workflow code from an event history
type WorkflowReplayer struct {
	registry                 *registry
	dataConverter            converter.DataConverter
	failureConverter         converter.FailureConverter
	contextPropagators       []ContextPropagator
	enableLoggingInReplay    bool
	disableDeadlockDetection bool
	mu                       sync.Mutex
	workflowExecutionResults map[string]*commonpb.Payloads
}

// WorkflowReplayerOptions are options for creating a workflow replayer.
type WorkflowReplayerOptions struct {
	// Optional custom data converter to provide for replay. If not set, the
	// default converter is used.
	DataConverter converter.DataConverter

	FailureConverter converter.FailureConverter

	// Optional: Sets ContextPropagators that allows users to control the context information passed through a workflow
	//
	// default: nil
	ContextPropagators []ContextPropagator

	// Interceptors to apply to the worker. Earlier interceptors wrap later
	// interceptors.
	Interceptors []WorkerInterceptor

	// Disable aliasing during registration. This should be set if it was set on
	// worker.Options.DisableRegistrationAliasing when originally run. See
	// documentation for that field for more information.
	DisableRegistrationAliasing bool

	// Optional: Enable logging in replay.
	// In the workflow code you can use workflow.GetLogger(ctx) to write logs. By default, the logger will skip log
	// entry during replay mode so you won't see duplicate logs. This option will enable the logging in replay mode.
	// This is only useful for debugging purpose.
	//
	// default: false
	EnableLoggingInReplay bool

	// Optional: Disable the default 1 second deadlock detection timeout. This option can be used to step through
	// workflow code with multiple breakpoints in a debugger.
	DisableDeadlockDetection bool
}

// ReplayWorkflowHistoryOptions are options for replaying a workflow.
type ReplayWorkflowHistoryOptions struct {
	// OriginalExecution - Overide the workflow execution details used for replay.
	// Optional
	OriginalExecution WorkflowExecution
}

// NewWorkflowReplayer creates an instance of the WorkflowReplayer.
func NewWorkflowReplayer(options WorkflowReplayerOptions) (*WorkflowReplayer, error) {
	registry := newRegistryWithOptions(registryOptions{disableAliasing: options.DisableRegistrationAliasing})
	registry.interceptors = options.Interceptors
	return &WorkflowReplayer{
		registry:                 registry,
		dataConverter:            options.DataConverter,
		failureConverter:         options.FailureConverter,
		contextPropagators:       options.ContextPropagators,
		enableLoggingInReplay:    options.EnableLoggingInReplay,
		disableDeadlockDetection: options.DisableDeadlockDetection,
		workflowExecutionResults: make(map[string]*commonpb.Payloads),
	}, nil
}

// RegisterWorkflow registers workflow function to replay
func (aw *WorkflowReplayer) RegisterWorkflow(w interface{}) {
	aw.registry.RegisterWorkflow(w)
}

// RegisterWorkflowWithOptions registers workflow function with custom workflow name to replay
func (aw *WorkflowReplayer) RegisterWorkflowWithOptions(w interface{}, options RegisterWorkflowOptions) {
	aw.registry.RegisterWorkflowWithOptions(w, options)
}

// RegisterDynamicWorkflow registers a dynamic workflow function to replay
func (aw *WorkflowReplayer) RegisterDynamicWorkflow(w interface{}, options DynamicRegisterWorkflowOptions) {
	aw.registry.RegisterDynamicWorkflow(w, options)
}

// ReplayWorkflowHistoryWithOptions executes a single workflow task for the given history.
// Use for testing the backwards compatibility of code changes and troubleshooting workflows in a debugger.
// The logger is an optional parameter. Defaults to the noop logger.
func (aw *WorkflowReplayer) ReplayWorkflowHistoryWithOptions(logger log.Logger, history *historypb.History, options ReplayWorkflowHistoryOptions) error {
	if logger == nil {
		logger = ilog.NewDefaultLogger()
	}

	controller := gomock.NewController(ilog.NewTestReporter(logger))
	service := workflowservicemock.NewMockWorkflowServiceClient(controller)

	return aw.replayWorkflowHistory(logger, service, ReplayNamespace, options.OriginalExecution, history)
}

// ReplayWorkflowHistory executes a single workflow task for the given history.
// Use for testing the backwards compatibility of code changes and troubleshooting workflows in a debugger.
// The logger is an optional parameter. Defaults to the noop logger.
func (aw *WorkflowReplayer) ReplayWorkflowHistory(logger log.Logger, history *historypb.History) error {
	return aw.ReplayWorkflowHistoryWithOptions(logger, history, ReplayWorkflowHistoryOptions{})
}

// ReplayWorkflowHistoryFromJSONFile executes a single workflow task for the given json history file.
// Use for testing the backwards compatibility of code changes and troubleshooting workflows in a debugger.
// The logger is an optional parameter. Defaults to the noop logger.
func (aw *WorkflowReplayer) ReplayWorkflowHistoryFromJSONFile(logger log.Logger, jsonfileName string) error {
	return aw.ReplayPartialWorkflowHistoryFromJSONFile(logger, jsonfileName, 0)
}

// ReplayPartialWorkflowHistoryFromJSONFile executes a single workflow task for the given json history file upto provided
// lastEventID(inclusive).
// Use for testing the backwards compatibility of code changes and troubleshooting workflows in a debugger.
// The logger is an optional parameter. Defaults to the noop logger.
func (aw *WorkflowReplayer) ReplayPartialWorkflowHistoryFromJSONFile(logger log.Logger, jsonfileName string, lastEventID int64) error {
	history, err := extractHistoryFromFile(jsonfileName, lastEventID)
	if err != nil {
		return err
	}

	if logger == nil {
		logger = ilog.NewDefaultLogger()
	}

	controller := gomock.NewController(ilog.NewTestReporter(logger))
	service := workflowservicemock.NewMockWorkflowServiceClient(controller)

	return aw.replayWorkflowHistory(logger, service, ReplayNamespace, WorkflowExecution{}, history)
}

// ReplayWorkflowExecution replays workflow execution loading it from Temporal service.
func (aw *WorkflowReplayer) ReplayWorkflowExecution(ctx context.Context, service workflowservice.WorkflowServiceClient, logger log.Logger, namespace string, execution WorkflowExecution) error {
	if logger == nil {
		logger = ilog.NewDefaultLogger()
	}

	sharedExecution := &commonpb.WorkflowExecution{
		RunId:      execution.RunID,
		WorkflowId: execution.ID,
	}
	request := &workflowservice.GetWorkflowExecutionHistoryRequest{
		Namespace: namespace,
		Execution: sharedExecution,
	}
	var history historypb.History
	for {
		resp, err := service.GetWorkflowExecutionHistory(ctx, request)
		if err != nil {
			return err
		}
		currHistory := resp.History
		if resp.RawHistory != nil {
			currHistory, err = serializer.DeserializeBlobDataToHistoryEvents(resp.RawHistory, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
			if err != nil {
				return err
			}
		}
		if currHistory == nil {
			break
		}
		history.Events = append(history.Events, currHistory.Events...)
		if len(resp.NextPageToken) == 0 {
			break
		}
		request.NextPageToken = resp.NextPageToken
	}
	return aw.replayWorkflowHistory(logger, service, namespace, execution, &history)
}

// GetWorkflowResult get the result of a succesfully replayed workflow.
func (aw *WorkflowReplayer) GetWorkflowResult(workflowID string, valuePtr interface{}) error {
	aw.mu.Lock()
	defer aw.mu.Unlock()
	if workflowID == "" {
		workflowID = "ReplayId"
	}
	payloads, ok := aw.workflowExecutionResults[workflowID]
	if !ok {
		return errors.New("workflow result not found")
	}
	dc := aw.dataConverter
	if dc == nil {
		dc = converter.GetDefaultDataConverter()
	}
	return dc.FromPayloads(payloads, valuePtr)
}

func (aw *WorkflowReplayer) replayWorkflowHistory(logger log.Logger, service workflowservice.WorkflowServiceClient, namespace string, originalExecution WorkflowExecution, history *historypb.History) error {
	taskQueue := "ReplayTaskQueue"
	events := history.Events
	if events == nil {
		return errors.New("empty events")
	}
	if len(events) < 3 {
		return errors.New("at least 3 events expected in the history")
	}
	first := events[0]
	if first.GetEventType() != enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED {
		return errors.New("first event is not WorkflowExecutionStarted")
	}
	last := events[len(events)-1]

	attr := first.GetWorkflowExecutionStartedEventAttributes()
	if attr == nil {
		return errors.New("corrupted WorkflowExecutionStarted")
	}
	workflowType := attr.WorkflowType
	execution := &commonpb.WorkflowExecution{
		RunId:      uuid.NewString(),
		WorkflowId: "ReplayId",
	}
	if originalExecution.ID != "" {
		execution.WorkflowId = originalExecution.ID
	}
	if originalExecution.RunID != "" {
		execution.RunId = originalExecution.RunID
	} else if first.GetWorkflowExecutionStartedEventAttributes().GetOriginalExecutionRunId() != "" {
		execution.RunId = first.GetWorkflowExecutionStartedEventAttributes().GetOriginalExecutionRunId()
	}

	if first.GetWorkflowExecutionStartedEventAttributes().GetTaskQueue().GetName() != "" {
		taskQueue = first.GetWorkflowExecutionStartedEventAttributes().GetTaskQueue().GetName()
	}

	task := &workflowservice.PollWorkflowTaskQueueResponse{
		Attempt:                1,
		TaskToken:              []byte("ReplayTaskToken"),
		WorkflowType:           workflowType,
		WorkflowExecution:      execution,
		History:                history,
		PreviousStartedEventId: math.MaxInt64,
	}

	iterator := &historyIteratorImpl{
		nextPageToken: task.NextPageToken,
		execution:     task.WorkflowExecution,
		namespace:     ReplayNamespace,
		service:       service,
		taskQueue:     taskQueue,
	}
	cache := NewWorkerCache()
	params := workerExecutionParameters{
		Namespace:             namespace,
		TaskQueue:             taskQueue,
		Identity:              "replayID",
		Logger:                logger,
		cache:                 cache,
		DataConverter:         aw.dataConverter,
		FailureConverter:      aw.failureConverter,
		ContextPropagators:    aw.contextPropagators,
		EnableLoggingInReplay: aw.enableLoggingInReplay,
		// Hardcoding NopHandler avoids "No metrics handler configured for temporal worker"
		// logs during replay.
		MetricsHandler: metrics.NopHandler,
		capabilities: &workflowservice.GetSystemInfoResponse_Capabilities{
			SignalAndQueryHeader:            true,
			InternalErrorDifferentiation:    true,
			ActivityFailureIncludeHeartbeat: true,
			SupportsSchedules:               true,
			EncodedFailureAttributes:        true,
			UpsertMemo:                      true,
			EagerWorkflowStart:              true,
			SdkMetadata:                     true,
		},
	}
	if aw.disableDeadlockDetection {
		params.DeadlockDetectionTimeout = math.MaxInt64
	}
	taskHandler := newWorkflowTaskHandler(params, nil, aw.registry)
	wfctx, err := taskHandler.GetOrCreateWorkflowContext(task, iterator)
	defer wfctx.Unlock(err)
	if err != nil {
		return err
	}
	resp, err := taskHandler.ProcessWorkflowTask(&workflowTask{task: task, historyIterator: iterator}, wfctx, nil)
	if err != nil {
		return err
	}

	if failedReq, ok := resp.(*workflowservice.RespondWorkflowTaskFailedRequest); ok {
		return fmt.Errorf("replay workflow failed with failure: %v", failedReq.GetFailure())
	}

	if last.GetEventType() != enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED && last.GetEventType() != enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CONTINUED_AS_NEW {
		return nil
	}

	if resp != nil {
		completeReq, ok := resp.(*workflowservice.RespondWorkflowTaskCompletedRequest)
		if ok {
			for _, d := range completeReq.Commands {
				if d.GetCommandType() == enumspb.COMMAND_TYPE_CONTINUE_AS_NEW_WORKFLOW_EXECUTION {
					if last.GetEventType() == enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CONTINUED_AS_NEW {
						return nil
					}
				}
				if d.GetCommandType() == enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION {
					if last.GetEventType() == enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED {
						aw.mu.Lock()
						defer aw.mu.Unlock()
						aw.workflowExecutionResults[execution.WorkflowId] = d.GetCompleteWorkflowExecutionCommandAttributes().Result
						return nil
					}
				}
			}
		}
	}
	return fmt.Errorf("replay workflow doesn't return the same result as the last event, resp: %[1]T{%[1]v}, last: %[2]T{%[2]v}", resp, last)
}

// HistoryFromJSON deserializes history from a reader of JSON bytes. This does
// not close the reader if it is closeable.
func HistoryFromJSON(r io.Reader, lastEventID int64) (*historypb.History, error) {
	// We set DiscardUnknown here because the history may have been created by a previous
	// version of our protos
	opts := temporalproto.CustomJSONUnmarshalOptions{
		DiscardUnknown: true,
	}
	bs, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	hist := &historypb.History{}
	if err := opts.Unmarshal(bs, hist); err != nil {
		return nil, err
	}

	// If there is a last event ID, slice the rest off
	if lastEventID > 0 {
		for i, event := range hist.Events {
			if event.EventId == lastEventID {
				// Inclusive
				hist.Events = hist.Events[:i+1]
				break
			}
		}
	}
	return hist, nil
}

func extractHistoryFromFile(jsonfileName string, lastEventID int64) (hist *historypb.History, err error) {
	reader, err := os.Open(jsonfileName)
	if err != nil {
		return nil, err
	}
	defer func() {
		closeErr := reader.Close()
		if closeErr != nil && err == nil {
			err = closeErr
		} else if closeErr != nil {
			ilog.NewDefaultLogger().Warn("failed to close json file", "path", jsonfileName, "error", closeErr)
		}
	}()

	opts := temporalproto.CustomJSONUnmarshalOptions{
		DiscardUnknown: true,
	}

	bs, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	hist = &historypb.History{}
	if err := opts.Unmarshal(bs, hist); err != nil {
		return nil, err
	}

	// If there is a last event ID, slice the rest off
	if lastEventID > 0 {
		for i, event := range hist.Events {
			if event.EventId == lastEventID {
				// Inclusive
				hist.Events = hist.Events[:i+1]
				break
			}
		}
	}

	return hist, err
}

// NewAggregatedWorker returns an instance to manage both activity and workflow workers
func NewAggregatedWorker(client *WorkflowClient, taskQueue string, options WorkerOptions) *AggregatedWorker {
	if strings.HasPrefix(taskQueue, temporalPrefix) {
		panic(temporalPrefixError)
	}
	setClientDefaults(client)
	setWorkerOptionsDefaults(&options)
	ctx := options.BackgroundActivityContext
	if ctx == nil {
		ctx = context.Background()
	}
	backgroundActivityContext, backgroundActivityContextCancel := context.WithCancelCause(ctx)

	// If max-concurrent workflow pollers is 1, the worker will only do
	// sticky-queue requests and never regular-queue requests. We disallow the
	// value of 1 here.
	if options.MaxConcurrentWorkflowTaskPollers == 1 {
		panic("cannot set MaxConcurrentWorkflowTaskPollers to 1")
	}

	// If max-concurrent workflow task execution size is 1, the worker will only do
	// sticky-queue requests and never regular-queue requests. This is because we
	// limit the number of running pollers to MaxConcurrentWorkflowTaskExecutionSize.
	// 	We disallow the value of 1 here.
	if options.MaxConcurrentWorkflowTaskExecutionSize == 1 {
		panic("cannot set MaxConcurrentWorkflowTaskExecutionSize to 1")
	}

	// Sessions are not currently compatible with worker versioning
	// See: https://github.com/temporalio/sdk-go/issues/1227
	if options.EnableSessionWorker && options.UseBuildIDForVersioning {
		panic("cannot set both EnableSessionWorker and UseBuildIDForVersioning")
	}

	if (options.DeploymentOptions.Version != WorkerDeploymentVersion{}) {
		options.BuildID = options.DeploymentOptions.Version.BuildID
	}
	if !options.DeploymentOptions.UseVersioning &&
		options.DeploymentOptions.DefaultVersioningBehavior != VersioningBehaviorUnspecified {
		panic("cannot set both DeploymentOptions.DefaultVersioningBehavior if DeploymentOptions.UseBuildIDForVersioning is false")
	}

	// Need reference to result for fatal error handler
	var aw *AggregatedWorker
	fatalErrorCallback := func(err error) {
		// Set the fatal error if not already set
		aw.fatalErrLock.Lock()
		alreadySet := aw.fatalErr != nil
		if !alreadySet {
			aw.fatalErr = err
		}
		aw.fatalErrLock.Unlock()
		// Only do the rest if not already set
		if !alreadySet {
			// Invoke the callback if present
			if options.OnFatalError != nil {
				options.OnFatalError(err)
			}
			// Stop the worker if not already stopped
			select {
			case <-aw.stopC:
			default:
				aw.Stop()
			}
		}
	}
	// Because of lazy clients we need to wait till the worker runs to fetch the capabilities.
	// All worker systems that depend on the capabilities to process workflow/activity tasks
	// should take a pointer to this struct and wait for it to be populated when the worker is run.
	var capabilities workflowservice.GetSystemInfoResponse_Capabilities
	workerDeploymentVersion := WorkerDeploymentVersion{}
	if (options.DeploymentOptions.Version != WorkerDeploymentVersion{}) {
		workerDeploymentVersion = options.DeploymentOptions.Version
	}

	cache := NewWorkerCache()
	workerParams := workerExecutionParameters{
		Namespace:                        client.namespace,
		TaskQueue:                        taskQueue,
		Tuner:                            options.Tuner,
		WorkerActivitiesPerSecond:        options.WorkerActivitiesPerSecond,
		WorkerLocalActivitiesPerSecond:   options.WorkerLocalActivitiesPerSecond,
		Identity:                         client.identity,
		WorkerBuildID:                    options.BuildID,
		UseBuildIDForVersioning:          options.UseBuildIDForVersioning || options.DeploymentOptions.UseVersioning,
		WorkerDeploymentVersion:          workerDeploymentVersion,
		DefaultVersioningBehavior:        options.DeploymentOptions.DefaultVersioningBehavior,
		MetricsHandler:                   client.metricsHandler.WithTags(metrics.TaskQueueTags(taskQueue)),
		Logger:                           client.logger,
		EnableLoggingInReplay:            options.EnableLoggingInReplay,
		BackgroundContext:                backgroundActivityContext,
		BackgroundContextCancel:          backgroundActivityContextCancel,
		StickyScheduleToStartTimeout:     options.StickyScheduleToStartTimeout,
		TaskQueueActivitiesPerSecond:     options.TaskQueueActivitiesPerSecond,
		WorkflowPanicPolicy:              options.WorkflowPanicPolicy,
		DataConverter:                    client.dataConverter,
		FailureConverter:                 client.failureConverter,
		WorkerStopTimeout:                options.WorkerStopTimeout,
		WorkerFatalErrorCallback:         fatalErrorCallback,
		ContextPropagators:               client.contextPropagators,
		DeadlockDetectionTimeout:         options.DeadlockDetectionTimeout,
		DefaultHeartbeatThrottleInterval: options.DefaultHeartbeatThrottleInterval,
		MaxHeartbeatThrottleInterval:     options.MaxHeartbeatThrottleInterval,
		cache:                            cache,
		eagerActivityExecutor: newEagerActivityExecutor(eagerActivityExecutorOptions{
			disabled:      options.DisableEagerActivities,
			taskQueue:     taskQueue,
			maxConcurrent: options.MaxConcurrentEagerActivityExecutionSize,
		}),
		capabilities: &capabilities,
	}

	if options.MaxConcurrentWorkflowTaskPollers != 0 {
		workerParams.WorkflowTaskPollerBehavior = NewPollerBehaviorSimpleMaximum(PollerBehaviorSimpleMaximumOptions{
			MaximumNumberOfPollers: options.MaxConcurrentWorkflowTaskPollers,
		})
	} else if options.WorkflowTaskPollerBehavior != nil {
		workerParams.WorkflowTaskPollerBehavior = options.WorkflowTaskPollerBehavior
	} else {
		panic("must set either MaxConcurrentWorkflowTaskPollers or WorkflowTaskPollerBehavior")
	}

	if options.MaxConcurrentActivityTaskPollers != 0 {
		workerParams.ActivityTaskPollerBehavior = NewPollerBehaviorSimpleMaximum(PollerBehaviorSimpleMaximumOptions{
			MaximumNumberOfPollers: options.MaxConcurrentActivityTaskPollers,
		})
	} else if options.ActivityTaskPollerBehavior != nil {
		workerParams.ActivityTaskPollerBehavior = options.ActivityTaskPollerBehavior
	} else {
		panic("must set either MaxConcurrentActivityTaskPollers or ActivityTaskPollerBehavior")
	}

	if options.MaxConcurrentNexusTaskPollers != 0 {
		workerParams.NexusTaskPollerBehavior = NewPollerBehaviorSimpleMaximum(PollerBehaviorSimpleMaximumOptions{
			MaximumNumberOfPollers: options.MaxConcurrentNexusTaskPollers,
		})
	} else if options.NexusTaskPollerBehavior != nil {
		workerParams.NexusTaskPollerBehavior = options.NexusTaskPollerBehavior
	} else {
		panic("must set either MaxConcurrentNexusTaskPollers or NexusTaskPollerBehavior")
	}

	if options.Identity != "" {
		workerParams.Identity = options.Identity
	}

	ensureRequiredParams(&workerParams)
	workerParams.Logger = log.With(workerParams.Logger,
		tagNamespace, client.namespace,
		tagTaskQueue, taskQueue,
		tagWorkerID, workerParams.Identity,
	)
	if workerParams.WorkerBuildID != "" {
		// Add worker build ID to the logs if it's set by user
		workerParams.Logger = log.With(workerParams.Logger,
			tagBuildID, workerParams.WorkerBuildID,
		)
	}

	processTestTags(&options, &workerParams)

	// worker specific registry
	registry := newRegistryWithOptions(registryOptions{disableAliasing: options.DisableRegistrationAliasing})
	// Build set of interceptors using the applicable client ones first (being
	// careful not to append to the existing slice)
	registry.interceptors = make([]WorkerInterceptor, 0, len(client.workerInterceptors)+len(options.Interceptors))
	registry.interceptors = append(append(registry.interceptors, client.workerInterceptors...), options.Interceptors...)

	// workflow factory.
	var workflowWorker *workflowWorker
	if !options.DisableWorkflowWorker {
		testTags := getTestTags(options.BackgroundActivityContext)
		if len(testTags) > 0 {
			workflowWorker = newWorkflowWorkerWithPressurePoints(client, workerParams, testTags, registry)
		} else {
			workflowWorker = newWorkflowWorker(client, workerParams, nil, registry)
		}
	}

	// activity types.
	var activityWorker *activityWorker
	if !options.LocalActivityWorkerOnly {
		activityWorker = newActivityWorker(client, workerParams, nil, registry, nil)
		workerParams.eagerActivityExecutor.activityWorker = activityWorker.worker
	}

	var sessionWorker *sessionWorker
	if options.EnableSessionWorker && !options.LocalActivityWorkerOnly {
		sessionWorker = newSessionWorker(client, workerParams, registry, options.MaxConcurrentSessionExecutionSize)
		registry.RegisterActivityWithOptions(sessionCreationActivity, RegisterActivityOptions{
			Name: sessionCreationActivityName,
		})
		registry.RegisterActivityWithOptions(sessionCompletionActivity, RegisterActivityOptions{
			Name: sessionCompletionActivityName,
		})
	}

	aw = &AggregatedWorker{
		client:          client,
		workflowWorker:  workflowWorker,
		activityWorker:  activityWorker,
		sessionWorker:   sessionWorker,
		logger:          workerParams.Logger,
		registry:        registry,
		stopC:           make(chan struct{}),
		capabilities:    &capabilities,
		executionParams: workerParams,
	}
	aw.memoizedStart = sync.OnceValue(aw.start)
	return aw
}

func processTestTags(wOptions *WorkerOptions, ep *workerExecutionParameters) {
	testTags := getTestTags(wOptions.BackgroundActivityContext)
	if testTags != nil {
		if paramsOverride, ok := testTags[workerOptionsConfig]; ok {
			for key, val := range paramsOverride {
				switch key {
				case workerOptionsConfigConcurrentPollRoutineSize:
					if size, err := strconv.Atoi(val); err == nil {
						ep.ActivityTaskPollerBehavior = NewPollerBehaviorSimpleMaximum(
							PollerBehaviorSimpleMaximumOptions{
								MaximumNumberOfPollers: size,
							},
						)
						ep.WorkflowTaskPollerBehavior = NewPollerBehaviorSimpleMaximum(
							PollerBehaviorSimpleMaximumOptions{
								MaximumNumberOfPollers: size,
							},
						)
					}
				}
			}
		}
	}
}

func isWorkflowContext(inType reflect.Type) bool {
	// NOTE: We don't expect any one to derive from workflow context.
	return inType == reflect.TypeOf((*Context)(nil)).Elem()
}

func isValidResultType(inType reflect.Type) bool {
	// https://golang.org/pkg/reflect/#Kind
	switch inType.Kind() {
	case reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return false
	}

	return true
}

func isError(inType reflect.Type) bool {
	errorElem := reflect.TypeOf((*error)(nil)).Elem()
	return inType != nil && inType.Implements(errorElem)
}

func getFunctionName(i interface{}) (name string, isMethod bool) {
	if fullName, ok := i.(string); ok {
		return fullName, false
	}
	fullName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	// Full function name that has a struct pointer receiver has the following format
	// <prefix>.(*<type>).<function>
	isMethod = strings.ContainsAny(fullName, "*")
	elements := strings.Split(fullName, ".")
	shortName := elements[len(elements)-1]
	// This allows to call activities by method pointer
	// Compiler adds -fm suffix to a function name which has a receiver
	// Note that this works even if struct pointer used to get the function is nil
	// It is possible because nil receivers are allowed.
	// For example:
	// var a *Activities
	// ExecuteActivity(ctx, a.Foo)
	// will call this function which is going to return "Foo"
	return strings.TrimSuffix(shortName, "-fm"), isMethod
}

func getActivityFunctionName(r *registry, i interface{}) string {
	result, _ := getFunctionName(i)
	if alias, ok := r.getActivityAlias(result); ok {
		result = alias
	}
	return result
}

func getWorkflowFunctionName(r *registry, workflowFunc interface{}) (string, error) {
	fnName := ""
	fType := reflect.TypeOf(workflowFunc)
	switch getKind(fType) {
	case reflect.String:
		fnName = reflect.ValueOf(workflowFunc).String()
	case reflect.Func:
		fnName, _ = getFunctionName(workflowFunc)
		if alias, ok := r.getWorkflowAlias(fnName); ok {
			fnName = alias
		}
	default:
		return "", fmt.Errorf("invalid type 'workflowFunc' parameter provided, it can be either worker function or function name: %v", workflowFunc)
	}

	return fnName, nil
}

func getReadOnlyChannel(c chan struct{}) <-chan struct{} {
	return c
}

func setWorkerOptionsDefaults(options *WorkerOptions) {
	if options.Tuner != nil {
		if options.MaxConcurrentWorkflowTaskExecutionSize != 0 ||
			options.MaxConcurrentActivityExecutionSize != 0 ||
			options.MaxConcurrentLocalActivityExecutionSize != 0 ||
			options.MaxConcurrentNexusTaskExecutionSize != 0 {
			panic("cannot set MaxConcurrentWorkflowTaskExecutionSize, MaxConcurrentActivityExecutionSize, MaxConcurrentLocalActivityExecutionSize, or MaxConcurrentNexusTaskExecutionSize with Tuner")
		}
	}
	maxConcurrentWFT := options.MaxConcurrentWorkflowTaskExecutionSize
	maxConcurrentAct := options.MaxConcurrentActivityExecutionSize
	maxConcurrentLA := options.MaxConcurrentLocalActivityExecutionSize
	maxConcurrentNexus := options.MaxConcurrentNexusTaskExecutionSize
	if options.MaxConcurrentActivityExecutionSize <= 0 {
		maxConcurrentAct = defaultMaxConcurrentActivityExecutionSize
	}
	if options.WorkerActivitiesPerSecond == 0 {
		options.WorkerActivitiesPerSecond = defaultWorkerActivitiesPerSecond
	}
	if options.MaxConcurrentActivityTaskPollers != 0 && options.ActivityTaskPollerBehavior != nil {
		panic("cannot set both MaxConcurrentActivityTaskPollers and ActivityTaskPollerBehavior")
	} else if options.ActivityTaskPollerBehavior == nil && options.MaxConcurrentActivityTaskPollers <= 0 {
		options.MaxConcurrentActivityTaskPollers = defaultConcurrentPollRoutineSize
	}
	if options.MaxConcurrentWorkflowTaskExecutionSize <= 0 {
		maxConcurrentWFT = defaultMaxConcurrentTaskExecutionSize
	}
	if options.MaxConcurrentWorkflowTaskPollers != 0 && options.WorkflowTaskPollerBehavior != nil {
		panic("cannot set both MaxConcurrentWorkflowTaskPollers and WorkflowTaskPollerBehavior")
	} else if options.WorkflowTaskPollerBehavior == nil && options.MaxConcurrentWorkflowTaskPollers <= 0 {
		options.MaxConcurrentWorkflowTaskPollers = defaultConcurrentPollRoutineSize
	}
	if options.MaxConcurrentLocalActivityExecutionSize <= 0 {
		maxConcurrentLA = defaultMaxConcurrentLocalActivityExecutionSize
	}
	if options.WorkerLocalActivitiesPerSecond == 0 {
		options.WorkerLocalActivitiesPerSecond = defaultWorkerLocalActivitiesPerSecond
	}
	if options.TaskQueueActivitiesPerSecond == 0 {
		options.TaskQueueActivitiesPerSecond = defaultTaskQueueActivitiesPerSecond
	} else {
		// Disable eager activities when the task queue rate limit is set because
		// the server does not rate limit eager activities.
		options.DisableEagerActivities = true
	}
	if options.MaxConcurrentNexusTaskPollers != 0 && options.NexusTaskPollerBehavior != nil {
		panic("cannot set both MaxConcurrentNexusTaskExecutionSize and NexusTaskPollerBehavior")
	} else if options.NexusTaskPollerBehavior == nil && options.MaxConcurrentNexusTaskPollers <= 0 {
		options.MaxConcurrentNexusTaskPollers = defaultConcurrentPollRoutineSize
	}
	if options.MaxConcurrentNexusTaskExecutionSize <= 0 {
		maxConcurrentNexus = defaultMaxConcurrentTaskExecutionSize
	}
	if options.StickyScheduleToStartTimeout.Seconds() == 0 {
		options.StickyScheduleToStartTimeout = stickyWorkflowTaskScheduleToStartTimeoutSeconds * time.Second
	}
	if options.MaxConcurrentSessionExecutionSize == 0 {
		options.MaxConcurrentSessionExecutionSize = defaultMaxConcurrentSessionExecutionSize
	}
	if options.DeadlockDetectionTimeout == 0 {
		if debugMode {
			options.DeadlockDetectionTimeout = unlimitedDeadlockDetectionTimeout
		} else {
			options.DeadlockDetectionTimeout = defaultDeadlockDetectionTimeout
		}
	}
	if options.DefaultHeartbeatThrottleInterval == 0 {
		options.DefaultHeartbeatThrottleInterval = defaultDefaultHeartbeatThrottleInterval
	}
	if options.MaxHeartbeatThrottleInterval == 0 {
		options.MaxHeartbeatThrottleInterval = defaultMaxHeartbeatThrottleInterval
	}
	if options.Tuner == nil {
		// Err cannot happen since these slot numbers are guaranteed valid
		options.Tuner, _ = NewFixedSizeTuner(FixedSizeTunerOptions{
			NumWorkflowSlots:      maxConcurrentWFT,
			NumActivitySlots:      maxConcurrentAct,
			NumLocalActivitySlots: maxConcurrentLA,
			NumNexusSlots:         maxConcurrentNexus})

	}
}

// setClientDefaults should be needed only in unit tests.
func setClientDefaults(client *WorkflowClient) {
	if client.dataConverter == nil {
		client.dataConverter = converter.GetDefaultDataConverter()
	}
	if client.namespace == "" {
		client.namespace = DefaultNamespace
	}
	if client.metricsHandler == nil {
		client.metricsHandler = metrics.NopHandler
	}
}

// getTestTags returns the test tags in the context.
func getTestTags(ctx context.Context) map[string]map[string]string {
	if ctx != nil {
		env := ctx.Value(testTagsContextKey)
		if env != nil {
			return env.(map[string]map[string]string)
		}
	}
	return nil
}

// Same as executeFunction but injects the workflow context as the first
// parameter if the function takes it (regardless of existing parameters).
func executeFunctionWithWorkflowContext(ctx Context, fn interface{}, args []interface{}) (interface{}, error) {
	if fnType := reflect.TypeOf(fn); fnType.NumIn() > 0 && isWorkflowContext(fnType.In(0)) {
		args = append([]interface{}{ctx}, args...)
	}
	return executeFunction(fn, args)
}

// Same as executeFunction but injects the context as the first parameter if the
// function takes it (regardless of existing parameters).
func executeFunctionWithContext(ctx context.Context, fn interface{}, args []interface{}) (interface{}, error) {
	if fnType := reflect.TypeOf(fn); fnType.NumIn() > 0 && isActivityContext(fnType.In(0)) {
		args = append([]interface{}{ctx}, args...)
	}
	return executeFunction(fn, args)
}

// Executes function and ensures that there is always 1 or 2 results and second
// result is error.
func executeFunction(fn interface{}, args []interface{}) (interface{}, error) {
	fnValue := reflect.ValueOf(fn)
	reflectArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		// If the argument is nil, use zero value
		if arg == nil {
			reflectArgs[i] = reflect.New(fnValue.Type().In(i)).Elem()
		} else {
			reflectArgs[i] = reflect.ValueOf(arg)
		}
	}
	retValues := fnValue.Call(reflectArgs)

	// Expect either error or (result, error)
	if len(retValues) == 0 || len(retValues) > 2 {
		fnName, _ := getFunctionName(fn)
		return nil, fmt.Errorf(
			"the function: %v signature returns %d results, it is expecting to return either error or (result, error)",
			fnName, len(retValues))
	}
	// Convert error
	var err error
	if errResult := retValues[len(retValues)-1].Interface(); errResult != nil {
		var ok bool
		if err, ok = errResult.(error); !ok {
			return nil, fmt.Errorf(
				"failed to serialize error result as it is not of error interface: %v",
				errResult)
		}
	}
	// If there are two results, convert the first only if it's not a nil pointer
	var res interface{}
	if len(retValues) > 1 && (retValues[0].Kind() != reflect.Ptr || !retValues[0].IsNil()) {
		res = retValues[0].Interface()
	}
	return res, err
}

func workerDeploymentVersionFromProto(wd *deploymentpb.WorkerDeploymentVersion) WorkerDeploymentVersion {
	return WorkerDeploymentVersion{
		DeploymentName: wd.DeploymentName,
		BuildID:        wd.BuildId,
	}
}

func (wd *WorkerDeploymentVersion) toProto() *deploymentpb.WorkerDeploymentVersion {
	return &deploymentpb.WorkerDeploymentVersion{
		DeploymentName: wd.DeploymentName,
		BuildId:        wd.BuildID,
	}
}

func (wd *WorkerDeploymentVersion) toCanonicalString() string {
	return fmt.Sprintf("%s.%s", wd.DeploymentName, wd.BuildID)
}

func workerDeploymentVersionFromString(version string) *WorkerDeploymentVersion {
	if splitVersion := strings.SplitN(version, ".", 2); len(splitVersion) == 2 {
		return &WorkerDeploymentVersion{
			DeploymentName: splitVersion[0],
			BuildID:        splitVersion[1],
		}
	}
	return nil
}

func workerDeploymentVersionFromProtoOrString(wd *deploymentpb.WorkerDeploymentVersion, fallback string) *WorkerDeploymentVersion {
	if wd == nil {
		return workerDeploymentVersionFromString(fallback)
	}
	return &WorkerDeploymentVersion{
		DeploymentName: wd.DeploymentName,
		BuildID:        wd.BuildId,
	}
}
