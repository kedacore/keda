package internal

// All code in this file is private to the package.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/serviceerror"
	"golang.org/x/time/rate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.temporal.io/sdk/internal/common/retry"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/backoff"
	"go.temporal.io/sdk/internal/common/metrics"
	internallog "go.temporal.io/sdk/internal/log"
	"go.temporal.io/sdk/log"
)

const (
	retryPollOperationInitialInterval         = 200 * time.Millisecond
	retryPollOperationMaxInterval             = 10 * time.Second
	retryPollResourceExhaustedInitialInterval = time.Second
	retryPollResourceExhaustedMaxInterval     = 10 * time.Second
	// How long the same poll task error can remain suppressed
	lastPollTaskErrSuppressTime     = 1 * time.Minute
	pollerAutoscalingReportInterval = 100 * time.Millisecond
)

var (
	pollOperationRetryPolicy         = createPollRetryPolicy()
	pollResourceExhaustedRetryPolicy = createPollResourceExhaustedRetryPolicy()
	retryLongPollGracePeriod         = 2 * time.Minute
	errStop                          = errors.New("worker stopping")
	// ErrWorkerStopped is returned when the worker is stopped
	//
	// Exposed as: [go.temporal.io/sdk/worker.ErrWorkerShutdown]
	ErrWorkerShutdown = errors.New("worker is now shutdown")
)

type (
	// ResultHandler that returns result
	ResultHandler func(result *commonpb.Payloads, err error)
	// LocalActivityResultHandler that returns local activity result
	LocalActivityResultHandler func(lar *LocalActivityResultWrapper)

	// LocalActivityResultWrapper contains result of a local activity
	LocalActivityResultWrapper struct {
		Err     error
		Result  *commonpb.Payloads
		Attempt int32
		Backoff time.Duration
	}

	LocalActivityMarkerParams struct {
		Summary string
	}

	executeNexusOperationParams struct {
		client      NexusClient
		operation   string
		input       *commonpb.Payload
		options     NexusOperationOptions
		nexusHeader map[string]string
	}

	// WorkflowEnvironment Represents the environment for workflow.
	// Should only be used within the scope of workflow definition.
	WorkflowEnvironment interface {
		AsyncActivityClient
		LocalActivityClient
		WorkflowTimerClient
		SideEffect(f func() (*commonpb.Payloads, error), callback ResultHandler)
		GetVersion(changeID string, minSupported, maxSupported Version) Version
		WorkflowInfo() *WorkflowInfo
		TypedSearchAttributes() SearchAttributes
		Complete(result *commonpb.Payloads, err error)
		RegisterCancelHandler(handler func())
		RequestCancelChildWorkflow(namespace, workflowID string)
		RequestCancelExternalWorkflow(namespace, workflowID, runID string, callback ResultHandler)
		ExecuteChildWorkflow(params ExecuteWorkflowParams, callback ResultHandler, startedHandler func(r WorkflowExecution, e error))
		ExecuteNexusOperation(params executeNexusOperationParams, callback func(*commonpb.Payload, error), startedHandler func(token string, e error)) int64
		RequestCancelNexusOperation(seq int64)
		GetLogger() log.Logger
		GetMetricsHandler() metrics.Handler
		// Must be called before WorkflowDefinition.Execute returns
		RegisterSignalHandler(
			handler func(name string, input *commonpb.Payloads, header *commonpb.Header) error,
		)
		SignalExternalWorkflow(
			namespace string,
			workflowID string,
			runID string,
			signalName string,
			input *commonpb.Payloads,
			arg interface{},
			header *commonpb.Header,
			childWorkflowOnly bool,
			callback ResultHandler,
		)
		RegisterQueryHandler(
			handler func(queryType string, queryArgs *commonpb.Payloads, header *commonpb.Header) (*commonpb.Payloads, error),
		)
		RegisterUpdateHandler(
			handler func(string, string, *commonpb.Payloads, *commonpb.Header, UpdateCallbacks),
		)
		IsReplaying() bool
		MutableSideEffect(id string, f func() interface{}, equals func(a, b interface{}) bool) converter.EncodedValue
		GetDataConverter() converter.DataConverter
		GetFailureConverter() converter.FailureConverter
		AddSession(sessionInfo *SessionInfo)
		RemoveSession(sessionID string)
		GetContextPropagators() []ContextPropagator
		UpsertSearchAttributes(attributes map[string]interface{}) error
		UpsertTypedSearchAttributes(attributes SearchAttributes) error
		UpsertMemo(memoMap map[string]interface{}) error
		GetRegistry() *registry
		// QueueUpdate request of type name
		QueueUpdate(name string, f func())
		// HandleQueuedUpdates unblocks all queued updates of type name
		HandleQueuedUpdates(name string)
		// DrainUnhandledUpdates unblocks all updates, meant to be used to drain
		// all unhandled updates at the end of a workflow task
		// returns true if any update was unblocked
		DrainUnhandledUpdates() bool
		// TryUse returns true if this flag may currently be used.
		TryUse(flag sdkFlag) bool
		// GetFlag returns if the flag is currently used.
		GetFlag(flag sdkFlag) bool
	}

	// WorkflowDefinitionFactory factory for creating WorkflowDefinition instances.
	WorkflowDefinitionFactory interface {
		// NewWorkflowDefinition must return a new instance of WorkflowDefinition on each call.
		NewWorkflowDefinition() WorkflowDefinition
	}

	// WorkflowDefinition wraps the code that can execute a workflow.
	WorkflowDefinition interface {
		// Execute implementation must be asynchronous.
		Execute(env WorkflowEnvironment, header *commonpb.Header, input *commonpb.Payloads)
		// OnWorkflowTaskStarted is called for each non timed out startWorkflowTask event.
		// Executed after all history events since the previous commands are applied to WorkflowDefinition
		// Application level code must be executed from this function only.
		// Execute call as well as callbacks called from WorkflowEnvironment functions can only schedule callbacks
		// which can be executed from OnWorkflowTaskStarted().
		OnWorkflowTaskStarted(deadlockDetectionTimeout time.Duration)
		// StackTrace of all coroutines owned by the Dispatcher instance.
		StackTrace() string
		// Close destroys all coroutines without waiting for their completion
		Close()
	}

	scalableTaskPoller struct {
		taskPollerType string
		// pollerCount is the number of pollers tasks to start. There may be less than this
		// due to limited slots, rate limiting, or poller autoscaling.
		pollerCount                  int
		taskPoller                   taskPoller
		pollerAutoscalerReportHandle *pollScalerReportHandle
		pollerSemaphore              *pollerSemaphore
	}

	// baseWorkerOptions options to configure base worker.
	baseWorkerOptions struct {
		pollerRate              int
		slotSupplier            SlotSupplier
		maxTaskPerSecond        float64
		taskPollers             []scalableTaskPoller
		taskProcessor           taskProcessor
		workerType              string
		identity                string
		buildId                 string
		logger                  log.Logger
		stopTimeout             time.Duration
		fatalErrCb              func(error)
		backgroundContextCancel context.CancelCauseFunc
		metricsHandler          metrics.Handler
		sessionTokenBucket      *sessionTokenBucket
		slotReservationData     slotReservationData
		isInternalWorker        bool
	}

	// baseWorker that wraps worker activities.
	baseWorker struct {
		options              baseWorkerOptions
		isWorkerStarted      bool
		stopCh               chan struct{}  // Channel used to stop the go routines.
		stopWG               sync.WaitGroup // The WaitGroup for stopping existing routines.
		pollLimiter          *rate.Limiter
		taskLimiter          *rate.Limiter
		limiterContext       context.Context
		limiterContextCancel func()
		retrier              *backoff.ConcurrentRetrier // Service errors back off retrier
		logger               log.Logger
		metricsHandler       metrics.Handler

		slotSupplier       *trackingSlotSupplier
		taskQueueCh        chan eagerOrPolledTask
		eagerTaskQueueCh   chan eagerTask
		fatalErrCb         func(error)
		sessionTokenBucket *sessionTokenBucket
		pollerBalancer     *pollerBalancer

		lastPollTaskErrMessage string
		lastPollTaskErrStarted time.Time
		lastPollTaskErrLock    sync.Mutex
	}

	eagerOrPolledTask interface {
		getTask() taskForWorker
		getPermit() *SlotPermit
	}

	polledTask struct {
		task   taskForWorker
		permit *SlotPermit
	}

	eagerTask struct {
		// task to process.
		task   taskForWorker
		permit *SlotPermit
	}

	pollScalerReportHandleOptions struct {
		initialPollerCount int
		maxPollerCount     int
		minPollerCount     int
		logger             log.Logger
		scaleCallback      func(int)
	}

	pollScalerReportHandle struct {
		minPollerCount         int
		maxPollerCount         int
		logger                 log.Logger
		target                 atomic.Int64
		scaleCallback          func(int)
		everSawScalingDecision atomic.Bool
		ingestedThisPeriod     atomic.Int64
		ingestedLastPeriod     atomic.Int64
		scaleUpAllowed         atomic.Bool
	}

	barrier chan struct{}

	// pollerSemaphore is a semaphore that limits the number of concurrent pollers.
	// it is effectively a resizable semaphore.
	pollerSemaphore struct {
		maxPermits int
		permits    int
		bs         chan barrier
	}

	// pollerBalancer is used to balance the number of poll requests from different poller types
	pollerBalancer struct {
		pollerCount   map[string]int
		pollerBarrier map[string]barrier
		mu            sync.Mutex
	}
)

func (h ResultHandler) wrap(callback ResultHandler) ResultHandler {
	return func(result *commonpb.Payloads, err error) {
		callback(result, err)
		h(result, err)
	}
}

func (t *polledTask) getTask() taskForWorker {
	return t.task
}
func (t *polledTask) getPermit() *SlotPermit {
	return t.permit
}
func (t *eagerTask) getTask() taskForWorker {
	return t.task
}
func (t *eagerTask) getPermit() *SlotPermit {
	return t.permit
}

// SetRetryLongPollGracePeriod sets the amount of time a long poller retries on
// fatal errors before it actually fails. For test use only,
// not safe to call with a running worker.
func SetRetryLongPollGracePeriod(period time.Duration) {
	retryLongPollGracePeriod = period
}

func getRetryLongPollGracePeriod() time.Duration {
	return retryLongPollGracePeriod
}

func createPollRetryPolicy() backoff.RetryPolicy {
	policy := backoff.NewExponentialRetryPolicy(retryPollOperationInitialInterval)
	policy.SetMaximumInterval(retryPollOperationMaxInterval)

	// NOTE: We don't use expiration interval since we don't use retries from retrier class.
	// We use it to calculate next backoff. We have additional layer that is built on poller
	// in the worker layer for to add some middleware for any poll retry that includes
	// (a) rate limiting across pollers (b) back-off across pollers when server is busy
	policy.SetExpirationInterval(retry.UnlimitedInterval) // We don't ever expire
	return policy
}

func createPollResourceExhaustedRetryPolicy() backoff.RetryPolicy {
	policy := backoff.NewExponentialRetryPolicy(retryPollResourceExhaustedInitialInterval)
	policy.SetMaximumInterval(retryPollResourceExhaustedMaxInterval)
	policy.SetExpirationInterval(retry.UnlimitedInterval)
	return policy
}

func newBaseWorker(
	options baseWorkerOptions,
) *baseWorker {
	ctx, cancel := context.WithCancel(context.Background())
	logger := log.With(options.logger, tagWorkerType, options.workerType)
	metricsHandler := options.metricsHandler.WithTags(metrics.WorkerTags(options.workerType))
	tss := newTrackingSlotSupplier(options.slotSupplier, trackingSlotSupplierOptions{
		logger:         logger,
		metricsHandler: metricsHandler,
		workerBuildId:  options.buildId,
		workerIdentity: options.identity,
	})
	bw := &baseWorker{
		options:        options,
		stopCh:         make(chan struct{}),
		taskLimiter:    rate.NewLimiter(rate.Limit(options.maxTaskPerSecond), 1),
		retrier:        backoff.NewConcurrentRetrier(pollOperationRetryPolicy),
		logger:         logger,
		metricsHandler: metricsHandler,

		slotSupplier: tss,
		// No buffer, so pollers are only able to poll for new tasks after the previous one is
		// dispatched.
		taskQueueCh: make(chan eagerOrPolledTask),
		// Allow enough capacity so that eager dispatch will not block. There's an upper limit of
		// 2k pending activities so this channel never needs to be larger than that.
		eagerTaskQueueCh: make(chan eagerTask, 2000),
		fatalErrCb:       options.fatalErrCb,

		limiterContext:       ctx,
		limiterContextCancel: cancel,
		sessionTokenBucket:   options.sessionTokenBucket,
	}
	// Set secondary retrier as resource exhausted
	bw.retrier.SetSecondaryRetryPolicy(pollResourceExhaustedRetryPolicy)
	if options.pollerRate > 0 {
		bw.pollLimiter = rate.NewLimiter(rate.Limit(options.pollerRate), 1)
	}
	// If we have multiple task workers, we need to balance the pollers
	if len(options.taskPollers) > 1 {
		bw.pollerBalancer = &pollerBalancer{
			pollerCount:   make(map[string]int),
			pollerBarrier: make(map[string]barrier),
		}
	}

	return bw
}

// Start starts a fixed set of routines to do the work.
func (bw *baseWorker) Start() {
	if bw.isWorkerStarted {
		return
	}

	bw.metricsHandler.Counter(metrics.WorkerStartCounter).Inc(1)

	for _, taskWorker := range bw.options.taskPollers {
		if bw.pollerBalancer != nil {
			bw.pollerBalancer.registerPollerType(taskWorker.taskPollerType)
		}

		for i := 0; i < taskWorker.pollerCount; i++ {
			bw.stopWG.Add(1)
			go bw.runPoller(taskWorker)
		}

		if taskWorker.pollerAutoscalerReportHandle != nil {
			bw.stopWG.Add(1)
			go func() {
				defer bw.stopWG.Done()
				taskWorker.pollerAutoscalerReportHandle.run(bw.stopCh)
			}()
		}
	}

	bw.stopWG.Add(1)
	go bw.runTaskDispatcher()

	bw.stopWG.Add(1)
	go bw.runEagerTaskDispatcher()

	bw.isWorkerStarted = true
	traceLog(func() {
		bw.logger.Info("Started Worker",
			"MaxTaskPerSecond", bw.options.maxTaskPerSecond,
		)
	})
}

func (bw *baseWorker) isStop() bool {
	select {
	case <-bw.stopCh:
		return true
	default:
		return false
	}
}

func (bw *baseWorker) runPoller(taskWorker scalableTaskPoller) {
	defer bw.stopWG.Done()
	// Note: With poller autoscaling, this metric doesn't make a lot of sense since the number of pollers can go up and down.
	bw.metricsHandler.Counter(metrics.PollerStartCounter).Inc(1)

	ctx, cancelfn := context.WithCancel(context.Background())
	defer cancelfn()
	reserveChan := make(chan *SlotPermit)

	for {
		if func() bool {
			if taskWorker.pollerSemaphore != nil {
				if taskWorker.pollerSemaphore.acquire(bw.limiterContext) != nil {
					return true
				}
				defer taskWorker.pollerSemaphore.release()
			}
			// Call the balancer to make sure one poller type doesn't starve the others of slots.
			if bw.pollerBalancer != nil {
				if bw.pollerBalancer.balance(bw.limiterContext, taskWorker.taskPollerType) != nil {
					return true
				}
			}

			bw.stopWG.Add(1)
			go func() {
				defer bw.stopWG.Done()
				s, err := bw.slotSupplier.ReserveSlot(ctx, &bw.options.slotReservationData)
				if err != nil {
					if !errors.Is(err, context.Canceled) {
						bw.logger.Error("Error while trying to reserve slot", "error", err)
						select {
						case reserveChan <- nil:
						case <-ctx.Done():
							return
						}
					}
					return
				}
				select {
				case reserveChan <- s:
				case <-ctx.Done():
					bw.releaseSlot(s, SlotReleaseReasonUnused)
				}
			}()

			select {
			case <-bw.stopCh:
				return true
			case permit := <-reserveChan:
				if permit == nil { // There was an error reserving a slot
					// Avoid spamming reserve hard in the event it's constantly failing
					if ctx.Err() == nil {
						time.Sleep(time.Second)
					}
					return false
				}
				if bw.sessionTokenBucket != nil {
					bw.sessionTokenBucket.waitForAvailableToken()
				}
				if bw.pollerBalancer != nil {
					bw.pollerBalancer.incrementPoller(taskWorker.taskPollerType)
				}
				bw.pollTask(taskWorker, permit)
				if bw.pollerBalancer != nil {
					bw.pollerBalancer.decrementPoller(taskWorker.taskPollerType)
				}
			}
			return false
		}() {
			return
		}
	}
}

func (bw *baseWorker) tryReserveSlot() *SlotPermit {
	if bw.isStop() {
		return nil
	}
	return bw.slotSupplier.TryReserveSlot(&bw.options.slotReservationData)
}

func (bw *baseWorker) releaseSlot(permit *SlotPermit, reason SlotReleaseReason) {
	bw.slotSupplier.ReleaseSlot(permit, reason)
}

func (bw *baseWorker) pushEagerTask(task eagerTask) {
	// Should always be non-blocking. Slots are reserved before requesting eager tasks.
	bw.eagerTaskQueueCh <- task
}

func (bw *baseWorker) processTaskAsync(eagerOrPolled eagerOrPolledTask) {
	bw.stopWG.Add(1)
	go func() {
		defer bw.stopWG.Done()

		task := eagerOrPolled.getTask()
		permit := eagerOrPolled.getPermit()

		if !task.isEmpty() {
			bw.slotSupplier.MarkSlotUsed(permit)
		}

		defer func() {
			bw.releaseSlot(permit, SlotReleaseReasonTaskProcessed)

			if p := recover(); p != nil {
				topLine := "base worker [panic]:"
				st := getStackTraceRaw(topLine, 7, 0)
				bw.logger.Error("Unhandled panic.",
					"PanicError", fmt.Sprintf("%v", p),
					"PanicStack", st)
			}
		}()
		err := bw.options.taskProcessor.ProcessTask(task)
		if err != nil {
			if isClientSideError(err) {
				bw.logger.Info("Task processing failed with client side error", tagError, err)
			} else {
				bw.logger.Info("Task processing failed with error", tagError, err)
			}
		}
	}()
}

func (bw *baseWorker) runTaskDispatcher() {
	defer bw.stopWG.Done()

	for {
		// wait for new task or worker stop
		select {
		case <-bw.stopCh:
			// Currently we can drop any tasks received when closing.
			// https://github.com/temporalio/sdk-go/issues/1197
			return
		case task := <-bw.taskQueueCh:
			// for non-polled-task (local activity result as task or eager task), we don't need to rate limit
			_, isPolledTask := task.(*polledTask)
			if isPolledTask && bw.taskLimiter.Wait(bw.limiterContext) != nil {
				if bw.isStop() {
					bw.releaseSlot(task.getPermit(), SlotReleaseReasonUnused)
					return
				}
			}
			bw.processTaskAsync(task)
		}
	}
}

func (bw *baseWorker) runEagerTaskDispatcher() {
	defer bw.stopWG.Done()
	for {
		select {
		case <-bw.stopCh:
			// drain eager dispatch queue
			for len(bw.eagerTaskQueueCh) > 0 {
				eagerTask := <-bw.eagerTaskQueueCh
				bw.processTaskAsync(&eagerTask)
			}
			return
		case eagerTask := <-bw.eagerTaskQueueCh:
			bw.processTaskAsync(&eagerTask)
		}
	}
}

func (bw *baseWorker) pollTask(taskWorker scalableTaskPoller, slotPermit *SlotPermit) {
	var err error
	var task taskForWorker
	didSendTask := false
	defer func() {
		if !didSendTask {
			bw.releaseSlot(slotPermit, SlotReleaseReasonUnused)
		}
	}()

	bw.retrier.Throttle(bw.stopCh)
	if bw.pollLimiter == nil || bw.pollLimiter.Wait(bw.limiterContext) == nil {
		task, err = taskWorker.taskPoller.PollTask()
		bw.logPollTaskError(err)
		if err != nil {
			// We retry "non retriable" errors while long polling for a while, because some proxies return
			// unexpected values causing unnecessary downtime.
			if isNonRetriableError(err) && bw.retrier.GetElapsedTime() > getRetryLongPollGracePeriod() {
				bw.logger.Error("Worker received non-retriable error. Shutting down.", tagError, err)
				if bw.fatalErrCb != nil {
					bw.fatalErrCb(err)
				}
				return
			}
			if taskWorker.pollerAutoscalerReportHandle != nil {
				taskWorker.pollerAutoscalerReportHandle.handleError(err)
			}
			// We use the secondary retrier on resource exhausted
			_, resourceExhausted := err.(*serviceerror.ResourceExhausted)
			bw.retrier.Failed(resourceExhausted)
		} else {
			bw.retrier.Succeeded()
		}
	}

	if task != nil {
		if taskWorker.pollerAutoscalerReportHandle != nil {
			taskWorker.pollerAutoscalerReportHandle.handleTask(task)
		}

		select {
		case bw.taskQueueCh <- &polledTask{task: task, permit: slotPermit}:
			didSendTask = true
		case <-bw.stopCh:
		}
	}
}

func (bw *baseWorker) logPollTaskError(err error) {
	// We do not want to log any errors after we were explicitly stopped
	select {
	case <-bw.stopCh:
		return
	default:
	}

	bw.lastPollTaskErrLock.Lock()
	defer bw.lastPollTaskErrLock.Unlock()
	// No error means reset the message and time
	if err == nil {
		bw.lastPollTaskErrMessage = ""
		bw.lastPollTaskErrStarted = time.Now()
		return
	}

	// Ignore connection loss on server shutdown. This helps with quiescing spurious error messages
	// upon server shutdown (where server is using the SDK).
	if bw.options.isInternalWorker {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Unavailable && strings.Contains(st.Message(), "graceful_stop") {
			return
		}
	}

	// Log the error as warn if it doesn't match the last error seen or its over
	// the time since
	if err.Error() != bw.lastPollTaskErrMessage || time.Since(bw.lastPollTaskErrStarted) > lastPollTaskErrSuppressTime {
		bw.logger.Warn("Failed to poll for task.", tagError, err)
		bw.lastPollTaskErrMessage = err.Error()
		bw.lastPollTaskErrStarted = time.Now()
	}
}

func isNonRetriableError(err error) bool {
	if err == nil {
		return false
	}
	switch err.(type) {
	case *serviceerror.InvalidArgument,
		*serviceerror.NamespaceNotFound,
		*serviceerror.ClientVersionNotSupported:
		return true
	}
	return false
}

// Stop is a blocking call and cleans up all the resources associated with worker.
func (bw *baseWorker) Stop() {
	if !bw.isWorkerStarted {
		return
	}
	close(bw.stopCh)
	bw.limiterContextCancel()

	for _, taskWorker := range bw.options.taskPollers {
		err := taskWorker.taskPoller.Cleanup()
		if err != nil {
			bw.logger.Error("Couldn't cleanup task worker", tagError, err)
		}
	}

	if success := awaitWaitGroup(&bw.stopWG, bw.options.stopTimeout); !success {
		traceLog(func() {
			bw.logger.Info("Worker graceful stop timed out.", "Stop timeout", bw.options.stopTimeout)
		})
	}

	// Close context
	if bw.options.backgroundContextCancel != nil {
		bw.options.backgroundContextCancel(ErrWorkerShutdown)
	}

	bw.isWorkerStarted = false
}

func newPollScalerReportHandle(options pollScalerReportHandleOptions) *pollScalerReportHandle {
	logger := options.logger
	if logger == nil {
		logger = internallog.NewNopLogger()
	}
	psr := &pollScalerReportHandle{
		maxPollerCount: options.maxPollerCount,
		minPollerCount: options.minPollerCount,
		logger:         logger,
		scaleCallback:  options.scaleCallback,
	}
	psr.target.Store(int64(options.initialPollerCount))
	return psr
}

func (prh *pollScalerReportHandle) handleTask(task taskForWorker) {
	if !task.isEmpty() {
		prh.ingestedThisPeriod.Add(1)
	}

	if sd, ok := task.scaleDecision(); ok {
		prh.everSawScalingDecision.Store(true)
		ds := sd.pollRequestDeltaSuggestion
		if ds > 0 {
			if prh.scaleUpAllowed.Load() {
				prh.updateTarget(func(target int64) int64 {
					return target + int64(ds)
				})
			}
		} else if ds < 0 {
			prh.updateTarget(func(target int64) int64 {
				return target + int64(ds)
			})
		}
	} else if task.isEmpty() && prh.everSawScalingDecision.Load() {
		// We want to avoid scaling down on empty polls if the server has never made any
		// scaling decisions - otherwise we might never scale up again.
		prh.updateTarget(func(target int64) int64 {
			return target - 1
		})
	}
}

func (prh *pollScalerReportHandle) updateTarget(f func(int64) int64) {
	target := prh.target.Load()
	newTarget := f(target)
	if newTarget < int64(prh.minPollerCount) {
		newTarget = int64(prh.minPollerCount)
	} else if newTarget > int64(prh.maxPollerCount) {
		newTarget = int64(prh.maxPollerCount)
	}
	for !prh.target.CompareAndSwap(target, newTarget) {
		target = prh.target.Load()
		newTarget = f(target)
		if newTarget < int64(prh.minPollerCount) {
			newTarget = int64(prh.minPollerCount)
		} else if newTarget > int64(prh.maxPollerCount) {
			newTarget = int64(prh.maxPollerCount)
		}
	}
	permits := int(newTarget)
	if prh.scaleCallback != nil {
		traceLog(func() {
			prh.logger.Debug("Updating number of permits", "permits", permits)
		})
		prh.scaleCallback(permits)
	}
}

func (prh *pollScalerReportHandle) handleError(err error) {
	// If we have never seen a scaling decision, we don't want to scale down
	// on errors, because we might never scale up again.
	if prh.everSawScalingDecision.Load() {
		_, resourceExhausted := err.(*serviceerror.ResourceExhausted)
		if resourceExhausted {
			prh.updateTarget(func(target int64) int64 {
				return target / 2
			})
		} else {
			prh.updateTarget(func(target int64) int64 {
				return target - 1
			})
		}
	}
}

func (prh *pollScalerReportHandle) run(stopCh <-chan struct{}) {
	ticker := time.NewTicker(pollerAutoscalingReportInterval)
	// Here we periodically check if we should permit increasing the
	// poller count further. We do this by comparing the number of ingested items in the
	// current period with the number of ingested items in the previous period. If we
	// are successfully ingesting more items, then it makes sense to allow scaling up.
	// If we aren't, then we're probably limited by how fast we can process the tasks
	// and it's not worth increasing the poller count further.
	for {
		select {
		case <-ticker.C:
			prh.newPeriod()
		case <-stopCh:
			return
		}
	}
}

func (prh *pollScalerReportHandle) newPeriod() {
	ingestedThisPeriod := prh.ingestedThisPeriod.Swap(0)
	ingestedLastPeriod := prh.ingestedLastPeriod.Swap(ingestedThisPeriod)
	prh.scaleUpAllowed.Store(float64(ingestedThisPeriod) >= float64(ingestedLastPeriod)*1.1)
}

func newPollerSemaphore(maxPermits int) *pollerSemaphore {
	ps := &pollerSemaphore{
		maxPermits: maxPermits,
		permits:    0,
		bs:         make(chan barrier, 1),
	}
	ps.bs <- make(barrier)
	return ps
}

func (ps *pollerSemaphore) acquire(ctx context.Context) error {
	for {
		// Acquire barrier.
		b := <-ps.bs
		if ps.permits < ps.maxPermits {
			ps.permits++
			// Release barrier.
			ps.bs <- b
			return nil
		}
		// Release barrier.
		ps.bs <- b

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-b:
			continue
		}
	}
}

func (ps *pollerSemaphore) release() {
	// Acquire barrier.
	b := <-ps.bs
	ps.permits--
	// Release one waiter if there are any waiting.
	select {
	case b <- struct{}{}:
	default:
	}
	// Release barrier.
	ps.bs <- b
}

func (ps *pollerSemaphore) updatePermits(maxPermits int) {
	// Acquire barrier.
	b := <-ps.bs
	ps.maxPermits = maxPermits
	// Release barrier.
	ps.bs <- b
}

func newScalableTaskPoller(
	poller taskPoller, logger log.Logger, pollerBehavior PollerBehavior) scalableTaskPoller {
	tw := scalableTaskPoller{
		taskPoller: poller,
	}
	switch p := pollerBehavior.(type) {
	case *pollerBehaviorAutoscaling:
		tw.pollerCount = p.initialNumberOfPollers
		tw.pollerSemaphore = newPollerSemaphore(p.initialNumberOfPollers)
		tw.pollerAutoscalerReportHandle = newPollScalerReportHandle(pollScalerReportHandleOptions{
			initialPollerCount: p.initialNumberOfPollers,
			maxPollerCount:     p.maximumNumberOfPollers,
			minPollerCount:     p.minimumNumberOfPollers,
			logger:             logger,
			scaleCallback: func(newTarget int) {
				tw.pollerSemaphore.updatePermits(newTarget)
			},
		})
	case *pollerBehaviorSimpleMaximum:
		tw.pollerCount = p.maximumNumberOfPollers
	}
	return tw
}

// balance checks if the poller type is balanced with other poller types. The goal is to ensure that
// at least one poller of each type is running before allowing any poller of the given type to increase.
func (pb *pollerBalancer) balance(ctx context.Context, pollerType string) error {
	pb.mu.Lock()
	// If there are no pollers of this type, we can skip balancing.
	if pb.pollerCount[pollerType] <= 0 {
		pb.mu.Unlock()
		return nil
	}
	for {
		var b barrier
		// Check if all other poller types have at least one poller running.
		for pt, count := range pb.pollerCount {
			if pt == pollerType {
				if count <= 0 {
					pb.mu.Unlock()
					return nil
				}
				continue
			}
			if count == 0 {
				b = pb.pollerBarrier[pt]
				break
			}
		}
		pb.mu.Unlock()
		// If all other poller types have at least one poller running, we are balanced
		if b == nil {
			return nil
		}
		// If we have a barrier that means that at least one other poller type has no pollers running.
		// We need to wait for that poller type to start a poller before we can continue.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-b:
			pb.mu.Lock()
			continue
		}
	}
}

func (pb *pollerBalancer) registerPollerType(pollerType string) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	if _, ok := pb.pollerCount[pollerType]; !ok {
		pb.pollerCount[pollerType] = 0
		pb.pollerBarrier[pollerType] = make(barrier)
	}
}

func (pb *pollerBalancer) incrementPoller(pollerType string) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	if pb.pollerCount[pollerType] == 0 {
		close(pb.pollerBarrier[pollerType])
		pb.pollerBarrier[pollerType] = make(barrier)
	}
	pb.pollerCount[pollerType]++
}

func (pb *pollerBalancer) decrementPoller(pollerType string) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.pollerCount[pollerType]--
}
