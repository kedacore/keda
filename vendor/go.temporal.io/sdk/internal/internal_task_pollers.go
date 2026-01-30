package internal

// All code in this file is private to the package.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.temporal.io/sdk/internal/common/retry"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/google/uuid"

	commonpb "go.temporal.io/api/common/v1"
	deploymentpb "go.temporal.io/api/deployment/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/internal/common/serializer"
	"go.temporal.io/sdk/log"
)

const (
	// Server returns empty task after dynamicconfig.MatchingLongPollExpirationInterval (default is 60 seconds).
	// pollTaskServiceTimeOut should be dynamicconfig.MatchingLongPollExpirationInterval + some delta for full round trip to matching
	// because empty task should be returned before timeout is expired (expired timeout counts against SLO).
	pollTaskServiceTimeOut = 70 * time.Second

	stickyWorkflowTaskScheduleToStartTimeoutSeconds = 5

	ratioToForceCompleteWorkflowTaskComplete = 0.8
)

type workflowTaskPollerMode int

const (
	Mixed workflowTaskPollerMode = iota
	NonSticky
	Sticky
)

type (
	// taskPoller interface to poll for tasks
	taskPoller interface {
		// PollTask polls for one new task
		PollTask() (taskForWorker, error)
		// Called when the poller will no longer be polled. Presently only useful for
		// workflow workers.
		Cleanup() error
	}

	// taskProcessor interface to process tasks
	taskProcessor interface {
		// ProcessTask processes a task
		ProcessTask(interface{}) error
	}

	pollerScaleDecision struct {
		pollRequestDeltaSuggestion int
	}

	taskForWorker interface {
		scaleDecision() (pollerScaleDecision, bool)
		isEmpty() bool
	}

	// basePoller is the base class for all poller implementations
	basePoller struct {
		metricsHandler metrics.Handler // base metric handler used for rpc calls
		stopC          <-chan struct{}
		// The worker's build ID, either as defined by the user or automatically set
		workerBuildID string
		// Whether the worker has opted in to the build-id based versioning feature
		useBuildIDVersioning bool
		// The worker's deployment version identifier.
		workerDeploymentVersion WorkerDeploymentVersion
		// Server's capabilities
		capabilities *workflowservice.GetSystemInfoResponse_Capabilities
	}

	// numPollerMetric tracks the number of active pollers and publishes a metric on it.
	numPollerMetric struct {
		lock       sync.Mutex
		numPollers int32
		gauge      metrics.Gauge
	}

	workflowTaskPoller struct {
		basePoller
		mode             workflowTaskPollerMode
		namespace        string
		taskQueueName    string
		identity         string
		service          workflowservice.WorkflowServiceClient
		taskHandler      WorkflowTaskHandler
		contextManager   WorkflowContextManager
		logger           log.Logger
		dataConverter    converter.DataConverter
		failureConverter converter.FailureConverter

		stickyUUID                   string
		StickyScheduleToStartTimeout time.Duration

		pendingRegularPollCount int
		pendingStickyPollCount  int
		stickyBacklog           int64
		requestLock             sync.Mutex
		stickyCacheSize         int
		eagerActivityExecutor   *eagerActivityExecutor

		numNormalPollerMetric *numPollerMetric
		numStickyPollerMetric *numPollerMetric
	}

	// workflowTaskProcessor implements processing of a workflow task and can create
	// workflow task pollers
	workflowTaskProcessor struct {
		basePoller
		namespace        string
		taskQueueName    string
		identity         string
		service          workflowservice.WorkflowServiceClient
		taskHandler      WorkflowTaskHandler
		contextManager   WorkflowContextManager
		logger           log.Logger
		dataConverter    converter.DataConverter
		failureConverter converter.FailureConverter

		stickyUUID                   string
		StickyScheduleToStartTimeout time.Duration

		pendingRegularPollCount int
		pendingStickyPollCount  int
		stickyBacklog           int64
		stickyCacheSize         int
		eagerActivityExecutor   *eagerActivityExecutor

		numNormalPollerMetric *numPollerMetric
		numStickyPollerMetric *numPollerMetric
	}

	// activityTaskPoller implements polling/processing a workflow task
	activityTaskPoller struct {
		basePoller
		namespace           string
		taskQueueName       string
		identity            string
		service             workflowservice.WorkflowServiceClient
		taskHandler         ActivityTaskHandler
		logger              log.Logger
		activitiesPerSecond float64
		numPollerMetric     *numPollerMetric
	}

	historyIteratorImpl struct {
		iteratorFunc  func(nextPageToken []byte) (*historypb.History, []byte, error)
		execution     *commonpb.WorkflowExecution
		nextPageToken []byte
		namespace     string
		service       workflowservice.WorkflowServiceClient
		// maxEventID is the maximum eventID that the history iterator is expected to return.
		// 0 means that the iterator will return all history events.
		maxEventID     int64
		metricsHandler metrics.Handler
		taskQueue      string
	}

	localActivityTaskPoller struct {
		basePoller
		handler      *localActivityTaskHandler
		logger       log.Logger
		laTunnel     *localActivityTunnel
		workerStopCh <-chan struct{}
	}

	localActivityTaskHandler struct {
		backgroundContext  context.Context
		metricsHandler     metrics.Handler
		logger             log.Logger
		dataConverter      converter.DataConverter
		contextPropagators []ContextPropagator
		interceptors       []WorkerInterceptor
		client             *WorkflowClient
		workerStopChannel  <-chan struct{}
	}

	localActivityResult struct {
		result  *commonpb.Payloads
		err     error
		task    *localActivityTask
		backoff time.Duration
	}

	localActivityTunnel struct {
		taskCh   chan *localActivityTask
		resultCh chan eagerOrPolledTask
		stopCh   <-chan struct{}
	}
)

func newNumPollerMetric(metricsHandler metrics.Handler, pollerType string) *numPollerMetric {
	return &numPollerMetric{
		gauge: metricsHandler.WithTags(metrics.PollerTags(pollerType)).Gauge(metrics.NumPoller),
	}
}

func (npm *numPollerMetric) increment() {
	npm.lock.Lock()
	defer npm.lock.Unlock()
	npm.numPollers += 1
	npm.gauge.Update(float64(npm.numPollers))
}

func (npm *numPollerMetric) decrement() {
	npm.lock.Lock()
	defer npm.lock.Unlock()
	npm.numPollers -= 1
	npm.gauge.Update(float64(npm.numPollers))
}

func newLocalActivityTunnel(stopCh <-chan struct{}) *localActivityTunnel {
	return &localActivityTunnel{
		taskCh:   make(chan *localActivityTask, 100000),
		resultCh: make(chan eagerOrPolledTask),
		stopCh:   stopCh,
	}
}

func (lat *localActivityTunnel) getTask() *localActivityTask {
	select {
	case task := <-lat.taskCh:
		return task
	case <-lat.stopCh:
		return nil
	}
}

func (lat *localActivityTunnel) sendTask(task *localActivityTask) bool {
	select {
	case lat.taskCh <- task:
		return true
	case <-lat.stopCh:
		return false
	}
}

func isClientSideError(err error) bool {
	// If an activity execution exceeds deadline.
	return err == context.DeadlineExceeded
}

// stopping returns true if worker is stopping right now
func (bp *basePoller) stopping() bool {
	select {
	case <-bp.stopC:
		return true
	default:
		return false
	}
}

// doPoll runs the given pollFunc in a separate go routine. Returns when any of the conditions are met:
//   - poll succeeds
//   - poll fails
//   - worker is stopping
func (bp *basePoller) doPoll(pollFunc func(ctx context.Context) (taskForWorker, error)) (taskForWorker, error) {
	if bp.stopping() {
		return nil, errStop
	}

	var err error
	var result taskForWorker

	doneC := make(chan struct{})
	ctx, cancel := newGRPCContext(context.Background(), grpcTimeout(pollTaskServiceTimeOut), grpcLongPoll(true))

	go func() {
		result, err = pollFunc(ctx)
		cancel()
		close(doneC)
	}()

	select {
	case <-doneC:
		return result, err
	case <-bp.stopC:
		cancel()
		return nil, errStop
	}
}

func (bp *basePoller) getCapabilities() *workflowservice.GetSystemInfoResponse_Capabilities {
	if bp.capabilities == nil {
		return &workflowservice.GetSystemInfoResponse_Capabilities{}
	}
	return bp.capabilities
}

func (bp *basePoller) getDeploymentName() string {
	return bp.workerDeploymentVersion.DeploymentName
}

// newWorkflowTaskProcessor creates a new workflow task poller which must have a one to one relationship to workflow worker
func newWorkflowTaskProcessor(
	taskHandler WorkflowTaskHandler,
	contextManager WorkflowContextManager,
	service workflowservice.WorkflowServiceClient,
	params workerExecutionParameters,
) *workflowTaskProcessor {
	return &workflowTaskProcessor{
		basePoller: basePoller{
			metricsHandler:          params.MetricsHandler,
			stopC:                   params.WorkerStopChannel,
			workerBuildID:           params.getBuildID(),
			useBuildIDVersioning:    params.UseBuildIDForVersioning,
			workerDeploymentVersion: params.DeploymentOptions.Version,
			capabilities:            params.capabilities,
		},
		service:                      service,
		namespace:                    params.Namespace,
		taskQueueName:                params.TaskQueue,
		identity:                     params.Identity,
		taskHandler:                  taskHandler,
		contextManager:               contextManager,
		logger:                       params.Logger,
		dataConverter:                params.DataConverter,
		failureConverter:             params.FailureConverter,
		stickyUUID:                   uuid.NewString(),
		StickyScheduleToStartTimeout: params.StickyScheduleToStartTimeout,
		stickyCacheSize:              params.cache.MaxWorkflowCacheSize(),
		eagerActivityExecutor:        params.eagerActivityExecutor,
		numNormalPollerMetric:        newNumPollerMetric(params.MetricsHandler, metrics.PollerTypeWorkflowTask),
		numStickyPollerMetric:        newNumPollerMetric(params.MetricsHandler, metrics.PollerTypeWorkflowStickyTask),
	}
}

// Best-effort attempt to indicate to Matching service that this workflow task
// poller's sticky queue will no longer be polled. Should be called when the
// poller is stopping. Failure to call ShutdownWorker is logged, but otherwise
// ignored.
func (wtp *workflowTaskPoller) Cleanup() error {
	ctx := context.Background()
	grpcCtx, cancel := newGRPCContext(ctx, grpcMetricsHandler(wtp.metricsHandler))
	defer cancel()

	_, err := wtp.service.ShutdownWorker(grpcCtx, &workflowservice.ShutdownWorkerRequest{
		Namespace:       wtp.namespace,
		StickyTaskQueue: getWorkerTaskQueue(wtp.stickyUUID),
		Identity:        wtp.identity,
		Reason:          "graceful shutdown",
	})

	// we ignore unimplemented
	if _, isUnimplemented := err.(*serviceerror.Unimplemented); isUnimplemented {
		return nil
	}

	if err != nil {
		traceLog(func() {
			wtp.logger.Debug("ShutdownWorker failed.", tagError, err)
		})
	}

	return err
}

// PollTask polls a new task
func (wtp *workflowTaskPoller) PollTask() (taskForWorker, error) {
	// Get the task.
	workflowTask, err := wtp.doPoll(wtp.poll)
	if err != nil {
		return nil, err
	}

	return workflowTask, nil
}

func (wtp *workflowTaskProcessor) createPoller(mode workflowTaskPollerMode) taskPoller {
	return &workflowTaskPoller{
		basePoller:                   wtp.basePoller,
		mode:                         mode,
		namespace:                    wtp.namespace,
		taskQueueName:                wtp.taskQueueName,
		identity:                     wtp.identity,
		service:                      wtp.service,
		taskHandler:                  wtp.taskHandler,
		contextManager:               wtp.contextManager,
		logger:                       wtp.logger,
		dataConverter:                wtp.dataConverter,
		failureConverter:             wtp.failureConverter,
		stickyUUID:                   wtp.stickyUUID,
		StickyScheduleToStartTimeout: wtp.StickyScheduleToStartTimeout,
		pendingRegularPollCount:      wtp.pendingRegularPollCount,
		pendingStickyPollCount:       wtp.pendingStickyPollCount,
		stickyBacklog:                wtp.stickyBacklog,
		stickyCacheSize:              wtp.stickyCacheSize,
		eagerActivityExecutor:        wtp.eagerActivityExecutor,
		numNormalPollerMetric:        wtp.numNormalPollerMetric,
		numStickyPollerMetric:        wtp.numStickyPollerMetric,
	}
}

// ProcessTask processes a task which could be workflow task or local activity result
func (wtp *workflowTaskProcessor) ProcessTask(task interface{}) error {
	if wtp.stopping() {
		return errStop
	}

	switch task := task.(type) {
	case *workflowTask:
		return wtp.processWorkflowTask(task)
	case *eagerWorkflowTask:
		return wtp.processWorkflowTask(wtp.toWorkflowTask(task.task))
	default:
		panic("unknown task type.")
	}
}

func (wtp *workflowTaskProcessor) processWorkflowTask(task *workflowTask) (retErr error) {
	if task.task == nil {
		// We didn't have task, poll might have timeout.
		traceLog(func() {
			wtp.logger.Debug("Workflow task unavailable")
		})
		return nil
	}

	doneCh := make(chan struct{})
	laResultCh := make(chan *localActivityResult)
	laRetryCh := make(chan *localActivityTask)
	// close doneCh so local activity worker won't get blocked forever when trying to send back result to laResultCh.
	defer close(doneCh)

	wfctx, err := wtp.contextManager.GetOrCreateWorkflowContext(task.task, task.historyIterator)
	if err != nil {
		return err
	}
	var taskErr error
	defer func() {
		// If we panic during processing the workflow task, we need to unlock the workflow context with an error to discard it.
		if p := recover(); p != nil {
			topLine := fmt.Sprintf("workflow task for %s [panic]:", wtp.taskQueueName)
			st := getStackTraceRaw(topLine, 7, 0)
			wtp.logger.Error("Workflow task processing panic.",
				tagWorkflowID, task.task.WorkflowExecution.GetWorkflowId(),
				tagRunID, task.task.WorkflowExecution.GetRunId(),
				tagWorkerType, task.task.GetWorkflowType().Name,
				tagAttempt, task.task.Attempt,
				tagPanicError, fmt.Sprintf("%v", p),
				tagPanicStack, st)
			taskErr = newPanicError(p, st)
			retErr = taskErr
		}
		wfctx.Unlock(taskErr)
	}()

	for {
		startTime := time.Now()
		task.doneCh = doneCh
		task.laResultCh = laResultCh
		task.laRetryCh = laRetryCh
		var taskCompletion *workflowTaskCompletion
		taskCompletion, taskErr = wtp.taskHandler.ProcessWorkflowTask(
			task,
			wfctx,
			func(taskCompletion *workflowTaskCompletion, startTime time.Time) (*workflowTask, error) {
				wtp.logger.Debug("Force RespondWorkflowTaskCompleted.", "TaskStartedEventID", task.task.GetStartedEventId())
				heartbeatResponse, err := wtp.RespondTaskCompletedWithMetrics(taskCompletion, nil, task.task, startTime)
				if err != nil {
					return nil, err
				}
				if heartbeatResponse == nil || heartbeatResponse.WorkflowTask == nil {
					return nil, nil
				}
				task := wtp.toWorkflowTask(heartbeatResponse.WorkflowTask)
				task.doneCh = doneCh
				task.laResultCh = laResultCh
				task.laRetryCh = laRetryCh
				return task, nil
			},
		)
		if taskCompletion == nil && taskErr == nil {
			return nil
		}
		if _, ok := taskErr.(workflowTaskHeartbeatError); ok {
			return taskErr
		}
		response, err := wtp.RespondTaskCompletedWithMetrics(taskCompletion, taskErr, task.task, startTime)
		if err != nil {
			// If we get an error responding to the workflow task we need to evict the execution from the cache.
			taskErr = err
			return err
		}

		if eventLevel := response.GetResetHistoryEventId(); eventLevel != 0 {
			wfctx.SetPreviousStartedEventID(eventLevel)
		}

		if response == nil || response.WorkflowTask == nil || taskErr != nil {
			return nil
		}

		// we are getting new workflow task, so reset the workflowTask and continue process the new one
		task = wtp.toWorkflowTask(response.WorkflowTask)
	}
}

func (wtp *workflowTaskProcessor) RespondTaskCompletedWithMetrics(
	taskCompletion *workflowTaskCompletion,
	taskErr error,
	task *workflowservice.PollWorkflowTaskQueueResponse,
	startTime time.Time,
) (response *workflowservice.RespondWorkflowTaskCompletedResponse, err error) {
	metricsHandler := wtp.metricsHandler.WithTags(metrics.WorkflowTags(task.WorkflowType.GetName()))

	emitFailMetric := false
	var failureReason string
	if taskErr != nil {
		wtp.logger.Warn("Failed to process workflow task.",
			tagWorkflowType, task.WorkflowType.GetName(),
			tagWorkflowID, task.WorkflowExecution.GetWorkflowId(),
			tagRunID, task.WorkflowExecution.GetRunId(),
			tagAttempt, task.Attempt,
			tagError, taskErr)
		emitFailMetric = true
		failWorkflowTask := wtp.errorToFailWorkflowTask(task.TaskToken, taskErr)
		failureReason = "WorkflowError"
		if failWorkflowTask.Cause == enumspb.WORKFLOW_TASK_FAILED_CAUSE_NON_DETERMINISTIC_ERROR {
			failureReason = "NonDeterminismError"
		}
		taskCompletion = &workflowTaskCompletion{rawRequest: failWorkflowTask}
	}

	metricsHandler.Timer(metrics.WorkflowTaskExecutionLatency).Record(time.Since(startTime))

	response, err = wtp.sendTaskCompletedRequest(taskCompletion, task)

	var grpcMessageTooLargeErr *retry.GrpcMessageTooLargeError
	if errors.As(err, &grpcMessageTooLargeErr) {
		secondEmitFailMetric, secondErr := wtp.reportGrpcMessageTooLarge(taskCompletion, task, err)
		if secondEmitFailMetric {
			emitFailMetric = true
			// Overwriting the original failure reason for metrics purposes
			failureReason = "GrpcMessageTooLarge"
		}
		// We already know the first error was GRPC message too large, if there was another error when reporting the first error
		// to the server it's probably more interesting for the user.
		if secondErr != nil {
			err = secondErr
		}
	}

	if emitFailMetric {
		incrementWorkflowTaskFailureCounter(metricsHandler, failureReason)
	}

	return
}

func (wtp *workflowTaskProcessor) sendTaskCompletedRequest(
	taskCompletion *workflowTaskCompletion,
	task *workflowservice.PollWorkflowTaskQueueResponse,
) (response *workflowservice.RespondWorkflowTaskCompletedResponse, err error) {
	ctx := context.Background()
	// Respond task completion.
	grpcCtx, cancel := newGRPCContext(ctx, grpcMetricsHandler(
		wtp.metricsHandler.WithTags(metrics.RPCTags(task.GetWorkflowType().GetName(),
			metrics.NoneTagValue, metrics.NoneTagValue))),
		defaultGrpcRetryParameters(ctx))
	defer cancel()
	if taskCompletion == nil {
		// should not happen
		panic("unknown request type from ProcessWorkflowTask()")
	}
	switch request := taskCompletion.rawRequest.(type) {
	case *workflowservice.RespondWorkflowTaskFailedRequest:
		// Only fail workflow task on first attempt, subsequent failure on the same workflow task will timeout.
		// This is to avoid spin on the failed workflow task. Checking Attempt not nil for older server.
		if task.GetAttempt() == 1 {
			_, err = wtp.service.RespondWorkflowTaskFailed(grpcCtx, request)
			if err != nil {
				traceLog(func() {
					wtp.logger.Debug("RespondWorkflowTaskFailed failed.", tagError, err)
				})
			} else if taskCompletion.applyCompletionMetrics != nil {
				taskCompletion.applyCompletionMetrics()
			}
		}
	case *workflowservice.RespondWorkflowTaskCompletedRequest:
		if request.StickyAttributes == nil && wtp.stickyCacheSize > 0 {
			request.StickyAttributes = &taskqueuepb.StickyExecutionAttributes{
				WorkerTaskQueue: &taskqueuepb.TaskQueue{
					Name:       getWorkerTaskQueue(wtp.stickyUUID),
					Kind:       enumspb.TASK_QUEUE_KIND_STICKY,
					NormalName: wtp.taskQueueName,
				},
				ScheduleToStartTimeout: durationpb.New(wtp.StickyScheduleToStartTimeout),
			}
		}
		eagerReserved := wtp.eagerActivityExecutor.applyToRequest(request)
		response, err = wtp.service.RespondWorkflowTaskCompleted(grpcCtx, request)
		if err != nil {
			traceLog(func() {
				wtp.logger.Debug("RespondWorkflowTaskCompleted failed.", tagError, err)
			})
		} else if taskCompletion.applyCompletionMetrics != nil {
			taskCompletion.applyCompletionMetrics()
		}
		wtp.eagerActivityExecutor.handleResponse(response, eagerReserved)
	case *workflowservice.RespondQueryTaskCompletedRequest:
		_, err = wtp.service.RespondQueryTaskCompleted(grpcCtx, request)
		if err != nil {
			traceLog(func() {
				wtp.logger.Debug("RespondQueryTaskCompleted failed.", tagError, err)
			})
		} else if taskCompletion.applyCompletionMetrics != nil {
			taskCompletion.applyCompletionMetrics()
		}
	default:
		// should not happen
		panic("unknown request type from ProcessWorkflowTask()")
	}
	return
}

func (wtp *workflowTaskProcessor) reportGrpcMessageTooLarge(
	taskCompletion *workflowTaskCompletion,
	task *workflowservice.PollWorkflowTaskQueueResponse,
	sendErr error,
) (emitFailMetric bool, err error) {
	if taskCompletion == nil {
		// should not happen
		panic("unknown request type from ProcessWorkflowTask()")
	}
	switch taskCompletion.rawRequest.(type) {
	case *workflowservice.RespondWorkflowTaskCompletedRequest, *workflowservice.RespondWorkflowTaskFailedRequest:
		emitFailMetric = true
		request := wtp.errorToFailWorkflowTask(task.TaskToken, sendErr)
		request.Cause = enumspb.WORKFLOW_TASK_FAILED_CAUSE_GRPC_MESSAGE_TOO_LARGE
		_, err = wtp.sendTaskCompletedRequest(&workflowTaskCompletion{rawRequest: request}, task)
	case *workflowservice.RespondQueryTaskCompletedRequest:
		request := &workflowservice.RespondQueryTaskCompletedRequest{
			TaskToken:     task.TaskToken,
			CompletedType: enumspb.QUERY_RESULT_TYPE_FAILED,
			ErrorMessage:  sendErr.Error(),
			Namespace:     wtp.namespace,
			Failure:       wtp.failureConverter.ErrorToFailure(sendErr),
			Cause:         enumspb.WORKFLOW_TASK_FAILED_CAUSE_GRPC_MESSAGE_TOO_LARGE,
		}
		_, err = wtp.sendTaskCompletedRequest(&workflowTaskCompletion{rawRequest: request}, task)
	default:
		// should not happen
		panic("unknown request type from ProcessWorkflowTask()")
	}
	return
}

func (wtp *workflowTaskProcessor) errorToFailWorkflowTask(taskToken []byte, err error) *workflowservice.RespondWorkflowTaskFailedRequest {
	cause := enumspb.WORKFLOW_TASK_FAILED_CAUSE_WORKFLOW_WORKER_UNHANDLED_FAILURE
	// If it was a panic due to a bad state machine or if it was a history
	// mismatch error, mark as non-deterministic
	if panicErr, _ := err.(*workflowPanicError); panicErr != nil {
		if _, badStateMachine := panicErr.value.(stateMachineIllegalStatePanic); badStateMachine {
			cause = enumspb.WORKFLOW_TASK_FAILED_CAUSE_NON_DETERMINISTIC_ERROR
		}
	} else if _, mismatch := err.(historyMismatchError); mismatch {
		cause = enumspb.WORKFLOW_TASK_FAILED_CAUSE_NON_DETERMINISTIC_ERROR
	} else if _, unknown := err.(unknownSdkFlagError); unknown {
		cause = enumspb.WORKFLOW_TASK_FAILED_CAUSE_NON_DETERMINISTIC_ERROR
	}

	return wtp.errorToFailWorkflowTaskWithCause(taskToken, err, cause)
}

func (wtp *workflowTaskProcessor) errorToFailWorkflowTaskWithCause(taskToken []byte, err error, cause enumspb.WorkflowTaskFailedCause) *workflowservice.RespondWorkflowTaskFailedRequest {
	builtRequest := &workflowservice.RespondWorkflowTaskFailedRequest{
		TaskToken:      taskToken,
		Cause:          cause,
		Failure:        wtp.failureConverter.ErrorToFailure(err),
		Identity:       wtp.identity,
		BinaryChecksum: wtp.workerBuildID,
		Namespace:      wtp.namespace,
		WorkerVersion: &commonpb.WorkerVersionStamp{
			BuildId:       wtp.workerBuildID,
			UseVersioning: wtp.useBuildIDVersioning,
		},
		Deployment: &deploymentpb.Deployment{
			BuildId:    wtp.workerBuildID,
			SeriesName: wtp.getDeploymentName(),
		},
		DeploymentOptions: workerDeploymentOptionsToProto(
			wtp.useBuildIDVersioning,
			wtp.workerDeploymentVersion,
		),
	}

	if wtp.getCapabilities().BuildIdBasedVersioning {
		//lint:ignore SA1019 ignore deprecated versioning APIs
		builtRequest.BinaryChecksum = ""
	}

	return builtRequest
}

func newLocalActivityPoller(
	params workerExecutionParameters,
	laTunnel *localActivityTunnel,
	interceptors []WorkerInterceptor,
	client *WorkflowClient,
	workerStopCh <-chan struct{},
) *localActivityTaskPoller {
	handler := &localActivityTaskHandler{
		backgroundContext:  params.BackgroundContext,
		metricsHandler:     params.MetricsHandler,
		logger:             params.Logger,
		dataConverter:      params.DataConverter,
		contextPropagators: params.ContextPropagators,
		interceptors:       interceptors,
		client:             client,
		workerStopChannel:  workerStopCh,
	}
	return &localActivityTaskPoller{
		basePoller:   basePoller{metricsHandler: params.MetricsHandler, stopC: params.WorkerStopChannel},
		handler:      handler,
		logger:       params.Logger,
		laTunnel:     laTunnel,
		workerStopCh: workerStopCh,
	}
}

func (latp *localActivityTaskPoller) Cleanup() error {
	return nil
}

func (latp *localActivityTaskPoller) PollTask() (taskForWorker, error) {
	return latp.laTunnel.getTask(), nil
}

func (latp *localActivityTaskPoller) ProcessTask(task interface{}) error {
	if latp.stopping() {
		return errStop
	}

	result := latp.handler.executeLocalActivityTask(task.(*localActivityTask))

	// If shutdown is initiated after we begin local activity execution, there is no need to send result back to
	// laResultCh, as both workers receive shutdown from top down.
	if latp.stopping() {
		return errStop
	}
	// We need to send back the local activity result to unblock workflowTaskPoller.processWorkflowTask() which is
	// synchronously listening on the laResultCh. We also want to make sure we don't block here forever in case
	// processWorkflowTask() already returns and nobody is receiving from laResultCh. We guarantee that doneCh is closed
	// before returning from workflowTaskPoller.processWorkflowTask().
	select {
	case result.task.workflowTask.laResultCh <- result:
		return nil
	case <-result.task.workflowTask.doneCh:
		// processWorkflowTask() already returns, just drop this local activity result.
		return nil
	}
}

func (lath *localActivityTaskHandler) executeLocalActivityTask(task *localActivityTask) (result *localActivityResult) {
	workflowType := task.params.WorkflowInfo.WorkflowType.Name
	activityType := task.params.ActivityType
	metricsHandler := lath.metricsHandler.WithTags(metrics.LocalActivityTags(workflowType, activityType))

	metricsHandler.Counter(metrics.LocalActivityTotalCounter).Inc(1)

	ae := activityExecutor{name: activityType, fn: task.params.ActivityFn}
	traceLog(func() {
		lath.logger.Debug("Processing new local activity task",
			tagWorkflowID, task.params.WorkflowInfo.WorkflowExecution.ID,
			tagRunID, task.params.WorkflowInfo.WorkflowExecution.RunID,
			tagActivityType, activityType,
			tagAttempt, task.attempt,
		)
	})
	ctx, err := WithLocalActivityTask(lath.backgroundContext, task, lath.logger, lath.metricsHandler,
		lath.dataConverter, lath.interceptors, lath.client, lath.workerStopChannel)
	if err != nil {
		return &localActivityResult{task: task, err: fmt.Errorf("failed building context: %w", err)}
	}

	// propagate context information into the local activity context from the headers
	ctx, err = contextWithHeaderPropagated(ctx, task.header, lath.contextPropagators)
	if err != nil {
		return &localActivityResult{task: task, err: err}
	}

	info := getActivityEnv(ctx)
	ctx, cancel := context.WithDeadline(ctx, info.deadline)
	defer cancel()

	task.Lock()
	if task.canceled {
		task.Unlock()
		return &localActivityResult{err: ErrCanceled, task: task}
	}
	task.attemptsThisWFT += 1
	task.cancelFunc = cancel
	task.Unlock()

	var laResult *commonpb.Payloads
	doneCh := make(chan struct{})
	go func(ch chan struct{}) {
		laStartTime := time.Now()
		defer close(ch)

		// panic handler
		defer func() {
			if p := recover(); p != nil {
				topLine := fmt.Sprintf("local activity for %s [panic]:", activityType)
				st := getStackTraceRaw(topLine, 7, 0)
				lath.logger.Error("LocalActivity panic.",
					tagWorkflowID, task.params.WorkflowInfo.WorkflowExecution.ID,
					tagRunID, task.params.WorkflowInfo.WorkflowExecution.RunID,
					tagActivityType, activityType,
					tagAttempt, task.attempt,
					tagPanicError, fmt.Sprintf("%v", p),
					tagPanicStack, st)
				metricsHandler.Counter(metrics.LocalActivityErrorCounter).Inc(1)
				err = newPanicError(p, st)
			}
			if err != nil && !isBenignApplicationError(err) {
				metricsHandler.Counter(metrics.LocalActivityFailedCounter).Inc(1)
				metricsHandler.Counter(metrics.LocalActivityExecutionFailedCounter).Inc(1)
			}
		}()

		laResult, err = ae.ExecuteWithActualArgs(ctx, task.params.InputArgs)
		executionLatency := time.Since(laStartTime)
		metricsHandler.Timer(metrics.LocalActivityExecutionLatency).Record(executionLatency)
		if time.Now().After(info.deadline) {
			// If local activity takes longer than expected timeout, the context would already be DeadlineExceeded and
			// the result would be discarded. Print a warning in this case.
			lath.logger.Warn("LocalActivity completed after activity deadline.",
				"LocalActivityID", task.activityID,
				"ActivityDeadline", info.deadline,
				"LocalActivityType", activityType,
				"ScheduleToCloseTimeout", task.params.ScheduleToCloseTimeout,
				"StartToCloseTimeout", task.params.StartToCloseTimeout,
				"ActualExecutionDuration", executionLatency)
		}
	}(doneCh)

WaitResult:
	select {
	case <-ctx.Done():
		select {
		case <-doneCh:
			// double check if result is ready.
			break WaitResult
		default:
		}

		// context is done
		if ctx.Err() == context.Canceled {
			metricsHandler.Counter(metrics.LocalActivityCanceledCounter).Inc(1)
			metricsHandler.Counter(metrics.LocalActivityExecutionCanceledCounter).Inc(1)
			return &localActivityResult{err: ErrCanceled, task: task}
		} else if ctx.Err() == context.DeadlineExceeded {
			if task.params.ScheduleToCloseTimeout != 0 && time.Now().After(info.scheduledTime.Add(task.params.ScheduleToCloseTimeout)) {
				return &localActivityResult{err: ErrDeadlineExceeded, task: task}
			} else {
				return &localActivityResult{err: NewTimeoutError("deadline exceeded", enumspb.TIMEOUT_TYPE_START_TO_CLOSE, nil), task: task}
			}
		} else {
			// should not happen
			return &localActivityResult{err: NewApplicationError("unexpected context done", "", true, nil), task: task}
		}
	case <-doneCh:
		// local activity completed
	}

	if err == nil {
		metricsHandler.
			Timer(metrics.LocalActivitySucceedEndToEndLatency).
			Record(time.Since(task.params.ScheduledTime))
	}
	return &localActivityResult{result: laResult, err: err, task: task}
}

func (wtp *workflowTaskPoller) release(kind enumspb.TaskQueueKind) {
	if wtp.stickyCacheSize <= 0 {
		return
	}

	wtp.requestLock.Lock()
	if kind == enumspb.TASK_QUEUE_KIND_STICKY {
		wtp.pendingStickyPollCount--
	} else {
		wtp.pendingRegularPollCount--
	}
	wtp.requestLock.Unlock()
}

func (wtp *workflowTaskPoller) updateBacklog(taskQueueKind enumspb.TaskQueueKind, backlogCountHint int64) {
	if taskQueueKind == enumspb.TASK_QUEUE_KIND_NORMAL || wtp.stickyCacheSize <= 0 {
		// we only care about sticky backlog for now.
		return
	}
	wtp.requestLock.Lock()
	wtp.stickyBacklog = backlogCountHint
	wtp.requestLock.Unlock()
}

// getNextPollRequest returns appropriate next poll request based on poller configuration and mode.
// Simple rules:
//  1. if mode is NonSticky, always poll from regular task queue
//  2. if mode is Sticky, always poll from sticky task queue
//  3. if mode is Mixed
//     3.1. if sticky execution is disabled, always poll for regular task queue
//     3.2. otherwise:
//     3.2.1) if sticky task queue has backlog, always prefer to process sticky task first
//     3.2.2) poll from the task queue that has less pending requests (prefer sticky when they are the same).
func (wtp *workflowTaskPoller) getNextPollRequest() (request *workflowservice.PollWorkflowTaskQueueRequest) {
	taskQueue := &taskqueuepb.TaskQueue{
		Name: wtp.taskQueueName,
		Kind: enumspb.TASK_QUEUE_KIND_NORMAL,
	}

	if wtp.mode == NonSticky || wtp.stickyCacheSize <= 0 {
		// Do nothing, taskQueue is already set to non-sticky
	} else if wtp.mode == Sticky {
		taskQueue.Name = getWorkerTaskQueue(wtp.stickyUUID)
		taskQueue.Kind = enumspb.TASK_QUEUE_KIND_STICKY
		taskQueue.NormalName = wtp.taskQueueName
	} else if wtp.mode == Mixed {
		wtp.requestLock.Lock()
		if wtp.stickyBacklog > 0 || wtp.pendingStickyPollCount <= wtp.pendingRegularPollCount {
			wtp.pendingStickyPollCount++
			taskQueue.Name = getWorkerTaskQueue(wtp.stickyUUID)
			taskQueue.Kind = enumspb.TASK_QUEUE_KIND_STICKY
			taskQueue.NormalName = wtp.taskQueueName
		} else {
			wtp.pendingRegularPollCount++
		}
		wtp.requestLock.Unlock()
	} else {
		panic("unknown workflow task poller mode")
	}

	builtRequest := &workflowservice.PollWorkflowTaskQueueRequest{
		Namespace:      wtp.namespace,
		TaskQueue:      taskQueue,
		Identity:       wtp.identity,
		BinaryChecksum: wtp.workerBuildID,
		WorkerVersionCapabilities: &commonpb.WorkerVersionCapabilities{
			BuildId:              wtp.workerBuildID,
			UseVersioning:        wtp.useBuildIDVersioning,
			DeploymentSeriesName: wtp.getDeploymentName(),
		},
		DeploymentOptions: workerDeploymentOptionsToProto(
			wtp.useBuildIDVersioning,
			wtp.workerDeploymentVersion,
		),
	}
	if wtp.getCapabilities().BuildIdBasedVersioning {
		//lint:ignore SA1019 ignore deprecated versioning APIs
		builtRequest.BinaryChecksum = ""
	}
	return builtRequest
}

// Poll the workflow task queue and update the num_poller metric
func (wtp *workflowTaskPoller) pollWorkflowTaskQueue(ctx context.Context, request *workflowservice.PollWorkflowTaskQueueRequest) (*workflowservice.PollWorkflowTaskQueueResponse, error) {
	if request.TaskQueue.GetKind() == enumspb.TASK_QUEUE_KIND_NORMAL {
		wtp.numNormalPollerMetric.increment()
		defer wtp.numNormalPollerMetric.decrement()
	} else {
		wtp.numStickyPollerMetric.increment()
		defer wtp.numStickyPollerMetric.decrement()
	}

	return wtp.service.PollWorkflowTaskQueue(ctx, request)
}

// Poll for a single workflow task from the service
func (wtp *workflowTaskPoller) poll(ctx context.Context) (taskForWorker, error) {
	traceLog(func() {
		wtp.logger.Debug("workflowTaskPoller::Poll")
	})

	request := wtp.getNextPollRequest()
	defer wtp.release(request.TaskQueue.GetKind())

	response, err := wtp.pollWorkflowTaskQueue(ctx, request)
	if err != nil {
		wtp.updateBacklog(request.TaskQueue.GetKind(), 0)
		return nil, err
	}

	if response == nil || len(response.TaskToken) == 0 {
		// Emit using base scope as no workflow type information is available in the case of empty poll
		wtp.metricsHandler.Counter(metrics.WorkflowTaskQueuePollEmptyCounter).Inc(1)
		wtp.updateBacklog(request.TaskQueue.GetKind(), 0)
		return &workflowTask{}, nil
	}

	wtp.updateBacklog(request.TaskQueue.GetKind(), response.GetBacklogCountHint())

	task := wtp.toWorkflowTask(response)
	traceLog(func() {
		var firstEventID int64 = -1
		if response.History != nil && len(response.History.Events) > 0 {
			firstEventID = response.History.Events[0].GetEventId()
		}
		wtp.logger.Debug("workflowTaskPoller::Poll Succeed",
			"StartedEventID", response.GetStartedEventId(),
			"Attempt", response.GetAttempt(),
			"FirstEventID", firstEventID,
			"IsQueryTask", response.Query != nil)
	})

	metricsHandler := wtp.metricsHandler.WithTags(metrics.WorkflowTags(response.WorkflowType.GetName()))
	metricsHandler.Counter(metrics.WorkflowTaskQueuePollSucceedCounter).Inc(1)

	scheduleToStartLatency := response.GetStartedTime().AsTime().Sub(response.GetScheduledTime().AsTime())
	metricsHandler.Timer(metrics.WorkflowTaskScheduleToStartLatency).Record(scheduleToStartLatency)
	return task, nil
}

func (wtp *workflowTaskPoller) toWorkflowTask(response *workflowservice.PollWorkflowTaskQueueResponse) *workflowTask {
	historyIterator := &historyIteratorImpl{
		execution:      response.WorkflowExecution,
		nextPageToken:  response.NextPageToken,
		namespace:      wtp.namespace,
		service:        wtp.service,
		maxEventID:     response.GetStartedEventId(),
		metricsHandler: wtp.metricsHandler,
		taskQueue:      wtp.taskQueueName,
	}
	task := &workflowTask{
		task:            response,
		historyIterator: historyIterator,
	}
	return task
}

func (wtp *workflowTaskProcessor) toWorkflowTask(response *workflowservice.PollWorkflowTaskQueueResponse) *workflowTask {
	historyIterator := &historyIteratorImpl{
		execution:      response.WorkflowExecution,
		nextPageToken:  response.NextPageToken,
		namespace:      wtp.namespace,
		service:        wtp.service,
		maxEventID:     response.GetStartedEventId(),
		metricsHandler: wtp.metricsHandler,
		taskQueue:      wtp.taskQueueName,
	}
	task := &workflowTask{
		task:            response,
		historyIterator: historyIterator,
	}
	return task
}

func (h *historyIteratorImpl) GetNextPage() (*historypb.History, error) {
	if h.iteratorFunc == nil {
		h.iteratorFunc = newGetHistoryPageFunc(
			context.Background(),
			h.service,
			h.namespace,
			h.execution,
			h.maxEventID,
			h.metricsHandler,
			h.taskQueue,
		)
	}

	history, token, err := h.iteratorFunc(h.nextPageToken)
	if err != nil {
		return nil, err
	}
	h.nextPageToken = token
	return history, nil
}

func (h *historyIteratorImpl) Reset() {
	h.nextPageToken = nil
}

func (h *historyIteratorImpl) HasNextPage() bool {
	return h.nextPageToken != nil
}

func newGetHistoryPageFunc(
	ctx context.Context,
	service workflowservice.WorkflowServiceClient,
	namespace string,
	execution *commonpb.WorkflowExecution,
	lastEventID int64,
	metricsHandler metrics.Handler,
	taskQueue string,
) func(nextPageToken []byte) (*historypb.History, []byte, error) {
	return func(nextPageToken []byte) (*historypb.History, []byte, error) {
		var resp *workflowservice.GetWorkflowExecutionHistoryResponse
		grpcCtx, cancel := newGRPCContext(ctx, grpcMetricsHandler(
			metricsHandler.WithTags(metrics.RPCTags(metrics.NoneTagValue, metrics.NoneTagValue, taskQueue))),
			defaultGrpcRetryParameters(ctx))
		defer cancel()

		resp, err := service.GetWorkflowExecutionHistory(grpcCtx, &workflowservice.GetWorkflowExecutionHistoryRequest{
			Namespace:     namespace,
			Execution:     execution,
			NextPageToken: nextPageToken,
		})
		if err != nil {
			return nil, nil, err
		}

		var h *historypb.History

		if resp.RawHistory != nil {
			h, err = serializer.DeserializeBlobDataToHistoryEvents(resp.RawHistory, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
			if err != nil {
				return nil, nil, nil
			}
		} else {
			h = resp.History
		}

		size := len(h.Events)
		// While the SDK is processing a workflow task, the workflow task could timeout and server would start
		// a new workflow task or the server looses the workflow task if it is a speculative workflow task. In either
		// case, the new workflow task could have events that are beyond the last event ID that the SDK expects to process.
		// In such cases, the SDK should return error indicating that the workflow task is stale since the result will not be used.
		if size > 0 && lastEventID > 0 &&
			h.Events[size-1].GetEventId() > lastEventID {
			return nil, nil, fmt.Errorf("history contains events past expected last event ID (%v) "+
				"likely this means the current workflow task is no longer valid", lastEventID)

		}

		return h, resp.NextPageToken, nil
	}
}

func newActivityTaskPoller(taskHandler ActivityTaskHandler, service workflowservice.WorkflowServiceClient, params workerExecutionParameters) *activityTaskPoller {
	return &activityTaskPoller{
		basePoller: basePoller{
			metricsHandler:          params.MetricsHandler,
			stopC:                   params.WorkerStopChannel,
			workerBuildID:           params.getBuildID(),
			useBuildIDVersioning:    params.UseBuildIDForVersioning,
			workerDeploymentVersion: params.DeploymentOptions.Version,
			capabilities:            params.capabilities,
		},
		taskHandler:         taskHandler,
		service:             service,
		namespace:           params.Namespace,
		taskQueueName:       params.TaskQueue,
		identity:            params.Identity,
		logger:              params.Logger,
		activitiesPerSecond: params.TaskQueueActivitiesPerSecond,
		numPollerMetric:     newNumPollerMetric(params.MetricsHandler, metrics.PollerTypeActivityTask),
	}
}

// Poll the activity task queue and update the num_poller metric
func (atp *activityTaskPoller) pollActivityTaskQueue(ctx context.Context, request *workflowservice.PollActivityTaskQueueRequest) (*workflowservice.PollActivityTaskQueueResponse, error) {
	atp.numPollerMetric.increment()
	defer atp.numPollerMetric.decrement()

	return atp.service.PollActivityTaskQueue(ctx, request)
}

// Poll for a single activity task from the service
func (atp *activityTaskPoller) poll(ctx context.Context) (taskForWorker, error) {
	traceLog(func() {
		atp.logger.Debug("activityTaskPoller::Poll")
	})
	request := &workflowservice.PollActivityTaskQueueRequest{
		Namespace:         atp.namespace,
		TaskQueue:         &taskqueuepb.TaskQueue{Name: atp.taskQueueName, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
		Identity:          atp.identity,
		TaskQueueMetadata: &taskqueuepb.TaskQueueMetadata{MaxTasksPerSecond: wrapperspb.Double(atp.activitiesPerSecond)},
		WorkerVersionCapabilities: &commonpb.WorkerVersionCapabilities{
			BuildId:              atp.workerBuildID,
			UseVersioning:        atp.useBuildIDVersioning,
			DeploymentSeriesName: atp.getDeploymentName(),
		},
		DeploymentOptions: workerDeploymentOptionsToProto(
			atp.useBuildIDVersioning,
			atp.workerDeploymentVersion,
		),
	}

	response, err := atp.pollActivityTaskQueue(ctx, request)
	if err != nil {
		return nil, err
	}
	if response == nil || len(response.TaskToken) == 0 {
		// No activity info is available on empty poll.  Emit using base scope.
		atp.metricsHandler.Counter(metrics.ActivityPollNoTaskCounter).Inc(1)
		return &activityTask{}, nil
	}

	workflowType := response.WorkflowType.GetName()
	activityType := response.ActivityType.GetName()
	metricsHandler := atp.metricsHandler.WithTags(metrics.ActivityTags(workflowType, activityType, atp.taskQueueName))

	scheduleToStartLatency := response.GetStartedTime().AsTime().Sub(response.GetCurrentAttemptScheduledTime().AsTime())
	metricsHandler.Timer(metrics.ActivityScheduleToStartLatency).Record(scheduleToStartLatency)

	return &activityTask{task: response}, nil
}

func (atp *activityTaskPoller) Cleanup() error {
	return nil
}

// PollTask polls a new task
func (atp *activityTaskPoller) PollTask() (taskForWorker, error) {
	// Get the task.
	activityTask, err := atp.doPoll(atp.poll)
	if err != nil {
		return nil, err
	}
	return activityTask, nil
}

// ProcessTask processes a new task
func (atp *activityTaskPoller) ProcessTask(task interface{}) error {
	if atp.stopping() {
		return errStop
	}

	activityTask := task.(*activityTask)
	if activityTask.task == nil {
		// We didn't have task, poll might have timeout.
		traceLog(func() {
			atp.logger.Debug("Activity task unavailable")
		})
		return nil
	}

	workflowType := activityTask.task.WorkflowType.GetName()
	activityType := activityTask.task.ActivityType.GetName()
	activityMetricsHandler := atp.metricsHandler.WithTags(metrics.ActivityTags(workflowType, activityType, atp.taskQueueName))

	executionStartTime := time.Now()
	// Process the activity task.
	request, err := atp.taskHandler.Execute(atp.taskQueueName, activityTask.task)
	// err is returned in case of internal failure, such as unable to propagate context or context timeout.
	if err != nil {
		activityMetricsHandler.Counter(metrics.ActivityExecutionFailedCounter).Inc(1)
		return err
	}
	// in case if activity execution failed, request should be of type RespondActivityTaskFailedRequest
	if req, ok := request.(*workflowservice.RespondActivityTaskFailedRequest); ok {
		if !isBenignProtoApplicationFailure(req.Failure) {
			activityMetricsHandler.Counter(metrics.ActivityExecutionFailedCounter).Inc(1)
		}
	}
	activityMetricsHandler.Timer(metrics.ActivityExecutionLatency).Record(time.Since(executionStartTime))

	if request == ErrActivityResultPending {
		return nil
	}

	rpcMetricsHandler := atp.metricsHandler.WithTags(metrics.RPCTags(workflowType, activityType, metrics.NoneTagValue))
	reportErr := reportActivityComplete(context.Background(), atp.service, request, rpcMetricsHandler)
	if reportErr != nil {
		traceLog(func() {
			atp.logger.Debug("reportActivityComplete failed", tagError, reportErr)
		})
		return reportErr
	}

	if _, ok := request.(*workflowservice.RespondActivityTaskCompletedRequest); ok {
		activityMetricsHandler.
			Timer(metrics.ActivitySucceedEndToEndLatency).
			Record(time.Since(activityTask.task.GetScheduledTime().AsTime()))
	}
	return nil
}

func reportActivityComplete(
	ctx context.Context,
	service workflowservice.WorkflowServiceClient,
	request interface{},
	rpcMetricsHandler metrics.Handler,
) error {
	if request == nil {
		// nothing to report
		return nil
	}

	var reportErr error
	switch rqst := request.(type) {
	case *workflowservice.RespondActivityTaskCanceledRequest:
		grpcCtx, cancel := newGRPCContext(ctx, grpcMetricsHandler(rpcMetricsHandler),
			defaultGrpcRetryParameters(ctx))
		defer cancel()
		_, err := service.RespondActivityTaskCanceled(grpcCtx, rqst)
		reportErr = err
	case *workflowservice.RespondActivityTaskFailedRequest:
		grpcCtx, cancel := newGRPCContext(ctx, grpcMetricsHandler(rpcMetricsHandler), defaultGrpcRetryParameters(ctx))
		defer cancel()
		_, err := service.RespondActivityTaskFailed(grpcCtx, rqst)
		reportErr = err
	case *workflowservice.RespondActivityTaskCompletedRequest:
		grpcCtx, cancel := newGRPCContext(ctx, grpcMetricsHandler(rpcMetricsHandler),
			defaultGrpcRetryParameters(ctx))
		defer cancel()
		_, err := service.RespondActivityTaskCompleted(grpcCtx, rqst)
		reportErr = err
	}
	return reportErr
}

func reportActivityCompleteByID(
	ctx context.Context,
	service workflowservice.WorkflowServiceClient,
	request interface{},
	rpcMetricsHandler metrics.Handler,
) error {
	if request == nil {
		// nothing to report
		return nil
	}

	var reportErr error
	switch request := request.(type) {
	case *workflowservice.RespondActivityTaskCanceledByIdRequest:
		grpcCtx, cancel := newGRPCContext(ctx, grpcMetricsHandler(rpcMetricsHandler),
			defaultGrpcRetryParameters(ctx))
		defer cancel()
		_, err := service.RespondActivityTaskCanceledById(grpcCtx, request)
		reportErr = err
	case *workflowservice.RespondActivityTaskFailedByIdRequest:
		grpcCtx, cancel := newGRPCContext(ctx, grpcMetricsHandler(rpcMetricsHandler),
			defaultGrpcRetryParameters(ctx))
		defer cancel()
		_, err := service.RespondActivityTaskFailedById(grpcCtx, request)
		reportErr = err
	case *workflowservice.RespondActivityTaskCompletedByIdRequest:
		grpcCtx, cancel := newGRPCContext(ctx, grpcMetricsHandler(rpcMetricsHandler),
			defaultGrpcRetryParameters(ctx))
		defer cancel()
		_, err := service.RespondActivityTaskCompletedById(grpcCtx, request)
		reportErr = err
	}
	return reportErr
}

func convertActivityResultToRespondRequest(
	identity string,
	taskToken []byte,
	result *commonpb.Payloads,
	err error,
	dataConverter converter.DataConverter,
	failureConverter converter.FailureConverter,
	namespace string,
	cancelAllowed bool,
	versionStamp *commonpb.WorkerVersionStamp,
	deployment *deploymentpb.Deployment,
	workerDeploymentOptions *deploymentpb.WorkerDeploymentOptions,
) interface{} {
	if err == ErrActivityResultPending {
		// activity result is pending and will be completed asynchronously.
		// nothing to report at this point
		return ErrActivityResultPending
	}

	if err == nil {
		return &workflowservice.RespondActivityTaskCompletedRequest{
			TaskToken:         taskToken,
			Result:            result,
			Identity:          identity,
			Namespace:         namespace,
			WorkerVersion:     versionStamp,
			Deployment:        deployment,
			DeploymentOptions: workerDeploymentOptions,
		}
	}

	// Only respond with canceled if allowed
	if cancelAllowed {
		var canceledErr *CanceledError
		if errors.As(err, &canceledErr) {
			return &workflowservice.RespondActivityTaskCanceledRequest{
				TaskToken:         taskToken,
				Details:           convertErrDetailsToPayloads(canceledErr.details, dataConverter),
				Identity:          identity,
				Namespace:         namespace,
				WorkerVersion:     versionStamp,
				Deployment:        deployment,
				DeploymentOptions: workerDeploymentOptions,
			}
		}
		if errors.Is(err, context.Canceled) {
			return &workflowservice.RespondActivityTaskCanceledRequest{
				TaskToken:         taskToken,
				Identity:          identity,
				Namespace:         namespace,
				WorkerVersion:     versionStamp,
				Deployment:        deployment,
				DeploymentOptions: workerDeploymentOptions,
			}
		}
	}

	// If a canceled error is returned but it wasn't allowed, we have to wrap in
	// an unexpected-cancel application error
	if _, isCanceledErr := err.(*CanceledError); isCanceledErr {
		err = fmt.Errorf("unexpected activity cancel error: %w", err)
	}

	return &workflowservice.RespondActivityTaskFailedRequest{
		TaskToken:         taskToken,
		Failure:           failureConverter.ErrorToFailure(err),
		Identity:          identity,
		Namespace:         namespace,
		WorkerVersion:     versionStamp,
		Deployment:        deployment,
		DeploymentOptions: workerDeploymentOptions,
	}
}

func convertActivityResultToRespondRequestByID(
	identity string,
	namespace string,
	workflowID string,
	runID string,
	activityID string,
	result *commonpb.Payloads,
	err error,
	dataConverter converter.DataConverter,
	failureConverter converter.FailureConverter,
	cancelAllowed bool,
) interface{} {
	if err == ErrActivityResultPending {
		// activity result is pending and will be completed asynchronously.
		// nothing to report at this point
		return nil
	}

	if err == nil {
		return &workflowservice.RespondActivityTaskCompletedByIdRequest{
			Namespace:  namespace,
			WorkflowId: workflowID,
			RunId:      runID,
			ActivityId: activityID,
			Result:     result,
			Identity:   identity,
		}
	}

	// Only respond with canceled if allowed
	if cancelAllowed {
		var canceledErr *CanceledError
		if errors.As(err, &canceledErr) {
			return &workflowservice.RespondActivityTaskCanceledByIdRequest{
				Namespace:  namespace,
				WorkflowId: workflowID,
				RunId:      runID,
				ActivityId: activityID,
				Details:    convertErrDetailsToPayloads(canceledErr.details, dataConverter),
				Identity:   identity,
			}
		}
		if errors.Is(err, context.Canceled) {
			return &workflowservice.RespondActivityTaskCanceledByIdRequest{
				Namespace:  namespace,
				WorkflowId: workflowID,
				RunId:      runID,
				ActivityId: activityID,
				Identity:   identity,
			}
		}
	}

	// If a canceled error is returned but it wasn't allowed, we have to wrap in
	// an unexpected-cancel application error
	if _, isCanceledErr := err.(*CanceledError); isCanceledErr {
		err = fmt.Errorf("unexpected activity cancel error: %w", err)
	}

	return &workflowservice.RespondActivityTaskFailedByIdRequest{
		Namespace:  namespace,
		WorkflowId: workflowID,
		RunId:      runID,
		ActivityId: activityID,
		Failure:    failureConverter.ErrorToFailure(err),
		Identity:   identity,
	}
}

func (wft *workflowTask) isEmpty() bool {
	return wft.task == nil
}

func (wft *workflowTask) scaleDecision() (pollerScaleDecision, bool) {
	if wft.task == nil || wft.task.PollerScalingDecision == nil {
		return pollerScaleDecision{}, false
	}
	return pollerScaleDecision{
		pollRequestDeltaSuggestion: int(wft.task.PollerScalingDecision.PollRequestDeltaSuggestion),
	}, true
}

func (at *activityTask) isEmpty() bool {
	return at.task == nil
}

func (at *activityTask) scaleDecision() (pollerScaleDecision, bool) {
	if at.task == nil || at.task.PollerScalingDecision == nil {
		return pollerScaleDecision{}, false
	}
	return pollerScaleDecision{
		pollRequestDeltaSuggestion: int(at.task.PollerScalingDecision.PollRequestDeltaSuggestion),
	}, true
}

func (*localActivityTask) isEmpty() bool {
	return false
}

func (*localActivityTask) scaleDecision() (pollerScaleDecision, bool) {
	return pollerScaleDecision{}, false
}

func (*eagerWorkflowTask) isEmpty() bool {
	return false
}

func (*eagerWorkflowTask) scaleDecision() (pollerScaleDecision, bool) {
	return pollerScaleDecision{}, false
}

func (nt *nexusTask) isEmpty() bool {
	return nt.task == nil
}

func (nt *nexusTask) scaleDecision() (pollerScaleDecision, bool) {
	if nt.task == nil || nt.task.PollerScalingDecision == nil {
		return pollerScaleDecision{}, false
	}
	return pollerScaleDecision{
		pollRequestDeltaSuggestion: int(nt.task.PollerScalingDecision.PollRequestDeltaSuggestion),
	}, true
}
