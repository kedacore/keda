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

// All code in this file is private to the package.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/serviceerror"
	"golang.org/x/time/rate"

	"go.temporal.io/sdk/internal/common/retry"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/backoff"
	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/log"
)

const (
	retryPollOperationInitialInterval         = 200 * time.Millisecond
	retryPollOperationMaxInterval             = 10 * time.Second
	retryPollResourceExhaustedInitialInterval = time.Second
	retryPollResourceExhaustedMaxInterval     = 10 * time.Second
	// How long the same poll task error can remain suppressed
	lastPollTaskErrSuppressTime = 1 * time.Minute
)

var (
	pollOperationRetryPolicy         = createPollRetryPolicy()
	pollResourceExhaustedRetryPolicy = createPollResourceExhaustedRetryPolicy()
	retryLongPollGracePeriod         = 2 * time.Minute
)

var errStop = errors.New("worker stopping")

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

	// WorkflowEnvironment Represents the environment for workflow.
	// Should only be used within the scope of workflow definition.
	WorkflowEnvironment interface {
		AsyncActivityClient
		LocalActivityClient
		WorkflowTimerClient
		SideEffect(f func() (*commonpb.Payloads, error), callback ResultHandler)
		GetVersion(changeID string, minSupported, maxSupported Version) Version
		WorkflowInfo() *WorkflowInfo
		Complete(result *commonpb.Payloads, err error)
		RegisterCancelHandler(handler func())
		RequestCancelChildWorkflow(namespace, workflowID string)
		RequestCancelExternalWorkflow(namespace, workflowID, runID string, callback ResultHandler)
		ExecuteChildWorkflow(params ExecuteWorkflowParams, callback ResultHandler, startedHandler func(r WorkflowExecution, e error))
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
			handler func(string, *commonpb.Payloads, *commonpb.Header, UpdateCallbacks),
		)
		IsReplaying() bool
		MutableSideEffect(id string, f func() interface{}, equals func(a, b interface{}) bool) converter.EncodedValue
		GetDataConverter() converter.DataConverter
		GetFailureConverter() converter.FailureConverter
		AddSession(sessionInfo *SessionInfo)
		RemoveSession(sessionID string)
		GetContextPropagators() []ContextPropagator
		UpsertSearchAttributes(attributes map[string]interface{}) error
		UpsertMemo(memoMap map[string]interface{}) error
		GetRegistry() *registry
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
		Close()
	}

	// baseWorkerOptions options to configure base worker.
	baseWorkerOptions struct {
		pollerCount       int
		pollerRate        int
		maxConcurrentTask int
		maxTaskPerSecond  float64
		taskWorker        taskPoller
		identity          string
		workerType        string
		stopTimeout       time.Duration
		fatalErrCb        func(error)
		userContextCancel context.CancelFunc
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

		// Must be atomically accessed
		taskSlotsAvailable      int32
		taskSlotsAvailableGauge metrics.Gauge

		pollerRequestCh    chan struct{}
		taskQueueCh        chan interface{}
		fatalErrCb         func(error)
		sessionTokenBucket *sessionTokenBucket

		lastPollTaskErrMessage string
		lastPollTaskErrStarted time.Time
		lastPollTaskErrLock    sync.Mutex
	}

	polledTask struct {
		task interface{}
	}
)

// SetRetryLongPollGracePeriod sets the amount of time a long poller retrys on
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
	logger log.Logger,
	metricsHandler metrics.Handler,
	sessionTokenBucket *sessionTokenBucket,
) *baseWorker {
	ctx, cancel := context.WithCancel(context.Background())
	bw := &baseWorker{
		options:            options,
		stopCh:             make(chan struct{}),
		taskLimiter:        rate.NewLimiter(rate.Limit(options.maxTaskPerSecond), 1),
		retrier:            backoff.NewConcurrentRetrier(pollOperationRetryPolicy),
		logger:             log.With(logger, tagWorkerType, options.workerType),
		metricsHandler:     metricsHandler.WithTags(metrics.WorkerTags(options.workerType)),
		taskSlotsAvailable: int32(options.maxConcurrentTask),
		pollerRequestCh:    make(chan struct{}, options.maxConcurrentTask),
		taskQueueCh:        make(chan interface{}), // no buffer, so poller only able to poll new task after previous is dispatched.
		fatalErrCb:         options.fatalErrCb,

		limiterContext:       ctx,
		limiterContextCancel: cancel,
		sessionTokenBucket:   sessionTokenBucket,
	}
	// Set secondary retrier as resource exhausted
	bw.retrier.SetSecondaryRetryPolicy(pollResourceExhaustedRetryPolicy)
	bw.taskSlotsAvailableGauge = bw.metricsHandler.Gauge(metrics.WorkerTaskSlotsAvailable)
	bw.taskSlotsAvailableGauge.Update(float64(bw.taskSlotsAvailable))
	if options.pollerRate > 0 {
		bw.pollLimiter = rate.NewLimiter(rate.Limit(options.pollerRate), 1)
	}

	return bw
}

// Start starts a fixed set of routines to do the work.
func (bw *baseWorker) Start() {
	if bw.isWorkerStarted {
		return
	}

	bw.metricsHandler.Counter(metrics.WorkerStartCounter).Inc(1)

	for i := 0; i < bw.options.pollerCount; i++ {
		bw.stopWG.Add(1)
		go bw.runPoller()
	}

	bw.stopWG.Add(1)
	go bw.runTaskDispatcher()

	bw.isWorkerStarted = true
	traceLog(func() {
		bw.logger.Info("Started Worker",
			"PollerCount", bw.options.pollerCount,
			"MaxConcurrentTask", bw.options.maxConcurrentTask,
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

func (bw *baseWorker) runPoller() {
	defer bw.stopWG.Done()
	bw.metricsHandler.Counter(metrics.PollerStartCounter).Inc(1)

	for {
		select {
		case <-bw.stopCh:
			return
		case <-bw.pollerRequestCh:
			if bw.sessionTokenBucket != nil {
				bw.sessionTokenBucket.waitForAvailableToken()
			}
			bw.pollTask()
		}
	}
}

func (bw *baseWorker) runTaskDispatcher() {
	defer bw.stopWG.Done()

	for i := 0; i < bw.options.maxConcurrentTask; i++ {
		bw.pollerRequestCh <- struct{}{}
	}

	for {
		// wait for new task or worker stop
		select {
		case <-bw.stopCh:
			return
		case task := <-bw.taskQueueCh:
			// for non-polled-task (local activity result as task), we don't need to rate limit
			_, isPolledTask := task.(*polledTask)
			if isPolledTask && bw.taskLimiter.Wait(bw.limiterContext) != nil {
				if bw.isStop() {
					return
				}
			}
			bw.stopWG.Add(1)
			go bw.processTask(task)
		}
	}
}

func (bw *baseWorker) pollTask() {
	var err error
	var task interface{}
	bw.retrier.Throttle(bw.stopCh)
	if bw.pollLimiter == nil || bw.pollLimiter.Wait(bw.limiterContext) == nil {
		task, err = bw.options.taskWorker.PollTask()
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
			// We use the secondary retrier on resource exhausted
			_, resourceExhausted := err.(*serviceerror.ResourceExhausted)
			bw.retrier.Failed(resourceExhausted)
		} else {
			bw.retrier.Succeeded()
		}
	}

	if task != nil {
		select {
		case bw.taskQueueCh <- &polledTask{task}:
		case <-bw.stopCh:
		}
	} else {
		bw.pollerRequestCh <- struct{}{} // poll failed, trigger a new poll
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

func (bw *baseWorker) processTask(task interface{}) {
	defer bw.stopWG.Done()

	// Update availability metric
	bw.taskSlotsAvailableGauge.Update(float64(atomic.AddInt32(&bw.taskSlotsAvailable, -1)))
	defer func() {
		bw.taskSlotsAvailableGauge.Update(float64(atomic.AddInt32(&bw.taskSlotsAvailable, 1)))
	}()

	// If the task is from poller, after processing it we would need to request a new poll. Otherwise, the task is from
	// local activity worker, we don't need a new poll from server.
	polledTask, isPolledTask := task.(*polledTask)
	if isPolledTask {
		task = polledTask.task
	}
	defer func() {
		if p := recover(); p != nil {
			topLine := fmt.Sprintf("base worker for %s [panic]:", bw.options.workerType)
			st := getStackTraceRaw(topLine, 7, 0)
			bw.logger.Error("Unhandled panic.",
				"PanicError", fmt.Sprintf("%v", p),
				"PanicStack", st)
		}

		if isPolledTask {
			bw.pollerRequestCh <- struct{}{}
		}
	}()
	err := bw.options.taskWorker.ProcessTask(task)
	if err != nil {
		if isClientSideError(err) {
			bw.logger.Info("Task processing failed with client side error", tagError, err)
		} else {
			bw.logger.Info("Task processing failed with error", tagError, err)
		}
	}
}

// Stop is a blocking call and cleans up all the resources associated with worker.
func (bw *baseWorker) Stop() {
	if !bw.isWorkerStarted {
		return
	}
	close(bw.stopCh)
	bw.limiterContextCancel()

	if success := awaitWaitGroup(&bw.stopWG, bw.options.stopTimeout); !success {
		traceLog(func() {
			bw.logger.Info("Worker graceful stop timed out.", "Stop timeout", bw.options.stopTimeout)
		})
	}

	// Close context
	if bw.options.userContextCancel != nil {
		bw.options.userContextCancel()
	}

	bw.isWorkerStarted = false
}
