package internal

// All code in this file is private to the package.

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	deploymentpb "go.temporal.io/api/deployment/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	protocolpb "go.temporal.io/api/protocol/v1"
	querypb "go.temporal.io/api/query/v1"
	"go.temporal.io/api/sdk/v1"
	"go.temporal.io/api/serviceerror"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"go.temporal.io/sdk/internal/common/retry"
	"go.temporal.io/sdk/internal/protocol"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/internal/common/util"
	"go.temporal.io/sdk/log"
)

const (
	defaultStickyCacheSize = 10000

	noRetryBackoff = time.Duration(-1)

	defaultDefaultHeartbeatThrottleInterval = 30 * time.Second
	defaultMaxHeartbeatThrottleInterval     = 60 * time.Second
)

var (
	// ErrActivityPaused is returned from an activity heartbeat or the cause of an activity's context to indicate that the activity is paused.
	//
	// WARNING: Activity pause is currently experimental
	ErrActivityPaused = errors.New("activity paused")
)

type (
	// workflowExecutionEventHandler process a single event.
	workflowExecutionEventHandler interface {
		// Process a single event and return the assosciated commands.
		// Return List of commands made, any error.
		ProcessEvent(event *historypb.HistoryEvent, isReplay bool, isLast bool) error
		// ProcessInteraction processes interaction inputs
		ProcessMessage(msg *protocolpb.Message, isReplay bool, isLast bool) error
		// ProcessQuery process a query request.
		ProcessQuery(queryType string, queryArgs *commonpb.Payloads, header *commonpb.Header) (*commonpb.Payloads, error)
		StackTrace() string
		// Close for cleaning up resources on this event handler
		Close()
	}

	// workflowTask wraps a workflow task.
	workflowTask struct {
		task            *workflowservice.PollWorkflowTaskQueueResponse
		historyIterator HistoryIterator
		doneCh          chan struct{}
		laResultCh      chan *localActivityResult

		// This channel must be initialized with a one-size buffer and is used to indicate when
		// it is time for a local activity to be retried
		laRetryCh chan *localActivityTask
	}

	// eagerWorkflowTask represents a workflow task sent from an eager workflow executor
	eagerWorkflowTask struct {
		task *workflowservice.PollWorkflowTaskQueueResponse
	}

	// activityTask wraps a activity task.
	activityTask struct {
		task   *workflowservice.PollActivityTaskQueueResponse
		permit *SlotPermit
	}

	// workflowExecutionContextImpl is the cached workflow state for sticky execution
	workflowExecutionContextImpl struct {
		mutex        sync.Mutex
		workflowInfo *WorkflowInfo
		wth          *workflowTaskHandlerImpl

		eventHandler *workflowExecutionEventHandler

		isWorkflowCompleted bool
		result              *commonpb.Payloads
		err                 error
		// previousStartedEventID is the event ID of the workflow task started event of the previous workflow task.
		previousStartedEventID int64
		// lastHandledEventID is the event ID of the last event that the workflow state machine processed.
		lastHandledEventID int64

		newCommands         []*commandpb.Command
		newMessages         []*protocolpb.Message
		currentWorkflowTask *workflowservice.PollWorkflowTaskQueueResponse
		laTunnel            *localActivityTunnel
		cached              bool
	}

	// workflowTaskHandlerImpl is the implementation of WorkflowTaskHandler
	workflowTaskHandlerImpl struct {
		namespace                 string
		metricsHandler            metrics.Handler
		ppMgr                     pressurePointMgr
		logger                    log.Logger
		identity                  string
		workerBuildID             string
		useBuildIDForVersioning   bool
		workerDeploymentVersion   WorkerDeploymentVersion
		defaultVersioningBehavior VersioningBehavior
		enableLoggingInReplay     bool
		registry                  *registry
		laTunnel                  *localActivityTunnel
		workflowPanicPolicy       WorkflowPanicPolicy
		dataConverter             converter.DataConverter
		failureConverter          converter.FailureConverter
		contextPropagators        []ContextPropagator
		cache                     *WorkerCache
		deadlockDetectionTimeout  time.Duration
		capabilities              *workflowservice.GetSystemInfoResponse_Capabilities
	}

	activityProvider func(name string) activity

	// activityTaskHandlerImpl is the implementation of ActivityTaskHandler
	activityTaskHandlerImpl struct {
		taskQueueName                    string
		identity                         string
		client                           *WorkflowClient
		metricsHandler                   metrics.Handler
		logger                           log.Logger
		backgroundContext                context.Context
		registry                         *registry
		activityProvider                 activityProvider
		dataConverter                    converter.DataConverter
		failureConverter                 converter.FailureConverter
		workerStopCh                     <-chan struct{}
		contextPropagators               []ContextPropagator
		namespace                        string
		defaultHeartbeatThrottleInterval time.Duration
		maxHeartbeatThrottleInterval     time.Duration
		versionStamp                     *commonpb.WorkerVersionStamp
		deployment                       *deploymentpb.Deployment
		workerDeploymentOptions          *deploymentpb.WorkerDeploymentOptions
	}

	// history wrapper method to help information about events.
	history struct {
		workflowTask       *workflowTask
		eventsHandler      *workflowExecutionEventHandlerImpl
		loadedEvents       []*historypb.HistoryEvent
		currentIndex       int
		nextEventID        int64 // next expected eventID for sanity
		lastEventID        int64 // last expected eventID, zero indicates read until end of stream
		lastHandledEventID int64 // last event ID that was processed
		next               []*historypb.HistoryEvent
		nextMessages       []*protocolpb.Message
		nextFlags          []sdkFlag
		binaryChecksum     string
		sdkVersion         string
		sdkName            string
	}

	workflowTaskHeartbeatError struct {
		Message string
	}

	historyMismatchError struct {
		message string
	}

	unknownSdkFlagError struct {
		message string
	}

	preparedTask struct {
		events         []*historypb.HistoryEvent
		markers        []*historypb.HistoryEvent
		flags          []sdkFlag
		acceptedMsgs   []*protocolpb.Message
		admittedMsgs   []*protocolpb.Message
		binaryChecksum string
		sdkVersion     string
		sdkName        string
		// Is null if there was no task completed event to read the build ID from (but may be
		// empty string if there was, and it was empty)
		buildID *string
	}

	finishedTask struct {
		isFailed       bool
		binaryChecksum string
		flags          []sdkFlag
		sdkVersion     string
		sdkName        string
	}
)

func newHistory(lastHandledEventID int64, task *workflowTask, eventsHandler *workflowExecutionEventHandlerImpl) *history {
	result := &history{
		workflowTask:       task,
		eventsHandler:      eventsHandler,
		loadedEvents:       task.task.History.Events,
		currentIndex:       0,
		lastEventID:        task.task.GetStartedEventId(),
		lastHandledEventID: lastHandledEventID,
	}
	if len(result.loadedEvents) > 0 {
		result.nextEventID = result.loadedEvents[0].GetEventId()
	}
	return result
}

func (e workflowTaskHeartbeatError) Error() string {
	return e.Message
}

func historyMismatchErrorf(f string, v ...interface{}) historyMismatchError {
	return historyMismatchError{message: fmt.Sprintf(f, v...)}
}

func (h historyMismatchError) Error() string {
	return h.message
}

func (s unknownSdkFlagError) Error() string {
	return s.message
}

// Get workflow start event.
func (eh *history) GetWorkflowStartedEvent() (*historypb.HistoryEvent, error) {
	events := eh.workflowTask.task.History.Events
	if len(events) == 0 || events[0].GetEventType() != enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED {
		return nil, errors.New("unable to find WorkflowExecutionStartedEventAttributes in the history")
	}
	return events[0], nil
}

func (eh *history) IsReplayEvent(event *historypb.HistoryEvent) bool {
	return event.GetEventId() <= eh.workflowTask.task.GetPreviousStartedEventId() || isCommandEvent(event.GetEventType())
}

// isNextWorkflowTaskFailed checks if the workflow task failed or completed. If it did complete returns some information
// on the completed workflow task.
func (eh *history) isNextWorkflowTaskFailed() (task finishedTask, err error) {
	nextIndex := eh.currentIndex + 1
	// Server can return an empty page so if we need the next event we must keep checking until we either get it
	// or know we have no more pages to check
	for nextIndex >= len(eh.loadedEvents) && eh.hasMoreEvents() { // current page ends and there is more pages
		if err := eh.loadMoreEvents(); err != nil {
			return finishedTask{}, err
		}
	}

	// If not replaying we should not expect to find any more events
	if nextIndex < len(eh.loadedEvents) {
		nextEvent := eh.loadedEvents[nextIndex]
		nextEventType := nextEvent.GetEventType()
		isFailed := nextEventType == enumspb.EVENT_TYPE_WORKFLOW_TASK_TIMED_OUT || nextEventType == enumspb.EVENT_TYPE_WORKFLOW_TASK_FAILED
		var binaryChecksum string
		var flags []sdkFlag
		if nextEventType == enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED {
			completedAttrs := nextEvent.GetWorkflowTaskCompletedEventAttributes()
			//lint:ignore SA1019 ignore deprecated versioning APIs
			binaryChecksum = completedAttrs.BinaryChecksum
			for _, flag := range completedAttrs.GetSdkMetadata().GetLangUsedFlags() {
				f := sdkFlagFromUint(flag)
				if !f.isValid() {
					// If a flag is not recognized (value is too high or not defined), it must fail the workflow task
					return finishedTask{}, unknownSdkFlagError{
						message: fmt.Sprintf("unknown SDK flag: %d", flag),
					}
				}
				flags = append(flags, f)
			}
		}
		return finishedTask{
			isFailed:       isFailed,
			binaryChecksum: binaryChecksum,
			flags:          flags,
			sdkName:        nextEvent.GetWorkflowTaskCompletedEventAttributes().GetSdkMetadata().GetSdkName(),
			sdkVersion:     nextEvent.GetWorkflowTaskCompletedEventAttributes().GetSdkMetadata().GetSdkVersion(),
		}, nil
	}
	return finishedTask{}, nil
}

func (eh *history) loadMoreEvents() error {
	historyPage, err := eh.getMoreEvents()
	if err != nil {
		return err
	}
	eh.loadedEvents = append(eh.loadedEvents, historyPage.Events...)
	if eh.nextEventID == 0 && len(eh.loadedEvents) > 0 {
		eh.nextEventID = eh.loadedEvents[0].GetEventId()
	}
	return nil
}

func isCommandEvent(eventType enumspb.EventType) bool {
	switch eventType {
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CONTINUED_AS_NEW,
		enumspb.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
		enumspb.EVENT_TYPE_ACTIVITY_TASK_CANCEL_REQUESTED,
		enumspb.EVENT_TYPE_TIMER_STARTED,
		enumspb.EVENT_TYPE_TIMER_CANCELED,
		enumspb.EVENT_TYPE_MARKER_RECORDED,
		enumspb.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED,
		enumspb.EVENT_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION_INITIATED,
		enumspb.EVENT_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_INITIATED,
		enumspb.EVENT_TYPE_UPSERT_WORKFLOW_SEARCH_ATTRIBUTES,
		enumspb.EVENT_TYPE_WORKFLOW_PROPERTIES_MODIFIED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_REJECTED,
		enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED,
		enumspb.EVENT_TYPE_NEXUS_OPERATION_CANCEL_REQUESTED:
		return true
	default:
		return false
	}
}

// nextTask returns the next task to be processed.
func (eh *history) nextTask() (*preparedTask, error) {
	if eh.next == nil {
		firstTask, err := eh.prepareTask()
		if err != nil {
			return nil, err
		}
		eh.next = firstTask.events
		eh.nextMessages = firstTask.admittedMsgs
		eh.nextFlags = firstTask.flags
		eh.sdkName = firstTask.sdkName
		eh.sdkVersion = firstTask.sdkVersion
	}

	result := eh.next
	requestMessages := eh.nextMessages
	checksum := eh.binaryChecksum
	sdkFlags := eh.nextFlags
	sdkName := eh.sdkName
	sdkVersion := eh.sdkVersion

	var markers []*historypb.HistoryEvent
	var acceptedMsgs []*protocolpb.Message
	var buildID *string
	if len(result) > 0 {
		nextTaskEvents, err := eh.prepareTask()
		if err != nil {
			return nil, err
		}
		eh.next = nextTaskEvents.events
		eh.nextMessages = nextTaskEvents.admittedMsgs
		eh.nextFlags = nextTaskEvents.flags
		eh.sdkName = nextTaskEvents.sdkName
		eh.sdkVersion = nextTaskEvents.sdkVersion
		markers = nextTaskEvents.markers
		acceptedMsgs = nextTaskEvents.acceptedMsgs
		buildID = nextTaskEvents.buildID
	}
	return &preparedTask{
		events:         result,
		markers:        markers,
		flags:          sdkFlags,
		acceptedMsgs:   acceptedMsgs,
		admittedMsgs:   requestMessages,
		binaryChecksum: checksum,
		sdkName:        sdkName,
		sdkVersion:     sdkVersion,
		buildID:        buildID,
	}, nil
}

func (eh *history) hasMoreEvents() bool {
	historyIterator := eh.workflowTask.historyIterator
	return historyIterator != nil && historyIterator.HasNextPage()
}

func (eh *history) getMoreEvents() (*historypb.History, error) {
	return eh.workflowTask.historyIterator.GetNextPage()
}

func (eh *history) verifyAllEventsProcessed() error {
	if eh.lastEventID > 0 && eh.nextEventID <= eh.lastEventID {
		return fmt.Errorf(
			"history_events: premature end of stream, expectedLastEventID=%v but no more events after eventID=%v",
			eh.lastEventID,
			eh.nextEventID-1)
	}
	if eh.lastEventID > 0 && eh.nextEventID != (eh.lastEventID+1) {
		eh.eventsHandler.logger.Warn(
			"history_events: processed events past the expected lastEventID",
			"expectedLastEventID", eh.lastEventID,
			"processedLastEventID", eh.nextEventID-1)
	}
	return nil
}

func (eh *history) prepareTask() (*preparedTask, error) {
	if eh.currentIndex == len(eh.loadedEvents) && !eh.hasMoreEvents() {
		if err := eh.verifyAllEventsProcessed(); err != nil {
			return nil, err
		}
		return &preparedTask{}, nil
	}

	// Process events
	var taskEvents preparedTask
OrderEvents:
	for {
		// load more history events if needed
		for eh.currentIndex == len(eh.loadedEvents) {
			if !eh.hasMoreEvents() {
				if err := eh.verifyAllEventsProcessed(); err != nil {
					return nil, err
				}
				break OrderEvents
			}
			if err := eh.loadMoreEvents(); err != nil {
				return nil, err
			}
		}

		event := eh.loadedEvents[eh.currentIndex]
		eventID := event.GetEventId()
		if eventID != eh.nextEventID {
			err := fmt.Errorf(
				"missing history events, expectedNextEventID=%v but receivedNextEventID=%v",
				eh.nextEventID, eventID)
			return nil, err
		}

		eh.nextEventID++
		if eventID <= eh.lastHandledEventID {
			eh.currentIndex++
			continue
		}
		eh.lastHandledEventID = eventID

		switch event.GetEventType() {
		case enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED:
			finishedTask, err1 := eh.isNextWorkflowTaskFailed()
			if err1 != nil {
				err := err1
				return nil, err
			}
			if !finishedTask.isFailed {
				eh.binaryChecksum = finishedTask.binaryChecksum
				eh.currentIndex++
				taskEvents.events = append(taskEvents.events, event)
				taskEvents.flags = append(taskEvents.flags, finishedTask.flags...)
				if finishedTask.sdkName != "" {
					taskEvents.sdkName = finishedTask.sdkName
				}
				if finishedTask.sdkVersion != "" {
					taskEvents.sdkVersion = finishedTask.sdkVersion
				}
				break OrderEvents
			}
		case enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED,
			enumspb.EVENT_TYPE_WORKFLOW_TASK_TIMED_OUT:
			// Skip
		default:
			if event.GetEventType() == enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED {
				bidStr := event.GetWorkflowTaskCompletedEventAttributes().
					GetDeploymentVersion().GetBuildId()
				if bidStr == "" {
					//lint:ignore SA1019 ignore deprecated versioning APIs
					version := event.GetWorkflowTaskCompletedEventAttributes().GetWorkerDeploymentVersion()
					if splitVersion := strings.SplitN(version, ".", 2); len(splitVersion) == 2 {
						bidStr = splitVersion[1]
					}
				}
				if bidStr == "" {
					//lint:ignore SA1019 ignore deprecated versioning APIs
					bidStr = event.GetWorkflowTaskCompletedEventAttributes().
						GetWorkerVersion().GetBuildId()
				}
				taskEvents.buildID = &bidStr
			} else if isPreloadMarkerEvent(event) {
				taskEvents.markers = append(taskEvents.markers, event)
			} else if attrs := event.GetWorkflowExecutionUpdateAcceptedEventAttributes(); attrs != nil {
				taskEvents.acceptedMsgs = append(taskEvents.acceptedMsgs, inferMessageFromAcceptedEvent(attrs))
			} else if attrs := event.GetWorkflowExecutionUpdateAdmittedEventAttributes(); attrs != nil {
				updateID := attrs.GetRequest().GetMeta().GetUpdateId()
				taskEvents.admittedMsgs = append(taskEvents.admittedMsgs, &protocolpb.Message{
					Id:                 updateID + "/request",
					ProtocolInstanceId: updateID,
					SequencingId: &protocolpb.Message_EventId{
						EventId: event.GetEventId(),
					},
					Body: protocol.MustMarshalAny(attrs.GetRequest()),
				})
			}
			taskEvents.events = append(taskEvents.events, event)
		}
		eh.currentIndex++
	}

	// shrink loaded events so it can be GCed
	eh.loadedEvents = append(
		make(
			[]*historypb.HistoryEvent,
			0,
			len(eh.loadedEvents)-eh.currentIndex),
		eh.loadedEvents[eh.currentIndex:]...,
	)

	eh.currentIndex = 0

	return &taskEvents, nil
}

func isPreloadMarkerEvent(event *historypb.HistoryEvent) bool {
	return event.GetEventType() == enumspb.EVENT_TYPE_MARKER_RECORDED
}

func inferMessageFromAcceptedEvent(attrs *historypb.WorkflowExecutionUpdateAcceptedEventAttributes) *protocolpb.Message {
	return &protocolpb.Message{
		Id:                 attrs.GetAcceptedRequestMessageId(),
		ProtocolInstanceId: attrs.GetProtocolInstanceId(),
		SequencingId: &protocolpb.Message_EventId{
			EventId: attrs.GetAcceptedRequestSequencingEventId(),
		},
		Body: protocol.MustMarshalAny(attrs.GetAcceptedRequest()),
	}
}

// newWorkflowTaskHandler returns an implementation of workflow task handler.
func newWorkflowTaskHandler(params workerExecutionParameters, ppMgr pressurePointMgr, registry *registry) WorkflowTaskHandler {
	ensureRequiredParams(&params)
	return &workflowTaskHandlerImpl{
		namespace:                 params.Namespace,
		logger:                    params.Logger,
		ppMgr:                     ppMgr,
		metricsHandler:            params.MetricsHandler,
		identity:                  params.Identity,
		workerBuildID:             params.getBuildID(),
		useBuildIDForVersioning:   params.UseBuildIDForVersioning,
		workerDeploymentVersion:   params.WorkerDeploymentVersion,
		defaultVersioningBehavior: params.DefaultVersioningBehavior,
		enableLoggingInReplay:     params.EnableLoggingInReplay,
		registry:                  registry,
		workflowPanicPolicy:       params.WorkflowPanicPolicy,
		dataConverter:             params.DataConverter,
		failureConverter:          params.FailureConverter,
		contextPropagators:        params.ContextPropagators,
		cache:                     params.cache,
		deadlockDetectionTimeout:  params.DeadlockDetectionTimeout,
		capabilities:              params.capabilities,
	}
}

func newWorkflowExecutionContext(
	workflowInfo *WorkflowInfo,
	taskHandler *workflowTaskHandlerImpl,
) *workflowExecutionContextImpl {
	workflowContext := &workflowExecutionContextImpl{
		workflowInfo: workflowInfo,
		wth:          taskHandler,
	}
	workflowContext.createEventHandler()
	return workflowContext
}

// Lock acquires the lock on this context object, use Unlock(error) to release
// the lock.
func (w *workflowExecutionContextImpl) Lock() {
	w.mutex.Lock()
}

// Unlock cleans up after the provided error and its own internal view of the
// workflow error state by clearing itself and removing itself from cache as
// needed. It is an error to call this function without having called the Lock
// function first and the behavior is undefined. Regardless of the error
// handling involved, the context will be unlocked when this call returns.
func (w *workflowExecutionContextImpl) Unlock(err error) {
	defer w.mutex.Unlock()
	if err != nil || w.err != nil || w.isWorkflowCompleted ||
		(w.wth.cache.MaxWorkflowCacheSize() <= 0 && !w.hasPendingLocalActivityWork()) {
		// TODO: in case of closed, it assumes the close command always succeed. need server side change to return
		// error to indicate the close failure case. This should be a rare case. For now, always remove the cache, and
		// if the close command failed, the next command will have to rebuild the state.
		if w.wth.cache.getWorkflowCache().Exist(w.workflowInfo.WorkflowExecution.RunID) {
			w.wth.cache.removeWorkflowContext(w.workflowInfo.WorkflowExecution.RunID)
			w.cached = false
		}
		// Clear the state so other tasks waiting on the context know it should be discarded.
		w.clearState()
	} else if !w.cached {
		// Clear the state if we never cached the workflow so coroutines can be
		// exited
		w.clearState()
	}
}

func (w *workflowExecutionContextImpl) getEventHandler() *workflowExecutionEventHandlerImpl {
	if w.eventHandler == nil {
		return nil
	}
	return (*w.eventHandler).(*workflowExecutionEventHandlerImpl)
}

func (w *workflowExecutionContextImpl) completeWorkflow(result *commonpb.Payloads, err error) {
	w.isWorkflowCompleted = true
	w.result = result
	w.err = err
}

func (w *workflowExecutionContextImpl) onEviction() {
	// onEviction is run by LRU cache's removeFunc in separate goroutinue
	w.mutex.Lock()

	// Emit force eviction metrics.
	// This metrics indicates too many concurrent running workflows to fit in sticky cache.
	// Eviction on error or on workflow complete is normal and expected.
	if w.err == nil && !w.isWorkflowCompleted {
		w.wth.metricsHandler.Counter(metrics.StickyCacheTotalForcedEviction).Inc(1)
	}

	w.clearState()
	w.mutex.Unlock()
}

func (w *workflowExecutionContextImpl) IsDestroyed() bool {
	return w.getEventHandler() == nil
}

func (w *workflowExecutionContextImpl) clearState() {
	w.clearCurrentTask()
	w.isWorkflowCompleted = false
	w.result = nil
	w.err = nil
	w.previousStartedEventID = 0
	w.lastHandledEventID = 0
	w.newCommands = nil
	w.newMessages = nil

	eventHandler := w.getEventHandler()
	if eventHandler != nil {
		// Set isReplay to true to prevent user code in defer guarded by !isReplaying() from running
		eventHandler.isReplay = true
		eventHandler.Close()
		w.eventHandler = nil
	}
}

func (w *workflowExecutionContextImpl) createEventHandler() {
	w.clearState()
	eventHandler := newWorkflowExecutionEventHandler(
		w.workflowInfo,
		w.completeWorkflow,
		w.wth.logger,
		w.wth.enableLoggingInReplay,
		w.wth.metricsHandler,
		w.wth.registry,
		w.wth.dataConverter,
		w.wth.failureConverter,
		w.wth.contextPropagators,
		w.wth.deadlockDetectionTimeout,
		w.wth.capabilities,
	)

	w.eventHandler = &eventHandler
}

func resetHistory(task *workflowservice.PollWorkflowTaskQueueResponse, historyIterator HistoryIterator) (*historypb.History, error) {
	historyIterator.Reset()
	firstPageHistory, err := historyIterator.GetNextPage()
	if err != nil {
		return nil, err
	}
	task.History = firstPageHistory
	return firstPageHistory, nil
}

func (wth *workflowTaskHandlerImpl) createWorkflowContext(task *workflowservice.PollWorkflowTaskQueueResponse) (*workflowExecutionContextImpl, error) {
	h := task.History
	startedEvent := h.Events[0]
	attributes := startedEvent.GetWorkflowExecutionStartedEventAttributes()
	if attributes == nil {
		return nil, errors.New("first history event is not WorkflowExecutionStarted")
	}
	taskQueue := attributes.TaskQueue
	if taskQueue == nil || taskQueue.Name == "" {
		return nil, errors.New("nil or empty TaskQueue in WorkflowExecutionStarted event")
	}

	runID := task.WorkflowExecution.GetRunId()
	workflowID := task.WorkflowExecution.GetWorkflowId()

	// Setup workflow Info
	var parentWorkflowExecution *WorkflowExecution
	if attributes.ParentWorkflowExecution != nil {
		parentWorkflowExecution = &WorkflowExecution{
			ID:    attributes.ParentWorkflowExecution.GetWorkflowId(),
			RunID: attributes.ParentWorkflowExecution.GetRunId(),
		}
	}

	var rootWorkflowExecution *WorkflowExecution
	if attributes.RootWorkflowExecution != nil {
		rootWorkflowExecution = &WorkflowExecution{
			ID:    attributes.RootWorkflowExecution.GetWorkflowId(),
			RunID: attributes.RootWorkflowExecution.GetRunId(),
		}
	}

	workflowInfo := &WorkflowInfo{
		WorkflowExecution: WorkflowExecution{
			ID:    workflowID,
			RunID: runID,
		},
		OriginalRunID:            attributes.OriginalExecutionRunId,
		FirstRunID:               attributes.FirstExecutionRunId,
		WorkflowType:             WorkflowType{Name: task.WorkflowType.GetName()},
		TaskQueueName:            taskQueue.GetName(),
		WorkflowExecutionTimeout: attributes.GetWorkflowExecutionTimeout().AsDuration(),
		WorkflowRunTimeout:       attributes.GetWorkflowRunTimeout().AsDuration(),
		WorkflowTaskTimeout:      attributes.GetWorkflowTaskTimeout().AsDuration(),
		Namespace:                wth.namespace,
		Attempt:                  attributes.GetAttempt(),
		WorkflowStartTime:        startedEvent.GetEventTime().AsTime(),
		lastCompletionResult:     attributes.LastCompletionResult,
		lastFailure:              attributes.ContinuedFailure,
		CronSchedule:             attributes.CronSchedule,
		ContinuedExecutionRunID:  attributes.ContinuedExecutionRunId,
		ParentWorkflowNamespace:  attributes.ParentWorkflowNamespace,
		ParentWorkflowExecution:  parentWorkflowExecution,
		RootWorkflowExecution:    rootWorkflowExecution,
		Memo:                     attributes.Memo,
		SearchAttributes:         attributes.SearchAttributes,
		RetryPolicy:              convertFromPBRetryPolicy(attributes.RetryPolicy),
		// Use the original execution run ID from the start event as the initial seed.
		// Original execution run ID stays the same for the entire chain of workflow resets.
		// This helps us keep child workflow IDs consistent up until a reset-point is encountered.
		currentRunID: attributes.GetOriginalExecutionRunId(),
		Priority:     convertFromPBPriority(attributes.Priority),
	}

	return newWorkflowExecutionContext(workflowInfo, wth), nil
}

func (wth *workflowTaskHandlerImpl) GetOrCreateWorkflowContext(
	task *workflowservice.PollWorkflowTaskQueueResponse,
	historyIterator HistoryIterator,
) (workflowContext *workflowExecutionContextImpl, err error) {
	metricsHandler := wth.metricsHandler.WithTags(metrics.WorkflowTags(task.WorkflowType.GetName()))
	defer func() {
		if err == nil && workflowContext != nil && workflowContext.laTunnel == nil {
			workflowContext.laTunnel = wth.laTunnel
		}
		metricsHandler.Gauge(metrics.StickyCacheSize).Update(float64(wth.cache.getWorkflowCache().Size()))
	}()

	runID := task.WorkflowExecution.GetRunId()

	history := task.History
	isFullHistory := isFullHistory(history)

	workflowContext = nil
	if task.Query == nil || (task.Query != nil && !isFullHistory) {
		workflowContext = wth.cache.getWorkflowContext(runID)
	}
	// Verify the cached state is current and for the correct worker
	if workflowContext != nil {
		workflowContext.Lock()
		if task.Query != nil && !isFullHistory && wth == workflowContext.wth && !workflowContext.IsDestroyed() {
			// query task and we have a valid cached state
			metricsHandler.Counter(metrics.StickyCacheHit).Inc(1)
		} else if len(history.Events) > 0 && history.Events[0].GetEventId() == workflowContext.previousStartedEventID+1 && wth == workflowContext.wth && !workflowContext.IsDestroyed() {
			// non query task and we have a valid cached state
			metricsHandler.Counter(metrics.StickyCacheHit).Inc(1)
		} else {
			// possible another task already destroyed this context.
			if !workflowContext.IsDestroyed() {
				// non query task and cached state is missing events, we need to discard the cached state and build a new one.
				if len(history.Events) > 0 && history.Events[0].GetEventId() != workflowContext.previousStartedEventID+1 {
					wth.logger.Debug("Cached state staled, new task has unexpected events",
						tagWorkflowID, task.WorkflowExecution.GetWorkflowId(),
						tagRunID, task.WorkflowExecution.GetRunId(),
						tagAttempt, task.Attempt,
						tagCachedPreviousStartedEventID, workflowContext.previousStartedEventID,
						tagTaskFirstEventID, task.History.Events[0].GetEventId(),
						tagTaskStartedEventID, task.GetStartedEventId(),
						tagPreviousStartedEventID, task.GetPreviousStartedEventId(),
					)
				} else {
					wth.logger.Debug("Cached state started on different worker, creating new context")
				}
				wth.cache.removeWorkflowContext(runID)
				workflowContext.clearState()
			}
			workflowContext.Unlock(err)
			workflowContext = nil
		}
	}
	// If the workflow was not cached or the cache was stale.
	if workflowContext == nil {
		if !isFullHistory {
			// we are getting partial history task, but cached state was already evicted.
			// we need to reset history so we get events from beginning to replay/rebuild the state
			metricsHandler.Counter(metrics.StickyCacheMiss).Inc(1)
			if _, err = resetHistory(task, historyIterator); err != nil {
				return
			}
		}

		if workflowContext, err = wth.createWorkflowContext(task); err != nil {
			return
		}

		if wth.cache.MaxWorkflowCacheSize() > 0 && task.Query == nil {
			workflowContext, _ = wth.cache.putWorkflowContext(runID, workflowContext)
			workflowContext.Lock()
			workflowContext.cached = true
		} else {
			workflowContext.Lock()
		}
	}

	err = workflowContext.resetStateIfDestroyed(task, historyIterator)
	if err != nil {
		workflowContext.Unlock(err)
	}

	return
}

func isFullHistory(history *historypb.History) bool {
	if len(history.Events) == 0 || history.Events[0].GetEventType() != enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED {
		return false
	}
	return true
}

func (w *workflowExecutionContextImpl) resetStateIfDestroyed(task *workflowservice.PollWorkflowTaskQueueResponse, historyIterator HistoryIterator) error {
	// It is possible that 2 threads (one for workflow task and one for query task) that both are getting this same
	// cached workflowContext. If one task finished with err, it would destroy the cached state. In that case, the
	// second task needs to reset the cache state and start from beginning of the history.
	if w.IsDestroyed() {
		w.createEventHandler()
		// reset history events if necessary
		if !isFullHistory(task.History) {
			if _, err := resetHistory(task, historyIterator); err != nil {
				return err
			}
		}
		if w.workflowInfo != nil {
			// Reset the search attributes and memos from the WorkflowExecutionStartedEvent.
			// The search attributes and memo may have been modified by calls like UpsertMemo
			// or UpsertSearchAttributes. They must be reset to avoid non determinism on replay.
			h := task.History
			startedEvent := h.Events[0]
			attributes := startedEvent.GetWorkflowExecutionStartedEventAttributes()
			if attributes == nil {
				return errors.New("first history event is not WorkflowExecutionStarted")
			}
			w.workflowInfo.SearchAttributes = attributes.SearchAttributes
			w.workflowInfo.Memo = attributes.Memo
		}
	}
	return nil
}

// ProcessWorkflowTask processes all the events of the workflow task.
func (wth *workflowTaskHandlerImpl) ProcessWorkflowTask(
	workflowTask *workflowTask,
	workflowContext *workflowExecutionContextImpl,
	heartbeatFunc workflowTaskHeartbeatFunc,
) (completeRequest interface{}, errRet error) {
	if workflowTask == nil || workflowTask.task == nil {
		return nil, errors.New("nil workflow task provided")
	}
	task := workflowTask.task
	if task.History == nil || len(task.History.Events) == 0 {
		task.History = &historypb.History{
			Events: []*historypb.HistoryEvent{},
		}
	}
	if task.Query == nil && len(task.History.Events) == 0 {
		return nil, errors.New("nil or empty history")
	}

	if task.Query != nil && len(task.Queries) != 0 {
		return nil, errors.New("invalid query workflow task")
	}

	runID := task.WorkflowExecution.GetRunId()
	workflowID := task.WorkflowExecution.GetWorkflowId()
	traceLog(func() {
		wth.logger.Debug("Processing new workflow task.",
			tagWorkflowType, task.WorkflowType.GetName(),
			tagWorkflowID, workflowID,
			tagRunID, runID,
			tagAttempt, task.Attempt,
			tagPreviousStartedEventID, task.GetPreviousStartedEventId())
	})

	var (
		response       interface{}
		err            error
		heartbeatTimer *time.Timer
	)

	defer func() {
		if heartbeatTimer != nil {
			heartbeatTimer.Stop()
		}
	}()

processWorkflowLoop:
	for {
		startTime := time.Now()
		response, err = workflowContext.ProcessWorkflowTask(workflowTask)
		if err == nil && response == nil {
		waitLocalActivityLoop:
			for {
				deadlineToTrigger := time.Duration(float32(ratioToForceCompleteWorkflowTaskComplete) * float32(workflowContext.workflowInfo.WorkflowTaskTimeout))
				delayDuration := time.Until(startTime.Add(deadlineToTrigger))

			heartbeatLoop:
				for {
					if delayDuration <= 0 {
						if heartbeatTimer != nil {
							heartbeatTimer.Stop()
							heartbeatTimer = nil
						}

						// For non-graceful shutdown, the LA worker stops before this function, so there
						// is no need to continue heartbeating. Instead, we can exit early, giving up
						// the slot this function takes, a little sooner.
						select {
						case <-workflowContext.laTunnel.stopCh:
							// stopCh closed means worker is shutting down and there's
							// no need for LA heartbeat
							return
						default:
							// force complete, call the workflow task heartbeat function
							workflowTask, err = heartbeatFunc(
								workflowContext.CompleteWorkflowTask(workflowTask, false),
								startTime,
							)
							if err != nil {
								errRet = &workflowTaskHeartbeatError{Message: fmt.Sprintf("error sending workflow task heartbeat %v", err)}
								return
							}
							if workflowTask == nil {
								return
							}
						}

						continue processWorkflowLoop
					}

					if heartbeatTimer == nil {
						heartbeatTimer = time.NewTimer(delayDuration)
					}

					select {
					case <-heartbeatTimer.C:
						delayDuration = 0
						continue heartbeatLoop

					case laRetry := <-workflowTask.laRetryCh:
						eventHandler := workflowContext.getEventHandler()

						// if workflow task heartbeat failed, the workflow execution context will be cleared and eventHandler will be nil
						if eventHandler == nil {
							break processWorkflowLoop
						}

						if _, ok := eventHandler.pendingLaTasks[laRetry.activityID]; !ok {
							break processWorkflowLoop
						}

						laRetry.attempt++

						if !wth.laTunnel.sendTask(laRetry) {
							laRetry.attempt--
						}

					case lar := <-workflowTask.laResultCh:
						// local activity result ready
						response, err = workflowContext.ProcessLocalActivityResult(workflowTask, lar)
						if err == nil && response == nil {
							// workflow task is not done yet, still waiting for more local activities
							continue waitLocalActivityLoop
						}
						break processWorkflowLoop
					}
				}
			}
		} else {
			break processWorkflowLoop
		}
	}
	errRet = err
	completeRequest = response
	return
}

func (w *workflowExecutionContextImpl) ProcessWorkflowTask(workflowTask *workflowTask) (interface{}, error) {
	task := workflowTask.task
	historyIterator := workflowTask.historyIterator
	if err := w.ResetIfStale(task, historyIterator); err != nil {
		return nil, err
	}
	w.SetCurrentTask(task)

	eventHandler := w.getEventHandler()
	reorderedHistory := newHistory(w.lastHandledEventID, workflowTask, eventHandler)
	defer func() {
		// After processing the workflow task, update the last handled event ID
		// to the last event ID in the history. We do this regardless of whether the workflow task
		// was successfully processed or not. This is because a failed workflow task will cause the
		// cache to be evicted and the next workflow task will start from the beginning of the history.
		w.lastHandledEventID = reorderedHistory.lastHandledEventID
	}()
	var replayOutbox []outboxEntry
	var replayCommands []*commandpb.Command
	var respondEvents []*historypb.HistoryEvent
	var partialHistory bool

	taskMessages := workflowTask.task.GetMessages()
	skipReplayCheck := w.skipReplayCheck()
	isInReplayer := IsReplayNamespace(w.wth.namespace)
	shouldForceReplayCheck := func() bool {
		// If we are in the replayer we should always check the history replay, even if the workflow is completed
		// Skip if the workflow panicked to avoid potentially breaking old histories
		_, wfPanicked := w.err.(*workflowPanicError)
		return !wfPanicked && isInReplayer
	}

	curReplayCmdsIndex := -1

	metricsHandler := w.wth.metricsHandler.WithTags(metrics.WorkflowTags(task.WorkflowType.GetName()))
	start := time.Now()
	// This is set to nil once recorded
	metricsTimer := metricsHandler.Timer(metrics.WorkflowTaskReplayLatency)

	eventHandler.ResetLAWFTAttemptCounts()
	eventHandler.sdkFlags.markSDKFlagsSent()

	w.workflowInfo.currentTaskBuildID = w.wth.workerBuildID
ProcessEvents:
	for {
		nextTask, err := reorderedHistory.nextTask()
		if err != nil {
			return nil, err
		}
		reorderedEvents := nextTask.events
		markers := nextTask.markers
		historyMessages := nextTask.acceptedMsgs
		flags := nextTask.flags
		binaryChecksum := nextTask.binaryChecksum
		nextTaskBuildId := nextTask.buildID
		admittedUpdates := nextTask.admittedMsgs

		// Peak ahead to confirm there are no more events
		isLastWFTForPartialWFE := len(reorderedEvents) > 0 &&
			reorderedEvents[len(reorderedEvents)-1].EventType == enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED &&
			len(reorderedHistory.next) == 0 &&
			isInReplayer
		if isLastWFTForPartialWFE {
			partialHistory = true
			break ProcessEvents
		}

		// Check if we are replaying so we know if we should use the messages in the WFT or the history
		isReplay := len(reorderedEvents) > 0 && reorderedHistory.IsReplayEvent(reorderedEvents[len(reorderedEvents)-1])
		var msgs *eventMsgIndex
		if isReplay {
			admittedUpdatesByID := make(map[string]*protocolpb.Message, len(admittedUpdates))
			for _, admittedUpdate := range admittedUpdates {
				admittedUpdatesByID[admittedUpdate.GetProtocolInstanceId()] = admittedUpdate
			}
			// Check if we need to replace the update message synthesize from an
			// accepted event with the update message synthesize from an admitted event
			for i, msg := range historyMessages {
				if admittedUpdate, ok := admittedUpdatesByID[msg.GetProtocolInstanceId()]; ok {
					historyMessages[i] = admittedUpdate
				}
				// At this point, all update messages should have a body
				if historyMessages[i].Body == nil {
					return nil, fmt.Errorf("missing body in message for update ID %v", msg.GetProtocolInstanceId())
				}
			}
			msgs = indexMessagesByEventID(historyMessages)

			eventHandler.sdkVersion = nextTask.sdkVersion
			eventHandler.sdkName = nextTask.sdkName
		} else {
			taskMessages = append(taskMessages, admittedUpdates...)
			msgs = indexMessagesByEventID(taskMessages)
			taskMessages = []*protocolpb.Message{}
			if eventHandler.sdkVersion != SDKVersion {
				eventHandler.sdkVersionUpdated = true
				eventHandler.sdkVersion = SDKVersion
			}
			if eventHandler.sdkName != SDKName {
				eventHandler.sdkNameUpdated = true
				eventHandler.sdkName = SDKName
			}
		}

		eventHandler.sdkFlags.set(flags...)
		if len(reorderedEvents) == 0 {
			break ProcessEvents
		}
		// Since replayCommands updates a loop early, keep track of index before the
		// early update to handle replaying incomplete WFE
		curReplayCmdsIndex = len(replayCommands)

		if binaryChecksum == "" {
			w.workflowInfo.BinaryChecksum = w.wth.workerBuildID
		} else {
			w.workflowInfo.BinaryChecksum = binaryChecksum
		}
		if isReplay && nextTaskBuildId != nil {
			w.workflowInfo.currentTaskBuildID = *nextTaskBuildId
		}
		// Reset the mutable side effect markers recorded
		eventHandler.mutableSideEffectsRecorded = nil
		// Markers are from the events that are produced from the current workflow task.
		for _, m := range markers {
			if m.GetMarkerRecordedEventAttributes().GetMarkerName() != localActivityMarkerName {
				// local activity marker needs to be applied after workflow task started event
				err := eventHandler.ProcessEvent(m, true, false)
				if err != nil {
					return nil, err
				}
				if w.isWorkflowCompleted && !shouldForceReplayCheck() {
					break ProcessEvents
				}
			}
		}

		for i, event := range reorderedEvents {
			isInReplay := reorderedHistory.IsReplayEvent(event)
			if !isInReplay && metricsTimer != nil {
				metricsTimer.Record(time.Since(start))
				metricsTimer = nil
			}

			isLast := !isInReplay && i == len(reorderedEvents)-1
			if !skipReplayCheck && isCommandEvent(event.GetEventType()) {
				respondEvents = append(respondEvents, event)
			}

			if isPreloadMarkerEvent(event) {
				// marker events are processed separately
				continue
			}

			// Any pressure points.
			err := w.wth.executeAnyPressurePoints(event, isInReplay)
			if err != nil {
				return nil, err
			}

			// because we don't run all events through this code path, we have
			// to run ProcessMessages both before and after ProcessEvent to
			// catch any messages that should have been delivered _before_ this
			// event but perhaps were not because there were attached to an
			// event (e.g. WFTScheduledEvent) that does not come through this
			// loop.
			for _, msg := range msgs.takeLTE(event.GetEventId() - 1) {
				err := eventHandler.ProcessMessage(msg, isInReplay, isLast)
				if err != nil {
					return nil, err
				}
				if w.isWorkflowCompleted && !shouldForceReplayCheck() {
					break ProcessEvents
				}
			}

			err = eventHandler.ProcessEvent(event, isInReplay, isLast)
			if err != nil {
				return nil, err
			}
			if w.isWorkflowCompleted && !shouldForceReplayCheck() {
				break ProcessEvents
			}

			for _, msg := range msgs.takeLTE(event.GetEventId()) {
				err := eventHandler.ProcessMessage(msg, isInReplay, isLast)
				if err != nil {
					return nil, err
				}
				if w.isWorkflowCompleted && !shouldForceReplayCheck() {
					break ProcessEvents
				}
			}
		}

		// now apply local activity markers
		for _, m := range markers {
			if m.GetMarkerRecordedEventAttributes().GetMarkerName() == localActivityMarkerName {
				err := eventHandler.ProcessEvent(m, true, false)
				if err != nil {
					return nil, err
				}
				if w.isWorkflowCompleted && !shouldForceReplayCheck() {
					break ProcessEvents
				}
			}
		}
		if isReplay {
			eventCommands := eventHandler.commandsHelper.getCommands(true)
			if !skipReplayCheck {
				replayCommands = append(replayCommands, eventCommands...)
				replayOutbox = append(replayOutbox, eventHandler.outbox...)
			}
			eventHandler.outbox = nil
		}
	}

	if partialHistory && curReplayCmdsIndex != -1 {
		replayCommands = replayCommands[:curReplayCmdsIndex]
	}

	if metricsTimer != nil {
		metricsTimer.Record(time.Since(start))
		metricsTimer = nil
	}

	// Non-deterministic error could happen in 2 different places:
	//   1) the replay commands does not match to history events. This is usually due to non backwards compatible code
	// change to workflow logic. For example, change calling one activity to a different activity.
	//   2) the command state machine is trying to make illegal state transition while replay a history event (like
	// activity task completed), but the corresponding workflow code that start the event has been removed. In that case
	// the replay of that event will panic on the command state machine and the workflow will be marked as completed
	// with the panic error.
	var workflowError error
	if !skipReplayCheck && (!w.isWorkflowCompleted || shouldForceReplayCheck()) {
		// check if commands from reply matches to the history events
		if err := matchReplayWithHistory(replayCommands, respondEvents, replayOutbox, w.getEventHandler().sdkFlags); err != nil {
			workflowError = err
			w.err = err
		}
	}

	return w.applyWorkflowPanicPolicy(workflowTask, workflowError)
}

func (w *workflowExecutionContextImpl) ProcessLocalActivityResult(workflowTask *workflowTask, lar *localActivityResult) (interface{}, error) {
	if lar.err != nil && w.retryLocalActivity(lar) {
		return nil, nil // nothing to do here as we are retrying...
	}

	return w.applyWorkflowPanicPolicy(workflowTask, w.getEventHandler().ProcessLocalActivityResult(lar))
}

func (w *workflowExecutionContextImpl) applyWorkflowPanicPolicy(workflowTask *workflowTask, workflowError error) (interface{}, error) {
	task := workflowTask.task

	if workflowError == nil && w.err != nil {
		if panicErr, ok := w.err.(*workflowPanicError); ok {
			workflowError = panicErr
		}
	}

	if workflowError != nil {
		if panicErr, ok := w.err.(*workflowPanicError); ok {
			w.wth.logger.Error("Workflow panic",
				tagWorkflowType, task.WorkflowType.GetName(),
				tagWorkflowID, task.WorkflowExecution.GetWorkflowId(),
				tagRunID, task.WorkflowExecution.GetRunId(),
				tagAttempt, task.Attempt,
				tagError, workflowError,
				tagStackTrace, panicErr.StackTrace())
		} else {
			w.wth.logger.Error("Workflow panic",
				tagWorkflowType, task.WorkflowType.GetName(),
				tagWorkflowID, task.WorkflowExecution.GetWorkflowId(),
				tagRunID, task.WorkflowExecution.GetRunId(),
				tagAttempt, task.Attempt,
				tagError, workflowError)
		}

		switch w.wth.workflowPanicPolicy {
		case FailWorkflow:
			// complete workflow with custom error will fail the workflow
			w.getEventHandler().Complete(nil, NewApplicationError(
				"Workflow failed on panic due to FailWorkflow workflow panic policy",
				"", false, workflowError))
		case BlockWorkflow:
			// return error here will be convert to WorkflowTaskFailed for the first time, and ignored for subsequent
			// attempts which will cause WorkflowTaskTimeout and server will retry forever until issue got fixed or
			// workflow timeout.
			return nil, workflowError
		default:
			panic("unknown mismatched workflow history policy.")
		}
	}

	return w.CompleteWorkflowTask(workflowTask, true), nil
}

func (w *workflowExecutionContextImpl) retryLocalActivity(lar *localActivityResult) bool {
	if lar.task.retryPolicy == nil || lar.err == nil || IsCanceledError(lar.err) {
		return false
	}

	retryBackoff := getRetryBackoff(lar, time.Now())
	if retryBackoff > 0 && retryBackoff <= w.workflowInfo.WorkflowTaskTimeout {
		// we need a local retry
		time.AfterFunc(retryBackoff, func() {
			// Send retry signal
			select {
			case lar.task.workflowTask.laRetryCh <- lar.task:
			case <-lar.task.workflowTask.doneCh:
				// Task is already done. Abort retrying.
			}
		})
		return true
	}
	// Backoff could be large and potentially much larger than WorkflowTaskTimeout. We cannot just sleep locally for
	// retry. Because it will delay the local activity from complete which keeps the workflow task open. In order to
	// keep workflow task open, we have to keep "heartbeating" current workflow task.
	// In that case, it is more efficient to create a server timer with backoff duration and retry when that backoff
	// timer fires. So here we will return false to indicate we don't need local retry anymore. However, we have to
	// store the current attempt and backoff to the same LocalActivityResultMarker so the replay can do the right thing.
	// The backoff timer will be created by workflow.ExecuteLocalActivity().
	lar.backoff = retryBackoff

	return false
}

func getRetryBackoff(lar *localActivityResult, now time.Time) time.Duration {
	return getRetryBackoffWithNowTime(lar.task.retryPolicy, lar.task.attempt, lar.err, now, lar.task.expireTime)
}

func getRetryBackoffWithNowTime(p *RetryPolicy, attempt int32, err error, now, expireTime time.Time) time.Duration {
	if !IsRetryable(err, p.NonRetryableErrorTypes) {
		return noRetryBackoff
	}

	if p.MaximumAttempts > 0 && attempt >= p.MaximumAttempts {
		return noRetryBackoff // max attempt reached
	}

	var backoffInterval time.Duration
	// Extract backoff interval from error if it is a retryable error.
	// Not using errors.As() since we don't want to explore the whole error chain.
	if applicationErr, ok := err.(*ApplicationError); ok {
		backoffInterval = applicationErr.nextRetryDelay
	}
	// Calculate next backoff interval if the error did not contain the next backoff interval.
	// attempt starts from 1
	if backoffInterval == 0 {
		backoffInterval = time.Duration(float64(p.InitialInterval) * math.Pow(p.BackoffCoefficient, float64(attempt-1)))
		if backoffInterval <= 0 {
			// math.Pow() could overflow
			if p.MaximumInterval > 0 {
				backoffInterval = p.MaximumInterval
			}
		}
		if p.MaximumInterval > 0 && backoffInterval > p.MaximumInterval {
			// cap next interval to MaxInterval
			backoffInterval = p.MaximumInterval
		}
	}
	if backoffInterval <= 0 {
		return noRetryBackoff
	}

	nextScheduleTime := now.Add(backoffInterval)
	if !expireTime.IsZero() && nextScheduleTime.After(expireTime) {
		return noRetryBackoff
	}

	return backoffInterval
}

func (w *workflowExecutionContextImpl) CompleteWorkflowTask(workflowTask *workflowTask, waitLocalActivities bool) interface{} {
	if w.currentWorkflowTask == nil {
		return nil
	}
	eventHandler := w.getEventHandler()

	// w.laTunnel could be nil for worker.ReplayHistory() because there is no worker started, in that case we don't
	// care about the pending local activities, and just return because the result is ignored anyway by the caller.
	if w.hasPendingLocalActivityWork() && w.laTunnel != nil {
		if len(eventHandler.unstartedLaTasks) > 0 {
			// start new local activity tasks
			unstartedLaTasks := make(map[string]struct{})
			for activityID := range eventHandler.unstartedLaTasks {
				task := eventHandler.pendingLaTasks[activityID]
				task.wc = w
				task.workflowTask = workflowTask

				task.scheduledTime = time.Now()

				if !w.laTunnel.sendTask(task) {
					unstartedLaTasks[activityID] = struct{}{}
					task.wc = nil
					task.workflowTask = nil
				}
			}
			eventHandler.unstartedLaTasks = unstartedLaTasks
		}
		// cannot complete workflow task as there are pending local activities
		if waitLocalActivities {
			return nil
		}
	}

	eventCommands := eventHandler.commandsHelper.getCommands(true)
	if len(eventCommands) > 0 {
		w.newCommands = append(w.newCommands, eventCommands...)
	}

	w.newMessages = append(w.newMessages, eventHandler.takeOutgoingMessages()...)
	eventHandler.protocols.ClearCompleted()

	completeRequest := w.wth.completeWorkflow(eventHandler, w.currentWorkflowTask, w, w.newCommands, w.newMessages, !waitLocalActivities)
	w.clearCurrentTask()

	return completeRequest
}

func (w *workflowExecutionContextImpl) hasPendingLocalActivityWork() bool {
	eventHandler := w.getEventHandler()
	return !w.isWorkflowCompleted &&
		w.currentWorkflowTask != nil &&
		w.currentWorkflowTask.Query == nil && // don't run local activity for query task
		eventHandler != nil &&
		len(eventHandler.pendingLaTasks) > 0
}

func (w *workflowExecutionContextImpl) clearCurrentTask() {
	w.newCommands = nil
	w.newMessages = nil
	w.currentWorkflowTask = nil
}

func (w *workflowExecutionContextImpl) skipReplayCheck() bool {
	return w.currentWorkflowTask.Query != nil || !isFullHistory(w.currentWorkflowTask.History)
}

func (w *workflowExecutionContextImpl) SetCurrentTask(task *workflowservice.PollWorkflowTaskQueueResponse) {
	w.currentWorkflowTask = task
	// do not update the previousStartedEventID for query task
	if task.Query == nil {
		w.previousStartedEventID = task.GetStartedEventId()
	}
}

func (w *workflowExecutionContextImpl) SetPreviousStartedEventID(eventID int64) {
	// We must reset the last event we handled to be after the last WFT we really completed
	// + any command events (since the SDK "processed" those when it emitted the commands). This
	// is also equal to what we just processed in the speculative task, minus two, since we
	// would've just handled the most recent WFT started event, and we need to drop that & the
	// schedule event just before it.
	w.lastHandledEventID = w.lastHandledEventID - 2
	w.previousStartedEventID = eventID
}

func (w *workflowExecutionContextImpl) ResetIfStale(task *workflowservice.PollWorkflowTaskQueueResponse, historyIterator HistoryIterator) error {
	if len(task.History.Events) > 0 && task.History.Events[0].GetEventId() != w.previousStartedEventID+1 {
		w.wth.logger.Debug("Cached state staled, new task has unexpected events",
			tagWorkflowID, task.WorkflowExecution.GetWorkflowId(),
			tagRunID, task.WorkflowExecution.GetRunId(),
			tagAttempt, task.Attempt,
			tagCachedPreviousStartedEventID, w.previousStartedEventID,
			tagTaskFirstEventID, task.History.Events[0].GetEventId(),
			tagTaskStartedEventID, task.GetStartedEventId(),
			tagPreviousStartedEventID, task.GetPreviousStartedEventId(),
		)
		w.clearState()
		return w.resetStateIfDestroyed(task, historyIterator)
	}
	return nil
}

func skipDeterministicCheckForCommand(d *commandpb.Command, _ *sdkFlags) bool {
	switch d.GetCommandType() {
	case enumspb.COMMAND_TYPE_RECORD_MARKER:
		markerName := d.GetRecordMarkerCommandAttributes().GetMarkerName()
		if markerName == versionMarkerName || markerName == mutableSideEffectMarkerName {
			return true
		}
	}
	return false
}

func skipDeterministicCheckForEvent(e *historypb.HistoryEvent, sdkFlags *sdkFlags) bool {
	switch e.GetEventType() {
	case enumspb.EVENT_TYPE_MARKER_RECORDED:
		markerName := e.GetMarkerRecordedEventAttributes().GetMarkerName()
		if markerName == versionMarkerName || markerName == mutableSideEffectMarkerName {
			return true
		}
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED:
		return true
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CONTINUED_AS_NEW:
		return true
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED:
		return true
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED:
		return true
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_TIMED_OUT:
		return true
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ADMITTED:
		return true
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_REJECTED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED:
		protocolMsgCommandInUse := sdkFlags.tryUse(SDKFlagProtocolMessageCommand, false)
		return !protocolMsgCommandInUse
	}
	return false
}

// special check for upsert change version event
func skipDeterministicCheckForUpsertChangeVersion(events []*historypb.HistoryEvent, idx int) bool {
	e := events[idx]
	if e.GetEventType() == enumspb.EVENT_TYPE_MARKER_RECORDED &&
		e.GetMarkerRecordedEventAttributes().GetMarkerName() == versionMarkerName &&
		idx < len(events)-1 &&
		events[idx+1].GetEventType() == enumspb.EVENT_TYPE_UPSERT_WORKFLOW_SEARCH_ATTRIBUTES {
		if _, ok := events[idx+1].GetUpsertWorkflowSearchAttributesEventAttributes().SearchAttributes.IndexedFields[TemporalChangeVersion]; ok {
			return true
		}
	}
	return false
}

func matchReplayWithHistory(
	replayCommands []*commandpb.Command,
	historyEvents []*historypb.HistoryEvent,
	msgs []outboxEntry,
	sdkFlags *sdkFlags,
) error {
	di := 0
	hi := 0
	hSize := len(historyEvents)
	dSize := len(replayCommands)
matchLoop:
	for hi < hSize || di < dSize {
		var e *historypb.HistoryEvent
		if hi < hSize {
			e = historyEvents[hi]
			if skipDeterministicCheckForUpsertChangeVersion(historyEvents, hi) {
				hi += 2
				continue matchLoop
			}
			if skipDeterministicCheckForEvent(e, sdkFlags) {
				hi++
				continue matchLoop
			}
		}

		var d *commandpb.Command
		if di < dSize {
			d = replayCommands[di]
			if skipDeterministicCheckForCommand(d, sdkFlags) {
				di++
				continue matchLoop
			}
		}

		if d == nil {
			return historyMismatchErrorf("[TMPRL1100] nondeterministic workflow: missing replay command for %s", util.HistoryEventToString(e))
		}

		if e == nil {
			return historyMismatchErrorf("[TMPRL1100] nondeterministic workflow: extra replay command for %s", util.CommandToString(d))
		}

		if !isCommandMatchEvent(d, e, msgs) {
			return historyMismatchErrorf("[TMPRL1100] nondeterministic workflow: history event is %s, replay command is %s",
				util.HistoryEventToString(e), util.CommandToString(d))
		}

		di++
		hi++
	}
	return nil
}

func lastPartOfName(name string) string {
	lastDotIdx := strings.LastIndex(name, ".")
	if lastDotIdx < 0 || lastDotIdx == len(name)-1 {
		return name
	}
	return name[lastDotIdx+1:]
}

func isCommandMatchEvent(d *commandpb.Command, e *historypb.HistoryEvent, obes []outboxEntry) bool {
	switch d.GetCommandType() {
	case enumspb.COMMAND_TYPE_PROTOCOL_MESSAGE:
		msgid := d.GetProtocolMessageCommandAttributes().GetMessageId()
		for _, entry := range obes {
			if entry.msg.Id == msgid {
				return entry.eventPredicate(e)
			}
		}
		return false

	case enumspb.COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK:
		if e.GetEventType() != enumspb.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED {
			return false
		}
		eventAttributes := e.GetActivityTaskScheduledEventAttributes()
		commandAttributes := d.GetScheduleActivityTaskCommandAttributes()

		if eventAttributes.GetActivityId() != commandAttributes.GetActivityId() ||
			lastPartOfName(eventAttributes.ActivityType.GetName()) != lastPartOfName(commandAttributes.ActivityType.GetName()) {
			return false
		}

		return true

	case enumspb.COMMAND_TYPE_REQUEST_CANCEL_ACTIVITY_TASK:
		if e.GetEventType() != enumspb.EVENT_TYPE_ACTIVITY_TASK_CANCEL_REQUESTED {
			return false
		}
		commandAttributes := d.GetRequestCancelActivityTaskCommandAttributes()
		eventAttributes := e.GetActivityTaskCancelRequestedEventAttributes()
		if eventAttributes.GetScheduledEventId() != commandAttributes.GetScheduledEventId() {
			return false
		}

		return true

	case enumspb.COMMAND_TYPE_START_TIMER:
		if e.GetEventType() != enumspb.EVENT_TYPE_TIMER_STARTED {
			return false
		}
		eventAttributes := e.GetTimerStartedEventAttributes()
		commandAttributes := d.GetStartTimerCommandAttributes()

		if eventAttributes.GetTimerId() != commandAttributes.GetTimerId() {
			return false
		}

		return true

	case enumspb.COMMAND_TYPE_CANCEL_TIMER:
		if e.GetEventType() != enumspb.EVENT_TYPE_TIMER_CANCELED {
			return false
		}
		commandAttributes := d.GetCancelTimerCommandAttributes()
		if e.GetEventType() == enumspb.EVENT_TYPE_TIMER_CANCELED {
			eventAttributes := e.GetTimerCanceledEventAttributes()
			if eventAttributes.GetTimerId() != commandAttributes.GetTimerId() {
				return false
			}
		}

		return true

	case enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION:
		if e.GetEventType() != enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED {
			return false
		}

		return true

	case enumspb.COMMAND_TYPE_FAIL_WORKFLOW_EXECUTION:
		if e.GetEventType() != enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED {
			return false
		}

		return true

	case enumspb.COMMAND_TYPE_RECORD_MARKER:
		if e.GetEventType() != enumspb.EVENT_TYPE_MARKER_RECORDED {
			return false
		}
		eventAttributes := e.GetMarkerRecordedEventAttributes()
		commandAttributes := d.GetRecordMarkerCommandAttributes()
		if eventAttributes.GetMarkerName() != commandAttributes.GetMarkerName() {
			return false
		}

		return true

	case enumspb.COMMAND_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION:
		if e.GetEventType() != enumspb.EVENT_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION_INITIATED {
			return false
		}
		eventAttributes := e.GetRequestCancelExternalWorkflowExecutionInitiatedEventAttributes()
		commandAttributes := d.GetRequestCancelExternalWorkflowExecutionCommandAttributes()
		if checkNamespacesInCommandAndEvent(eventAttributes.GetNamespace(), commandAttributes.GetNamespace()) ||
			eventAttributes.WorkflowExecution.GetWorkflowId() != commandAttributes.GetWorkflowId() {
			return false
		}

		return true

	case enumspb.COMMAND_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION:
		if e.GetEventType() != enumspb.EVENT_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_INITIATED {
			return false
		}
		eventAttributes := e.GetSignalExternalWorkflowExecutionInitiatedEventAttributes()
		commandAttributes := d.GetSignalExternalWorkflowExecutionCommandAttributes()
		if checkNamespacesInCommandAndEvent(eventAttributes.GetNamespace(), commandAttributes.GetNamespace()) ||
			eventAttributes.GetSignalName() != commandAttributes.GetSignalName() ||
			eventAttributes.WorkflowExecution.GetWorkflowId() != commandAttributes.Execution.GetWorkflowId() {
			return false
		}

		return true

	case enumspb.COMMAND_TYPE_CANCEL_WORKFLOW_EXECUTION:
		if e.GetEventType() != enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED {
			return false
		}
		return true

	case enumspb.COMMAND_TYPE_CONTINUE_AS_NEW_WORKFLOW_EXECUTION:
		if e.GetEventType() != enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CONTINUED_AS_NEW {
			return false
		}

		return true

	case enumspb.COMMAND_TYPE_START_CHILD_WORKFLOW_EXECUTION:
		if e.GetEventType() != enumspb.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED {
			return false
		}
		eventAttributes := e.GetStartChildWorkflowExecutionInitiatedEventAttributes()
		commandAttributes := d.GetStartChildWorkflowExecutionCommandAttributes()
		if lastPartOfName(eventAttributes.WorkflowType.GetName()) != lastPartOfName(commandAttributes.WorkflowType.GetName()) {
			return false
		}

		return true

	case enumspb.COMMAND_TYPE_UPSERT_WORKFLOW_SEARCH_ATTRIBUTES:
		if e.GetEventType() != enumspb.EVENT_TYPE_UPSERT_WORKFLOW_SEARCH_ATTRIBUTES {
			return false
		}
		return true

	case enumspb.COMMAND_TYPE_MODIFY_WORKFLOW_PROPERTIES:
		if e.GetEventType() != enumspb.EVENT_TYPE_WORKFLOW_PROPERTIES_MODIFIED {
			return false
		}
		return true

	case enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION:
		if e.GetEventType() != enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED {
			return false
		}
		eventAttributes := e.GetNexusOperationScheduledEventAttributes()
		commandAttributes := d.GetScheduleNexusOperationCommandAttributes()

		return eventAttributes.GetService() == commandAttributes.GetService() &&
			eventAttributes.GetOperation() == commandAttributes.GetOperation()

	case enumspb.COMMAND_TYPE_REQUEST_CANCEL_NEXUS_OPERATION:
		if e.GetEventType() != enumspb.EVENT_TYPE_NEXUS_OPERATION_CANCEL_REQUESTED {
			return false
		}

		eventAttributes := e.GetNexusOperationCancelRequestedEventAttributes()
		commandAttributes := d.GetRequestCancelNexusOperationCommandAttributes()

		return eventAttributes.GetScheduledEventId() == commandAttributes.GetScheduledEventId()
	}

	return false
}

func isSearchAttributesMatched(attrFromEvent, attrFromCommand *commonpb.SearchAttributes) bool {
	if attrFromEvent != nil && attrFromCommand != nil {
		return reflect.DeepEqual(attrFromEvent.IndexedFields, attrFromCommand.IndexedFields)
	}
	return attrFromEvent == nil && attrFromCommand == nil
}

func isMemoMatched(attrFromEvent, attrFromCommand *commonpb.Memo) bool {
	if attrFromEvent != nil && attrFromCommand != nil {
		return reflect.DeepEqual(attrFromEvent.Fields, attrFromCommand.Fields)
	}
	return attrFromEvent == nil && attrFromCommand == nil
}

// return true if the check fails:
//
//	namespace is not empty in command
//	and namespace is not replayNamespace
//	and namespaces unmatch in command and events
func checkNamespacesInCommandAndEvent(eventNamespace, commandNamespace string) bool {
	if commandNamespace == "" || IsReplayNamespace(commandNamespace) {
		return false
	}
	return eventNamespace != commandNamespace
}

func (wth *workflowTaskHandlerImpl) completeWorkflow(
	eventHandler *workflowExecutionEventHandlerImpl,
	task *workflowservice.PollWorkflowTaskQueueResponse,
	workflowContext *workflowExecutionContextImpl,
	commands []*commandpb.Command,
	messages []*protocolpb.Message,
	forceNewWorkflowTask bool,
) interface{} {
	// for query task
	if task.Query != nil {
		queryCompletedRequest := &workflowservice.RespondQueryTaskCompletedRequest{
			TaskToken: task.TaskToken,
			Namespace: wth.namespace,
		}
		var panicErr *PanicError
		if errors.As(workflowContext.err, &panicErr) {
			queryCompletedRequest.CompletedType = enumspb.QUERY_RESULT_TYPE_FAILED
			queryCompletedRequest.ErrorMessage = "Workflow panic: " + panicErr.Error()
			return queryCompletedRequest
		}

		result, err := eventHandler.ProcessQuery(task.Query.GetQueryType(), task.Query.QueryArgs, task.Query.Header)
		if err != nil {
			queryCompletedRequest.CompletedType = enumspb.QUERY_RESULT_TYPE_FAILED
			queryCompletedRequest.ErrorMessage = err.Error()
		} else {
			queryCompletedRequest.CompletedType = enumspb.QUERY_RESULT_TYPE_ANSWERED
			queryCompletedRequest.QueryResult = result
		}
		return queryCompletedRequest
	}

	metricsHandler := wth.metricsHandler.WithTags(metrics.WorkflowTags(
		eventHandler.workflowEnvironmentImpl.workflowInfo.WorkflowType.Name))

	// complete workflow task
	var closeCommand *commandpb.Command
	var canceledErr *CanceledError
	var contErr *ContinueAsNewError

	if errors.As(workflowContext.err, &canceledErr) {
		// Workflow canceled
		metricsHandler.Counter(metrics.WorkflowCanceledCounter).Inc(1)
		closeCommand = createNewCommand(enumspb.COMMAND_TYPE_CANCEL_WORKFLOW_EXECUTION)
		closeCommand.Attributes = &commandpb.Command_CancelWorkflowExecutionCommandAttributes{CancelWorkflowExecutionCommandAttributes: &commandpb.CancelWorkflowExecutionCommandAttributes{
			Details: convertErrDetailsToPayloads(canceledErr.details, wth.dataConverter),
		}}
	} else if errors.As(workflowContext.err, &contErr) {
		// Continue as new error.
		metricsHandler.Counter(metrics.WorkflowContinueAsNewCounter).Inc(1)
		closeCommand = createNewCommand(enumspb.COMMAND_TYPE_CONTINUE_AS_NEW_WORKFLOW_EXECUTION)

		// ContinueAsNewError.RetryPolicy is optional.
		// If not set, use the retry policy from the workflow context.
		retryPolicy := contErr.RetryPolicy
		if retryPolicy == nil {
			retryPolicy = workflowContext.workflowInfo.RetryPolicy
		}

		useCompat := determineInheritBuildIdFlagForCommand(
			contErr.VersioningIntent, workflowContext.workflowInfo.TaskQueueName, contErr.TaskQueueName)
		closeCommand.Attributes = &commandpb.Command_ContinueAsNewWorkflowExecutionCommandAttributes{ContinueAsNewWorkflowExecutionCommandAttributes: &commandpb.ContinueAsNewWorkflowExecutionCommandAttributes{
			WorkflowType:        &commonpb.WorkflowType{Name: contErr.WorkflowType.Name},
			Input:               contErr.Input,
			TaskQueue:           &taskqueuepb.TaskQueue{Name: contErr.TaskQueueName, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
			WorkflowRunTimeout:  durationpb.New(contErr.WorkflowRunTimeout),
			WorkflowTaskTimeout: durationpb.New(contErr.WorkflowTaskTimeout),
			Header:              contErr.Header,
			Memo:                workflowContext.workflowInfo.Memo,
			SearchAttributes:    workflowContext.workflowInfo.SearchAttributes,
			RetryPolicy:         convertToPBRetryPolicy(retryPolicy),
			InheritBuildId:      useCompat,
		}}
	} else if workflowContext.err != nil {
		// Workflow failures
		if !isBenignApplicationError(workflowContext.err) {
			metricsHandler.Counter(metrics.WorkflowFailedCounter).Inc(1)
		}
		closeCommand = createNewCommand(enumspb.COMMAND_TYPE_FAIL_WORKFLOW_EXECUTION)
		failure := wth.failureConverter.ErrorToFailure(workflowContext.err)
		closeCommand.Attributes = &commandpb.Command_FailWorkflowExecutionCommandAttributes{FailWorkflowExecutionCommandAttributes: &commandpb.FailWorkflowExecutionCommandAttributes{
			Failure: failure,
		}}
	} else if workflowContext.isWorkflowCompleted {
		// Workflow completion
		metricsHandler.Counter(metrics.WorkflowCompletedCounter).Inc(1)
		closeCommand = createNewCommand(enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION)
		closeCommand.Attributes = &commandpb.Command_CompleteWorkflowExecutionCommandAttributes{CompleteWorkflowExecutionCommandAttributes: &commandpb.CompleteWorkflowExecutionCommandAttributes{
			Result: workflowContext.result,
		}}
	}

	if closeCommand != nil {
		commands = append(commands, closeCommand)
		elapsed := time.Since(workflowContext.workflowInfo.WorkflowStartTime)
		metricsHandler.Timer(metrics.WorkflowEndToEndLatency).Record(elapsed)
		forceNewWorkflowTask = false
	}

	var queryResults map[string]*querypb.WorkflowQueryResult
	if len(task.Queries) != 0 {
		queryResults = make(map[string]*querypb.WorkflowQueryResult)
		for queryID, query := range task.Queries {
			result, err := eventHandler.ProcessQuery(query.GetQueryType(), query.QueryArgs, query.Header)
			if err != nil {
				queryResults[queryID] = &querypb.WorkflowQueryResult{
					ResultType:   enumspb.QUERY_RESULT_TYPE_FAILED,
					ErrorMessage: err.Error(),
				}
			} else {
				queryResults[queryID] = &querypb.WorkflowQueryResult{
					ResultType: enumspb.QUERY_RESULT_TYPE_ANSWERED,
					Answer:     result,
				}
			}
		}
	}

	nonfirstLAAttempts := eventHandler.GatherLAAttemptsThisWFT()

	sdkFlags := eventHandler.sdkFlags.gatherNewSDKFlags()
	langUsedFlags := make([]uint32, 0, len(sdkFlags))
	for _, flag := range sdkFlags {
		langUsedFlags = append(langUsedFlags, uint32(flag))
	}

	seriesName := ""
	if (wth.workerDeploymentVersion != WorkerDeploymentVersion{}) {
		seriesName = wth.workerDeploymentVersion.DeploymentName
	}
	builtRequest := &workflowservice.RespondWorkflowTaskCompletedRequest{
		TaskToken:                  task.TaskToken,
		Commands:                   commands,
		Messages:                   messages,
		Identity:                   wth.identity,
		ReturnNewWorkflowTask:      true,
		ForceCreateNewWorkflowTask: forceNewWorkflowTask,
		BinaryChecksum:             wth.workerBuildID,
		QueryResults:               queryResults,
		Namespace:                  wth.namespace,
		MeteringMetadata:           &commonpb.MeteringMetadata{NonfirstLocalActivityExecutionAttempts: nonfirstLAAttempts},
		SdkMetadata: &sdk.WorkflowTaskCompletedMetadata{
			LangUsedFlags: langUsedFlags,
			SdkName:       eventHandler.getNewSdkNameAndReset(),
			SdkVersion:    eventHandler.getNewSdkVersionAndReset(),
		},
		WorkerVersionStamp: &commonpb.WorkerVersionStamp{
			BuildId:       wth.workerBuildID,
			UseVersioning: wth.useBuildIDForVersioning,
		},
		Capabilities: &workflowservice.RespondWorkflowTaskCompletedRequest_Capabilities{
			DiscardSpeculativeWorkflowTaskWithEvents: true,
		},
		Deployment: &deploymentpb.Deployment{
			BuildId:    wth.workerBuildID,
			SeriesName: seriesName,
		},
		DeploymentOptions: workerDeploymentOptionsToProto(
			wth.useBuildIDForVersioning,
			wth.workerDeploymentVersion,
		),
	}
	if wth.capabilities != nil && wth.capabilities.BuildIdBasedVersioning {
		//lint:ignore SA1019 ignore deprecated versioning APIs
		builtRequest.BinaryChecksum = ""
	}
	if wth.useBuildIDForVersioning || (wth.workerDeploymentVersion != WorkerDeploymentVersion{}) {
		workflowType := workflowContext.workflowInfo.WorkflowType
		if behavior, ok := wth.registry.getWorkflowVersioningBehavior(workflowType); ok {
			builtRequest.VersioningBehavior = versioningBehaviorToProto(behavior)
		} else {
			builtRequest.VersioningBehavior = versioningBehaviorToProto(wth.defaultVersioningBehavior)
		}
	}
	return builtRequest
}

func (wth *workflowTaskHandlerImpl) executeAnyPressurePoints(event *historypb.HistoryEvent, isInReplay bool) error {
	if wth.ppMgr != nil && !reflect.ValueOf(wth.ppMgr).IsNil() && !isInReplay {
		switch event.GetEventType() {
		case enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED:
			return wth.ppMgr.Execute(pressurePointTypeWorkflowTaskStartTimeout)
		case enumspb.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
			return wth.ppMgr.Execute(pressurePointTypeActivityTaskScheduleTimeout)
		case enumspb.EVENT_TYPE_ACTIVITY_TASK_STARTED:
			return wth.ppMgr.Execute(pressurePointTypeActivityTaskStartTimeout)
		case enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED:
			return wth.ppMgr.Execute(pressurePointTypeWorkflowTaskCompleted)
		}
	}
	return nil
}

func newActivityTaskHandler(
	client *WorkflowClient,
	params workerExecutionParameters,
	registry *registry,
) ActivityTaskHandler {
	return newActivityTaskHandlerWithCustomProvider(client, params, registry, nil)
}

func newActivityTaskHandlerWithCustomProvider(
	client *WorkflowClient,
	params workerExecutionParameters,
	registry *registry,
	activityProvider activityProvider,
) ActivityTaskHandler {
	seriesName := ""
	if (params.WorkerDeploymentVersion != WorkerDeploymentVersion{}) {
		seriesName = params.WorkerDeploymentVersion.DeploymentName
	}
	return &activityTaskHandlerImpl{
		taskQueueName:                    params.TaskQueue,
		identity:                         params.Identity,
		client:                           client,
		logger:                           params.Logger,
		metricsHandler:                   params.MetricsHandler,
		backgroundContext:                params.BackgroundContext,
		registry:                         registry,
		activityProvider:                 activityProvider,
		dataConverter:                    params.DataConverter,
		failureConverter:                 params.FailureConverter,
		workerStopCh:                     params.WorkerStopChannel,
		contextPropagators:               params.ContextPropagators,
		namespace:                        params.Namespace,
		defaultHeartbeatThrottleInterval: params.DefaultHeartbeatThrottleInterval,
		maxHeartbeatThrottleInterval:     params.MaxHeartbeatThrottleInterval,
		versionStamp: &commonpb.WorkerVersionStamp{
			BuildId:       params.getBuildID(),
			UseVersioning: params.UseBuildIDForVersioning,
		},
		deployment: &deploymentpb.Deployment{
			BuildId:    params.getBuildID(),
			SeriesName: seriesName,
		},
		workerDeploymentOptions: workerDeploymentOptionsToProto(
			params.UseBuildIDForVersioning,
			params.WorkerDeploymentVersion,
		),
	}
}

type temporalInvoker struct {
	sync.Mutex
	identity       string
	service        workflowservice.WorkflowServiceClient
	metricsHandler metrics.Handler
	taskToken      []byte
	// cancelHandler is called when the activity is canceled by a heartbeat request.
	cancelHandler context.CancelCauseFunc
	// Amount of time to wait between each pending heartbeat send
	heartbeatThrottleInterval time.Duration
	hbBatchEndTimer           *time.Timer // Whether we started a batch of operations that need to be reported in the cycle. This gets started on a user call.
	lastDetailsToReport       **commonpb.Payloads
	closeCh                   chan struct{}
	workerStopChannel         <-chan struct{}
	namespace                 string
	excludeInternalFromRetry  *atomic.Bool // borrowed from client in order to tell if internal errors are retriable
}

func (i *temporalInvoker) Heartbeat(ctx context.Context, details *commonpb.Payloads, skipBatching bool) error {
	i.Lock()
	defer i.Unlock()

	if i.hbBatchEndTimer != nil && !skipBatching {
		// If we have started batching window, keep track of last reported progress.
		i.lastDetailsToReport = &details
		return nil
	}

	isActivityCanceled, err := i.internalHeartBeat(ctx, details)

	// If the activity is canceled, the activity can ignore the cancellation and do its work
	// and complete. Our cancellation is co-operative, so we will try to heartbeat.
	if (err == nil || isActivityCanceled) && !skipBatching {
		// We have successfully sent heartbeat, start next batching window.
		i.lastDetailsToReport = nil

		// Create timer to fire before the threshold to report.
		i.hbBatchEndTimer = time.NewTimer(i.heartbeatThrottleInterval)

		go func() {
			select {
			case <-i.hbBatchEndTimer.C:
				// We are close to deadline.
			case <-i.workerStopChannel:
				// Activity worker is close to stop. This does the same steps as batch timer ends.
			case <-i.closeCh:
				// We got closed.
				return
			}

			// We close the batch and report the progress.
			var detailsToReport **commonpb.Payloads

			i.Lock()
			detailsToReport = i.lastDetailsToReport
			i.hbBatchEndTimer.Stop()
			i.hbBatchEndTimer = nil
			i.Unlock()

			if detailsToReport != nil {
				// TODO: there is a potential race condition here as the lock is released here and
				// locked again in the Hearbeat() method. This possible that a heartbeat call from
				// user activity grabs the lock first and calls internalHeartBeat before this
				// batching goroutine, which means some activity progress will be lost.
				_ = i.Heartbeat(ctx, *detailsToReport, false)
			}
		}()
	}

	return err
}

func (i *temporalInvoker) internalHeartBeat(ctx context.Context, details *commonpb.Payloads) (bool, error) {
	isActivityCanceled := false
	// We don't want the recording of the heartbeat to keep retrying the RPC
	// longer than the throttle interval. However, sometimes the interval is so
	// small that the context is cancelled before it even starts the call.
	// Therefore, we'll make sure not to timeout the context faster than the
	// minimum RPC timeout.
	recordTimeout := i.heartbeatThrottleInterval
	if recordTimeout < minRPCTimeout {
		recordTimeout = minRPCTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, recordTimeout)
	defer cancel()

	err := recordActivityHeartbeat(ctx, i.service, i.metricsHandler, i.identity, i.taskToken, details)

	switch err.(type) {
	case *CanceledError:
		// We are asked to cancel. inform the activity about cancellation through context.
		i.cancelHandler(err)
		isActivityCanceled = true
	case *serviceerror.NotFound, *serviceerror.NamespaceNotActive, *serviceerror.NamespaceNotFound:
		// We will pass these through as cancellation for now but something we can change
		// later when we have setter on cancel handler.
		i.cancelHandler(err)
		isActivityCanceled = true
	case nil:
		// No error, do nothing.
	default:
		if errors.Is(err, ErrActivityPaused) {
			// We are asked to pause. inform the activity about cancellation through context.
			i.cancelHandler(err)
			isActivityCanceled = true
		}
		// Transient errors are getting retried for the duration of the heartbeat timeout.
		// The fact that error has been returned means that activity should now be timed out, hence we should
		// propagate cancellation to the handler.
		statusErr, _ := status.FromError(err)
		if retry.IsRetryable(statusErr, i.excludeInternalFromRetry) {
			i.cancelHandler(err)
			isActivityCanceled = true
		}
	}

	if err != nil {
		logger := GetActivityLogger(ctx)
		logger.Warn("RecordActivityHeartbeat with error", tagError, err)
	}

	// This error won't be returned to user check RecordActivityHeartbeat().
	return isActivityCanceled, err
}

func (i *temporalInvoker) Close(ctx context.Context, flushBufferedHeartbeat bool) {
	i.Lock()
	defer i.Unlock()
	close(i.closeCh)
	if i.hbBatchEndTimer != nil {
		i.hbBatchEndTimer.Stop()
		if flushBufferedHeartbeat && i.lastDetailsToReport != nil {
			_, _ = i.internalHeartBeat(ctx, *i.lastDetailsToReport)
			i.lastDetailsToReport = nil
		}
	}
}

func (i *temporalInvoker) GetClient(options ClientOptions) Client {
	return NewServiceClient(i.service, nil, options)
}

func newServiceInvoker(
	taskToken []byte,
	identity string,
	service workflowservice.WorkflowServiceClient,
	metricsHandler metrics.Handler,
	cancelHandler context.CancelCauseFunc,
	heartbeatThrottleInterval time.Duration,
	workerStopChannel <-chan struct{},
	namespace string,
	excludeInternalFromRetry *atomic.Bool,
) ServiceInvoker {
	return &temporalInvoker{
		taskToken:                 taskToken,
		identity:                  identity,
		service:                   service,
		metricsHandler:            metricsHandler,
		cancelHandler:             cancelHandler,
		heartbeatThrottleInterval: heartbeatThrottleInterval,
		closeCh:                   make(chan struct{}),
		workerStopChannel:         workerStopChannel,
		namespace:                 namespace,
		excludeInternalFromRetry:  excludeInternalFromRetry,
	}
}

// Execute executes an implementation of the activity.
func (ath *activityTaskHandlerImpl) Execute(taskQueue string, t *workflowservice.PollActivityTaskQueueResponse) (result interface{}, err error) {
	traceLog(func() {
		ath.logger.Debug("Processing new activity task",
			tagWorkflowID, t.WorkflowExecution.GetWorkflowId(),
			tagRunID, t.WorkflowExecution.GetRunId(),
			tagActivityType, t.ActivityType.GetName(),
			tagAttempt, t.Attempt,
		)
	})
	// The root context is only cancelled when the worker is finished shutting down.
	rootCtx := ath.backgroundContext
	if rootCtx == nil {
		rootCtx = context.Background()
	}
	canCtx, cancel := context.WithCancelCause(rootCtx)
	defer cancel(nil)

	heartbeatThrottleInterval := ath.getHeartbeatThrottleInterval(t.GetHeartbeatTimeout().AsDuration())
	invoker := newServiceInvoker(
		t.TaskToken, ath.identity, ath.client.workflowService, ath.metricsHandler, cancel, heartbeatThrottleInterval,
		ath.workerStopCh, ath.namespace, ath.client.excludeInternalFromRetry)

	workflowType := t.WorkflowType.GetName()
	activityType := t.ActivityType.GetName()
	metricsHandler := ath.metricsHandler.WithTags(metrics.ActivityTags(workflowType, activityType, ath.taskQueueName))
	ctx, err := WithActivityTask(canCtx, t, taskQueue, invoker, ath.logger, metricsHandler,
		ath.dataConverter, ath.workerStopCh, ath.contextPropagators, ath.registry.interceptors, ath.client)
	if err != nil {
		return nil, err
	}

	// We must capture the context here because it is changed later to one that is
	// cancelled when the activity is done
	defer func(ctx context.Context) {
		_, activityCompleted := result.(*workflowservice.RespondActivityTaskCompletedRequest)
		invoker.Close(ctx, !activityCompleted) // flush buffered heartbeat if activity was not successfully completed.
	}(ctx)

	activityImplementation := ath.getActivity(activityType)
	if activityImplementation == nil {
		// In case if activity is not registered we should report a failure to the server to allow activity retry
		// instead of making it stuck on the same attempt.
		metricsHandler.Counter(metrics.UnregisteredActivityInvocationCounter).Inc(1)
		return convertActivityResultToRespondRequest(ath.identity, t.TaskToken, nil,
			NewActivityNotRegisteredError(activityType, ath.getRegisteredActivityNames()),
			ath.dataConverter, ath.failureConverter, ath.namespace, false, ath.versionStamp, ath.deployment, ath.workerDeploymentOptions), nil
	}

	// panic handler
	defer func() {
		if p := recover(); p != nil {
			topLine := fmt.Sprintf("activity for %s [panic]:", ath.taskQueueName)
			st := getStackTraceRaw(topLine, 7, 0)
			ath.logger.Error("Activity panic.",
				tagWorkflowID, t.WorkflowExecution.GetWorkflowId(),
				tagRunID, t.WorkflowExecution.GetRunId(),
				tagActivityType, activityType,
				tagAttempt, t.Attempt,
				tagPanicError, fmt.Sprintf("%v", p),
				tagPanicStack, st)
			metricsHandler.Counter(metrics.ActivityTaskErrorCounter).Inc(1)
			panicErr := newPanicError(p, st)
			result = convertActivityResultToRespondRequest(ath.identity, t.TaskToken, nil, panicErr,
				ath.dataConverter, ath.failureConverter, ath.namespace, false, ath.versionStamp, ath.deployment, ath.workerDeploymentOptions)
		}
	}()

	// propagate context information into the activity context from the headers
	ctx, err = contextWithHeaderPropagated(ctx, t.Header, ath.contextPropagators)
	if err != nil {
		return nil, err
	}

	info := getActivityEnv(ctx)
	ctx, dlCancelFunc := context.WithDeadline(ctx, info.deadline)
	defer dlCancelFunc()

	output, err := activityImplementation.Execute(ctx, t.Input)
	// Check if context canceled at a higher level before we cancel it ourselves

	// Cancels that don't originate from the server will have separate cancel reasons, like
	// ErrWorkerShutdown or ErrActivityPaused
	isActivityCanceled := ctx.Err() == context.Canceled && errors.Is(context.Cause(ctx), &CanceledError{})

	dlCancelFunc()
	if <-ctx.Done(); ctx.Err() == context.DeadlineExceeded {
		ath.logger.Info("Activity complete after timeout.",
			tagWorkflowID, t.WorkflowExecution.GetWorkflowId(),
			tagRunID, t.WorkflowExecution.GetRunId(),
			tagActivityType, activityType,
			tagAttempt, t.Attempt,
			tagResult, output,
			tagError, err,
		)
		return nil, ctx.Err()
	}
	if err != nil && err != ErrActivityResultPending {
		logFunc := ath.logger.Error // Default to Error
		if isBenignApplicationError(err) {
			logFunc = ath.logger.Debug // Downgrade to Debug for benign application errors
		}
		logFunc("Activity error.",
			tagWorkflowID, t.WorkflowExecution.GetWorkflowId(),
			tagRunID, t.WorkflowExecution.GetRunId(),
			tagActivityType, activityType,
			tagAttempt, t.Attempt,
			tagError, err,
		)
	}
	return convertActivityResultToRespondRequest(ath.identity, t.TaskToken, output, err,
		ath.dataConverter, ath.failureConverter, ath.namespace, isActivityCanceled, ath.versionStamp, ath.deployment, ath.workerDeploymentOptions), nil
}

func (ath *activityTaskHandlerImpl) getActivity(name string) activity {
	if ath.activityProvider != nil {
		return ath.activityProvider(name)
	}

	if a, ok := ath.registry.GetActivity(name); ok {
		return a
	}

	return nil
}

func (ath *activityTaskHandlerImpl) getRegisteredActivityNames() (activityNames []string) {
	for _, a := range ath.registry.getRegisteredActivities() {
		activityNames = append(activityNames, a.ActivityType().Name)
	}
	return
}

func (ath *activityTaskHandlerImpl) getHeartbeatThrottleInterval(heartbeatTimeout time.Duration) time.Duration {
	// Set interval as 80% of timeout if present, or the configured default if
	// present, or the system default otherwise
	var heartbeatThrottleInterval time.Duration
	if heartbeatTimeout > 0 {
		heartbeatThrottleInterval = time.Duration(0.8 * float64(heartbeatTimeout))
	} else if ath.defaultHeartbeatThrottleInterval > 0 {
		heartbeatThrottleInterval = ath.defaultHeartbeatThrottleInterval
	} else {
		heartbeatThrottleInterval = defaultDefaultHeartbeatThrottleInterval
	}

	// Use the configured max if present, or the system default otherwise
	maxHeartbeatThrottleInterval := ath.maxHeartbeatThrottleInterval
	if maxHeartbeatThrottleInterval == 0 {
		maxHeartbeatThrottleInterval = defaultMaxHeartbeatThrottleInterval
	}

	// Limit interval to a max
	if heartbeatThrottleInterval > maxHeartbeatThrottleInterval {
		heartbeatThrottleInterval = maxHeartbeatThrottleInterval
	}
	return heartbeatThrottleInterval
}

func createNewCommand(commandType enumspb.CommandType) *commandpb.Command {
	return &commandpb.Command{
		CommandType: commandType,
	}
}

func createNewCommandWithMetadata(commandType enumspb.CommandType, metadata *sdk.UserMetadata) *commandpb.Command {
	return &commandpb.Command{
		CommandType:  commandType,
		UserMetadata: metadata,
	}
}

func recordActivityHeartbeat(ctx context.Context, service workflowservice.WorkflowServiceClient, metricsHandler metrics.Handler,
	identity string, taskToken []byte, details *commonpb.Payloads,
) error {
	namespace := getNamespaceFromActivityCtx(ctx)
	request := &workflowservice.RecordActivityTaskHeartbeatRequest{
		TaskToken: taskToken,
		Details:   details,
		Identity:  identity,
		Namespace: namespace,
	}

	var heartbeatResponse *workflowservice.RecordActivityTaskHeartbeatResponse
	grpcCtx, cancel := newGRPCContext(ctx,
		grpcMetricsHandler(metricsHandler),
		defaultGrpcRetryParameters(ctx))
	defer cancel()

	heartbeatResponse, err := service.RecordActivityTaskHeartbeat(grpcCtx, request)
	if err == nil && heartbeatResponse != nil {
		if heartbeatResponse.GetCancelRequested() {
			return NewCanceledError()
		} else if heartbeatResponse.GetActivityPaused() {
			return ErrActivityPaused
		}
	}
	return err
}

func recordActivityHeartbeatByID(ctx context.Context, service workflowservice.WorkflowServiceClient, metricsHandler metrics.Handler,
	identity, namespace, workflowID, runID, activityID string, details *commonpb.Payloads,
) error {
	request := &workflowservice.RecordActivityTaskHeartbeatByIdRequest{
		Namespace:  namespace,
		WorkflowId: workflowID,
		RunId:      runID,
		ActivityId: activityID,
		Details:    details,
		Identity:   identity,
	}

	var heartbeatResponse *workflowservice.RecordActivityTaskHeartbeatByIdResponse
	grpcCtx, cancel := newGRPCContext(ctx,
		grpcMetricsHandler(metricsHandler),
		defaultGrpcRetryParameters(ctx))
	defer cancel()

	heartbeatResponse, err := service.RecordActivityTaskHeartbeatById(grpcCtx, request)
	if err == nil && heartbeatResponse != nil && heartbeatResponse.GetCancelRequested() {
		return NewCanceledError()
	}
	return err
}

// This enables verbose logging in the client library.
// check worker.EnableVerboseLogging()
func traceLog(fn func()) {
	if enableVerboseLogging {
		fn()
	}
}
