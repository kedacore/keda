package internal

import (
	"math/rand"
	"sync"
	"sync/atomic"

	"go.temporal.io/api/workflowservice/v1"
)

// eagerWorkflowDispatcher is responsible for finding an available worker for an eager workflow task.
type eagerWorkflowDispatcher struct {
	lock               sync.RWMutex
	workersByTaskQueue map[string]map[eagerWorker]struct{}
}

// registerWorker registers a worker that can be used for eager workflow dispatch
func (e *eagerWorkflowDispatcher) registerWorker(worker *workflowWorker) {
	e.lock.Lock()
	defer e.lock.Unlock()
	taskQueue := worker.executionParameters.TaskQueue
	if e.workersByTaskQueue[taskQueue] == nil {
		e.workersByTaskQueue[taskQueue] = make(map[eagerWorker]struct{})
	}
	e.workersByTaskQueue[taskQueue][worker.worker] = struct{}{}
}

// deregisterWorker deregister a worker so that it will not be used for eager workflow dispatch
func (e *eagerWorkflowDispatcher) deregisterWorker(worker *workflowWorker) {
	e.lock.Lock()
	defer e.lock.Unlock()
	delete(e.workersByTaskQueue[worker.executionParameters.TaskQueue], worker.worker)
}

// applyToRequest updates request if eager workflow dispatch is possible and returns the eagerWorkflowExecutor to use
func (e *eagerWorkflowDispatcher) applyToRequest(request *workflowservice.StartWorkflowExecutionRequest) *eagerWorkflowExecutor {
	// Try every worker that is assigned to the desired task queue.
	e.lock.RLock()
	workers := e.workersByTaskQueue[request.GetTaskQueue().Name]
	randWorkers := make([]eagerWorker, 0, len(workers))
	// Copy the workers so we can release the lock.
	for worker := range workers {
		randWorkers = append(randWorkers, worker)
	}
	e.lock.RUnlock()
	rand.Shuffle(len(randWorkers), func(i, j int) { randWorkers[i], randWorkers[j] = randWorkers[j], randWorkers[i] })
	for _, worker := range randWorkers {
		maybePermit := worker.tryReserveSlot()
		if maybePermit != nil {
			request.RequestEagerExecution = true
			return &eagerWorkflowExecutor{
				worker: worker,
				permit: maybePermit,
			}
		}
	}
	return nil
}

// eagerWorkflowExecutor is a worker-scoped executor for an eager workflow task.
type eagerWorkflowExecutor struct {
	handledResponse atomic.Bool
	worker          eagerWorker
	permit          *SlotPermit
}

// handleResponse of an eager workflow task from a StartWorkflowExecution request.
func (e *eagerWorkflowExecutor) handleResponse(response *workflowservice.PollWorkflowTaskQueueResponse) {
	if !e.handledResponse.CompareAndSwap(false, true) {
		panic("eagerWorkflowExecutor trying to handle multiple responses")
	}
	// Asynchronously execute the task
	e.worker.pushEagerTask(
		eagerTask{
			task: &eagerWorkflowTask{
				task: response,
			},
			permit: e.permit,
		})
}

// releaseUnused should be called if the executor cannot be used because no eager task was received.
// It will error if handleResponse was already called, as this would indicate misuse.
func (e *eagerWorkflowExecutor) releaseUnused() {
	if e.handledResponse.CompareAndSwap(false, true) {
		e.worker.releaseSlot(e.permit, SlotReleaseReasonUnused)
	} else {
		panic("trying to release an eagerWorkflowExecutor that was used")
	}
}
