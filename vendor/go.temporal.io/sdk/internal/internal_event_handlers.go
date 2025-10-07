package internal

// All code in this file is private to the package.

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	historypb "go.temporal.io/api/history/v1"
	protocolpb "go.temporal.io/api/protocol/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/metrics"
	ilog "go.temporal.io/sdk/internal/log"
	"go.temporal.io/sdk/internal/protocol"
	"go.temporal.io/sdk/log"
)

const (
	queryResultSizeLimit             = 2000000 // 2MB
	changeVersionSearchAttrSizeLimit = 2048
)

// Assert that structs do indeed implement the interfaces
var (
	_ WorkflowEnvironment           = (*workflowEnvironmentImpl)(nil)
	_ workflowExecutionEventHandler = (*workflowExecutionEventHandlerImpl)(nil)
)

type (
	// completionHandler Handler to indicate completion result
	completionHandler func(result *commonpb.Payloads, err error)

	// workflowExecutionEventHandlerImpl handler to handle workflowExecutionEventHandler
	workflowExecutionEventHandlerImpl struct {
		*workflowEnvironmentImpl
		workflowDefinition WorkflowDefinition
	}

	scheduledTimer struct {
		callback ResultHandler
		handled  bool
	}

	scheduledActivity struct {
		callback             ResultHandler
		waitForCancelRequest bool
		handled              bool
		activityType         ActivityType
	}

	scheduledNexusOperation struct {
		startedCallback   func(token string, err error)
		completedCallback func(result *commonpb.Payload, err error)
		cancellationType  NexusOperationCancellationType
		endpoint          string
		service           string
		operation         string
	}

	scheduledChildWorkflow struct {
		resultCallback      ResultHandler
		startedCallback     func(r WorkflowExecution, e error)
		waitForCancellation bool
		handled             bool
	}

	scheduledCancellation struct {
		callback ResultHandler
		handled  bool
	}

	scheduledSignal struct {
		callback ResultHandler
		handled  bool
	}

	sendCfg struct {
		addCmd bool
		pred   func(*historypb.HistoryEvent) bool
	}

	msgSendOpt func(so *sendCfg)

	outboxEntry struct {
		eventPredicate func(*historypb.HistoryEvent) bool
		msg            *protocolpb.Message
	}

	// workflowEnvironmentImpl an implementation of WorkflowEnvironment represents a environment for workflow execution.
	workflowEnvironmentImpl struct {
		workflowInfo *WorkflowInfo

		commandsHelper             *commandsHelper
		outbox                     []outboxEntry
		sideEffectResult           map[int64]*commonpb.Payloads
		changeVersions             map[string]Version
		pendingLaTasks             map[string]*localActivityTask
		completedLaAttemptsThisWFT uint32
		// mutableSideEffect is a map for each mutable side effect ID where each key is the
		// number of times the mutable side effect was called in a workflow
		// execution per ID.
		mutableSideEffect map[string]map[int]*commonpb.Payloads
		unstartedLaTasks  map[string]struct{}
		openSessions      map[string]*SessionInfo

		// Set of mutable side effect IDs that are recorded on the next task for use
		// during replay to determine whether a command should be created. The keys
		// are the user-provided IDs + "_" + the command counter.
		mutableSideEffectsRecorded map[string]bool
		// Records the number of times a mutable side effect was called per ID over the
		// life of the workflow. Used to help distinguish multiple calls to MutableSideEffect in the same
		// WorkflowTask.
		mutableSideEffectCallCounter map[string]int

		// LocalActivities have a separate, individual counter instead of relying on actual commandEventIDs.
		// This is because command IDs are only incremented on activity completion, which breaks
		// local activities that are spawned in parallel as they would all share the same command ID
		localActivityCounterID int64

		sideEffectCounterID int64

		currentReplayTime time.Time // Indicates current replay time of the command.
		currentLocalTime  time.Time // Local time when currentReplayTime was updated.

		completeHandler completionHandler                                                          // events completion handler
		cancelHandler   func()                                                                     // A cancel handler to be invoked on a cancel notification
		signalHandler   func(name string, input *commonpb.Payloads, header *commonpb.Header) error // A signal handler to be invoked on a signal event
		queryHandler    func(queryType string, queryArgs *commonpb.Payloads, header *commonpb.Header) (*commonpb.Payloads, error)
		updateHandler   func(name string, id string, args *commonpb.Payloads, header *commonpb.Header, callbacks UpdateCallbacks)

		logger                log.Logger
		isReplay              bool // flag to indicate if workflow is in replay mode
		enableLoggingInReplay bool // flag to indicate if workflow should enable logging in replay mode

		metricsHandler           metrics.Handler
		registry                 *registry
		dataConverter            converter.DataConverter
		failureConverter         converter.FailureConverter
		contextPropagators       []ContextPropagator
		deadlockDetectionTimeout time.Duration
		sdkFlags                 *sdkFlags
		sdkVersionUpdated        bool
		sdkVersion               string
		sdkNameUpdated           bool
		sdkName                  string
		// Any update requests received in a workflow task before we have registered
		// any handlers are not scheduled and are queued here until either their
		// handler is registered or the event loop runs out of work and they are rejected.
		bufferedUpdateRequests map[string][]func()

		protocols *protocol.Registry
	}

	localActivityTask struct {
		sync.Mutex
		workflowTask    *workflowTask
		activityID      string
		params          *ExecuteLocalActivityParams
		callback        LocalActivityResultHandler
		wc              *workflowExecutionContextImpl
		canceled        bool
		cancelFunc      func()
		attempt         int32  // attempt starting from 1
		attemptsThisWFT uint32 // Number of attempts started during this workflow task
		pastFirstWFT    bool   // Set true once this LA has lived for more than one workflow task
		retryPolicy     *RetryPolicy
		expireTime      time.Time
		scheduledTime   time.Time // Time the activity was scheduled initially.
		header          *commonpb.Header
	}

	localActivityMarkerData struct {
		ActivityID   string
		ActivityType string
		ReplayTime   time.Time
		Attempt      int32         // record attempt, starting from 1.
		Backoff      time.Duration // retry backoff duration.
	}
)

var (
	// ErrUnknownMarkerName is returned if there is unknown marker name in the history.
	ErrUnknownMarkerName = errors.New("unknown marker name")
	// ErrMissingMarkerDetails is returned when marker details are nil.
	ErrMissingMarkerDetails = errors.New("marker details are nil")
	// ErrMissingMarkerDataKey is returned when marker details doesn't have data key.
	ErrMissingMarkerDataKey = errors.New("marker key is missing in details")
	// ErrUnknownHistoryEvent is returned if there is an unknown event in history and the SDK needs to handle it
	ErrUnknownHistoryEvent = errors.New("unknown history event")
)

func newWorkflowExecutionEventHandler(
	workflowInfo *WorkflowInfo,
	completeHandler completionHandler,
	logger log.Logger,
	enableLoggingInReplay bool,
	metricsHandler metrics.Handler,
	registry *registry,
	dataConverter converter.DataConverter,
	failureConverter converter.FailureConverter,
	contextPropagators []ContextPropagator,
	deadlockDetectionTimeout time.Duration,
	capabilities *workflowservice.GetSystemInfoResponse_Capabilities,
) workflowExecutionEventHandler {
	context := &workflowEnvironmentImpl{
		workflowInfo:                 workflowInfo,
		commandsHelper:               newCommandsHelper(),
		sideEffectResult:             make(map[int64]*commonpb.Payloads),
		mutableSideEffect:            make(map[string]map[int]*commonpb.Payloads),
		changeVersions:               make(map[string]Version),
		pendingLaTasks:               make(map[string]*localActivityTask),
		unstartedLaTasks:             make(map[string]struct{}),
		openSessions:                 make(map[string]*SessionInfo),
		completeHandler:              completeHandler,
		enableLoggingInReplay:        enableLoggingInReplay,
		registry:                     registry,
		dataConverter:                dataConverter,
		failureConverter:             failureConverter,
		contextPropagators:           contextPropagators,
		deadlockDetectionTimeout:     deadlockDetectionTimeout,
		protocols:                    protocol.NewRegistry(),
		mutableSideEffectCallCounter: make(map[string]int),
		sdkFlags:                     newSDKFlags(capabilities),
		bufferedUpdateRequests:       make(map[string][]func()),
	}
	// Attempt to skip 1 log level to remove the ReplayLogger from the stack.
	context.logger = log.Skip(ilog.NewReplayLogger(
		log.With(logger,
			tagWorkflowType, workflowInfo.WorkflowType.Name,
			tagWorkflowID, workflowInfo.WorkflowExecution.ID,
			tagRunID, workflowInfo.WorkflowExecution.RunID,
			tagAttempt, workflowInfo.Attempt,
		),
		&context.isReplay,
		&context.enableLoggingInReplay), 1)

	if metricsHandler != nil {
		context.metricsHandler = metrics.NewReplayAwareHandler(&context.isReplay, metricsHandler).
			WithTags(metrics.WorkflowTags(workflowInfo.WorkflowType.Name))
	}

	return &workflowExecutionEventHandlerImpl{context, nil}
}

func (s *scheduledTimer) handle(result *commonpb.Payloads, err error) {
	if s.handled {
		panic(fmt.Sprintf("timer already handled %v", s))
	}
	s.handled = true
	s.callback(result, err)
}

func (s *scheduledActivity) handle(result *commonpb.Payloads, err error) {
	if s.handled {
		panic(fmt.Sprintf("activity already handled %v", s))
	}
	s.handled = true
	s.callback(result, err)
}

func (s *scheduledChildWorkflow) handle(result *commonpb.Payloads, err error) {
	if s.handled {
		panic(fmt.Sprintf("child workflow already handled %v", s))
	}
	s.handled = true
	s.resultCallback(result, err)
}

func (s *scheduledChildWorkflow) handleFailedToStart(result *commonpb.Payloads, err error) {
	if s.handled {
		panic(fmt.Sprintf("child workflow already handled %v", s))
	}
	s.handled = true
	s.resultCallback(result, err)
	s.startedCallback(WorkflowExecution{}, err)
}

func (t *localActivityTask) cancel() {
	t.Lock()
	t.canceled = true
	if t.cancelFunc != nil {
		t.cancelFunc()
	}
	t.Unlock()
}

func (s *scheduledCancellation) handle(result *commonpb.Payloads, err error) {
	if s.handled {
		panic(fmt.Sprintf("cancellation already handled %v", s))
	}
	s.handled = true
	s.callback(result, err)
}

func (s *scheduledSignal) handle(result *commonpb.Payloads, err error) {
	if s.handled {
		panic(fmt.Sprintf("signal already handled %v", s))
	}
	s.handled = true
	s.callback(result, err)
}

func (wc *workflowEnvironmentImpl) takeOutgoingMessages() []*protocolpb.Message {
	retval := make([]*protocolpb.Message, 0, len(wc.outbox))
	for _, entry := range wc.outbox {
		retval = append(retval, entry.msg)
	}
	wc.outbox = nil
	return retval
}

func (wc *workflowEnvironmentImpl) ScheduleUpdate(name string, id string, args *commonpb.Payloads, hdr *commonpb.Header, callbacks UpdateCallbacks) {
	wc.updateHandler(name, id, args, hdr, callbacks)
}

func withExpectedEventPredicate(pred func(*historypb.HistoryEvent) bool) msgSendOpt {
	return func(so *sendCfg) {
		so.addCmd = true
		so.pred = pred
	}
}

func (wc *workflowEnvironmentImpl) Send(msg *protocolpb.Message, opts ...msgSendOpt) {
	sendCfg := sendCfg{
		pred: func(*historypb.HistoryEvent) bool { return false },
	}
	for _, opt := range opts {
		opt(&sendCfg)
	}
	canSendCmd := wc.sdkFlags.tryUse(SDKFlagProtocolMessageCommand, !wc.isReplay)
	if canSendCmd && sendCfg.addCmd {
		wc.commandsHelper.addProtocolMessage(msg.Id)
	}
	wc.outbox = append(wc.outbox, outboxEntry{msg: msg, eventPredicate: sendCfg.pred})
}

func (wc *workflowEnvironmentImpl) getNewSdkNameAndReset() string {
	if wc.sdkNameUpdated {
		wc.sdkNameUpdated = false
		return wc.sdkName
	}
	return ""
}

func (wc *workflowEnvironmentImpl) getNewSdkVersionAndReset() string {
	if wc.sdkVersionUpdated {
		wc.sdkVersionUpdated = false
		return wc.sdkVersion
	}
	return ""
}

func (wc *workflowEnvironmentImpl) getNextLocalActivityID() string {
	wc.localActivityCounterID++
	return getStringID(wc.localActivityCounterID)
}

func (wc *workflowEnvironmentImpl) getNextSideEffectID() int64 {
	wc.sideEffectCounterID++
	return wc.sideEffectCounterID
}

func (wc *workflowEnvironmentImpl) WorkflowInfo() *WorkflowInfo {
	return wc.workflowInfo
}

func (wc *workflowEnvironmentImpl) TypedSearchAttributes() SearchAttributes {
	return convertToTypedSearchAttributes(wc.logger, wc.workflowInfo.SearchAttributes.GetIndexedFields())
}

func (wc *workflowEnvironmentImpl) Complete(result *commonpb.Payloads, err error) {
	wc.completeHandler(result, err)
}

func (wc *workflowEnvironmentImpl) RequestCancelChildWorkflow(namespace string, workflowID string) {
	// For cancellation of child workflow only, we do not use cancellation ID and run ID
	wc.commandsHelper.requestCancelExternalWorkflowExecution(namespace, workflowID, "", "", true)
}

func (wc *workflowEnvironmentImpl) RequestCancelExternalWorkflow(namespace, workflowID, runID string, callback ResultHandler) {
	// for cancellation of external workflow, we have to use cancellation ID and set isChildWorkflowOnly to false
	cancellationID := wc.GenerateSequenceID()
	command := wc.commandsHelper.requestCancelExternalWorkflowExecution(namespace, workflowID, runID, cancellationID, false)
	command.setData(&scheduledCancellation{callback: callback})
}

func (wc *workflowEnvironmentImpl) SignalExternalWorkflow(
	namespace string,
	workflowID string,
	runID string,
	signalName string,
	input *commonpb.Payloads,
	_ /* THIS IS FOR TEST FRAMEWORK. DO NOT USE HERE. */ interface{},
	header *commonpb.Header,
	childWorkflowOnly bool,
	callback ResultHandler,
) {
	signalID := wc.GenerateSequenceID()
	command := wc.commandsHelper.signalExternalWorkflowExecution(namespace, workflowID, runID, signalName, input,
		header, signalID, childWorkflowOnly)
	command.setData(&scheduledSignal{callback: callback})
}

func (wc *workflowEnvironmentImpl) UpsertSearchAttributes(attributes map[string]interface{}) error {
	// This has to be used in WorkflowEnvironment implementations instead of in Workflow for testsuite mock purpose.
	attr, err := validateAndSerializeSearchAttributes(attributes)
	if err != nil {
		return err
	}

	var upsertID string
	if changeVersion, ok := attributes[TemporalChangeVersion]; ok {
		// to ensure backward compatibility on searchable GetVersion, use latest changeVersion as upsertID
		upsertID = changeVersion.([]string)[0]
	} else {
		upsertID = wc.GenerateSequenceID()
	}

	wc.commandsHelper.upsertSearchAttributes(upsertID, attr)
	wc.updateWorkflowInfoWithSearchAttributes(attr) // this is for getInfo correctness
	return nil
}

func (wc *workflowEnvironmentImpl) UpsertTypedSearchAttributes(attributes SearchAttributes) error {
	rawSearchAttributes, err := serializeTypedSearchAttributes(attributes.untypedValue)
	if err != nil {
		return err
	}

	if _, ok := rawSearchAttributes.GetIndexedFields()[TemporalChangeVersion]; ok {
		return errors.New("TemporalChangeVersion is a reserved key that cannot be set, please use other key")
	}

	attr := make(map[string]interface{})
	for k, v := range rawSearchAttributes.GetIndexedFields() {
		attr[k] = v
	}
	return wc.UpsertSearchAttributes(attr)
}

func (wc *workflowEnvironmentImpl) updateWorkflowInfoWithSearchAttributes(attributes *commonpb.SearchAttributes) {
	wc.workflowInfo.SearchAttributes = mergeSearchAttributes(wc.workflowInfo.SearchAttributes, attributes)
}

func mergeSearchAttributes(current, upsert *commonpb.SearchAttributes) *commonpb.SearchAttributes {
	if current == nil || len(current.IndexedFields) == 0 {
		if upsert == nil || len(upsert.IndexedFields) == 0 {
			return nil
		}
		current = &commonpb.SearchAttributes{
			IndexedFields: make(map[string]*commonpb.Payload),
		}
	}

	fields := current.IndexedFields
	for k, v := range upsert.IndexedFields {
		fields[k] = v
	}
	return current
}

func validateAndSerializeSearchAttributes(attributes map[string]interface{}) (*commonpb.SearchAttributes, error) {
	if len(attributes) == 0 {
		return nil, errSearchAttributesNotSet
	}
	attr, err := serializeUntypedSearchAttributes(attributes)
	if err != nil {
		return nil, err
	}
	return attr, nil
}

func (wc *workflowEnvironmentImpl) UpsertMemo(memoMap map[string]interface{}) error {
	// This has to be used in WorkflowEnvironment implementations instead of in Workflow for testsuite mock purpose.
	memo, err := validateAndSerializeMemo(memoMap, wc.dataConverter)
	if err != nil {
		return err
	}

	changeID := wc.GenerateSequenceID()
	wc.commandsHelper.modifyProperties(changeID, memo)
	wc.updateWorkflowInfoWithMemo(memo) // this is for getInfo correctness
	return nil
}

func (wc *workflowEnvironmentImpl) updateWorkflowInfoWithMemo(memo *commonpb.Memo) {
	wc.workflowInfo.Memo = mergeMemo(wc.workflowInfo.Memo, memo)
}

func mergeMemo(current, upsert *commonpb.Memo) *commonpb.Memo {
	if current == nil || len(current.Fields) == 0 {
		if upsert == nil || len(upsert.Fields) == 0 {
			return nil
		}
		current = &commonpb.Memo{
			Fields: make(map[string]*commonpb.Payload),
		}
	}

	fields := current.Fields
	for k, v := range upsert.Fields {
		if v.Data == nil {
			delete(fields, k)
		} else {
			fields[k] = v
		}
	}
	return current
}

func validateAndSerializeMemo(memoMap map[string]interface{}, dc converter.DataConverter) (*commonpb.Memo, error) {
	if len(memoMap) == 0 {
		return nil, errMemoNotSet
	}
	return getWorkflowMemo(memoMap, dc)
}

func (wc *workflowEnvironmentImpl) RegisterCancelHandler(handler func()) {
	wrappedHandler := func() {
		handler()
	}
	wc.cancelHandler = wrappedHandler
}

func (wc *workflowEnvironmentImpl) ExecuteChildWorkflow(
	params ExecuteWorkflowParams, callback ResultHandler, startedHandler func(r WorkflowExecution, e error),
) {
	if params.WorkflowID == "" {
		params.WorkflowID = wc.workflowInfo.currentRunID + "_" + wc.GenerateSequenceID()
	}
	memo, err := getWorkflowMemo(params.Memo, wc.dataConverter)
	if err != nil {
		if wc.sdkFlags.tryUse(SDKFlagChildWorkflowErrorExecution, !wc.isReplay) {
			startedHandler(WorkflowExecution{}, &ChildWorkflowExecutionAlreadyStartedError{})
		}
		callback(nil, err)
		return
	}
	searchAttr, err := serializeSearchAttributes(params.SearchAttributes, params.TypedSearchAttributes)
	if err != nil {
		if wc.sdkFlags.tryUse(SDKFlagChildWorkflowErrorExecution, !wc.isReplay) {
			startedHandler(WorkflowExecution{}, &ChildWorkflowExecutionAlreadyStartedError{})
		}
		callback(nil, err)
		return
	}

	attributes := &commandpb.StartChildWorkflowExecutionCommandAttributes{}

	attributes.Namespace = params.Namespace
	attributes.TaskQueue = &taskqueuepb.TaskQueue{Name: params.TaskQueueName, Kind: enumspb.TASK_QUEUE_KIND_NORMAL}
	attributes.WorkflowId = params.WorkflowID
	attributes.WorkflowExecutionTimeout = durationpb.New(params.WorkflowExecutionTimeout)
	attributes.WorkflowRunTimeout = durationpb.New(params.WorkflowRunTimeout)
	attributes.WorkflowTaskTimeout = durationpb.New(params.WorkflowTaskTimeout)
	attributes.Input = params.Input
	attributes.WorkflowType = &commonpb.WorkflowType{Name: params.WorkflowType.Name}
	attributes.WorkflowIdReusePolicy = params.WorkflowIDReusePolicy
	attributes.ParentClosePolicy = params.ParentClosePolicy
	attributes.RetryPolicy = params.RetryPolicy
	attributes.Priority = params.Priority
	attributes.Header = params.Header
	attributes.Memo = memo
	attributes.SearchAttributes = searchAttr
	if len(params.CronSchedule) > 0 {
		attributes.CronSchedule = params.CronSchedule
	}
	//lint:ignore SA1019 ignore deprecated old versioning APIs
	attributes.InheritBuildId = determineInheritBuildIdFlagForCommand(
		params.VersioningIntent, wc.workflowInfo.TaskQueueName, params.TaskQueueName)

	startMetadata, err := buildUserMetadata(params.StaticSummary, params.StaticDetails, wc.dataConverter)
	if err != nil {
		callback(nil, err)
		return
	}

	command, err := wc.commandsHelper.startChildWorkflowExecution(attributes, startMetadata)
	if _, ok := err.(*childWorkflowExistsWithId); ok {
		if wc.sdkFlags.tryUse(SDKFlagChildWorkflowErrorExecution, !wc.isReplay) {
			startedHandler(WorkflowExecution{}, &ChildWorkflowExecutionAlreadyStartedError{})
		}
		callback(nil, &ChildWorkflowExecutionAlreadyStartedError{})
		return
	}
	command.setData(&scheduledChildWorkflow{
		resultCallback:      callback,
		startedCallback:     startedHandler,
		waitForCancellation: params.WaitForCancellation,
	})

	wc.logger.Debug("ExecuteChildWorkflow",
		tagChildWorkflowID, params.WorkflowID,
		tagWorkflowType, params.WorkflowType.Name)
}

func (wc *workflowEnvironmentImpl) ExecuteNexusOperation(params executeNexusOperationParams, callback func(*commonpb.Payload, error), startedHandler func(token string, e error)) int64 {
	seq := wc.GenerateSequence()
	scheduleTaskAttr := &commandpb.ScheduleNexusOperationCommandAttributes{
		Endpoint:               params.client.Endpoint(),
		Service:                params.client.Service(),
		Operation:              params.operation,
		Input:                  params.input,
		ScheduleToCloseTimeout: durationpb.New(params.options.ScheduleToCloseTimeout),
		NexusHeader:            params.nexusHeader,
	}

	startMetadata, err := buildUserMetadata(params.options.Summary, "", wc.dataConverter)
	if err != nil {
		callback(nil, err)
		return 0
	}

	command := wc.commandsHelper.scheduleNexusOperation(seq, scheduleTaskAttr, startMetadata)
	command.setData(&scheduledNexusOperation{
		startedCallback:   startedHandler,
		completedCallback: callback,
		cancellationType:  params.options.CancellationType,
		endpoint:          params.client.Endpoint(),
		service:           params.client.Service(),
		operation:         params.operation,
	})

	wc.logger.Debug("ScheduleNexusOperation",
		tagNexusEndpoint, params.client.Endpoint(),
		tagNexusService, params.client.Service(),
		tagNexusOperation, params.operation,
	)

	return command.seq
}

func (wc *workflowEnvironmentImpl) RequestCancelNexusOperation(seq int64) {
	command := wc.commandsHelper.requestCancelNexusOperation(seq)
	data := command.getData().(*scheduledNexusOperation)

	// Make sure to unblock the futures.
	if command.getState() == commandStateCreated || command.getState() == commandStateCommandSent {
		if data.startedCallback != nil {
			data.startedCallback("", ErrCanceled)
			data.startedCallback = nil
		}
		if data.completedCallback != nil {
			data.completedCallback(nil, ErrCanceled)
			data.completedCallback = nil
		}
	}
	wc.logger.Debug("RequestCancelNexusOperation",
		tagNexusEndpoint, data.endpoint,
		tagNexusService, data.service,
		tagNexusOperation, data.operation,
	)
}

func (wc *workflowEnvironmentImpl) RegisterSignalHandler(
	handler func(name string, input *commonpb.Payloads, header *commonpb.Header) error,
) {
	wc.signalHandler = handler
}

func (wc *workflowEnvironmentImpl) RegisterQueryHandler(
	handler func(string, *commonpb.Payloads, *commonpb.Header) (*commonpb.Payloads, error),
) {
	wc.queryHandler = handler
}

func (wc *workflowEnvironmentImpl) RegisterUpdateHandler(
	handler func(string, string, *commonpb.Payloads, *commonpb.Header, UpdateCallbacks),
) {
	wc.updateHandler = handler
}

func (wc *workflowEnvironmentImpl) GetLogger() log.Logger {
	return wc.logger
}

func (wc *workflowEnvironmentImpl) GetMetricsHandler() metrics.Handler {
	return wc.metricsHandler
}

func (wc *workflowEnvironmentImpl) GetDataConverter() converter.DataConverter {
	return wc.dataConverter
}

func (wc *workflowEnvironmentImpl) GetFailureConverter() converter.FailureConverter {
	return wc.failureConverter
}

func (wc *workflowEnvironmentImpl) GetContextPropagators() []ContextPropagator {
	return wc.contextPropagators
}

func (wc *workflowEnvironmentImpl) IsReplaying() bool {
	return wc.isReplay
}

func (wc *workflowEnvironmentImpl) GenerateSequenceID() string {
	return getStringID(wc.GenerateSequence())
}

func (wc *workflowEnvironmentImpl) GenerateSequence() int64 {
	return wc.commandsHelper.getNextID()
}

func (wc *workflowEnvironmentImpl) CreateNewCommand(commandType enumspb.CommandType) *commandpb.Command {
	return &commandpb.Command{
		CommandType: commandType,
	}
}

func (wc *workflowEnvironmentImpl) ExecuteActivity(parameters ExecuteActivityParams, callback ResultHandler) ActivityID {
	scheduleTaskAttr := &commandpb.ScheduleActivityTaskCommandAttributes{}
	scheduleID := wc.GenerateSequence()
	if parameters.ActivityID == "" {
		scheduleTaskAttr.ActivityId = getStringID(scheduleID)
	} else {
		scheduleTaskAttr.ActivityId = parameters.ActivityID
	}
	activityID := scheduleTaskAttr.GetActivityId()
	scheduleTaskAttr.ActivityType = &commonpb.ActivityType{Name: parameters.ActivityType.Name}
	scheduleTaskAttr.TaskQueue = &taskqueuepb.TaskQueue{Name: parameters.TaskQueueName, Kind: enumspb.TASK_QUEUE_KIND_NORMAL}
	scheduleTaskAttr.Input = parameters.Input
	scheduleTaskAttr.ScheduleToCloseTimeout = durationpb.New(parameters.ScheduleToCloseTimeout)
	scheduleTaskAttr.StartToCloseTimeout = durationpb.New(parameters.StartToCloseTimeout)
	scheduleTaskAttr.ScheduleToStartTimeout = durationpb.New(parameters.ScheduleToStartTimeout)
	scheduleTaskAttr.HeartbeatTimeout = durationpb.New(parameters.HeartbeatTimeout)
	scheduleTaskAttr.RetryPolicy = parameters.RetryPolicy
	scheduleTaskAttr.Header = parameters.Header
	// We set this as true if not disabled on the params knowing it will be set as
	// false just before request by the eager activity executor if eager activity
	// execution is otherwise disallowed
	scheduleTaskAttr.RequestEagerExecution = !parameters.DisableEagerExecution
	scheduleTaskAttr.UseWorkflowBuildId = determineInheritBuildIdFlagForCommand(
		parameters.VersioningIntent, wc.workflowInfo.TaskQueueName, parameters.TaskQueueName)
	scheduleTaskAttr.Priority = parameters.Priority

	startMetadata, err := buildUserMetadata(parameters.Summary, "", wc.dataConverter)
	if err != nil {
		callback(nil, err)
		return ActivityID{}
	}

	command := wc.commandsHelper.scheduleActivityTask(scheduleID, scheduleTaskAttr, startMetadata)
	command.setData(&scheduledActivity{
		callback:             callback,
		waitForCancelRequest: parameters.WaitForCancellation,
		activityType:         parameters.ActivityType,
	})

	wc.logger.Debug("ExecuteActivity",
		tagActivityID, activityID,
		tagActivityType, scheduleTaskAttr.ActivityType.GetName())

	return ActivityID{id: activityID}
}

func (wc *workflowEnvironmentImpl) RequestCancelActivity(activityID ActivityID) {
	command := wc.commandsHelper.requestCancelActivityTask(activityID.id)
	activity := command.getData().(*scheduledActivity)
	if activity.handled {
		return
	}

	if command.isDone() || !activity.waitForCancelRequest {
		activity.handle(nil, ErrCanceled)
	}

	wc.logger.Debug("RequestCancelActivity", tagActivityID, activityID)
}

func (wc *workflowEnvironmentImpl) ExecuteLocalActivity(params ExecuteLocalActivityParams, callback LocalActivityResultHandler) LocalActivityID {
	activityID := wc.getNextLocalActivityID()
	task := newLocalActivityTask(params, callback, activityID)
	wc.pendingLaTasks[activityID] = task
	wc.unstartedLaTasks[activityID] = struct{}{}
	return LocalActivityID{id: activityID}
}

func newLocalActivityTask(params ExecuteLocalActivityParams, callback LocalActivityResultHandler, activityID string) *localActivityTask {
	task := &localActivityTask{
		activityID:    activityID,
		params:        &params,
		callback:      callback,
		retryPolicy:   params.RetryPolicy,
		attempt:       params.Attempt,
		header:        params.Header,
		scheduledTime: time.Now(),
	}

	if params.ScheduleToCloseTimeout > 0 {
		task.expireTime = params.ScheduledTime.Add(params.ScheduleToCloseTimeout)
	}
	return task
}

func (wc *workflowEnvironmentImpl) RequestCancelLocalActivity(activityID LocalActivityID) {
	if task, ok := wc.pendingLaTasks[activityID.id]; ok {
		task.cancel()
	}
}

func (wc *workflowEnvironmentImpl) SetCurrentReplayTime(replayTime time.Time) {
	if replayTime.Before(wc.currentReplayTime) {
		return
	}
	wc.currentReplayTime = replayTime
	wc.currentLocalTime = time.Now()
}

func (wc *workflowEnvironmentImpl) Now() time.Time {
	return wc.currentReplayTime
}

func (wc *workflowEnvironmentImpl) NewTimer(d time.Duration, options TimerOptions, callback ResultHandler) *TimerID {
	if d < 0 {
		callback(nil, fmt.Errorf("negative duration provided %v", d))
		return nil
	}
	if d == 0 {
		callback(nil, nil)
		return nil
	}

	timerID := wc.GenerateSequenceID()
	startTimerAttr := &commandpb.StartTimerCommandAttributes{}
	startTimerAttr.TimerId = timerID
	startTimerAttr.StartToFireTimeout = durationpb.New(d)

	command := wc.commandsHelper.startTimer(startTimerAttr, options, wc.GetDataConverter())
	command.setData(&scheduledTimer{callback: callback})

	wc.logger.Debug("NewTimer",
		tagTimerID, startTimerAttr.GetTimerId(),
		"Duration", d)

	return &TimerID{id: timerID}
}

func (wc *workflowEnvironmentImpl) RequestCancelTimer(timerID TimerID) {
	command := wc.commandsHelper.cancelTimer(timerID)
	timer := command.getData().(*scheduledTimer)
	if timer != nil {
		if timer.handled {
			return
		}
		timer.handle(nil, ErrCanceled)
	}
	wc.logger.Debug("RequestCancelTimer", tagTimerID, timerID)
}

func validateVersion(changeID string, version, minSupported, maxSupported Version) {
	if version < minSupported {
		panicIllegalState(fmt.Sprintf("[TMPRL1100] Workflow code removed support of version %v. "+
			"for \"%v\" changeID. The oldest supported version is %v",
			version, changeID, minSupported))
	}
	if version > maxSupported {
		panicIllegalState(fmt.Sprintf("[TMPRL1100] Workflow code is too old to support version %v "+
			"for \"%v\" changeID. The maximum supported version is %v",
			version, changeID, maxSupported))
	}
}

func (wc *workflowEnvironmentImpl) GetVersion(changeID string, minSupported, maxSupported Version) Version {
	if version, ok := wc.changeVersions[changeID]; ok {
		validateVersion(changeID, version, minSupported, maxSupported)
		return version
	}

	var version Version
	if wc.isReplay {
		// GetVersion for changeID is called first time in replay mode, use DefaultVersion
		version = DefaultVersion
	} else {
		// GetVersion for changeID is called first time (non-replay mode), generate a marker command for it.
		// Also upsert search attributes to enable ability to search by changeVersion.
		version = maxSupported
		changeVersionSA := createSearchAttributesForChangeVersion(changeID, version, wc.changeVersions)
		attr, err := validateAndSerializeSearchAttributes(changeVersionSA)
		if err != nil {
			wc.logger.Warn(fmt.Sprintf("Failed to seralize %s search attribute with: %v", TemporalChangeVersion, err))
		} else {
			// Server has a limit for the max size of a single search attribute value. If we exceed the default limit
			// do not try to upsert as it will cause the workflow to fail.
			updateSearchAttribute := true
			if wc.sdkFlags.tryUse(SDKFlagLimitChangeVersionSASize, !wc.isReplay) && len(attr.IndexedFields[TemporalChangeVersion].GetData()) >= changeVersionSearchAttrSizeLimit {
				wc.logger.Warn(fmt.Sprintf("Serialized size of %s search attribute update would "+
					"exceed the maximum value size. Skipping this upsert. Be aware that your "+
					"visibility records will not include the following patch: %s", TemporalChangeVersion, getChangeVersion(changeID, version)),
				)
				updateSearchAttribute = false
			}
			wc.commandsHelper.recordVersionMarker(changeID, version, wc.GetDataConverter(), updateSearchAttribute)
			if updateSearchAttribute {
				_ = wc.UpsertSearchAttributes(changeVersionSA)
			}
		}
	}

	validateVersion(changeID, version, minSupported, maxSupported)
	wc.changeVersions[changeID] = version
	return version
}

func createSearchAttributesForChangeVersion(changeID string, version Version, existingChangeVersions map[string]Version) map[string]interface{} {
	return map[string]interface{}{
		TemporalChangeVersion: getChangeVersions(changeID, version, existingChangeVersions),
	}
}

func getChangeVersions(changeID string, version Version, existingChangeVersions map[string]Version) []string {
	res := []string{getChangeVersion(changeID, version)}
	for k, v := range existingChangeVersions {
		res = append(res, getChangeVersion(k, v))
	}
	return res
}

func getChangeVersion(changeID string, version Version) string {
	return fmt.Sprintf("%s-%v", changeID, version)
}

func (wc *workflowEnvironmentImpl) SideEffect(f func() (*commonpb.Payloads, error), callback ResultHandler) {
	sideEffectID := wc.getNextSideEffectID()
	var result *commonpb.Payloads
	if wc.isReplay {
		var ok bool
		result, ok = wc.sideEffectResult[sideEffectID]
		if !ok {
			keys := make([]int64, 0, len(wc.sideEffectResult))
			for k := range wc.sideEffectResult {
				keys = append(keys, k)
			}
			panicIllegalState(fmt.Sprintf("[TMPRL1100] No cached result found for side effectID=%v. KnownSideEffects=%v",
				sideEffectID, keys))
		}

		// Once the SideEffect has been consumed, we can free the referenced payload
		// to reduce memory pressure
		delete(wc.sideEffectResult, sideEffectID)
		wc.logger.Debug("SideEffect returning already calculated result.",
			tagSideEffectID, sideEffectID)
	} else {
		var err error
		result, err = f()
		if err != nil {
			callback(result, err)
			return
		}
	}

	wc.commandsHelper.recordSideEffectMarker(sideEffectID, result, wc.dataConverter)

	callback(result, nil)
	wc.logger.Debug("SideEffect Marker added", tagSideEffectID, sideEffectID)
}

func (wc *workflowEnvironmentImpl) TryUse(flag sdkFlag) bool {
	return wc.sdkFlags.tryUse(flag, !wc.isReplay)
}

func (wc *workflowEnvironmentImpl) GetFlag(flag sdkFlag) bool {
	return wc.sdkFlags.getFlag(flag)
}

func (wc *workflowEnvironmentImpl) QueueUpdate(name string, f func()) {
	wc.bufferedUpdateRequests[name] = append(wc.bufferedUpdateRequests[name], f)
}

func (wc *workflowEnvironmentImpl) HandleQueuedUpdates(name string) {
	if bufferedUpdateRequests, ok := wc.bufferedUpdateRequests[name]; ok {
		for _, request := range bufferedUpdateRequests {
			request()
		}
		delete(wc.bufferedUpdateRequests, name)
	}
}

func (wc *workflowEnvironmentImpl) DrainUnhandledUpdates() bool {
	anyExecuted := false
	// Check if any buffered update requests remain when we have no more coroutines to run and let them schedule so they are rejected.
	// Generally iterating a map in workflow code is bad because it is non deterministic
	// this case is fine since all these update handles will be rejected and not recorded in history.
	for name, requests := range wc.bufferedUpdateRequests {
		for _, request := range requests {
			request()
			anyExecuted = true
		}
		delete(wc.bufferedUpdateRequests, name)
	}
	return anyExecuted
}

// lookupMutableSideEffect gets the current value of the MutableSideEffect for id for the
// current call count of id.
func (wc *workflowEnvironmentImpl) lookupMutableSideEffect(id string) *commonpb.Payloads {
	// Fail if ID not found
	callCountPayloads := wc.mutableSideEffect[id]
	if len(callCountPayloads) == 0 {
		return nil
	}
	currentCallCount := wc.mutableSideEffectCallCounter[id]

	// Find the most recent call at/before the current call count
	var payloads *commonpb.Payloads
	payloadIndex := -1
	for callCount, maybePayloads := range callCountPayloads {
		if callCount <= currentCallCount && callCount > payloadIndex {
			payloads = maybePayloads
			payloadIndex = callCount
		}
	}

	// Garbage collect old entries
	for callCount := range callCountPayloads {
		if callCount <= currentCallCount && callCount != payloadIndex {
			delete(callCountPayloads, callCount)
		}
	}

	return payloads
}

func (wc *workflowEnvironmentImpl) MutableSideEffect(id string, f func() interface{}, equals func(a, b interface{}) bool) converter.EncodedValue {
	wc.mutableSideEffectCallCounter[id]++
	callCount := wc.mutableSideEffectCallCounter[id]

	if result := wc.lookupMutableSideEffect(id); result != nil {
		encodedResult := newEncodedValue(result, wc.GetDataConverter())
		if wc.isReplay {
			// During replay, we only generate a command if there was a known marker
			// recorded on the next task. We have to append the current command
			// counter to the user-provided ID to avoid duplicates.
			if wc.mutableSideEffectsRecorded[fmt.Sprintf("%v_%v", id, wc.commandsHelper.getNextID())] {
				return wc.recordMutableSideEffect(id, callCount, result)
			}
			return encodedResult
		}

		newValue := f()
		if wc.isEqualValue(newValue, result, equals) {
			return encodedResult
		}

		return wc.recordMutableSideEffect(id, callCount, wc.encodeValue(newValue))
	}

	if wc.isReplay {
		// This should not happen
		panicIllegalState(fmt.Sprintf("[TMPRL1100] Non deterministic workflow code change detected. MutableSideEffect API call doesn't have a correspondent event in the workflow history. MutableSideEffect ID: %s", id))
	}

	return wc.recordMutableSideEffect(id, callCount, wc.encodeValue(f()))
}

func (wc *workflowEnvironmentImpl) isEqualValue(newValue interface{}, encodedOldValue *commonpb.Payloads, equals func(a, b interface{}) bool) bool {
	if newValue == nil {
		// new value is nil
		newEncodedValue := wc.encodeValue(nil)
		return proto.Equal(newEncodedValue, encodedOldValue)
	}

	oldValue := decodeValue(newEncodedValue(encodedOldValue, wc.GetDataConverter()), newValue)
	return equals(newValue, oldValue)
}

func decodeValue(encodedValue converter.EncodedValue, value interface{}) interface{} {
	// We need to decode oldValue out of encodedValue, first we need to prepare valuePtr as the same type as value
	valuePtr := reflect.New(reflect.TypeOf(value)).Interface()
	if err := encodedValue.Get(valuePtr); err != nil {
		panic(err)
	}
	decodedValue := reflect.ValueOf(valuePtr).Elem().Interface()
	return decodedValue
}

func (wc *workflowEnvironmentImpl) encodeValue(value interface{}) *commonpb.Payloads {
	payload, err := wc.encodeArg(value)
	if err != nil {
		panic(err)
	}
	return payload
}

func (wc *workflowEnvironmentImpl) encodeArg(arg interface{}) (*commonpb.Payloads, error) {
	return wc.GetDataConverter().ToPayloads(arg)
}

func (wc *workflowEnvironmentImpl) recordMutableSideEffect(id string, callCountHint int, data *commonpb.Payloads) converter.EncodedValue {
	details, err := encodeArgs(wc.GetDataConverter(), []interface{}{id, data})
	if err != nil {
		panic(err)
	}
	wc.commandsHelper.recordMutableSideEffectMarker(id, callCountHint, details, wc.dataConverter)
	if wc.mutableSideEffect[id] == nil {
		wc.mutableSideEffect[id] = make(map[int]*commonpb.Payloads)
	}
	wc.mutableSideEffect[id][callCountHint] = data
	return newEncodedValue(data, wc.GetDataConverter())
}

func (wc *workflowEnvironmentImpl) AddSession(sessionInfo *SessionInfo) {
	wc.openSessions[sessionInfo.SessionID] = sessionInfo
}

func (wc *workflowEnvironmentImpl) RemoveSession(sessionID string) {
	delete(wc.openSessions, sessionID)
}

func (wc *workflowEnvironmentImpl) getOpenSessions() []*SessionInfo {
	openSessions := make([]*SessionInfo, 0, len(wc.openSessions))
	for _, info := range wc.openSessions {
		openSessions = append(openSessions, info)
	}
	return openSessions
}

func (wc *workflowEnvironmentImpl) GetRegistry() *registry {
	return wc.registry
}

// ResetLAWFTAttemptCounts resets the number of attempts in this WFT for all LAs to 0 - should be
// called at the beginning of every WFT
func (wc *workflowEnvironmentImpl) ResetLAWFTAttemptCounts() {
	wc.completedLaAttemptsThisWFT = 0
	for _, task := range wc.pendingLaTasks {
		task.Lock()
		task.attemptsThisWFT = 0
		task.pastFirstWFT = true
		task.Unlock()
	}
}

// GatherLAAttemptsThisWFT returns the total number of attempts in this WFT for all LAs who are
// past their first WFT
func (wc *workflowEnvironmentImpl) GatherLAAttemptsThisWFT() uint32 {
	var attempts uint32
	for _, task := range wc.pendingLaTasks {
		task.Lock()
		if task.pastFirstWFT {
			attempts += task.attemptsThisWFT
		}
		task.Unlock()
	}
	return attempts + wc.completedLaAttemptsThisWFT
}

func (weh *workflowExecutionEventHandlerImpl) ProcessEvent(
	event *historypb.HistoryEvent,
	isReplay bool,
	isLast bool,
) (err error) {
	if event == nil {
		return errors.New("nil event provided")
	}
	defer func() {
		if p := recover(); p != nil {
			incrementWorkflowTaskFailureCounter(weh.metricsHandler, "NonDeterminismError")
			topLine := fmt.Sprintf("process event for %s [panic]:", weh.workflowInfo.TaskQueueName)
			st := getStackTraceRaw(topLine, 7, 0)
			weh.Complete(nil, newWorkflowPanicError(p, st))
		}
	}()

	weh.isReplay = isReplay
	traceLog(func() {
		weh.logger.Debug("ProcessEvent",
			tagEventID, event.GetEventId(),
			tagEventType, event.GetEventType().String())
	})

	switch event.GetEventType() {
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED:
		err = weh.handleWorkflowExecutionStarted(event.GetWorkflowExecutionStartedEventAttributes())

	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED:
		// No Operation
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED:
		// No Operation
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_TIMED_OUT:
		// No Operation
	case enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED:
		// No Operation
	case enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED:
		// Set replay clock.
		weh.SetCurrentReplayTime(event.GetEventTime().AsTime())
		// Update workflow info fields
		weh.workflowInfo.currentHistoryLength = int(event.EventId)
		weh.workflowInfo.continueAsNewSuggested = event.GetWorkflowTaskStartedEventAttributes().GetSuggestContinueAsNew()
		weh.workflowInfo.currentHistorySize = int(event.GetWorkflowTaskStartedEventAttributes().GetHistorySizeBytes())
		// Reset the counter on command helper used for generating ID for commands
		weh.commandsHelper.setCurrentWorkflowTaskStartedEventID(event.GetEventId())
		weh.workflowDefinition.OnWorkflowTaskStarted(weh.deadlockDetectionTimeout)

	case enumspb.EVENT_TYPE_WORKFLOW_TASK_TIMED_OUT:
		// No Operation
	case enumspb.EVENT_TYPE_WORKFLOW_TASK_FAILED:
		// update the childWorkflowIDSeed if the workflow was reset at this point.
		attr := event.GetWorkflowTaskFailedEventAttributes()
		if attr.GetCause() == enumspb.WORKFLOW_TASK_FAILED_CAUSE_RESET_WORKFLOW {
			weh.workflowInfo.currentRunID = attr.GetNewRunId()
		}
	case enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED:
		// No Operation
	case enumspb.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
		weh.commandsHelper.handleActivityTaskScheduled(
			event.GetActivityTaskScheduledEventAttributes().GetActivityId(), event.GetEventId())

	case enumspb.EVENT_TYPE_ACTIVITY_TASK_STARTED:
		// No Operation

	case enumspb.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
		err = weh.handleActivityTaskCompleted(event)

	case enumspb.EVENT_TYPE_ACTIVITY_TASK_FAILED:
		err = weh.handleActivityTaskFailed(event)

	case enumspb.EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT:
		err = weh.handleActivityTaskTimedOut(event)

	case enumspb.EVENT_TYPE_ACTIVITY_TASK_CANCEL_REQUESTED:
		weh.commandsHelper.handleActivityTaskCancelRequested(
			event.GetActivityTaskCancelRequestedEventAttributes().GetScheduledEventId())

	case enumspb.EVENT_TYPE_ACTIVITY_TASK_CANCELED:
		err = weh.handleActivityTaskCanceled(event)

	case enumspb.EVENT_TYPE_TIMER_STARTED:
		weh.commandsHelper.handleTimerStarted(event.GetTimerStartedEventAttributes().GetTimerId())

	case enumspb.EVENT_TYPE_TIMER_FIRED:
		weh.handleTimerFired(event)

	case enumspb.EVENT_TYPE_TIMER_CANCELED:
		weh.commandsHelper.handleTimerCanceled(event.GetTimerCanceledEventAttributes().GetTimerId())

	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CANCEL_REQUESTED:
		weh.handleWorkflowExecutionCancelRequested()

	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED:
		// No Operation.

	case enumspb.EVENT_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION_INITIATED:
		_ = weh.handleRequestCancelExternalWorkflowExecutionInitiated(event)

	case enumspb.EVENT_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION_FAILED:
		_ = weh.handleRequestCancelExternalWorkflowExecutionFailed(event)

	case enumspb.EVENT_TYPE_EXTERNAL_WORKFLOW_EXECUTION_CANCEL_REQUESTED:
		_ = weh.handleExternalWorkflowExecutionCancelRequested(event)

	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_TERMINATED:
		// No Operation

	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CONTINUED_AS_NEW:
		// No Operation.

	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED:
		err = weh.handleWorkflowExecutionSignaled(event.GetWorkflowExecutionSignaledEventAttributes())

	case enumspb.EVENT_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_INITIATED:
		//lint:ignore SA1019 ignore deprecated control
		signalID := event.GetSignalExternalWorkflowExecutionInitiatedEventAttributes().Control
		weh.commandsHelper.handleSignalExternalWorkflowExecutionInitiated(event.GetEventId(), signalID)

	case enumspb.EVENT_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_FAILED:
		_ = weh.handleSignalExternalWorkflowExecutionFailed(event)

	case enumspb.EVENT_TYPE_EXTERNAL_WORKFLOW_EXECUTION_SIGNALED:
		_ = weh.handleSignalExternalWorkflowExecutionCompleted(event)

	case enumspb.EVENT_TYPE_MARKER_RECORDED:
		err = weh.handleMarkerRecorded(event.GetEventId(), event.GetMarkerRecordedEventAttributes())

	case enumspb.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED:
		weh.commandsHelper.handleStartChildWorkflowExecutionInitiated(
			event.GetStartChildWorkflowExecutionInitiatedEventAttributes().GetWorkflowId())

	case enumspb.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_FAILED:
		err = weh.handleStartChildWorkflowExecutionFailed(event)

	case enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_STARTED:
		err = weh.handleChildWorkflowExecutionStarted(event)

	case enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_COMPLETED:
		err = weh.handleChildWorkflowExecutionCompleted(event)

	case enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_FAILED:
		err = weh.handleChildWorkflowExecutionFailed(event)

	case enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_CANCELED:
		err = weh.handleChildWorkflowExecutionCanceled(event)

	case enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_TIMED_OUT:
		err = weh.handleChildWorkflowExecutionTimedOut(event)

	case enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_TERMINATED:
		err = weh.handleChildWorkflowExecutionTerminated(event)

	case enumspb.EVENT_TYPE_UPSERT_WORKFLOW_SEARCH_ATTRIBUTES:
		weh.handleUpsertWorkflowSearchAttributes(event)

	case enumspb.EVENT_TYPE_WORKFLOW_PROPERTIES_MODIFIED:
		weh.handleWorkflowPropertiesModified(event)

	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ADMITTED:
		// No Operation

	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED:
		// No Operation

	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_REJECTED:
		// No Operation

	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED:
		// No Operation

	case enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED:
		weh.commandsHelper.handleNexusOperationScheduled(event)
	case enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED:
		err = weh.handleNexusOperationStarted(event)
	// all forms of completions are handled by the same method.
	case enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED,
		enumspb.EVENT_TYPE_NEXUS_OPERATION_FAILED,
		enumspb.EVENT_TYPE_NEXUS_OPERATION_CANCELED,
		enumspb.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT:
		err = weh.handleNexusOperationCompleted(event)
	case enumspb.EVENT_TYPE_NEXUS_OPERATION_CANCEL_REQUESTED:
		err = weh.handleNexusOperationCancelRequested(event)
	case enumspb.EVENT_TYPE_NEXUS_OPERATION_CANCEL_REQUEST_COMPLETED,
		enumspb.EVENT_TYPE_NEXUS_OPERATION_CANCEL_REQUEST_FAILED:
		err = weh.handleNexusOperationCancelRequestDelivered(event)

	default:
		if event.WorkerMayIgnore {
			// Do not fail to be forward compatible with new events
			weh.logger.Debug("unknown event type",
				tagEventID, event.GetEventId(),
				tagEventType, event.GetEventType().String())
		} else {
			weh.logger.Error("unknown event type",
				tagEventID, event.GetEventId(),
				tagEventType, event.GetEventType().String())
			return ErrUnknownHistoryEvent
		}
	}

	if err != nil {
		return err
	}

	// When replaying histories to get stack trace or current state the last event might be not
	// workflow task started. So always call OnWorkflowTaskStarted on the last event.
	// Don't call for EventType_WorkflowTaskStarted as it was already called when handling it.
	if isLast && event.GetEventType() != enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED {
		weh.workflowDefinition.OnWorkflowTaskStarted(weh.deadlockDetectionTimeout)
	}

	return nil
}

func (weh *workflowExecutionEventHandlerImpl) ProcessMessage(
	msg *protocolpb.Message,
	isReplay bool,
	isLast bool,
) error {
	defer func() {
		if p := recover(); p != nil {
			incrementWorkflowTaskFailureCounter(weh.metricsHandler, "NonDeterminismError")
			topLine := fmt.Sprintf("process message for %s [panic]:", weh.workflowInfo.TaskQueueName)
			st := getStackTraceRaw(topLine, 7, 0)
			weh.Complete(nil, newWorkflowPanicError(p, st))
		}
	}()

	ctor, err := weh.protocolConstructorForMessage(msg)
	if err != nil {
		return nil
	}
	instance := weh.protocols.FindOrAdd(msg.ProtocolInstanceId, ctor)
	return instance.HandleMessage(msg)
}

func (weh *workflowExecutionEventHandlerImpl) ProcessQuery(
	queryType string,
	queryArgs *commonpb.Payloads,
	header *commonpb.Header,
) (*commonpb.Payloads, error) {
	switch queryType {
	case QueryTypeStackTrace:
		return weh.encodeArg(weh.StackTrace())
	case QueryTypeOpenSessions:
		return weh.encodeArg(weh.getOpenSessions())
	case QueryTypeWorkflowMetadata:
		// We are intentionally not handling this here but rather in the
		// normal handler so it has access to the options/context as
		// needed.
		fallthrough
	default:
		result, err := weh.queryHandler(queryType, queryArgs, header)
		if err != nil {
			return nil, err
		}

		if result.Size() > queryResultSizeLimit {
			weh.logger.Error("Query result size exceeds limit.",
				tagQueryType, queryType,
				tagWorkflowID, weh.workflowInfo.WorkflowExecution.ID,
				tagRunID, weh.workflowInfo.WorkflowExecution.RunID)
			return nil, fmt.Errorf("query result size (%v) exceeds limit (%v)", result.Size(), queryResultSizeLimit)
		}

		return result, nil
	}
}

func (weh *workflowExecutionEventHandlerImpl) StackTrace() string {
	return weh.workflowDefinition.StackTrace()
}

func (weh *workflowExecutionEventHandlerImpl) Close() {
	if weh.workflowDefinition != nil {
		weh.workflowDefinition.Close()
	}
}

func (weh *workflowExecutionEventHandlerImpl) handleWorkflowExecutionStarted(
	attributes *historypb.WorkflowExecutionStartedEventAttributes,
) (err error) {
	weh.workflowDefinition, err = weh.registry.getWorkflowDefinition(
		weh.workflowInfo.WorkflowType,
	)
	if err != nil {
		return err
	}

	// We set this flag at workflow start because changing it on a mid-workflow
	// WFT results in inconsistent values for SDKFlags during replay (i.e.
	// replay sees the _final_ value of applied flags, not intermediate values
	// as the value varies by WFT)
	weh.sdkFlags.tryUse(SDKFlagProtocolMessageCommand, !weh.isReplay)

	// Invoke the workflow.
	weh.workflowDefinition.Execute(weh, attributes.Header, attributes.Input)
	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleActivityTaskCompleted(event *historypb.HistoryEvent) error {
	activityID, scheduledEventID := weh.commandsHelper.getActivityAndScheduledEventIDs(event)
	command := weh.commandsHelper.handleActivityTaskClosed(activityID, scheduledEventID)
	activity := command.getData().(*scheduledActivity)
	if activity.handled {
		return nil
	}
	activity.handle(event.GetActivityTaskCompletedEventAttributes().Result, nil)

	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleActivityTaskFailed(event *historypb.HistoryEvent) error {
	activityID, scheduledEventID := weh.commandsHelper.getActivityAndScheduledEventIDs(event)
	command := weh.commandsHelper.handleActivityTaskClosed(activityID, scheduledEventID)
	activity := command.getData().(*scheduledActivity)
	if activity.handled {
		return nil
	}

	attributes := event.GetActivityTaskFailedEventAttributes()
	activityTaskErr := NewActivityError(
		attributes.GetScheduledEventId(),
		attributes.GetStartedEventId(),
		attributes.GetIdentity(),
		&commonpb.ActivityType{Name: activity.activityType.Name},
		activityID,
		attributes.GetRetryState(),
		weh.GetFailureConverter().FailureToError(attributes.GetFailure()),
	)

	activity.handle(nil, activityTaskErr)
	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleActivityTaskTimedOut(event *historypb.HistoryEvent) error {
	activityID, scheduledEventID := weh.commandsHelper.getActivityAndScheduledEventIDs(event)
	command := weh.commandsHelper.handleActivityTaskClosed(activityID, scheduledEventID)
	activity := command.getData().(*scheduledActivity)
	if activity.handled {
		return nil
	}

	attributes := event.GetActivityTaskTimedOutEventAttributes()
	timeoutError := weh.GetFailureConverter().FailureToError(attributes.GetFailure())

	activityTaskErr := NewActivityError(
		attributes.GetScheduledEventId(),
		attributes.GetStartedEventId(),
		"",
		&commonpb.ActivityType{Name: activity.activityType.Name},
		activityID,
		attributes.GetRetryState(),
		timeoutError,
	)

	activity.handle(nil, activityTaskErr)
	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleActivityTaskCanceled(event *historypb.HistoryEvent) error {
	activityID, scheduledEventID := weh.commandsHelper.getActivityAndScheduledEventIDs(event)
	command := weh.commandsHelper.handleActivityTaskCanceled(activityID, scheduledEventID)
	activity := command.getData().(*scheduledActivity)
	if activity.handled {
		return nil
	}

	if command.isDone() || !activity.waitForCancelRequest {
		// Clear this so we don't have a recursive call that while executing might call the cancel one.

		attributes := event.GetActivityTaskCanceledEventAttributes()
		details := newEncodedValues(attributes.GetDetails(), weh.GetDataConverter())

		activityTaskErr := NewActivityError(
			attributes.GetScheduledEventId(),
			attributes.GetStartedEventId(),
			attributes.GetIdentity(),
			&commonpb.ActivityType{Name: activity.activityType.Name},
			activityID,
			enumspb.RETRY_STATE_NON_RETRYABLE_FAILURE,
			NewCanceledError(details),
		)

		activity.handle(nil, activityTaskErr)
	}

	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleTimerFired(event *historypb.HistoryEvent) {
	timerID := event.GetTimerFiredEventAttributes().GetTimerId()
	command := weh.commandsHelper.handleTimerClosed(timerID)
	timer := command.getData().(*scheduledTimer)
	if timer.handled {
		return
	}

	timer.handle(nil, nil)
}

func (weh *workflowExecutionEventHandlerImpl) handleWorkflowExecutionCancelRequested() {
	weh.cancelHandler()
}

func (weh *workflowExecutionEventHandlerImpl) handleMarkerRecorded(
	eventID int64,
	attributes *historypb.MarkerRecordedEventAttributes,
) error {
	var err error
	if attributes.GetDetails() == nil {
		err = ErrMissingMarkerDetails
	} else {
		switch attributes.GetMarkerName() {
		case sideEffectMarkerName:
			if sideEffectIDPayload, ok := attributes.GetDetails()[sideEffectMarkerIDName]; !ok {
				err = fmt.Errorf("key %q: %w", sideEffectMarkerIDName, ErrMissingMarkerDataKey)
			} else {
				if sideEffectData, ok := attributes.GetDetails()[sideEffectMarkerDataName]; !ok {
					err = fmt.Errorf("key %q: %w", sideEffectMarkerDataName, ErrMissingMarkerDataKey)
				} else {
					var sideEffectID int64
					_ = weh.dataConverter.FromPayloads(sideEffectIDPayload, &sideEffectID)
					weh.sideEffectResult[sideEffectID] = sideEffectData
				}
			}
		case versionMarkerName:
			if changeIDPayload, ok := attributes.GetDetails()[versionMarkerChangeIDName]; !ok {
				err = fmt.Errorf("key %q: %w", versionMarkerChangeIDName, ErrMissingMarkerDataKey)
			} else {
				if versionPayload, ok := attributes.GetDetails()[versionMarkerDataName]; !ok {
					err = fmt.Errorf("key %q: %w", versionMarkerDataName, ErrMissingMarkerDataKey)
				} else {
					// versionSearchAttributeUpdatedName is optional and was only added later so do not expect all version
					// markers to have this.
					searchAttrUpdated := true
					if searchAttrUpdatedPayload, ok := attributes.GetDetails()[versionSearchAttributeUpdatedName]; ok {
						_ = weh.dataConverter.FromPayloads(searchAttrUpdatedPayload, &searchAttrUpdated)
					}
					var changeID string
					_ = weh.dataConverter.FromPayloads(changeIDPayload, &changeID)
					var version Version
					_ = weh.dataConverter.FromPayloads(versionPayload, &version)
					weh.changeVersions[changeID] = version
					weh.commandsHelper.handleVersionMarker(eventID, changeID, searchAttrUpdated)
				}
			}
		case localActivityMarkerName:
			err = weh.handleLocalActivityMarker(attributes.GetDetails(), attributes.GetFailure(), LocalActivityMarkerParams{})
		case mutableSideEffectMarkerName:
			var sideEffectIDWithCounterPayload, sideEffectDataPayload *commonpb.Payloads
			if sideEffectIDWithCounterPayload = attributes.GetDetails()[sideEffectMarkerIDName]; sideEffectIDWithCounterPayload == nil {
				err = fmt.Errorf("key %q: %w", sideEffectMarkerIDName, ErrMissingMarkerDataKey)
			}
			if err == nil {
				if sideEffectDataPayload = attributes.GetDetails()[sideEffectMarkerDataName]; sideEffectDataPayload == nil {
					err = fmt.Errorf("key %q: %w", sideEffectMarkerDataName, ErrMissingMarkerDataKey)
				}
			}
			var sideEffectIDWithCounter, sideEffectDataID string
			var sideEffectDataContents commonpb.Payloads
			if err == nil {
				err = weh.dataConverter.FromPayloads(sideEffectIDWithCounterPayload, &sideEffectIDWithCounter)
			}
			// Side effect data is actually a wrapper of ID + data, so we need to
			// extract the second value as the actual data
			if err == nil {
				err = weh.dataConverter.FromPayloads(sideEffectDataPayload, &sideEffectDataID, &sideEffectDataContents)
			}
			if err == nil {
				counterHintPayload, ok := attributes.GetDetails()[mutableSideEffectCallCounterName]
				var counterHint int
				if ok {
					err = weh.dataConverter.FromPayloads(counterHintPayload, &counterHint)
				} else {
					// An old version of the SDK did not write the counter hint so we have to assume.
					// If multiple mutable side effects on the same ID are in a WFT only the last value is used.
					counterHint = weh.mutableSideEffectCallCounter[sideEffectDataID]
				}
				if err == nil {
					if weh.mutableSideEffect[sideEffectDataID] == nil {
						weh.mutableSideEffect[sideEffectDataID] = make(map[int]*commonpb.Payloads)
					}
					weh.mutableSideEffect[sideEffectDataID][counterHint] = &sideEffectDataContents
					// We must mark that it is recorded so we can know whether a command
					// needs to be generated during replay
					if weh.mutableSideEffectsRecorded == nil {
						weh.mutableSideEffectsRecorded = map[string]bool{}
					}
					// This must be stored with the counter
					weh.mutableSideEffectsRecorded[sideEffectIDWithCounter] = true
				}
			}
		default:
			err = ErrUnknownMarkerName
		}
	}

	if err != nil {
		return fmt.Errorf("marker name %q for eventId %d: %w", attributes.GetMarkerName(), eventID, err)
	}

	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleLocalActivityMarker(details map[string]*commonpb.Payloads, failure *failurepb.Failure, params LocalActivityMarkerParams) error {
	var markerData *commonpb.Payloads
	var ok bool
	if markerData, ok = details[localActivityMarkerDataName]; !ok {
		return fmt.Errorf("key %q: %w", localActivityMarkerDataName, ErrMissingMarkerDataKey)
	}

	lamd := localActivityMarkerData{}
	if err := weh.dataConverter.FromPayloads(markerData, &lamd); err != nil {
		return err
	}

	if la, ok := weh.pendingLaTasks[lamd.ActivityID]; ok {
		if len(lamd.ActivityType) > 0 && lamd.ActivityType != la.params.ActivityType {
			// history marker mismatch to the current code.
			panicMsg := fmt.Sprintf("[TMPRL1100] code executed local activity %v, but history event found %v, markerData: %v", la.params.ActivityType, lamd.ActivityType, markerData)
			panicIllegalState(panicMsg)
		}
		startMetadata, err := buildUserMetadata(la.params.Summary, "", weh.dataConverter)
		if err != nil {
			return err
		}
		weh.commandsHelper.recordLocalActivityMarker(lamd.ActivityID, details, failure, startMetadata)
		if la.pastFirstWFT {
			weh.completedLaAttemptsThisWFT += la.attemptsThisWFT
		}
		delete(weh.pendingLaTasks, lamd.ActivityID)
		delete(weh.unstartedLaTasks, lamd.ActivityID)
		lar := &LocalActivityResultWrapper{}
		if failure != nil {
			lar.Attempt = lamd.Attempt
			lar.Backoff = lamd.Backoff
			lar.Err = weh.GetFailureConverter().FailureToError(failure)
		} else {
			// Result might not be there if local activity doesn't have return value.
			lar.Result = details[localActivityResultName]
		}
		la.callback(lar)

		// update time
		weh.SetCurrentReplayTime(lamd.ReplayTime)

		// resume workflow execution after apply local activity result
		weh.workflowDefinition.OnWorkflowTaskStarted(weh.deadlockDetectionTimeout)
	}

	return nil
}

func (weh *workflowExecutionEventHandlerImpl) ProcessLocalActivityResult(lar *localActivityResult) error {
	details := make(map[string]*commonpb.Payloads)

	// convert local activity result and error to marker data
	lamd := localActivityMarkerData{
		ActivityID:   lar.task.activityID,
		ActivityType: lar.task.params.ActivityType,
		ReplayTime:   weh.currentReplayTime.Add(time.Since(weh.currentLocalTime)),
		Attempt:      lar.task.attempt,
	}
	if lar.err != nil {
		lamd.Backoff = lar.backoff
	} else if lar.result != nil {
		details[localActivityResultName] = lar.result
	}

	// encode marker data
	markerData, err := weh.encodeArg(lamd)
	if err != nil {
		return err
	}
	details[localActivityMarkerDataName] = markerData

	// create marker event for local activity result
	markerEvent := &historypb.HistoryEvent{
		EventType: enumspb.EVENT_TYPE_MARKER_RECORDED,
		Attributes: &historypb.HistoryEvent_MarkerRecordedEventAttributes{MarkerRecordedEventAttributes: &historypb.MarkerRecordedEventAttributes{
			MarkerName: localActivityMarkerName,
			Failure:    weh.GetFailureConverter().ErrorToFailure(lar.err),
			Details:    details,
		}},
	}

	// apply the local activity result to workflow
	return weh.ProcessEvent(markerEvent, false, false)
}

func (weh *workflowExecutionEventHandlerImpl) handleWorkflowExecutionSignaled(
	attributes *historypb.WorkflowExecutionSignaledEventAttributes,
) error {
	return weh.signalHandler(attributes.GetSignalName(), attributes.Input, attributes.Header)
}

func (weh *workflowExecutionEventHandlerImpl) handleStartChildWorkflowExecutionFailed(event *historypb.HistoryEvent) error {
	attributes := event.GetStartChildWorkflowExecutionFailedEventAttributes()
	childWorkflowID := attributes.GetWorkflowId()
	command := weh.commandsHelper.handleStartChildWorkflowExecutionFailed(childWorkflowID)
	childWorkflow := command.getData().(*scheduledChildWorkflow)
	if childWorkflow.handled {
		return nil
	}

	var causeErr error
	switch attributes.GetCause() {
	case enumspb.START_CHILD_WORKFLOW_EXECUTION_FAILED_CAUSE_WORKFLOW_ALREADY_EXISTS:
		causeErr = &ChildWorkflowExecutionAlreadyStartedError{}
	case enumspb.START_CHILD_WORKFLOW_EXECUTION_FAILED_CAUSE_NAMESPACE_NOT_FOUND:
		causeErr = &NamespaceNotFoundError{}
	default:
		causeErr = fmt.Errorf("unable to start child workflow for unknown cause: %v", attributes.GetCause())
	}

	err := NewChildWorkflowExecutionError(
		attributes.GetNamespace(),
		attributes.GetWorkflowId(),
		"",
		attributes.GetWorkflowType().GetName(),
		attributes.GetInitiatedEventId(),
		0,
		enumspb.RETRY_STATE_NON_RETRYABLE_FAILURE,
		causeErr,
	)
	childWorkflow.handleFailedToStart(nil, err)
	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleChildWorkflowExecutionStarted(event *historypb.HistoryEvent) error {
	attributes := event.GetChildWorkflowExecutionStartedEventAttributes()
	childWorkflowID := attributes.WorkflowExecution.GetWorkflowId()
	childRunID := attributes.WorkflowExecution.GetRunId()
	command := weh.commandsHelper.handleChildWorkflowExecutionStarted(childWorkflowID)
	childWorkflow := command.getData().(*scheduledChildWorkflow)
	if childWorkflow.handled {
		return nil
	}

	childWorkflowExecution := WorkflowExecution{
		ID:    childWorkflowID,
		RunID: childRunID,
	}
	childWorkflow.startedCallback(childWorkflowExecution, nil)

	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleChildWorkflowExecutionCompleted(event *historypb.HistoryEvent) error {
	attributes := event.GetChildWorkflowExecutionCompletedEventAttributes()
	childWorkflowID := attributes.WorkflowExecution.GetWorkflowId()
	command := weh.commandsHelper.handleChildWorkflowExecutionClosed(childWorkflowID)
	childWorkflow := command.getData().(*scheduledChildWorkflow)
	if childWorkflow.handled {
		return nil
	}
	childWorkflow.handle(attributes.Result, nil)

	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleChildWorkflowExecutionFailed(event *historypb.HistoryEvent) error {
	attributes := event.GetChildWorkflowExecutionFailedEventAttributes()
	childWorkflowID := attributes.WorkflowExecution.GetWorkflowId()
	command := weh.commandsHelper.handleChildWorkflowExecutionClosed(childWorkflowID)
	childWorkflow := command.getData().(*scheduledChildWorkflow)
	if childWorkflow.handled {
		return nil
	}

	childWorkflowExecutionError := NewChildWorkflowExecutionError(
		attributes.GetNamespace(),
		attributes.GetWorkflowExecution().GetWorkflowId(),
		attributes.GetWorkflowExecution().GetRunId(),
		attributes.GetWorkflowType().GetName(),
		attributes.GetInitiatedEventId(),
		attributes.GetStartedEventId(),
		attributes.GetRetryState(),
		weh.GetFailureConverter().FailureToError(attributes.GetFailure()),
	)
	childWorkflow.handle(nil, childWorkflowExecutionError)
	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleChildWorkflowExecutionCanceled(event *historypb.HistoryEvent) error {
	attributes := event.GetChildWorkflowExecutionCanceledEventAttributes()
	childWorkflowID := attributes.WorkflowExecution.GetWorkflowId()
	command := weh.commandsHelper.handleChildWorkflowExecutionCanceled(childWorkflowID)
	childWorkflow := command.getData().(*scheduledChildWorkflow)
	if childWorkflow.handled {
		return nil
	}
	details := newEncodedValues(attributes.Details, weh.GetDataConverter())

	childWorkflowExecutionError := NewChildWorkflowExecutionError(
		attributes.GetNamespace(),
		attributes.GetWorkflowExecution().GetWorkflowId(),
		attributes.GetWorkflowExecution().GetRunId(),
		attributes.GetWorkflowType().GetName(),
		attributes.GetInitiatedEventId(),
		attributes.GetStartedEventId(),
		enumspb.RETRY_STATE_NON_RETRYABLE_FAILURE,
		NewCanceledError(details),
	)
	childWorkflow.handle(nil, childWorkflowExecutionError)
	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleChildWorkflowExecutionTimedOut(event *historypb.HistoryEvent) error {
	attributes := event.GetChildWorkflowExecutionTimedOutEventAttributes()
	childWorkflowID := attributes.WorkflowExecution.GetWorkflowId()
	command := weh.commandsHelper.handleChildWorkflowExecutionClosed(childWorkflowID)
	childWorkflow := command.getData().(*scheduledChildWorkflow)
	if childWorkflow.handled {
		return nil
	}

	childWorkflowExecutionError := NewChildWorkflowExecutionError(
		attributes.GetNamespace(),
		attributes.GetWorkflowExecution().GetWorkflowId(),
		attributes.GetWorkflowExecution().GetRunId(),
		attributes.GetWorkflowType().GetName(),
		attributes.GetInitiatedEventId(),
		attributes.GetStartedEventId(),
		attributes.GetRetryState(),
		NewTimeoutError("Child workflow timeout", enumspb.TIMEOUT_TYPE_START_TO_CLOSE, nil),
	)
	childWorkflow.handle(nil, childWorkflowExecutionError)
	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleChildWorkflowExecutionTerminated(event *historypb.HistoryEvent) error {
	attributes := event.GetChildWorkflowExecutionTerminatedEventAttributes()
	childWorkflowID := attributes.WorkflowExecution.GetWorkflowId()
	command := weh.commandsHelper.handleChildWorkflowExecutionClosed(childWorkflowID)
	childWorkflow := command.getData().(*scheduledChildWorkflow)
	if childWorkflow.handled {
		return nil
	}

	childWorkflowExecutionError := NewChildWorkflowExecutionError(
		attributes.GetNamespace(),
		attributes.GetWorkflowExecution().GetWorkflowId(),
		attributes.GetWorkflowExecution().GetRunId(),
		attributes.GetWorkflowType().GetName(),
		attributes.GetInitiatedEventId(),
		attributes.GetStartedEventId(),
		enumspb.RETRY_STATE_NON_RETRYABLE_FAILURE,
		newTerminatedError(),
	)
	childWorkflow.handle(nil, childWorkflowExecutionError)
	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleNexusOperationStarted(event *historypb.HistoryEvent) error {
	attributes := event.GetNexusOperationStartedEventAttributes()
	command := weh.commandsHelper.handleNexusOperationStarted(attributes.ScheduledEventId)
	state := command.getData().(*scheduledNexusOperation)
	if state.startedCallback != nil {
		token := attributes.OperationToken
		if token == "" {
			token = attributes.OperationId //lint:ignore SA1019 this field is sent by servers older than 1.27.0.
		}
		state.startedCallback(token, nil)
		state.startedCallback = nil
	}
	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleNexusOperationCompleted(event *historypb.HistoryEvent) error {
	var result *commonpb.Payload
	var failure *failurepb.Failure
	var scheduledEventId int64

	switch event.EventType {
	case enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED:
		attrs := event.GetNexusOperationCompletedEventAttributes()
		result = attrs.GetResult()
		scheduledEventId = attrs.GetScheduledEventId()
	case enumspb.EVENT_TYPE_NEXUS_OPERATION_FAILED:
		attrs := event.GetNexusOperationFailedEventAttributes()
		failure = attrs.GetFailure()
		scheduledEventId = attrs.GetScheduledEventId()
	case enumspb.EVENT_TYPE_NEXUS_OPERATION_CANCELED:
		attrs := event.GetNexusOperationCanceledEventAttributes()
		failure = attrs.GetFailure()
		scheduledEventId = attrs.GetScheduledEventId()
	case enumspb.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT:
		attrs := event.GetNexusOperationTimedOutEventAttributes()
		failure = attrs.GetFailure()
		scheduledEventId = attrs.GetScheduledEventId()
	default:
		// This is only called internally and should never happen.
		panic(fmt.Errorf("invalid event type, not a Nexus Operation resolution: %v", event.EventType))
	}
	command := weh.commandsHelper.handleNexusOperationCompleted(scheduledEventId)
	state := command.getData().(*scheduledNexusOperation)
	var err error
	if failure != nil {
		err = weh.failureConverter.FailureToError(failure)
	}
	// Also unblock the start future
	if state.startedCallback != nil {
		state.startedCallback("", err) // We didn't get a started event, the operation completed synchronously.
		state.startedCallback = nil
	}
	if state.completedCallback != nil {
		state.completedCallback(result, err)
		state.completedCallback = nil
	}
	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleNexusOperationCancelRequested(event *historypb.HistoryEvent) error {
	attrs := event.GetNexusOperationCancelRequestedEventAttributes()
	scheduledEventId := attrs.GetScheduledEventId()

	command := weh.commandsHelper.handleNexusOperationCancelRequested(scheduledEventId)
	state := command.getData().(*scheduledNexusOperation)
	err := ErrCanceled
	if state.cancellationType == NexusOperationCancellationTypeTryCancel {
		if state.startedCallback != nil {
			state.startedCallback("", err)
			state.startedCallback = nil
		}
		if state.completedCallback != nil {
			state.completedCallback(nil, err)
			state.completedCallback = nil
		}
	}
	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleNexusOperationCancelRequestDelivered(event *historypb.HistoryEvent) error {
	var scheduledEventID int64
	var failure *failurepb.Failure

	switch event.EventType {
	case enumspb.EVENT_TYPE_NEXUS_OPERATION_CANCEL_REQUEST_COMPLETED:
		attrs := event.GetNexusOperationCancelRequestCompletedEventAttributes()
		scheduledEventID = attrs.GetScheduledEventId()
	case enumspb.EVENT_TYPE_NEXUS_OPERATION_CANCEL_REQUEST_FAILED:
		attrs := event.GetNexusOperationCancelRequestFailedEventAttributes()
		scheduledEventID = attrs.GetScheduledEventId()
		failure = attrs.GetFailure()
	default:
		// This is only called internally and should never happen.
		panic(fmt.Errorf("invalid event type, not a Nexus Operation cancel request resolution: %v", event.EventType))
	}

	if scheduledEventID == 0 {
		// API version 1.47.0 was released without the ScheduledEventID field on these events, so if we got this event
		// without that field populated, then just ignore and fall back to default WaitCompleted behavior.
		return nil
	}

	command := weh.commandsHelper.handleNexusOperationCancelRequestDelivered(scheduledEventID)
	state := command.getData().(*scheduledNexusOperation)
	err := ErrCanceled
	if failure != nil {
		err = weh.failureConverter.FailureToError(failure)
	}

	if state.cancellationType == NexusOperationCancellationTypeWaitRequested {
		if state.startedCallback != nil {
			state.startedCallback("", err)
			state.startedCallback = nil
		}
		if state.completedCallback != nil {
			state.completedCallback(nil, err)
			state.completedCallback = nil
		}
	}

	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleUpsertWorkflowSearchAttributes(event *historypb.HistoryEvent) {
	weh.updateWorkflowInfoWithSearchAttributes(event.GetUpsertWorkflowSearchAttributesEventAttributes().SearchAttributes)
}

func (weh *workflowExecutionEventHandlerImpl) handleWorkflowPropertiesModified(
	event *historypb.HistoryEvent,
) {
	attributes := event.GetWorkflowPropertiesModifiedEventAttributes()
	weh.updateWorkflowInfoWithMemo(attributes.UpsertedMemo)
}

func (weh *workflowExecutionEventHandlerImpl) handleRequestCancelExternalWorkflowExecutionInitiated(event *historypb.HistoryEvent) error {
	// For cancellation of child workflow only, we do not use cancellation ID
	// for cancellation of external workflow, we have to use cancellation ID
	attribute := event.GetRequestCancelExternalWorkflowExecutionInitiatedEventAttributes()
	workflowID := attribute.WorkflowExecution.GetWorkflowId()
	//lint:ignore SA1019 ignore deprecated control
	cancellationID := attribute.Control
	weh.commandsHelper.handleRequestCancelExternalWorkflowExecutionInitiated(event.GetEventId(), workflowID, cancellationID)
	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleExternalWorkflowExecutionCancelRequested(event *historypb.HistoryEvent) error {
	// For cancellation of child workflow only, we do not use cancellation ID
	// for cancellation of external workflow, we have to use cancellation ID
	attributes := event.GetExternalWorkflowExecutionCancelRequestedEventAttributes()
	workflowID := attributes.WorkflowExecution.GetWorkflowId()
	isExternal, command := weh.commandsHelper.handleExternalWorkflowExecutionCancelRequested(attributes.GetInitiatedEventId(), workflowID)
	if isExternal {
		// for cancel external workflow, we need to set the future
		cancellation := command.getData().(*scheduledCancellation)
		if cancellation.handled {
			return nil
		}
		cancellation.handle(nil, nil)
	}

	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleRequestCancelExternalWorkflowExecutionFailed(event *historypb.HistoryEvent) error {
	// For cancellation of child workflow only, we do not use cancellation ID
	// for cancellation of external workflow, we have to use cancellation ID
	attributes := event.GetRequestCancelExternalWorkflowExecutionFailedEventAttributes()
	workflowID := attributes.WorkflowExecution.GetWorkflowId()
	isExternal, command := weh.commandsHelper.handleRequestCancelExternalWorkflowExecutionFailed(attributes.GetInitiatedEventId(), workflowID)
	if isExternal {
		// for cancel external workflow, we need to set the future
		cancellation := command.getData().(*scheduledCancellation)
		if cancellation.handled {
			return nil
		}

		var err error
		switch attributes.GetCause() {
		case enumspb.CANCEL_EXTERNAL_WORKFLOW_EXECUTION_FAILED_CAUSE_EXTERNAL_WORKFLOW_EXECUTION_NOT_FOUND:
			err = newUnknownExternalWorkflowExecutionError()
		case enumspb.CANCEL_EXTERNAL_WORKFLOW_EXECUTION_FAILED_CAUSE_NAMESPACE_NOT_FOUND:
			err = &NamespaceNotFoundError{}
		default:
			err = fmt.Errorf("unable to cancel external workflow for unknown cause: %v", attributes.GetCause())
		}
		cancellation.handle(nil, err)
	}

	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleSignalExternalWorkflowExecutionCompleted(event *historypb.HistoryEvent) error {
	attributes := event.GetExternalWorkflowExecutionSignaledEventAttributes()
	command := weh.commandsHelper.handleSignalExternalWorkflowExecutionCompleted(attributes.GetInitiatedEventId())
	signal := command.getData().(*scheduledSignal)
	if signal.handled {
		return nil
	}
	signal.handle(nil, nil)

	return nil
}

func (weh *workflowExecutionEventHandlerImpl) handleSignalExternalWorkflowExecutionFailed(event *historypb.HistoryEvent) error {
	attributes := event.GetSignalExternalWorkflowExecutionFailedEventAttributes()
	command := weh.commandsHelper.handleSignalExternalWorkflowExecutionFailed(attributes.GetInitiatedEventId())
	signal := command.getData().(*scheduledSignal)
	if signal.handled {
		return nil
	}

	var err error
	switch attributes.GetCause() {
	case enumspb.SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_FAILED_CAUSE_EXTERNAL_WORKFLOW_EXECUTION_NOT_FOUND:
		err = newUnknownExternalWorkflowExecutionError()
	case enumspb.SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_FAILED_CAUSE_NAMESPACE_NOT_FOUND:
		err = &NamespaceNotFoundError{}
	default:
		err = fmt.Errorf("unable to signal external workflow for unknown cause: %v", attributes.GetCause())
	}

	signal.handle(nil, err)

	return nil
}

func (weh *workflowExecutionEventHandlerImpl) protocolConstructorForMessage(
	msg *protocolpb.Message,
) (func() protocol.Instance, error) {
	protoName, err := protocol.NameFromMessage(msg)
	if err != nil {
		return nil, err
	}

	switch protoName {
	case updateProtocolV1:
		return func() protocol.Instance {
			return newUpdateProtocol(msg.ProtocolInstanceId, weh.updateHandler, weh)
		}, nil
	}
	return nil, fmt.Errorf("unsupported protocol: %v", protoName)
}
